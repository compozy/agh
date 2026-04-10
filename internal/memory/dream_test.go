package memory

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/pedronauck/agh/internal/testutil"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func TestServiceConstructionDefaults(t *testing.T) {
	t.Parallel()

	service := NewService()

	if service.minHours != defaultMinHours {
		t.Fatalf("minHours = %v, want %v", service.minHours, defaultMinHours)
	}
	if service.minSessions != defaultMinSessions {
		t.Fatalf("minSessions = %d, want %d", service.minSessions, defaultMinSessions)
	}
	if service.logger == nil {
		t.Fatal("logger = nil, want non-nil")
	}
	if service.goal != defaultGoal {
		t.Fatalf("goal = %q, want %q", service.goal, defaultGoal)
	}
	if service.prompt != ConsolidationPrompt() {
		t.Fatal("prompt does not match embedded consolidation prompt")
	}
	if service.lock == nil {
		t.Fatal("lock = nil, want non-nil")
	}
}

func TestServiceConstructionOverridesDefaults(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	store := NewStore(filepath.Join(t.TempDir(), "memory"))

	service := NewService(
		WithMemoryStore(store),
		WithSessionsDir("/tmp/sessions"),
		WithLockPath("/tmp/lock"),
		WithMinHours(12),
		WithMinSessions(5),
		WithLogger(logger),
		withGoal("custom-goal"),
	)

	if service.memStore != store {
		t.Fatal("memStore was not applied")
	}
	if service.sessionsDir != "/tmp/sessions" {
		t.Fatalf("sessionsDir = %q, want /tmp/sessions", service.sessionsDir)
	}
	if service.lockPath != "/tmp/lock" {
		t.Fatalf("lockPath = %q, want /tmp/lock", service.lockPath)
	}
	if service.minHours != 12 {
		t.Fatalf("minHours = %v, want 12", service.minHours)
	}
	if service.minSessions != 5 {
		t.Fatalf("minSessions = %d, want 5", service.minSessions)
	}
	if service.logger != logger {
		t.Fatal("logger override was not applied")
	}
	if service.goal != "custom-goal" {
		t.Fatalf("goal = %q, want custom-goal", service.goal)
	}
}

func TestServiceShouldRunTimeGateFails(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Round(0)
	lock := &stubLock{lastConsolidatedAt: now.Add(-time.Hour)}
	sessionsScanned := 0
	service := NewService(
		withLock(lock),
		WithMinHours(24),
		WithMinSessions(3),
		withNow(func() time.Time { return now }),
		withSessionCounter(func(time.Time) (int, error) {
			sessionsScanned++
			return 10, nil
		}),
	)

	ok, err := service.ShouldRun()
	if err != nil {
		t.Fatalf("ShouldRun() error = %v", err)
	}
	if ok {
		t.Fatal("ShouldRun() = true, want false")
	}
	if sessionsScanned != 0 {
		t.Fatalf("session scans = %d, want 0", sessionsScanned)
	}
	if lock.tryAcquireCalls != 0 {
		t.Fatalf("lock acquisitions = %d, want 0", lock.tryAcquireCalls)
	}
}

func TestServiceShouldRunSessionGateFails(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Round(0)
	lock := &stubLock{lastConsolidatedAt: now.Add(-48 * time.Hour)}
	service := NewService(
		withLock(lock),
		WithMinHours(24),
		WithMinSessions(3),
		withNow(func() time.Time { return now }),
		withSessionCounter(func(time.Time) (int, error) {
			return 2, nil
		}),
	)

	ok, err := service.ShouldRun()
	if err != nil {
		t.Fatalf("ShouldRun() error = %v", err)
	}
	if ok {
		t.Fatal("ShouldRun() = true, want false")
	}
	if lock.tryAcquireCalls != 0 {
		t.Fatalf("lock acquisitions = %d, want 0", lock.tryAcquireCalls)
	}
}

func TestServiceShouldRunAllGatesPassWithoutLockSideEffects(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Round(0)
	lock := &stubLock{lastConsolidatedAt: now.Add(-48 * time.Hour)}
	service := NewService(
		withLock(lock),
		WithMinHours(24),
		WithMinSessions(3),
		withNow(func() time.Time { return now }),
		withSessionCounter(func(time.Time) (int, error) {
			return 3, nil
		}),
	)

	ok, err := service.ShouldRun()
	if err != nil {
		t.Fatalf("ShouldRun() error = %v", err)
	}
	if !ok {
		t.Fatal("ShouldRun() = false, want true")
	}
	if lock.tryAcquireCalls != 0 {
		t.Fatalf("lock acquisitions = %d, want 0", lock.tryAcquireCalls)
	}
	if service.pending {
		t.Fatal("service.pending = true, want false")
	}
	if !service.priorMtime.IsZero() {
		t.Fatalf("service.priorMtime = %v, want zero", service.priorMtime)
	}
}

