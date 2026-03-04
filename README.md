# APT Exporter

Prometheus exporter for APT package upgrade metrics on Debian/Ubuntu systems.

Exposes pending package upgrades, per-package details, and reboot-required status as Prometheus metrics via HTTP.

## Metrics

| Metric | Labels | Description |
|--------|--------|-------------|
| `apt_upgrades_pending` | `origin`, `arch` | Number of pending upgrades by origin |
| `apt_upgrade_package` | `package`, `from_version`, `to_version`, `origin`, `arch` | Per-package upgrade details (value=1) |
| `node_reboot_required` | | Whether a reboot is required (0 or 1) |

## Quick Start

```bash
# Build
make build

# Run (no root required for --just-print)
./bin/apt_exporter

# Fetch metrics
curl http://localhost:9120/metrics
```

## CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--web.listen-address` | `:9120` | Address to listen on |
| `--web.telemetry-path` | `/metrics` | Path for metrics endpoint |
| `--apt.refresh-interval` | `24h` | Periodic refresh interval |
| `--apt.rootfs` | `/` | Root filesystem path (`/host` in containers) |
| `--log.level` | `info` | Log level (debug, info, warn, error) |

## Cache Refresh

The exporter caches APT data in memory and refreshes via three triggers:

1. **inotify** - watches `/var/lib/apt/lists` and `/var/lib/dpkg` for changes (5s debounce)
2. **APT hook** (optional, bare-metal only) - post-invoke hook calls `/-/reload` after apt operations
3. **Periodic timer** - safety net refresh (default 24h)

Metrics are always served from cache, so scrapes never block on `apt-get`.

## APT Hook

Install a hook so apt operations automatically trigger a cache refresh:

```bash
sudo apt_exporter hook install
```

This creates `/etc/apt/apt.conf.d/80-apt-exporter` which calls the exporter's reload endpoint after `apt update` and `dpkg` operations. The hook uses short timeouts (`--connect-timeout 1 --max-time 5`) so apt operations are never blocked if the exporter is down.

To remove:

```bash
sudo apt_exporter hook uninstall
```

## Systemd Service

Install as a systemd service (runs unprivileged via `DynamicUser=yes`):

```bash
sudo cp bin/apt_exporter /usr/local/bin/
sudo apt_exporter service install
sudo systemctl daemon-reload
sudo systemctl enable --now apt-exporter
```

To remove:

```bash
sudo systemctl disable --now apt-exporter
sudo apt_exporter service uninstall
sudo systemctl daemon-reload
```

## Docker

```bash
# From GHCR
docker run -d \
  -v /:/host:ro \
  --cap-add SYS_CHROOT \
  -p 9120:9120 \
  ghcr.io/mnorrsken/apt-exporter:latest

# Or build locally
make docker-build
docker run -d \
  -v /:/host:ro \
  --cap-add SYS_CHROOT \
  -p 9120:9120 \
  apt-exporter:dev
```

## Helm Chart

Deploy as a DaemonSet (runs on every node, like node_exporter):

```bash
# From GHCR OCI registry
helm install apt-exporter oci://ghcr.io/mnorrsken/charts/apt-exporter

# Or from local source
helm install apt-exporter ./charts/apt-exporter
```

With prometheus-operator ServiceMonitor:

```bash
helm install apt-exporter oci://ghcr.io/mnorrsken/charts/apt-exporter \
  --set serviceMonitor.enabled=true
```

See [charts/apt-exporter/values.yaml](charts/apt-exporter/values.yaml) for all options.

The Helm chart runs as root with `SYS_CHROOT` capability so it can chroot into the host filesystem and run the host's apt-get against the host's own libc, avoiding GLIBC version mismatches. inotify on the host's apt and dpkg directories detects all package changes automatically.

The Service is headless so that when `serviceMonitor.enabled=true`, Prometheus scrapes every node individually. Each target gets a `node` label from Kubernetes pod metadata.

## Development

```bash
# Run unit tests
make test

# Run integration tests (requires Docker)
make test-integration

# Lint
make lint

# Format
make fmt
```

## License

Apache License 2.0. See [LICENSE](LICENSE).
