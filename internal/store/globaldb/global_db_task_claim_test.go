package globaldb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/compozy/agh/internal/store"
	taskpkg "github.com/compozy/agh/internal/task"
	"github.com/compozy/agh/internal/testutil"
)

func TestGlobalDBClaimNextRunConcurrentSingleWinner(t *testing.T) {
	globalDB := openTestGlobalDB(t)
	ctx := testutil.Context(t)
	taskRecord := taskRecordForTest("task-claim-concurrent")
	taskRecord.Status = taskpkg.TaskStatusReady
	if err := globalDB.CreateTask(ctx, taskRecord); err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	run := taskRunForTest("run-claim-concurrent", taskRecord.ID)
	if err := globalDB.CreateTaskRun(ctx, run); err != nil {
		t.Fatalf("CreateTaskRun() error = %v", err)
	}

	type claimAttempt struct {
		result taskpkg.ClaimResult
		err    error
	}
	attempts := make([]claimAttempt, 5)
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(len(attempts))
	for idx := range attempts {
		go func() {
			defer wg.Done()
			<-start
			attempts[idx].result, attempts[idx].err = globalDB.ClaimNextRun(ctx, taskpkg.ClaimCriteria{
				Scope:            taskpkg.ScopeGlobal,
				ClaimerSessionID: "sess-race-" + string(rune('a'+idx)),
				LeaseDuration:    time.Minute,
				Now:              time.Date(2026, 4, 26, 12, 0, 0, idx, time.UTC),
			})
		}()
	}
	close(start)
	wg.Wait()

	successes := 0
	for idx, attempt := range attempts {
		if attempt.err == nil {
			successes++
			if got, want := attempt.result.Run.ID, run.ID; got != want {
				t.Fatalf("attempt %d claimed run %q, want %q", idx, got, want)
			}
			if attempt.result.ClaimToken == "" {
				t.Fatalf("attempt %d returned empty claim token", idx)
			}
			if !taskpkg.VerifyClaimToken(attempt.result.ClaimToken, attempt.result.Run.ClaimTokenHash) {
				t.Fatalf("attempt %d claim token does not match stored hash", idx)
			}
			continue
		}
		if !errors.Is(attempt.err, taskpkg.ErrNoClaimableRun) {
			t.Fatalf("attempt %d error = %v, want %v", idx, attempt.err, taskpkg.ErrNoClaimableRun)
		}
	}
	if successes != 1 {
		t.Fatalf("successful claims = %d, want exactly 1 (attempts=%#v)", successes, attempts)
	}

	stored, err := globalDB.GetTaskRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetTaskRun() error = %v", err)
	}
	if got, want := stored.Status, taskpkg.TaskRunStatusClaimed; got != want {
		t.Fatalf("stored.Status = %q, want %q", got, want)
	}
	if stored.SessionID == "" {
		t.Fatal("stored.SessionID = empty, want winning session id")
	}
	t.Logf(
		"claim attempts=%d successes=%d winner_session_id=%s run_id=%s",
		len(attempts),
		successes,
		stored.SessionID,
		stored.ID,
	)
}

