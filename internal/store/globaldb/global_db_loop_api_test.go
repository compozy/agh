package globaldb

import (
	"testing"
	"time"

	looppkg "github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/loop/dsl"
	"github.com/compozy/agh/internal/testutil"
)

func TestGlobalDBLoopAPIRunsShouldRemainWorkspaceScoped(t *testing.T) {
	t.Parallel()

	t.Run("Should list only the requested workspace runs with filters and aggregates inputs", func(t *testing.T) {
		t.Parallel()

		globalDB := openFreshLoopTestGlobalDB(t, "ws-a", "ws-b")
		ctx := testutil.Context(t)
		now := time.Date(2026, 7, 5, 10, 0, 0, 0, time.UTC)

		alpha := testLoopRun("looprun-api-alpha", now, looppkg.StatusRunning)
		alpha.WorkspaceID = "ws-a"
		alpha.LoopName = "delivery"
		alpha.Inputs = map[string]any{"ticket": "A"}
		if _, err := globalDB.CreateLoopRunForStart(ctx, alpha, dsl.ConcurrencyAllow); err != nil {
			t.Fatalf("CreateLoopRunForStart(alpha) error = %v", err)
		}
		alphaDone := testLoopRun("looprun-api-alpha-done", now.Add(-time.Minute), looppkg.StatusRunning)
		alphaDone.WorkspaceID = "ws-a"
		alphaDone.LoopName = "delivery"
		if _, err := globalDB.CreateLoopRunForStart(ctx, alphaDone, dsl.ConcurrencyAllow); err != nil {
			t.Fatalf("CreateLoopRunForStart(alpha done) error = %v", err)
		}
		if err := globalDB.CompareAndSwapLoopRunStatus(
			ctx,
			alphaDone.ID,
			looppkg.StatusRunning,
			looppkg.StatusDone,
			looppkg.TransitionCauseContract,
			now,
		); err != nil {
			t.Fatalf("CompareAndSwapLoopRunStatus(alpha done) error = %v", err)
		}
		beta := testLoopRun("looprun-api-beta", now.Add(time.Minute), looppkg.StatusRunning)
		beta.WorkspaceID = "ws-b"
		beta.LoopName = "delivery"
		if _, err := globalDB.CreateLoopRunForStart(ctx, beta, dsl.ConcurrencyAllow); err != nil {
			t.Fatalf("CreateLoopRunForStart(beta) error = %v", err)
		}

		runs, err := globalDB.ListLoopRuns(ctx, looppkg.RunListQuery{
			WorkspaceID: "ws-a",
			LoopName:    "delivery",
			Status:      looppkg.StatusRunning,
			Limit:       10,
		})
		if err != nil {
			t.Fatalf("ListLoopRuns() error = %v", err)
		}
		if got, want := len(runs), 1; got != want {
			t.Fatalf("len(runs) = %d, want %d: %#v", got, want, runs)
		}
		if runs[0].ID != alpha.ID || runs[0].WorkspaceID != "ws-a" || runs[0].Inputs["ticket"] != "A" {
			t.Fatalf("ListLoopRuns() run = %#v", runs[0])
		}

		foreign, err := globalDB.ListLoopRuns(ctx, looppkg.RunListQuery{WorkspaceID: "ws-b", Limit: 10})
		if err != nil {
			t.Fatalf("ListLoopRuns(foreign) error = %v", err)
		}
		if got, want := len(foreign), 1; got != want {
			t.Fatalf("len(foreign runs) = %d, want %d: %#v", got, want, foreign)
		}
		if foreign[0].ID != beta.ID || foreign[0].WorkspaceID != "ws-b" {
			t.Fatalf("ListLoopRuns(foreign) run = %#v", foreign[0])
		}
	})
}

