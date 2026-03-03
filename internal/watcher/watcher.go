package watcher

import (
	"context"
	"log/slog"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher monitors for APT changes via inotify and periodic timer.
type Watcher struct {
	triggerCh       chan<- struct{}
	watchPath       string
	refreshInterval time.Duration
	logger          *slog.Logger
}

// New creates a new Watcher.
func New(triggerCh chan<- struct{}, watchPath string, interval time.Duration, logger *slog.Logger) *Watcher {
	return &Watcher{
		triggerCh:       triggerCh,
		watchPath:       watchPath,
		refreshInterval: interval,
		logger:          logger,
	}
}

// nonBlockingSend sends to triggerCh without blocking (coalescing multiple triggers).
func (w *Watcher) nonBlockingSend() {
	select {
	case w.triggerCh <- struct{}{}:
	default:
		// Already a trigger pending, skip.
	}
}

// Run starts both inotify and periodic timer. Blocks until ctx is cancelled.
func (w *Watcher) Run(ctx context.Context) error {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer fsWatcher.Close()

	if err := fsWatcher.Add(w.watchPath); err != nil {
		w.logger.Warn("failed to watch apt lists dir, continuing without inotify", "path", w.watchPath, "err", err)
	} else {
		w.logger.Info("watching for apt changes", "path", w.watchPath)
	}

	ticker := time.NewTicker(w.refreshInterval)
	defer ticker.Stop()

	var debounceTimer *time.Timer

	for {
		select {
		case <-ctx.Done():
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			return nil
		case event, ok := <-fsWatcher.Events:
			if !ok {
				continue
			}
			if event.Op&(fsnotify.Create|fsnotify.Write) != 0 {
				if debounceTimer != nil {
					debounceTimer.Stop()
				}
				debounceTimer = time.AfterFunc(5*time.Second, func() {
					w.logger.Debug("inotify trigger", "file", event.Name)
					w.nonBlockingSend()
				})
			}
		case err, ok := <-fsWatcher.Errors:
			if !ok {
				continue
			}
			w.logger.Error("fsnotify error", "err", err)
		case <-ticker.C:
			w.logger.Debug("periodic refresh trigger")
			w.nonBlockingSend()
		}
	}
}
