# Role

**API Version**: `role.discord.m.crossplane.io/v1beta1`

Permission roles and hierarchy.

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.name` | string | yes | Role name |
| `forProvider.guildId` | string | yes | Owning guild ID |
| `forProvider.color` | int | no | RGB color |
| `forProvider.hoist` | bool | no | Display separately |
| `forProvider.mentionable` | bool | no | Allow mentions |
| `forProvider.permissions` | string | no | Permission bitfield |
| `forProvider.position` | int | no | Hierarchy position |

## Example

```yaml
apiVersion: role.discord.m.crossplane.io/v1beta1
kind: Role
metadata:
  name: example-role
  namespace: default
spec:
  forProvider:
    name: <value>
  providerConfigRef:
    name: default
```

## Behavior

- **Create/Update/Delete**: Full managed lifecycle via the Discord API v10; observe reconciles name/color/permissions/position.
- Identity tracked as `roleId` in status.
