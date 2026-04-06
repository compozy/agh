package skills

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
)

const defaultWatcherInterval = 3 * time.Second

type globalRefresher interface {
	RefreshGlobal(ctx context.Context) error
}

type fileChange struct {
	path   string
	action string
}

// Watcher polls global skill directories and refreshes the registry when their
// SKILL.md snapshots change.
type Watcher struct {
	registry globalRefresher
	interval time.Duration
	roots    []string
	logger   *slog.Logger

	mu          sync.Mutex
	initialized bool
	snapshots   map[string]fileSnapshot
}

// NewWatcher constructs a watcher that polls the registry's global skill
// directories. A non-positive interval falls back to the default poll interval.
func NewWatcher(registry *Registry, interval time.Duration) *Watcher {
	var roots []string
	if registry != nil {
		roots = watcherRoots(registry.cfg.UserSkillsDir, registry.cfg.UserAgentsDir)
	}

	return &Watcher{
		registry:  registry,
		interval:  watcherInterval(interval),
		roots:     roots,
		logger:    slog.Default(),
		snapshots: make(map[string]fileSnapshot),
	}
}

func newWatcher(registry globalRefresher, interval time.Duration, roots []string) *Watcher {
	return &Watcher{
		registry:  registry,
		interval:  watcherInterval(interval),
		roots:     watcherRoots(roots...),
		logger:    slog.Default(),
		snapshots: make(map[string]fileSnapshot),
	}
}

// Start runs the polling loop until the provided context is cancelled.
func (w *Watcher) Start(ctx context.Context) {
	if ctx == nil {
		return
	}

	if w.logger == nil {
		w.logger = slog.Default()
	}

	w.logger.Info("skills: watcher started", "roots", w.roots, "interval", w.interval)
	if err := w.pollOnce(ctx); err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		w.logger.Warn("skills: watcher poll failed", "error", err)
	}

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.pollOnce(ctx); err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return
				}
				w.logger.Warn("skills: watcher poll failed", "error", err)
			}
		}
	}
}

func (w *Watcher) pollOnce(ctx context.Context) error {
	changed, snapshots, changes, err := w.detectChanges(ctx)
	if err != nil {
		return err
	}
	if !changed {
		return nil
	}

	for _, change := range changes {
		w.logger.Debug("skills: watcher detected change", "path", change.path, "action", change.action)
	}

	if w.registry != nil {
		if err := w.registry.RefreshGlobal(ctx); err != nil {
			return fmt.Errorf("skills: refresh global registry: %w", err)
		}
	}

	w.commitSnapshots(snapshots)
	return nil
}

func (w *Watcher) detectChanges(ctx context.Context) (bool, map[string]fileSnapshot, []fileChange, error) {
	if err := checkRegistryContext(ctx); err != nil {
		return false, nil, nil, err
	}

	current, err := w.snapshotRoots(ctx)
	if err != nil {
		return false, nil, nil, err
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.initialized {
		w.snapshots = current
		w.initialized = true
		return false, nil, nil, nil
	}

	changes := diffSnapshots(w.snapshots, current)
	if len(changes) == 0 {
		return false, nil, nil, nil
	}

	return true, current, changes, nil
}

func (w *Watcher) commitSnapshots(snapshots map[string]fileSnapshot) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.snapshots = snapshots
	if w.snapshots == nil {
		w.snapshots = make(map[string]fileSnapshot)
	}
	w.initialized = true
}

func (w *Watcher) snapshotRoots(ctx context.Context) (map[string]fileSnapshot, error) {
	snapshots := make(map[string]fileSnapshot)
	for _, root := range w.roots {
		if err := checkRegistryContext(ctx); err != nil {
			return nil, err
		}

		paths, err := scanDirectory(root)
		if err != nil {
			return nil, fmt.Errorf("skills: scan watcher root %q: %w", root, err)
		}

		for _, skillPath := range paths {
			if err := checkRegistryContext(ctx); err != nil {
				return nil, err
			}

			snapshot, err := snapshotFile(skillPath)
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					continue
				}
				return nil, fmt.Errorf("skills: snapshot watcher file %q: %w", skillPath, err)
			}

			snapshots[skillPath] = snapshot
		}
	}

	return snapshots, nil
}

func diffSnapshots(previous, current map[string]fileSnapshot) []fileChange {
	changes := make([]fileChange, 0)

	for path, snapshot := range current {
		previousSnapshot, ok := previous[path]
		if !ok {
			changes = append(changes, fileChange{path: path, action: "added"})
			continue
		}

		if snapshot.size != previousSnapshot.size || !snapshot.modTime.Equal(previousSnapshot.modTime) {
			changes = append(changes, fileChange{path: path, action: "modified"})
		}
	}

	for path := range previous {
		if _, ok := current[path]; ok {
			continue
		}
		changes = append(changes, fileChange{path: path, action: "deleted"})
	}

	slices.SortFunc(changes, func(left, right fileChange) int {
		if cmp := strings.Compare(left.path, right.path); cmp != 0 {
			return cmp
		}
		return strings.Compare(left.action, right.action)
	})

	return changes
}

func watcherInterval(interval time.Duration) time.Duration {
	if interval <= 0 {
		return defaultWatcherInterval
	}

	return interval
}

func watcherRoots(roots ...string) []string {
	if len(roots) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(roots))
	seen := make(map[string]struct{}, len(roots))
	for _, root := range roots {
		trimmed := strings.TrimSpace(root)
		if trimmed == "" {
			continue
		}

		absRoot, err := filepath.Abs(trimmed)
		if err != nil {
			absRoot = trimmed
		}
		if _, ok := seen[absRoot]; ok {
			continue
		}

		seen[absRoot] = struct{}{}
		normalized = append(normalized, absRoot)
	}

	slices.Sort(normalized)
	return normalized
}
