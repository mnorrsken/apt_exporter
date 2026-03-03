package hook

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	// DefaultHookPath is the default location for the APT hook file.
	DefaultHookPath = "/etc/apt/apt.conf.d/80-apt-exporter"
	// DefaultEndpoint is the default reload endpoint.
	DefaultEndpoint = "http://localhost:9120/-/reload"
)

// Content generates the APT hook configuration content.
func Content(endpoint string) string {
	return fmt.Sprintf(`// Installed by apt_exporter. Do not edit.
APT::Update::Post-Invoke {"curl -fsS --connect-timeout 1 --max-time 5 -o /dev/null %s || true";};
DPkg::Post-Invoke {"curl -fsS --connect-timeout 1 --max-time 5 -o /dev/null %s || true";};
`, endpoint, endpoint)
}

// Install writes the APT hook file.
func Install(hookPath, endpoint, rootfs string) error {
	path := filepath.Join(rootfs, hookPath)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating hook directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(Content(endpoint)), 0o644); err != nil {
		return fmt.Errorf("writing hook file: %w", err)
	}
	return nil
}

// Uninstall removes the APT hook file.
func Uninstall(hookPath, rootfs string) error {
	path := filepath.Join(rootfs, hookPath)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing hook file: %w", err)
	}
	return nil
}
