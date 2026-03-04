package apt

import (
	"os"
	"path/filepath"
	"testing"
)

func readFixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", name))
	if err != nil {
		t.Fatalf("reading fixture %s: %v", name, err)
	}
	return string(data)
}

func TestParseUbuntuOutput(t *testing.T) {
	output := readFixture(t, "apt_output_ubuntu.txt")
	result, err := Parse(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := result.TotalPending(); got != 4 {
		t.Errorf("TotalPending() = %d, want 4", got)
	}

	expected := []PackageUpgrade{
		{Package: "base-files", FromVersion: "12ubuntu4.4", ToVersion: "12ubuntu4.5", Origin: "Ubuntu:22.04/jammy-updates", Arch: "amd64"},
		{Package: "libssl3", FromVersion: "3.0.2-0ubuntu1.12", ToVersion: "3.0.2-0ubuntu1.13", Origin: "Ubuntu:22.04/jammy-updates", Arch: "amd64"},
		{Package: "openssl", FromVersion: "3.0.2-0ubuntu1.12", ToVersion: "3.0.2-0ubuntu1.13", Origin: "Ubuntu:22.04/jammy-updates", Arch: "amd64"},
		{Package: "curl", FromVersion: "7.81.0-1ubuntu1.14", ToVersion: "7.81.0-1ubuntu1.15", Origin: "Ubuntu:22.04/jammy-security", Arch: "amd64"},
	}

	assertPackages(t, result.Packages, expected)

	// Check aggregation by origin/arch.
	pending := result.PendingByOriginArch()
	if got := pending[[2]string{"Ubuntu:22.04/jammy-updates", "amd64"}]; got != 3 {
		t.Errorf("jammy-updates count = %d, want 3", got)
	}
	if got := pending[[2]string{"Ubuntu:22.04/jammy-security", "amd64"}]; got != 1 {
		t.Errorf("jammy-security count = %d, want 1", got)
	}
}

func TestParseDebianOutput(t *testing.T) {
	output := readFixture(t, "apt_output_debian.txt")
	result, err := Parse(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := result.TotalPending(); got != 2 {
		t.Errorf("TotalPending() = %d, want 2", got)
	}

	expected := []PackageUpgrade{
		{Package: "apt", FromVersion: "2.6.1", ToVersion: "2.6.2", Origin: "Debian:bookworm-updates", Arch: "arm64"},
		{Package: "libapt-pkg6.0", FromVersion: "2.6.1", ToVersion: "2.6.2", Origin: "Debian:bookworm-updates", Arch: "arm64"},
	}

	assertPackages(t, result.Packages, expected)
}

func TestParseEmptyOutput(t *testing.T) {
	output := readFixture(t, "apt_output_empty.txt")
	result, err := Parse(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := result.TotalPending(); got != 0 {
		t.Errorf("TotalPending() = %d, want 0", got)
	}

	pending := result.PendingByOriginArch()
	if len(pending) != 0 {
		t.Errorf("PendingByOriginArch() should be empty, got %v", pending)
	}
}

func TestParseNoCurrentVersion(t *testing.T) {
	output := readFixture(t, "apt_output_no_current.txt")
	result, err := Parse(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := result.TotalPending(); got != 2 {
		t.Errorf("TotalPending() = %d, want 2", got)
	}

	// First package has no current version.
	if result.Packages[0].FromVersion != "" {
		t.Errorf("Packages[0].FromVersion = %q, want empty", result.Packages[0].FromVersion)
	}
	if result.Packages[0].Package != "linux-headers-6.1.0-18-amd64" {
		t.Errorf("Packages[0].Package = %q, want linux-headers-6.1.0-18-amd64", result.Packages[0].Package)
	}
	if result.Packages[0].ToVersion != "6.1.76-1" {
		t.Errorf("Packages[0].ToVersion = %q, want 6.1.76-1", result.Packages[0].ToVersion)
	}

	// Second package has a current version.
	if result.Packages[1].FromVersion != "12.4+deb12u5" {
		t.Errorf("Packages[1].FromVersion = %q, want 12.4+deb12u5", result.Packages[1].FromVersion)
	}
}

func TestParseMultiOrigin(t *testing.T) {
	output := readFixture(t, "apt_output_multi_origin.txt")
	result, err := Parse(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := result.TotalPending(); got != 2 {
		t.Errorf("TotalPending() = %d, want 2", got)
	}

	expected := []PackageUpgrade{
		{Package: "sosreport", FromVersion: "4.9.2-0ubuntu0~24.04.1", ToVersion: "4.10.2-0ubuntu0~24.04.1", Origin: "Ubuntu:24.04/noble-updates", Arch: "amd64"},
		{Package: "intel-microcode", FromVersion: "3.20250812.0ubuntu0.24.04.1", ToVersion: "3.20260210.0ubuntu0.24.04.1", Origin: "Ubuntu:24.04/noble-updates", Arch: "amd64"},
	}

	assertPackages(t, result.Packages, expected)
}

func TestParseEmptyString(t *testing.T) {
	result, err := Parse("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := result.TotalPending(); got != 0 {
		t.Errorf("TotalPending() = %d, want 0", got)
	}
}

func TestCheckReboot(t *testing.T) {
	dir := t.TempDir()

	// No reboot-required file.
	if CheckReboot(dir) {
		t.Error("CheckReboot() = true, want false (no file)")
	}

	// Create the reboot-required file.
	runDir := filepath.Join(dir, "run")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "reboot-required"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	if !CheckReboot(dir) {
		t.Error("CheckReboot() = false, want true (file exists)")
	}
}

func assertPackages(t *testing.T, got, want []PackageUpgrade) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d packages, want %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("package[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}