func TestGlobalDBClaimNextRunFiltersByCapabilitiesScopeAndChannel(t *testing.T) {
	globalDB := openTestGlobalDB(t)
	ctx := testutil.Context(t)
	workspaceID := registerWorkspaceForGlobalTests(
		t,
		globalDB,
		"claim-filters",
		filepath.Join(t.TempDir(), "claim-filters"),
	)
	otherWorkspaceID := registerWorkspaceForGlobalTests(
		t,
		globalDB,
		"claim-filters-other",
		filepath.Join(t.TempDir(), "claim-filters-other"),
	)

	matchingTask := taskRecordForTest("task-claim-match")
	matchingTask.Scope = taskpkg.ScopeWorkspace
	matchingTask.WorkspaceID = workspaceID
	matchingTask.Status = taskpkg.TaskStatusReady
	matchingTask.Priority = taskpkg.PriorityHigh
	if err := globalDB.CreateTask(ctx, matchingTask); err != nil {
		t.Fatalf("CreateTask(matching) error = %v", err)
	}
	matchingRun := taskRunForTest("run-claim-match", matchingTask.ID)
	matchingRun.CoordinationChannelID = "coord.filters"
	matchingRun.RequiredCapabilities = []string{"golang", "sqlite"}
	matchingRun.PreferredCapabilities = []string{"codex"}
	if err := globalDB.CreateTaskRun(ctx, matchingRun); err != nil {
		t.Fatalf("CreateTaskRun(matching) error = %v", err)
	}

	missingCapabilityTask := taskRecordForTest("task-claim-rust")
	missingCapabilityTask.Scope = taskpkg.ScopeWorkspace
	missingCapabilityTask.WorkspaceID = workspaceID
	missingCapabilityTask.Status = taskpkg.TaskStatusReady
	if err := globalDB.CreateTask(ctx, missingCapabilityTask); err != nil {
		t.Fatalf("CreateTask(missing capability) error = %v", err)
	}
	missingCapabilityRun := taskRunForTest("run-claim-rust", missingCapabilityTask.ID)
	missingCapabilityRun.CoordinationChannelID = "coord.filters"
	missingCapabilityRun.RequiredCapabilities = []string{"rust"}
	if err := globalDB.CreateTaskRun(ctx, missingCapabilityRun); err != nil {
		t.Fatalf("CreateTaskRun(missing capability) error = %v", err)
	}

	otherWorkspaceTask := taskRecordForTest("task-claim-other-workspace")
	otherWorkspaceTask.Scope = taskpkg.ScopeWorkspace
	otherWorkspaceTask.WorkspaceID = otherWorkspaceID
	otherWorkspaceTask.Status = taskpkg.TaskStatusReady
	if err := globalDB.CreateTask(ctx, otherWorkspaceTask); err != nil {
		t.Fatalf("CreateTask(other workspace) error = %v", err)
	}
	otherWorkspaceRun := taskRunForTest("run-claim-other-workspace", otherWorkspaceTask.ID)
	otherWorkspaceRun.CoordinationChannelID = "coord.filters"
	otherWorkspaceRun.RequiredCapabilities = []string{"golang"}
	if err := globalDB.CreateTaskRun(ctx, otherWorkspaceRun); err != nil {
		t.Fatalf("CreateTaskRun(other workspace) error = %v", err)
	}

	claim, err := globalDB.ClaimNextRun(ctx, taskpkg.ClaimCriteria{
		Scope:                 taskpkg.ScopeWorkspace,
		WorkspaceID:           workspaceID,
		ClaimerSessionID:      "sess-capable",
		RequiredCapabilities:  []string{"golang", "sqlite", "codex"},
		CoordinationChannelID: "coord.filters",
		LeaseDuration:         time.Minute,
		Now:                   time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("ClaimNextRun() error = %v", err)
	}
	if got, want := claim.Run.ID, matchingRun.ID; got != want {
		t.Fatalf("ClaimNextRun() run id = %q, want %q", got, want)
	}

	if _, err := globalDB.ClaimNextRun(ctx, taskpkg.ClaimCriteria{
		Scope:                 taskpkg.ScopeWorkspace,
		WorkspaceID:           workspaceID,
		ClaimerSessionID:      "sess-golang-only",
		RequiredCapabilities:  []string{"golang"},
		CoordinationChannelID: "coord.filters",
		LeaseDuration:         time.Minute,
		Now:                   time.Date(2026, 4, 26, 12, 1, 0, 0, time.UTC),
	}); !errors.Is(err, taskpkg.ErrNoClaimableRun) {
		t.Fatalf("ClaimNextRun(golang only) error = %v, want %v", err, taskpkg.ErrNoClaimableRun)
	}

	storedOther, err := globalDB.GetTaskRun(ctx, otherWorkspaceRun.ID)
	if err != nil {
		t.Fatalf("GetTaskRun(other workspace) error = %v", err)
	}
	if got, want := storedOther.Status, taskpkg.TaskRunStatusQueued; got != want {
		t.Fatalf("other workspace run status = %q, want %q", got, want)
	}
}

func TestGlobalDBClaimNextRunRespectsSchedulerPause(t *testing.T) {
	t.Run("Should stop new claims while preserving queued runs", func(t *testing.T) {
		globalDB := openTestGlobalDB(t)
		ctx := testutil.Context(t)
		now := time.Date(2026, 5, 21, 10, 30, 0, 0, time.UTC)
		taskRecord := taskRecordForTest("task-claim-scheduler-paused")
		taskRecord.Status = taskpkg.TaskStatusReady
		if err := globalDB.CreateTask(ctx, taskRecord); err != nil {
			t.Fatalf("CreateTask() error = %v", err)
		}
		run := taskRunForTest("run-claim-scheduler-paused", taskRecord.ID)
		if err := globalDB.CreateTaskRun(ctx, run); err != nil {
			t.Fatalf("CreateTaskRun() error = %v", err)
		}
		if _, err := globalDB.SetSchedulerPaused(ctx, "operator:ops", "maintenance"); err != nil {
			t.Fatalf("SetSchedulerPaused() error = %v", err)
		}

		_, err := globalDB.ClaimNextRun(ctx, taskpkg.ClaimCriteria{
			Scope:            taskpkg.ScopeGlobal,
			ClaimerSessionID: "sess-paused",
			LeaseDuration:    time.Minute,
			Now:              now,
		})
		if !errors.Is(err, taskpkg.ErrNoClaimableRun) {
			t.Fatalf("ClaimNextRun(paused) error = %v, want %v", err, taskpkg.ErrNoClaimableRun)
		}
		stored, err := globalDB.GetTaskRun(ctx, run.ID)
		if err != nil {
			t.Fatalf("GetTaskRun() error = %v", err)
		}
		if got, want := stored.Status, taskpkg.TaskRunStatusQueued; got != want {
			t.Fatalf("paused run status = %q, want %q", got, want)
		}
		if _, err := globalDB.SetSchedulerResumed(ctx); err != nil {
			t.Fatalf("SetSchedulerResumed() error = %v", err)
		}
		claim, err := globalDB.ClaimNextRun(ctx, taskpkg.ClaimCriteria{
			Scope:            taskpkg.ScopeGlobal,
			ClaimerSessionID: "sess-resumed",
			LeaseDuration:    time.Minute,
			Now:              now.Add(time.Second),
		})
		if err != nil {
			t.Fatalf("ClaimNextRun(resumed) error = %v", err)
		}
		if got, want := claim.Run.ID, run.ID; got != want {
			t.Fatalf("claim.Run.ID = %q, want %q", got, want)
		}
	})
}

func TestGlobalDBClaimNextRunSkipsNeedsAttention(t *testing.T) {
	t.Parallel()

	t.Run("Should not return a run escalated to needs_attention", func(t *testing.T) {
		t.Parallel()

		globalDB := openTestGlobalDB(t)
		ctx := testutil.Context(t)
		now := time.Date(2026, 5, 28, 12, 0, 0, 0, time.UTC)
		taskRecord := taskRecordForTest("task-needs-attention")
		taskRecord.Status = taskpkg.TaskStatusReady
		if err := globalDB.CreateTask(ctx, taskRecord); err != nil {
			t.Fatalf("CreateTask() error = %v", err)
		}
		run := taskRunForTest("run-needs-attention", taskRecord.ID)
		if err := globalDB.CreateTaskRun(ctx, run); err != nil {
			t.Fatalf("CreateTaskRun() error = %v", err)
		}
		if _, err := globalDB.db.ExecContext(
			ctx,
			`UPDATE task_runs SET status = 'needs_attention' WHERE id = ?`,
			run.ID,
		); err != nil {
			t.Fatalf("escalate to needs_attention error = %v", err)
		}

		_, err := globalDB.ClaimNextRun(ctx, taskpkg.ClaimCriteria{
			Scope:            taskpkg.ScopeGlobal,
			ClaimerSessionID: "sess-needs-attention",
			LeaseDuration:    time.Minute,
			Now:              now,
		})
		if !errors.Is(err, taskpkg.ErrNoClaimableRun) {
			t.Fatalf("ClaimNextRun(needs_attention) error = %v, want %v", err, taskpkg.ErrNoClaimableRun)
		}
	})
}

func TestGlobalDBClaimNextRunRespectsEffectiveTaskPause(t *testing.T) {
	t.Run("Should block descendants of a paused task without mutating child rows", func(t *testing.T) {
		globalDB := openTestGlobalDB(t)
		ctx := testutil.Context(t)
		now := time.Date(2026, 5, 21, 11, 0, 0, 0, time.UTC)
		parent := taskRecordForTest("task-claim-paused-parent")
		parent.Status = taskpkg.TaskStatusReady
		if err := globalDB.CreateTask(ctx, parent); err != nil {
			t.Fatalf("CreateTask(parent) error = %v", err)
		}
		child := taskRecordForTest("task-claim-paused-child")
		child.Status = taskpkg.TaskStatusReady
		child.ParentTaskID = parent.ID
		if err := globalDB.CreateTask(ctx, child); err != nil {
			t.Fatalf("CreateTask(child) error = %v", err)
		}
		run := taskRunForTest("run-claim-paused-child", child.ID)
		if err := globalDB.CreateTaskRun(ctx, run); err != nil {
			t.Fatalf("CreateTaskRun(child) error = %v", err)
		}
		if _, err := globalDB.PauseTask(ctx, taskpkg.PauseMutation{
			TaskID:   parent.ID,
			Actor:    "operator:ops",
			Reason:   "parent incident",
			PausedAt: now,
		}); err != nil {
			t.Fatalf("PauseTask(parent) error = %v", err)
		}

		paused, pausedBy, err := globalDB.IsTaskEffectivelyPaused(ctx, child.ID)
		if err != nil {
			t.Fatalf("IsTaskEffectivelyPaused() error = %v", err)
		}
		if !paused || pausedBy != parent.ID {
			t.Fatalf("effective pause = (%v, %q), want true by %q", paused, pausedBy, parent.ID)
		}
		if _, err := globalDB.ClaimNextRun(ctx, taskpkg.ClaimCriteria{
			Scope:            taskpkg.ScopeGlobal,
			ClaimerSessionID: "sess-child-paused",
			LeaseDuration:    time.Minute,
			Now:              now.Add(time.Second),
		}); !errors.Is(err, taskpkg.ErrNoClaimableRun) {
			t.Fatalf("ClaimNextRun(paused child) error = %v, want %v", err, taskpkg.ErrNoClaimableRun)
		}
		storedChild, err := globalDB.GetTask(ctx, child.ID)
		if err != nil {
			t.Fatalf("GetTask(child) error = %v", err)
		}
		if storedChild.Paused {
			t.Fatal("child.Paused = true, want inherited pause without mutating child row")
		}
		if _, err := globalDB.ResumeTask(ctx, taskpkg.ResumeMutation{
			TaskID:    parent.ID,
			ResumedAt: now.Add(2 * time.Second),
		}); err != nil {
			t.Fatalf("ResumeTask(parent) error = %v", err)
		}
		claim, err := globalDB.ClaimNextRun(ctx, taskpkg.ClaimCriteria{
			Scope:            taskpkg.ScopeGlobal,
			ClaimerSessionID: "sess-child-resumed",
			LeaseDuration:    time.Minute,
			Now:              now.Add(3 * time.Second),
		})
		if err != nil {
			t.Fatalf("ClaimNextRun(resumed child) error = %v", err)
		}
		if got, want := claim.Run.ID, run.ID; got != want {
			t.Fatalf("claim.Run.ID = %q, want %q", got, want)
		}
	})
}

func TestGlobalDBClaimNextRunAppliesExecutionProfileEligibility(t *testing.T) {
	t.Run("Should reject ineligible agents and missing profile capabilities", func(t *testing.T) {
		globalDB := openTestGlobalDB(t)
		ctx := testutil.Context(t)
		now := time.Date(2026, 4, 26, 12, 15, 0, 0, time.UTC)

		taskRecord := taskRecordForTest("task-profile-claim")
		taskRecord.Status = taskpkg.TaskStatusReady
		if err := globalDB.CreateTask(ctx, taskRecord); err != nil {
			t.Fatalf("CreateTask() error = %v", err)
		}
		run := taskRunForTest("run-profile-claim", taskRecord.ID)
		if err := globalDB.CreateTaskRun(ctx, run); err != nil {
			t.Fatalf("CreateTaskRun() error = %v", err)
		}
		if _, err := globalDB.UpsertExecutionProfile(ctx, &taskpkg.ExecutionProfile{
			TaskID: taskRecord.ID,
			Worker: taskpkg.WorkerProfile{
				Mode:                 taskpkg.WorkerModeSelect,
				AgentName:            "codex-worker",
				AllowedAgentNames:    []string{"codex-worker"},
				RequiredCapabilities: []string{"golang"},
			},
			Participants: taskpkg.ParticipantPolicy{
				AllowedAgentNames:    []string{"codex-worker"},
				RequiredCapabilities: []string{"sqlite"},
			},
		}); err != nil {
			t.Fatalf("UpsertExecutionProfile() error = %v", err)
		}

		if _, err := globalDB.ClaimNextRun(ctx, taskpkg.ClaimCriteria{
			Scope:                taskpkg.ScopeGlobal,
			ClaimerSessionID:     "sess-wrong-agent",
			AgentName:            "other-worker",
			RequiredCapabilities: []string{"golang", "sqlite"},
			LeaseDuration:        time.Minute,
			Now:                  now,
		}); !errors.Is(err, taskpkg.ErrNoClaimableRun) {
			t.Fatalf("ClaimNextRun(wrong agent) error = %v, want %v", err, taskpkg.ErrNoClaimableRun)
		}
		if _, err := globalDB.ClaimNextRun(ctx, taskpkg.ClaimCriteria{
			Scope:                taskpkg.ScopeGlobal,
			ClaimerSessionID:     "sess-missing-agent-name",
			RequiredCapabilities: []string{"golang", "sqlite"},
			LeaseDuration:        time.Minute,
			Now:                  now.Add(500 * time.Millisecond),
		}); !errors.Is(err, taskpkg.ErrNoClaimableRun) {
			t.Fatalf("ClaimNextRun(blank agent) error = %v, want %v", err, taskpkg.ErrNoClaimableRun)
		}
		if _, err := globalDB.ClaimNextRun(ctx, taskpkg.ClaimCriteria{
			Scope:                taskpkg.ScopeGlobal,
			ClaimerSessionID:     "sess-missing-capability",
			AgentName:            "codex-worker",
			RequiredCapabilities: []string{"golang"},
			LeaseDuration:        time.Minute,
			Now:                  now.Add(time.Second),
		}); !errors.Is(err, taskpkg.ErrNoClaimableRun) {
			t.Fatalf("ClaimNextRun(missing capability) error = %v, want %v", err, taskpkg.ErrNoClaimableRun)
		}
		storedQueued, err := globalDB.GetTaskRun(ctx, run.ID)
		if err != nil {
			t.Fatalf("GetTaskRun(before eligible claim) error = %v", err)
		}
		if got, want := storedQueued.Status, taskpkg.TaskRunStatusQueued; got != want {
			t.Fatalf("run status before eligible claim = %q, want %q", got, want)
		}

		claim, err := globalDB.ClaimNextRun(ctx, taskpkg.ClaimCriteria{
			Scope:                taskpkg.ScopeGlobal,
			ClaimerSessionID:     "sess-codex-worker",
			AgentName:            "codex-worker",
			RequiredCapabilities: []string{"golang", "sqlite"},
			LeaseDuration:        time.Minute,
			Now:                  now.Add(2 * time.Second),
		})
		if err != nil {
			t.Fatalf("ClaimNextRun(eligible) error = %v", err)
		}
		if got, want := claim.Run.ID, run.ID; got != want {
			t.Fatalf("ClaimNextRun(eligible) run id = %q, want %q", got, want)
		}
	})
}

func TestGlobalDBClaimNextRunFiltersByTaskOwner(t *testing.T) {
	t.Parallel()

	t.Run("Should require a matching pool owner agent name", func(t *testing.T) {
		t.Parallel()

		globalDB := openTestGlobalDB(t)
		ctx := testutil.Context(t)
		workspaceID := registerWorkspaceForGlobalTests(
			t,
			globalDB,
			"claim-owner-filter",
			filepath.Join(t.TempDir(), "claim-owner-filter"),
		)
		now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)

		taskRecord := taskRecordForTest("task-owner-filter")
		taskRecord.Scope = taskpkg.ScopeWorkspace
		taskRecord.WorkspaceID = workspaceID
		taskRecord.Status = taskpkg.TaskStatusReady
		taskRecord.Owner = &taskpkg.Ownership{Kind: taskpkg.OwnerKindPool, Ref: "frontend-engineer-agent"}
		if err := globalDB.CreateTask(ctx, taskRecord); err != nil {
			t.Fatalf("CreateTask() error = %v", err)
		}
		run := taskRunForTest("run-owner-filter", taskRecord.ID)
		run.CoordinationChannelID = "design-review"
		if err := globalDB.CreateTaskRun(ctx, run); err != nil {
			t.Fatalf("CreateTaskRun() error = %v", err)
		}

		_, err := globalDB.ClaimNextRun(ctx, taskpkg.ClaimCriteria{
			Scope:                 taskpkg.ScopeWorkspace,
			WorkspaceID:           workspaceID,
			ClaimerSessionID:      "sess-wrong-agent",
			AgentName:             "analytics-engineer-agent",
			CoordinationChannelID: "design-review",
			LeaseDuration:         time.Minute,
			Now:                   now,
		})
		if !errors.Is(err, taskpkg.ErrNoClaimableRun) {
			t.Fatalf("ClaimNextRun(wrong owner) error = %v, want %v", err, taskpkg.ErrNoClaimableRun)
		}

		stored, err := globalDB.GetTaskRun(ctx, run.ID)
		if err != nil {
			t.Fatalf("GetTaskRun(after wrong owner) error = %v", err)
		}
		if got, want := stored.Status, taskpkg.TaskRunStatusQueued; got != want {
			t.Fatalf("stored.Status = %q, want %q", got, want)
		}

		claim, err := globalDB.ClaimNextRun(ctx, taskpkg.ClaimCriteria{
			Scope:                 taskpkg.ScopeWorkspace,
			WorkspaceID:           workspaceID,
			ClaimerSessionID:      "sess-frontend",
			AgentName:             "frontend-engineer-agent",
			CoordinationChannelID: "design-review",
			LeaseDuration:         time.Minute,
			Now:                   now.Add(time.Second),
		})
		if err != nil {
			t.Fatalf("ClaimNextRun(matching owner) error = %v", err)
		}
		if got, want := claim.Run.ID, run.ID; got != want {
			t.Fatalf("ClaimNextRun(matching owner) run = %q, want %q", got, want)
		}
	})
}

