# Discord Introspection Tool

A comprehensive tool for introspecting existing Discord servers and generating Crossplane manifests for provider-discord.

## Features

### ✅ Supported Resources (Provider Ready)
- **Guilds** - Complete server discovery and manifest generation
- **Channels** - All channel types including categories with parent-child relationships  
- **Roles** - Custom roles with permissions and properties
- **Webhooks** - Server webhooks for automated messaging and CI/CD integration
- **Invites** - Server invitations with expiration and usage tracking

## Usage

### Basic Usage

```bash
# Set your Discord bot token
export DISCORD_BOT_TOKEN=your_bot_token_here

# Introspect all guilds and generate manifests
go run tools/discord-introspect.go

# Introspect a specific guild
go run tools/discord-introspect.go -guild="123456789012345678"
```

### Advanced Usage

```bash
# Custom output directory
go run tools/discord-introspect.go -output="my-discord-manifests"

# Include only channels and roles (skip guilds)
go run tools/discord-introspect.go -guilds=false

# Include all resource types (webhooks, invites, etc.)
go run tools/discord-introspect.go -webhooks=true -invites=true

# Selective resource introspection  
go run tools/discord-introspect.go -roles=false -webhooks=false -invites=false
```

### CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-guild` | "" | Specific guild ID to introspect (optional) |
| `-output` | "discord-resources" | Output directory for generated manifests |
| `-guilds` | true | Include guilds in introspection |
| `-channels` | true | Include channels in introspection |  
| `-roles` | true | Include roles in introspection |
| `-webhooks` | true | Include webhooks in introspection |
| `-invites` | true | Include invites in introspection |

## Output Structure

The tool generates organized manifests with proper dependency ordering:

```
discord-resources/
├── guild-my-server.yaml           # Guild configuration
├── channel-my-server-general.yaml         # Category channels first
├── channel-my-server-development.yaml     # Then regular channels  
├── channel-my-server-announcements.yaml  # With parent relationships
├── role-my-server-admin.yaml       # Custom roles
├── role-my-server-developer.yaml
├── webhook-my-server-ci-bot.yaml   # CI/CD integration webhooks
└── invite-my-server-general.yaml   # Server invitations
```

## Channel Features

### ✅ Category Support
- **Category Discovery**: Finds all category channels (type 4)
- **Hierarchy Management**: Creates categories before child channels  
- **Parent Relationships**: Generates `parentId` references properly
- **Dependency Ordering**: Categories → Regular channels

### ✅ Channel Properties
- **All Channel Types**: Text (0), Voice (2), Category (4), News (5), Stage (13), Forum (15)
- **Complete Properties**: Topic, NSFW, bitrate, user limits, rate limits
- **Type-Specific Fields**: Bitrate/user limits for voice, topic/NSFW for text
- **Position Ordering**: Maintains Discord's channel ordering

## Example Generated Manifests

### Category Channel
```yaml
apiVersion: channel.discord.crossplane.io/v1alpha1
kind: Channel
metadata:
  name: my-server-development
  annotations:
    discord.crossplane.io/id: "987654321098765432"
    discord.crossplane.io/type: "category"
spec:
  forProvider:
    name: "DEVELOPMENT"
    type: 4
    guildId: "123456789012345678"
    position: 0
  providerConfigRef:
    name: discord-provider-config
```

### Text Channel Under Category
```yaml
apiVersion: channel.discord.crossplane.io/v1alpha1
kind: Channel
metadata:
  name: my-server-general
  annotations:
    discord.crossplane.io/id: "876543210987654321"
    discord.crossplane.io/type: "text"
spec:
  forProvider:
    name: "general"
    type: 0
    guildId: "123456789012345678"
    position: 1
    parentId: "987654321098765432"  # References category
    topic: "General discussion"
    rateLimitPerUser: 5
  providerConfigRef:
    name: discord-provider-config
```

