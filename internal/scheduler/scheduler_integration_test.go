//go:build integration

package scheduler

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/pedronauck/agh/internal/network"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/store/globaldb"
	taskpkg "github.com/pedronauck/agh/internal/task"
	"github.com/pedronauck/agh/internal/testutil"
	aghworkspace "github.com/pedronauck/agh/internal/workspace"
)

func TestSchedulerWakeLeavesClaimToTaskServiceIntegration(t *testing.T) {
	t.Parallel()

	t.Run("Should wake an eligible session without claiming the run for it", func(t *testing.T) {
		ctx := testutil.Context(t)
		base := time.Date(2026, 4, 26, 14, 0, 0, 0, time.UTC)
		db := openSchedulerGlobalDB(t, filepath.Join(t.TempDir(), "agh.db"))
		workspaceID := registerSchedulerWorkspace(t, db, "wake-claim", filepath.Join(t.TempDir(), "workspace"))
		manager := newSchedulerTaskManager(t, db)
		execution := createSchedulerTaskRun(t, ctx, manager, workspaceID, "Wake then claim")
		runChannel := execution.Run.CoordinationChannelID
		if runChannel == "" {
			t.Fatal("execution.Run.CoordinationChannelID = empty, want derived channel")
		}
		run := execution.Run
		run.RequiredCapabilities = []string{"go"}
		if err := db.UpdateTaskRun(ctx, run); err != nil {
			t.Fatalf("UpdateTaskRun(required capabilities) error = %v", err)
		}
		waker := &fakeWaker{}
		scheduler := newTestScheduler(
			t,
			integrationTaskSource{manager: manager, store: db},
			&fakeSessionSource{sessions: []SessionSnapshot{
				integrationSessionSnapshot("sess-worker", workspaceID, runChannel, "active", false, []string{"go", "sqlite"}, base),
			}},
			waker,
			WithClock(clockwork.NewFakeClockAt(base)),
		)

		before, err := db.GetTaskRun(ctx, run.ID)
		if err != nil {
			t.Fatalf("GetTaskRun(before) error = %v", err)
		}
		if before.Status != taskpkg.TaskRunStatusQueued || before.SessionID != "" {
			t.Fatalf("before run = %#v, want queued and unowned", before)
		}

		result, err := scheduler.RunOnce(ctx)
		if err != nil {
			t.Fatalf("RunOnce() error = %v", err)
		}
		if result.WakeSucceeded != 1 {
			t.Fatalf("WakeSucceeded = %d, want 1 (result %#v)", result.WakeSucceeded, result)
		}
		targets := waker.targetsSnapshot()
		if got, want := len(targets), 1; got != want {
			t.Fatalf("wake targets = %d, want %d", got, want)
		}
		if got, want := targets[0].Work.Run.ID, run.ID; got != want {
			t.Fatalf("woken run = %q, want %q", got, want)
		}

		afterWake, err := db.GetTaskRun(ctx, run.ID)
		if err != nil {
			t.Fatalf("GetTaskRun(after wake) error = %v", err)
		}
		if afterWake.Status != taskpkg.TaskRunStatusQueued || afterWake.SessionID != "" {
			t.Fatalf("after wake run = %#v, want scheduler to leave queued ownership untouched", afterWake)
		}

		claimActor, err := taskpkg.DeriveAgentSessionActorContext("sess-worker")
		if err != nil {
			t.Fatalf("DeriveAgentSessionActorContext() error = %v", err)
		}
		claim, err := manager.ClaimNextRun(ctx, taskpkg.ClaimCriteria{
			Scope:                 taskpkg.ScopeWorkspace,
			WorkspaceID:           workspaceID,
			ClaimerSessionID:      "sess-worker",
			CoordinationChannelID: runChannel,
			RequiredCapabilities:  []string{"go"},
			LeaseDuration:         time.Minute,
			Now:                   base.Add(time.Second),
		}, claimActor)
		if err != nil {
			t.Fatalf("ClaimNextRun() error = %v", err)
		}
		if got, want := claim.Run.ID, run.ID; got != want {
			t.Fatalf("ClaimNextRun().Run.ID = %q, want %q", got, want)
		}
	})
}

