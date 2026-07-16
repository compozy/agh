package core_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/compozy/agh/internal/api/contract"
	"github.com/compozy/agh/internal/api/core"
	"github.com/compozy/agh/internal/api/testutil"
	"github.com/compozy/agh/internal/store"
	workspacepkg "github.com/compozy/agh/internal/workspace"
)

func usageWorkspaceService() testutil.StubWorkspaceService {
	return testutil.StubWorkspaceService{
		GetFn: func(_ context.Context, ref string) (workspacepkg.Workspace, error) {
			return workspacepkg.Workspace{ID: "ws-alpha", Name: ref, RootDir: "/workspace"}, nil
		},
	}
}

type stubNetworkUsageStore struct {
	report    store.NetworkUsageReport
	err       error
	lastQuery store.NetworkUsageQuery
}

func (s *stubNetworkUsageStore) GetNetworkUsage(
	_ context.Context,
	query store.NetworkUsageQuery,
) (store.NetworkUsageReport, error) {
	s.lastQuery = query
	if s.err != nil {
		return store.NetworkUsageReport{}, s.err
	}
	return s.report, nil
}

func TestNetworkUsageHandlers(t *testing.T) {
	t.Parallel()

	t.Run("Should return workspace-scoped ledger totals and details", func(t *testing.T) {
		t.Parallel()

		fixture := newHandlerFixture(
			t,
			testutil.StubSessionManager{},
			testutil.StubObserver{},
			usageWorkspaceService(),
			nil,
			nil,
		)
		settled := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
		usage := &stubNetworkUsageStore{
			report: store.NetworkUsageReport{
				Details: []store.NetworkWakeUsageDetail{{
					WakeID:       "wake-1",
					TaskRunID:    "run-1",
					OwnerKey:     "session:sess-1",
					WorkspaceID:  "ws-alpha",
					Channel:      "builders",
					State:        "settled",
					UsageState:   "actual",
					InputTokens:  11,
					OutputTokens: 7,
					SettledAt:    &settled,
				}},
				Total: store.NetworkUsageSummary{
					WakeCount:       1,
					ActualWakeCount: 1,
					InputTokens:     11,
					OutputTokens:    7,
				},
				Budget: &store.NetworkBudgetUsage{
					OwnerKey:         "task_run:run-1",
					WakesUsed:        1,
					WallTimeUsed:     time.Second,
					InputTokensUsed:  11,
					OutputTokensUsed: 7,
					UpdatedAt:        settled,
				},
			},
		}
		fixture.Handlers.NetworkUsage = usage

		resp := performRequest(
			t,
			fixture.Engine,
			http.MethodGet,
			"/workspaces/alpha/network/usage?channel=builders&run_id=run-1",
			nil,
		)
		if resp.Code != http.StatusOK {
			t.Fatalf("GET usage status = %d body=%s", resp.Code, resp.Body.String())
		}
		var body contract.NetworkUsageResponse
		if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode usage response: %v", err)
		}
		if body.WorkspaceID != "ws-alpha" {
			t.Fatalf("workspace_id = %q, want ws-alpha", body.WorkspaceID)
		}
		if body.Total.ActualWakeCount != 1 || body.Total.InputTokens != 11 {
			t.Fatalf("total = %#v, want actual wake + tokens", body.Total)
		}
		if len(body.Details) != 1 || body.Details[0].WakeID != "wake-1" {
			t.Fatalf("details = %#v, want wake-1", body.Details)
		}
		if body.Budget == nil || body.Budget.OwnerKey != "task_run:run-1" ||
			body.Budget.WallTimeUsed != "1s" {
			t.Fatalf("budget = %#v, want run owner consumption", body.Budget)
		}
		if usage.lastQuery.WorkspaceID != "ws-alpha" ||
			usage.lastQuery.Channel != "builders" ||
			usage.lastQuery.RunID != "run-1" {
			t.Fatalf("query = %#v, want workspace/channel/run filters", usage.lastQuery)
		}
	})

	t.Run("Should convert store report without inventing totals", func(t *testing.T) {
		t.Parallel()

		report := store.NetworkUsageReport{
			Total: store.NetworkUsageSummary{
				WakeCount:            2,
				ReservedWakeCount:    1,
				ActualWakeCount:      1,
				UnavailableWakeCount: 0,
				InputTokens:          3,
				OutputTokens:         4,
			},
		}
		got := core.NetworkUsageResponseFromReport("ws-beta", report)
		if got.WorkspaceID != "ws-beta" {
			t.Fatalf("workspace_id = %q, want ws-beta", got.WorkspaceID)
		}
		if got.Total.WakeCount != 2 || got.Total.ActualWakeCount != 1 {
			t.Fatalf("total = %#v, want ledger totals", got.Total)
		}
		if len(got.Details) != 0 {
			t.Fatalf("details = %#v, want empty", got.Details)
		}
	})
}
