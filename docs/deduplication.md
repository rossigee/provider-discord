# Discord Channel Deduplication Feature

## Overview

The Discord channel deduplication feature provides an automated way to identify and remove duplicate channels from Discord servers managed by provider-discord. This feature addresses the common problem of accidental channel duplication that can occur during server management.

### Key Features

- **Safe Analysis**: Run in "report" mode to analyze duplicates without making changes
- **Automated Deletion**: Optionally delete duplicates automatically in "action" mode
- **Intelligent Selection**: Keeps the oldest channel (by position) with the most message history
- **Audit Trail**: Complete Kubernetes Events and Deduplication CRD tracking
- **Resource Cleanup**: Optionally deletes orphaned Crossplane resources when channels are deleted
- **Guild Targeting**: Optionally limit deduplication to specific Discord servers

## Architecture

### Components

1. **DeduplicationService** (`internal/services/deduplication.go`)
   - Core logic for analyzing and deduplicating channels
   - Interacts with Discord API via HTTP
   - Returns detailed results per guild

2. **ProviderConfigReconciler** (`internal/controller/deduplication/deduplication.go`)
   - Watches ProviderConfig resources for deduplication annotations
   - Triggers the deduplication service
   - Creates/updates Deduplication CRD instances
   - Emits Kubernetes Events for audit trail

3. **Deduplication CRD** (`apis/deduplication/v1alpha1/types.go`)
   - Cluster-scoped resource tracking deduplication operations
   - Records per-guild results and summary statistics
   - Queryable history of all deduplication runs

### Workflow


```

ProviderConfig with annotation "report"/"action"
         ↓
   DeduplicationController (watches annotation)
         ↓
   DeduplicationService.AnalyzeAndDeduplicate()
         ↓
   Creates Deduplication CRD with results
         ↓
   Emits Kubernetes Events
         ↓
   Updates ProviderConfig annotation (marks as processed)

```


## Usage

### Prerequisites

1. A valid Discord bot token with these permissions:
   - View Channels
   - Manage Channels
   - Read Message History (optional)

2. The bot must be a member of the guilds to be deduplicated

3. ProviderConfig with valid credentials:
   ```yaml
   apiVersion: discord.crossplane.io/v1alpha1
   kind: ProviderConfig
   metadata:
     name: default
   spec:
     credentials:
       source: Secret
       secretRef:
         name: discord-token
         namespace: crossplane-system
   ```

### Phase 1: Report Mode (Safe Analysis)

Run deduplication in report mode to analyze without making changes:


```bash

# Apply report mode annotation
kubectl annotate providerconfig default \
  discord.crossplane.io/deduplication=report --overwrite

```


Check the ProviderConfig for events:

```bash

kubectl describe providerconfig default

```


Expected output:

```

Events:
  Type    Reason                    Message
  ----    ------                    -------
  Normal  DeduplicationCompleted    Deduplication report completed. Mode: report. Guilds analyzed: 2, Duplicates found: 6, Channels deleted: 0

```


View detailed results in the Deduplication CRD:

```bash

# List all deduplication operations
kubectl get deduplication

# View detailed results
kubectl get deduplication <name> -o yaml

```


### Phase 2: Review Results

Carefully inspect the Deduplication CRD results:


```yaml

# Example output
status:
  phase: completed
  summary:
    totalGuildsAnalyzed: 2
    totalChannelsAnalyzed: 42
    duplicateGroupsFound: 3
    totalDuplicateChannelsFound: 6
    channelsDeleted: 0
  results:
    "123456789012345678":
      guildId: "123456789012345678"
      guildName: "My Server"
      totalChannels: 28
      duplicateGroups:
        - channelName: "general"
          count: 2
          keptChannelId: "987654321098765432"
          deletedChannelIds:
            - "876543210987654321"
        - channelName: "development"
          count: 2
          keptChannelId: "765432109876543210"
          deletedChannelIds:
            - "654321098765432109"

```


**Verify:**
- Guild names are correct
- Duplicate channel groups are identified correctly
- The "kept" channels are the ones you want to preserve
- No critical channels will be deleted

