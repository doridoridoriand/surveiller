# Changelog

All notable changes to surveiller will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed
- Clarify platform support levels in documentation
- Update macOS and Windows builds to experimental status
- Improve platform-specific documentation and issue reporting guidelines

### Documentation
- Add platform support matrix with clear support levels (Linux: fully supported, macOS/Windows: experimental)
- Add platform-specific notes for ICMP privileges and external ping fallback
- Update contributing guidelines with platform-specific testing requirements
- Enhance bug reporting template for experimental platforms
- Update release notes template to include platform support information

## [0.0.4] - 2025-12-30

### Added
- Comprehensive unit test coverage for ping package (External, ICMP, and Fallback pingers)
- Property-based tests for timeout handling validation
- Unit tests for CLI flags parsing and validation
- Unit tests for metrics collection and export functionality
- Test coverage tracking and improvement documentation

### Changed
- Translate Japanese comments to English in sample configuration file (surveiller.sample.conf)
- Update Go version from 1.24.11 to stable 1.23.4 in CI workflow
- Enhance clean command to remove test and module caches (-testcache, -modcache)
- Improve test separation with build tags for property tests
- Scope property test execution to ping package only for better performance

### Fixed
- Resolve CI build issues with non-existent Go version
- Fix Go module configuration by removing invalid toolchain directive
- Apply gofmt formatting fixes to test files
- Ensure all tests pass and build succeeds in CI environment

### Technical
- Achieve 49.8% overall test coverage across all packages
- Individual package coverage: CLI (97.5%), Config (83.5%), Metrics (90.4%), Ping (60.4%), Scheduler (81.0%), State (84.1%)
- Add 618 lines of comprehensive test code for ping functionality
- Add 258 lines of unit tests for CLI and metrics packages
- Implement timeout handling validation through property-based testing

## [0.0.3] - 2025-12-28

### Added
- Add RTT label and LOSS percentage display to TUI
- Display current configuration in TUI header (interval, timeout, max_concurrency, ui.scale)
- Add clean-build target to Makefile
- Add sample configuration file (surveiller.sample.conf)
- Add demo recording and README preview

### Changed
- Change threshold calculation to use average of recent 10 RTT data points
- Fallback to available data points if less than 10 are available for average calculation

### Fixed
- Improve timeout error handling in ICMP/External ping implementations
- ICMPPinger: properly detect and handle timeout errors from ReadFrom
- ExternalPinger: check for context.DeadlineExceeded to distinguish timeout errors
- Format code with gofmt

### Removed
- Remove recorded configuration files from repository
- Remove public prep checklist and quick release script

### Documentation
- Update demo GIF
- Add detailed documentation for status thresholds and UI display changes
- Document RTT-based and failure-based thresholds (25% RTT threshold, 3 consecutive failures)
- Add descriptions for UI components (RTT label, LOSS percentage)

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