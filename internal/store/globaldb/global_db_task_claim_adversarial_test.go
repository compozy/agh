package globaldb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	taskpkg "github.com/pedronauck/agh/internal/task"
	"github.com/pedronauck/agh/internal/testutil"
)

type transitionResult struct {
	name string
	run  taskpkg.Run
	err  error
}

func TestGlobalDBTaskRunLeaseAdversarialFencing(t *testing.T) {
	t.Parallel()

	t.Run("Should claim each queued run exactly once under concurrent workers", func(t *testing.T) {
		t.Parallel()

		globalDB := openTestGlobalDB(t)
		ctx := testutil.Context(t)
		now := time.Date(2026, 5, 19, 10, 0, 0, 0, time.UTC)
		runs := seedAdversarialClaimRunsAcrossTasks(ctx, t, globalDB, "concurrent", 3, now)

		type claimAttempt struct {
			result taskpkg.ClaimResult
			err    error
		}
		attempts := make([]claimAttempt, 12)
		start := make(chan struct{})
		var wg sync.WaitGroup
		wg.Add(len(attempts))
		for idx := range attempts {
			go func(idx int) {
				defer wg.Done()
				<-start
				attempts[idx].result, attempts[idx].err = globalDB.ClaimNextRun(
					ctx,
					taskpkg.ClaimCriteria{
						Scope:            taskpkg.ScopeGlobal,
						ClaimerSessionID: fmt.Sprintf("sess-concurrent-%02d", idx),
						LeaseDuration:    time.Minute,
						Now:              now.Add(time.Duration(idx) * time.Millisecond),
					},
				)
			}(idx)
		}
		close(start)
		wg.Wait()

		claimedRunIDs := map[string]int{}
		for idx, attempt := range attempts {
			if attempt.err != nil {
				if !errors.Is(attempt.err, taskpkg.ErrNoClaimableRun) {
					t.Fatalf("attempt %d error = %v, want ErrNoClaimableRun", idx, attempt.err)
				}
				continue
			}
			if attempt.result.ClaimToken == "" {
				t.Fatalf("attempt %d claim token = empty, want raw token returned once", idx)
			}
			if !taskpkg.VerifyClaimToken(attempt.result.ClaimToken, attempt.result.Run.ClaimTokenHash) {
				t.Fatalf("attempt %d claim token does not verify against persisted hash", idx)
			}
			claimedRunIDs[attempt.result.Run.ID]++
		}
		if got, want := len(claimedRunIDs), len(runs); got != want {
			t.Fatalf("claimed unique runs = %d, want %d (attempts=%#v)", got, want, attempts)
		}
		for _, run := range runs {
			if got := claimedRunIDs[run.ID]; got != 1 {
				t.Fatalf("claimedRunIDs[%s] = %d, want 1", run.ID, got)
			}
			stored, err := globalDB.GetTaskRun(ctx, run.ID)
			if err != nil {
				t.Fatalf("GetTaskRun(%q) error = %v", run.ID, err)
			}
			if stored.Status != taskpkg.TaskRunStatusClaimed ||
				stored.SessionID == "" ||
				stored.ClaimTokenHash == "" {
				t.Fatalf("stored run %q = %#v, want claimed with owner and token hash", run.ID, stored)
			}
		}
	})

	t.Run("Should allow only one release complete or fail transition for one active lease", func(t *testing.T) {
		t.Parallel()

		globalDB := openTestGlobalDB(t)
		ctx := testutil.Context(t)
		now := time.Date(2026, 5, 19, 10, 30, 0, 0, time.UTC)
		runs := seedAdversarialClaimRuns(ctx, t, globalDB, "transition-race", 1, now)
		claim, err := globalDB.ClaimNextRun(ctx, taskpkg.ClaimCriteria{
			Scope:            taskpkg.ScopeGlobal,
			ClaimerSessionID: "sess-transition-race",
			LeaseDuration:    time.Minute,
			Now:              now,
		})
		if err != nil {
			t.Fatalf("ClaimNextRun() error = %v", err)
		}
		if got, want := claim.Run.ID, runs[0].ID; got != want {
			t.Fatalf("ClaimNextRun().Run.ID = %q, want %q", got, want)
		}

		assertLeaseRejectsWrongTokens(ctx, t, globalDB, &claim, now.Add(10*time.Second))

		type transitionAttempt struct {
			name string
			run  func() (taskpkg.Run, error)
		}
		attempts := []transitionAttempt{
			{
				name: "release",
				run: func() (taskpkg.Run, error) {
					return globalDB.ReleaseRunLease(ctx, taskpkg.LeaseRelease{
						RunID:      claim.Run.ID,
						ClaimToken: claim.ClaimToken,
						Reason:     "handoff-race",
						Now:        now.Add(20 * time.Second),
					})
				},
			},
			{
				name: "complete",
				run: func() (taskpkg.Run, error) {
					return globalDB.CompleteRunLease(ctx, taskpkg.LeaseCompletion{
						RunID:      claim.Run.ID,
						ClaimToken: claim.ClaimToken,
						Result:     taskpkg.RunResult{Value: json.RawMessage(`{"ok":true}`)},
						Now:        now.Add(20 * time.Second),
					})
				},
			},
			{
				name: "fail",
				run: func() (taskpkg.Run, error) {
					return globalDB.FailRunLease(ctx, taskpkg.LeaseFailure{
						RunID:      claim.Run.ID,
						ClaimToken: claim.ClaimToken,
						Failure:    taskpkg.RunFailure{Error: "worker failed during race"},
						Now:        now.Add(20 * time.Second),
					})
				},
			},
		}

		results := make([]transitionResult, len(attempts))
		start := make(chan struct{})
		var wg sync.WaitGroup
		wg.Add(len(attempts))
		for idx := range attempts {
			go func(idx int) {
				defer wg.Done()
				<-start
				updated, err := attempts[idx].run()
				results[idx] = transitionResult{name: attempts[idx].name, run: updated, err: err}
			}(idx)
		}
		close(start)
		wg.Wait()

		successes := make([]transitionResult, 0, 1)
		for _, result := range results {
			if result.err == nil {
				successes = append(successes, result)
				continue
			}
			if !isExpectedLeaseRaceError(result.err) {
				t.Fatalf("%s error = %v, want invalid token or status transition", result.name, result.err)
			}
		}
		if got, want := len(successes), 1; got != want {
			t.Fatalf("successful transitions = %d, want %d (results=%#v)", got, want, results)
		}
		assertLeaseRaceWinner(ctx, t, globalDB, claim.Run.TaskID, successes[0])
		if _, err := globalDB.HeartbeatRunLease(ctx, taskpkg.LeaseHeartbeat{
			RunID:         claim.Run.ID,
			ClaimToken:    claim.ClaimToken,
			LeaseDuration: time.Minute,
			Now:           now.Add(30 * time.Second),
		}); !isExpectedLeaseRaceError(err) {
			t.Fatalf("HeartbeatRunLease(after race winner) error = %v, want invalid token or status transition", err)
		}
	})
}

