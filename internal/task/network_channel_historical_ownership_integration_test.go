//go:build integration

package task_test

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/compozy/agh/internal/network"
	"github.com/compozy/agh/internal/store"
	taskpkg "github.com/compozy/agh/internal/task"
	"github.com/compozy/agh/internal/testutil"
)

func TestTaskManagerHistoricalNetworkChannelOwnershipIntegration(t *testing.T) {
	t.Parallel()

	t.Run(
		"Should reject human completion and failure for token-fenced historical runs while allowing operator cancel",
		func(t *testing.T) {
			t.Parallel()

			ctx := testutil.Context(t)
			db := openTaskManagerGlobalDB(t)
			workspaceID := registerTaskManagerWorkspace(
				t,
				db,
				"historical-channel-ownership",
				filepath.Join(t.TempDir(), "workspace"),
			)
			manager := newTaskManagerIntegration(
				t,
				db,
				taskpkg.WithNetworkChannelValidator(network.ValidateChannel),
			)

			channelTimestamp := time.Date(2026, 4, 28, 8, 29, 19, 0, time.UTC)
			if err := db.WriteNetworkChannel(ctx, store.NetworkChannelEntry{
				Channel:     "scope-direct-history",
				WorkspaceID: workspaceID,
				Purpose:     "Historical token-fence ownership validation",
				CreatedBy:   "founder",
				CreatedAt:   channelTimestamp,
				UpdatedAt:   channelTimestamp,
			}); err != nil {
				t.Fatalf("WriteNetworkChannel() error = %v", err)
			}

			operator, err := taskpkg.DeriveHumanActorContext("user-1", taskpkg.OriginKindCLI, "agh task cancel")
			if err != nil {
				t.Fatalf("DeriveHumanActorContext() error = %v", err)
			}
			agent, err := taskpkg.DeriveAgentSessionActorContext("sess-history-mixed")
			if err != nil {
				t.Fatalf("DeriveAgentSessionActorContext() error = %v", err)
			}

			taskRecord, err := manager.CreateTask(ctx, taskpkg.CreateTask{
				Scope:          taskpkg.ScopeWorkspace,
				WorkspaceID:    workspaceID,
				NetworkChannel: "scope-direct-history",
				Title:          "Historical mixed-ownership token fence",
			}, operator)
			if err != nil {
				t.Fatalf("CreateTask() error = %v", err)
			}

			queuedRun, err := manager.EnqueueRun(ctx, taskpkg.EnqueueRun{
				TaskID:         taskRecord.ID,
				NetworkChannel: "scope-direct-history",
			}, operator)
			if err != nil {
				t.Fatalf("EnqueueRun() error = %v", err)
			}
			if got, want := queuedRun.NetworkChannel, "scope-direct-history"; got != want {
				t.Fatalf("queuedRun.NetworkChannel = %q, want %q", got, want)
			}
			if got, want := queuedRun.CoordinationChannelID, "scope-direct-history"; got != want {
				t.Fatalf("queuedRun.CoordinationChannelID = %q, want %q", got, want)
			}

			claim, err := manager.ClaimNextRun(ctx, taskpkg.ClaimCriteria{
				WorkspaceID:           workspaceID,
				ClaimerSessionID:      "sess-history-mixed",
				CoordinationChannelID: "scope-direct-history",
			}, agent)
			if err != nil {
				t.Fatalf("ClaimNextRun() error = %v", err)
			}
			if claim == nil {
				t.Fatal("ClaimNextRun() = nil, want claimed run")
			}
			if got, want := claim.Run.ID, queuedRun.ID; got != want {
				t.Fatalf("claim.Run.ID = %q, want %q", got, want)
			}
			if claim.ClaimToken == "" {
				t.Fatal("claim.ClaimToken = empty, want raw token")
			}
			if got, want := claim.Run.SessionID, "sess-history-mixed"; got != want {
				t.Fatalf("claim.Run.SessionID = %q, want %q", got, want)
			}
			if got, want := claim.Run.NetworkChannel, "scope-direct-history"; got != want {
				t.Fatalf("claim.Run.NetworkChannel = %q, want %q", got, want)
			}
			if got, want := claim.Run.CoordinationChannelID, "scope-direct-history"; got != want {
				t.Fatalf("claim.Run.CoordinationChannelID = %q, want %q", got, want)
			}

			_, err = manager.CompleteRun(ctx, claim.Run.ID, taskpkg.RunResult{
				Value: json.RawMessage(`{"ok":true,"mode":"mixed-token-fence"}`),
			}, operator)
			if !errors.Is(err, taskpkg.ErrInvalidClaimToken) {
				t.Fatalf("CompleteRun(token-fenced) error = %v, want %v", err, taskpkg.ErrInvalidClaimToken)
			}

			_, err = manager.FailRun(ctx, claim.Run.ID, taskpkg.RunFailure{
				Error:    "boom",
				Metadata: json.RawMessage(`{"mode":"mixed-token-fence"}`),
			}, operator)
			if !errors.Is(err, taskpkg.ErrInvalidClaimToken) {
				t.Fatalf("FailRun(token-fenced) error = %v, want %v", err, taskpkg.ErrInvalidClaimToken)
			}

			canceled, err := manager.CancelRun(ctx, claim.Run.ID, taskpkg.CancelRun{
				Reason:   "operator override",
				Metadata: json.RawMessage(`{"mode":"mixed-token-fence"}`),
			}, operator)
			if err != nil {
				t.Fatalf("CancelRun() error = %v", err)
			}
			if got, want := canceled.Status, taskpkg.TaskRunStatusCanceled; got != want {
				t.Fatalf("canceled.Status = %q, want %q", got, want)
			}
			if got, want := canceled.SessionID, "sess-history-mixed"; got != want {
				t.Fatalf("canceled.SessionID = %q, want %q", got, want)
			}
			if canceled.ClaimedBy == nil || canceled.ClaimedBy.Ref != "sess-history-mixed" {
				t.Fatalf("canceled.ClaimedBy = %#v, want sess-history-mixed", canceled.ClaimedBy)
			}
			if got, want := canceled.NetworkChannel, "scope-direct-history"; got != want {
				t.Fatalf("canceled.NetworkChannel = %q, want %q", got, want)
			}
			if got, want := canceled.CoordinationChannelID, "scope-direct-history"; got != want {
				t.Fatalf("canceled.CoordinationChannelID = %q, want %q", got, want)
			}

			_, err = manager.CompleteRunLease(ctx, taskpkg.LeaseCompletion{
				RunID:      claim.Run.ID,
				ClaimToken: claim.ClaimToken,
				Result: taskpkg.RunResult{
					Value: json.RawMessage(`{"ok":true,"mode":"after-cancel"}`),
				},
			}, agent)
			if !errors.Is(err, taskpkg.ErrInvalidStatusTransition) {
				t.Fatalf("CompleteRunLease(after cancel) error = %v, want %v", err, taskpkg.ErrInvalidStatusTransition)
			}

			storedTask, err := db.GetTask(ctx, taskRecord.ID)
			if err != nil {
				t.Fatalf("GetTask() error = %v", err)
			}
			if got, want := storedTask.Status, taskpkg.TaskStatusCanceled; got != want {
				t.Fatalf("storedTask.Status = %q, want %q", got, want)
			}

			storedRun, err := db.GetTaskRun(ctx, claim.Run.ID)
			if err != nil {
				t.Fatalf("GetTaskRun() error = %v", err)
			}
			if got, want := storedRun.Status, taskpkg.TaskRunStatusCanceled; got != want {
				t.Fatalf("storedRun.Status = %q, want %q", got, want)
			}
			if got, want := storedRun.NetworkChannel, "scope-direct-history"; got != want {
				t.Fatalf("storedRun.NetworkChannel = %q, want %q", got, want)
			}
			if got, want := storedRun.CoordinationChannelID, "scope-direct-history"; got != want {
				t.Fatalf("storedRun.CoordinationChannelID = %q, want %q", got, want)
			}

			events, err := db.ListTaskEvents(ctx, taskpkg.EventQuery{TaskID: taskRecord.ID})
			if err != nil {
				t.Fatalf("ListTaskEvents() error = %v", err)
			}
			eventCounts := map[string]int{}
			for _, event := range events {
				eventCounts[event.EventType]++
			}
			if got, want := eventCounts["task.run_claimed"], 1; got != want {
				t.Fatalf(
					"eventCounts[task.run_claimed] = %d, want %d (events=%#v)",
					got,
					want,
					sortedEventTypes(events),
				)
			}
			if got, want := eventCounts["task.run_canceled"], 1; got != want {
				t.Fatalf(
					"eventCounts[task.run_canceled] = %d, want %d (events=%#v)",
					got,
					want,
					sortedEventTypes(events),
				)
			}
			if got := eventCounts["task.run_completed"]; got != 0 {
				t.Fatalf("eventCounts[task.run_completed] = %d, want 0 (events=%#v)", got, sortedEventTypes(events))
			}
			if got := eventCounts["task.run_failed"]; got != 0 {
				t.Fatalf("eventCounts[task.run_failed] = %d, want 0 (events=%#v)", got, sortedEventTypes(events))
			}
		},
	)
}
