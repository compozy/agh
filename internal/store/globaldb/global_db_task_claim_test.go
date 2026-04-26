package globaldb

import (
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/store"
	taskpkg "github.com/pedronauck/agh/internal/task"
	"github.com/pedronauck/agh/internal/testutil"
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
	attempts := make([]claimAttempt, 2)
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
	if storedRaw.Valid {
		t.Fatalf("stored raw claim_token = %q, want NULL", storedRaw.String)
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
	channel, err := globalDB.GetNetworkChannel(ctx, first.CoordinationChannelID)
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
