# Discord Bot Setup Guide

This guide walks you through setting up a Discord bot for use with the Crossplane Discord Provider.

## Prerequisites

- Discord account
- Administrator permissions on target Discord server
- Basic understanding of Discord permissions system

## Step 1: Create a Discord Application

1. **Navigate to Discord Developer Portal**
   - Go to https://discord.com/developers/applications
   - Log in with your Discord account

2. **Create New Application**
   - Click "New Application"
   - Enter application name (e.g., "Crossplane Bot")
   - Accept terms and click "Create"

3. **Configure Application**
   - Add description: "Crossplane provider for Discord server management"
   - Add icon/avatar (optional)
   - Save changes

## Step 2: Create a Bot User

1. **Navigate to Bot Section**
   - In your application, click "Bot" in the left sidebar
   - Click "Add Bot"
   - Confirm by clicking "Yes, do it!"

2. **Configure Bot Settings**
   - **Username**: Choose a clear name (e.g., "Crossplane-Discord")
   - **Public Bot**: Disable (uncheck) for security
   - **Requires OAuth2 Code Grant**: Disable (uncheck)
   - **Message Content Intent**: Enable if you plan to manage messages
   - **Server Members Intent**: Enable if you plan to manage members
   - **Presence Intent**: Enable for user status management

