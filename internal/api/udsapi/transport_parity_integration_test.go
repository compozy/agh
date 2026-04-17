//go:build integration

package udsapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	aghcontract "github.com/pedronauck/agh/internal/api/contract"
	automationpkg "github.com/pedronauck/agh/internal/automation"
	"github.com/pedronauck/agh/internal/testutil/acpmock"
	e2etest "github.com/pedronauck/agh/internal/testutil/e2e"
)

const (
	transportUDSApprovalAgent   = "transport-uds-approver"
	transportUDSAutomationAgent = "transport-uds-automation-runner"
)

var errStopAfterUDSApprovalGap = errors.New("stop after documenting UDS approval gap")

func TestUDSTransportApprovalRouteDocumentsNotImplementedGap(t *testing.T) {
	acpmock.RequireDriver(t)
	t.Parallel()

	runtimeHarness := e2etest.StartRuntimeHarness(t, e2etest.RuntimeHarnessOptions{
		MockAgents: []e2etest.MockAgentSpec{{
			FixturePath:  transportMockFixturePath(t, "permission_env_fixture.json"),
			FixtureAgent: "approver",
			AgentName:    transportUDSApprovalAgent,
		}},
	})

	clients, err := runtimeHarness.TransportClients()
	if err != nil {
		t.Fatalf("TransportClients() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	session, err := runtimeHarness.CreateSession(ctx, aghcontract.CreateSessionRequest{
		AgentName:     transportUDSApprovalAgent,
		WorkspacePath: runtimeHarness.WorkspaceRoot,
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	attemptedApproval := false
	_, err = runtimeHarness.PromptSessionHTTPWithEvents(
		ctx,
		session.ID,
		"request permission",
		func(record e2etest.SSEEvent) error {
			payload, ok := e2etest.PermissionPayloadFromSSE(record)
			if !ok || payload.Decision != "" || attemptedApproval {
				return nil
			}
			attemptedApproval = true

			resp := mustUnixRequest(
				t,
				clients.UDSClient,
				http.MethodPost,
				clients.UDSBaseURL+"/api/sessions/"+url.PathEscape(session.ID)+"/approve",
				[]byte(fmt.Sprintf(`{"request_id":"%s","decision":"allow-always"}`, payload.RequestID)),
				nil,
			)
			body, readErr := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if readErr != nil {
				return fmt.Errorf("read UDS approval response: %w", readErr)
			}
			if err := e2etest.ValidateUDSApprovalNotImplemented(resp.StatusCode, body); err != nil {
				return err
			}
			return errStopAfterUDSApprovalGap
		},
	)
	if err != nil && !errors.Is(err, errStopAfterUDSApprovalGap) {
		t.Fatalf("PromptSessionHTTPWithEvents() error = %v", err)
	}
	if !attemptedApproval {
		t.Fatal("attemptedApproval = false, want UDS approval attempt while permission request is pending")
	}
}

func TestUDSTransportProjectionParityMatchesHTTPAndCLI(t *testing.T) {
	acpmock.RequireDriver(t)
	t.Parallel()

	runtimeHarness := e2etest.StartRuntimeHarness(t, e2etest.RuntimeHarnessOptions{
		ConfigSeed: e2etest.ConfigSeedOptions{
			DefaultAgent: transportUDSAutomationAgent,
		},
		MockAgents: []e2etest.MockAgentSpec{{
			FixturePath:  transportMockFixturePath(t, "automation_task_fixture.json"),
			FixtureAgent: "automation-runner",
			AgentName:    transportUDSAutomationAgent,
		}},
	})

	clients, err := runtimeHarness.TransportClients()
	if err != nil {
		t.Fatalf("TransportClients() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	trigger, endpoint := seedTransportWebhookTrigger(t, ctx, runtimeHarness)
	delivery, err := runtimeHarness.DeliverGlobalWebhook(
		ctx,
		endpoint,
		"shared-secret",
		[]byte(`{"payload":"deploy","branch":"main"}`),
		"delivery-uds-transport",
		time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("DeliverGlobalWebhook() error = %v", err)
	}
	if got, want := len(delivery.Runs), 1; got != want {
		t.Fatalf("len(delivery.Runs) = %d, want %d; delivery=%#v", got, want, delivery)
	}

	runID := delivery.Runs[0].ID
	httpRun := waitForHTTPAutomationRun(t, ctx, runtimeHarness, runID)
	udsRun := waitForUDSAutomationRun(t, ctx, runtimeHarness, runID)
	cliRun := waitForCLIAutomationRun(t, ctx, clients.CLI, runID)

	if err := e2etest.ValidateWebhookRunProjection(delivery, httpRun, udsRun, cliRun); err != nil {
		t.Fatalf("ValidateWebhookRunProjection() error = %v", err)
	}
	if got, want := cliRun.TriggerID, trigger.ID; got != want {
		t.Fatalf("cliRun.TriggerID = %q, want %q", got, want)
	}
}

func waitForUDSAutomationRun(
	t testing.TB,
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	runID string,
) aghcontract.RunPayload {
	t.Helper()

	return waitForTransportAutomationRun(t, ctx, runID, "waitForUDSAutomationRun", func(fetchCtx context.Context) (aghcontract.RunPayload, error) {
		return harness.GetAutomationRun(fetchCtx, runID)
	})
}

func waitForCLIAutomationRun(
	t testing.TB,
	ctx context.Context,
	client *e2etest.CLIClient,
	runID string,
) aghcontract.RunPayload {
	t.Helper()

	return waitForTransportAutomationRun(t, ctx, runID, "waitForCLIAutomationRun", func(fetchCtx context.Context) (aghcontract.RunPayload, error) {
		var run aghcontract.RunPayload
		err := client.RunJSON(fetchCtx, &run, "automation", "runs", "get", runID, "-o", "json")
		return run, err
	})
}

func seedTransportWebhookTrigger(
	t testing.TB,
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
) (aghcontract.TriggerPayload, string) {
	t.Helper()

	state, err := harness.SeedAutomationFixtures(ctx, e2etest.AutomationFixtureSeed{
		Triggers: []aghcontract.CreateTriggerRequest{{
			Scope:         automationpkg.AutomationScopeGlobal,
			Name:          "deploy-review",
			AgentName:     transportUDSAutomationAgent,
			Prompt:        `Review payload {{ index .Data "payload" }} for {{ index .Data "branch" }}`,
			Event:         "webhook",
			EndpointSlug:  "deploy-review",
			WebhookSecret: "shared-secret",
		}},
	})
	if err != nil {
		t.Fatalf("SeedAutomationFixtures() error = %v", err)
	}
	if got, want := len(state.Triggers), 1; got != want {
		t.Fatalf("len(state.Triggers) = %d, want %d", got, want)
	}

	endpoint, err := automationpkg.FormatWebhookEndpoint(
		state.Triggers[0].EndpointSlug,
		state.Triggers[0].WebhookID,
	)
	if err != nil {
		t.Fatalf("FormatWebhookEndpoint() error = %v", err)
	}
	return state.Triggers[0], endpoint
}

func waitForHTTPAutomationRun(
	t testing.TB,
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	runID string,
) aghcontract.RunPayload {
	t.Helper()

	return waitForTransportAutomationRun(t, ctx, runID, "waitForHTTPAutomationRun", func(fetchCtx context.Context) (aghcontract.RunPayload, error) {
		var response aghcontract.RunResponse
		err := harness.HTTPJSON(fetchCtx, http.MethodGet, "/api/automation/runs/"+url.PathEscape(runID), nil, &response)
		return response.Run, err
	})
}

func waitForTransportAutomationRun(
	t testing.TB,
	ctx context.Context,
	runID string,
	label string,
	fetch func(context.Context) (aghcontract.RunPayload, error),
) aghcontract.RunPayload {
	t.Helper()

	waitCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()

	var (
		lastErr error
		lastRun aghcontract.RunPayload
	)
	for {
		run, err := fetch(waitCtx)
		if err == nil && run.ID == runID {
			return run
		}
		lastErr = err
		lastRun = run
		select {
		case <-waitCtx.Done():
			t.Fatalf(
				"%s(%q) timed out: %v; last error=%v last run=%#v",
				label,
				runID,
				waitCtx.Err(),
				lastErr,
				lastRun,
			)
		case <-ticker.C:
		}
	}
}

func transportMockFixturePath(t testing.TB, name string) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "testutil", "acpmock", "testdata", name)
}
