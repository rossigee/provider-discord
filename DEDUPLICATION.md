# Discord Channel Deduplication

Quick reference for the Discord channel deduplication feature.

## What is it?

An automated feature to identify and remove duplicate Discord channels managed by provider-discord. Converts the manual `discord-channel-dedupe` tool into a Kubernetes-native provider capability with full audit trail and safety mechanisms.

## Quick Start

### Step 1: Analyze (Safe)

```bash

kubectl annotate providerconfig default \
  discord.crossplane.io/deduplication=report --overwrite

```


### Step 2: Review Results

```bash

kubectl get deduplication
kubectl describe deduplication <name>

```


### Step 3: Delete (If Satisfied)

```bash

kubectl annotate providerconfig default \
  discord.crossplane.io/deduplication=action --overwrite

```


### Step 4: Verify

```bash

kubectl describe deduplication <name>
# Check Discord for deleted channels

```


## Key Features

- ✅ **Safe by Default**: Report mode analyzes without changes
- ✅ **Annotation-Driven**: Trigger via ProviderConfig annotation
- ✅ **Idempotent**: Prevents accidental re-runs
- ✅ **Audit Trail**: Full Kubernetes Events and CRD tracking
- ✅ **Guild Filtering**: Optional targeting of specific servers
- ✅ **Oldest Preserved**: Keeps channel with lowest position (most history)
- ✅ **Error Resilient**: Continues on individual channel failures

## Implementation

### Components
- **DeduplicationService**: Core logic for analyzing/deleting channels
- **DeduplicationController**: Watches annotations and triggers service
- **Deduplication CRD**: Tracks operations and results
- **ProviderConfig Extension**: Configuration via Kubernetes

### Architecture

```

ProviderConfig (annotation)
    ↓ (watches)
DeduplicationController
    ├→ Extract credentials
    ├→ DeduplicationService.AnalyzeAndDeduplicate()
    ├→ Create Deduplication CRD
    ├→ Emit Events
    └→ Update annotation (idempotency)

```


## Operational Modes

### Report Mode (Safe Analysis)

```bash

kubectl annotate providerconfig default \
  discord.crossplane.io/deduplication=report --overwrite

```

- Analyzes all guilds for duplicate channels
- Reports findings via Kubernetes Events
- **Makes NO changes to Discord**
- Safe to run multiple times

### Action Mode (Deletion)

```bash

kubectl annotate providerconfig default \
  discord.crossplane.io/deduplication=action --overwrite

```

- Deletes duplicate channels identified in report
- Keeps oldest channel (lowest position)
- Updates Deduplication CRD with results
- **Only run after reviewing report!**

## Configuration

### In ProviderConfig Spec

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
    deleteOrphanedResources: true
    targetGuilds:  # Optional: only these guilds
      - "123456789012345678"

```


### Via Annotation

```bash

# Apply annotation to trigger
kubectl annotate providerconfig default \
  discord.crossplane.io/deduplication=report --overwrite

# Switch modes
kubectl annotate providerconfig default \
  discord.crossplane.io/deduplication=action --overwrite --force

```


## Monitoring

### Check Results

```bash

# List all deduplication operations
kubectl get deduplication

# View detailed results
kubectl get deduplication <name> -o yaml

# Check events on ProviderConfig
kubectl describe providerconfig default

```


### Expected Output

```yaml

status:
  phase: completed
  summary:
    totalGuildsAnalyzed: 2
    totalChannelsAnalyzed: 42
    duplicateGroupsFound: 3
    totalDuplicateChannelsFound: 6
    channelsDeleted: 6
    orphanedResourcesDeleted: 0
  results:
    "guild-id":
      guildName: "My Server"
      totalChannels: 28
      duplicateGroups:
        - channelName: "general"
          count: 2
          keptChannelId: "id-to-keep"
          deletedChannelIds: ["id-to-delete"]

```


## Deduplication Strategy

**Channel Selection**: When duplicates are found with the same name:
1. **Group** channels by exact name match (case-sensitive)
2. **Find oldest** - Select channel with lowest position number
3. **Keep oldest** - Preserve it with all message history
4. **Delete others** - Remove remaining duplicates

**Why position?**: Discord's position field determines display order. Lower position = higher in list = older = more likely to have complete message history.

## Recovery

### If Wrong Channels Were Deleted

1. **Check Audit Log**: Discord logs all deletions
2. **Recreate Channels**: Manually recreate in Discord UI
3. **Clean Crossplane**: Remove resources for deleted channels
4. **Verify**: Run report mode again to confirm cleanup

### Crossplane Resource Cleanup

If `deleteOrphanedResources: true`:
- Deletes Crossplane Channel resources for deleted Discord channels
- Keeps cluster state consistent with Discord

If `deleteOrphanedResources: false`:
- Discord channels are deleted
- Crossplane resources remain (cleanup manually)

## Troubleshooting

### Deduplication Doesn't Run

**Check**:
1. Annotation exists: `kubectl get providerconfig default -o jsonpath='{.metadata.annotations.discord\.crossplane\.io/deduplication}'`
2. Controller logs: `kubectl logs -n crossplane-system -l app=provider-discord`
3. Secret exists: `kubectl get secret discord-token -n crossplane-system`

### API Errors

**Check**:
1. Bot token valid: `curl -H "Authorization: Bot $TOKEN" https://discord.com/api/v10/users/@me`
2. Bot permissions: Needs "View Channels" and "Manage Channels"
3. Bot in guild: Verify bot is actually a member of target servers

### No Duplicates Found

1. Guild may actually have no duplicates
2. Bot may lack permissions
3. Check Deduplication CRD for errors: `kubectl describe deduplication <name>`

## Documentation

- **Feature Guide**: [docs/deduplication.md](docs/deduplication.md)
- **Operational Runbook**: [docs/deduplication-runbook.md](docs/deduplication-runbook.md)
- **Testing Guide**: [docs/TESTING-GUIDE.md](docs/TESTING-GUIDE.md)
- **Examples**: [examples/deduplication-*.yaml](examples/)
- **Implementation**: [docs/DEDUPLICATION-IMPLEMENTATION.md](docs/DEDUPLICATION-IMPLEMENTATION.md)

## Testing


```bash

# Run unit tests
make test

# Run tests with coverage
go test -cover ./internal/services ./internal/controller/deduplication

# Run full validation
make reviewable

```


## FAQ

**Q: Will it delete channels with messages?**
A: Yes, deduplication only looks at channel names. It will delete duplicates even if they have messages. However, it always keeps the oldest (lowest position), which typically has the most history.

**Q: What if I run report mode twice?**
A: Safe - idempotency prevents re-running. Must switch to action mode to proceed.

**Q: Can I deduplicate specific guilds?**
A: Yes, use `targetGuilds` in spec or filter in your own orchestration.

**Q: Can I undo a deletion?**
A: No. Discord doesn't restore deleted channels. Always review report mode first!

## Requirements

- Discord bot token with permissions:
  - View Channels
  - Manage Channels (for action mode)
- Bot must be member of target Discord servers
- Valid Kubernetes cluster with Crossplane

## Status

✅ **Production Ready**
- Fully implemented and tested
- Comprehensive documentation
- Operational procedures included
- Safe, annotation-based triggering

## See Also

- [ProviderConfig Documentation](docs/provider-config.md)
- [Discord API Reference](https://discord.com/developers/docs)
- [Crossplane Documentation](https://crossplane.io/docs)
