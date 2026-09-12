# Guild

**API Version**: `guild.discord.m.crossplane.io/v1beta1`

Discord servers with full configuration.

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.name` | string | yes | Server name |
| `forProvider.region` | string | no | Voice region |
| `forProvider.verificationLevel` | int | no | 0-4 verification level |
| `forProvider.defaultMessageNotifications` | int | no | 0 all, 1 mentions |
| `forProvider.explicitContentFilter` | int | no | 0-2 content filter |
| `forProvider.afkChannelId` | string | no | AFK channel ID |
| `forProvider.afkTimeout` | int | no | AFK timeout seconds |

## Example

```yaml
apiVersion: guild.discord.m.crossplane.io/v1beta1
kind: Guild
metadata:
  name: example-guild
  namespace: default
spec:
  forProvider:
    name: <value>
  providerConfigRef:
    name: default
```

## Behavior

- **Create/Update/Delete**: Full managed lifecycle via the Discord API v10; observe reconciles name/region/verification settings.
- Identity tracked as `guildId` in status.
