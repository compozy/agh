package cli

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/compozy/agh/internal/api/contract"
)

func TestNetworkPublicSurfaceCommands(t *testing.T) {
	t.Parallel()

	t.Run("Should print coordination status as structured JSON", func(t *testing.T) {
		t.Parallel()

		updatedAt := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
		deps := newTestDeps(t, &stubClient{
			getNetworkCoordinationFn: func(
				_ context.Context,
				workspaceRef string,
				_ string,
			) (NetworkCoordinationRecord, error) {
				if workspaceRef != "ws-alpha" {
					t.Fatalf("workspace = %q, want ws-alpha", workspaceRef)
				}
				return NetworkCoordinationRecord{
					WorkspaceID: "ws-alpha",
					Enabled:     true,
					Revision:    4,
					UpdatedAt:   updatedAt,
					UpdatedBy:   "operator",
				}, nil
			},
		})
		stdout, _, err := executeRootCommand(
			t,
			deps,
			"network",
			"--workspace",
			"ws-alpha",
			"coordination",
			"status",
			"--json",
		)
		if err != nil {
			t.Fatalf("network coordination status error = %v", err)
		}
		var envelope struct {
			Coordination NetworkCoordinationRecord `json:"coordination"`
		}
		if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
			t.Fatalf("decode stdout: %v\nstdout=%s", err, stdout)
		}
		payload := envelope.Coordination
		if !payload.Enabled || payload.Revision != 4 {
			t.Fatalf("payload = %#v, want enabled revision=4", payload)
		}
	})

	t.Run("Should enable coordination through CLI PUT", func(t *testing.T) {
		t.Parallel()

		var captured PutNetworkCoordinationRequest
		deps := newTestDeps(t, &stubClient{
			putNetworkCoordinationFn: func(
				_ context.Context,
				_ string,
				request PutNetworkCoordinationRequest,
				_ string,
			) (NetworkCoordinationRecord, error) {
				captured = request
				return NetworkCoordinationRecord{
					WorkspaceID: "ws-alpha",
					Enabled:     request.Enabled,
					Revision:    1,
					UpdatedAt:   time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC),
					UpdatedBy:   "operator",
				}, nil
			},
		})
		if _, _, err := executeRootCommand(
			t,
			deps,
			"network",
			"--workspace",
			"ws-alpha",
			"coordination",
			"enable",
			"--json",
		); err != nil {
			t.Fatalf("network coordination enable error = %v", err)
		}
		if !captured.Enabled {
			t.Fatal("captured.Enabled = false, want true")
		}
	})

	t.Run("Should print network usage totals as structured JSON", func(t *testing.T) {
		t.Parallel()

		deps := newTestDeps(t, &stubClient{
			getNetworkUsageFn: func(_ context.Context, workspaceRef string) (NetworkUsageRecord, error) {
				if workspaceRef != "ws-alpha" {
					t.Fatalf("workspace = %q, want ws-alpha", workspaceRef)
				}
				return NetworkUsageRecord{
					WorkspaceID: "ws-alpha",
					Total: contract.NetworkUsageSummaryPayload{
						WakeCount:       2,
						ActualWakeCount: 2,
						InputTokens:     9,
						OutputTokens:    3,
					},
				}, nil
			},
		})
		stdout, _, err := executeRootCommand(
			t,
			deps,
			"network",
			"--workspace",
			"ws-alpha",
			"usage",
			"--json",
		)
		if err != nil {
			t.Fatalf("network usage error = %v", err)
		}
		if !strings.Contains(stdout, "ws-alpha") {
			t.Fatalf("stdout = %s, want usage payload", stdout)
		}
		var payload NetworkUsageRecord
		if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
			t.Fatalf("decode stdout: %v\nstdout=%s", err, stdout)
		}
		if payload.Total.ActualWakeCount != 2 {
			t.Fatalf("total = %#v, want actual_wake_count=2", payload.Total)
		}
	})
}
