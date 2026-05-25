# Discord Channel Deduplication Runbook

Operational procedures for identifying and removing duplicate Discord channels.

## Quick Start

### Analyze Duplicates (Safe)


```bash

# Step 1: Apply report mode annotation
kubectl annotate providerconfig default \
  discord.crossplane.io/deduplication=report --overwrite

# Step 2: Wait for analysis to complete (usually <1 minute)
kubectl get deduplication -w

# Step 3: Review results
kubectl get deduplication -o wide
kubectl describe deduplication <name>

# Step 4: Check events
kubectl describe providerconfig default

```


### Delete Duplicates (Destructive)


```bash

# Only after reviewing report mode results!

# Step 1: Apply action mode annotation
kubectl annotate providerconfig default \
  discord.crossplane.io/deduplication=action --overwrite

# Step 2: Monitor deletion
kubectl get deduplication -w

# Step 3: Verify in Discord
# Log in and check affected servers

# Step 4: Clean up Crossplane resources (if needed)
kubectl get channels | grep -E "deleted-channel-id"
kubectl delete channel <resource-name>

```


## Operational Procedures

### SOP 1: Regular Deduplication Check

**Purpose**: Periodic audit to identify accidental duplicates
**Frequency**: Monthly or as needed
**Duration**: 5-10 minutes


```bash

# 1. Trigger analysis
kubectl annotate providerconfig default \
  discord.crossplane.io/deduplication=report --overwrite

# 2. Wait for completion
kubectl wait --for=condition=completed deduplication/default-* \
  --timeout=5m 2>/dev/null || true

# 3. Export results for record-keeping
kubectl get deduplication -o json > dedup-report-$(date +%Y%m%d).json

# 4. Summarize findings
kubectl get deduplication -o json | jq '.items[] | select(.status.phase=="completed") | {
  name: .metadata.name,
  guilds: .status.summary.totalGuildsAnalyzed,
  duplicates: .status.summary.totalDuplicateChannelsFound,
  channels: .status.summary.totalChannelsAnalyzed
}'

# 5. Report if duplicates found
# If duplicates > 0, escalate to SOP 2 (Duplicate Cleanup)

```


### SOP 2: Duplicate Cleanup

**Purpose**: Remove identified duplicate channels
**Risk Level**: HIGH (destructive operation)
**Requires Approval**: YES
**Duration**: 10-30 minutes

**Prerequisites:**
- [ ] Report mode analysis completed and results reviewed
- [ ] Approval from Discord server owner/manager
- [ ] Backup of critical channel configuration (if applicable)

**Procedure:**


```bash

# STEP 1: Confirm we have a clean analysis
echo "Recent deduplication analyses:"
kubectl get deduplication --sort-by=.metadata.creationTimestamp | tail -5

echo "Select the MOST RECENT analysis with mode: report"
read -p "Enter deduplication name: " dedup_name

# STEP 2: Review the report one more time
echo "=== FINAL REVIEW BEFORE ACTION ==="
kubectl get deduplication/$dedup_name -o yaml | grep -A 100 "results:" | head -50

# STEP 3: Get confirmation
read -p "Confirm deletion? Type 'DELETE' to proceed: " confirm
if [ "$confirm" != "DELETE" ]; then
    echo "Cancelled."
    exit 1
fi

# STEP 4: Apply action mode
echo "Triggering deduplication action..."
kubectl annotate providerconfig default \
  discord.crossplane.io/deduplication=action --overwrite

# STEP 5: Monitor progress
kubectl get deduplication --sort-by=.metadata.creationTimestamp -w

# STEP 6: Verify completion
kubectl get deduplication -o json | \
  jq '.items[-1] | select(.status.phase=="completed") | .status.summary'

# STEP 7: Document results
echo "Deduplication completed. Results:"
kubectl get deduplication -o json | \
  jq '.items[-1] | {
    phase: .status.phase,
    guilds: .status.summary.totalGuildsAnalyzed,
    deleted: .status.summary.channelsDeleted,
    orphaned: .status.summary.orphanedResourcesDeleted
  }' | tee dedup-action-$(date +%Y%m%d-%H%M%S).json

# STEP 8: Clean up Crossplane resources
echo "Cleaning up orphaned Crossplane resources..."
kubectl get channels -o name | while read channel; do
    # Check if the channel's Discord ID exists (this is simplified)
    # In reality, you'd need to check the status or annotations
    echo "Verify: $channel"
done

```


