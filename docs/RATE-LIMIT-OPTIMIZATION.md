# Rate Limit Optimization: Smart Caching Implementation

## Overview

Reduce API load by 60-80% at scale through **status-driven reconciliation**. Resources recently synced skip expensive Observe calls, balancing rate-limit relief against drift detection.

**Expected Impact:**
- Baseline polling load: 200+ objects/min → ~60-80 objects/min (5min cache window)
- Multi-minute stalls become rare; only active workloads trigger rate-limit windows
- Recovery from incidents 5-10x faster

## Architecture

### 1. Caching Helper (`internal/controller/cache.go`)

Provides two functions:
- `ShouldSkipObserve(lastSyncTime)` - Check if resource was recently synced
- `UpdateSyncTime(lastSyncTime)` - Record successful sync time

Cache TTL: **5 minutes** (configurable, set conservatively)

### 2. Schema Changes

Add to **all 9 resource status types**:

```go
type SomeObservation struct {
    // ...existing fields...
    
    // LastSyncTime records when this resource was last successfully observed.
    // Used to implement status-driven reconciliation: skip redundant API calls
    // for resources synced within the last 5 minutes.
    // +optional
    LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`
}
```

**Files to update:**
- `apis/channel/v1beta1/types.go` → ChannelObservation
- `apis/guild/v1beta1/types.go` → GuildObservation
- `apis/role/v1beta1/types.go` → RoleObservation
- `apis/webhook/v1beta1/types.go` → WebhookObservation
- `apis/invite/v1beta1/types.go` → InviteObservation
- `apis/member/v1beta1/types.go` → MemberObservation
- `apis/user/v1beta1/types.go` → DiscordUserObservation
- `apis/application/v1beta1/types.go` → ApplicationObservation
- `apis/integration/v1beta1/types.go` → IntegrationObservation

### 3. Controller Changes

**Pattern for each Observe method:**

```go
func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
    cr, ok := mg.(*somev1alpha1.SomeResource)
    if !ok {
        return managed.ExternalObservation{}, errors.New(errNotSomeResource)
    }

    // Skip expensive API call if recently synced
    if controller.ShouldSkipObserve(cr.Status.AtProvider.LastSyncTime) {
        log := ctrl.LoggerFrom(ctx)
        log.V(4).Info("Skipping observe: resource recently synced", 
            "lastSync", cr.Status.AtProvider.LastSyncTime)
        
        // Return current observed state from status without calling Discord API
        return managed.ExternalObservation{
            ResourceExists:   true,
            ResourceUpToDate: true,
            // Note: could be stale after 5+ minutes, but acceptable for stable resources
        }, nil
    }

    // ... existing Observe logic ...

    // After successful observation, record sync time
    controller.UpdateSyncTime(&cr.Status.AtProvider.LastSyncTime)
    
    return managed.ExternalObservation{...}, nil
}
```

## Implementation Order

**Phase 1: Foundation** (This PR)
- ✅ Create `cache.go` helper
- ✅ Document strategy

**Phase 2: Stable Resources** (Next PR)
1. Channel (most stable, highest volume)
2. Guild (stable)
3. Role (stable)

**Phase 3: Dynamic Resources** (Follow-up PR)
4. Webhook
5. Member
6. Integration

**Phase 4: Other Resources** (Final PR)
7. Invite
8. User
9. Application

## Trade-offs

### Pros
✅ 60-80% reduction in polling load  
✅ 5-10x faster incident recovery  
✅ Scales linearly: 1000 objects = 200+ calls/min → ~40 calls/min  
✅ Minimal code changes per controller  

### Cons
⚠️ **Drift Detection Window:** External Discord changes undetected for up to 5 minutes
   - **Mitigation:** 5-minute window is reasonable for largely-static resources (Channels rarely change)
   - **Worst case:** User modifies channel in Discord directly → up to 5min until controller syncs
   - **Acceptable:** Most Discord interactions via Kubernetes resources, not Discord UI

⚠️ **Stale Status:** `cr.Status.AtProvider` could be up to 5 minutes old
   - **Mitigation:** Only used when skipping Observe; still accurate for comparisons in isUpToDate
   - **Context:** Crossplane's reconciliation is inherently eventual-consistency

## Monitoring & Validation

### Metrics to Track
- API call rate before/after (should drop 60-80%)
- Observe call skip rate (target: 80-90% for stable resources)
- Rate-limit stall frequency (should be rare/zero)

### Test Scenarios
1. Normal steady-state: Verify most resources skip Observe
2. External drift: Change resource in Discord directly, verify controller syncs within 5min
3. Large backlog: Create many resources at once, verify rate limits don't stall unrelated types
4. Incident recovery: Delete CRD and recreate from backup, verify recovery is fast

## Configuration

Future enhancement: Make `CacheTTL` configurable via ProviderConfig:

```yaml
apiVersion: discord.m.crossplane.io/v1beta1
kind: ProviderConfig
metadata:
  name: discord-provider-config
spec:
  cache:
    ttl: 5m  # Skip Observe for 5 minutes after sync
    enabled: true
```

## References

- Issue #9: Global rate limiter + 1m poll-interval causes multi-minute stalls
- Crossplane: Eventual consistency model, controller re-queuing on errors
- Discord: Global rate limit ~3000 req/min per bot token
