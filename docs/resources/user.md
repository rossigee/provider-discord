# User

**API Version**: `user.discord.m.crossplane.io/v1beta1`

User profile observation.

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.userId` | string | yes | Discord user ID |

## Example

```yaml
apiVersion: user.discord.m.crossplane.io/v1beta1
kind: User
metadata:
  name: example-user
  namespace: default
spec:
  forProvider:
    userId: <value>
  providerConfigRef:
    name: default
```

## Behavior

- **Create/Update/Delete**: Full managed lifecycle via the Discord API v10; observe reconciles read-only profile mirror.
- Identity tracked as `userId` in status.
