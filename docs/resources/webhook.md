# Webhook

**API Version**: `webhook.discord.m.crossplane.io/v1beta1`

Channel webhooks for messaging and CI/CD.

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.name` | string | yes | Webhook name |
| `forProvider.channelId` | string | yes | Owning channel ID |
| `forProvider.avatar` | string | no | Avatar hash |

## Example

```yaml
apiVersion: webhook.discord.m.crossplane.io/v1beta1
kind: Webhook
metadata:
  name: example-webhook
  namespace: default
spec:
  forProvider:
    name: <value>
  providerConfigRef:
    name: default
```

## Behavior

- **Create/Update/Delete**: Full managed lifecycle via the Discord API v10; observe reconciles name/avatar; token exposed via `writeConnectionSecretToRef`.
- Identity tracked as `webhookId` in status.
