# Discord Provider v1beta1 Examples

This directory contains examples for using the Discord provider's v1beta1 APIs (namespaced resources).

## Key Differences from v1alpha1

- **Namespace Isolation**: All resources are created within a specific namespace
- **API Groups**: Use `.m.` pattern (e.g., `guild.discord.m.crossplane.io/v1beta1`)
- **Multi-tenancy**: Different teams can manage Discord resources in their own namespaces
- **RBAC**: Namespace-level permissions for better access control

## Usage

1. **Create Namespace** (if not exists):
   ```bash
   kubectl apply -f namespace.yaml
   ```

2. **Create Provider Configuration** (use the same ProviderConfig as v1alpha1):
   ```bash
   kubectl apply -f ../provider-config.yaml
   ```

3. **Deploy Resources**:
   ```bash
   # Apply all v1beta1 examples
   kubectl apply -f . -n discord-resources

   # Or apply individually
   kubectl apply -f guild.yaml
   kubectl apply -f channel.yaml  # Update guildId first
   kubectl apply -f role.yaml     # Update guildId first
   kubectl apply -f webhook.yaml  # Update channelId first
   ```

## Migration from v1alpha1

v1beta1 resources can coexist with v1alpha1 resources. The underlying Discord API calls are identical, so you can gradually migrate:

1. Create new resources using v1beta1 APIs in namespaces
2. Leave existing v1alpha1 resources unchanged
3. Migrate when convenient by recreating resources with v1beta1 APIs

## Resource Dependencies

- **Guild**: Independent resource, create first
- **Channel**: Requires valid `guildId` from a Guild resource
- **Role**: Requires valid `guildId` from a Guild resource
- **Webhook**: Requires valid `channelId` from a Channel resource

## Namespace-specific RBAC Example

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  namespace: discord-resources
  name: discord-manager
rules:
- apiGroups: ["guild.discord.m.crossplane.io"]
  resources: ["guilds"]
  verbs: ["get", "list", "create", "update", "patch", "delete"]
- apiGroups: ["channel.discord.m.crossplane.io"]
  resources: ["channels"]
  verbs: ["get", "list", "create", "update", "patch", "delete"]
# ... other resource permissions
```
