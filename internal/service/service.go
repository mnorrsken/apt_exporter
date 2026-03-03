package service

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	// DefaultUnitPath is the default location for the systemd unit file.
	DefaultUnitPath = "/etc/systemd/system/apt-exporter.service"
	// DefaultExecPath is the default path to the apt_exporter binary.
	DefaultExecPath = "/usr/local/bin/apt_exporter"
)

// Content generates the systemd unit file content.
func Content(execPath string) string {
	return fmt.Sprintf(`# Installed by apt_exporter. Do not edit.
[Unit]
Description=Prometheus APT Exporter
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=%s serve
DynamicUser=yes
Restart=on-failure
RestartSec=5
ProtectSystem=strict
ProtectHome=yes
PrivateTmp=yes
ReadOnlyPaths=/var/lib/apt /var/lib/dpkg /run
NoNewPrivileges=yes
CapabilityBoundingSet=

[Install]
WantedBy=multi-user.target
`, execPath)
}

// Install writes the systemd unit file.
func Install(unitPath, execPath string) error {
	dir := filepath.Dir(unitPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating unit directory: %w", err)
	}
	if err := os.WriteFile(unitPath, []byte(Content(execPath)), 0o644); err != nil {
		return fmt.Errorf("writing unit file: %w", err)
	}
	return nil
}

// Uninstall removes the systemd unit file.
func Uninstall(unitPath string) error {
	if err := os.Remove(unitPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing unit file: %w", err)
	}
	return nil
}
