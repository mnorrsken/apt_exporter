//go:build integration

package test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestExporterOnUbuntu(t *testing.T) {
	testExporterOnDistro(t, "ubuntu:22.04")
}

func TestExporterOnDebian(t *testing.T) {
	testExporterOnDistro(t, "debian:bookworm")
}

func testExporterOnDistro(t *testing.T, image string) {
	t.Helper()
	ctx := context.Background()

	// Build the binary for linux/amd64.
	buildBinary(t)

	req := testcontainers.ContainerRequest{
		Image: image,
		ExposedPorts: []string{"9120/tcp"},
		Files: []testcontainers.ContainerFile{
			{
				HostFilePath:      "../bin/apt_exporter",
				ContainerFilePath: "/usr/local/bin/apt_exporter",
				FileMode:          0o755,
			},
		},
		Cmd: []string{
			"bash", "-c",
			"apt-get update -qq && /usr/local/bin/apt_exporter --log.level=debug &" +
				" sleep 2 && wait",
		},
		WaitingFor: wait.ForHTTP("/").WithPort("9120/tcp").WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start container (%s): %v", image, err)
	}
	defer func() {
		_ = container.Terminate(ctx)
	}()

	endpoint, err := container.Endpoint(ctx, "http")
	if err != nil {
		t.Fatalf("failed to get endpoint: %v", err)
	}

	// Test 1: Metrics endpoint returns apt metrics.
	t.Run("metrics", func(t *testing.T) {
		body := httpGet(t, endpoint+"/metrics")

		if !strings.Contains(body, "apt_upgrades_pending") {
			t.Error("metrics missing apt_upgrades_pending")
		}
		if !strings.Contains(body, "node_reboot_required") {
			t.Error("metrics missing node_reboot_required")
		}
		// Fresh container should not require reboot.
		if !strings.Contains(body, "node_reboot_required 0") {
			t.Error("expected node_reboot_required 0 in fresh container")
		}
	})

	// Test 2: Create reboot-required file, trigger reload, check metric.
	t.Run("reboot_required", func(t *testing.T) {
		exitCode, _, err := container.Exec(ctx, []string{"touch", "/run/reboot-required"})
		if err != nil || exitCode != 0 {
			t.Fatalf("failed to create reboot-required: exit=%d err=%v", exitCode, err)
		}

		// Trigger reload.
		body := httpGet(t, endpoint+"/-/reload")
		if !strings.Contains(body, "Reload") {
			t.Errorf("unexpected reload response: %s", body)
		}

		// Wait for cache update.
		time.Sleep(2 * time.Second)

		body = httpGet(t, endpoint+"/metrics")
		if !strings.Contains(body, "node_reboot_required 1") {
			t.Error("expected node_reboot_required 1 after creating reboot-required file")
		}
	})

	// Test 3: Landing page.
	t.Run("landing_page", func(t *testing.T) {
		body := httpGet(t, endpoint+"/")
		if !strings.Contains(body, "APT Exporter") {
			t.Error("landing page missing title")
		}
	})
}

func buildBinary(t *testing.T) {
	t.Helper()
	// The Makefile builds to bin/apt_exporter. Assume it's already built
	// via `make build` before running integration tests (CGO_ENABLED=0 for static binary).
	// Alternatively, build it here:
	// We skip building in the test to keep it simple - the Makefile target
	// test-integration should build first.
}

func httpGet(t *testing.T, url string) string {
	t.Helper()
	client := &http.Client{Timeout: 10 * time.Second}

	var lastErr error
	for i := 0; i < 5; i++ {
		resp, err := client.Get(url)
		if err != nil {
			lastErr = err
			time.Sleep(1 * time.Second)
			continue
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("reading response body: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("GET %s returned status %d: %s", url, resp.StatusCode, string(body))
		}
		return string(body)
	}
	t.Fatalf("GET %s failed after retries: %v", url, lastErr)
	return ""
}