func TestGlobalDBLoopAPIEventsShouldResumeBySequenceAndWorkspace(t *testing.T) {
	t.Parallel()

	t.Run("Should return status_changed events after seq without leaking another workspace", func(t *testing.T) {
		t.Parallel()

		globalDB := openFreshLoopTestGlobalDB(t, "ws-a", "ws-b")
		ctx := testutil.Context(t)
		now := time.Date(2026, 7, 5, 11, 0, 0, 0, time.UTC)
		run := testLoopRun("looprun-api-events", now, looppkg.StatusRunning)
		run.WorkspaceID = "ws-a"
		if _, err := globalDB.CreateLoopRunForStart(ctx, run, dsl.ConcurrencyAllow); err != nil {
			t.Fatalf("CreateLoopRunForStart(run) error = %v", err)
		}
		if err := globalDB.CompareAndSwapLoopRunStatus(
			ctx,
			run.ID,
			looppkg.StatusRunning,
			looppkg.StatusPaused,
			looppkg.TransitionCausePauseBoundary,
			now.Add(time.Second),
		); err != nil {
			t.Fatalf("CompareAndSwapLoopRunStatus(run) error = %v", err)
		}

		events, err := globalDB.ListLoopRunEvents(ctx, looppkg.RunEventQuery{
			WorkspaceID: "ws-a",
			RunID:       run.ID,
			AfterSeq:    1,
		})
		if err != nil {
			t.Fatalf("ListLoopRunEvents() error = %v", err)
		}
		if got, want := len(events), 1; got != want {
			t.Fatalf("len(events) = %d, want %d: %#v", got, want, events)
		}
		if events[0].Seq != 2 || events[0].Kind != "status_changed" || events[0].WorkspaceID != "ws-a" {
			t.Fatalf("ListLoopRunEvents() event = %#v", events[0])
		}

		foreign, err := globalDB.ListLoopRunEvents(ctx, looppkg.RunEventQuery{
			WorkspaceID: "ws-b",
			RunID:       run.ID,
			AfterSeq:    0,
		})
		if err != nil {
			t.Fatalf("ListLoopRunEvents(foreign) error = %v", err)
		}
		if len(foreign) != 0 {
			t.Fatalf("foreign events = %#v, want none", foreign)
		}
	})
}

func TestGlobalDBLoopAPIAnnotationsShouldRemainWorkspaceScoped(t *testing.T) {
	t.Parallel()

	t.Run("Should round trip positions for same loop name independently per workspace", func(t *testing.T) {
		t.Parallel()

		globalDB := openFreshLoopTestGlobalDB(t, "ws-a", "ws-b")
		ctx := testutil.Context(t)

		if err := globalDB.ReplaceLoopUIAnnotations(ctx, "ws-a", "delivery", []looppkg.UIAnnotation{
			{NodeID: "draft", X: 10, Y: 20},
		}); err != nil {
			t.Fatalf("ReplaceLoopUIAnnotations(ws-a) error = %v", err)
		}
		if err := globalDB.ReplaceLoopUIAnnotations(ctx, "ws-b", "delivery", []looppkg.UIAnnotation{
			{NodeID: "draft", X: 30, Y: 40},
		}); err != nil {
			t.Fatalf("ReplaceLoopUIAnnotations(ws-b) error = %v", err)
		}

		alpha, err := globalDB.ListLoopUIAnnotations(ctx, "ws-a", "delivery")
		if err != nil {
			t.Fatalf("ListLoopUIAnnotations(ws-a) error = %v", err)
		}
		if got, want := len(alpha), 1; got != want {
			t.Fatalf("len(alpha) = %d, want %d: %#v", got, want, alpha)
		}
		if alpha[0].NodeID != "draft" || alpha[0].X != 10 || alpha[0].Y != 20 {
			t.Fatalf("alpha annotation = %#v", alpha[0])
		}

		beta, err := globalDB.ListLoopUIAnnotations(ctx, "ws-b", "delivery")
		if err != nil {
			t.Fatalf("ListLoopUIAnnotations(ws-b) error = %v", err)
		}
		if got, want := len(beta), 1; got != want {
			t.Fatalf("len(beta) = %d, want %d: %#v", got, want, beta)
		}
		if beta[0].NodeID != "draft" || beta[0].X != 30 || beta[0].Y != 40 {
			t.Fatalf("beta annotation = %#v", beta[0])
		}

		if err := globalDB.ReplaceLoopUIAnnotations(ctx, "ws-a", "delivery", []looppkg.UIAnnotation{
			{NodeID: "review", X: 50, Y: 60},
		}); err != nil {
			t.Fatalf("ReplaceLoopUIAnnotations(ws-a replace) error = %v", err)
		}
		alpha, err = globalDB.ListLoopUIAnnotations(ctx, "ws-a", "delivery")
		if err != nil {
			t.Fatalf("ListLoopUIAnnotations(ws-a replace) error = %v", err)
		}
		if got, want := len(alpha), 1; got != want {
			t.Fatalf("len(alpha replaced) = %d, want %d: %#v", got, want, alpha)
		}
		if alpha[0].NodeID != "review" || alpha[0].X != 50 || alpha[0].Y != 60 {
			t.Fatalf("alpha replaced annotation = %#v", alpha[0])
		}

		beta, err = globalDB.ListLoopUIAnnotations(ctx, "ws-b", "delivery")
		if err != nil {
			t.Fatalf("ListLoopUIAnnotations(ws-b after replace) error = %v", err)
		}
		if beta[0].NodeID != "draft" || beta[0].X != 30 || beta[0].Y != 40 {
			t.Fatalf("beta annotation after alpha replace = %#v", beta[0])
		}
	})
}
