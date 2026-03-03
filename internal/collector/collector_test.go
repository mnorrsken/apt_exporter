package collector

import (
	"testing"

	"github.com/mnorrsken/apt_exporter/internal/apt"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestDescribe(t *testing.T) {
	cache := NewCache()
	col := NewAptCollector(cache)

	ch := make(chan *prometheus.Desc, 10)
	col.Describe(ch)
	close(ch)

	var descs []*prometheus.Desc
	for d := range ch {
		descs = append(descs, d)
	}

	if len(descs) != 3 {
		t.Errorf("Describe() sent %d descriptors, want 3", len(descs))
	}
}

func TestCollectEmpty(t *testing.T) {
	cache := NewCache()
	col := NewAptCollector(cache)

	registry := prometheus.NewRegistry()
	registry.MustRegister(col)

	families, err := registry.Gather()
	if err != nil {
		t.Fatalf("Gather() error: %v", err)
	}

	assertMetricValue(t, families, "apt_upgrades_pending", 0)
	assertMetricValue(t, families, "node_reboot_required", 0)
	assertMetricNotPresent(t, families, "apt_upgrade_package")
}

func TestCollectWithPackages(t *testing.T) {
	cache := NewCache()
	result := &apt.ParseResult{
		Packages: []apt.PackageUpgrade{
			{Package: "curl", FromVersion: "7.81.0-1", ToVersion: "7.81.0-2", Origin: "Ubuntu:22.04/jammy-security", Arch: "amd64"},
			{Package: "openssl", FromVersion: "3.0.2-1", ToVersion: "3.0.2-2", Origin: "Ubuntu:22.04/jammy-updates", Arch: "amd64"},
			{Package: "libssl3", FromVersion: "3.0.2-1", ToVersion: "3.0.2-2", Origin: "Ubuntu:22.04/jammy-updates", Arch: "amd64"},
		},
	}
	cache.Update(result, true)

	col := NewAptCollector(cache)
	registry := prometheus.NewRegistry()
	registry.MustRegister(col)

	families, err := registry.Gather()
	if err != nil {
		t.Fatalf("Gather() error: %v", err)
	}

	// Check reboot required.
	assertMetricValue(t, families, "node_reboot_required", 1)

	// Check aggregated pending counts.
	pendingFamily := findFamily(families, "apt_upgrades_pending")
	if pendingFamily == nil {
		t.Fatal("apt_upgrades_pending not found")
	}
	if len(pendingFamily.Metric) != 2 {
		t.Errorf("apt_upgrades_pending has %d series, want 2", len(pendingFamily.Metric))
	}

	// Check per-package metrics.
	pkgFamily := findFamily(families, "apt_upgrade_package")
	if pkgFamily == nil {
		t.Fatal("apt_upgrade_package not found")
	}
	if len(pkgFamily.Metric) != 3 {
		t.Errorf("apt_upgrade_package has %d series, want 3", len(pkgFamily.Metric))
	}
}

func TestCollectNoReboot(t *testing.T) {
	cache := NewCache()
	result := &apt.ParseResult{
		Packages: []apt.PackageUpgrade{
			{Package: "curl", FromVersion: "1.0", ToVersion: "2.0", Origin: "Test", Arch: "amd64"},
		},
	}
	cache.Update(result, false)

	col := NewAptCollector(cache)
	registry := prometheus.NewRegistry()
	registry.MustRegister(col)

	families, err := registry.Gather()
	if err != nil {
		t.Fatalf("Gather() error: %v", err)
	}

	assertMetricValue(t, families, "node_reboot_required", 0)
}

func findFamily(families []*dto.MetricFamily, name string) *dto.MetricFamily {
	for _, f := range families {
		if f.GetName() == name {
			return f
		}
	}
	return nil
}

func assertMetricValue(t *testing.T, families []*dto.MetricFamily, name string, want float64) {
	t.Helper()
	f := findFamily(families, name)
	if f == nil {
		t.Errorf("metric %q not found", name)
		return
	}
	if len(f.Metric) == 0 {
		t.Errorf("metric %q has no samples", name)
		return
	}
	// For single-value metrics, check the first (or only) sample.
	got := f.Metric[0].GetGauge().GetValue()
	if got != want {
		t.Errorf("metric %q = %v, want %v", name, got, want)
	}
}

func assertMetricNotPresent(t *testing.T, families []*dto.MetricFamily, name string) {
	t.Helper()
	f := findFamily(families, name)
	if f != nil && len(f.Metric) > 0 {
		t.Errorf("metric %q should not be present, but has %d samples", name, len(f.Metric))
	}
}