func seedAdversarialClaimRuns(
	ctx context.Context,
	t *testing.T,
	globalDB *GlobalDB,
	suffix string,
	runCount int,
	now time.Time,
) []taskpkg.Run {
	t.Helper()

	taskRecord := taskRecordForTest("task-adversarial-" + suffix)
	taskRecord.Status = taskpkg.TaskStatusReady
	if err := globalDB.CreateTask(ctx, taskRecord); err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	runs := make([]taskpkg.Run, 0, runCount)
	for idx := range runCount {
		run := taskRunForTest(fmt.Sprintf("run-adversarial-%s-%02d", suffix, idx), taskRecord.ID)
		run.QueuedAt = now.Add(time.Duration(idx) * time.Second)
		if err := globalDB.CreateTaskRun(ctx, run); err != nil {
			t.Fatalf("CreateTaskRun(%q) error = %v", run.ID, err)
		}
		runs = append(runs, run)
	}
	return runs
}

func seedAdversarialClaimRunsAcrossTasks(
	ctx context.Context,
	t *testing.T,
	globalDB *GlobalDB,
	suffix string,
	runCount int,
	now time.Time,
) []taskpkg.Run {
	t.Helper()

	runs := make([]taskpkg.Run, 0, runCount)
	for idx := range runCount {
		taskRecord := taskRecordForTest(fmt.Sprintf("task-adversarial-%s-%02d", suffix, idx))
		taskRecord.Status = taskpkg.TaskStatusReady
		if err := globalDB.CreateTask(ctx, taskRecord); err != nil {
			t.Fatalf("CreateTask(%q) error = %v", taskRecord.ID, err)
		}
		run := taskRunForTest(fmt.Sprintf("run-adversarial-%s-%02d", suffix, idx), taskRecord.ID)
		run.QueuedAt = now.Add(time.Duration(idx) * time.Second)
		if err := globalDB.CreateTaskRun(ctx, run); err != nil {
			t.Fatalf("CreateTaskRun(%q) error = %v", run.ID, err)
		}
		runs = append(runs, run)
	}
	return runs
}

