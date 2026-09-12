# Application

**API Version**: `application.discord.m.crossplane.io/v1beta1`

Bot application configuration.

## Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `forProvider.applicationId` | string | yes | Application ID |
| `forProvider.name` | string | no | App name |
| `forProvider.description` | string | no | Description |
| `forProvider.botPublic` | bool | no | Public bot flag |

## Example

```yaml
apiVersion: application.discord.m.crossplane.io/v1beta1
kind: Application
metadata:
  name: example-application
  namespace: default
spec:
  forProvider:
    applicationId: <value>
  providerConfigRef:
    name: default
```

## Behavior

- **Create/Update/Delete**: Full managed lifecycle via the Discord API v10; observe reconciles name/description/bot flags.
- Identity tracked as `applicationId` in status.