func TestSchedulerRecoversExpiredLeaseAfterDatabaseRestartIntegration(t *testing.T) {
	t.Parallel()

	t.Run("Should recover an expired lease after restart and make the run claimable again", func(t *testing.T) {
		ctx := testutil.Context(t)
		base := time.Date(2026, 4, 26, 15, 0, 0, 0, time.UTC)
		dbPath := filepath.Join(t.TempDir(), "agh.db")
		first := openSchedulerGlobalDB(t, dbPath)
		workspaceID := registerSchedulerWorkspace(t, first, "restart-recovery", filepath.Join(t.TempDir(), "workspace"))
		firstManager := newSchedulerTaskManager(t, first)
		execution := createSchedulerTaskRun(t, ctx, firstManager, workspaceID, "Restart recovery")
		runChannel := execution.Run.CoordinationChannelID
		if runChannel == "" {
			t.Fatal("execution.Run.CoordinationChannelID = empty, want derived channel")
		}

		oldActor, err := taskpkg.DeriveAgentSessionActorContext("sess-old")
		if err != nil {
			t.Fatalf("DeriveAgentSessionActorContext(old) error = %v", err)
		}
		claimed, err := firstManager.ClaimNextRun(ctx, taskpkg.ClaimCriteria{
			Scope:                 taskpkg.ScopeWorkspace,
			WorkspaceID:           workspaceID,
			ClaimerSessionID:      "sess-old",
			CoordinationChannelID: runChannel,
			LeaseDuration:         time.Second,
			Now:                   base,
		}, oldActor)
		if err != nil {
			t.Fatalf("ClaimNextRun(old) error = %v", err)
		}
		if got, want := claimed.Run.ID, execution.Run.ID; got != want {
			t.Fatalf("old claim run = %q, want %q", got, want)
		}
		if err := first.Close(ctx); err != nil {
			t.Fatalf("first Close() error = %v", err)
		}

		second := openSchedulerGlobalDB(t, dbPath)
		secondManager := newSchedulerTaskManager(t, second)
		waker := &fakeWaker{}
		scheduler := newTestScheduler(
			t,
			integrationTaskSource{manager: secondManager, store: second},
			&fakeSessionSource{sessions: []SessionSnapshot{
				integrationSessionSnapshot("sess-new", workspaceID, runChannel, "active", false, nil, base.Add(2*time.Second)),
			}},
			waker,
			WithClock(clockwork.NewFakeClockAt(base.Add(2*time.Second))),
		)

		result, err := scheduler.RunOnce(ctx)
		if err != nil {
			t.Fatalf("RunOnce() error = %v", err)
		}
		if result.RecoveredLeases != 1 {
			t.Fatalf("RecoveredLeases = %d, want 1 (result %#v)", result.RecoveredLeases, result)
		}
		if !slices.Contains(result.RecoveredRunIDs, execution.Run.ID) {
			t.Fatalf("RecoveredRunIDs = %v, want %q", result.RecoveredRunIDs, execution.Run.ID)
		}
		if got := len(waker.targetsSnapshot()); got != 1 {
			t.Fatalf("wake targets after recovery = %d, want 1", got)
		}

		events, err := second.ListTaskEvents(ctx, taskpkg.EventQuery{TaskID: execution.Task.ID})
		if err != nil {
			t.Fatalf("ListTaskEvents() error = %v", err)
		}
		if !schedulerIntegrationHasEvent(events, "task.run_lease_expired") {
			t.Fatalf("event types = %v, want task.run_lease_expired", schedulerIntegrationEventTypes(events))
		}
		for _, event := range events {
			if strings.HasPrefix(event.EventType, "scheduler.") {
				t.Fatalf("unexpected scheduler hook-like event persisted: %#v", event)
			}
		}

		newActor, err := taskpkg.DeriveAgentSessionActorContext("sess-new")
		if err != nil {
			t.Fatalf("DeriveAgentSessionActorContext(new) error = %v", err)
		}
		claim, err := secondManager.ClaimNextRun(ctx, taskpkg.ClaimCriteria{
			Scope:                 taskpkg.ScopeWorkspace,
			WorkspaceID:           workspaceID,
			ClaimerSessionID:      "sess-new",
			CoordinationChannelID: runChannel,
			LeaseDuration:         time.Minute,
			Now:                   base.Add(3 * time.Second),
		}, newActor)
		if err != nil {
			t.Fatalf("ClaimNextRun(new) error = %v", err)
		}
		if got, want := claim.Run.ID, execution.Run.ID; got != want {
			t.Fatalf("new claim run = %q, want %q", got, want)
		}
	})
}