func assertLeaseRejectsWrongTokens(
	ctx context.Context,
	t *testing.T,
	globalDB *GlobalDB,
	claim *taskpkg.ClaimResult,
	now time.Time,
) {
	t.Helper()

	checks := []struct {
		name string
		run  func() error
	}{
		{
			name: "heartbeat",
			run: func() error {
				_, err := globalDB.HeartbeatRunLease(ctx, taskpkg.LeaseHeartbeat{
					RunID:         claim.Run.ID,
					ClaimToken:    "agh_claim_wrong",
					LeaseDuration: time.Minute,
					Now:           now,
				})
				return err
			},
		},
		{
			name: "release",
			run: func() error {
				_, err := globalDB.ReleaseRunLease(ctx, taskpkg.LeaseRelease{
					RunID:      claim.Run.ID,
					ClaimToken: "agh_claim_wrong",
					Reason:     "wrong-token",
					Now:        now,
				})
				return err
			},
		},
		{
			name: "complete",
			run: func() error {
				_, err := globalDB.CompleteRunLease(ctx, taskpkg.LeaseCompletion{
					RunID:      claim.Run.ID,
					ClaimToken: "agh_claim_wrong",
					Result:     taskpkg.RunResult{Value: json.RawMessage(`{"ok":false}`)},
					Now:        now,
				})
				return err
			},
		},
		{
			name: "fail",
			run: func() error {
				_, err := globalDB.FailRunLease(ctx, taskpkg.LeaseFailure{
					RunID:      claim.Run.ID,
					ClaimToken: "agh_claim_wrong",
					Failure:    taskpkg.RunFailure{Error: "wrong token should not fail run"},
					Now:        now,
				})
				return err
			},
		},
	}
	for _, check := range checks {
		if err := check.run(); !errors.Is(err, taskpkg.ErrInvalidClaimToken) {
			t.Fatalf("%s wrong-token error = %v, want %v", check.name, err, taskpkg.ErrInvalidClaimToken)
		}
	}

	stored, err := globalDB.GetTaskRun(ctx, claim.Run.ID)
	if err != nil {
		t.Fatalf("GetTaskRun(after wrong-token attempts) error = %v", err)
	}
	if stored.Status != taskpkg.TaskRunStatusClaimed ||
		stored.SessionID != claim.Run.SessionID ||
		stored.ClaimTokenHash != claim.Run.ClaimTokenHash {
		t.Fatalf("stored after wrong-token attempts = %#v, want original active lease", stored)
	}
}

func isExpectedLeaseRaceError(err error) bool {
	return errors.Is(err, taskpkg.ErrInvalidClaimToken) ||
		errors.Is(err, taskpkg.ErrInvalidStatusTransition)
}

func assertLeaseRaceWinner(
	ctx context.Context,
	t *testing.T,
	globalDB *GlobalDB,
	taskID string,
	winner transitionResult,
) {
	t.Helper()

	stored, err := globalDB.GetTaskRun(ctx, winner.run.ID)
	if err != nil {
		t.Fatalf("GetTaskRun(%q) error = %v", winner.run.ID, err)
	}
	switch winner.name {
	case "release":
		if stored.Status != taskpkg.TaskRunStatusQueued ||
			stored.SessionID != "" ||
			stored.ClaimTokenHash != "" {
			t.Fatalf("release winner stored run = %#v, want queued and unowned", stored)
		}
	case "complete":
		if stored.Status != taskpkg.TaskRunStatusCompleted || stored.Result == nil {
			t.Fatalf("complete winner stored run = %#v, want completed with result", stored)
		}
	case "fail":
		if stored.Status != taskpkg.TaskRunStatusFailed || stored.Error != "worker failed during race" {
			t.Fatalf("fail winner stored run = %#v, want failed with race error", stored)
		}
	default:
		t.Fatalf("unexpected race winner %q", winner.name)
	}
	taskRecord, err := globalDB.GetTask(ctx, taskID)
	if err != nil {
		t.Fatalf("GetTask(%q) error = %v", taskID, err)
	}
	if taskRecord.CurrentRunID != "" {
		t.Fatalf("task.CurrentRunID = %q, want cleared after %s", taskRecord.CurrentRunID, winner.name)
	}
}