func TestGlobalDBClaimNextRunManualAndAgentCreatedRunsSharePrimitive(t *testing.T) {
	globalDB := openTestGlobalDB(t)
	ctx := testutil.Context(t)
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)

	humanTask := taskRecordForTest("task-human-created-claim")
	humanTask.Status = taskpkg.TaskStatusReady
	humanTask.CreatedBy = taskpkg.ActorIdentity{Kind: taskpkg.ActorKindHuman, Ref: "user:alice"}
	if err := globalDB.CreateTask(ctx, humanTask); err != nil {
		t.Fatalf("CreateTask(human) error = %v", err)
	}
	humanRun := taskRunForTest("run-human-created-claim", humanTask.ID)
	humanRun.QueuedAt = now
	if err := globalDB.CreateTaskRun(ctx, humanRun); err != nil {
		t.Fatalf("CreateTaskRun(human) error = %v", err)
	}

	agentTask := taskRecordForTest("task-agent-created-claim")
	agentTask.Status = taskpkg.TaskStatusReady
	agentTask.CreatedBy = taskpkg.ActorIdentity{Kind: taskpkg.ActorKindAgentSession, Ref: "sess-parent"}
	if err := globalDB.CreateTask(ctx, agentTask); err != nil {
		t.Fatalf("CreateTask(agent) error = %v", err)
	}
	agentRun := taskRunForTest("run-agent-created-claim", agentTask.ID)
	agentRun.QueuedAt = now.Add(time.Second)
	if err := globalDB.CreateTaskRun(ctx, agentRun); err != nil {
		t.Fatalf("CreateTaskRun(agent) error = %v", err)
	}

	first, err := globalDB.ClaimNextRun(ctx, taskpkg.ClaimCriteria{
		Scope:            taskpkg.ScopeGlobal,
		ClaimerSessionID: "sess-worker-1",
		LeaseDuration:    time.Minute,
		Now:              now.Add(10 * time.Second),
	})
	if err != nil {
		t.Fatalf("ClaimNextRun(first) error = %v", err)
	}
	if got, want := first.Task.CreatedBy.Kind, taskpkg.ActorKindHuman; got != want {
		t.Fatalf("first.Task.CreatedBy.Kind = %q, want %q", got, want)
	}

	second, err := globalDB.ClaimNextRun(ctx, taskpkg.ClaimCriteria{
		Scope:            taskpkg.ScopeGlobal,
		ClaimerSessionID: "sess-worker-2",
		LeaseDuration:    time.Minute,
		Now:              now.Add(20 * time.Second),
	})
	if err != nil {
		t.Fatalf("ClaimNextRun(second) error = %v", err)
	}
	if got, want := second.Task.CreatedBy.Kind, taskpkg.ActorKindAgentSession; got != want {
		t.Fatalf("second.Task.CreatedBy.Kind = %q, want %q", got, want)
	}
}

func TestGlobalDBClaimNextRunPersistsSoulProvenanceMetadata(t *testing.T) {
	t.Parallel()

	t.Run("Should merge pre-resolved soul provenance without reading SOUL.md", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		dbPath := filepath.Join(t.TempDir(), GlobalDatabaseName)
		globalDB, err := OpenGlobalDB(ctx, dbPath)
		if err != nil {
			t.Fatalf("OpenGlobalDB() error = %v", err)
		}
		t.Cleanup(func() {
			if globalDB == nil {
				return
			}
			if err := globalDB.Close(ctx); err != nil {
				t.Fatalf("Close() error = %v", err)
			}
		})

		workspaceRoot := filepath.Join(t.TempDir(), "workspace")
		writeInvalidSoulFixture(t, workspaceRoot)
		workspaceID := registerWorkspaceForGlobalTests(t, globalDB, "claim-soul", workspaceRoot)
		now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)

		taskRecord := taskRecordForTest("task-soul-claim")
		taskRecord.Scope = taskpkg.ScopeWorkspace
		taskRecord.WorkspaceID = workspaceID
		taskRecord.Status = taskpkg.TaskStatusReady
		if err := globalDB.CreateTask(ctx, taskRecord); err != nil {
			t.Fatalf("CreateTask() error = %v", err)
		}
		run := taskRunForTest("run-soul-claim", taskRecord.ID)
		run.Metadata = json.RawMessage(`{"workflow_id":"wf-soul"}`)
		if err := globalDB.CreateTaskRun(ctx, run); err != nil {
			t.Fatalf("CreateTaskRun() error = %v", err)
		}

		capturedAt := now.Add(-time.Minute)
		claim, err := globalDB.ClaimNextRun(ctx, taskpkg.ClaimCriteria{
			Scope:            taskpkg.ScopeWorkspace,
			WorkspaceID:      workspaceID,
			ClaimerSessionID: "sess-soul",
			AgentName:        "coder",
			Soul: &taskpkg.SoulClaimProvenance{
				SnapshotID: "soul-snapshot-1",
				Digest:     "sha256:resolved",
				AgentName:  "coder",
				CapturedAt: capturedAt,
			},
			LeaseDuration: time.Minute,
			Now:           now,
		})
		if err != nil {
			t.Fatalf("ClaimNextRun() error = %v", err)
		}

		assertRunSoulMetadata(
			t,
			claim.Run.Metadata,
			"wf-soul",
			"soul-snapshot-1",
			"sha256:resolved",
			"coder",
			capturedAt,
		)
		stored, err := globalDB.GetTaskRun(ctx, run.ID)
		if err != nil {
			t.Fatalf("GetTaskRun() error = %v", err)
		}
		assertRunSoulMetadata(t, stored.Metadata, "wf-soul", "soul-snapshot-1", "sha256:resolved", "coder", capturedAt)

		if err := globalDB.Close(ctx); err != nil {
			t.Fatalf("Close(before reopen) error = %v", err)
		}
		globalDB = nil
		reopened, err := OpenGlobalDB(ctx, dbPath)
		if err != nil {
			t.Fatalf("OpenGlobalDB(reopen) error = %v", err)
		}
		globalDB = reopened
		reopenedRun, err := globalDB.GetTaskRun(ctx, run.ID)
		if err != nil {
			t.Fatalf("GetTaskRun(reopen) error = %v", err)
		}
		assertRunSoulMetadata(
			t,
			reopenedRun.Metadata,
			"wf-soul",
			"soul-snapshot-1",
			"sha256:resolved",
			"coder",
			capturedAt,
		)
	})
}

