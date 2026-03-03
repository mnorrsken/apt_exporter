package collector

import (
	"sync"

	"github.com/mnorrsken/apt_exporter/internal/apt"
	"github.com/prometheus/client_golang/prometheus"
)

// Cache holds the latest parsed APT data, protected by RWMutex.
type Cache struct {
	mu       sync.RWMutex
	packages []apt.PackageUpgrade
	pending  map[[2]string]int // (origin, arch) -> count
	reboot   bool
}

// NewCache creates a new empty Cache.
func NewCache() *Cache {
	return &Cache{
		pending: make(map[[2]string]int),
	}
}

// Update replaces the cached data with new parse results.
func (c *Cache) Update(result *apt.ParseResult, reboot bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.packages = result.Packages
	c.pending = result.PendingByOriginArch()
	c.reboot = reboot
}

// AptCollector implements prometheus.Collector for APT metrics.
type AptCollector struct {
	cache *Cache

	upgradesPending *prometheus.Desc
	upgradePackage  *prometheus.Desc
	rebootRequired  *prometheus.Desc
}

// NewAptCollector creates a new AptCollector reading from the given cache.
func NewAptCollector(cache *Cache) *AptCollector {
	return &AptCollector{
		cache: cache,
		upgradesPending: prometheus.NewDesc(
			"apt_upgrades_pending",
			"Apt package pending updates by origin.",
			[]string{"origin", "arch"}, nil,
		),
		upgradePackage: prometheus.NewDesc(
			"apt_upgrade_package",
			"Pending package upgrade details.",
			[]string{"package", "from_version", "to_version", "origin", "arch"}, nil,
		),
		rebootRequired: prometheus.NewDesc(
			"node_reboot_required",
			"Node reboot is required for software updates.",
			nil, nil,
		),
	}
}

// Describe sends the descriptor for each metric to the channel.
func (c *AptCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.upgradesPending
	ch <- c.upgradePackage
	ch <- c.rebootRequired
}

// Collect reads from the cache and sends metrics to the channel.
func (c *AptCollector) Collect(ch chan<- prometheus.Metric) {
	c.cache.mu.RLock()
	defer c.cache.mu.RUnlock()

	if len(c.cache.pending) == 0 {
		ch <- prometheus.MustNewConstMetric(c.upgradesPending, prometheus.GaugeValue, 0, "", "")
	} else {
		for key, count := range c.cache.pending {
			ch <- prometheus.MustNewConstMetric(c.upgradesPending, prometheus.GaugeValue, float64(count), key[0], key[1])
		}
	}

	for _, pkg := range c.cache.packages {
		ch <- prometheus.MustNewConstMetric(c.upgradePackage, prometheus.GaugeValue, 1,
			pkg.Package, pkg.FromVersion, pkg.ToVersion, pkg.Origin, pkg.Arch)
	}

	val := 0.0
	if c.cache.reboot {
		val = 1.0
	}
	ch <- prometheus.MustNewConstMetric(c.rebootRequired, prometheus.GaugeValue, val)
}
