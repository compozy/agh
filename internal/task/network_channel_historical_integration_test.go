//go:build integration

package task_test

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/network"
	"github.com/pedronauck/agh/internal/store"
	taskpkg "github.com/pedronauck/agh/internal/task"
	"github.com/pedronauck/agh/internal/testutil"
)

func TestTaskManagerHistoricalNetworkChannelIntegration(t *testing.T) {
	t.Parallel()

	t.Run("Should start tasks against historical network channels without deriving a new coordination channel", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		db := openTaskManagerGlobalDB(t)
		workspaceID := registerTaskManagerWorkspace(t, db, "historical-channel", filepath.Join(t.TempDir(), "workspace"))
		manager := newTaskManagerIntegration(
			t,
			db,
			taskpkg.WithNetworkChannelValidator(network.ValidateChannel),
		)

		channelTimestamp := time.Date(2026, 4, 28, 4, 54, 52, 0, time.UTC)
		if err := db.WriteNetworkChannel(ctx, store.NetworkChannelEntry{
			Channel:     "scope-direct-history",
			WorkspaceID: workspaceID,
			Purpose:     "History-only direct lane persistence validation",
			CreatedBy:   "founder",
			CreatedAt:   channelTimestamp,
			UpdatedAt:   channelTimestamp,
		}); err != nil {
			t.Fatalf("WriteNetworkChannel() error = %v", err)
		}

		actor, err := taskpkg.DeriveHumanActorContext("user-1", taskpkg.OriginKindCLI, "agh task start")
		if err != nil {
			t.Fatalf("DeriveHumanActorContext() error = %v", err)
		}

		taskRecord, err := manager.CreateTask(ctx, taskpkg.CreateTask{
			Scope:          taskpkg.ScopeWorkspace,
			WorkspaceID:    workspaceID,
			NetworkChannel: "scope-direct-history",
			Title:          "History-only channel task repro",
		}, actor)
		if err != nil {
			t.Fatalf("CreateTask() error = %v", err)
		}

		execution, err := manager.StartTask(ctx, taskRecord.ID, taskpkg.ExecutionRequest{}, actor)
		if err != nil {
			t.Fatalf("StartTask() error = %v", err)
		}
		if got, want := execution.Run.NetworkChannel, "scope-direct-history"; got != want {
			t.Fatalf("execution.Run.NetworkChannel = %q, want %q", got, want)
		}
		if got, want := execution.Run.CoordinationChannelID, "scope-direct-history"; got != want {
			t.Fatalf("execution.Run.CoordinationChannelID = %q, want %q", got, want)
		}

		storedRun, err := db.GetTaskRun(ctx, execution.Run.ID)
		if err != nil {
			t.Fatalf("GetTaskRun() error = %v", err)
		}
		if got, want := storedRun.NetworkChannel, "scope-direct-history"; got != want {
			t.Fatalf("storedRun.NetworkChannel = %q, want %q", got, want)
		}
		if got, want := storedRun.CoordinationChannelID, "scope-direct-history"; got != want {
			t.Fatalf("storedRun.CoordinationChannelID = %q, want %q", got, want)
		}

		channels, err := db.ListNetworkChannels(ctx, store.NetworkChannelQuery{WorkspaceID: workspaceID})
		if err != nil {
			t.Fatalf("ListNetworkChannels() error = %v", err)
		}
		if got, want := len(channels), 1; got != want {
			t.Fatalf("len(channels) = %d, want %d", got, want)
		}
		if got, want := channels[0].Channel, "scope-direct-history"; got != want {
			t.Fatalf("channels[0].Channel = %q, want %q", got, want)
		}
	})

	t.Run("Should preserve historical network channels through claim attach start and completion", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		db := openTaskManagerGlobalDB(t)
		workspaceID := registerTaskManagerWorkspace(t, db, "historical-channel-lifecycle", filepath.Join(t.TempDir(), "workspace"))
		executor := &integrationSessionExecutor{}
		manager := newTaskManagerIntegration(
			t,
			db,
			taskpkg.WithNetworkChannelValidator(network.ValidateChannel),
			taskpkg.WithSessionExecutor(executor),
		)

		channelTimestamp := time.Date(2026, 4, 28, 5, 45, 3, 0, time.UTC)
		if err := db.WriteNetworkChannel(ctx, store.NetworkChannelEntry{
			Channel:     "scope-direct-history",
			WorkspaceID: workspaceID,
			Purpose:     "Historical task-run lifecycle validation",
			CreatedBy:   "founder",
			CreatedAt:   channelTimestamp,
			UpdatedAt:   channelTimestamp,
		}); err != nil {
			t.Fatalf("WriteNetworkChannel() error = %v", err)
		}

		actor, err := taskpkg.DeriveHumanActorContext("user-1", taskpkg.OriginKindCLI, "agh task run start")
		if err != nil {
			t.Fatalf("DeriveHumanActorContext() error = %v", err)
		}

		taskRecord, err := manager.CreateTask(ctx, taskpkg.CreateTask{
			Scope:          taskpkg.ScopeWorkspace,
			WorkspaceID:    workspaceID,
			NetworkChannel: "scope-direct-history",
			Title:          "History-only channel lifecycle repro",
		}, actor)
		if err != nil {
			t.Fatalf("CreateTask() error = %v", err)
		}

		execution, err := manager.StartTask(ctx, taskRecord.ID, taskpkg.ExecutionRequest{}, actor)
		if err != nil {
			t.Fatalf("StartTask() error = %v", err)
		}
		if got, want := execution.Run.Status, taskpkg.TaskRunStatusQueued; got != want {
			t.Fatalf("execution.Run.Status = %q, want %q", got, want)
		}
		if got, want := execution.Run.NetworkChannel, "scope-direct-history"; got != want {
			t.Fatalf("execution.Run.NetworkChannel = %q, want %q", got, want)
		}
		if got, want := execution.Run.CoordinationChannelID, "scope-direct-history"; got != want {
			t.Fatalf("execution.Run.CoordinationChannelID = %q, want %q", got, want)
		}

		run, err := manager.ClaimRun(ctx, execution.Run.ID, taskpkg.ClaimRun{}, actor)
		if err != nil {
			t.Fatalf("ClaimRun() error = %v", err)
		}
		if got, want := run.Status, taskpkg.TaskRunStatusClaimed; got != want {
			t.Fatalf("claimed run status = %q, want %q", got, want)
		}
		if got, want := run.NetworkChannel, "scope-direct-history"; got != want {
			t.Fatalf("claimed run network channel = %q, want %q", got, want)
		}
		if got, want := run.CoordinationChannelID, "scope-direct-history"; got != want {
			t.Fatalf("claimed run coordination channel = %q, want %q", got, want)
		}

		run, err = manager.AttachRunSession(ctx, run.ID, "sess-history-worker", actor)
		if err != nil {
			t.Fatalf("AttachRunSession() error = %v", err)
		}
		if got, want := run.Status, taskpkg.TaskRunStatusStarting; got != want {
			t.Fatalf("attached run status = %q, want %q", got, want)
		}
		if got, want := run.SessionID, "sess-history-worker"; got != want {
			t.Fatalf("attached run session id = %q, want %q", got, want)
		}
		if got, want := run.NetworkChannel, "scope-direct-history"; got != want {
			t.Fatalf("attached run network channel = %q, want %q", got, want)
		}
		if got, want := run.CoordinationChannelID, "scope-direct-history"; got != want {
			t.Fatalf("attached run coordination channel = %q, want %q", got, want)
		}

		run, err = manager.StartRun(ctx, run.ID, taskpkg.StartRun{}, actor)
		if err != nil {
			t.Fatalf("StartRun() error = %v", err)
		}
		if got, want := run.Status, taskpkg.TaskRunStatusRunning; got != want {
			t.Fatalf("started run status = %q, want %q", got, want)
		}
		if got, want := run.SessionID, "sess-history-worker"; got != want {
			t.Fatalf("started run session id = %q, want %q", got, want)
		}

		run, err = manager.CompleteRun(ctx, run.ID, taskpkg.RunResult{
			Value: json.RawMessage(`{"ok":true,"path":"historical-channel-lifecycle"}`),
		}, actor)
		if err != nil {
			t.Fatalf("CompleteRun() error = %v", err)
		}
		if got, want := run.Status, taskpkg.TaskRunStatusCompleted; got != want {
			t.Fatalf("completed run status = %q, want %q", got, want)
		}
		if got, want := string(run.Result), `{"ok":true,"path":"historical-channel-lifecycle"}`; got != want {
			t.Fatalf("completed run result = %s, want %s", got, want)
		}
		if got, want := run.NetworkChannel, "scope-direct-history"; got != want {
			t.Fatalf("completed run network channel = %q, want %q", got, want)
		}
		if got, want := run.CoordinationChannelID, "scope-direct-history"; got != want {
			t.Fatalf("completed run coordination channel = %q, want %q", got, want)
		}

		storedTask, err := db.GetTask(ctx, taskRecord.ID)
		if err != nil {
			t.Fatalf("GetTask() error = %v", err)
		}
		if got, want := storedTask.Status, taskpkg.TaskStatusCompleted; got != want {
			t.Fatalf("storedTask.Status = %q, want %q", got, want)
		}

		storedRun, err := db.GetTaskRun(ctx, run.ID)
		if err != nil {
			t.Fatalf("GetTaskRun() error = %v", err)
		}
		if got, want := storedRun.Status, taskpkg.TaskRunStatusCompleted; got != want {
			t.Fatalf("storedRun.Status = %q, want %q", got, want)
		}
		if got, want := storedRun.SessionID, "sess-history-worker"; got != want {
			t.Fatalf("storedRun.SessionID = %q, want %q", got, want)
		}
		if got, want := storedRun.NetworkChannel, "scope-direct-history"; got != want {
			t.Fatalf("storedRun.NetworkChannel = %q, want %q", got, want)
		}
		if got, want := storedRun.CoordinationChannelID, "scope-direct-history"; got != want {
			t.Fatalf("storedRun.CoordinationChannelID = %q, want %q", got, want)
		}

		if got, want := len(executor.requestStopCalls), 1; got != want {
			t.Fatalf("len(requestStopCalls) = %d, want %d", got, want)
		}
		if got, want := executor.requestStopCalls[0].SessionID, "sess-history-worker"; got != want {
			t.Fatalf("requestStopCalls[0].SessionID = %q, want %q", got, want)
		}
		if got, want := executor.requestStopCalls[0].Reason, taskpkg.StopReasonCompleted; got != want {
			t.Fatalf("requestStopCalls[0].Reason = %q, want %q", got, want)
		}
		if got, want := len(executor.forceStopCalls), 1; got != want {
			t.Fatalf("len(forceStopCalls) = %d, want %d", got, want)
		}
		if got, want := executor.forceStopCalls[0].SessionID, "sess-history-worker"; got != want {
			t.Fatalf("forceStopCalls[0].SessionID = %q, want %q", got, want)
		}
		if got, want := executor.forceStopCalls[0].Reason, taskpkg.StopReasonCompleted; got != want {
			t.Fatalf("forceStopCalls[0].Reason = %q, want %q", got, want)
		}

		events, err := db.ListTaskEvents(ctx, taskpkg.EventQuery{TaskID: taskRecord.ID})
		if err != nil {
			t.Fatalf("ListTaskEvents() error = %v", err)
		}
		for _, eventType := range []string{
			"task.run_claimed",
			"task.run_session_bound",
			"task.run_started",
			"task.run_completed",
		} {
			if !containsEventType(events, eventType) {
				t.Fatalf("event types = %#v, want %q", sortedEventTypes(events), eventType)
			}
		}
	})

	t.Run("Should preserve historical network channels through lease claim heartbeat and completion", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		db := openTaskManagerGlobalDB(t)
		workspaceID := registerTaskManagerWorkspace(t, db, "historical-channel-lease", filepath.Join(t.TempDir(), "workspace"))
		manager := newTaskManagerIntegration(
			t,
			db,
			taskpkg.WithNetworkChannelValidator(network.ValidateChannel),
		)

		channelTimestamp := time.Date(2026, 4, 28, 6, 10, 36, 0, time.UTC)
		if err := db.WriteNetworkChannel(ctx, store.NetworkChannelEntry{
			Channel:     "scope-direct-history",
			WorkspaceID: workspaceID,
			Purpose:     "Historical task-run lease validation",
			CreatedBy:   "founder",
			CreatedAt:   channelTimestamp,
			UpdatedAt:   channelTimestamp,
		}); err != nil {
			t.Fatalf("WriteNetworkChannel() error = %v", err)
		}

		operator, err := taskpkg.DeriveHumanActorContext("user-1", taskpkg.OriginKindCLI, "agh task next")
		if err != nil {
			t.Fatalf("DeriveHumanActorContext() error = %v", err)
		}

		taskRecord, err := manager.CreateTask(ctx, taskpkg.CreateTask{
			Scope:          taskpkg.ScopeWorkspace,
			WorkspaceID:    workspaceID,
			NetworkChannel: "scope-direct-history",
			Title:          "History-only channel lease repro",
		}, operator)
		if err != nil {
			t.Fatalf("CreateTask() error = %v", err)
		}

		execution, err := manager.StartTask(ctx, taskRecord.ID, taskpkg.ExecutionRequest{}, operator)
		if err != nil {
			t.Fatalf("StartTask() error = %v", err)
		}
		if got, want := execution.Run.Status, taskpkg.TaskRunStatusQueued; got != want {
			t.Fatalf("execution.Run.Status = %q, want %q", got, want)
		}
		if got, want := execution.Run.NetworkChannel, "scope-direct-history"; got != want {
			t.Fatalf("execution.Run.NetworkChannel = %q, want %q", got, want)
		}
		if got, want := execution.Run.CoordinationChannelID, "scope-direct-history"; got != want {
			t.Fatalf("execution.Run.CoordinationChannelID = %q, want %q", got, want)
		}

		agent, err := taskpkg.DeriveAgentSessionActorContext("sess-history-lease")
		if err != nil {
			t.Fatalf("DeriveAgentSessionActorContext() error = %v", err)
		}
		claim, err := manager.ClaimNextRun(ctx, taskpkg.ClaimCriteria{
			Scope:                 taskpkg.ScopeWorkspace,
			WorkspaceID:           workspaceID,
			ClaimerSessionID:      "sess-history-lease",
			CoordinationChannelID: "scope-direct-history",
			LeaseDuration:         time.Minute,
		}, agent)
		if err != nil {
			t.Fatalf("ClaimNextRun() error = %v", err)
		}
		if claim == nil {
			t.Fatal("ClaimNextRun() = nil, want claimed run")
		}
		if got, want := claim.Run.ID, execution.Run.ID; got != want {
			t.Fatalf("claim.Run.ID = %q, want %q", got, want)
		}
		if got, want := claim.Run.Status, taskpkg.TaskRunStatusClaimed; got != want {
			t.Fatalf("claim.Run.Status = %q, want %q", got, want)
		}
		if got, want := claim.Run.SessionID, "sess-history-lease"; got != want {
			t.Fatalf("claim.Run.SessionID = %q, want %q", got, want)
		}
		if got, want := claim.Run.NetworkChannel, "scope-direct-history"; got != want {
			t.Fatalf("claim.Run.NetworkChannel = %q, want %q", got, want)
		}
		if got, want := claim.Run.CoordinationChannelID, "scope-direct-history"; got != want {
			t.Fatalf("claim.Run.CoordinationChannelID = %q, want %q", got, want)
		}
		if claim.CoordinationChannel == nil || claim.CoordinationChannel.ID != "scope-direct-history" {
			t.Fatalf("claim.CoordinationChannel = %#v, want scope-direct-history", claim.CoordinationChannel)
		}
		if claim.ClaimToken == "" {
			t.Fatal("claim.ClaimToken = empty, want raw token for lease mutations")
		}

		heartbeat, err := manager.HeartbeatRunLease(ctx, taskpkg.LeaseHeartbeat{
			RunID:         claim.Run.ID,
			ClaimToken:    claim.ClaimToken,
			LeaseDuration: 2 * time.Minute,
		}, agent)
		if err != nil {
			t.Fatalf("HeartbeatRunLease() error = %v", err)
		}
		if got, want := heartbeat.Status, taskpkg.TaskRunStatusClaimed; got != want {
			t.Fatalf("heartbeat.Status = %q, want %q", got, want)
		}
		if got, want := heartbeat.SessionID, "sess-history-lease"; got != want {
			t.Fatalf("heartbeat.SessionID = %q, want %q", got, want)
		}
		if got, want := heartbeat.NetworkChannel, "scope-direct-history"; got != want {
			t.Fatalf("heartbeat.NetworkChannel = %q, want %q", got, want)
		}
		if got, want := heartbeat.CoordinationChannelID, "scope-direct-history"; got != want {
			t.Fatalf("heartbeat.CoordinationChannelID = %q, want %q", got, want)
		}
		if !heartbeat.LeaseUntil.After(claim.Run.ClaimedAt) {
			t.Fatalf("heartbeat.LeaseUntil = %v, want after %v", heartbeat.LeaseUntil, claim.Run.ClaimedAt)
		}
		if heartbeat.HeartbeatAt.IsZero() {
			t.Fatal("heartbeat.HeartbeatAt = zero, want refreshed heartbeat")
		}

		completed, err := manager.CompleteRunLease(ctx, taskpkg.LeaseCompletion{
			RunID:      claim.Run.ID,
			ClaimToken: claim.ClaimToken,
			Result: taskpkg.RunResult{
				Value: json.RawMessage(`{"ok":true,"path":"historical-channel-lease"}`),
			},
		}, agent)
		if err != nil {
			t.Fatalf("CompleteRunLease() error = %v", err)
		}
		if got, want := completed.Status, taskpkg.TaskRunStatusCompleted; got != want {
			t.Fatalf("completed.Status = %q, want %q", got, want)
		}
		if got, want := completed.SessionID, "sess-history-lease"; got != want {
			t.Fatalf("completed.SessionID = %q, want %q", got, want)
		}
		if got, want := string(completed.Result), `{"ok":true,"path":"historical-channel-lease"}`; got != want {
			t.Fatalf("completed.Result = %s, want %s", got, want)
		}
		if got, want := completed.NetworkChannel, "scope-direct-history"; got != want {
			t.Fatalf("completed.NetworkChannel = %q, want %q", got, want)
		}
		if got, want := completed.CoordinationChannelID, "scope-direct-history"; got != want {
			t.Fatalf("completed.CoordinationChannelID = %q, want %q", got, want)
		}

		if _, err := manager.CompleteRunLease(ctx, taskpkg.LeaseCompletion{
			RunID:      claim.Run.ID,
			ClaimToken: claim.ClaimToken,
			Result: taskpkg.RunResult{
				Value: json.RawMessage(`{"ok":true,"path":"historical-channel-lease-second"}`),
			},
		}, agent); !errors.Is(err, taskpkg.ErrInvalidStatusTransition) {
			t.Fatalf("CompleteRunLease(second) error = %v, want %v", err, taskpkg.ErrInvalidStatusTransition)
		}

		storedTask, err := db.GetTask(ctx, taskRecord.ID)
		if err != nil {
			t.Fatalf("GetTask() error = %v", err)
		}
		if got, want := storedTask.Status, taskpkg.TaskStatusCompleted; got != want {
			t.Fatalf("storedTask.Status = %q, want %q", got, want)
		}

		storedRun, err := db.GetTaskRun(ctx, claim.Run.ID)
		if err != nil {
			t.Fatalf("GetTaskRun() error = %v", err)
		}
		if got, want := storedRun.Status, taskpkg.TaskRunStatusCompleted; got != want {
			t.Fatalf("storedRun.Status = %q, want %q", got, want)
		}
		if got, want := storedRun.SessionID, "sess-history-lease"; got != want {
			t.Fatalf("storedRun.SessionID = %q, want %q", got, want)
		}
		if got, want := storedRun.NetworkChannel, "scope-direct-history"; got != want {
			t.Fatalf("storedRun.NetworkChannel = %q, want %q", got, want)
		}
		if got, want := storedRun.CoordinationChannelID, "scope-direct-history"; got != want {
			t.Fatalf("storedRun.CoordinationChannelID = %q, want %q", got, want)
		}
		if got, want := string(storedRun.Result), `{"ok":true,"path":"historical-channel-lease"}`; got != want {
			t.Fatalf("storedRun.Result = %s, want %s", got, want)
		}

		events, err := db.ListTaskEvents(ctx, taskpkg.EventQuery{TaskID: taskRecord.ID})
		if err != nil {
			t.Fatalf("ListTaskEvents() error = %v", err)
		}
		for _, eventType := range []string{
			"task.run_claimed",
			"task.run_lease_extended",
			"task.run_completed",
		} {
			if !containsEventType(events, eventType) {
				t.Fatalf("event types = %#v, want %q", sortedEventTypes(events), eventType)
			}
		}
	})

	t.Run("Should preserve historical network channels through lease release reclaim and completion", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		db := openTaskManagerGlobalDB(t)
		workspaceID := registerTaskManagerWorkspace(t, db, "historical-channel-release-reclaim", filepath.Join(t.TempDir(), "workspace"))
		manager := newTaskManagerIntegration(
			t,
			db,
			taskpkg.WithNetworkChannelValidator(network.ValidateChannel),
		)

		channelTimestamp := time.Date(2026, 4, 28, 6, 28, 17, 0, time.UTC)
		if err := db.WriteNetworkChannel(ctx, store.NetworkChannelEntry{
			Channel:     "scope-direct-history",
			WorkspaceID: workspaceID,
			Purpose:     "Historical task-run release reclaim validation",
			CreatedBy:   "founder",
			CreatedAt:   channelTimestamp,
			UpdatedAt:   channelTimestamp,
		}); err != nil {
			t.Fatalf("WriteNetworkChannel() error = %v", err)
		}

		operator, err := taskpkg.DeriveHumanActorContext("user-1", taskpkg.OriginKindCLI, "agh task release")
		if err != nil {
			t.Fatalf("DeriveHumanActorContext() error = %v", err)
		}

		taskRecord, err := manager.CreateTask(ctx, taskpkg.CreateTask{
			Scope:          taskpkg.ScopeWorkspace,
			WorkspaceID:    workspaceID,
			NetworkChannel: "scope-direct-history",
			Title:          "History-only channel release reclaim repro",
		}, operator)
		if err != nil {
			t.Fatalf("CreateTask() error = %v", err)
		}

		execution, err := manager.StartTask(ctx, taskRecord.ID, taskpkg.ExecutionRequest{}, operator)
		if err != nil {
			t.Fatalf("StartTask() error = %v", err)
		}
		if got, want := execution.Run.Status, taskpkg.TaskRunStatusQueued; got != want {
			t.Fatalf("execution.Run.Status = %q, want %q", got, want)
		}
		if got, want := execution.Run.NetworkChannel, "scope-direct-history"; got != want {
			t.Fatalf("execution.Run.NetworkChannel = %q, want %q", got, want)
		}
		if got, want := execution.Run.CoordinationChannelID, "scope-direct-history"; got != want {
			t.Fatalf("execution.Run.CoordinationChannelID = %q, want %q", got, want)
		}

		agentA, err := taskpkg.DeriveAgentSessionActorContext("sess-history-release-a")
		if err != nil {
			t.Fatalf("DeriveAgentSessionActorContext(agentA) error = %v", err)
		}
		claimA, err := manager.ClaimNextRun(ctx, taskpkg.ClaimCriteria{
			Scope:                 taskpkg.ScopeWorkspace,
			WorkspaceID:           workspaceID,
			ClaimerSessionID:      "sess-history-release-a",
			CoordinationChannelID: "scope-direct-history",
			LeaseDuration:         time.Minute,
		}, agentA)
		if err != nil {
			t.Fatalf("ClaimNextRun(first) error = %v", err)
		}
		if claimA == nil {
			t.Fatal("ClaimNextRun(first) = nil, want claimed run")
		}
		if got, want := claimA.Run.ID, execution.Run.ID; got != want {
			t.Fatalf("claimA.Run.ID = %q, want %q", got, want)
		}
		if got, want := claimA.Run.Status, taskpkg.TaskRunStatusClaimed; got != want {
			t.Fatalf("claimA.Run.Status = %q, want %q", got, want)
		}
		if got, want := claimA.Run.SessionID, "sess-history-release-a"; got != want {
			t.Fatalf("claimA.Run.SessionID = %q, want %q", got, want)
		}
		if got, want := claimA.Run.NetworkChannel, "scope-direct-history"; got != want {
			t.Fatalf("claimA.Run.NetworkChannel = %q, want %q", got, want)
		}
		if got, want := claimA.Run.CoordinationChannelID, "scope-direct-history"; got != want {
			t.Fatalf("claimA.Run.CoordinationChannelID = %q, want %q", got, want)
		}
		if claimA.CoordinationChannel == nil || claimA.CoordinationChannel.ID != "scope-direct-history" {
			t.Fatalf("claimA.CoordinationChannel = %#v, want scope-direct-history", claimA.CoordinationChannel)
		}
		if claimA.ClaimToken == "" {
			t.Fatal("claimA.ClaimToken = empty, want raw token for release")
		}

		released, err := manager.ReleaseRunLease(ctx, taskpkg.LeaseRelease{
			RunID:      claimA.Run.ID,
			ClaimToken: claimA.ClaimToken,
			Reason:     "historical-reclaim-proof",
		}, agentA)
		if err != nil {
			t.Fatalf("ReleaseRunLease() error = %v", err)
		}
		if got, want := released.Status, taskpkg.TaskRunStatusQueued; got != want {
			t.Fatalf("released.Status = %q, want %q", got, want)
		}
		if got := released.SessionID; got != "" {
			t.Fatalf("released.SessionID = %q, want empty", got)
		}
		if got := released.ClaimTokenHash; got != "" {
			t.Fatalf("released.ClaimTokenHash = %q, want empty", got)
		}
		if got, want := released.NetworkChannel, "scope-direct-history"; got != want {
			t.Fatalf("released.NetworkChannel = %q, want %q", got, want)
		}
		if got, want := released.CoordinationChannelID, "scope-direct-history"; got != want {
			t.Fatalf("released.CoordinationChannelID = %q, want %q", got, want)
		}

		if _, err := manager.HeartbeatRunLease(ctx, taskpkg.LeaseHeartbeat{
			RunID:         claimA.Run.ID,
			ClaimToken:    claimA.ClaimToken,
			LeaseDuration: 2 * time.Minute,
		}, agentA); !errors.Is(err, taskpkg.ErrInvalidClaimToken) {
			t.Fatalf("HeartbeatRunLease(after release) error = %v, want %v", err, taskpkg.ErrInvalidClaimToken)
		}

		agentB, err := taskpkg.DeriveAgentSessionActorContext("sess-history-release-b")
		if err != nil {
			t.Fatalf("DeriveAgentSessionActorContext(agentB) error = %v", err)
		}
		claimB, err := manager.ClaimNextRun(ctx, taskpkg.ClaimCriteria{
			Scope:                 taskpkg.ScopeWorkspace,
			WorkspaceID:           workspaceID,
			ClaimerSessionID:      "sess-history-release-b",
			CoordinationChannelID: "scope-direct-history",
			LeaseDuration:         2 * time.Minute,
		}, agentB)
		if err != nil {
			t.Fatalf("ClaimNextRun(second) error = %v", err)
		}
		if claimB == nil {
			t.Fatal("ClaimNextRun(second) = nil, want reclaimed run")
		}
		if got, want := claimB.Run.ID, execution.Run.ID; got != want {
			t.Fatalf("claimB.Run.ID = %q, want %q", got, want)
		}
		if got, want := claimB.Run.Status, taskpkg.TaskRunStatusClaimed; got != want {
			t.Fatalf("claimB.Run.Status = %q, want %q", got, want)
		}
		if got, want := claimB.Run.SessionID, "sess-history-release-b"; got != want {
			t.Fatalf("claimB.Run.SessionID = %q, want %q", got, want)
		}
		if got, want := claimB.Run.NetworkChannel, "scope-direct-history"; got != want {
			t.Fatalf("claimB.Run.NetworkChannel = %q, want %q", got, want)
		}
		if got, want := claimB.Run.CoordinationChannelID, "scope-direct-history"; got != want {
			t.Fatalf("claimB.Run.CoordinationChannelID = %q, want %q", got, want)
		}
		if claimB.CoordinationChannel == nil || claimB.CoordinationChannel.ID != "scope-direct-history" {
			t.Fatalf("claimB.CoordinationChannel = %#v, want scope-direct-history", claimB.CoordinationChannel)
		}
		if claimB.ClaimToken == "" {
			t.Fatal("claimB.ClaimToken = empty, want raw token for completion")
		}
		if got, want := claimB.ClaimToken == claimA.ClaimToken, false; got != want {
			t.Fatalf("claimB reused first raw claim token = %t, want %t", got, want)
		}

		completed, err := manager.CompleteRunLease(ctx, taskpkg.LeaseCompletion{
			RunID:      claimB.Run.ID,
			ClaimToken: claimB.ClaimToken,
			Result: taskpkg.RunResult{
				Value: json.RawMessage(`{"ok":true,"path":"historical-channel-release-reclaim"}`),
			},
		}, agentB)
		if err != nil {
			t.Fatalf("CompleteRunLease() error = %v", err)
		}
		if got, want := completed.Status, taskpkg.TaskRunStatusCompleted; got != want {
			t.Fatalf("completed.Status = %q, want %q", got, want)
		}
		if got, want := completed.SessionID, "sess-history-release-b"; got != want {
			t.Fatalf("completed.SessionID = %q, want %q", got, want)
		}
		if got, want := string(completed.Result), `{"ok":true,"path":"historical-channel-release-reclaim"}`; got != want {
			t.Fatalf("completed.Result = %s, want %s", got, want)
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

		storedRun, err := db.GetTaskRun(ctx, claimB.Run.ID)
		if err != nil {
			t.Fatalf("GetTaskRun() error = %v", err)
		}
		if got, want := storedRun.Status, taskpkg.TaskRunStatusCompleted; got != want {
			t.Fatalf("storedRun.Status = %q, want %q", got, want)
		}
		if got, want := storedRun.SessionID, "sess-history-release-b"; got != want {
			t.Fatalf("storedRun.SessionID = %q, want %q", got, want)
		}
		if got, want := storedRun.NetworkChannel, "scope-direct-history"; got != want {
			t.Fatalf("storedRun.NetworkChannel = %q, want %q", got, want)
		}
		if got, want := storedRun.CoordinationChannelID, "scope-direct-history"; got != want {
			t.Fatalf("storedRun.CoordinationChannelID = %q, want %q", got, want)
		}
		if got, want := string(storedRun.Result), `{"ok":true,"path":"historical-channel-release-reclaim"}`; got != want {
			t.Fatalf("storedRun.Result = %s, want %s", got, want)
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
			t.Fatalf("eventCounts[task.run_claimed] = %d, want %d (events=%#v)", got, want, sortedEventTypes(events))
		}
		if got, want := eventCounts["task.run_released"], 1; got != want {
			t.Fatalf("eventCounts[task.run_released] = %d, want %d (events=%#v)", got, want, sortedEventTypes(events))
		}
		if got, want := eventCounts["task.run_completed"], 1; got != want {
			t.Fatalf("eventCounts[task.run_completed] = %d, want %d (events=%#v)", got, want, sortedEventTypes(events))
		}
	})

	t.Run("Should preserve historical network channels through lease failure retry and completion", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		db := openTaskManagerGlobalDB(t)
		workspaceID := registerTaskManagerWorkspace(t, db, "historical-channel-fail-retry", filepath.Join(t.TempDir(), "workspace"))
		manager := newTaskManagerIntegration(
			t,
			db,
			taskpkg.WithNetworkChannelValidator(network.ValidateChannel),
		)

		channelTimestamp := time.Date(2026, 4, 28, 6, 36, 14, 0, time.UTC)
		if err := db.WriteNetworkChannel(ctx, store.NetworkChannelEntry{
			Channel:     "scope-direct-history",
			WorkspaceID: workspaceID,
			Purpose:     "Historical task-run fail retry validation",
			CreatedBy:   "founder",
			CreatedAt:   channelTimestamp,
			UpdatedAt:   channelTimestamp,
		}); err != nil {
			t.Fatalf("WriteNetworkChannel() error = %v", err)
		}

		operator, err := taskpkg.DeriveHumanActorContext("user-1", taskpkg.OriginKindCLI, "agh task fail")
		if err != nil {
			t.Fatalf("DeriveHumanActorContext() error = %v", err)
		}

		taskRecord, err := manager.CreateTask(ctx, taskpkg.CreateTask{
			Scope:          taskpkg.ScopeWorkspace,
			WorkspaceID:    workspaceID,
			NetworkChannel: "scope-direct-history",
			Title:          "History-only channel fail retry repro",
		}, operator)
		if err != nil {
			t.Fatalf("CreateTask() error = %v", err)
		}

		firstExecution, err := manager.StartTask(ctx, taskRecord.ID, taskpkg.ExecutionRequest{}, operator)
		if err != nil {
			t.Fatalf("StartTask() error = %v", err)
		}
		if got, want := firstExecution.Run.Status, taskpkg.TaskRunStatusQueued; got != want {
			t.Fatalf("firstExecution.Run.Status = %q, want %q", got, want)
		}
		if got, want := firstExecution.Run.Attempt, 1; got != want {
			t.Fatalf("firstExecution.Run.Attempt = %d, want %d", got, want)
		}
		if got, want := firstExecution.Run.NetworkChannel, "scope-direct-history"; got != want {
			t.Fatalf("firstExecution.Run.NetworkChannel = %q, want %q", got, want)
		}
		if got, want := firstExecution.Run.CoordinationChannelID, "scope-direct-history"; got != want {
			t.Fatalf("firstExecution.Run.CoordinationChannelID = %q, want %q", got, want)
		}

		agentA, err := taskpkg.DeriveAgentSessionActorContext("sess-history-fail-a")
		if err != nil {
			t.Fatalf("DeriveAgentSessionActorContext(agentA) error = %v", err)
		}
		claimA, err := manager.ClaimNextRun(ctx, taskpkg.ClaimCriteria{
			Scope:                 taskpkg.ScopeWorkspace,
			WorkspaceID:           workspaceID,
			ClaimerSessionID:      "sess-history-fail-a",
			CoordinationChannelID: "scope-direct-history",
			LeaseDuration:         time.Minute,
		}, agentA)
		if err != nil {
			t.Fatalf("ClaimNextRun(first) error = %v", err)
		}
		if claimA == nil {
			t.Fatal("ClaimNextRun(first) = nil, want claimed run")
		}
		if got, want := claimA.Run.ID, firstExecution.Run.ID; got != want {
			t.Fatalf("claimA.Run.ID = %q, want %q", got, want)
		}
		if got, want := claimA.Run.Attempt, 1; got != want {
			t.Fatalf("claimA.Run.Attempt = %d, want %d", got, want)
		}
		if got, want := claimA.Run.NetworkChannel, "scope-direct-history"; got != want {
			t.Fatalf("claimA.Run.NetworkChannel = %q, want %q", got, want)
		}
		if got, want := claimA.Run.CoordinationChannelID, "scope-direct-history"; got != want {
			t.Fatalf("claimA.Run.CoordinationChannelID = %q, want %q", got, want)
		}
		if claimA.CoordinationChannel == nil || claimA.CoordinationChannel.ID != "scope-direct-history" {
			t.Fatalf("claimA.CoordinationChannel = %#v, want scope-direct-history", claimA.CoordinationChannel)
		}
		if claimA.ClaimToken == "" {
			t.Fatal("claimA.ClaimToken = empty, want raw token for failure")
		}

		failed, err := manager.FailRunLease(ctx, taskpkg.LeaseFailure{
			RunID:      claimA.Run.ID,
			ClaimToken: claimA.ClaimToken,
			Failure: taskpkg.RunFailure{
				Error:    "historical retry probe failed",
				Metadata: json.RawMessage(`{"phase":"attempt-1"}`),
			},
		}, agentA)
		if err != nil {
			t.Fatalf("FailRunLease() error = %v", err)
		}
		if got, want := failed.Status, taskpkg.TaskRunStatusFailed; got != want {
			t.Fatalf("failed.Status = %q, want %q", got, want)
		}
		if got, want := failed.Attempt, 1; got != want {
			t.Fatalf("failed.Attempt = %d, want %d", got, want)
		}
		if got, want := failed.Error, "historical retry probe failed"; got != want {
			t.Fatalf("failed.Error = %q, want %q", got, want)
		}
		if got, want := failed.NetworkChannel, "scope-direct-history"; got != want {
			t.Fatalf("failed.NetworkChannel = %q, want %q", got, want)
		}
		if got, want := failed.CoordinationChannelID, "scope-direct-history"; got != want {
			t.Fatalf("failed.CoordinationChannelID = %q, want %q", got, want)
		}

		if _, err := manager.CompleteRunLease(ctx, taskpkg.LeaseCompletion{
			RunID:      claimA.Run.ID,
			ClaimToken: claimA.ClaimToken,
			Result: taskpkg.RunResult{
				Value: json.RawMessage(`{"summary":"should fail"}`),
			},
		}, agentA); !errors.Is(err, taskpkg.ErrInvalidStatusTransition) {
			t.Fatalf("CompleteRunLease(after fail) error = %v, want %v", err, taskpkg.ErrInvalidStatusTransition)
		}

		retryRun, err := manager.EnqueueRun(ctx, taskpkg.EnqueueRun{
			TaskID:         taskRecord.ID,
			IdempotencyKey: "historical-fail-retry-attempt-2",
		}, operator)
		if err != nil {
			t.Fatalf("EnqueueRun(retry) error = %v", err)
		}
		if got, want := retryRun.Attempt, 2; got != want {
			t.Fatalf("retryRun.Attempt = %d, want %d", got, want)
		}
		if got, want := retryRun.Status, taskpkg.TaskRunStatusQueued; got != want {
			t.Fatalf("retryRun.Status = %q, want %q", got, want)
		}
		if got, want := retryRun.NetworkChannel, "scope-direct-history"; got != want {
			t.Fatalf("retryRun.NetworkChannel = %q, want %q", got, want)
		}
		if got, want := retryRun.CoordinationChannelID, "scope-direct-history"; got != want {
			t.Fatalf("retryRun.CoordinationChannelID = %q, want %q", got, want)
		}

		agentB, err := taskpkg.DeriveAgentSessionActorContext("sess-history-fail-b")
		if err != nil {
			t.Fatalf("DeriveAgentSessionActorContext(agentB) error = %v", err)
		}
		claimB, err := manager.ClaimNextRun(ctx, taskpkg.ClaimCriteria{
			Scope:                 taskpkg.ScopeWorkspace,
			WorkspaceID:           workspaceID,
			ClaimerSessionID:      "sess-history-fail-b",
			CoordinationChannelID: "scope-direct-history",
			LeaseDuration:         2 * time.Minute,
		}, agentB)
		if err != nil {
			t.Fatalf("ClaimNextRun(second) error = %v", err)
		}
		if claimB == nil {
			t.Fatal("ClaimNextRun(second) = nil, want retry run")
		}
		if got, want := claimB.Run.ID, retryRun.ID; got != want {
			t.Fatalf("claimB.Run.ID = %q, want %q", got, want)
		}
		if got, want := claimB.Run.Attempt, 2; got != want {
			t.Fatalf("claimB.Run.Attempt = %d, want %d", got, want)
		}
		if got, want := claimB.Run.NetworkChannel, "scope-direct-history"; got != want {
			t.Fatalf("claimB.Run.NetworkChannel = %q, want %q", got, want)
		}
		if got, want := claimB.Run.CoordinationChannelID, "scope-direct-history"; got != want {
			t.Fatalf("claimB.Run.CoordinationChannelID = %q, want %q", got, want)
		}
		if claimB.CoordinationChannel == nil || claimB.CoordinationChannel.ID != "scope-direct-history" {
			t.Fatalf("claimB.CoordinationChannel = %#v, want scope-direct-history", claimB.CoordinationChannel)
		}
		if claimB.ClaimToken == "" {
			t.Fatal("claimB.ClaimToken = empty, want raw token for retry completion")
		}

		completed, err := manager.CompleteRunLease(ctx, taskpkg.LeaseCompletion{
			RunID:      claimB.Run.ID,
			ClaimToken: claimB.ClaimToken,
			Result: taskpkg.RunResult{
				Value: json.RawMessage(`{"ok":true,"path":"historical-channel-fail-retry"}`),
			},
		}, agentB)
		if err != nil {
			t.Fatalf("CompleteRunLease(retry) error = %v", err)
		}
		if got, want := completed.Status, taskpkg.TaskRunStatusCompleted; got != want {
			t.Fatalf("completed.Status = %q, want %q", got, want)
		}
		if got, want := completed.Attempt, 2; got != want {
			t.Fatalf("completed.Attempt = %d, want %d", got, want)
		}
		if got, want := completed.SessionID, "sess-history-fail-b"; got != want {
			t.Fatalf("completed.SessionID = %q, want %q", got, want)
		}
		if got, want := completed.NetworkChannel, "scope-direct-history"; got != want {
			t.Fatalf("completed.NetworkChannel = %q, want %q", got, want)
		}
		if got, want := completed.CoordinationChannelID, "scope-direct-history"; got != want {
			t.Fatalf("completed.CoordinationChannelID = %q, want %q", got, want)
		}
		if got, want := string(completed.Result), `{"ok":true,"path":"historical-channel-fail-retry"}`; got != want {
			t.Fatalf("completed.Result = %s, want %s", got, want)
		}

		storedTask, err := db.GetTask(ctx, taskRecord.ID)
		if err != nil {
			t.Fatalf("GetTask() error = %v", err)
		}
		if got, want := storedTask.Status, taskpkg.TaskStatusCompleted; got != want {
			t.Fatalf("storedTask.Status = %q, want %q", got, want)
		}

		storedRuns, err := db.ListTaskRuns(ctx, taskpkg.RunQuery{TaskID: taskRecord.ID})
		if err != nil {
			t.Fatalf("ListTaskRuns() error = %v", err)
		}
		if got, want := len(storedRuns), 2; got != want {
			t.Fatalf("len(storedRuns) = %d, want %d", got, want)
		}
		runsByAttempt := map[int]taskpkg.Run{}
		for _, run := range storedRuns {
			runsByAttempt[run.Attempt] = run
		}
		firstStored, ok := runsByAttempt[1]
		if !ok {
			t.Fatalf("runsByAttempt missing attempt 1: %#v", runsByAttempt)
		}
		secondStored, ok := runsByAttempt[2]
		if !ok {
			t.Fatalf("runsByAttempt missing attempt 2: %#v", runsByAttempt)
		}
		if got, want := firstStored.Status, taskpkg.TaskRunStatusFailed; got != want {
			t.Fatalf("firstStored.Status = %q, want %q", got, want)
		}
		if got, want := firstStored.NetworkChannel, "scope-direct-history"; got != want {
			t.Fatalf("firstStored.NetworkChannel = %q, want %q", got, want)
		}
		if got, want := firstStored.CoordinationChannelID, "scope-direct-history"; got != want {
			t.Fatalf("firstStored.CoordinationChannelID = %q, want %q", got, want)
		}
		if got, want := secondStored.Status, taskpkg.TaskRunStatusCompleted; got != want {
			t.Fatalf("secondStored.Status = %q, want %q", got, want)
		}
		if got, want := secondStored.NetworkChannel, "scope-direct-history"; got != want {
			t.Fatalf("secondStored.NetworkChannel = %q, want %q", got, want)
		}
		if got, want := secondStored.CoordinationChannelID, "scope-direct-history"; got != want {
			t.Fatalf("secondStored.CoordinationChannelID = %q, want %q", got, want)
		}

		events, err := db.ListTaskEvents(ctx, taskpkg.EventQuery{TaskID: taskRecord.ID})
		if err != nil {
			t.Fatalf("ListTaskEvents() error = %v", err)
		}
		eventCounts := map[string]int{}
		for _, event := range events {
			eventCounts[event.EventType]++
		}
		if got, want := eventCounts["task.run_enqueued"], 2; got != want {
			t.Fatalf("eventCounts[task.run_enqueued] = %d, want %d (events=%#v)", got, want, sortedEventTypes(events))
		}
		if got, want := eventCounts["task.run_claimed"], 2; got != want {
			t.Fatalf("eventCounts[task.run_claimed] = %d, want %d (events=%#v)", got, want, sortedEventTypes(events))
		}
		if got, want := eventCounts["task.run_failed"], 1; got != want {
			t.Fatalf("eventCounts[task.run_failed] = %d, want %d (events=%#v)", got, want, sortedEventTypes(events))
		}
		if got, want := eventCounts["task.run_completed"], 1; got != want {
			t.Fatalf("eventCounts[task.run_completed] = %d, want %d (events=%#v)", got, want, sortedEventTypes(events))
		}
	})
}