func TestGlobalDBClaimLeaseLifecycleFencing(t *testing.T) {
	globalDB := openTestGlobalDB(t)
	ctx := testutil.Context(t)
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	taskRecord := taskRecordForTest("task-lease-lifecycle")
	taskRecord.Status = taskpkg.TaskStatusReady
	if err := globalDB.CreateTask(ctx, taskRecord); err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	firstRun := taskRunForTest("run-lease-lifecycle-first", taskRecord.ID)
	if err := globalDB.CreateTaskRun(ctx, firstRun); err != nil {
		t.Fatalf("CreateTaskRun(first) error = %v", err)
	}

	claim, err := globalDB.ClaimNextRun(ctx, taskpkg.ClaimCriteria{
		Scope:            taskpkg.ScopeGlobal,
		ClaimerSessionID: "sess-lease",
		LeaseDuration:    time.Minute,
		Now:              now,
	})
	if err != nil {
		t.Fatalf("ClaimNextRun() error = %v", err)
	}
	if claim.Run.ClaimToken != "" {
		t.Fatalf("claim.Run.ClaimToken = %q, want empty read model", claim.Run.ClaimToken)
	}
	var storedRaw sql.NullString
	if err := globalDB.db.QueryRowContext(ctx, `SELECT claim_token FROM task_runs WHERE id = ?`, claim.Run.ID).
		Scan(&storedRaw); err != nil {
		t.Fatalf("query claim_token error = %v", err)
	}
	if !storedRaw.Valid || storedRaw.String != claim.ClaimToken {
		t.Fatalf("stored raw claim_token = %#v, want internal active lease token", storedRaw)
	}

	secondRun := taskRunForTest("run-lease-lifecycle-second", taskRecord.ID)
	secondRun.QueuedAt = firstRun.QueuedAt.Add(time.Second)
	if err := globalDB.CreateTaskRun(ctx, secondRun); err != nil {
		t.Fatalf("CreateTaskRun(second) error = %v", err)
	}
	if _, err := globalDB.ClaimNextRun(ctx, taskpkg.ClaimCriteria{
		Scope:            taskpkg.ScopeGlobal,
		ClaimerSessionID: "sess-lease",
		LeaseDuration:    time.Minute,
		Now:              now.Add(5 * time.Second),
	}); !errors.Is(err, taskpkg.ErrActiveRunLease) {
		t.Fatalf("ClaimNextRun(second active same session) error = %v, want %v", err, taskpkg.ErrActiveRunLease)
	}

	if _, err := globalDB.HeartbeatRunLease(ctx, taskpkg.LeaseHeartbeat{
		RunID:         claim.Run.ID,
		ClaimToken:    "stale-token",
		LeaseDuration: time.Minute,
		Now:           now.Add(10 * time.Second),
	}); !errors.Is(err, taskpkg.ErrInvalidClaimToken) {
		t.Fatalf("HeartbeatRunLease(stale token) error = %v, want %v", err, taskpkg.ErrInvalidClaimToken)
	}
	if _, err := globalDB.HeartbeatRunLease(ctx, taskpkg.LeaseHeartbeat{
		RunID:         claim.Run.ID,
		ClaimToken:    claim.ClaimToken,
		LeaseDuration: time.Minute,
		Now:           claim.LeaseUntil,
	}); !errors.Is(err, taskpkg.ErrLeaseExpired) {
		t.Fatalf("HeartbeatRunLease(expired token) error = %v, want %v", err, taskpkg.ErrLeaseExpired)
	}
	heartbeat, err := globalDB.HeartbeatRunLease(ctx, taskpkg.LeaseHeartbeat{
		RunID:         claim.Run.ID,
		ClaimToken:    claim.ClaimToken,
		LeaseDuration: 2 * time.Minute,
		Now:           now.Add(30 * time.Second),
	})
	if err != nil {
		t.Fatalf("HeartbeatRunLease(current token) error = %v", err)
	}
	if got, want := heartbeat.LeaseUntil, now.Add(150*time.Second); !got.Equal(want) {
		t.Fatalf("heartbeat.LeaseUntil = %v, want %v", got, want)
	}
	if err := globalDB.db.QueryRowContext(ctx, `SELECT claim_token FROM task_runs WHERE id = ?`, claim.Run.ID).
		Scan(&storedRaw); err != nil {
		t.Fatalf("query heartbeat claim_token error = %v", err)
	}
	if !storedRaw.Valid || storedRaw.String != claim.ClaimToken {
		t.Fatalf("heartbeat stored raw claim_token = %#v, want retained internal active lease token", storedRaw)
	}

	if _, err := globalDB.CompleteRunLease(ctx, taskpkg.LeaseCompletion{
		RunID:      claim.Run.ID,
		ClaimToken: "stale-token",
		Result:     taskpkg.RunResult{Value: json.RawMessage(`{"ok":false}`)},
		Now:        now.Add(35 * time.Second),
	}); !errors.Is(err, taskpkg.ErrInvalidClaimToken) {
		t.Fatalf("CompleteRunLease(stale token) error = %v, want %v", err, taskpkg.ErrInvalidClaimToken)
	}
	completed, err := globalDB.CompleteRunLease(ctx, taskpkg.LeaseCompletion{
		RunID:      claim.Run.ID,
		ClaimToken: claim.ClaimToken,
		Result:     taskpkg.RunResult{Value: json.RawMessage(`{"ok":true}`)},
		Now:        now.Add(40 * time.Second),
	})
	if err != nil {
		t.Fatalf("CompleteRunLease(current token) error = %v", err)
	}
	if got, want := completed.Status, taskpkg.TaskRunStatusCompleted; got != want {
		t.Fatalf("completed.Status = %q, want %q", got, want)
	}
	if completed.LeaseUntil.IsZero() == false || completed.HeartbeatAt.IsZero() == false {
		t.Fatalf("completed lease fields = lease_until %v heartbeat_at %v, want zero",
			completed.LeaseUntil,
			completed.HeartbeatAt,
		)
	}
	if completed.ClaimTokenHash == "" {
		t.Fatal("completed.ClaimTokenHash = empty, want retained hash")
	}
	if err := globalDB.db.QueryRowContext(ctx, `SELECT claim_token FROM task_runs WHERE id = ?`, claim.Run.ID).
		Scan(&storedRaw); err != nil {
		t.Fatalf("query completed claim_token error = %v", err)
	}
	if storedRaw.Valid {
		t.Fatalf("completed stored raw claim_token = %q, want NULL", storedRaw.String)
	}

	releaseClaim, err := globalDB.ClaimNextRun(ctx, taskpkg.ClaimCriteria{
		Scope:            taskpkg.ScopeGlobal,
		ClaimerSessionID: "sess-lease",
		LeaseDuration:    time.Minute,
		Now:              now.Add(45 * time.Second),
	})
	if err != nil {
		t.Fatalf("ClaimNextRun(after completion) error = %v", err)
	}
	released, err := globalDB.ReleaseRunLease(ctx, taskpkg.LeaseRelease{
		RunID:      releaseClaim.Run.ID,
		ClaimToken: releaseClaim.ClaimToken,
		Reason:     "handoff",
		Now:        now.Add(50 * time.Second),
	})
	if err != nil {
		t.Fatalf("ReleaseRunLease() error = %v", err)
	}
	if got, want := released.Status, taskpkg.TaskRunStatusQueued; got != want {
		t.Fatalf("released.Status = %q, want %q", got, want)
	}
	if released.ClaimTokenHash != "" || released.SessionID != "" || released.ClaimedBy != nil {
		t.Fatalf("released ownership fields = hash %q session %q claimed_by %#v, want cleared",
			released.ClaimTokenHash,
			released.SessionID,
			released.ClaimedBy,
		)
	}

	failClaim, err := globalDB.ClaimNextRun(ctx, taskpkg.ClaimCriteria{
		Scope:            taskpkg.ScopeGlobal,
		ClaimerSessionID: "sess-lease-fail",
		LeaseDuration:    time.Minute,
		Now:              now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("ClaimNextRun(for failure) error = %v", err)
	}
	failed, err := globalDB.FailRunLease(ctx, taskpkg.LeaseFailure{
		RunID:      failClaim.Run.ID,
		ClaimToken: failClaim.ClaimToken,
		Failure:    taskpkg.RunFailure{Error: "worker failed"},
		Now:        now.Add(70 * time.Second),
	})
	if err != nil {
		t.Fatalf("FailRunLease() error = %v", err)
	}
	if got, want := failed.Status, taskpkg.TaskRunStatusFailed; got != want {
		t.Fatalf("failed.Status = %q, want %q", got, want)
	}
	if got, want := failed.Error, "worker failed"; got != want {
		t.Fatalf("failed.Error = %q, want %q", got, want)
	}
}

func TestGlobalDBBlockRunLeaseParksNeedsAttention(t *testing.T) {
	globalDB := openTestGlobalDB(t)
	ctx := testutil.Context(t)
	now := time.Date(2026, 4, 26, 13, 0, 0, 0, time.UTC)
	taskRecord := taskRecordForTest("task-block-lease")
	taskRecord.Status = taskpkg.TaskStatusReady
	if err := globalDB.CreateTask(ctx, taskRecord); err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	run := taskRunForTest("run-block-lease", taskRecord.ID)
	if err := globalDB.CreateTaskRun(ctx, run); err != nil {
		t.Fatalf("CreateTaskRun() error = %v", err)
	}
	claim, err := globalDB.ClaimNextRun(ctx, taskpkg.ClaimCriteria{
		Scope:            taskpkg.ScopeGlobal,
		ClaimerSessionID: "sess-block-lease",
		LeaseDuration:    time.Minute,
		Now:              now,
	})
	if err != nil {
		t.Fatalf("ClaimNextRun() error = %v", err)
	}
	blocked, err := globalDB.BlockRunLease(ctx, taskpkg.LeaseBlock{
		RunID:      claim.Run.ID,
		ClaimToken: claim.ClaimToken,
		Reason:     "blocked_on_human",
		Now:        now.Add(10 * time.Second),
	})
	if err != nil {
		t.Fatalf("BlockRunLease() error = %v", err)
	}
	if got, want := blocked.Status, taskpkg.TaskRunStatusNeedsAttention; got != want {
		t.Fatalf("blocked.Status = %q, want %q", got, want)
	}
	if blocked.ClaimTokenHash != "" ||
		blocked.SessionID != "" ||
		blocked.ClaimedBy != nil ||
		!blocked.LeaseUntil.IsZero() {
		t.Fatalf(
			"blocked ownership fields = hash %q session %q claimed_by %#v lease %v, want cleared",
			blocked.ClaimTokenHash,
			blocked.SessionID,
			blocked.ClaimedBy,
			blocked.LeaseUntil,
		)
	}
	if got, want := blocked.Error, "blocked_on_human"; got != want {
		t.Fatalf("blocked.Error = %q, want %q", got, want)
	}
	var rawClaimToken sql.NullString
	if err := globalDB.db.QueryRowContext(ctx, `SELECT claim_token FROM task_runs WHERE id = ?`, blocked.ID).
		Scan(&rawClaimToken); err != nil {
		t.Fatalf("query blocked claim_token error = %v", err)
	}
	if rawClaimToken.Valid {
		t.Fatalf("blocked stored raw claim_token = %q, want NULL", rawClaimToken.String)
	}
	if _, err := globalDB.ClaimNextRun(ctx, taskpkg.ClaimCriteria{
		Scope:            taskpkg.ScopeGlobal,
		ClaimerSessionID: "sess-other",
		LeaseDuration:    time.Minute,
		Now:              now.Add(20 * time.Second),
	}); !errors.Is(err, taskpkg.ErrNoClaimableRun) {
		t.Fatalf("ClaimNextRun(after block) error = %v, want %v", err, taskpkg.ErrNoClaimableRun)
	}
}

func TestGlobalDBRecoverExpiredRunLeasesThenClaim(t *testing.T) {
	globalDB := openTestGlobalDB(t)
	ctx := testutil.Context(t)
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	taskRecord := taskRecordForTest("task-expired-lease-recovery")
	taskRecord.Status = taskpkg.TaskStatusReady
	if err := globalDB.CreateTask(ctx, taskRecord); err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	expiredRun := leasedRunForGlobalTest(
		t,
		"run-expired-lease-recovery",
		taskRecord.ID,
		"sess-expired",
		"expired-token",
		now.Add(-time.Minute),
	)
	if err := globalDB.CreateTaskRun(ctx, expiredRun); err != nil {
		t.Fatalf("CreateTaskRun(expired) error = %v", err)
	}
	unexpiredRun := leasedRunForGlobalTest(
		t,
		"run-unexpired-lease-recovery",
		taskRecord.ID,
		"sess-active",
		"active-token",
		now.Add(time.Minute),
	)
	if err := globalDB.CreateTaskRun(ctx, unexpiredRun); err != nil {
		t.Fatalf("CreateTaskRun(unexpired) error = %v", err)
	}

	if _, err := globalDB.ClaimNextRun(ctx, taskpkg.ClaimCriteria{
		Scope:            taskpkg.ScopeGlobal,
		ClaimerSessionID: "sess-before-recovery",
		LeaseDuration:    time.Minute,
		Now:              now,
	}); !errors.Is(err, taskpkg.ErrNoClaimableRun) {
		t.Fatalf("ClaimNextRun(before recovery) error = %v, want %v", err, taskpkg.ErrNoClaimableRun)
	}

	recovered, err := globalDB.RecoverExpiredRunLeases(ctx, taskpkg.ExpiredLeaseRecovery{
		Now:    now,
		Reason: "orphaned_on_boot",
	})
	if err != nil {
		t.Fatalf("RecoverExpiredRunLeases() error = %v", err)
	}
	if got, want := len(recovered), 1; got != want {
		t.Fatalf("len(RecoverExpiredRunLeases()) = %d, want %d", got, want)
	}
	if got, want := recovered[0].Run.ID, expiredRun.ID; got != want {
		t.Fatalf("recovered run id = %q, want %q", got, want)
	}
	if got, want := recovered[0].PreviousSessionID, "sess-expired"; got != want {
		t.Fatalf("PreviousSessionID = %q, want %q", got, want)
	}
	if recovered[0].PreviousClaimTokenHash == "" {
		t.Fatal("PreviousClaimTokenHash = empty, want expired hash")
	}
	if got, want := recovered[0].Run.Status, taskpkg.TaskRunStatusQueued; got != want {
		t.Fatalf("recovered status = %q, want %q", got, want)
	}
	if recovered[0].Run.ClaimTokenHash != "" || recovered[0].Run.SessionID != "" {
		t.Fatalf("recovered ownership = hash %q session %q, want cleared",
			recovered[0].Run.ClaimTokenHash,
			recovered[0].Run.SessionID,
		)
	}

	if _, err := globalDB.HeartbeatRunLease(ctx, taskpkg.LeaseHeartbeat{
		RunID:         expiredRun.ID,
		ClaimToken:    "expired-token",
		LeaseDuration: time.Minute,
		Now:           now.Add(time.Second),
	}); !errors.Is(err, taskpkg.ErrInvalidClaimToken) {
		t.Fatalf("HeartbeatRunLease(stale recovered lease) error = %v, want %v", err, taskpkg.ErrInvalidClaimToken)
	}

	claim, err := globalDB.ClaimNextRun(ctx, taskpkg.ClaimCriteria{
		Scope:            taskpkg.ScopeGlobal,
		ClaimerSessionID: "sess-after-recovery",
		LeaseDuration:    time.Minute,
		Now:              now.Add(2 * time.Second),
	})
	if err != nil {
		t.Fatalf("ClaimNextRun(after recovery) error = %v", err)
	}
	if got, want := claim.Run.ID, expiredRun.ID; got != want {
		t.Fatalf("ClaimNextRun(after recovery) run id = %q, want %q", got, want)
	}

	active, err := globalDB.GetTaskRun(ctx, unexpiredRun.ID)
	if err != nil {
		t.Fatalf("GetTaskRun(unexpired) error = %v", err)
	}
	if got, want := active.Status, taskpkg.TaskRunStatusClaimed; got != want {
		t.Fatalf("unexpired status = %q, want %q", got, want)
	}
	if got, want := active.SessionID, "sess-active"; got != want {
		t.Fatalf("unexpired session id = %q, want %q", got, want)
	}
}

func TestGlobalDBTaskCurrentRunProjection(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "Should set projection when claiming the next run",
			run: func(t *testing.T) {
				globalDB, ctx, taskRecord, run, now := setupCurrentRunProjectionTest(t, "claim")

				claim := claimProjectionRunForTest(ctx, t, globalDB, "sess-projection-claim", now)
				if got, want := claim.Run.ID, run.ID; got != want {
					t.Fatalf("ClaimNextRun() run id = %q, want %q", got, want)
				}
				if got, want := claim.Task.CurrentRunID, run.ID; got != want {
					t.Fatalf("claim.Task.CurrentRunID = %q, want %q", got, want)
				}

				assertTaskCurrentRunProjection(ctx, t, globalDB, taskRecord.ID, run.ID)
				summaries, err := globalDB.ListTasks(ctx, taskpkg.Query{Scope: taskpkg.ScopeGlobal})
				if err != nil {
					t.Fatalf("ListTasks() error = %v", err)
				}
				if got, want := summaries[0].CurrentRunID, run.ID; got != want {
					t.Fatalf("summary.CurrentRunID = %q, want %q", got, want)
				}
			},
		},
		{
			name: "Should clear projection when completing a lease",
			run: func(t *testing.T) {
				globalDB, ctx, taskRecord, _, now := setupCurrentRunProjectionTest(t, "complete")
				claim := claimProjectionRunForTest(ctx, t, globalDB, "sess-projection-complete", now)

				if _, err := globalDB.CompleteRunLease(ctx, taskpkg.LeaseCompletion{
					RunID:      claim.Run.ID,
					ClaimToken: claim.ClaimToken,
					Result:     taskpkg.RunResult{Value: json.RawMessage(`{"ok":true}`)},
					Now:        now.Add(10 * time.Second),
				}); err != nil {
					t.Fatalf("CompleteRunLease() error = %v", err)
				}

				assertTaskCurrentRunProjection(ctx, t, globalDB, taskRecord.ID, "")
			},
		},
		{
			name: "Should clear projection when failing a lease",
			run: func(t *testing.T) {
				globalDB, ctx, taskRecord, _, now := setupCurrentRunProjectionTest(t, "fail")
				claim := claimProjectionRunForTest(ctx, t, globalDB, "sess-projection-fail", now)

				if _, err := globalDB.FailRunLease(ctx, taskpkg.LeaseFailure{
					RunID:      claim.Run.ID,
					ClaimToken: claim.ClaimToken,
					Failure:    taskpkg.RunFailure{Error: "worker failed"},
					Now:        now.Add(10 * time.Second),
				}); err != nil {
					t.Fatalf("FailRunLease() error = %v", err)
				}

				assertTaskCurrentRunProjection(ctx, t, globalDB, taskRecord.ID, "")
			},
		},
		{
			name: "Should clear projection when releasing a lease",
			run: func(t *testing.T) {
				globalDB, ctx, taskRecord, _, now := setupCurrentRunProjectionTest(t, "release")
				claim := claimProjectionRunForTest(ctx, t, globalDB, "sess-projection-release", now)

				if _, err := globalDB.ReleaseRunLease(ctx, taskpkg.LeaseRelease{
					RunID:      claim.Run.ID,
					ClaimToken: claim.ClaimToken,
					Reason:     "handoff",
					Now:        now.Add(10 * time.Second),
				}); err != nil {
					t.Fatalf("ReleaseRunLease() error = %v", err)
				}

				assertTaskCurrentRunProjection(ctx, t, globalDB, taskRecord.ID, "")
			},
		},
		{
			name: "Should clear projection when recovering an expired lease",
			run: func(t *testing.T) {
				globalDB := openTestGlobalDB(t)
				ctx := testutil.Context(t)
				now := time.Date(2026, 4, 26, 15, 0, 0, 0, time.UTC)
				taskRecord := taskRecordForTest("task-current-projection-recovery")
				taskRecord.Status = taskpkg.TaskStatusReady
				if err := globalDB.CreateTask(ctx, taskRecord); err != nil {
					t.Fatalf("CreateTask() error = %v", err)
				}
				run := leasedRunForGlobalTest(
					t,
					"run-current-projection-recovery",
					taskRecord.ID,
					"sess-projection-recovery",
					"expired-projection-token",
					now.Add(-time.Minute),
				)
				if err := globalDB.CreateTaskRun(ctx, run); err != nil {
					t.Fatalf("CreateTaskRun() error = %v", err)
				}
				if _, err := globalDB.db.ExecContext(
					ctx,
					`UPDATE tasks SET current_run_id = ? WHERE id = ?`,
					run.ID,
					taskRecord.ID,
				); err != nil {
					t.Fatalf("seed current_run_id error = %v", err)
				}

				recovered, err := globalDB.RecoverExpiredRunLeases(ctx, taskpkg.ExpiredLeaseRecovery{
					Now:    now,
					Reason: "orphaned_on_boot",
				})
				if err != nil {
					t.Fatalf("RecoverExpiredRunLeases() error = %v", err)
				}
				if got, want := len(recovered), 1; got != want {
					t.Fatalf("len(RecoverExpiredRunLeases()) = %d, want %d", got, want)
				}

				assertTaskCurrentRunProjection(ctx, t, globalDB, taskRecord.ID, "")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t)
		})
	}
}