### SOP 3: Failure Recovery

**Purpose**: Recover from failed or incorrect deduplication
**Duration**: 15-45 minutes depending on scope

**Symptoms:**
- Wrong channels were deleted
- Some channels failed to delete
- Crossplane resources out of sync with Discord

**Recovery:**


```bash

# STEP 1: Assess the situation
echo "=== FAILURE ASSESSMENT ==="

# List failed deduplication operations
kubectl get deduplication --field-selector=status.phase=failed

# Check for errors in recent operation
kubectl get deduplication -o json | \
  jq '.items[-1] | select(.status.phase != "completed")'

# STEP 2: Check Discord audit log
echo "Discord audit log will show what was deleted"
echo "1. Open Discord server settings"
echo "2. Go to Audit Log"
echo "3. Filter by 'Channel deletions'"
echo "4. Note the deleted channel names and IDs"

# STEP 3: Identify orphaned Crossplane resources
echo "Finding Crossplane resources for deleted channels..."
kubectl get channels -o custom-columns=NAME:.metadata.name,CHANNEL_ID:.spec.forProvider.name

# STEP 4: Manually recreate channels if needed
echo "If critical channels were deleted:"
echo "1. Recreate them in Discord"
echo "2. Note their new IDs"
echo "3. Update Crossplane manifests with new IDs"

# STEP 5: Clean up resources
echo "Deleting Crossplane resources for deleted channels..."
kubectl delete channel <resource-name> --dry-run=client

# STEP 6: Verify cleanup
echo "Run deduplication in report mode to verify:"
kubectl annotate providerconfig default \
  discord.crossplane.io/deduplication=report --overwrite

# STEP 7: Document incident
cat > incident-$(date +%Y%m%d).md << EOF
# Deduplication Incident Report

**Date**: $(date)
**Severity**: HIGH
**Root Cause**: [TO BE FILLED]
**Impact**: [TO BE FILLED]
**Resolution**: [TO BE FILLED]
**Lessons Learned**: [TO BE FILLED]
EOF

echo "Document incident in incident-$(date +%Y%m%d).md"

```


## Troubleshooting

### Issue: Deduplication Annotation Applied But Nothing Happens

**Diagnosis:**


```bash

# Check controller logs
kubectl logs -n crossplane-system -l app=provider-discord --tail=100

# Look for: "Starting deduplication" or error messages

# Check if annotation is recognized
kubectl get providerconfig default -o jsonpath='{.metadata.annotations.discord\.crossplane\.io/deduplication}'

# Check if ProviderConfig is being reconciled
kubectl get providerconfig default -o yaml | grep -A 3 "Status:"

```


**Solutions:**


```bash

# 1. Verify annotation syntax
kubectl annotate providerconfig default \
  discord.crossplane.io/deduplication=report --overwrite --dry-run=client

# 2. Check for permission issues
kubectl get clusterrole provider-discord -o yaml | grep -i "dedup"

# 3. Restart controller if stuck
kubectl rollout restart deployment/provider-discord \
  -n crossplane-system

# 4. Check for rate limiting
kubectl describe event providerconfig/default | grep -i rate

```


### Issue: "failed to extract credentials" Error

**Diagnosis:**


```bash

# Verify secret exists
kubectl get secret discord-token -n crossplane-system

# Check secret contents
kubectl get secret discord-token -n crossplane-system -o yaml

# Verify secret reference in ProviderConfig
kubectl get providerconfig default -o yaml | grep -A 3 "secretRef:"

```


**Solutions:**


```bash

# 1. Create or update secret
kubectl create secret generic discord-token \
  --from-literal=token='your-bot-token' \
  -n crossplane-system \
  --dry-run=client -o yaml | kubectl apply -f -

# 2. Update ProviderConfig if namespace is wrong
kubectl patch providerconfig default \
  -p '{"spec":{"credentials":{"secretRef":{"namespace":"crossplane-system"}}}}'

# 3. Verify secret key name (should be "token")
kubectl get secret discord-token \
  -n crossplane-system \
  -o jsonpath='{.data}' | jq 'keys'

```


