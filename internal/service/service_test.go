package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContent(t *testing.T) {
	content := Content("/usr/local/bin/apt_exporter")

	checks := []struct {
		name string
		want string
	}{
		{"ExecStart", "ExecStart=/usr/local/bin/apt_exporter serve"},
		{"DynamicUser", "DynamicUser=yes"},
		{"ProtectSystem", "ProtectSystem=strict"},
		{"NoNewPrivileges", "NoNewPrivileges=yes"},
		{"WantedBy", "WantedBy=multi-user.target"},
		{"PrivateTmp", "PrivateTmp=yes"},
		{"ReadOnlyPaths", "ReadOnlyPaths=/var/lib/apt /var/lib/dpkg /run"},
	}

	for _, c := range checks {
		if !strings.Contains(content, c.want) {
			t.Errorf("content missing %s: %q", c.name, c.want)
		}
	}
}

func TestInstallAndUninstall(t *testing.T) {
	dir := t.TempDir()
	unitPath := filepath.Join(dir, "apt-exporter.service")

	if err := Install(unitPath, DefaultExecPath); err != nil {
		t.Fatalf("Install() error: %v", err)
	}

	data, err := os.ReadFile(unitPath)
	if err != nil {
		t.Fatalf("reading installed unit: %v", err)
	}

	if !strings.Contains(string(data), "ExecStart=") {
		t.Error("installed file missing ExecStart")
	}

	if err := Uninstall(unitPath); err != nil {
		t.Fatalf("Uninstall() error: %v", err)
	}

	if _, err := os.Stat(unitPath); !os.IsNotExist(err) {
		t.Error("unit file still exists after Uninstall()")
	}
}

func TestUninstallNonExistent(t *testing.T) {
	dir := t.TempDir()
	if err := Uninstall(filepath.Join(dir, "nonexistent.service")); err != nil {
		t.Fatalf("Uninstall() on non-existent file error: %v", err)
	}
}