func TestSetTaskCurrentRunProjectionDetectsConcurrentOverwrite(t *testing.T) {
	t.Parallel()

	globalDB, ctx, taskRecord, run, _ := setupCurrentRunProjectionTest(t, "set-race")
	otherRun := taskRunForTest("run-current-projection-set-race-other", taskRecord.ID)
	if err := globalDB.CreateTaskRun(ctx, otherRun); err != nil {
		t.Fatalf("CreateTaskRun(other) error = %v", err)
	}

	injected := false
	exec := projectionRaceExecutor{
		taskSQLExecutor: globalDB.db,
		beforeExec: func(ctx context.Context) error {
			if injected {
				return nil
			}
			injected = true
			_, err := globalDB.db.ExecContext(
				ctx,
				`UPDATE tasks SET current_run_id = ? WHERE id = ?`,
				otherRun.ID,
				taskRecord.ID,
			)
			return err
		},
	}

	err := setTaskCurrentRunProjection(ctx, exec, taskRecord.ID, run.ID)
	if !errors.Is(err, taskpkg.ErrInvalidStatusTransition) {
		t.Fatalf("setTaskCurrentRunProjection() error = %v, want %v", err, taskpkg.ErrInvalidStatusTransition)
	}
	assertTaskCurrentRunProjection(ctx, t, globalDB, taskRecord.ID, otherRun.ID)
}

