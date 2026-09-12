# Invite

**API Version**: `invite.discord.m.crossplane.io/v1beta1`

Server invitations.

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.channelId` | string | yes | Channel ID |
| `forProvider.maxAge` | int | no | Expiry seconds |
| `forProvider.maxUses` | int | no | Max uses |
| `forProvider.temporary` | bool | no | Temporary membership |
| `forProvider.unique` | bool | no | Unique code |

## Example

```yaml
apiVersion: invite.discord.m.crossplane.io/v1beta1
kind: Invite
metadata:
  name: example-invite
  namespace: default
spec:
  forProvider:
    channelId: <value>
  providerConfigRef:
    name: default
```

## Behavior

- **Create/Update/Delete**: Full managed lifecycle via the Discord API v10; observe reconciles expiry/uses; token via `writeConnectionSecretToRef`.
- Identity tracked as `inviteCode` in status.
