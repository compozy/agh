//go:build integration

package httpapi

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	aghcontract "github.com/pedronauck/agh/internal/api/contract"
	automationpkg "github.com/pedronauck/agh/internal/automation"
	"github.com/pedronauck/agh/internal/testutil/acpmock"
	e2etest "github.com/pedronauck/agh/internal/testutil/e2e"
)

const (
	transportApprovalAgentName = "transport-approver"
	transportAutomationAgent   = "transport-automation-runner"
	transportFaultyAgent       = "transport-faulty-runner"
)

func TestHTTPTransportApprovalFlowUsesSharedRuntimeHarness(t *testing.T) {
	acpmock.RequireDriver(t)
	t.Parallel()

	runtimeHarness := e2etest.StartRuntimeHarness(t, e2etest.RuntimeHarnessOptions{
		MockAgents: []e2etest.MockAgentSpec{{
			FixturePath:  transportMockFixturePath(t, "permission_env_fixture.json"),
			FixtureAgent: "approver",
			AgentName:    transportApprovalAgentName,
		}},
	})

	clients, err := runtimeHarness.TransportClients()
	if err != nil {
		t.Fatalf("TransportClients() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	session, err := runtimeHarness.CreateSession(ctx, aghcontract.CreateSessionRequest{
		AgentName:     transportApprovalAgentName,
		WorkspacePath: runtimeHarness.WorkspaceRoot,
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	var approvedRequestID string
	events, err := runtimeHarness.PromptSessionHTTPWithEvents(
		ctx,
		session.ID,
		"request permission",
		func(record e2etest.SSEEvent) error {
			payload, ok := e2etest.PermissionPayloadFromSSE(record)
			if !ok || payload.Decision != "" || approvedRequestID != "" {
				return nil
			}
			approvedRequestID = payload.RequestID
			return runtimeHarness.ApproveSessionPermission(ctx, session.ID, aghcontract.ApproveSessionRequest{
				RequestID: payload.RequestID,
				Decision:  "allow-always",
			})
		},
	)
	if err != nil {
		t.Fatalf("PromptSessionHTTPWithEvents() error = %v", err)
	}
	if approvedRequestID == "" {
		t.Fatal("approvedRequestID = empty, want pending permission request")
	}

	payloads := e2etest.PermissionPayloads(events)
	if got, want := len(payloads), 2; got != want {
		t.Fatalf("len(permission payloads) = %d, want %d; payloads=%#v", got, want, payloads)
	}
	if got, want := payloads[0].RequestID, approvedRequestID; got != want {
		t.Fatalf("payloads[0].RequestID = %q, want %q", got, want)
	}
	if got := payloads[len(payloads)-1].Decision; got != "allow-always" {
		t.Fatalf("final permission decision = %q, want %q", got, "allow-always")
	}
	if !e2etest.RecordsContainTextDelta(events, "allow-always") {
		t.Fatalf("prompt events = %#v, want allow-always text delta", events)
	}

	sessionResp := mustHTTPRequest(
		t,
		clients.HTTPClient,
		http.MethodGet,
		runtimeHarness.HTTPURL("/api/sessions/"+url.PathEscape(session.ID)),
		nil,
		nil,
	)
	if sessionResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(sessionResp.Body)
		_ = sessionResp.Body.Close()
		t.Fatalf("session status = %d, want %d; body=%s", sessionResp.StatusCode, http.StatusOK, string(body))
	}
	var detail struct {
		Session sessionPayload `json:"session"`
	}
	decodeHTTPJSON(t, sessionResp, &detail)
	if got, want := detail.Session.ID, session.ID; got != want {
		t.Fatalf("detail.Session.ID = %q, want %q", got, want)
	}
}

func TestHTTPTransportWebhookIngressUsesSharedRuntimeHarness(t *testing.T) {
	acpmock.RequireDriver(t)
	t.Parallel()

	runtimeHarness := e2etest.StartRuntimeHarness(t, e2etest.RuntimeHarnessOptions{
		ConfigSeed: e2etest.ConfigSeedOptions{
			DefaultAgent: transportAutomationAgent,
		},
		MockAgents: []e2etest.MockAgentSpec{{
			FixturePath:  transportMockFixturePath(t, "automation_task_fixture.json"),
			FixtureAgent: "automation-runner",
			AgentName:    transportAutomationAgent,
		}},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	trigger, endpoint := seedTransportWebhookTrigger(t, ctx, runtimeHarness)
	payload := []byte(`{"payload":"deploy","branch":"main"}`)
	delivery, err := runtimeHarness.DeliverGlobalWebhook(
		ctx,
		endpoint,
		"shared-secret",
		payload,
		"delivery-http-transport",
		time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("DeliverGlobalWebhook() error = %v", err)
	}
	if got, want := len(delivery.Runs), 1; got != want {
		t.Fatalf("len(delivery.Runs) = %d, want %d; delivery=%#v", got, want, delivery)
	}

	httpRun := waitForHTTPAutomationRun(t, ctx, runtimeHarness, delivery.Runs[0].ID)
	if err := e2etest.ValidateWebhookRunProjection(delivery, httpRun); err != nil {
		t.Fatalf("ValidateWebhookRunProjection() error = %v", err)
	}
	if got, want := httpRun.TriggerID, trigger.ID; got != want {
		t.Fatalf("httpRun.TriggerID = %q, want %q", got, want)
	}
	if httpRun.SessionID == "" {
		t.Fatalf("httpRun.SessionID = %q, want non-empty session linkage", httpRun.SessionID)
	}
}

func TestHTTPTransportPromptFailureProjectionUsesSharedRuntimeHarness(t *testing.T) {
	acpmock.RequireDriver(t)
	t.Parallel()

	runtimeHarness := e2etest.StartRuntimeHarness(t, e2etest.RuntimeHarnessOptions{
		MockAgents: []e2etest.MockAgentSpec{{
			FixturePath:  transportMockFixturePath(t, "driver_fault_fixture.json"),
			FixtureAgent: "faulty",
			AgentName:    transportFaultyAgent,
		}},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	session, err := runtimeHarness.CreateSession(ctx, aghcontract.CreateSessionRequest{
		AgentName:     transportFaultyAgent,
		WorkspacePath: runtimeHarness.WorkspaceRoot,
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	stream, err := runtimeHarness.PromptSessionHTTP(ctx, session.ID, "trigger invalid frame")
	if err != nil {
		t.Fatalf("PromptSessionHTTP() error = %v", err)
	}
	if !e2etest.RecordsContainTextDelta(stream, "partial before invalid frame") {
		t.Fatalf("HTTP prompt stream = %#v, want partial assistant delta", stream)
	}
	if !httpSSEContainsEvent(stream, "error") {
		t.Fatalf("HTTP prompt stream = %#v, want error event", stream)
	}

	var eventsResp aghcontract.SessionEventsResponse
	if err := runtimeHarness.HTTPJSON(
		ctx,
		http.MethodGet,
		"/api/sessions/"+url.PathEscape(session.ID)+"/events",
		nil,
		&eventsResp,
	); err != nil {
		t.Fatalf("HTTP session events error = %v", err)
	}
	if !httpSessionEventsContainType(eventsResp.Events, "error") {
		t.Fatalf("HTTP session events = %#v, want error projection", eventsResp.Events)
	}
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
			AgentName:     transportAutomationAgent,
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

	waitCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()

	var (
		lastErr  error
		lastSeen string
	)
	for {
		var response aghcontract.RunResponse
		err := harness.HTTPJSON(waitCtx, http.MethodGet, "/api/automation/runs/"+url.PathEscape(runID), nil, &response)
		if err == nil && response.Run.ID == runID {
			return response.Run
		}
		lastErr = err
		lastSeen = response.Run.ID
		select {
		case <-waitCtx.Done():
			t.Fatalf(
				"waitForHTTPAutomationRun(%q) timed out: %v; last error=%v last run=%q",
				runID,
				waitCtx.Err(),
				lastErr,
				lastSeen,
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

func httpSSEContainsEvent(records []e2etest.SSEEvent, want string) bool {
	for _, record := range records {
		if record.Event == want {
			return true
		}
	}
	return false
}

func httpSessionEventsContainType(events []aghcontract.SessionEventPayload, want string) bool {
	for _, event := range events {
		if strings.Contains(string(event.Content), `"type":"`+want+`"`) {
			return true
		}
	}
	return false
}
