package daemon

import (
	"context"
	"testing"
	"time"

	automationpkg "github.com/compozy/agh/internal/automation"
	looppkg "github.com/compozy/agh/internal/loop"
	loopdsl "github.com/compozy/agh/internal/loop/dsl"
)

func TestAutomationLoopStartMetadataShouldIncludeCatchUpEvidence(t *testing.T) {
	t.Parallel()

	scheduledAt := time.Date(2026, 7, 7, 9, 45, 0, 0, time.UTC)
	metadata := automationLoopStartMetadata(automationpkg.LoopStartRequest{
		AutomationRunID: "autorun-1",
		ScheduledAt:     &scheduledAt,
		CatchUp:         true,
		CatchUpPolicy:   automationpkg.SchedulerCatchUpPolicyCoalesce,
	})

	if got, want := metadata["automation_run_id"], "autorun-1"; got != want {
		t.Fatalf("automation_run_id = %#v, want %q", got, want)
	}
	if got, want := metadata["scheduled_at"], scheduledAt.Format(time.RFC3339Nano); got != want {
		t.Fatalf("scheduled_at = %#v, want %q", got, want)
	}
	if got, want := metadata["original_due_at"], scheduledAt.Format(time.RFC3339Nano); got != want {
		t.Fatalf("original_due_at = %#v, want %q", got, want)
	}
	if got, want := metadata["catch_up"], true; got != want {
		t.Fatalf("catch_up = %#v, want %v", got, want)
	}
	if got, want := metadata["catch_up_policy"], "coalesce"; got != want {
		t.Fatalf("catch_up_policy = %#v, want %q", got, want)
	}
}

func TestAutomationLoopStarterDefaultCatchUpPolicyShouldCoalesceWatchLoops(t *testing.T) {
	t.Parallel()

	starter := &automationLoopStarter{
		resolver: looppkg.DefinitionResolverFunc(
			func(context.Context, looppkg.WorkspaceID, string) (*looppkg.ResolvedDefinition, error) {
				return &looppkg.ResolvedDefinition{
					Definition: loopdsl.Definition{
						Graph: loopdsl.Graph{Nodes: []loopdsl.Node{{
							ID:    "watch",
							Class: loopdsl.NodeClassSource,
							Kind:  string(loopdsl.SourceWatchSource),
						}}},
					},
				}, nil
			},
		),
	}

	policy, err := starter.DefaultLoopCatchUpPolicy(context.Background(), automationpkg.LoopCatchUpPolicyRequest{
		WorkspaceID: "ws-1",
		LoopName:    "watch-loop",
		Kind:        automationpkg.LoopStartKindSchedule,
	})
	if err != nil {
		t.Fatalf("DefaultLoopCatchUpPolicy() error = %v", err)
	}
	if policy != automationpkg.SchedulerCatchUpPolicyCoalesce {
		t.Fatalf("DefaultLoopCatchUpPolicy() = %q, want coalesce", policy)
	}
}
