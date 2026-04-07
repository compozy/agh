package memory

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/procutil"
)

const (
	consolidationLockName = ".consolidate-lock"
	lockFilePerm          = 0o644
	defaultLockStaleAge   = time.Hour
)

// ConsolidationLock coordinates cross-process consolidation runs via a PID file
// whose mtime doubles as the last successful consolidation timestamp.
type ConsolidationLock struct {
	path         string
	staleAge     time.Duration
	pidFn        func() int
	now          func() time.Time
	processAlive func(int) bool
}

// NewConsolidationLock constructs a lock rooted at path with the default stale age.
func NewConsolidationLock(path string) *ConsolidationLock {
	return &ConsolidationLock{
		path:     strings.TrimSpace(path),
		staleAge: defaultLockStaleAge,
		pidFn:    os.Getpid,
		now: func() time.Time {
			return time.Now().UTC()
		},
		processAlive: procutil.Alive,
	}
}

// LastConsolidatedAt returns the lock file mtime, or the zero time when the lock file does not exist.
func (l *ConsolidationLock) LastConsolidatedAt() (time.Time, error) {
	if err := l.validate(); err != nil {
		return time.Time{}, err
	}

	info, err := os.Stat(l.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return time.Time{}, nil
		}
		return time.Time{}, fmt.Errorf("memory: stat consolidation lock %q: %w", l.path, err)
	}

	return info.ModTime(), nil
}

// TryAcquire attempts to acquire the consolidation lock and returns the prior
// mtime for rollback when acquisition succeeds.
func (l *ConsolidationLock) TryAcquire() (time.Time, bool, error) {
	if err := l.validate(); err != nil {
		return time.Time{}, false, err
	}
	if err := os.MkdirAll(filepath.Dir(l.path), dirPerm); err != nil {
		return time.Time{}, false, fmt.Errorf("memory: create lock parent directory for %q: %w", l.path, err)
	}

	state, err := l.readState()
	if err != nil {
		return time.Time{}, false, err
	}
	if state.exists && state.validPID && !l.canReclaim(state) {
		return state.modTime, false, nil
	}

	priorMtime := state.modTime
	if state.exists {
		if err := os.Remove(l.path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return priorMtime, false, fmt.Errorf("memory: remove stale consolidation lock %q: %w", l.path, err)
		}
	}

	pid := l.pidFn()
	if pid <= 0 {
		restoreErr := l.restoreUnlocked(priorMtime)
		return priorMtime, false, errors.Join(
			fmt.Errorf("memory: invalid process pid %d", pid),
			restoreErr,
		)
	}

	if err := l.createLockFile(pid); err != nil {
		if errors.Is(err, os.ErrExist) {
			return priorMtime, false, nil
		}
		restoreErr := l.restoreUnlocked(priorMtime)
		return priorMtime, false, errors.Join(
			fmt.Errorf("memory: create consolidation lock %q: %w", l.path, err),
			restoreErr,
		)
	}

	verified, err := l.readState()
	if err != nil {
		restoreErr := l.restoreUnlocked(priorMtime)
		return priorMtime, false, errors.Join(err, restoreErr)
	}
	if !verified.exists || !verified.validPID || verified.pid != pid {
		restoreErr := l.restoreUnlocked(priorMtime)
		return priorMtime, false, errors.Join(
			fmt.Errorf("memory: verify consolidation lock %q: ownership lost to pid %d", l.path, verified.pid),
			restoreErr,
		)
	}

	return priorMtime, true, nil
}

// Release clears the lock PID and leaves the file mtime at the release time.
func (l *ConsolidationLock) Release() error {
	if err := l.validate(); err != nil {
		return err
	}

	return l.writeUnlockedAt(l.now())
}

// Rollback clears the lock PID and restores the mtime from before acquisition.
func (l *ConsolidationLock) Rollback(priorMtime time.Time) error {
	if err := l.validate(); err != nil {
		return err
	}
	if priorMtime.IsZero() {
		if err := os.Remove(l.path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("memory: remove consolidation lock %q during rollback: %w", l.path, err)
		}
		return nil
	}

	return l.writeUnlockedAt(priorMtime)
}

type lockState struct {
	exists   bool
	modTime  time.Time
	rawPID   string
	pid      int
	validPID bool
}

func (l *ConsolidationLock) validate() error {
	switch {
	case l == nil:
		return errors.New("memory: consolidation lock is required")
	case strings.TrimSpace(l.path) == "":
		return errors.New("memory: consolidation lock path is required")
	case l.pidFn == nil:
		return errors.New("memory: consolidation lock pid function is required")
	case l.now == nil:
		return errors.New("memory: consolidation lock clock is required")
	case l.processAlive == nil:
		return errors.New("memory: consolidation lock process liveness function is required")
	default:
		return nil
	}
}

func (l *ConsolidationLock) readState() (lockState, error) {
	info, err := os.Stat(l.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return lockState{}, nil
		}
		return lockState{}, fmt.Errorf("memory: stat consolidation lock %q: %w", l.path, err)
	}

	data, err := os.ReadFile(l.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return lockState{}, nil
		}
		return lockState{}, fmt.Errorf("memory: read consolidation lock %q: %w", l.path, err)
	}

	state := lockState{
		exists:  true,
		modTime: info.ModTime(),
		rawPID:  strings.TrimSpace(string(data)),
	}
	if state.rawPID == "" {
		return state, nil
	}

	pid, err := strconv.Atoi(state.rawPID)
	if err != nil {
		return state, nil
	}

	state.pid = pid
	state.validPID = pid > 0
	return state, nil
}

func (l *ConsolidationLock) canReclaim(state lockState) bool {
	if !state.validPID {
		return true
	}
	if l.staleAge > 0 && l.now().Sub(state.modTime) > l.staleAge {
		return true
	}
	return !l.processAlive(state.pid)
}

func (l *ConsolidationLock) restoreUnlocked(priorMtime time.Time) error {
	if priorMtime.IsZero() {
		if err := os.Remove(l.path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("memory: remove consolidation lock %q during restore: %w", l.path, err)
		}
		return nil
	}

	return l.writeUnlockedAt(priorMtime)
}

func (l *ConsolidationLock) writeUnlockedAt(timestamp time.Time) error {
	if err := os.MkdirAll(filepath.Dir(l.path), dirPerm); err != nil {
		return fmt.Errorf("memory: create lock parent directory for %q: %w", l.path, err)
	}
	if err := os.WriteFile(l.path, nil, lockFilePerm); err != nil {
		return fmt.Errorf("memory: write unlocked consolidation lock %q: %w", l.path, err)
	}
	if err := os.Chtimes(l.path, timestamp, timestamp); err != nil {
		return fmt.Errorf("memory: restore consolidation lock mtime for %q: %w", l.path, err)
	}

	return nil
}

func (l *ConsolidationLock) createLockFile(pid int) error {
	tempFile, err := os.CreateTemp(filepath.Dir(l.path), ".consolidate-lock-*")
	if err != nil {
		return err
	}

	tempPath := tempFile.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tempPath)
		}
	}()

	if _, err := tempFile.WriteString(strconv.Itoa(pid)); err != nil {
		_ = tempFile.Close()
		return err
	}
	if err := tempFile.Chmod(lockFilePerm); err != nil {
		_ = tempFile.Close()
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}
	if err := os.Link(tempPath, l.path); err != nil {
		return err
	}

	cleanup = false
	return os.Remove(tempPath)
}
