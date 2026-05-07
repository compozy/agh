package automation

import (
	"context"
	"testing"

	"github.com/pedronauck/agh/internal/testutil"
)

type mutatingTriggerFilterDispatcher struct{}

func (mutatingTriggerFilterDispatcher) Dispatch(_ context.Context, req DispatchRequest) (*Run, error) {
	if req.Trigger != nil && req.Trigger.Filter != nil {
		req.Trigger.Filter["data.agent_name"] = "mutated"
	}
	return nil, nil
}

func TestTriggerFilterPathMatching(t *testing.T) {
	t.Parallel()

	envelope := ActivationEnvelope{
		Kind:        "session.stopped",
		Scope:       AutomationScopeWorkspace,
		WorkspaceID: "ws_alpha",
		Source:      ActivationSourceObserver,
		Data: map[string]any{
			"agent_name": "researcher",
			"metadata": map[string]any{
				"step": "complete",
			},
			"labels": map[string]string{
				"priority": "high",
			},
		},
	}

	tests := []struct {
		name   string
		filter map[string]string
		want   bool
	}{
		{
			name: "Should match nested data while trimming path segments",
			filter: map[string]string{
				" data.metadata. step ": "complete",
			},
			want: true,
		},
		{
			name: "Should match nested string maps",
			filter: map[string]string{
				"data.labels.priority": "high",
			},
			want: true,
		},
		{
			name: "Should reject empty nested path segments",
			filter: map[string]string{
				"data.metadata..step": "complete",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := exactFilterMatch(tt.filter, envelope); got != tt.want {
				t.Fatalf("exactFilterMatch() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTriggerDispatchSnapshotIsolation(t *testing.T) {
	t.Parallel()

	t.Run("Should keep registered filter isolated from dispatcher mutations", func(t *testing.T) {
		t.Parallel()

		engine, err := NewTriggerEngine(mutatingTriggerFilterDispatcher{})
		if err != nil {
			t.Fatalf("NewTriggerEngine() error = %v", err)
		}
		trigger := testEventTrigger(AutomationScopeWorkspace, "snapshot-isolation", "ws_alpha", "session.stopped")
		trigger.Filter = map[string]string{
			"data.agent_name": "researcher",
		}
		if err := engine.Register(TriggerRegistration{Trigger: trigger}); err != nil {
			t.Fatalf("Register() error = %v", err)
		}

		envelope := ActivationEnvelope{
			Kind:        "session.stopped",
			Scope:       AutomationScopeWorkspace,
			WorkspaceID: "ws_alpha",
			Source:      ActivationSourceObserver,
			Data: map[string]any{
				"agent_name": "researcher",
			},
		}
		for i := range 2 {
			result, err := engine.Fire(testutil.Context(t), envelope)
			if err != nil {
				t.Fatalf("Fire(%d) error = %v", i, err)
			}
			if got, want := result.Matched, 1; got != want {
				t.Fatalf("Fire(%d).Matched = %d, want %d", i, got, want)
			}
		}
	})
}
