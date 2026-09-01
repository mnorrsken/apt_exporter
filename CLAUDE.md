# apt_exporter

Prometheus exporter for APT package upgrade metrics on Debian/Ubuntu. Go.
Ships as a multi-arch container image and Helm chart on GHCR, and as a `.deb`
with a systemd unit (`packaging/`).

## Commands

- `make build` (to `bin/apt_exporter`), `make build-static`, `make run`
- `make test` (unit), `make test-integration` (needs Docker), `make lint`,
  `make vet`, `make fmt`
- `make docker-build`, `make deb`

## Verify before done

`make test` and `make lint` green. `make test-integration` when the change
touches `internal/apt`, `internal/hook` or `internal/watcher` (they shell out
to apt and watch the dpkg state on a real system).

## Layout

`internal/apt` runs and parses apt, `internal/collector` exposes the metrics,
`internal/hook` and `internal/watcher` react to apt/dpkg activity,
`internal/service` wires it together. Metrics use the stdlib exposition
convention from the wiki skill "Go Prometheus Metrics".

## Release notes (what differs from the wiki "GitHub Release Process" skill)

- Changelog heading: `## vX.Y.Z` (Keep a Changelog sections, only the ones
  that apply).
- README: add or update a "Features" bullet and the relevant "Usage" section
  for a new capability.
- One commit for code, changelog and readme: `Add <feature> (vX.Y.Z)` or
  `Fix <thing> (vX.Y.Z)`. No version string in Go source; git tags only.
- On `v*` tags: `release.yml` builds and pushes the multi-arch image to GHCR;
  `helm.yml` sets `Chart.yaml` `version`/`appVersion` from the tag (never bump
  by hand), packages and pushes the chart to the GHCR OCI registry.
