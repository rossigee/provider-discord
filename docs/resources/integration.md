# Integration

**API Version**: `integration.discord.m.crossplane.io/v1beta1`

Third-party service integrations.

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.guildId` | string | yes | Guild ID |
| `forProvider.integrationId` | string | yes | Integration ID |

## Example

```yaml
apiVersion: integration.discord.m.crossplane.io/v1beta1
kind: Integration
metadata:
  name: example-integration
  namespace: default
spec:
  forProvider:
    guildId: <value>
  providerConfigRef:
    name: default
```

## Behavior

- **Create/Update/Delete**: Full managed lifecycle via the Discord API v10; observe reconciles largely read-only; enable/sync limited.
- Identity tracked as `integrationId` in status.
