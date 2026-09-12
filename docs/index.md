# Provider Discord Documentation

A Crossplane v2 provider for managing Discord resources. All managed resources are namespaced (`*.discord.m.crossplane.io/v1beta1`) with full multi-tenancy support.

## Quick Links

- [Setup](discord-setup.md) — Bot tokens and ProviderConfig
- [Deduplication](deduplication.md) — Channel deduplication workflow
- [Troubleshooting](troubleshooting.md) — Common issues
- [Production Deployment](production-deployment.md) — Hardened installs

## Resource Documentation

| Resource | API Group | Description |
|----------|-----------|-------------|
| [Guild](resources/guild.md) | `guild.discord.m.crossplane.io/v1beta1` | Discord servers |
| [Channel](resources/channel.md) | `channel.discord.m.crossplane.io/v1beta1` | Text, voice, and category channels |
| [Role](resources/role.md) | `role.discord.m.crossplane.io/v1beta1` | Permission roles and hierarchy |
| [Webhook](resources/webhook.md) | `webhook.discord.m.crossplane.io/v1beta1` | Messaging and CI/CD webhooks |
| [Member](resources/member.md) | `member.discord.m.crossplane.io/v1beta1` | Membership, roles, moderation state |
| [User](resources/user.md) | `user.discord.m.crossplane.io/v1beta1` | User profile observation |
| [Application](resources/application.md) | `application.discord.m.crossplane.io/v1beta1` | Bot application configuration |
| [Integration](resources/integration.md) | `integration.discord.m.crossplane.io/v1beta1` | Third-party integrations |
| [Invite](resources/invite.md) | `invite.discord.m.crossplane.io/v1beta1` | Server invitations |
| Deduplication | `deduplication.discord.m.crossplane.io/v1beta1` | Channel deduplication operations (cluster-scoped) |
| ProviderConfig | `discord.m.crossplane.io/v1beta1` | Provider credentials (cluster-scoped) |

## API Coverage Gaps

Discord API surface not yet modeled by this provider:

- **Guild content**: emojis, stickers, scheduled events, soundboard, audit-log access, bans/kick lists as resources.
- **Channels**: threads (creation/archival), forum tags, stage instances, polls.
- **Moderation**: AutoMod rules, guild bans, onboarding/screening configuration.
- **Members**: multi-member bulk operations beyond per-member reconcile.
- **Webhooks/Invites**: message execution history, invite-use tracking.
- **Applications**: command (slash-command) registration and entitlements/SKUs.
- **Events**: no gateway event subscriptions; everything is poll-based observe.