### Webhook Resource
```yaml
apiVersion: webhook.discord.crossplane.io/v1alpha1
kind: Webhook
metadata:
  name: my-server-ci-bot
  annotations:
    discord.crossplane.io/id: "765432109876543210"
    discord.crossplane.io/type: "incoming"
spec:
  forProvider:
    name: "CI Bot"
    channelId: "876543210987654321"
  writeConnectionSecretsToRef:
    name: ci-bot-webhook-secret
    namespace: default
  providerConfigRef:
    name: discord-provider-config
---
apiVersion: invite.discord.crossplane.io/v1alpha1
kind: Invite
metadata:
  name: my-server-general-invite
  annotations:
    discord.crossplane.io/id: "abcdef123456"
spec:
  forProvider:
    channelId: "876543210987654321"
    maxAge: 86400      # 24 hours
    maxUses: 100       # 100 uses
    temporary: false   # Permanent membership
  writeConnectionSecretsToRef:
    name: general-invite-secret
    namespace: default
  providerConfigRef:
    name: discord-provider-config
```

## Bot Permissions Required

Your Discord bot needs these permissions:
- **Read Messages/View Channels**: To discover channels and categories
- **Manage Roles**: To introspect custom roles  
- **Manage Webhooks**: To discover webhooks
- **Create Instant Invite**: To discover invites
- **View Audit Log**: For comprehensive server information

## Integration with GitOps

1. **Generate manifests** from existing Discord server
2. **Review and customize** the generated YAML files
3. **Commit to Git** repository for GitOps workflow
4. **Deploy with ArgoCD/Flux** to manage Discord infrastructure as code

## Troubleshooting

### Common Issues

**"DISCORD_BOT_TOKEN environment variable not set"**
- Set your bot token: `export DISCORD_BOT_TOKEN=your_token`

**"Guild with ID XXX not found"**
- Ensure your bot is a member of the specified guild
- Check the guild ID is correct

**"Warning: Failed to decode webhooks/invites"**
- Bot may lack permissions for these resources
- Non-fatal warnings - other resources will still be processed

**Empty webhook/invite discovery**
- Use `-discovery` flag to generate manifests even for unsupported resources
- Webhooks and invites require elevated bot permissions

### Validation

Verify generated manifests:
```bash
# Check YAML syntax
kubectl --dry-run=client apply -f discord-resources/

# Validate with provider (if deployed)
kubectl apply -f discord-resources/ --server-dry-run
```

## Development Notes

This tool provides comprehensive Discord infrastructure introspection:
1. **Production Use**: Generate manifests for all supported resources (Guild, Channel, Role, Webhook, Invite)
2. **GitOps Integration**: Ready-to-use manifests for immediate deployment
3. **Infrastructure Migration**: Import existing Discord servers into Crossplane management

The introspection tool enables seamless migration of existing Discord infrastructure to GitOps-managed workflows.

# Discord Channel Deduplication Tool

A safe tool for analyzing and removing duplicate channels created by the previous bug in provider-discord.

## Features

### ✅ Duplicate Analysis
- **Identify Duplicates**: Finds channels with identical names in the same guild
- **Safe Planning**: Generates detailed reports before any deletions
- **Preserve History**: Keeps the oldest channel (by position) with all message history
- **Dry Run Mode**: Analyze without making any changes

### ✅ Safe Deletion Strategy
- **Position-Based**: Keeps channel with lowest position (oldest/highest priority)
- **Confirmation Required**: Must explicitly enable `--confirm` to perform deletions
- **Detailed Logging**: Shows exactly what will be deleted
- **Backup Planning**: Generates migration reports for review

## Usage

### Analyze Duplicates (Safe)

```bash
# Set your Discord bot token
export DISCORD_BOT_TOKEN=your_bot_token_here

# Analyze all guilds for duplicates (dry run by default)
go run tools/discord-channel-dedupe.go

# Analyze a specific guild
go run tools/discord-channel-dedupe.go -guild="123456789012345678"

# Save analysis to file
go run tools/discord-channel-dedupe.go -guild="123456789012345678" -output="dedupe-plan.md"
```

### Perform Deletions (Dangerous)

```bash
# ⚠️  WARNING: This will actually delete channels!

# Perform actual deletions after review
go run tools/discord-channel-dedupe.go -guild="123456789012345678" -confirm

# Or use the compiled binary
./discord-channel-dedupe -guild="123456789012345678" -confirm
```

### CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-guild` | "" | Specific guild ID to analyze (required for deletions) |
| `-dry-run` | true | Dry run mode - analyze only, don't delete |
| `-confirm` | false | Actually perform deletions (DANGER - requires review) |
| `-output` | "" | Output file for deletion plan (optional) |

## Safety Features

### 🛡️ Multiple Safeguards
1. **Dry Run Default**: All analysis is safe by default
2. **Explicit Confirmation**: Must use `--confirm` to delete anything
3. **Detailed Reports**: Shows exactly what will be kept vs deleted
4. **Position Priority**: Keeps oldest channel with message history
5. **Guild-Specific**: Only affects specified guild

### 📋 Deletion Strategy
- **Keep Oldest**: Preserves channel with lowest position number
- **Preserve History**: Maintains all messages, permissions, and settings
- **Clean References**: Removes duplicate Crossplane resources
- **Audit Trail**: Logs all deletions for rollback if needed

## Example Output

```
🔍 Analyzing guild: My Server (123456789012345678)
  Found 25 total channels
  ⚠️  Found 3 duplicate channel groups with 6 total duplicate channels

## Duplicate Group: 'general'
# Found 2 channels with this name

- KEEP ✅ (oldest position) Channel ID: 987654321098765432, Position: 0, Type: text
- DELETE ❌ Channel ID: 876543210987654321, Position: 5, Type: text

## Duplicate Group: 'development'
# Found 3 channels with this name

- KEEP ✅ (oldest position) Channel ID: 765432109876543210, Position: 1, Type: category
- DELETE ❌ Channel ID: 654321098765432109, Position: 3, Type: category
- DELETE ❌ Channel ID: 543210987654321098, Position: 7, Type: category
```

## Migration Steps

### Phase 1: Analysis (Safe)
```bash
# 1. Analyze your Discord server
go run tools/discord-channel-dedupe.go -guild="YOUR_GUILD_ID" -output="dedupe-plan.md"

# 2. Review the plan carefully
cat dedupe-plan.md

# 3. Verify no important channels will be deleted
```

### Phase 2: Backup (Recommended)
```bash
# Backup important channel data if needed
# Note: Discord channels contain message history that cannot be recovered
```

### Phase 3: Execute (Careful)
```bash
# Execute the deduplication
go run tools/discord-channel-dedupe.go -guild="YOUR_GUILD_ID" -confirm

# Verify results
go run tools/discord-channel-dedupe.go -guild="YOUR_GUILD_ID"
```

## Recovery

### If Wrong Channels Deleted
1. **Check Audit Log**: Discord keeps logs of deleted channels
2. **Recreate Manually**: Use Discord UI to recreate important channels
3. **Update Crossplane**: Remove duplicate resources from your manifests

### Crossplane Resource Cleanup
After channel deletion, remove duplicate Crossplane resources:

```bash
# Find resources pointing to deleted channels
kubectl get channels -o wide

# Delete Crossplane resources for deleted channels
kubectl delete channel duplicate-channel-name
```

## Bot Permissions Required

Your Discord bot needs these permissions for deduplication:
- **View Channels**: To analyze channel structure
- **Manage Channels**: To delete duplicate channels
- **Read Message History**: To understand channel usage (optional)

## Integration with Provider Fix

This tool is designed to work with the provider-discord v0.8.0+ fix that prevents future duplicates:

1. **Run this tool** to clean up existing duplicates
2. **Deploy provider-discord v0.8.0+** to prevent new duplicates
3. **Monitor** for any remaining issues

## Technical Details

### Position-Based Selection
Discord channels have position numbers that determine display order. Lower position = higher in the list = older/more important. The tool keeps the channel with the lowest position number to preserve history and importance.

### Duplicate Detection
- Groups channels by exact name match (case-sensitive)
- Ignores system channels (@everyone, etc.)
- Considers all channel types (text, voice, category, etc.)

### Error Handling
- Safe failure: Stops on first API error
- Detailed logging: Shows progress and errors
- Rollback hints: Provides recovery guidance
