# Rate Limit Optimization - Implementation Tasks

## Phase 1: Foundation ✅
- [x] Create caching helper (`internal/controller/cache.go`)
- [x] Document architecture and strategy
- [x] Define implementation phases and tradeoffs

## Phase 2: Stable Resources (High Priority)

### Channel Controller
- [ ] Add `LastSyncTime *metav1.Time` to `ChannelObservation` in `apis/channel/v1beta1/types.go`
- [ ] Import `controller` package in `internal/controller/channel/channel.go`
- [ ] Update `Observe()` method:
  - Add `ShouldSkipObserve()` check at top
  - Add `UpdateSyncTime()` call after successful observation
  - Test with mock to verify skip logic
- [ ] Run tests: `go test ./internal/controller/channel/...`
- [ ] Manual test: Deploy 50 channels, verify API calls drop ~80%

### Guild Controller
- [ ] Add `LastSyncTime *metav1.Time` to `GuildObservation` in `apis/guild/v1beta1/types.go`
- [ ] Update `Observe()` in `internal/controller/guild/guild.go`
- [ ] Run tests: `go test ./internal/controller/guild/...`

### Role Controller
- [ ] Add `LastSyncTime *metav1.Time` to `RoleObservation` in `apis/role/v1beta1/types.go`
- [ ] Update `Observe()` in `internal/controller/role/role.go`
- [ ] Run tests: `go test ./internal/controller/role/...`

## Phase 3: Dynamic Resources (Medium Priority)

### Webhook Controller
- [ ] Add `LastSyncTime` to WebhookObservation
- [ ] Update `Observe()` method
- [ ] Test

### Member Controller
- [ ] Add `LastSyncTime` to MemberObservation (⚠️ members change more frequently)
- [ ] Use shorter TTL or disable caching for Members? (members join/leave frequently)
- [ ] Update `Observe()` method
- [ ] Test

### Integration Controller
- [ ] Add `LastSyncTime` to IntegrationObservation
- [ ] Update `Observe()` method
- [ ] Test

## Phase 4: Other Resources (Lower Priority)

### Invite Controller
- [ ] Add `LastSyncTime` to InviteObservation
- [ ] Update methods (if applicable)

### User Controller
- [ ] Add `LastSyncTime` to DiscordUserObservation
- [ ] Update methods

### Application Controller
- [ ] Add `LastSyncTime` to ApplicationObservation
- [ ] Update methods

## Validation & Testing

### Unit Tests
- [ ] Test `ShouldSkipObserve()` returns false for nil timestamp
- [ ] Test `ShouldSkipObserve()` returns false for old timestamp
- [ ] Test `ShouldSkipObserve()` returns true for recent timestamp
- [ ] Test `UpdateSyncTime()` sets current time

### Integration Tests
- [ ] Deploy provider with 100+ resources
- [ ] Monitor API call rate: should drop from 200+/min to ~40-80/min
- [ ] Verify rate limit stalls decrease significantly
- [ ] Test external drift: modify resource in Discord UI, verify sync within 5 minutes

### Performance Metrics
- [ ] Measure baseline API calls (v0.13.0 without caching)
- [ ] Measure optimized API calls (with caching)
- [ ] Target: 60-80% reduction
- [ ] Calculate cost savings (Discord API quota)

## Documentation Updates
- [ ] Add caching behavior to README
- [ ] Document drift detection window (5 minutes)
- [ ] Document cache TTL configuration option (future)
- [ ] Update troubleshooting guide with cache info

## Release Checklist
- [ ] All tests passing
- [ ] Integration tested at scale (200+ resources)
- [ ] Performance validated (60-80% API reduction)
- [ ] Changelog updated
- [ ] Version bumped (minor version, feature addition)
- [ ] Release notes explain rate-limit improvement

## Future Enhancements
- [ ] Make `CacheTTL` configurable via ProviderConfig
- [ ] Add per-resource-type cache TTLs (Members shorter, Channels longer)
- [ ] Add metrics export (cache hit rate, API call reduction %)
- [ ] Webhook-driven reconciliation (listen for Discord events instead of polling)
- [ ] Smart backoff: increase TTL after N successful syncs, reset on mismatch
