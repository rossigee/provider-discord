# Member

**API Version**: `member.discord.m.crossplane.io/v1beta1`

Guild membership, roles, and moderation state.

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.guildId` | string | yes | Guild ID |
| `forProvider.userId` | string | yes | User ID |
| `forProvider.nick` | string | no | Guild nickname |
| `forProvider.roles` | array | no | Role IDs |
| `forProvider.mute` | bool | no | Voice mute |
| `forProvider.deaf` | bool | no | Voice deafen |
| `forProvider.communicationDisabledUntil` | string | no | Timeout expiry |

## Example

```yaml
apiVersion: member.discord.m.crossplane.io/v1beta1
kind: Member
metadata:
  name: example-member
  namespace: default
spec:
  forProvider:
    guildId: <value>
  providerConfigRef:
    name: default
```

## Behavior

- **Create/Update/Delete**: Full managed lifecycle via the Discord API v10; observe reconciles nickname/roles/mute/timeout.
- Identity tracked as `userId` in status.
