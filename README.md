# multi-ping

`multi-ping` is a command-line application written in Go that continuously probes one or more targets through one or more network interfaces and renders the live results in a multi-pane terminal dashboard.

## Features

- Repeated probes using the system `ping` command bound to a specific interface
- Per-interface panes with live status, RTT, packet loss, and the last note/error
- Two input modes:
  - matrix mode: every target is checked through every listed interface
  - explicit mode: map specific targets to specific interfaces
- No third-party Go dependencies

## Requirements

- Go 1.22 or newer for local builds
- Linux for runtime and packaged binaries
- A `ping` binary available on `PATH`

## Usage

During development, you can run the application directly with `go run`. Once installed system-wide from a packaged release, the command is available as `multi-ping`.

Probe every target through every interface:

```bash
go run ./cmd/multi-ping \
  -interfaces lo,eth0 \
  -targets 127.0.0.1,8.8.8.8
```

```bash
multi-ping \
  -interfaces lo,eth0 \
  -targets 127.0.0.1,8.8.8.8
```

Map targets to specific interfaces:

```bash
go run ./cmd/multi-ping \
  -check lo=127.0.0.1 \
  -check tun0=10.10.10.1,10.10.10.2
```

```bash
multi-ping \
  -check lo=127.0.0.1 \
  -check tun0=10.10.10.1,10.10.10.2
```

Adjust the probe cadence:

```bash
go run ./cmd/multi-ping \
  -interfaces lo \
  -targets 127.0.0.1 \
  -interval 3s \
  -timeout 2s
```

```bash
multi-ping \
  -interfaces lo \
  -targets 127.0.0.1 \
  -interval 3s \
  -timeout 2s
```

## Notes

- The app currently shells out to `ping -I <interface>`, which is the most reliable way to enforce interface selection without additional privileges or raw socket handling in Go.
- Interface names are validated before the dashboard starts.
- Press `Ctrl+C` to exit.

## Build

Build static Linux binaries locally:

```bash
make build
```

Create tarballs and checksums in `dist/`:

```bash
make checksums
```

Artifacts produced by the `Makefile`:

- `dist/multi-ping-linux-amd64`
- `dist/multi-ping-linux-arm64`
- `dist/multi-ping-linux-amd64.tar.gz`
- `dist/multi-ping-linux-arm64.tar.gz`
- `dist/checksums.txt`

## GitHub Actions

The workflow template at `.github/workflows/release.yml` does the following:

- builds static Linux binaries for `amd64` and `arm64`
- uploads them as workflow artifacts on pull requests, manual runs, and pushes to `main`
- publishes the binaries, tarballs, and checksum files to a GitHub release when you push a tag like `v1.0.0`

## Auto-Built Platforms

The release workflow currently publishes the following native Linux binaries:

| OS    | Architecture      | Artifact                  |
| ----- | ----------------- | ------------------------- |
| Linux | x86_64 (`amd64`)  | `multi-ping-linux-amd64`  |
| Linux | ARM64 (`arm64`)   | `multi-ping-linux-arm64`  |