func TestServiceShouldRunEvaluatesGatesInOrder(t *testing.T) {
	t.Parallel()

	t.Run("time gate short-circuits session and lock", func(t *testing.T) {
		now := time.Now().UTC().Round(0)
		lock := &stubLock{lastConsolidatedAt: now.Add(-time.Hour)}
		sequence := make([]string, 0, 2)
		service := NewService(
			withLock(lock),
			WithMinHours(24),
			WithMinSessions(3),
			withNow(func() time.Time { return now }),
			withSessionCounter(func(time.Time) (int, error) {
				sequence = append(sequence, "sessions")
				return 10, nil
			}),
		)

		ok, err := service.ShouldRun()
		if err != nil {
			t.Fatalf("ShouldRun() error = %v", err)
		}
		if ok {
			t.Fatal("ShouldRun() = true, want false")
		}
		if got := strings.Join(sequence, ","); got != "" {
			t.Fatalf("sequence = %q, want empty", got)
		}
		if lock.tryAcquireCalls != 0 {
			t.Fatalf("lock acquisitions = %d, want 0", lock.tryAcquireCalls)
		}
	})

	t.Run("session gate runs after time gate without touching lock", func(t *testing.T) {
		now := time.Now().UTC().Round(0)
		sequence := make([]string, 0, 1)
		lock := &stubLock{
			lastConsolidatedAt: now.Add(-48 * time.Hour),
		}
		service := NewService(
			withLock(lock),
			WithMinHours(24),
			WithMinSessions(1),
			withNow(func() time.Time { return now }),
			withSessionCounter(func(time.Time) (int, error) {
				sequence = append(sequence, "sessions")
				return 1, nil
			}),
		)

		ok, err := service.ShouldRun()
		if err != nil {
			t.Fatalf("ShouldRun() error = %v", err)
		}
		if !ok {
			t.Fatal("ShouldRun() = false, want true")
		}
		if got := strings.Join(sequence, ","); got != "sessions" {
			t.Fatalf("sequence = %q, want sessions", got)
		}
		if lock.tryAcquireCalls != 0 {
			t.Fatalf("lock acquisitions = %d, want 0", lock.tryAcquireCalls)
		}
	})
}

func TestServiceRunCallsSessionSpawnerWithGoalPromptAndWorkspaceID(t *testing.T) {
	t.Parallel()

	prior := time.Now().UTC().Add(-24 * time.Hour).Round(0)
	lock := &stubLock{
		tryAcquireFn: func() (time.Time, bool, error) {
			return prior, true, nil
		},
	}
	globalMemoryDir := filepath.Join(t.TempDir(), "memory")
	workspaceRoot := filepath.Join(t.TempDir(), "workspace")
	workspaceID := "ws-dream"
	service := NewService(
		withLock(lock),
		withGoal("custom-goal"),
		WithMemoryStore(NewStore(globalMemoryDir)),
		WithWorkspaceResolver(&fakeDreamWorkspaceResolver{
			resolved: workspacepkg.ResolvedWorkspace{
				Workspace: workspacepkg.Workspace{
					ID:      workspaceID,
					RootDir: workspaceRoot,
				},
			},
		}),
	)

	var gotGoal string
	var gotPrompt string
	var gotWorkspace string
	err := service.Run(testutil.Context(t), func(_ context.Context, goal, prompt, workspace string) error {
		gotGoal = goal
		gotPrompt = prompt
		gotWorkspace = workspace
		return nil
	}, workspaceID)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if gotGoal != "custom-goal" {
		t.Fatalf("goal = %q, want custom-goal", gotGoal)
	}
	if gotPrompt != ConsolidationPrompt() {
		t.Fatal("prompt passed to session spawner did not match embedded prompt")
	}
	if gotWorkspace != workspaceID {
		t.Fatalf("workspace = %q, want %q", gotWorkspace, workspaceID)
	}
	if _, err := os.Stat(filepath.Join(workspaceRoot, ".agh", "memory")); err != nil {
		t.Fatalf("workspace memory dir stat error = %v", err)
	}
	if lock.tryAcquireCalls != 1 {
		t.Fatalf("lock acquisitions = %d, want 1", lock.tryAcquireCalls)
	}
	if lock.releaseCalls != 1 {
		t.Fatalf("release calls = %d, want 1", lock.releaseCalls)
	}
}