func TestSchedulerRecoversExpiredHistoricalNetworkLeaseIntegration(t *testing.T) {
	t.Parallel()

	t.Run("Should recover an expired historical network-channel lease and preserve reclaim semantics", func(t *testing.T) {
		ctx := testutil.Context(t)
		base := time.Date(2027, 4, 28, 9, 46, 36, 0, time.UTC)
		db := openSchedulerGlobalDB(t, filepath.Join(t.TempDir(), "agh.db"))
		workspaceID := registerSchedulerWorkspace(t, db, "historical-expiry", filepath.Join(t.TempDir(), "workspace"))
		manager := newSchedulerTaskManagerWithOptions(
			t,
			db,
			taskpkg.WithNetworkChannelValidator(network.ValidateChannel),
		)

		channelTimestamp := base.Add(-3 * time.Second)
		if err := db.WriteNetworkChannel(ctx, store.NetworkChannelEntry{
			Channel:     "scope-direct-history",
			WorkspaceID: workspaceID,
			Purpose:     "Historical channel lease expiry recovery validation",
			CreatedBy:   "founder",
			CreatedAt:   channelTimestamp,
			UpdatedAt:   channelTimestamp,
		}); err != nil {
			t.Fatalf("WriteNetworkChannel() error = %v", err)
		}

		operator, err := taskpkg.DeriveHumanActorContext("operator", taskpkg.OriginKindCLI, "agh task start")
		if err != nil {
			t.Fatalf("DeriveHumanActorContext() error = %v", err)
		}
		taskRecord, err := manager.CreateTask(ctx, taskpkg.CreateTask{
			Scope:          taskpkg.ScopeWorkspace,
			WorkspaceID:    workspaceID,
			NetworkChannel: "scope-direct-history",
			Title:          "Historical lease expiry recovery",
		}, operator)
		if err != nil {
			t.Fatalf("CreateTask() error = %v", err)
		}
		execution, err := manager.StartTask(ctx, taskRecord.ID, taskpkg.ExecutionRequest{}, operator)
		if err != nil {
			t.Fatalf("StartTask() error = %v", err)
		}
		if got, want := execution.Run.NetworkChannel, "scope-direct-history"; got != want {
			t.Fatalf("execution.Run.NetworkChannel = %q, want %q", got, want)
		}
		if got, want := execution.Run.CoordinationChannelID, "scope-direct-history"; got != want {
			t.Fatalf("execution.Run.CoordinationChannelID = %q, want %q", got, want)
		}

		oldActor, err := taskpkg.DeriveAgentSessionActorContext("sess-old")
		if err != nil {
			t.Fatalf("DeriveAgentSessionActorContext(old) error = %v", err)
		}
		firstClaim, err := manager.ClaimNextRun(ctx, taskpkg.ClaimCriteria{
			Scope:                 taskpkg.ScopeWorkspace,
			WorkspaceID:           workspaceID,
			ClaimerSessionID:      "sess-old",
			CoordinationChannelID: "scope-direct-history",
			LeaseDuration:         time.Second,
			Now:                   base,
		}, oldActor)
		if err != nil {
			t.Fatalf("ClaimNextRun(old) error = %v", err)
		}
		if got, want := firstClaim.Run.ID, execution.Run.ID; got != want {
			t.Fatalf("firstClaim.Run.ID = %q, want %q", got, want)
		}
		if got, want := firstClaim.Run.NetworkChannel, "scope-direct-history"; got != want {
			t.Fatalf("firstClaim.Run.NetworkChannel = %q, want %q", got, want)
		}
		if got, want := firstClaim.Run.CoordinationChannelID, "scope-direct-history"; got != want {
			t.Fatalf("firstClaim.Run.CoordinationChannelID = %q, want %q", got, want)
		}
		if firstClaim.ClaimToken == "" {
			t.Fatal("firstClaim.ClaimToken = empty, want raw claim token")
		}

		waker := &fakeWaker{}
		scheduler := newTestScheduler(
			t,
			integrationTaskSource{manager: manager, store: db},
			&fakeSessionSource{},
			waker,
			WithClock(clockwork.NewFakeClockAt(base.Add(12*time.Second))),
		)

		result, err := scheduler.RunOnce(ctx)
		if err != nil {
			t.Fatalf("RunOnce() error = %v", err)
		}
		if got, want := result.RecoveredLeases, 1; got != want {
			t.Fatalf("RecoveredLeases = %d, want %d (result %#v)", got, want, result)
		}
		if !slices.Contains(result.RecoveredRunIDs, execution.Run.ID) {
			t.Fatalf("RecoveredRunIDs = %v, want %q", result.RecoveredRunIDs, execution.Run.ID)
		}
		if got := len(waker.targetsSnapshot()); got != 0 {
			t.Fatalf("wake targets after historical recovery = %d, want 0", got)
		}

		if _, err := manager.HeartbeatRunLease(ctx, taskpkg.LeaseHeartbeat{
			RunID:         execution.Run.ID,
			ClaimToken:    firstClaim.ClaimToken,
			LeaseDuration: time.Minute,
			Now:           base.Add(13 * time.Second),
		}, oldActor); !errors.Is(err, taskpkg.ErrInvalidClaimToken) {
			t.Fatalf("HeartbeatRunLease(stale recovered lease) error = %v, want %v", err, taskpkg.ErrInvalidClaimToken)
		}

		newActor, err := taskpkg.DeriveAgentSessionActorContext("sess-new")
		if err != nil {
			t.Fatalf("DeriveAgentSessionActorContext(new) error = %v", err)
		}
		secondClaim, err := manager.ClaimNextRun(ctx, taskpkg.ClaimCriteria{
			Scope:                 taskpkg.ScopeWorkspace,
			WorkspaceID:           workspaceID,
			ClaimerSessionID:      "sess-new",
			CoordinationChannelID: "scope-direct-history",
			LeaseDuration:         time.Minute,
			Now:                   base.Add(14 * time.Second),
		}, newActor)
		if err != nil {
			t.Fatalf("ClaimNextRun(new) error = %v", err)
		}
		if got, want := secondClaim.Run.ID, execution.Run.ID; got != want {
			t.Fatalf("secondClaim.Run.ID = %q, want %q", got, want)
		}
		if got, want := secondClaim.Run.SessionID, "sess-new"; got != want {
			t.Fatalf("secondClaim.Run.SessionID = %q, want %q", got, want)
		}
		if got, want := secondClaim.Run.NetworkChannel, "scope-direct-history"; got != want {
			t.Fatalf("secondClaim.Run.NetworkChannel = %q, want %q", got, want)
		}
		if got, want := secondClaim.Run.CoordinationChannelID, "scope-direct-history"; got != want {
			t.Fatalf("secondClaim.Run.CoordinationChannelID = %q, want %q", got, want)
		}
		if secondClaim.ClaimToken == "" {
			t.Fatal("secondClaim.ClaimToken = empty, want raw claim token")
		}

		completed, err := manager.CompleteRunLease(ctx, taskpkg.LeaseCompletion{
			RunID:      secondClaim.Run.ID,
			ClaimToken: secondClaim.ClaimToken,
			Result: taskpkg.RunResult{
				Value: []byte(`{"ok":true,"path":"scheduler-historical-expiry"}`),
			},
		}, newActor)
		if err != nil {
			t.Fatalf("CompleteRunLease() error = %v", err)
		}
		if got, want := completed.Status, taskpkg.TaskRunStatusCompleted; got != want {
			t.Fatalf("completed.Status = %q, want %q", got, want)
		}
		if got, want := completed.NetworkChannel, "scope-direct-history"; got != want {
			t.Fatalf("completed.NetworkChannel = %q, want %q", got, want)
		}
		if got, want := completed.CoordinationChannelID, "scope-direct-history"; got != want {
			t.Fatalf("completed.CoordinationChannelID = %q, want %q", got, want)
		}

		storedTask, err := db.GetTask(ctx, taskRecord.ID)
		if err != nil {
			t.Fatalf("GetTask() error = %v", err)
		}
		if got, want := storedTask.Status, taskpkg.TaskStatusCompleted; got != want {
			t.Fatalf("storedTask.Status = %q, want %q", got, want)
		}

		events, err := db.ListTaskEvents(ctx, taskpkg.EventQuery{TaskID: taskRecord.ID})
		if err != nil {
			t.Fatalf("ListTaskEvents() error = %v", err)
		}
		eventCounts := map[string]int{}
		for _, event := range events {
			eventCounts[event.EventType]++
		}
		if got, want := eventCounts["task.run_claimed"], 2; got != want {
			t.Fatalf("eventCounts[task.run_claimed] = %d, want %d (events=%#v)", got, want, schedulerIntegrationEventTypes(events))
		}
		if got, want := eventCounts["task.run_lease_expired"], 1; got != want {
			t.Fatalf("eventCounts[task.run_lease_expired] = %d, want %d (events=%#v)", got, want, schedulerIntegrationEventTypes(events))
		}
		if got, want := eventCounts["task.run_completed"], 1; got != want {
			t.Fatalf("eventCounts[task.run_completed] = %d, want %d (events=%#v)", got, want, schedulerIntegrationEventTypes(events))
		}
	})
}