### Phase 3: Action Mode (Actual Deletion)

Only proceed if you're satisfied with the report. Update the annotation to "action":


```bash

# Apply action mode annotation
kubectl annotate providerconfig default \
  discord.crossplane.io/deduplication=action --overwrite

```


Monitor the deletion:

```bash

# Watch deduplication progress
kubectl get deduplication --watch

# Check final state
kubectl get deduplication <name> -o yaml

```


Expected output (action completed):

```yaml

status:
  phase: completed
  summary:
    channelsDeleted: 6
    orphanedResourcesDeleted: 2  # If deleteOrphanedResources: true

```


### Phase 4: Verify in Discord

1. Log into Discord
2. Check each affected server
3. Confirm duplicate channels are removed
4. Verify the kept channel still has message history

## Configuration

### ProviderConfig Deduplication Spec


```yaml

apiVersion: discord.crossplane.io/v1alpha1
kind: ProviderConfig
metadata:
  name: default
spec:
  credentials:
    source: Secret
    secretRef:
      name: discord-token
      namespace: crossplane-system

  # Optional deduplication configuration
  deduplication:
    enabled: true
    mode: "report"  # or "action"

    # Delete Crossplane resources for deleted Discord channels
    deleteOrphanedResources: true

    # Optional: Only deduplicate specific guilds
    # If empty, all guilds are processed
    targetGuilds:
      - "123456789012345678"
      - "987654321098765432"

```


### Annotation-Based Triggering

The primary way to trigger deduplication is via annotations:


```bash

# Report mode (analysis only)
kubectl annotate providerconfig default \
  discord.crossplane.io/deduplication=report --overwrite

# Action mode (actual deletion)
kubectl annotate providerconfig default \
  discord.crossplane.io/deduplication=action --overwrite

```


**Note:** The controller tracks which mode was last processed via the `discord.crossplane.io/deduplication-processed` annotation. This prevents duplicate work if the same annotation is applied multiple times.

## Deduplication Strategy

### Channel Selection

When duplicate channels are found:

1. **Group by Name**: Find all channels with identical names in the same guild
2. **Sort by Position**: Discord channels have a position field determining display order
3. **Keep Oldest**: Preserve the channel with the **lowest position number** (oldest/highest priority)
4. **Delete Others**: Remove all other channels in the duplicate group

### Why Position?

- Lower position = higher in the channel list = typically created first
- The oldest channel is most likely to have complete message history
- User expectations: the "original" channel is more important than later duplicates

## Crossplane Resource Cleanup

When `deleteOrphanedResources: true` (recommended), the controller will:

1. Identify all Crossplane Channel resources referencing deleted Discord channels
2. Delete those resources from the cluster
3. Keep cluster state consistent with Discord

If `deleteOrphanedResources: false`:
- Discord channels are deleted, but Crossplane resources remain
- You must manually clean up Crossplane resources:
  ```bash
  # Find resources with deleted channel IDs
  kubectl get channels -o wide

  # Delete obsolete resources
  kubectl delete channel <resource-name>
  ```

## Recovery and Troubleshooting

### Wrong Channels Deleted

If the deduplication deleted channels you wanted to keep:

1. **Check Discord's Audit Log**
   - Discord logs all deletions
   - See what was deleted and when

2. **Manually Recreate Channels**
   - Discord doesn't restore deleted channels
   - Recreate critical channels manually in Discord

3. **Update Crossplane Resources**
   - Find resources pointing to deleted channels
   - Delete the Crossplane resources

4. **Verify Cleanup**
   - Run deduplication in report mode again
   - Ensure no more duplicates are flagged

### Deduplication Fails to Run

**Symptoms**: Annotation applied but no Deduplication CRD created

**Troubleshooting:**

1. **Check controller logs**
   ```bash
   kubectl logs -n crossplane-system -l app=provider-discord --tail=100 | grep -i dedup
   ```

2. **Verify credentials**
   ```bash
   # Check secret exists
   kubectl get secret discord-token -n crossplane-system

   # Verify secret has token key
   kubectl get secret discord-token -n crossplane-system -o jsonpath='{.data}' | jq -r 'keys'
   ```

