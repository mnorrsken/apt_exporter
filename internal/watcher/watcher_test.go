package watcher

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPeriodicTrigger(t *testing.T) {
	triggerCh := make(chan struct{}, 1)
	dir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	w := New(triggerCh, dir, 100*time.Millisecond, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		_ = w.Run(ctx)
	}()

	select {
	case <-triggerCh:
		// Got periodic trigger.
	case <-time.After(1 * time.Second):
		t.Error("periodic trigger did not fire within 1s")
	}
}

func TestInotifyTrigger(t *testing.T) {
	triggerCh := make(chan struct{}, 1)
	dir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	// Use a long periodic interval so only inotify fires.
	w := New(triggerCh, dir, 1*time.Hour, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	go func() {
		_ = w.Run(ctx)
	}()

	// Give inotify time to set up.
	time.Sleep(100 * time.Millisecond)

	// Write a file to trigger inotify.
	if err := os.WriteFile(filepath.Join(dir, "test-pkg"), []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	select {
	case <-triggerCh:
		// Got inotify trigger after debounce.
	case <-time.After(8 * time.Second):
		t.Error("inotify trigger did not fire within 8s (5s debounce + margin)")
	}
}

func TestNonBlockingSend(t *testing.T) {
	triggerCh := make(chan struct{}, 1)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	w := New(triggerCh, "/tmp", time.Hour, logger)

	// Fill the channel.
	triggerCh <- struct{}{}

	// This should not block.
	done := make(chan struct{})
	go func() {
		w.nonBlockingSend()
		close(done)
	}()

	select {
	case <-done:
		// Good, did not block.
	case <-time.After(1 * time.Second):
		t.Error("nonBlockingSend blocked when channel was full")
	}
}
