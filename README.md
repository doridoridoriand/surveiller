# deadman-go

![deadman-go](assets/running.gif)

A Go implementation of the [deadman](https://github.com/upa/deadman) ping monitoring tool, providing efficient host status monitoring with a terminal-based interface.

## About

deadman-go is inspired by and maintains compatibility with the original [deadman](https://github.com/upa/deadman) tool by [upa](https://github.com/upa). This Go implementation offers:

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

Download the latest release from the [releases page](https://github.com/doridoridoriand/deadman-go/releases).

### Build from Source

```bash
git clone https://github.com/doridoridoriand/deadman-go.git
cd deadman-go
make build
```

Requirements:
- Go 1.24.0 or later

## Usage

### Basic Usage

```bash
./bin/deadman-go path/to/deadman.conf
```

### Configuration Format

The configuration format is compatible with the original deadman:

```conf
# deadman-go: interval=2s timeout=1500ms max_concurrency=50 ui.scale=25 ui.disable=false
google 216.58.197.174
googleDNS 8.8.8.8
---
kame 203.178.141.194
```

- Each target line: `name address`
- Use `---` to start a new group
- `# deadman-go:` directives set global options
- Lines starting with `#` are comments

### CLI Options

CLI options override config file values:

```bash
./bin/deadman-go \
  -interval 1s \
  -timeout 500ms \
  -max-concurrency 10 \
  -metrics-mode per-target \
  -metrics-listen :9100 \
  -no-ui \
  path/to/deadman.conf
```

### Available Options

- `-i, --interval duration`: Ping interval per target
- `-t, --timeout duration`: Ping timeout
- `--max-concurrency int`: Maximum concurrent pings
- `--metrics-mode string`: Metrics mode (per-target|aggregated|both)
- `--metrics-listen string`: Prometheus metrics listen address
- `--no-ui`: Run without TUI (log only mode)
- `-v, --version`: Show version

## Configuration Reference

### Global Options

Set in config file using `# deadman-go:` directive:

- `interval`: Ping interval (e.g., `1s`, `500ms`)
- `timeout`: Ping timeout
- `max_concurrency`: Maximum simultaneous pings
- `metrics.mode`: Prometheus metrics granularity
- `metrics.listen`: HTTP address for metrics endpoint
- `ui.scale`: RTT bar scale in milliseconds
- `ui.disable`: Disable terminal UI

### Example Configuration

```conf
# Global settings
# deadman-go: interval=1s timeout=1s max_concurrency=100 ui.scale=10

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

deadman-go uses four status levels to indicate target health:

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
4. **RTT**: Average RTT with label prefix (`RTT:XXms` or `RTT:XX.Xs`)
   - Calculated from ping history
   - Falls back to last RTT if history is empty
5. **LOSS**: Packet loss percentage (`LOSS:XX.X%`)
   - Calculated as: `(TotalFailures / (TotalSuccesses + TotalFailures)) × 100`
   - Shows `0.0%` when no pings have been executed
6. **RTT Bar**: Visual bar graph representing RTT (scaled by `ui.scale` setting)

## Prometheus Metrics

When `metrics.listen` is configured, deadman-go exposes Prometheus metrics:

```bash
curl http://localhost:9100/metrics
```

Available metrics:
- `deadman_ping_rtt_seconds`: Current RTT per target
- `deadman_ping_success_total`: Successful ping count
- `deadman_ping_failure_total`: Failed ping count
- `deadman_ping_up`: Target status (1=up, 0=down)

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
- [ ] macOS and Windows support
- [ ] Enhanced Grafana dashboard templates
- [ ] Additional monitoring protocols (HTTP, TCP)
- [ ] Configuration validation and better error messages

## Support

- Create an [issue](https://github.com/doridoridoriand/deadman-go/issues) for bug reports or feature requests
- Check the [documentation](docs/) for detailed design information
- Review [example configurations](example/) for usage patterns