3. **Verify bot permissions**
   - Ensure bot has "View Channels" permission in guilds
   - Ensure bot has "Manage Channels" permission for action mode

4. **Check annotation syntax**
   ```bash
   # Verify annotation is correct
   kubectl get providerconfig default -o jsonpath='{.metadata.annotations.discord\.crossplane\.io/deduplication}'
   ```

### Deduplication Completes But Shows Errors

Check the Deduplication CRD status for per-guild errors:


```bash

kubectl get deduplication <name> -o yaml | grep -A 10 "errors:"

```


Common errors:
- `failed to fetch channels: 403 Forbidden` → Bot lacks permissions
- `failed to delete channel: 429 Too Many Requests` → Discord rate limiting
- `failed to delete channel: 404 Not Found` → Channel already deleted

## Monitoring and Auditing

### Track All Deduplication Operations


```bash

# List all deduplication runs
kubectl get deduplication --sort-by=.metadata.creationTimestamp

# Filter by status
kubectl get deduplication --field-selector=status.phase=completed
kubectl get deduplication --field-selector=status.phase=failed

```


### Get Statistics


```bash

# Total duplicates found across all runs
kubectl get deduplication -o json | jq '.items[] | .status.summary' | \
  jq -s 'map(.totalDuplicateChannelsFound) | add'

# Total channels deleted
kubectl get deduplication -o json | jq '.items[] | .status.summary' | \
  jq -s 'map(.channelsDeleted) | add'

```


### Audit Trail

Complete audit trail is maintained via:

1. **Kubernetes Events** on ProviderConfig
   ```bash
   kubectl describe providerconfig default | grep -A 20 Events:
   ```

2. **Deduplication CRD Status**
   - Records start and completion times
   - Per-guild results with channel details
   - Summary statistics

3. **Controller Logs**
   ```bash
   kubectl logs -n crossplane-system -l app=provider-discord --tail=1000 | \
     grep -i dedup
   ```

## Advanced Usage

### Deduplicate Only Specific Guilds


```yaml

apiVersion: discord.crossplane.io/v1alpha1
kind: ProviderConfig
metadata:
  name: default
  annotations:
    discord.crossplane.io/deduplication: "report"
spec:
  credentials:
    source: Secret
    secretRef:
      name: discord-token
      namespace: crossplane-system
  deduplication:
    targetGuilds:
      - "123456789012345678"  # Only this guild

```


### Custom DeduplicationService Usage

For advanced use cases, you can use the DeduplicationService directly:


```go

import "github.com/rossigee/provider-discord/internal/services"

httpClient := &http.Client{Timeout: 30 * time.Second}
svc := services.NewDeduplicationService(
    httpClient,
    "https://discord.com/api/v10",
    botToken,
    kubeClient,
)

result, err := svc.AnalyzeAndDeduplicate(ctx, "action", []string{})

```


## FAQ

**Q: Will deduplication delete channels with messages?**
A: No, deduplication only considers channel names. It will delete any duplicate with the same name, even if it has messages. However, it keeps the oldest channel (lowest position), which typically has more history.

**Q: Can I run deduplication on a schedule?**
A: Not currently. Deduplication is triggered via annotation. You can automate this with a CronJob that patches the ProviderConfig annotation.

**Q: What if the bot loses permissions while deleting?**
A: The Deduplication CRD will record per-channel deletion errors. You can retry from the last successful state by re-running in report mode and then action mode.

**Q: Does deduplication work with partial failures?**
A: Yes. If some channels fail to delete, others will continue to process. Failed deletions are recorded in the Deduplication CRD status.

**Q: Can I prevent accidental deduplication?**
A: Yes. Always run in report mode first and carefully review results. Consider using Kubernetes RBAC to restrict who can modify ProviderConfig annotations.

## Related Resources

- [ProviderConfig Documentation](./provider-config.md)
- [Discord API Channel Operations](https://discord.com/developers/docs/resources/channel)
- [Deduplication Examples](../examples/deduplication-*.yaml)