func TestSchedulerNoEligibleSessionDoesNotClaimIntegration(t *testing.T) {
	t.Parallel()

	t.Run("Should leave a queued run untouched when no eligible session exists", func(t *testing.T) {
		ctx := testutil.Context(t)
		base := time.Date(2026, 4, 26, 16, 0, 0, 0, time.UTC)
		db := openSchedulerGlobalDB(t, filepath.Join(t.TempDir(), "agh.db"))
		workspaceID := registerSchedulerWorkspace(t, db, "no-eligible", filepath.Join(t.TempDir(), "workspace"))
		manager := newSchedulerTaskManager(t, db)
		execution := createSchedulerTaskRun(t, ctx, manager, workspaceID, "No eligible")
		waker := &fakeWaker{}
		scheduler := newTestScheduler(
			t,
			integrationTaskSource{manager: manager, store: db},
			&fakeSessionSource{sessions: []SessionSnapshot{
				sessionSnapshot("sess-other", "ws-other", "active", false, nil, base),
			}},
			waker,
			WithClock(clockwork.NewFakeClockAt(base)),
		)

		result, err := scheduler.RunOnce(ctx)
		if err != nil {
			t.Fatalf("RunOnce() error = %v", err)
		}
		if result.NoMatchRuns != 1 || result.WakeAttempts != 0 {
			t.Fatalf("scheduler result = %#v, want one no-match and no wake attempts", result)
		}
		if got := len(waker.targetsSnapshot()); got != 0 {
			t.Fatalf("wake targets = %d, want 0", got)
		}
		stored, err := db.GetTaskRun(ctx, execution.Run.ID)
		if err != nil {
			t.Fatalf("GetTaskRun() error = %v", err)
		}
		if stored.Status != taskpkg.TaskRunStatusQueued || stored.SessionID != "" || stored.ClaimTokenHash != "" {
			t.Fatalf("stored run = %#v, want queued with no owner or claim token", stored)
		}

		otherActor, err := taskpkg.DeriveAgentSessionActorContext("sess-other")
		if err != nil {
			t.Fatalf("DeriveAgentSessionActorContext(other) error = %v", err)
		}
		_, err = manager.ClaimNextRun(ctx, taskpkg.ClaimCriteria{
			Scope:            taskpkg.ScopeWorkspace,
			WorkspaceID:      "ws-other",
			ClaimerSessionID: "sess-other",
			Now:              base.Add(time.Second),
		}, otherActor)
		if !errors.Is(err, taskpkg.ErrNoClaimableRun) {
			t.Fatalf("ClaimNextRun(wrong workspace) error = %v, want ErrNoClaimableRun", err)
		}
	})
}

