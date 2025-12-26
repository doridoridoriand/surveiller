# Deadman go

Basic usage

Build:

```sh
make build
```

Run:

```sh
./bin/deadman-go path/to/deadman.conf
```

Config format:

```conf
# deadman-go: interval=2s timeout=1500ms max_concurrency=50 ui.scale=25 ui.disable=false
google 216.58.197.174
googleDNS 8.8.8.8
---
kame 203.178.141.194
```

- Each target line is `name address`.
- Use `---` to start a new group.
- `# deadman-go:` directives set global options.
- Lines starting with `#` are comments.

CLI options override config values:

```sh
./bin/deadman-go \
  -interval 5s \
  -timeout 500ms \
  -max-concurrency 10 \
  -metrics-mode per-target \
  -metrics-listen :9100 \
  -no-ui \
  path/to/deadman.conf
```

Version:

```sh
./bin/deadman-go -version
```
