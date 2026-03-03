# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added
- Initial release of apt_exporter
- Prometheus metrics: `apt_upgrades_pending`, `apt_upgrade_package`, `node_reboot_required`
- HTTP server with `/metrics`, `/-/reload`, and landing page endpoints
- Three cache refresh triggers: inotify on `/var/lib/apt/lists`, APT post-invoke hook, periodic timer (default 24h)
- `hook install` / `hook uninstall` subcommands for managing the APT hook
- `/-/reload` endpoint restricted to localhost only
- Runs as non-root user (uses `apt-get -o RootDir=` instead of chroot)
- Multi-stage Dockerfile with dedicated non-root user
- Helm chart for DaemonSet deployment with:
  - Optional APT hook installation via privileged init container (`aptHook.enabled`)
  - hostNetwork support when hook is enabled
  - Hardened securityContext (non-root, read-only rootfs, all capabilities dropped)
  - Optional ServiceMonitor for prometheus-operator
- Unit tests for parser, collector, watcher, and hook packages
- Integration tests using testcontainers-go against Ubuntu 22.04 and Debian Bookworm
- Makefile with build, test, test-integration, docker-build, lint, fmt, vet targets
- Apache 2.0 license