3. **Copy Bot Token**
   - Click "Reset Token" to generate a new token
   - Copy the token immediately (you won't see it again)
   - Store securely - this is your `DISCORD_BOT_TOKEN`

## Step 3: Configure Bot Permissions

### Required Permissions

The bot needs these permissions for provider functionality:

#### Server Management
- **Manage Server** - Required for guild operations
- **View Channels** - Required for resource observation
- **Manage Channels** - Required for channel operations
- **Manage Roles** - Required for role operations

#### Member & User Management
- **Manage Members** - Required for member management operations
- **Create Instant Invite** - For invite management
- **Kick Members** - For member management
- **Ban Members** - For moderation features
- **Manage Nicknames** - For user management
- **View Audit Log** - For audit operations

#### Application & Integration Management
- **Applications.Commands** - For application resource management
- **Read Message History** - For integration monitoring

#### Optional Permissions (based on use case)
- **Manage Messages** - For message operations
- **Use Slash Commands** - For bot application features
- **Manage Events** - For event-based integrations

### Permission Calculator

Use Discord's permission calculator:
1. Go to https://discordapi.com/permissions.html
2. Select required permissions
3. Copy the permission integer for your configurations

**Minimum Required Permissions Integer**: `402653200`
- Manage Server (MANAGE_GUILD): 32
- View Channels (VIEW_CHANNEL): 1024
- Manage Channels (MANAGE_CHANNELS): 16
- Manage Roles (MANAGE_ROLES): 268435456
- Manage Members (KICK_MEMBERS): 2
- Manage Nicknames (MANAGE_NICKNAMES): 134217728
- View Audit Log (VIEW_AUDIT_LOG): 128

**Extended Permissions Integer (with optional features)**: `402719744`
- Includes above plus:
- Create Instant Invite (CREATE_INSTANT_INVITE): 1
- Ban Members (BAN_MEMBERS): 4
- Read Message History (READ_MESSAGE_HISTORY): 65536

## Step 4: Invite Bot to Server

1. **Generate OAuth2 URL**
   - In Discord Developer Portal, go to "OAuth2" → "URL Generator"
   - **Scopes**: Select "bot"
   - **Bot Permissions**: Select required permissions from Step 3
   - Copy the generated URL

2. **Invite Bot**
   - Open the generated URL in browser
   - Select target Discord server
   - Authorize the bot
   - Complete CAPTCHA if prompted

3. **Verify Installation**
   - Bot should appear in server member list
   - Bot should have appropriate role with permissions
   - Test basic functionality (if bot is offline, that's normal)

## Step 5: Configure Provider

### Create Kubernetes Secret

```bash
kubectl create secret generic discord-creds \
  -n crossplane-system \
  --from-literal=token=YOUR_BOT_TOKEN_HERE
```

### Create ProviderConfig

```yaml
apiVersion: discord.crossplane.io/v1beta1
kind: ProviderConfig
metadata:
  name: default
spec:
  credentials:
    source: Secret
    secretRef:
      namespace: crossplane-system
      name: discord-creds
      key: token
  baseURL: "https://discord.com/api/v10"
```

## Security Best Practices

### Bot Token Security
- **Never commit tokens to version control**
- **Use environment variables or Kubernetes secrets**
- **Rotate tokens regularly**
- **Monitor token usage in Discord Developer Portal**

### Permission Management
- **Principle of least privilege** - grant minimum required permissions
- **Regular permission audits** - review and update as needed
- **Role hierarchy** - ensure bot role is positioned correctly

### Server Security
- **Enable 2FA** on Discord account
- **Server verification** - enable server verification features
- **Audit logs** - monitor Discord audit logs for bot actions
- **Rate limiting** - provider includes built-in rate limiting

## Troubleshooting

### Common Issues

#### "Missing Permissions" Error
- **Cause**: Bot lacks required permissions
- **Solution**: Check bot role permissions in Discord server settings
- **Check**: Ensure bot role is above managed resource roles in hierarchy

#### "Invalid Token" Error
- **Cause**: Token expired, regenerated, or malformed
- **Solution**: Generate new token and update Kubernetes secret
- **Verify**: Token format should be `MTAxNTYyNjA4MDQwNzU5OTIzNQ.GkQTzg.example`

#### "Rate Limited" Error
- **Cause**: Discord API rate limits exceeded
- **Solution**: Provider includes automatic retry logic
- **Check**: Review provider logs for rate limit patterns

#### "Bot Not Responding"
- **Cause**: Provider not running or misconfigured
- **Solution**: Check provider pod logs and status
- **Verify**: ProviderConfig and Secret configuration

### Verification Commands

Test bot functionality:

```bash
# Check provider status
kubectl get providers

# Check provider config
kubectl describe providerconfig default

# Check provider logs
kubectl logs -n crossplane-system deployment/provider-discord

# Test resource creation
kubectl apply -f examples/guild.yaml
kubectl get guilds
```

## Discord API Limits

### Rate Limits
- **Global**: 50 requests per second
- **Per-route**: Varies by endpoint (typically 5-10/second)
- **Bot-specific**: Additional limits for bot actions

### Resource Limits
- **Guilds**: 100 servers per bot (can be increased)
- **Channels**: 500 channels per server
- **Roles**: 250 roles per server
- **Members**: Varies by server boost level

## Advanced Configuration

### Custom API Endpoint
For Discord Enterprise or self-hosted instances:

```yaml
spec:
  baseURL: "https://your-discord-api.company.com/api/v10"
```

### Webhook Configuration
For advanced monitoring and automation:

```yaml
# Webhook resource example
apiVersion: webhook.discord.crossplane.io/v1alpha1
kind: Webhook
metadata:
  name: monitoring-webhook
spec:
  forProvider:
    channelId: "123456789"
    name: "Crossplane Alerts"
    avatar: "https://example.com/avatar.png"
```

## Support Resources

- **Discord Provider Documentation**: [GitHub Repository](https://github.com/rossigee/provider-discord)
- **Discord Developer Documentation**: https://discord.com/developers/docs
- **Discord API Reference**: https://discord.com/developers/docs/reference
- **Crossplane Documentation**: https://crossplane.io/docs
- **Community Support**: [Crossplane Slack #providers](https://slack.crossplane.io/)

## Legal and Compliance

- **Discord Terms of Service**: https://discord.com/terms
- **Discord Developer Terms**: https://discord.com/developers/docs/legal
- **Bot Guidelines**: https://discord.com/developers/docs/policies-and-agreements/developer-terms-of-service
- **Privacy Policy**: Ensure your bot usage complies with privacy regulations