type integrationTaskSource struct {
	manager *taskpkg.Service
	store   taskpkg.Store
}

func (s integrationTaskSource) PendingRuns(ctx context.Context) ([]RunSnapshot, error) {
	runs, err := s.store.ListTaskRunsByStatus(ctx, []taskpkg.RunStatus{taskpkg.TaskRunStatusQueued})
	if err != nil {
		return nil, err
	}
	return s.joinRuns(ctx, runs)
}

func (s integrationTaskSource) ActiveRuns(ctx context.Context) ([]taskpkg.Run, error) {
	return s.store.ListTaskRunsByStatus(ctx, []taskpkg.RunStatus{
		taskpkg.TaskRunStatusClaimed,
		taskpkg.TaskRunStatusStarting,
		taskpkg.TaskRunStatusRunning,
	})
}

func (s integrationTaskSource) RecoverExpiredRunLeases(
	ctx context.Context,
	recovery taskpkg.ExpiredLeaseRecovery,
	actor taskpkg.ActorContext,
) ([]taskpkg.ExpiredLeaseRecoveryResult, error) {
	return s.manager.RecoverExpiredRunLeases(ctx, recovery, actor)
}

func (s integrationTaskSource) joinRuns(ctx context.Context, runs []taskpkg.Run) ([]RunSnapshot, error) {
	work := make([]RunSnapshot, 0, len(runs))
	for _, run := range runs {
		taskRecord, err := s.store.GetTask(ctx, run.TaskID)
		if err != nil {
			return nil, err
		}
		work = append(work, RunSnapshot{Task: taskRecord, Run: run})
	}
	return work, nil
}