func TestServiceRunRequiresWorkspaceResolverForExplicitWorkspace(t *testing.T) {
	t.Parallel()

	lock := &stubLock{
		tryAcquireFn: func() (time.Time, bool, error) {
			return time.Time{}, true, nil
		},
	}
	service := NewService(
		withLock(lock),
		WithMemoryStore(NewStore(filepath.Join(t.TempDir(), "memory"))),
	)

	err := service.Run(testutil.Context(t), func(context.Context, string, string, string) error { return nil }, "ws-missing")
	if err == nil {
		t.Fatal("Run() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "workspace resolver is required") {
		t.Fatalf("Run() error = %v, want workspace resolver error", err)
	}
	if len(lock.rollbackCalls) != 1 {
		t.Fatalf("rollback calls = %d, want 1", len(lock.rollbackCalls))
	}
}

func TestServiceRunResolvesWorkspaceRefBeforeSpawn(t *testing.T) {
	t.Parallel()

	lock := &stubLock{
		tryAcquireFn: func() (time.Time, bool, error) {
			return time.Time{}, true, nil
		},
	}
	resolver := fakeDreamWorkspaceResolver{
		resolved: workspacepkg.ResolvedWorkspace{
			Workspace: workspacepkg.Workspace{
				ID:      "ws-resolved",
				RootDir: filepath.Join(t.TempDir(), "workspace"),
			},
		},
	}
	service := NewService(
		withLock(lock),
		WithWorkspaceResolver(&resolver),
	)

	var gotWorkspace string
	err := service.Run(testutil.Context(t), func(_ context.Context, _, _, workspace string) error {
		gotWorkspace = workspace
		return nil
	}, "workspace-alias")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if gotWorkspace != "ws-resolved" {
		t.Fatalf("workspace = %q, want ws-resolved", gotWorkspace)
	}
	if got, want := resolver.resolveCalls, 1; got != want {
		t.Fatalf("resolver Resolve() calls = %d, want %d", got, want)
	}
	if got, want := resolver.lastArg, "workspace-alias"; got != want {
		t.Fatalf("resolver Resolve() arg = %q, want %q", got, want)
	}
}

func TestServiceRunWrapsWorkspaceResolveErrors(t *testing.T) {
	t.Parallel()

	lock := &stubLock{
		tryAcquireFn: func() (time.Time, bool, error) {
			return time.Time{}, true, nil
		},
	}
	resolveErr := errors.New("lookup failed")
	service := NewService(
		withLock(lock),
		WithWorkspaceResolver(&fakeDreamWorkspaceResolver{err: resolveErr}),
	)

	err := service.Run(testutil.Context(t), func(context.Context, string, string, string) error { return nil }, "workspace-alias")
	if err == nil {
		t.Fatal("Run() error = nil, want non-nil")
	}
	if !errors.Is(err, resolveErr) {
		t.Fatalf("Run() error = %v, want wrapped resolve error", err)
	}
	if !strings.Contains(err.Error(), `resolve workspace "workspace-alias"`) {
		t.Fatalf("Run() error = %v, want resolve workspace context", err)
	}
}

func TestServiceRunWrapsWorkspaceEnsureDirsErrors(t *testing.T) {
	t.Parallel()

	lock := &stubLock{
		tryAcquireFn: func() (time.Time, bool, error) {
			return time.Time{}, true, nil
		},
	}
	rootDir := filepath.Join(t.TempDir(), "workspace")
	service := NewService(
		withLock(lock),
		WithWorkspaceResolver(&fakeDreamWorkspaceResolver{
			resolved: workspacepkg.ResolvedWorkspace{
				Workspace: workspacepkg.Workspace{
					ID:      "ws-resolved",
					RootDir: rootDir,
				},
			},
		}),
		WithMemoryStore(NewStore("")),
	)

	err := service.Run(testutil.Context(t), func(context.Context, string, string, string) error { return nil }, "workspace-alias")
	if err == nil {
		t.Fatal("Run() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), `ensure workspace memory dirs for "`) {
		t.Fatalf("Run() error = %v, want ensure dirs context", err)
	}
	if !strings.Contains(err.Error(), rootDir) {
		t.Fatalf("Run() error = %v, want workspace root in wrapped error", err)
	}
}

func TestServiceRunRollsBackLockOnSessionSpawnerFailure(t *testing.T) {
	t.Parallel()

	prior := time.Now().UTC().Add(-24 * time.Hour).Round(0)
	lock := &stubLock{
		tryAcquireFn: func() (time.Time, bool, error) {
			return prior, true, nil
		},
	}
	service := NewService(withLock(lock))

	err := service.Run(testutil.Context(t), func(context.Context, string, string, string) error {
		return errors.New("boom")
	}, "")
	if err == nil {
		t.Fatal("Run() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("Run() error = %v, want wrapped spawner failure", err)
	}
	if len(lock.rollbackCalls) != 1 {
		t.Fatalf("rollback calls = %d, want 1", len(lock.rollbackCalls))
	}
	if !lock.rollbackCalls[0].Equal(prior) {
		t.Fatalf("rollback prior mtime = %v, want %v", lock.rollbackCalls[0], prior)
	}
}

func TestServiceRunReturnsJoinedSpawnAndRollbackErrors(t *testing.T) {
	t.Parallel()

	prior := time.Now().UTC().Add(-24 * time.Hour).Round(0)
	spawnErr := errors.New("spawn failed")
	rollbackErr := errors.New("rollback failed")
	lock := &stubLock{
		tryAcquireFn: func() (time.Time, bool, error) {
			return prior, true, nil
		},
		rollbackErr: rollbackErr,
	}
	service := NewService(withLock(lock))

	err := service.Run(testutil.Context(t), func(context.Context, string, string, string) error {
		return spawnErr
	}, "")
	if err == nil {
		t.Fatal("Run() error = nil, want non-nil")
	}
	if !errors.Is(err, spawnErr) {
		t.Fatalf("Run() error = %v, want errors.Is(spawnErr)", err)
	}
	if !errors.Is(err, rollbackErr) {
		t.Fatalf("Run() error = %v, want errors.Is(rollbackErr)", err)
	}
}

func TestServiceRunReturnsErrLockUnavailableWhenBusy(t *testing.T) {
	t.Parallel()

	lock := &stubLock{
		tryAcquireFn: func() (time.Time, bool, error) {
			return time.Time{}, false, nil
		},
	}
	service := NewService(withLock(lock))

	err := service.Run(testutil.Context(t), func(context.Context, string, string, string) error { return nil }, "")
	if !errors.Is(err, ErrLockUnavailable) {
		t.Fatalf("Run() error = %v, want ErrLockUnavailable", err)
	}
}

func TestServiceRunValidatesInputs(t *testing.T) {
	t.Parallel()

	service := NewService(withLock(&stubLock{
		tryAcquireFn: func() (time.Time, bool, error) {
			return time.Time{}, true, nil
		},
	}))

	nilContext := func() context.Context { return nil }

	if err := service.Run(nilContext(), func(context.Context, string, string, string) error { return nil }, ""); err == nil {
		t.Fatal("Run(nil context, spawner) error = nil, want non-nil")
	}
	if err := service.Run(testutil.Context(t), nil, ""); err == nil {
		t.Fatal("Run(ctx, nil) error = nil, want non-nil")
	}
}

func TestServiceOptionsClampNegativeThresholds(t *testing.T) {
	t.Parallel()

	service := NewService(
		WithMinHours(-5),
		WithMinSessions(-3),
	)

	if service.minHours != 0 {
		t.Fatalf("minHours = %v, want 0", service.minHours)
	}
	if service.minSessions != 0 {
		t.Fatalf("minSessions = %d, want 0", service.minSessions)
	}
}

func TestServiceValidateRequiresDependencies(t *testing.T) {
	t.Parallel()

	if err := (*Service)(nil).validate(); err == nil {
		t.Fatal("validate(nil) error = nil, want non-nil")
	}

	service := &Service{}
	if err := service.validate(); err == nil {
		t.Fatal("validate(empty service) error = nil, want non-nil")
	}

	service = NewService(withLock(&stubLock{}))
	service.logger = nil
	if err := service.validate(); err == nil {
		t.Fatal("validate(nil logger) error = nil, want non-nil")
	}
}

func TestServiceScanCompletedSessionsSinceFiltersByStoppedAt(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Round(0)
	root := t.TempDir()
	since := now.Add(-48 * time.Hour)
	writeSessionMeta(t, root, "equal", persistedSessionMetadata{StoppedAt: ptrTime(since)})
	writeSessionMeta(t, root, "after", persistedSessionMetadata{StoppedAt: ptrTime(since.Add(time.Minute))})
	writeSessionMeta(t, root, "before", persistedSessionMetadata{StoppedAt: ptrTime(since.Add(-time.Minute))})
	writeMalformedSessionMeta(t, root, "malformed")

	service := NewService(
		WithSessionsDir(root),
		WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))),
	)

	count, err := service.scanCompletedSessionsSince(since)
	if err != nil {
		t.Fatalf("scanCompletedSessionsSince() error = %v", err)
	}
	if count != 2 {
		t.Fatalf("scanCompletedSessionsSince() = %d, want 2", count)
	}
}

func TestServiceScanCompletedSessionsSinceIgnoresIncompleteSessions(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Round(0)
	root := t.TempDir()
	writeSessionMeta(t, root, "complete", persistedSessionMetadata{StoppedAt: ptrTime(now)})
	writeSessionMeta(t, root, "incomplete", persistedSessionMetadata{})

	service := NewService(
		WithSessionsDir(root),
		WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))),
	)

	count, err := service.scanCompletedSessionsSince(time.Time{})
	if err != nil {
		t.Fatalf("scanCompletedSessionsSince() error = %v", err)
	}
	if count != 1 {
		t.Fatalf("scanCompletedSessionsSince() = %d, want 1", count)
	}
}

func TestServiceScanCompletedSessionsSinceFallsBackToStoppedStateUpdatedAt(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Round(0)
	root := t.TempDir()
	writeSessionMeta(t, root, "stopped", persistedSessionMetadata{
		State:     "stopped",
		UpdatedAt: now,
	})
	writeSessionMeta(t, root, "active", persistedSessionMetadata{
		State:     "active",
		UpdatedAt: now,
	})

	service := NewService(
		WithSessionsDir(root),
		WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))),
	)

	count, err := service.scanCompletedSessionsSince(time.Time{})
	if err != nil {
		t.Fatalf("scanCompletedSessionsSince() error = %v", err)
	}
	if count != 1 {
		t.Fatalf("scanCompletedSessionsSince() = %d, want 1", count)
	}
}

func TestServiceScanCompletedSessionsSinceValidatesDirectoryAndMissingPath(t *testing.T) {
	t.Parallel()

	service := NewService(WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))))

	if _, err := service.scanCompletedSessionsSince(time.Time{}); err == nil {
		t.Fatal("scanCompletedSessionsSince() error = nil, want non-nil")
	}

	service.sessionsDir = filepath.Join(t.TempDir(), "missing")
	count, err := service.scanCompletedSessionsSince(time.Time{})
	if err != nil {
		t.Fatalf("scanCompletedSessionsSince() error = %v", err)
	}
	if count != 0 {
		t.Fatalf("scanCompletedSessionsSince() = %d, want 0", count)
	}
}

func TestServiceScanCompletedSessionsSinceFailsWhenPathIsFile(t *testing.T) {
	t.Parallel()

	service := NewService(WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))))
	path := filepath.Join(t.TempDir(), "sessions-file")
	if err := os.WriteFile(path, []byte("x"), filePerm); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	service.sessionsDir = path

	if _, err := service.scanCompletedSessionsSince(time.Time{}); err == nil {
		t.Fatal("scanCompletedSessionsSince() error = nil, want non-nil")
	}
}

func TestServiceTimeGatePassesForZeroTimestampAndZeroThreshold(t *testing.T) {
	t.Parallel()

	service := NewService(WithMinHours(0))
	if !service.timeGatePasses(time.Time{}) {
		t.Fatal("timeGatePasses(zero) = false, want true")
	}
	if !service.timeGatePasses(time.Now().UTC()) {
		t.Fatal("timeGatePasses(now) with zero threshold = false, want true")
	}
}

func TestServiceAcquireLockReturnsFalseWhenPending(t *testing.T) {
	t.Parallel()

	service := NewService(withLock(&stubLock{}))
	service.pending = true

	_, ok, err := service.acquireLock()
	if err != nil {
		t.Fatalf("acquireLock() error = %v", err)
	}
	if ok {
		t.Fatal("acquireLock() ok = true, want false")
	}
}

func TestServiceCompleteRunReturnsReleaseError(t *testing.T) {
	t.Parallel()

	releaseErr := errors.New("release failed")
	service := NewService(withLock(&stubLock{releaseErr: releaseErr}))
	service.pending = true

	err := service.completeRun(true, time.Time{})
	if !errors.Is(err, releaseErr) {
		t.Fatalf("completeRun(true) error = %v, want release error", err)
	}
	if service.pending {
		t.Fatal("service.pending = true, want false")
	}
}

func TestServiceRunSerializesConcurrentCalls(t *testing.T) {
	t.Parallel()

	lock := &stubLock{
		tryAcquireFn: func() (time.Time, bool, error) {
			return time.Time{}, true, nil
		},
	}
	service := NewService(withLock(lock))

	var active atomic.Int32
	var maxActive atomic.Int32
	started := make(chan struct{}, 2)
	releaseFirst := make(chan struct{})
	errCh := make(chan error, 2)
	var first sync.Once

	spawner := func(context.Context, string, string, string) error {
		current := active.Add(1)
		for {
			previous := maxActive.Load()
			if current <= previous || maxActive.CompareAndSwap(previous, current) {
				break
			}
		}

		started <- struct{}{}
		shouldBlock := false
		first.Do(func() { shouldBlock = true })
		if shouldBlock {
			<-releaseFirst
		}

		active.Add(-1)
		return nil
	}

	go func() {
		errCh <- service.Run(testutil.Context(t), spawner, "")
	}()
	<-started

	go func() {
		errCh <- service.Run(testutil.Context(t), spawner, "")
	}()

	select {
	case <-started:
		t.Fatal("second spawner started before first run finished")
	case <-time.After(150 * time.Millisecond):
	}

	close(releaseFirst)
	<-started

	for range 2 {
		if err := <-errCh; err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	}
	if got := maxActive.Load(); got != 1 {
		t.Fatalf("max concurrent spawners = %d, want 1", got)
	}
}

type stubLock struct {
	lastConsolidatedAt time.Time
	tryAcquireFn       func() (time.Time, bool, error)
	releaseErr         error
	rollbackErr        error

	tryAcquireCalls int
	releaseCalls    int
	rollbackCalls   []time.Time
}

func (s *stubLock) LastConsolidatedAt() (time.Time, error) {
	return s.lastConsolidatedAt, nil
}

func (s *stubLock) TryAcquire() (time.Time, bool, error) {
	s.tryAcquireCalls++
	if s.tryAcquireFn != nil {
		return s.tryAcquireFn()
	}
	return time.Time{}, true, nil
}

func (s *stubLock) Release() error {
	s.releaseCalls++
	return s.releaseErr
}

func (s *stubLock) Rollback(priorMtime time.Time) error {
	s.rollbackCalls = append(s.rollbackCalls, priorMtime)
	return s.rollbackErr
}

func writeSessionMeta(t *testing.T, sessionsDir string, sessionID string, meta persistedSessionMetadata) {
	t.Helper()

	sessionDir := filepath.Join(sessionsDir, sessionID)
	if err := os.MkdirAll(sessionDir, dirPerm); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}

	payload, err := json.Marshal(meta)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	if err := os.WriteFile(filepath.Join(sessionDir, "meta.json"), payload, filePerm); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
}

func writeMalformedSessionMeta(t *testing.T, sessionsDir string, sessionID string) {
	t.Helper()

	sessionDir := filepath.Join(sessionsDir, sessionID)
	if err := os.MkdirAll(sessionDir, dirPerm); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(sessionDir, "meta.json"), []byte("{bad json"), filePerm); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
}

func ptrTime(value time.Time) *time.Time {
	return &value
}

type fakeDreamWorkspaceResolver struct {
	resolved workspacepkg.ResolvedWorkspace
	err      error

	resolveCalls int
	lastArg      string
}

func (r *fakeDreamWorkspaceResolver) Resolve(_ context.Context, arg string) (workspacepkg.ResolvedWorkspace, error) {
	if r.err != nil {
		return workspacepkg.ResolvedWorkspace{}, r.err
	}
	r.resolveCalls++
	r.lastArg = strings.TrimSpace(arg)
	return r.resolved, nil
}

func (r *fakeDreamWorkspaceResolver) ResolveOrRegister(context.Context, string) (workspacepkg.ResolvedWorkspace, error) {
	if r.err != nil {
		return workspacepkg.ResolvedWorkspace{}, r.err
	}
	return r.resolved, nil
}
