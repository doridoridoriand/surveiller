# Changelog

All notable changes to deadman-go will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial Go implementation of deadman ping monitoring tool
- Configuration compatibility with original deadman
- Terminal UI with real-time status display
- Concurrent ping monitoring with configurable limits
- Prometheus metrics export support
- Fallback to external ping command
- SIGHUP configuration reload support
- CLI options to override configuration values

### Features
- ICMP ping monitoring with RTT measurement
- Group-based target organization
- Configurable ping intervals and timeouts
- Status tracking (OK/WARN/DOWN) based on consecutive failures
- Real-time terminal interface with bar graphs
- Optional text-only output mode
- Metrics export for Prometheus integration

### Technical
- Built with Go 1.24.0
- Uses goroutines for concurrent monitoring
- Property-based testing for core functionality
- Modular architecture for extensibility

## [0.1.0] - TBD

### Added
- Initial release
- Core ping monitoring functionality
- Terminal user interface
- Configuration file parsing
- Basic Prometheus metrics

---

## Acknowledgments

This project is inspired by the original [deadman](https://github.com/upa/deadman) tool by [upa](https://github.com/upa), maintaining configuration compatibility while providing the benefits of Go's concurrency and single-binary distribution.