### Issue: Deduplication Takes Too Long

**Diagnosis:**


```bash

# Check phase
kubectl get deduplication -w

# Monitor resource usage
kubectl top pod -n crossplane-system | grep provider-discord

# Check for stuck operations
kubectl get deduplication | grep analyzing

```


**Solutions:**


```bash

# 1. If analyzing is stuck, check Discord API health
# Manually test Discord connectivity:
curl -H "Authorization: Bot $DISCORD_TOKEN" \
  https://discord.com/api/v10/users/@me/guilds

# 2. Check for Discord rate limiting
# Deduplication respects rate limits automatically
# If seeing rate limit errors, wait and retry

# 3. If targeting many guilds, consider splitting into batches:
kubectl annotate providerconfig default \
  discord.crossplane.io/deduplication=report --overwrite
# Edit spec to target fewer guilds:
kubectl patch providerconfig default \
  -p '{"spec":{"deduplication":{"targetGuilds":["123456789"]}}}'

```


### Issue: Channels Deleted But Some Resources Remain

**Diagnosis:**


```bash

# Find orphaned resources
kubectl get channels -o wide | grep "no-status"

# Check if resource status shows deleted channel
kubectl describe channel <name>

```


**Solutions:**


```bash

# 1. If deleteOrphanedResources was false, manually clean up:
for channel in $(kubectl get channels -o name); do
    echo "Review: $channel"
    kubectl describe $channel | grep -i "forprovider"
done

# 2. Identify deleted channel IDs
# Compare with Discord: https://discord.com/channels/@me

# 3. Delete resources for deleted channels
kubectl delete channel <name-with-deleted-id>

# 4. Run deduplication in report mode to verify cleanup
kubectl annotate providerconfig default \
  discord.crossplane.io/deduplication=report --overwrite

```


## Monitoring and Alerting

### Metrics to Track


```bash

# Total deduplication operations
kubectl get deduplication | wc -l

# Success rate
kubectl get deduplication -o json | \
  jq '[.items[] | select(.status.phase=="completed")] | length'

# Failed operations
kubectl get deduplication --field-selector=status.phase=failed

# Total channels deleted
kubectl get deduplication -o json | \
  jq '[.items[] | .status.summary.channelsDeleted] | add'

```


### Set Up Alerts


```bash

# Prometheus rule for failed deduplication
cat << 'EOF' | kubectl apply -f -
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: discord-dedup-alerts
spec:
  groups:
    - name: discord-dedup
      interval: 30s
      rules:
        - alert: DeduplicationFailed
          expr: count(deduplication_status_phase{phase="failed"}) > 0
          for: 5m
          labels:
            severity: warning
          annotations:
            summary: "Discord deduplication failed"
            description: "Deduplication operation failed to complete"
EOF

```


## Rollback Procedures

**Note**: Discord does not support rollback of deleted channels.

If critical channels were deleted:

1. **Immediate**: Document what was deleted (check audit log)
2. **Short-term**: Manually recreate essential channels in Discord
3. **Medium-term**: Restore from backups if available
4. **Long-term**: Implement Discord's backup feature to prevent future data loss

## Best Practices

1. **Always test in report mode first**
   - Never jump directly to action mode
   - Review results carefully before proceeding

2. **Schedule deduplication during maintenance windows**
   - Notify Discord server members
   - Perform during low-traffic periods

3. **Keep historical records**
   - Save deduplication reports
   - Maintain incident logs
   - Track metrics over time

4. **Implement access controls**
   - Restrict who can apply the annotation
   - Use Kubernetes RBAC to limit changes

5. **Monitor regularly**
   - Schedule monthly deduplication checks
   - Alert on unexpected duplicates

6. **Document exceptions**
   - If you want to keep duplicate channels, document why
   - Consider renaming duplicates instead of deleting

## Related Documentation

- [Deduplication Feature Guide](./deduplication.md)
- [ProviderConfig Documentation](./provider-config.md)
- [Discord API Reference](https://discord.com/developers/docs/)
