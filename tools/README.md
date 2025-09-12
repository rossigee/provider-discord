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