func TestClearTaskCurrentRunProjectionDetectsConcurrentProjectionChange(t *testing.T) {
	t.Parallel()

	globalDB, ctx, taskRecord, run, _ := setupCurrentRunProjectionTest(t, "clear-race")
	otherRun := taskRunForTest("run-current-projection-clear-race-other", taskRecord.ID)
	if err := globalDB.CreateTaskRun(ctx, otherRun); err != nil {
		t.Fatalf("CreateTaskRun(other) error = %v", err)
	}
	if _, err := globalDB.db.ExecContext(
		ctx,
		`UPDATE tasks SET current_run_id = ? WHERE id = ?`,
		run.ID,
		taskRecord.ID,
	); err != nil {
		t.Fatalf("seed current_run_id error = %v", err)
	}

	injected := false
	exec := projectionRaceExecutor{
		taskSQLExecutor: globalDB.db,
		beforeExec: func(ctx context.Context) error {
			if injected {
				return nil
			}
			injected = true
			_, err := globalDB.db.ExecContext(
				ctx,
				`UPDATE tasks SET current_run_id = ? WHERE id = ?`,
				otherRun.ID,
				taskRecord.ID,
			)
			return err
		},
	}

	err := clearTaskCurrentRunProjection(ctx, exec, taskRecord.ID, run.ID)
	if !errors.Is(err, taskpkg.ErrInvalidStatusTransition) {
		t.Fatalf("clearTaskCurrentRunProjection() error = %v, want %v", err, taskpkg.ErrInvalidStatusTransition)
	}
	assertTaskCurrentRunProjection(ctx, t, globalDB, taskRecord.ID, otherRun.ID)
}

