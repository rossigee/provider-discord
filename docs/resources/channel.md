# Channel

**API Version**: `channel.discord.m.crossplane.io/v1beta1`

Text, voice, and category channels.

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.name` | string | yes | Channel name |
| `forProvider.type` | int | yes | 0 text, 2 voice, 4 category, etc. |
| `forProvider.guildId` | string | yes | Owning guild ID |
| `forProvider.topic` | string | no | Channel topic |
| `forProvider.position` | int | no | Sort position |
| `forProvider.parentId` | string | no | Category parent ID |
| `forProvider.nsfw` | bool | no | Age-restricted flag |
| `forProvider.bitrate` | int | no | Voice bitrate |
| `forProvider.rateLimitPerUser` | int | no | Slow-mode seconds |

## Example

```yaml
apiVersion: channel.discord.m.crossplane.io/v1beta1
kind: Channel
metadata:
  name: example-channel
  namespace: default
spec:
  forProvider:
    name: <value>
  providerConfigRef:
    name: default
```

## Behavior

- **Create/Update/Delete**: Full managed lifecycle via the Discord API v10; observe reconciles name/topic/position/slow-mode.
- Identity tracked as `channelId` in status.