func openSchedulerGlobalDB(t *testing.T, path string) *globaldb.GlobalDB {
	t.Helper()

	db, err := globaldb.OpenGlobalDB(testutil.Context(t), path)
	if err != nil {
		t.Fatalf("OpenGlobalDB() error = %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(context.Background()); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})
	return db
}

func newSchedulerTaskManager(t *testing.T, store taskpkg.Store) *taskpkg.Service {
	t.Helper()
	return newSchedulerTaskManagerWithOptions(t, store)
}

func newSchedulerTaskManagerWithOptions(t *testing.T, store taskpkg.Store, opts ...taskpkg.Option) *taskpkg.Service {
	t.Helper()

	managerOptions := append([]taskpkg.Option{taskpkg.WithStore(store)}, opts...)
	manager, err := taskpkg.NewManager(managerOptions...)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	return manager
}

func integrationSessionSnapshot(
	id string,
	workspaceID string,
	channel string,
	state string,
	prompting bool,
	capabilities []string,
	createdAt time.Time,
) SessionSnapshot {
	return SessionSnapshot{
		ID:           id,
		WorkspaceID:  workspaceID,
		Channel:      channel,
		State:        state,
		Prompting:    prompting,
		Capabilities: append([]string(nil), capabilities...),
		CreatedAt:    createdAt,
	}
}

func registerSchedulerWorkspace(t *testing.T, db *globaldb.GlobalDB, name string, rootDir string) string {
	t.Helper()

	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", rootDir, err)
	}
	workspace := aghworkspace.Workspace{
		ID:        "ws-" + strings.ReplaceAll(name, " ", "-"),
		RootDir:   rootDir,
		Name:      name,
		CreatedAt: time.Date(2026, 4, 26, 8, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 4, 26, 8, 0, 0, 0, time.UTC),
	}
	if err := db.InsertWorkspace(testutil.Context(t), workspace); err != nil {
		t.Fatalf("InsertWorkspace() error = %v", err)
	}
	return workspace.ID
}

func createSchedulerTaskRun(
	t *testing.T,
	ctx context.Context,
	manager *taskpkg.Service,
	workspaceID string,
	title string,
) *taskpkg.Execution {
	t.Helper()

	actor, err := taskpkg.DeriveHumanActorContext("operator", taskpkg.OriginKindCLI, "agh task start")
	if err != nil {
		t.Fatalf("DeriveHumanActorContext() error = %v", err)
	}
	taskRecord, err := manager.CreateTask(ctx, taskpkg.CreateTask{
		Scope:       taskpkg.ScopeWorkspace,
		WorkspaceID: workspaceID,
		Title:       title,
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	execution, err := manager.StartTask(ctx, taskRecord.ID, taskpkg.ExecutionRequest{}, actor)
	if err != nil {
		t.Fatalf("StartTask() error = %v", err)
	}
	return execution
}

func schedulerIntegrationHasEvent(events []taskpkg.Event, want string) bool {
	for _, event := range events {
		if event.EventType == want {
			return true
		}
	}
	return false
}

func schedulerIntegrationEventTypes(events []taskpkg.Event) []string {
	types := make([]string, 0, len(events))
	for _, event := range events {
		types = append(types, event.EventType)
	}
	slices.Sort(types)
	return types
}
