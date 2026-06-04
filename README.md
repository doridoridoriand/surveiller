# surveiller (formerly deadman-go)

![surveiller](assets/running.gif)

A Go implementation of the ping monitoring tool, providing efficient host status monitoring with a terminal-based interface.

## About

surveiller is inspired by and maintains compatibility with the original [deadman](https://github.com/upa/deadman) tool by [upa](https://github.com/upa). This Go implementation offers:

- **Single binary distribution** - No Python dependencies required
- **High concurrency** - Efficient monitoring of hundreds of hosts using Go goroutines
- **Configuration compatibility** - Uses the same config format as the original deadman
- **Future extensibility** - Designed for Prometheus metrics integration

## Features

- ICMP ping monitoring with configurable intervals and timeouts
- Terminal UI with real-time status display and RTT bar graphs
- Group-based target organization with `---` separators
- Concurrent monitoring with configurable limits
- Prometheus metrics export (optional)
- Configuration hot-reload with SIGHUP
- Fallback to external ping command when ICMP privileges unavailable
- Status-based health monitoring (OK / WARN / DOWN) with configurable thresholds
- Packet loss percentage display in TUI

## Installation

### Pre-built Binaries

Download the latest release from the [releases page](https://github.com/doridoridoriand/surveiller/releases).

#### Platform Support

- **Linux (AMD64/ARM64)**: Fully supported and tested
- **macOS (Intel/Apple Silicon)**: Experimental build - basic functionality verified but not continuously tested
- **Windows (AMD64)**: Experimental build - basic functionality verified but not continuously tested

**Note**: macOS and Windows builds are provided as experimental releases. While basic functionality has been verified, these platforms are not part of our continuous testing pipeline. Community feedback and contributions for platform-specific issues are welcome.

### Package Manager Installation

Package-manager distribution is supported for `apt`, `dnf`, `brew`, and `choco`.

#### APT (Debian/Ubuntu)

```bash
VERSION="v0.0.12"
VERSION_NO_V="${VERSION#v}"
ARCH="$(dpkg --print-architecture)"
DEB_FILE="surveiller_${VERSION_NO_V}_${ARCH}.deb"
curl -fsSL -o "/tmp/${DEB_FILE}" \
  "https://github.com/doridoridoriand/surveiller/releases/download/${VERSION}/${DEB_FILE}"
sudo apt install -y "/tmp/${DEB_FILE}"
```

#### DNF (Fedora/RHEL family)

```bash
VERSION="v0.0.12"
VERSION_NO_V="${VERSION#v}"
ARCH="$(uname -m)" # x86_64 or aarch64
RPM_FILE="surveiller-${VERSION_NO_V}-1.${ARCH}.rpm"
curl -fsSL -o "/tmp/${RPM_FILE}" \
  "https://github.com/doridoridoriand/surveiller/releases/download/${VERSION}/${RPM_FILE}"
sudo dnf install -y "/tmp/${RPM_FILE}"
```

#### Homebrew (macOS/Linux)

```bash
VERSION="v0.0.12"
brew install \
  "https://github.com/doridoridoriand/surveiller/releases/download/${VERSION}/surveiller.rb"
```

#### Chocolatey (Windows)

```powershell
# Chocolatey community feed
choco install surveiller -y

# or private feed
choco install surveiller -y --source="'https://community.chocolatey.org/api/v2/'"
```

### Package Manager Troubleshooting

- General checks:
  - Confirm package name is `surveiller`.
  - Confirm `VERSION` matches an existing GitHub release tag.
  - After install, verify with `surveiller -version`.
- APT:
  - If install fails with dependency errors, run `sudo apt update` and retry.
  - Ensure `ARCH` is either `amd64` or `arm64`.
  - Ensure the `.deb` URL exists for your selected `VERSION`.
- DNF:
  - Ensure `ARCH` is either `x86_64` or `aarch64`.
  - If install fails with dependency errors, run `sudo dnf clean all && sudo dnf makecache --refresh`.
  - Ensure the `.rpm` URL exists for your selected `VERSION`.
- Homebrew:
  - If formula URL returns 404, verify the release exists and includes `surveiller.rb`.
  - If checksum mismatch occurs, run `brew update` and retry install.
  - If an old formula is cached, run `brew uninstall surveiller` then install again.
- Chocolatey:
  - If package is not found in private feeds, verify source configuration with `choco source list`.
  - If authentication fails, verify API key and source URL.
  - If a bad version was already published to an immutable feed, install/roll forward to the next fixed patch version.

### Build from Source

```bash
git clone https://github.com/doridoridoriand/surveiller.git
cd surveiller
make build
```

Requirements:
- Go 1.24.0 or later

## Usage

### Basic Usage

```bash
./bin/surveiller path/to/surveiller.conf
```

### Configuration Format

The configuration format is compatible with the original deadman:

```conf
# surveiller: interval=2s timeout=1500ms max_concurrency=50 ui.scale=25 ui.disable=false
google 216.58.197.174
googleDNS 8.8.8.8
---
kame 203.178.141.194
```

- Each target line: `name address`
- Use `---` to start a new group
- `# surveiller:` directives set global options
- Lines starting with `#` are comments

### CLI Options

CLI options override config file values:

```bash
./bin/surveiller \
  -interval 1s \
  -timeout 500ms \
  -max-concurrency 10 \
  -metrics-mode per-target \
  -metrics-listen :9100 \
  -no-ui \
  --log-file /var/log/surveiller.log \
  path/to/surveiller.conf
```

### Available Options

- `-i, --interval duration`: Ping interval per target
- `-t, --timeout duration`: Ping timeout
- `--max-concurrency int`: Maximum concurrent pings
- `--metrics-mode string`: Metrics mode (per-target|aggregated|both)
- `--metrics-listen string`: Prometheus metrics listen address
- `--no-ui`: Run without TUI (log only mode)
- `--log-file string`: Log file path (default: logging disabled)
  - When specified, structured logs (JSON format) are written to the file
  - Logs are not output to stdout/stderr to avoid interfering with TUI
- `-v, --version`: Show version

## Configuration Reference

### Global Options

Set in config file using `# surveiller:` directive:

- `interval`: Ping interval (e.g., `1s`, `500ms`) — must be a positive duration
- `timeout`: Ping timeout — must be a positive duration
- `max_concurrency`: Maximum simultaneous pings — must be a positive integer
- `metrics.mode`: Prometheus metrics granularity — one of `per-target`, `aggregated`, `both`
- `metrics.listen`: HTTP address for metrics endpoint (e.g. `:9100` or `9100`)
- `ui.scale`: RTT bar scale in milliseconds — must be a positive integer
- `ui.disable`: Disable terminal UI — `true` or `false`

**Validation rules:** All numeric/interval options must be **positive** (greater than zero). Target lines must have a non-empty name and address; duplicate target names in the same config are not allowed.

### Configuration errors and troubleshooting

When the config file is invalid, surveiller prints an error to stderr in the form `config error: <path>:<line>: <reason>` (or `config error: <path>: <reason>` for global-level errors). Common cases:

| Message / keyword                              | Cause                                     | Fix                                                                 |
| ---                                            | ---                                       | ---                                                                 |
| `invalid interval` / `invalid timeout`         | Unparseable or non-positive duration      | Use a positive duration, e.g. `1s`, `500ms`, `2s`                   |
| `invalid max_concurrency` / `must be positive` | Value is zero or not an integer           | Use a positive integer, e.g. `10`, `100`                            |
| `invalid metrics.mode`                         | Unknown mode                              | Use one of: `per-target`, `aggregated`, `both`                      |
| `invalid target line`                          | Line does not match `name address [key=value ...]` | Ensure each target line has at least two space-separated tokens (name and address) |
| `duplicate target name`                        | Same target name appears more than once   | Use unique names per target                                         |
| `interval must be positive` / `timeout must be positive` | Global option is 0 or negative (e.g. via CLI override) | Set a positive value in config or CLI                               |

### Example Configuration

```conf
# Global settings
# surveiller: interval=1s timeout=1s max_concurrency=100 ui.scale=10

# Internet connectivity
google 216.58.197.174
cloudflare 1.1.1.1
---
# Internal network
router 192.168.1.1
server1 192.168.1.10
server2 192.168.1.11
```

## Status Monitoring

### Status Levels

surveiller uses four status levels to indicate target health:

- **OK**: Ping successful and RTT is within 25% of the configured timeout
- **WARN**: Either:
  - Ping successful but RTT exceeds 25% of timeout
  - Ping failed but consecutive failures are less than the threshold
- **DOWN**: Ping failed and consecutive failures reach the threshold (default: 3)
- **UNKNOWN**: Target initialized but no ping has been executed yet

### Status Thresholds

**Success-based thresholds (RTT-based):**
- OK: `RTT ≤ timeout × 25%`
- WARN: `RTT > timeout × 25%` (even if RTT exceeds 50% of timeout)

**Failure-based thresholds (consecutive failures):**
- WARN: Consecutive failures < 3
- DOWN: Consecutive failures ≥ 3

**Note:** These thresholds are currently hardcoded and cannot be changed via configuration file or CLI options.

**Example:**
- With `timeout=100ms`:
  - RTT ≤ 25ms → **OK**
  - RTT > 25ms → **WARN**
  - 1-2 consecutive failures → **WARN**
  - 3+ consecutive failures → **DOWN**

## Terminal UI

The TUI displays the following information for each target:

1. **Name**: Target label/name
2. **Address**: IP address
3. **Status**: Current status (OK / WARN / DOWN / UNKNOWN) with color coding:
   - Green: OK
   - Yellow: WARN
   - Red: DOWN
   - Gray: UNKNOWN
4. **RTT**: Latest RTT with label prefix (`RTT:XXms` or `RTT:XX.Xs`)
5. **AVG**: Average RTT with label prefix (`AVG:XXms` or `AVG:XX.Xs`)
   - Calculated from ping history
   - Falls back to last RTT if history is empty
6. **LOSS**: Packet loss percentage (`LOSS:XX.X%`)
   - Calculated as: `(TotalFailures / (TotalSuccesses + TotalFailures)) × 100`
   - Shows `0.0%` when no pings have been executed
7. **RTT Bar**: Visual bar graph representing RTT (scaled by `ui.scale` setting)

## Prometheus Metrics

When `metrics.listen` is configured, surveiller exposes Prometheus metrics:

```bash
curl http://localhost:9100/metrics
```

Available metrics:
- `surveiller_ping_rtt_seconds`: Current RTT per target
- `surveiller_ping_success_total`: Successful ping count
- `surveiller_ping_failure_total`: Failed ping count
- `surveiller_ping_up`: Target status (1=up, 0=down)

## Development

### Building

```bash
make build          # Build binary
make test           # Run tests
make test-prop      # Run property-based tests
make clean          # Clean build artifacts
```

### Project Structure

```
├── internal/
│   ├── cli/        # Command-line flag handling
│   ├── config/     # Configuration parsing
│   ├── metrics/    # Prometheus metrics
│   ├── ping/       # ICMP and external ping implementations
│   ├── scheduler/  # Concurrent monitoring scheduler
│   ├── state/      # Target state management
│   └── ui/         # Terminal user interface
├── example/        # Sample configuration
└── docs/           # Design documentation
```

## Acknowledgments

This project is inspired by and maintains compatibility with the original [deadman](https://github.com/upa/deadman) tool created by [upa](https://github.com/upa). We thank the original author for creating such a useful monitoring tool.

The original deadman was designed for Interop Tokyo ShowNet and has been widely used for temporary network monitoring in conference and event environments.

## License

MIT License - see [LICENSE](LICENSE) file for details.

This project is licensed under the same MIT license as the original deadman project.

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

### Development Guidelines

- Follow Go conventions and best practices
- Add tests for new functionality
- Update documentation as needed
- Maintain compatibility with original deadman config format

## Roadmap

- [ ] SSH relay support (`relay=` option)
- [ ] Enhanced Grafana dashboard templates
- [ ] Additional monitoring protocols (HTTP, TCP)
- [x] Configuration validation and better error messages
- [ ] Improved macOS and Windows support (currently experimental)

## Platform-Specific Notes

### Linux
- Fully supported with comprehensive testing
- ICMP ping requires root privileges or `CAP_NET_RAW` capability
- Falls back to external `ping` command when privileges unavailable

### macOS (Experimental)
- Basic functionality verified but not continuously tested
- May require `sudo` for ICMP ping functionality
- External `ping` command fallback available
- Community feedback welcome for platform-specific issues

### Windows (Experimental)
- Basic functionality verified but not continuously tested
- May require administrator privileges for ICMP ping
- External `ping` command fallback available
- Community feedback welcome for platform-specific issues

**Note**: For production use, Linux is the recommended platform with full testing coverage.

## Support

- Create an [issue](https://github.com/doridoridoriand/surveiller/issues) for bug reports or feature requests
- Check the [documentation](docs/) for detailed design information
- Review [example configurations](example/) for usage patterns

### Reporting Platform-Specific Issues

When reporting issues on macOS or Windows (experimental platforms), please include:
- Operating system version
- Whether running with elevated privileges (sudo/administrator)
- ICMP ping vs external ping command behavior
- Complete error messages and logs

### Reporting Platform-Specific Issues

When reporting issues on macOS or Windows (experimental platforms), please include:
- Operating system version
- Whether running with elevated privileges (sudo/administrator)
- ICMP ping vs external ping command behavior
- Complete error messages and logs
