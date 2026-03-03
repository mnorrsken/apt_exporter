package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/mnorrsken/apt_exporter/internal/apt"
	"github.com/mnorrsken/apt_exporter/internal/collector"
	"github.com/mnorrsken/apt_exporter/internal/hook"
	"github.com/mnorrsken/apt_exporter/internal/watcher"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	version   = "dev"
	buildDate = "unknown"
)

func main() {
	app := kingpin.New("apt_exporter", "Prometheus exporter for APT package upgrades.")
	app.Version(fmt.Sprintf("%s (built %s)", version, buildDate))
	app.HelpFlag.Short('h')

	listenAddress := app.Flag("web.listen-address", "Address to listen on for web interface and telemetry.").
		Default(":9120").String()
	metricsPath := app.Flag("web.telemetry-path", "Path under which to expose metrics.").
		Default("/metrics").String()
	refreshInterval := app.Flag("apt.refresh-interval", "Interval between periodic APT cache refreshes.").
		Default("24h").Duration()
	rootfs := app.Flag("apt.rootfs", "Root filesystem path (set to /host when running in a container).").
		Default("/").String()
	logLevel := app.Flag("log.level", "Log level (debug, info, warn, error).").
		Default("info").Enum("debug", "info", "warn", "error")

	app.Command("serve", "Run the exporter (default).").Default()

	hookCmd := app.Command("hook", "Manage APT hook.")
	hookInstallCmd := hookCmd.Command("install", "Install the APT post-invoke hook.")
	hookInstallEndpoint := hookInstallCmd.Flag("endpoint", "Reload endpoint URL.").
		Default(hook.DefaultEndpoint).String()
	hookInstallPath := hookInstallCmd.Flag("hook-path", "APT hook file path.").
		Default(hook.DefaultHookPath).String()
	hookInstallRootfs := hookInstallCmd.Flag("rootfs", "Root filesystem prefix.").
		Default("/").String()

	hookUninstallCmd := hookCmd.Command("uninstall", "Uninstall the APT post-invoke hook.")
	hookUninstallPath := hookUninstallCmd.Flag("hook-path", "APT hook file path.").
		Default(hook.DefaultHookPath).String()
	hookUninstallRootfs := hookUninstallCmd.Flag("rootfs", "Root filesystem prefix.").
		Default("/").String()

	parsed, err := app.DefaultEnvars().Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	switch parsed {
	case hookInstallCmd.FullCommand():
		if err := hook.Install(*hookInstallPath, *hookInstallEndpoint, *hookInstallRootfs); err != nil {
			fmt.Fprintf(os.Stderr, "error installing hook: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("APT hook installed successfully.")
		return

	case hookUninstallCmd.FullCommand():
		if err := hook.Uninstall(*hookUninstallPath, *hookUninstallRootfs); err != nil {
			fmt.Fprintf(os.Stderr, "error uninstalling hook: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("APT hook uninstalled successfully.")
		return
	}

	// Set up logger.
	var level slog.Level
	switch *logLevel {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))

	logger.Info("starting apt_exporter", "version", version, "listen", *listenAddress, "rootfs", *rootfs)

	// Set up cache and collector.
	cache := collector.NewCache()
	col := collector.NewAptCollector(cache)
	prometheus.MustRegister(col)

	// Set up trigger channel (buffered to coalesce).
	triggerCh := make(chan struct{}, 1)

	// Set up context with signal handling.
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Start update loop.
	runner := apt.NewRunner(*rootfs)
	go updateLoop(ctx, triggerCh, runner, cache, logger)

	// Trigger initial update.
	triggerCh <- struct{}{}

	// Start watcher.
	watchPath := filepath.Join(*rootfs, "var", "lib", "apt", "lists")
	w := watcher.New(triggerCh, watchPath, *refreshInterval, logger)
	go func() {
		if err := w.Run(ctx); err != nil {
			logger.Error("watcher error", "err", err)
		}
	}()

	// Set up HTTP server.
	mux := http.NewServeMux()
	mux.Handle(*metricsPath, promhttp.Handler())
	mux.HandleFunc("/-/reload", func(w http.ResponseWriter, r *http.Request) {
		if !isLoopback(r.RemoteAddr) {
			http.Error(w, "Forbidden: reload only allowed from localhost.", http.StatusForbidden)
			return
		}
		select {
		case triggerCh <- struct{}{}:
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, "Reload triggered.")
		default:
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, "Reload already pending.")
		}
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprintf(w, `<html>
<head><title>APT Exporter</title></head>
<body>
<h1>APT Exporter</h1>
<p><a href="%s">Metrics</a></p>
<p>Version: %s</p>
</body>
</html>`, *metricsPath, version)
	})

	server := &http.Server{
		Addr:              *listenAddress,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		<-ctx.Done()
		logger.Info("shutting down HTTP server")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	logger.Info("listening", "address", *listenAddress)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		logger.Error("HTTP server error", "err", err)
		os.Exit(1)
	}
}

func updateLoop(ctx context.Context, triggerCh <-chan struct{}, runner *apt.Runner, cache *collector.Cache, logger *slog.Logger) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-triggerCh:
			output, err := runner.Run(ctx)
			if err != nil {
				logger.Error("apt-get failed", "err", err)
				continue
			}
			result, err := apt.Parse(output)
			if err != nil {
				logger.Error("parse failed", "err", err)
				continue
			}
			reboot := apt.CheckReboot(runner.RootFS())
			cache.Update(result, reboot)
			logger.Info("cache updated", "pending", result.TotalPending(), "reboot_required", reboot)
		}
	}
}

// isLoopback returns true if remoteAddr is a loopback address (127.0.0.0/8 or ::1).
func isLoopback(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
