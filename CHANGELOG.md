# Changelog

All notable changes to deadman-go will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.0.2] - 2025-12-27

### Changed
- Updated dependencies to latest versions
  - golang.org/x/net: v0.38.0
  - golang.org/x/sys: v0.39.0
  - golang.org/x/term: v0.38.0
  - golang.org/x/text: v0.32.0
- Updated Go toolchain to 1.24.11

## [0.0.1] - 2025-01-XX

### Added
- Initial Go implementation of deadman ping monitoring tool
- Configuration compatibility with original deadman project
- Terminal UI with real-time status display and RTT bar graphs
- Concurrent ping monitoring with configurable limits
- Prometheus metrics export support (per-target, aggregated, or both modes)
- Fallback to external ping command when ICMP privileges unavailable
- SIGHUP configuration reload support
- CLI options to override configuration values
- Cross-platform support (Linux, macOS, Windows)

### Features
- ICMP ping monitoring with RTT measurement and status tracking
- Group-based target organization with `---` separators
- Configurable ping intervals, timeouts, and concurrency limits
- Status classification (OK/WARN/DOWN/UNKNOWN) based on RTT and consecutive failures
- Real-time terminal interface with customizable RTT bar scale
- Optional text-only output mode for headless environments
- HTTP metrics endpoint for Prometheus integration
- Automated release management with GitHub Actions

### Technical
- Built with Go 1.24.0 for modern language features
- Goroutine-based concurrent monitoring architecture
- Property-based testing for core functionality validation
- Modular design for easy extension and maintenance
- Single binary distribution with no external dependencies

### Documentation
- Comprehensive README with installation and usage instructions
- Contributing guidelines for open source development
- Security policy and best practices documentation
- Automated release process documentation
- Acknowledgment to original deadman project by upa

---

## Acknowledgments

This project is inspired by and maintains configuration compatibility with the original [deadman](https://github.com/upa/deadman) tool created by [upa](https://github.com/upa). We thank the original author for creating such a useful monitoring tool that has served the network monitoring community well, particularly in conference and event network environments like Interop Tokyo ShowNet.