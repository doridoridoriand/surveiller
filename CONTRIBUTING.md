# Contributing to deadman-go

Thank you for your interest in contributing to deadman-go! This document provides guidelines for contributing to the project.

## Code of Conduct

Please be respectful and constructive in all interactions. We aim to maintain a welcoming environment for all contributors.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR_USERNAME/deadman-go.git`
3. Create a feature branch: `git checkout -b feature/your-feature-name`
4. Make your changes
5. Test your changes: `make test`
6. Commit your changes: `git commit -m "Add your feature"`
7. Push to your fork: `git push origin feature/your-feature-name`
8. Create a Pull Request

## Development Setup

### Prerequisites

- Go 1.24.0 or later
- Make
- Git

### Building and Testing

```bash
# Build the project
make build

# Run unit tests
make test

# Run property-based tests
make test-prop

# Run all tests
make test-all

# Clean build artifacts
make clean
```

## Contribution Guidelines

### Code Style

- Follow standard Go conventions (`gofmt`, `golint`)
- Use meaningful variable and function names
- Add comments for exported functions and complex logic
- Keep functions focused and reasonably sized

### Platform Support

When contributing platform-specific fixes or improvements:

#### Linux (Primary Platform)
- All changes must include tests
- CI/CD pipeline provides comprehensive testing
- Performance benchmarks required for core changes

#### macOS and Windows (Experimental Platforms)
- Basic functionality testing appreciated but not required
- Community testing and feedback valuable
- Platform-specific issues should be clearly documented
- Consider fallback mechanisms for platform limitations

### Testing Guidelines

- Unit tests required for all new functionality
- Property-based tests for core ping logic
- Integration tests for configuration parsing
- Manual testing on experimental platforms welcome but not mandatory

### Compatibility

- Maintain compatibility with original deadman configuration format
- Preserve existing CLI interface behavior
- Document any breaking changes clearly

### Documentation

- Update README.md for user-facing changes
- Add or update code comments
- Update design documentation in `docs/` if architecture changes

## Types of Contributions

### Bug Reports

When reporting bugs, please include:
- Operating system and version
- Go version (if building from source)
- deadman-go version
- Configuration file (sanitized)
- Steps to reproduce
- Expected vs actual behavior
- Error messages or logs

#### For macOS/Windows (Experimental Platforms)
Additionally include:
- Whether running with elevated privileges (sudo/administrator)
- ICMP vs external ping command behavior
- Network configuration details if relevant

### Feature Requests

For feature requests, please:
- Check existing issues first
- Describe the use case clearly
- Explain how it fits with project goals
- Consider backward compatibility

### Code Contributions

We welcome contributions for:
- Bug fixes
- Performance improvements
- New features (discuss in issue first)
- Documentation improvements
- Test coverage improvements

## Pull Request Process

1. Ensure your PR has a clear description
2. Reference related issues using `#issue-number`
3. Include tests for new functionality
4. Update documentation as needed
5. Ensure CI passes
6. Be responsive to review feedback

## Project Structure

```
├── internal/
│   ├── cli/        # Command-line interface
│   ├── config/     # Configuration parsing
│   ├── metrics/    # Prometheus metrics
│   ├── ping/       # Ping implementations
│   ├── scheduler/  # Monitoring scheduler
│   ├── state/      # State management
│   └── ui/         # Terminal UI
├── example/        # Example configurations
├── docs/           # Design documentation
└── main.go         # Application entry point
```

## Release Process

Releases are managed by project maintainers:
1. Version bump in `main.go`
2. Update CHANGELOG.md
3. Create release tag
4. Build and publish binaries

## Questions?

Feel free to open an issue for questions about contributing or project direction.

Thank you for contributing to deadman-go!