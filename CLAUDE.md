# Provider Discord Planning

## Discord API Resources to Support

### Core Resources
1. **Guild** - Discord server management
   - Create/delete guilds
   - Guild settings (name, icon, region, etc.)
   - Guild features and permissions

2. **Channel** - Text/voice channel management
   - Text channels
   - Voice channels  
   - Category channels
   - Channel permissions and settings

3. **Role** - Permission management
   - Role creation/deletion
   - Permission assignment
   - Role hierarchy management

4. **User/Member** - User management within guilds
   - Guild member management
   - Role assignment to members
   - User permissions

5. **Webhook** - Automated message posting
   - Webhook creation/deletion
   - Webhook configuration
   - Message posting via webhooks

6. **Invite** - Server invitation management
   - Create/revoke invites
   - Invite expiration settings
   - Usage tracking

### Secondary Resources (Future)
- **Application/Bot** - Bot application management
- **Integration** - Third-party integrations
- **AuditLog** - Audit log access
- **Emoji** - Custom emoji management
- **Sticker** - Custom sticker management

## Provider Architecture

### API Groups
- `discord.golder.tech/v1alpha1` - Main API group
- Resources will follow Crossplane patterns with:
  - Spec/Status structure
  - Managed resource lifecycle
  - Connection/credential management

### Authentication
- Bot tokens for API access
- OAuth2 for user-level operations
- Webhook tokens for webhook management

### Controllers
Each resource type will have dedicated controllers following Crossplane patterns:
- `GuildController`
- `ChannelController`
- `RoleController`
- `WebhookController`
- etc.

## Implementation Strategy
1. Start with core resources (Guild, Channel, Role)
2. Implement webhook support for automation
3. Add user/member management
4. Extend to secondary resources as needed

## Use Cases
- GitOps-managed Discord server configuration
- Automated channel creation for projects
- Role-based access control automation
- Integration with CI/CD for notifications
- Multi-environment Discord server management