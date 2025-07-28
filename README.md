# Provider Discord

[![CI](https://github.com/crossplane-contrib/provider-discord/actions/workflows/ci.yml/badge.svg)](https://github.com/crossplane-contrib/provider-discord/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/crossplane-contrib/provider-discord)](https://goreportcard.com/report/github.com/crossplane-contrib/provider-discord)
[![Coverage Status](https://coveralls.io/repos/github/crossplane-contrib/provider-discord/badge.svg)](https://coveralls.io/github/crossplane-contrib/provider-discord)

A Crossplane provider for managing Discord resources through Kubernetes.

## Features

- **Guild Management**: Create and manage Discord servers declaratively
- **Channel Management**: Manage text, voice, and category channels
- **GitOps Ready**: Full integration with Kubernetes and GitOps workflows
- **Discord API v10**: Uses the latest Discord API version
- **Crossplane Native**: Follows all Crossplane provider patterns and conventions

## Supported Resources

| Resource | API Version | Description |
|----------|-------------|-------------|
| Guild | `guild.discord.golder.tech/v1alpha1` | Discord servers with configuration |
| Channel | `channel.discord.golder.tech/v1alpha1` | Text, voice, and category channels |
| ProviderConfig | `discord.golder.tech/v1beta1` | Provider authentication configuration |

## Quick Start

### Prerequisites

1. **Discord Bot**: Create a Discord application and bot at [Discord Developer Portal](https://discord.com/developers/applications)
2. **Bot Permissions**: Ensure your bot has these permissions:
   - Manage Server
   - Manage Channels
   - Manage Roles
   - View Channels
   - Send Messages

### Installation

1. Install the provider:
```bash
kubectl apply -f https://github.com/crossplane-contrib/provider-discord/releases/latest/download/provider.yaml
```

2. Create a secret with your Discord bot token:
```bash
kubectl create secret generic discord-creds \
  -n crossplane-system \
  --from-literal=token=YOUR_BOT_TOKEN_HERE
```

3. Create a ProviderConfig:
```yaml
apiVersion: discord.golder.tech/v1beta1
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
```

### Example Usage

#### Create a Discord Guild (Server)

```yaml
apiVersion: guild.discord.golder.tech/v1alpha1
kind: Guild
metadata:
  name: my-guild
  annotations:
    kubernetes.io/description: "My awesome Discord server"
spec:
  forProvider:
    name: "My Crossplane Guild"
    region: "us-east"
    verificationLevel: 1  # Low verification
    defaultMessageNotifications: 1  # Only mentions
    explicitContentFilter: 1  # Members without roles
    afkTimeout: 300  # 5 minutes
  providerConfigRef:
    name: default
```

#### Create Discord Channels

```yaml
apiVersion: channel.discord.golder.tech/v1alpha1
kind: Channel
metadata:
  name: general-chat
spec:
  forProvider:
    name: "general"
    type: 0  # Text channel
    guildId: "GUILD_ID_HERE"
    topic: "General discussion"
    rateLimitPerUser: 5
  providerConfigRef:
    name: default
---
apiVersion: channel.discord.golder.tech/v1alpha1
kind: Channel
metadata:
  name: voice-chat
spec:
  forProvider:
    name: "Voice Chat"
    type: 2  # Voice channel
    guildId: "GUILD_ID_HERE"
    bitrate: 64000
    userLimit: 10
  providerConfigRef:
    name: default
```

## Configuration

### Authentication

The provider uses Discord bot tokens for authentication. Store your bot token in a Kubernetes secret and reference it in the ProviderConfig.

### Discord API Configuration

- **Base URL**: Defaults to `https://discord.com/api/v10`
- **Rate Limiting**: Built-in respect for Discord API rate limits
- **Permissions**: Bot must have appropriate permissions in target guilds

## Development

### Prerequisites

- Go 1.24.5+
- Docker
- Kind (for integration tests)
- Pre-commit hooks (optional but recommended)

### Building

```bash
# Clone the repository
git clone https://github.com/crossplane-contrib/provider-discord.git
cd provider-discord

# Install dependencies
make vendor

# Build the provider
make build

# Build Docker image
make docker-build

# Build Crossplane package
make xpkg.build
```

### Testing

```bash
# Run unit tests
make test

# Run integration tests (requires Kind cluster)
make integration-test

# Run all tests with coverage
make test.cover
```

### Code Generation

```bash
# Generate code and CRDs
make generate
```

### Local Development

```bash
# Run the provider locally
make run
```

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for details.

### Development Workflow

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for your changes
5. Ensure all tests pass: `make test`
6. Ensure code is properly formatted: `make lint`
7. Submit a pull request

### Pre-commit Hooks

Install pre-commit hooks to automatically check your code:

```bash
pip install pre-commit
pre-commit install
```

## Documentation

- [API Reference](docs/api-reference.md)
- [Discord Bot Setup](docs/discord-setup.md)
- [Advanced Configuration](docs/advanced-config.md)
- [Troubleshooting](docs/troubleshooting.md)

## Community

- [Crossplane Slack](https://slack.crossplane.io/) - `#provider-discord` channel
- [GitHub Discussions](https://github.com/crossplane-contrib/provider-discord/discussions)
- [GitHub Issues](https://github.com/crossplane-contrib/provider-discord/issues)

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Security

Please report security vulnerabilities to [security@crossplane.io](mailto:security@crossplane.io).

## Roadmap

### v0.2.0 (Planned)
- Role management
- Webhook support
- Invite management
- User/Member management

### v0.3.0 (Planned)
- Message management
- Emoji/Sticker management
- Integration with external systems
- Advanced permission management

### v1.0.0 (Planned)
- Production ready
- Full Discord API coverage
- Performance optimizations
- Comprehensive documentation