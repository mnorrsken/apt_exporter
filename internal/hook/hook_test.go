package hook

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContent(t *testing.T) {
	content := Content("http://localhost:9120/-/reload")

	if !strings.Contains(content, "APT::Update::Post-Invoke") {
		t.Error("content missing APT::Update::Post-Invoke")
	}
	if !strings.Contains(content, "DPkg::Post-Invoke") {
		t.Error("content missing DPkg::Post-Invoke")
	}
	if !strings.Contains(content, "http://localhost:9120/-/reload") {
		t.Error("content missing endpoint URL")
	}
	if !strings.Contains(content, "|| true") {
		t.Error("content missing '|| true' safety guard")
	}
	if !strings.Contains(content, "--connect-timeout 1") {
		t.Error("content missing --connect-timeout")
	}
	if !strings.Contains(content, "--max-time 5") {
		t.Error("content missing --max-time")
	}
}

func TestInstallAndUninstall(t *testing.T) {
	rootfs := t.TempDir()
	hookPath := "/etc/apt/apt.conf.d/80-apt-exporter"

	// Install.
	if err := Install(hookPath, DefaultEndpoint, rootfs); err != nil {
		t.Fatalf("Install() error: %v", err)
	}

	fullPath := filepath.Join(rootfs, hookPath)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("reading installed hook: %v", err)
	}

	if !strings.Contains(string(data), "APT::Update::Post-Invoke") {
		t.Error("installed file missing APT::Update::Post-Invoke")
	}

	// Uninstall.
	if err := Uninstall(hookPath, rootfs); err != nil {
		t.Fatalf("Uninstall() error: %v", err)
	}

	if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
		t.Error("hook file still exists after Uninstall()")
	}
}

func TestUninstallNonExistent(t *testing.T) {
	rootfs := t.TempDir()
	// Should not error on missing file.
	if err := Uninstall("/etc/apt/apt.conf.d/80-apt-exporter", rootfs); err != nil {
		t.Fatalf("Uninstall() on non-existent file error: %v", err)
	}
}
