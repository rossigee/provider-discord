# Contributing to Provider Discord

Thank you for your interest in contributing to Provider Discord! This document provides guidelines and information for contributors.

## Code of Conduct

This project adheres to the [Crossplane Code of Conduct](https://github.com/crossplane/crossplane/blob/master/CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## Getting Started

### Prerequisites

- Go 1.24.5 or later
- Docker
- Kind (for integration testing)
- kubectl
- Make

### Development Environment Setup

1. Fork and clone the repository:
```bash
git clone https://github.com/YOUR_USERNAME/provider-discord.git
cd provider-discord
```

2. Install dependencies:
```bash
make vendor
```

3. Install pre-commit hooks (recommended):
```bash
pip install pre-commit
pre-commit install
```

4. Verify your setup:
```bash
make test
```

## Development Workflow

### Making Changes

1. Create a new branch for your feature/fix:
```bash
git checkout -b feature/your-feature-name
```

2. Make your changes following the coding standards below

3. Add or update tests for your changes

4. Ensure all tests pass:
```bash
make test
make lint
```

5. Commit your changes with a descriptive commit message

6. Push your branch and create a pull request

### Coding Standards

#### Go Code
- Follow standard Go formatting (`go fmt`)
- Use `goimports` for import organization
- Write meaningful variable and function names
- Add comments for exported functions and complex logic
- Ensure code passes `golangci-lint`

#### API Design
- Follow Kubernetes API conventions
- Use proper kubebuilder annotations
- Include validation rules where appropriate
- Provide meaningful status fields
- Follow Crossplane patterns for managed resources

#### Testing
- Write unit tests for all new functionality
- Include integration tests for controller logic
- Mock external dependencies appropriately
- Aim for good test coverage (>80%)

### Project Structure

```
provider-discord/
├── apis/                   # API definitions
│   ├── v1beta1/           # Provider config APIs
│   ├── guild/v1alpha1/    # Guild resource APIs
│   └── channel/v1alpha1/  # Channel resource APIs
├── cmd/provider/          # Main provider entry point
├── examples/              # Example manifests
├── internal/              # Internal code
│   ├── clients/           # Discord API client
│   └── controller/        # Crossplane controllers
├── package/               # Crossplane package definitions
└── test/                  # Test files and fixtures
```

## Types of Contributions

### Bug Reports

When filing a bug report, please include:
- Clear description of the issue
- Steps to reproduce
- Expected vs actual behavior
- Environment information (provider version, Kubernetes version)
- Relevant logs or error messages

### Feature Requests

For feature requests, please include:
- Clear description of the desired functionality
- Use case and motivation
- Proposed implementation approach (if any)
- Consider if it fits with the project's goals

### Documentation

Documentation improvements are always welcome:
- Fix typos or unclear explanations
- Add examples or tutorials
- Improve API documentation
- Add troubleshooting guides

### Code Contributions

#### Adding New Resources

When adding new Discord resources:

1. Create API types in `apis/RESOURCE/v1alpha1/`
2. Implement Discord client methods in `internal/clients/`
3. Create controller in `internal/controller/RESOURCE/`
4. Add to controller registration in `internal/controller/controller.go`
5. Update API registration in `apis/apis.go`
6. Add examples in `examples/`
7. Write comprehensive tests
8. Update documentation

#### Extending Existing Resources

When extending existing resources:
- Maintain backward compatibility
- Update validation rules appropriately
- Add tests for new fields/functionality
- Update examples and documentation

## Testing

### Unit Tests
```bash
# Run all unit tests
make test

# Run tests with coverage
make test.cover

# Run tests for specific package
go test ./internal/clients/...
```

### Integration Tests
```bash
# Setup Kind cluster and run integration tests
make integration-test
```

### Manual Testing

For manual testing:

1. Build and load the provider:
```bash
make docker-build
kind load docker-image provider-discord:latest
```

2. Install in a test cluster:
```bash
kubectl apply -f examples/providerconfig.yaml
```

3. Test with your Discord bot token

## Pull Request Process

### Before Submitting

- [ ] Code compiles without warnings
- [ ] All tests pass (`make test`)
- [ ] Code passes linting (`make lint`)
- [ ] API changes include proper validation
- [ ] New features include tests
- [ ] Documentation is updated
- [ ] Examples are provided for new features

### Pull Request Description

Include in your PR description:
- Summary of changes
- Motivation and context
- Type of change (bug fix, feature, docs, etc.)
- Testing performed
- Screenshots (if applicable)

### Review Process

1. Automated checks must pass
2. At least one maintainer review required
3. Address any feedback promptly
4. Maintain a clean commit history

## Release Process

Releases are handled by maintainers:

1. Version bump in appropriate files
2. Update CHANGELOG.md
3. Create and push version tag
4. GitHub Actions handles the rest

## Community

### Communication Channels

- **Slack**: Join [Crossplane Slack](https://slack.crossplane.io/) and find us in `#provider-discord`
- **GitHub Discussions**: For longer-form discussions and questions
- **GitHub Issues**: For bug reports and feature requests

### Getting Help

If you need help:
1. Check existing documentation and examples
2. Search GitHub issues for similar problems
3. Ask in Slack `#provider-discord` channel
4. Create a GitHub discussion

## Maintainers

Current maintainers:
- @maintainer1
- @maintainer2

## Recognition

Contributors are recognized in:
- Release notes
- CONTRIBUTORS.md file
- Annual contributor highlights

Thank you for contributing to Provider Discord! 🚀
