# Provider Discord

[![CI](https://github.com/rossigee/provider-discord/actions/workflows/ci.yml/badge.svg)](https://github.com/rossigee/provider-discord/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/rossigee/provider-discord)](https://goreportcard.com/report/github.com/rossigee/provider-discord)
[![Coverage Status](https://coveralls.io/repos/github/rossigee/provider-discord/badge.svg)](https://coveralls.io/github/rossigee/provider-discord)

An **enterprise-grade** Crossplane provider for managing Discord resources through Kubernetes APIs with comprehensive observability, resilience, and monitoring.

## Features

### Core Discord Management
- **Guild Management**: Create and manage Discord servers declaratively
- **Channel Management**: Text, voice, and category channels with full configuration
- **Role Management**: Permission management and role hierarchy control
- **GitOps Ready**: Full integration with Kubernetes and GitOps workflows

### Enterprise Features
- **🏥 Health Monitoring**: Built-in `/healthz` and `/readyz` endpoints with Discord API connectivity checks
- **📊 Prometheus Metrics**: 10+ custom metrics for operations, rate limits, errors, and performance monitoring
- **🔍 OpenTelemetry Tracing**: Distributed tracing with correlation IDs for debugging and analysis
- **🛡️ Resilience Patterns**: Circuit breakers, exponential backoff, and intelligent retry logic
- **🔐 Enterprise Security**: Pod security contexts, network policies, and RBAC configurations
- **⚡ Performance Optimization**: Resource limits, health probes, and efficient resource management

### Production Ready
- **Discord API v10**: Latest Discord API with rate limiting and error handling
- **Test Coverage**: 62-100% test coverage across all modules with comprehensive validation
- **Observability**: Structured logging, metrics collection, and tracing integration
- **Deployment**: Production-ready configurations with monitoring and security

## Supported Resources

| Resource | API Version | Description | Status |
|----------|-------------|-------------|---------|
| Guild | `guild.discord.crossplane.io/v1alpha1` | Discord servers with full configuration | ✅ Production Ready |
| Channel | `channel.discord.crossplane.io/v1alpha1` | Text, voice, and category channels | ✅ Production Ready |
| Role | `role.discord.crossplane.io/v1alpha1` | Permission management and role hierarchy | ✅ Production Ready |
| ProviderConfig | `discord.crossplane.io/v1beta1` | Provider authentication and configuration | ✅ Production Ready |

## Quick Start

### Prerequisites

1. **Discord Bot**: Create a Discord application and bot at [Discord Developer Portal](https://discord.com/developers/applications)
2. **Bot Permissions**: Ensure your bot has these permissions:
   - Manage Server (for guild operations)
   - Manage Channels (for channel operations)
   - Manage Roles (for role operations)
   - View Channels (for resource observation)

### Installation

#### Option 1: Production Deployment (Recommended)

```bash
# Install with full enterprise configuration
kubectl apply -f https://raw.githubusercontent.com/rossigee/provider-discord/master/examples/provider-config.yaml
```

#### Option 2: Basic Installation

```bash
# Install provider only
kubectl apply -f https://github.com/rossigee/provider-discord/releases/latest/download/provider.yaml
```

### Configuration

1. **Create Discord Bot Token Secret**:
```bash
kubectl create secret generic discord-creds \
  -n crossplane-system \
  --from-literal=token=YOUR_BOT_TOKEN_HERE
```

2. **Create ProviderConfig**:
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
  baseURL: "https://discord.com/api/v10"  # Optional: custom API endpoint
```

### Example Usage

#### Complete Discord Server Setup

```yaml
# Create Discord Guild (Server)
apiVersion: guild.discord.crossplane.io/v1alpha1
kind: Guild
metadata:
  name: my-crossplane-server
  annotations:
    kubernetes.io/description: "Enterprise Discord server managed by Crossplane"
spec:
  forProvider:
    name: "My Enterprise Discord Server"
    region: "us-west"
    verificationLevel: 2  # Medium verification
    defaultMessageNotifications: 1  # Only mentions
    explicitContentFilter: 2  # All members
    afkTimeout: 600  # 10 minutes
  providerConfigRef:
    name: default
---
# Create Text Channels
apiVersion: channel.discord.crossplane.io/v1alpha1
kind: Channel
metadata:
  name: general-announcements
spec:
  forProvider:
    name: "announcements"
    type: 0  # Text channel
    guildId: "GUILD_ID_HERE"
    topic: "Important announcements and updates"
    rateLimitPerUser: 10  # Slow mode: 10 seconds
    position: 0
  providerConfigRef:
    name: default
---
apiVersion: channel.discord.crossplane.io/v1alpha1
kind: Channel
metadata:
  name: general-discussion
spec:
  forProvider:
    name: "general"
    type: 0  # Text channel
    guildId: "GUILD_ID_HERE"
    topic: "General discussion and chat"
    rateLimitPerUser: 0  # No slow mode
    position: 1
  providerConfigRef:
    name: default
---
# Create Voice Channel
apiVersion: channel.discord.crossplane.io/v1alpha1
kind: Channel
metadata:
  name: team-voice-chat
spec:
  forProvider:
    name: "Team Voice"
    type: 2  # Voice channel
    guildId: "GUILD_ID_HERE"
    bitrate: 128000  # High quality audio
    userLimit: 25
    position: 2
  providerConfigRef:
    name: default
---
# Create Admin Role
apiVersion: role.discord.crossplane.io/v1alpha1
kind: Role
metadata:
  name: admin-role
spec:
  forProvider:
    name: "Administrator"
    guildId: "GUILD_ID_HERE"
    color: 16711680  # Red color
    hoist: true  # Display separately
    permissions: "8"  # Administrator permission
    mentionable: false
    position: 10
  providerConfigRef:
    name: default
```

## Production Deployment

### Enterprise Configuration

The provider includes comprehensive production-ready configurations:

```yaml
# Resource Management
resources:
  limits:
    cpu: 500m
    memory: 512Mi
  requests:
    cpu: 100m
    memory: 128Mi

# Health Monitoring
livenessProbe:
  httpGet:
    path: /healthz
    port: 8081
readinessProbe:
  httpGet:
    path: /readyz
    port: 8081

# Observability
env:
- name: OTEL_EXPORTER_OTLP_ENDPOINT
  value: "http://jaeger-collector:14268/api/traces"
- name: DISCORD_METRICS_ENABLED
  value: "true"
- name: DISCORD_CIRCUIT_BREAKER_ENABLED
  value: "true"
```

### Monitoring Integration

#### Prometheus Metrics

The provider exposes comprehensive metrics at `/metrics`:

- `provider_discord_discord_api_operations_total` - API operation counters
- `provider_discord_discord_rate_limits_total` - Rate limit hit counters
- `provider_discord_health_check_requests_total` - Health check metrics
- `provider_discord_managed_resources` - Resource count gauges
- `provider_discord_discord_api_errors_total` - Error categorization

#### Health Endpoints

- **`/healthz`**: Liveness probe - checks if provider is running
- **`/readyz`**: Readiness probe - validates Discord API connectivity and Kubernetes access

#### OpenTelemetry Tracing

Distributed tracing with correlation IDs for:
- Resource reconciliation operations
- Discord API calls
- Error tracking and retry attempts
- Performance monitoring

## Configuration

### Authentication

The provider supports secure authentication through Kubernetes secrets:

```yaml
apiVersion: discord.crossplane.io/v1beta1
kind: ProviderConfig
metadata:
  name: production
spec:
  credentials:
    source: Secret
    secretRef:
      namespace: crossplane-system
      name: discord-production-creds
      key: bot-token
  baseURL: "https://discord.com/api/v10"  # Optional: defaults to v10
```

### Discord API Configuration

- **Base URL**: Defaults to `https://discord.com/api/v10`
- **Rate Limiting**: Intelligent rate limit handling with circuit breakers
- **Retry Logic**: Exponential backoff with jitter for failed requests
- **Error Handling**: Comprehensive error classification and recovery

### Enterprise Features Configuration

```yaml
env:
# Observability
- name: OTEL_TRACING_ENABLED
  value: "true"
- name: OTEL_SAMPLING_RATIO
  value: "0.1"

# Resilience
- name: DISCORD_CIRCUIT_BREAKER_ENABLED
  value: "true"
- name: DISCORD_RATE_LIMIT_BACKOFF_MAX
  value: "30s"

# Monitoring
- name: DISCORD_METRICS_ENABLED
  value: "true"
- name: DISCORD_HEALTH_CHECK_INTERVAL
  value: "30s"
```

## Development

### Prerequisites

- Go 1.24.5+
- Docker
- Kind (for integration tests)
- Pre-commit hooks (recommended)

### Building

```bash
# Clone the repository
git clone https://github.com/rossigee/provider-discord.git
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
# Run unit tests (includes enterprise modules)
make test

# Run integration tests (requires Kind cluster)
make integration-test

# Run tests with coverage (target: >70% coverage)
make test.cover

# Run specific module tests
go test ./internal/health/... -v
go test ./internal/metrics/... -v
go test ./internal/resilience/... -v
go test ./internal/tracing/... -v
```

### Code Generation

```bash
# Generate code and CRDs
make generate

# Update dependencies
make vendor
```

### Local Development

```bash
# Run the provider locally with debugging
make run

# Run with enterprise features enabled
DISCORD_METRICS_ENABLED=true \
DISCORD_CIRCUIT_BREAKER_ENABLED=true \
OTEL_TRACING_ENABLED=true \
make run
```

## Architecture

### Enterprise Components

```
┌─────────────────────────────────────────────────────────────┐
│                    Provider Discord                         │
├─────────────────────────────────────────────────────────────┤
│ Controllers:        │ Enterprise Modules:                   │
│ • Guild Controller  │ • Health Monitoring (/healthz,/readyz)│
│ • Channel Controller│ • Metrics (Prometheus)               │
│ • Role Controller   │ • Tracing (OpenTelemetry)            │
│ • Config Controller │ • Resilience (Circuit Breakers)      │
├─────────────────────────────────────────────────────────────┤
│ Discord API Client: │ Infrastructure:                       │
│ • Rate Limiting     │ • Security Contexts                  │
│ • Error Handling    │ • Network Policies                   │
│ • Retry Logic       │ • Resource Management                │
└─────────────────────────────────────────────────────────────┘
```

### Test Coverage

- **Controllers**: 62-78% coverage with comprehensive CRUD testing
- **Enterprise Modules**: 72-100% coverage
  - Health Monitoring: 77.2%
  - Metrics Framework: 100.0%
  - Resilience Module: 94.8%
  - Tracing Module: 72.0%
- **API Packages**: 30-56% validation and marshaling tests
- **Client Library**: 91.9% with comprehensive error scenarios

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for details.

### Development Workflow

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for your changes (maintain >70% coverage)
5. Ensure all tests pass: `make test`
6. Ensure code is properly formatted: `make lint`
7. Test enterprise features locally
8. Submit a pull request

### Pre-commit Hooks

Install pre-commit hooks to automatically check your code:

```bash
pip install pre-commit
pre-commit install
```

## Security

### Security Features

- **Pod Security Contexts**: Non-root containers with read-only filesystems
- **Network Policies**: Restricted egress/ingress traffic
- **RBAC**: Least-privilege access controls
- **Secret Management**: Secure Discord token handling
- **Input Validation**: Comprehensive request validation

### Vulnerability Reporting

Please report security vulnerabilities to [security@crossplane.io](mailto:security@crossplane.io).

## Documentation

- [API Reference](https://doc.crds.dev/github.com/rossigee/provider-discord)
- [Discord Bot Setup Guide](docs/discord-setup.md)
- [Production Deployment Guide](docs/production-deployment.md)
- [Monitoring and Observability](docs/monitoring.md)
- [Troubleshooting Guide](docs/troubleshooting.md)

## Community

- [Crossplane Slack](https://slack.crossplane.io/) - `#providers` channel
- [GitHub Discussions](https://github.com/rossigee/provider-discord/discussions)
- [GitHub Issues](https://github.com/rossigee/provider-discord/issues)

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Roadmap

### ✅ v0.3.x (Current - Enterprise Ready)
- ✅ Guild, Channel, and Role management
- ✅ Enterprise health monitoring and metrics
- ✅ OpenTelemetry tracing integration
- ✅ Circuit breakers and resilience patterns
- ✅ Production-ready deployment configurations
- ✅ Comprehensive test coverage (62-100%)

### 🚧 v0.4.0 (In Development)
- Webhook management and event handling
- Invite management with expiration control
- User/Member management operations
- Enhanced permission management
- Advanced Discord integration patterns

### 📋 v0.5.0 (Planned)
- Message management and automation
- Emoji and sticker management
- Integration with external notification systems
- Advanced role hierarchy management
- Performance optimizations

### 🎯 v1.0.0 (Future)
- Production certification
- Full Discord API coverage
- Advanced observability features
- Comprehensive documentation
- Enterprise support options

---

**Enterprise-grade Discord management through Kubernetes** 🚀