func TestGlobalDBClaimNextRunReturnsSafeCoordinationChannelMetadata(t *testing.T) {
	globalDB := openTestGlobalDB(t)
	ctx := testutil.Context(t)
	workspaceID := registerWorkspaceForGlobalTests(
		t,
		globalDB,
		"claim-channel",
		filepath.Join(t.TempDir(), "claim-channel"),
	)
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	globalDB.now = func() time.Time { return now }
	if err := globalDB.WriteNetworkChannel(ctx, store.NetworkChannelEntry{
		Channel:     "coord.core",
		WorkspaceID: workspaceID,
		Purpose:     "Worker coordination",
		CreatedBy:   "coordinator",
	}); err != nil {
		t.Fatalf("WriteNetworkChannel() error = %v", err)
	}

	taskRecord := taskRecordForTest("task-channel-claim")
	taskRecord.Scope = taskpkg.ScopeWorkspace
	taskRecord.WorkspaceID = workspaceID
	taskRecord.Status = taskpkg.TaskStatusReady
	if err := globalDB.CreateTask(ctx, taskRecord); err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	run := taskRunForTest("run-channel-claim", taskRecord.ID)
	run.CoordinationChannelID = "coord.core"
	run.Metadata = json.RawMessage(`{"workflow_id":"wf-1"}`)
	if err := globalDB.CreateTaskRun(ctx, run); err != nil {
		t.Fatalf("CreateTaskRun() error = %v", err)
	}

	claim, err := globalDB.ClaimNextRun(ctx, taskpkg.ClaimCriteria{
		Scope:            taskpkg.ScopeWorkspace,
		WorkspaceID:      workspaceID,
		ClaimerSessionID: "sess-channel",
		LeaseDuration:    time.Minute,
		Now:              now.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("ClaimNextRun() error = %v", err)
	}
	if claim.CoordinationChannel == nil {
		t.Fatal("CoordinationChannel = nil, want metadata for channel-bound run")
	}
	if got, want := claim.CoordinationChannel.ID, "coord.core"; got != want {
		t.Fatalf("CoordinationChannel.ID = %q, want %q", got, want)
	}
	if got, want := claim.CoordinationChannel.DisplayName, "coord.core"; got != want {
		t.Fatalf("CoordinationChannel.DisplayName = %q, want %q", got, want)
	}
	if got, want := claim.CoordinationChannel.Purpose, "Worker coordination"; got != want {
		t.Fatalf("CoordinationChannel.Purpose = %q, want %q", got, want)
	}
	if got, want := claim.CoordinationChannel.WorkflowID, "wf-1"; got != want {
		t.Fatalf("CoordinationChannel.WorkflowID = %q, want %q", got, want)
	}
	encodedChannel, err := json.Marshal(claim.CoordinationChannel)
	if err != nil {
		t.Fatalf("json.Marshal(CoordinationChannel) error = %v", err)
	}
	var channelObject map[string]any
	if err := json.Unmarshal(encodedChannel, &channelObject); err != nil {
		t.Fatalf("CoordinationChannel did not marshal to JSON object: %s: %v", encodedChannel, err)
	}
	if containsJSONKey(t, encodedChannel, "claim_token") {
		t.Fatalf("CoordinationChannel JSON contains claim_token: %s", encodedChannel)
	}
	if _, err := globalDB.ReleaseRunLease(ctx, taskpkg.LeaseRelease{
		RunID:      claim.Run.ID,
		ClaimToken: "coord.core",
		Reason:     "channel metadata is not ownership",
		Now:        now.Add(2 * time.Second),
	}); !errors.Is(err, taskpkg.ErrInvalidClaimToken) {
		t.Fatalf("ReleaseRunLease(channel as token) error = %v, want %v", err, taskpkg.ErrInvalidClaimToken)
	}
}

func TestGlobalDBClaimNextRunReturnsWorkspaceNetworkChannelMetadata(t *testing.T) {
	globalDB := openTestGlobalDB(t)
	ctx := testutil.Context(t)
	workspaceID := registerWorkspaceForGlobalTests(
		t,
		globalDB,
		"claim-network-channel",
		filepath.Join(t.TempDir(), "claim-network-channel"),
	)
	now := time.Date(2026, 4, 26, 12, 30, 0, 0, time.UTC)
	globalDB.now = func() time.Time { return now }
	if err := globalDB.WriteNetworkChannel(ctx, store.NetworkChannelEntry{
		Channel:     "builders",
		WorkspaceID: workspaceID,
		Purpose:     "Build coordination",
		CreatedBy:   "coordinator",
	}); err != nil {
		t.Fatalf("WriteNetworkChannel() error = %v", err)
	}

	taskRecord := taskRecordForTest("task-network-channel-claim")
	taskRecord.Scope = taskpkg.ScopeWorkspace
	taskRecord.WorkspaceID = workspaceID
	taskRecord.Status = taskpkg.TaskStatusReady
	if err := globalDB.CreateTask(ctx, taskRecord); err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	_, run, existing, err := globalDB.ReserveQueuedRun(
		ctx,
		taskRecord.ID,
		"run-network-channel-claim",
		"idem-network-channel-claim",
		taskpkg.Origin{Kind: taskpkg.OriginKindDaemon, Ref: "scheduler"},
		"builders",
		json.RawMessage(`{"workflow_id":"wf-build"}`),
		now,
	)
	if err != nil {
		t.Fatalf("ReserveQueuedRun() error = %v", err)
	}
	if existing {
		t.Fatal("ReserveQueuedRun() existing = true, want new run")
	}
	if got, want := run.CoordinationChannelID, "builders"; got != want {
		t.Fatalf("ReserveQueuedRun().CoordinationChannelID = %q, want %q", got, want)
	}

	claim, err := globalDB.ClaimNextRun(ctx, taskpkg.ClaimCriteria{
		Scope:            taskpkg.ScopeWorkspace,
		WorkspaceID:      workspaceID,
		ClaimerSessionID: "sess-build",
		LeaseDuration:    time.Minute,
		Now:              now.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("ClaimNextRun() error = %v", err)
	}
	if claim.CoordinationChannel == nil {
		t.Fatal("CoordinationChannel = nil, want metadata for workspace channel-bound run")
	}
	if got, want := claim.CoordinationChannel.ID, "builders"; got != want {
		t.Fatalf("CoordinationChannel.ID = %q, want %q", got, want)
	}
	if got, want := claim.CoordinationChannel.Purpose, "Build coordination"; got != want {
		t.Fatalf("CoordinationChannel.Purpose = %q, want %q", got, want)
	}
	if got, want := claim.CoordinationChannel.WorkflowID, "wf-build"; got != want {
		t.Fatalf("CoordinationChannel.WorkflowID = %q, want %q", got, want)
	}
}

func TestGlobalDBReserveQueuedRunCreatesStableWorkspaceCoordinationChannel(t *testing.T) {
	globalDB := openTestGlobalDB(t)
	ctx := testutil.Context(t)
	workspaceID := registerWorkspaceForGlobalTests(
		t,
		globalDB,
		"derived-run-channel",
		filepath.Join(t.TempDir(), "derived-run-channel"),
	)

	taskRecord := taskRecordForTest("task-derived-channel")
	taskRecord.Scope = taskpkg.ScopeWorkspace
	taskRecord.WorkspaceID = workspaceID
	taskRecord.Status = taskpkg.TaskStatusReady
	if err := globalDB.CreateTask(ctx, taskRecord); err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	origin := taskpkg.Origin{Kind: taskpkg.OriginKindCLI, Ref: "agh task start"}
	queuedAt := time.Date(2026, 4, 26, 13, 0, 0, 0, time.UTC)

	_, first, existing, err := globalDB.ReserveQueuedRun(
		ctx,
		taskRecord.ID,
		"run-derived-channel",
		"idem-derived-channel",
		origin,
		"",
		nil,
		queuedAt,
	)
	if err != nil {
		t.Fatalf("ReserveQueuedRun(first) error = %v", err)
	}
	if existing {
		t.Fatal("ReserveQueuedRun(first) existing = true, want false")
	}
	if got, want := first.CoordinationChannelID, "coord-run-derived-channel"; got != want {
		t.Fatalf("first.CoordinationChannelID = %q, want %q", got, want)
	}
	channel, err := globalDB.GetNetworkChannel(ctx, store.NetworkChannelRef{
		WorkspaceID: workspaceID,
		Channel:     first.CoordinationChannelID,
	})
	if err != nil {
		t.Fatalf("GetNetworkChannel(derived) error = %v", err)
	}
	if got, want := channel.WorkspaceID, workspaceID; got != want {
		t.Fatalf("channel.WorkspaceID = %q, want %q", got, want)
	}
	if got, want := channel.Purpose, "task_run_coordination"; got != want {
		t.Fatalf("channel.Purpose = %q, want %q", got, want)
	}

	_, second, existing, err := globalDB.ReserveQueuedRun(
		ctx,
		taskRecord.ID,
		"run-duplicate-ignored",
		"idem-derived-channel",
		origin,
		"",
		nil,
		queuedAt.Add(time.Minute),
	)
	if err != nil {
		t.Fatalf("ReserveQueuedRun(second) error = %v", err)
	}
	if !existing {
		t.Fatal("ReserveQueuedRun(second) existing = false, want true")
	}
	if got, want := second.ID, first.ID; got != want {
		t.Fatalf("second.ID = %q, want %q", got, want)
	}
	channels, err := globalDB.ListNetworkChannels(ctx, store.NetworkChannelQuery{WorkspaceID: workspaceID})
	if err != nil {
		t.Fatalf("ListNetworkChannels() error = %v", err)
	}
	if len(channels) != 1 {
		t.Fatalf("len(ListNetworkChannels) = %d, want 1", len(channels))
	}
}

func TestGlobalDBClaimNextRunSkipsBlockedTasks(t *testing.T) {
	globalDB := openTestGlobalDB(t)
	ctx := testutil.Context(t)
	taskRecord := taskRecordForTest("task-blocked-claim")
	taskRecord.Status = taskpkg.TaskStatusBlocked
	if err := globalDB.CreateTask(ctx, taskRecord); err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	run := taskRunForTest("run-blocked-claim", taskRecord.ID)
	if err := globalDB.CreateTaskRun(ctx, run); err != nil {
		t.Fatalf("CreateTaskRun() error = %v", err)
	}

	if _, err := globalDB.ClaimNextRun(ctx, taskpkg.ClaimCriteria{
		Scope:            taskpkg.ScopeGlobal,
		ClaimerSessionID: "sess-blocked",
		LeaseDuration:    time.Minute,
		Now:              time.Date(2026, 4, 26, 13, 5, 0, 0, time.UTC),
	}); !errors.Is(err, taskpkg.ErrNoClaimableRun) {
		t.Fatalf("ClaimNextRun(blocked) error = %v, want %v", err, taskpkg.ErrNoClaimableRun)
	}
}

func TestGlobalDBTaskPauseControlsClaimEligibilityAndBacklog(t *testing.T) {
	t.Parallel()

	t.Run("Should skip direct and inherited paused tasks in claim and backlog scans", func(t *testing.T) {
		t.Parallel()

		globalDB := openTestGlobalDB(t)
		ctx := testutil.Context(t)
		pausedAt := time.Date(2026, 5, 21, 9, 30, 0, 0, time.UTC)

		root := taskRecordForTest("task-pause-root")
		root.Status = taskpkg.TaskStatusReady
		if err := globalDB.CreateTask(ctx, root); err != nil {
			t.Fatalf("CreateTask(root) error = %v", err)
		}
		child := taskRecordForTest("task-pause-child")
		child.ParentTaskID = root.ID
		child.Status = taskpkg.TaskStatusReady
		if err := globalDB.CreateTask(ctx, child); err != nil {
			t.Fatalf("CreateTask(child) error = %v", err)
		}
		peer := taskRecordForTest("task-pause-peer")
		peer.Status = taskpkg.TaskStatusReady
		peer.Priority = taskpkg.PriorityHigh
		if err := globalDB.CreateTask(ctx, peer); err != nil {
			t.Fatalf("CreateTask(peer) error = %v", err)
		}
		for _, run := range []taskpkg.Run{
			taskRunForTest("run-pause-child", child.ID),
			taskRunForTest("run-pause-peer", peer.ID),
		} {
			if err := globalDB.CreateTaskRun(ctx, run); err != nil {
				t.Fatalf("CreateTaskRun(%q) error = %v", run.ID, err)
			}
		}
		if _, err := globalDB.PauseTask(ctx, taskpkg.PauseMutation{
			TaskID:   root.ID,
			Actor:    "human:operator",
			Reason:   "provider incident",
			PausedAt: pausedAt,
		}); err != nil {
			t.Fatalf("PauseTask(root) error = %v", err)
		}

		effectivePaused, pausedByTaskID, err := globalDB.IsTaskEffectivelyPaused(ctx, child.ID)
		if err != nil {
			t.Fatalf("IsTaskEffectivelyPaused(child) error = %v", err)
		}
		if !effectivePaused || pausedByTaskID != root.ID {
			t.Fatalf("child effective pause = %v by %q, want true by root", effectivePaused, pausedByTaskID)
		}
		visibleCount, err := globalDB.CountQueuedTaskRuns(ctx, false)
		if err != nil {
			t.Fatalf("CountQueuedTaskRuns(false) error = %v", err)
		}
		allCount, err := globalDB.CountQueuedTaskRuns(ctx, true)
		if err != nil {
			t.Fatalf("CountQueuedTaskRuns(true) error = %v", err)
		}
		if visibleCount != 1 || allCount != 2 {
			t.Fatalf("queued counts = visible %d all %d, want 1 and 2", visibleCount, allCount)
		}

		backlog, err := globalDB.SchedulerBacklog(ctx, taskpkg.SchedulerBacklogQuery{IncludePaused: true})
		if err != nil {
			t.Fatalf("SchedulerBacklog(include paused) error = %v", err)
		}
		pausedByRun := make(map[string]taskpkg.SchedulerBacklogRun, len(backlog.Runs))
		for _, item := range backlog.Runs {
			pausedByRun[item.Run.ID] = item
		}
		if item := pausedByRun["run-pause-child"]; !item.EffectivePaused || item.PausedByTaskID != root.ID {
			t.Fatalf("child backlog item = %#v, want inherited pause from root", item)
		}
		if item := pausedByRun["run-pause-peer"]; item.EffectivePaused {
			t.Fatalf("peer backlog item = %#v, want unpaused", item)
		}

		claim, err := globalDB.ClaimNextRun(ctx, taskpkg.ClaimCriteria{
			Scope:            taskpkg.ScopeGlobal,
			ClaimerSessionID: "sess-pause-peer",
			LeaseDuration:    time.Minute,
			Now:              pausedAt.Add(time.Minute),
		})
		if err != nil {
			t.Fatalf("ClaimNextRun(peer) error = %v", err)
		}
		if got, want := claim.Run.ID, "run-pause-peer"; got != want {
			t.Fatalf("ClaimNextRun(peer) run = %q, want %q", got, want)
		}
		if _, err := globalDB.ResumeTask(ctx, taskpkg.ResumeMutation{
			TaskID:    root.ID,
			ResumedAt: pausedAt.Add(2 * time.Minute),
		}); err != nil {
			t.Fatalf("ResumeTask(root) error = %v", err)
		}
		childClaim, err := globalDB.ClaimNextRun(ctx, taskpkg.ClaimCriteria{
			Scope:            taskpkg.ScopeGlobal,
			ClaimerSessionID: "sess-pause-child",
			LeaseDuration:    time.Minute,
			Now:              pausedAt.Add(3 * time.Minute),
		})
		if err != nil {
			t.Fatalf("ClaimNextRun(child after resume) error = %v", err)
		}
		if got, want := childClaim.Run.ID, "run-pause-child"; got != want {
			t.Fatalf("ClaimNextRun(child after resume) run = %q, want %q", got, want)
		}
	})
}

func setupCurrentRunProjectionTest(
	t *testing.T,
	suffix string,
) (*GlobalDB, context.Context, taskpkg.Task, taskpkg.Run, time.Time) {
	t.Helper()

	globalDB := openTestGlobalDB(t)
	ctx := testutil.Context(t)
	now := time.Date(2026, 4, 26, 14, 0, 0, 0, time.UTC)
	taskRecord := taskRecordForTest("task-current-projection-" + suffix)
	taskRecord.Status = taskpkg.TaskStatusReady
	if err := globalDB.CreateTask(ctx, taskRecord); err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	run := taskRunForTest("run-current-projection-"+suffix, taskRecord.ID)
	if err := globalDB.CreateTaskRun(ctx, run); err != nil {
		t.Fatalf("CreateTaskRun() error = %v", err)
	}
	return globalDB, ctx, taskRecord, run, now
}

func claimProjectionRunForTest(
	ctx context.Context,
	t *testing.T,
	globalDB *GlobalDB,
	sessionID string,
	now time.Time,
) taskpkg.ClaimResult {
	t.Helper()

	claim, err := globalDB.ClaimNextRun(ctx, taskpkg.ClaimCriteria{
		Scope:            taskpkg.ScopeGlobal,
		ClaimerSessionID: sessionID,
		LeaseDuration:    time.Minute,
		Now:              now,
	})
	if err != nil {
		t.Fatalf("ClaimNextRun() error = %v", err)
	}
	return claim
}

type projectionRaceExecutor struct {
	taskSQLExecutor
	beforeExec func(ctx context.Context) error
}

func (e projectionRaceExecutor) ExecContext(
	ctx context.Context,
	query string,
	args ...any,
) (sql.Result, error) {
	if e.beforeExec != nil {
		if err := e.beforeExec(ctx); err != nil {
			return nil, err
		}
	}
	return e.taskSQLExecutor.ExecContext(ctx, query, args...)
}

func assertTaskCurrentRunProjection(
	ctx context.Context,
	t *testing.T,
	globalDB *GlobalDB,
	taskID string,
	want string,
) {
	t.Helper()

	taskRecord, err := globalDB.GetTask(ctx, taskID)
	if err != nil {
		t.Fatalf("GetTask(%q) error = %v", taskID, err)
	}
	if got := taskRecord.CurrentRunID; got != want {
		t.Fatalf("task.CurrentRunID = %q, want %q", got, want)
	}
}

func leasedRunForGlobalTest(
	t *testing.T,
	id string,
	taskID string,
	sessionID string,
	rawToken string,
	leaseUntil time.Time,
) taskpkg.Run {
	t.Helper()

	hash, err := taskpkg.ClaimTokenHash(rawToken)
	if err != nil {
		t.Fatalf("ClaimTokenHash(%q) error = %v", rawToken, err)
	}
	run := taskRunForTest(id, taskID)
	run.Status = taskpkg.TaskRunStatusClaimed
	run.ClaimedBy = actorForTest(taskpkg.ActorKindAgentSession, sessionID)
	run.SessionID = sessionID
	run.ClaimTokenHash = hash
	run.ClaimedAt = leaseUntil.Add(-time.Minute)
	run.HeartbeatAt = leaseUntil.Add(-30 * time.Second)
	run.LeaseUntil = leaseUntil
	return run
}

func containsJSONKey(t *testing.T, raw []byte, key string) bool {
	t.Helper()

	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(%s) error = %v", raw, err)
	}
	return containsJSONKeyValue(decoded, key)
}

func containsJSONKeyValue(value any, key string) bool {
	switch typed := value.(type) {
	case map[string]any:
		for field, nested := range typed {
			if field == key {
				return true
			}
			if containsJSONKeyValue(nested, key) {
				return true
			}
		}
	case []any:
		for _, nested := range typed {
			if containsJSONKeyValue(nested, key) {
				return true
			}
		}
	}
	return false
}

func writeInvalidSoulFixture(t *testing.T, workspaceRoot string) {
	t.Helper()

	soulDir := filepath.Join(workspaceRoot, ".agh", "agents", "coder")
	if err := os.MkdirAll(soulDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", soulDir, err)
	}
	soulPath := filepath.Join(soulDir, "SOUL.md")
	content := []byte("---\nprovider: claude\n---\nThis invalid file must not be read during claim.\n")
	if err := os.WriteFile(soulPath, content, 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", soulPath, err)
	}
}

func assertRunSoulMetadata(
	t *testing.T,
	raw json.RawMessage,
	workflowID string,
	snapshotID string,
	digest string,
	agentName string,
	capturedAt time.Time,
) {
	t.Helper()

	var decoded struct {
		WorkflowID string `json:"workflow_id"`
		Soul       struct {
			SnapshotID string    `json:"snapshot_id"`
			Digest     string    `json:"digest"`
			AgentName  string    `json:"agent_name"`
			CapturedAt time.Time `json:"captured_at"`
		} `json:"soul"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(run.Metadata) error = %v; raw=%s", err, raw)
	}
	if decoded.WorkflowID != workflowID {
		t.Fatalf("metadata.workflow_id = %q, want %q", decoded.WorkflowID, workflowID)
	}
	if decoded.Soul.SnapshotID != snapshotID ||
		decoded.Soul.Digest != digest ||
		decoded.Soul.AgentName != agentName ||
		!decoded.Soul.CapturedAt.Equal(capturedAt) {
		t.Fatalf(
			"metadata.soul = %#v, want snapshot=%q digest=%q agent=%q captured_at=%s",
			decoded.Soul,
			snapshotID,
			digest,
			agentName,
			capturedAt,
		)
	}
}
