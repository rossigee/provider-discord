# Discord Provider Examples

This directory contains example manifests for using the Discord Crossplane provider.

## Prerequisites

1. **Discord Bot Token**: Create a Discord application and bot at https://discord.com/developers/applications
2. **Bot Permissions**: Ensure your bot has the following permissions:
   - Manage Server
   - Manage Channels
   - Manage Roles
   - Manage Webhooks
   - Create Instant Invite
   - View Channels
   - Send Messages
   - Manage Members (for member management)
   - Applications.Commands (for application resources)

## Setup

1. Create the credentials secret:
```bash
kubectl create secret generic discord-creds \
  -n crossplane-system \
  --from-literal=token=YOUR_BOT_TOKEN_HERE
```

2. Apply the provider configuration:
```bash
kubectl apply -f providerconfig.yaml
```

## Examples

### Guild Management
- `guild.yaml` - Creates a Discord server (guild) with basic configuration

### Channel Management  
- `channel.yaml` - Creates various types of Discord channels:
  - Text channel with topic and rate limiting
  - Voice channel with bitrate and user limits
  - Category channel for organization

### Role Management
- `role.yaml` - Creates Discord roles with permissions and properties

### Webhook Management
- `webhook.yaml` - Creates webhooks for CI/CD integration and automated messaging

### Invite Management
- `invite.yaml` - Creates server invitations with expiration and usage controls

### Member Management
- `member.yaml` - Manages Discord guild members, roles, and permissions
- Use for assigning roles, setting nicknames, and managing member state

### User Management
- `user.yaml` - Retrieves Discord user information and profiles
- Support for both user lookup and current user (@me) operations

### Application Management
- `application.yaml` - Manages Discord application/bot configuration
- Handles OAuth2 settings, installation parameters, and app metadata

### Integration Management
- `integration.yaml` - Observes third-party service integrations
- Monitor connected services like Twitch, YouTube, Spotify, etc.

## Usage

1. Install the provider:
```bash
kubectl apply -f https://github.com/rossigee/provider-discord/releases/latest/download/provider.yaml
```

2. Apply provider configuration:
```bash
kubectl apply -f examples/providerconfig.yaml
```

3. Create Discord resources:
```bash
kubectl apply -f examples/guild.yaml
kubectl apply -f examples/channel.yaml
kubectl apply -f examples/role.yaml
kubectl apply -f examples/webhook.yaml
kubectl apply -f examples/invite.yaml
kubectl apply -f examples/member.yaml
kubectl apply -f examples/user.yaml
kubectl apply -f examples/application.yaml
kubectl apply -f examples/integration.yaml
```

4. Check resource status:
```bash
kubectl get guild,channel,role,webhook,invite,member,user,application,integration
kubectl describe guild example-guild
kubectl describe webhook example-webhook
kubectl describe member example-member
kubectl describe application example-app
```

## Notes

- Replace `GUILD_ID_HERE` in channel examples with actual guild IDs
- Bot must be added to guilds before managing channels, members, and integrations
- Member management requires "Manage Members" permission and appropriate role hierarchy
- User resources support both user lookup (@everyone) and current user (@me) operations
- Application resources can only modify current application (@me), not arbitrary applications
- Integration resources are read-only for monitoring connected services
- Some Discord features require specific server boost levels
- Rate limits apply to Discord API calls
