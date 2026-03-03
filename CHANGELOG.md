# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## v0.1.1

### Added
- **dpkg inotify watching** — watcher now monitors `/var/lib/dpkg` in addition to `/var/lib/apt/lists`, catching package install/remove/upgrade events without requiring an APT hook
- **`service install` / `service uninstall`** subcommands — sets up a hardened systemd unit with `DynamicUser=yes`, no root required at runtime

### Fixed
- **APT hook curl timeout** — added `--connect-timeout 1 --max-time 5` so apt operations are never blocked if the exporter is down

### Changed
- **Helm chart** — removed `aptHook` option (no longer needed); inotify on both apt lists and dpkg directories provides the same coverage without any privilege escalation, hostNetwork, or sidecar containers

## v0.1.0

### Added
- **Prometheus metrics** — `apt_upgrades_pending` (by origin/arch), `apt_upgrade_package` (per-package details), `node_reboot_required`
- **HTTP server** with `/metrics`, `/-/reload` (localhost-only), and landing page
- **Three cache refresh triggers** — inotify on `/var/lib/apt/lists` (5s debounce), APT post-invoke hook, periodic timer (default 24h)
- **`hook install` / `hook uninstall`** subcommands for managing the APT post-invoke hook
- **Non-root operation** — uses `apt-get -o RootDir=` instead of chroot, runs as dedicated unprivileged user
- **Graceful sleep mode** when apt-get is not available (serves empty metrics, ready for future package manager support)
- **Multi-arch Docker image** (amd64/arm64) with dedicated non-root user
- **Helm chart** for DaemonSet deployment with optional APT hook init container, ServiceMonitor, and hardened securityContext
- **GitHub Actions** for Docker image and Helm chart releases on version tags
- Unit and integration tests (Ubuntu 22.04, Debian Bookworm via testcontainers-go)
