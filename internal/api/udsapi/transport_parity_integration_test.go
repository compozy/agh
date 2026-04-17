//go:build integration

package udsapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	aghcontract "github.com/pedronauck/agh/internal/api/contract"
	automationpkg "github.com/pedronauck/agh/internal/automation"
	"github.com/pedronauck/agh/internal/testutil/acpmock"
	e2etest "github.com/pedronauck/agh/internal/testutil/e2e"
	"github.com/pedronauck/agh/internal/transcript"
)

const (
	transportUDSApprovalAgent   = "transport-uds-approver"
	transportUDSAutomationAgent = "transport-uds-automation-runner"
	transportUDSFaultyAgent     = "transport-uds-faulty-runner"
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
			closeErr := resp.Body.Close()
			if readErr != nil && closeErr != nil {
				return errors.Join(
					fmt.Errorf("read UDS approval response: %w", readErr),
					fmt.Errorf("close UDS approval response body: %w", closeErr),
				)
			}
			if readErr != nil {
				return fmt.Errorf("read UDS approval response: %w", readErr)
			}
			if closeErr != nil {
				return fmt.Errorf("close UDS approval response body: %w", closeErr)
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

func TestUDSTransportPromptFailureProjectionUsesSharedRuntimeHarness(t *testing.T) {
	acpmock.RequireDriver(t)
	t.Parallel()

	runtimeHarness := e2etest.StartRuntimeHarness(t, e2etest.RuntimeHarnessOptions{
		MockAgents: []e2etest.MockAgentSpec{{
			FixturePath:  transportMockFixturePath(t, "driver_fault_fixture.json"),
			FixtureAgent: "faulty",
			AgentName:    transportUDSFaultyAgent,
		}},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	session, err := runtimeHarness.CreateSession(ctx, aghcontract.CreateSessionRequest{
		AgentName:     transportUDSFaultyAgent,
		WorkspacePath: runtimeHarness.WorkspaceRoot,
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	stream, err := runtimeHarness.PromptSession(ctx, session.ID, "trigger crash mid-stream")
	if err != nil {
		t.Fatalf("PromptSession() error = %v", err)
	}
	if !udsSSEContainsEvent(stream, "error") {
		t.Fatalf("UDS prompt stream = %#v, want error event", stream)
	}

	transcript, err := runtimeHarness.SessionTranscript(ctx, session.ID)
	if err != nil {
		t.Fatalf("SessionTranscript() error = %v", err)
	}
	if !strings.Contains(joinTransportTranscript(transcript.Messages), "partial before crash") {
		t.Fatalf("transcript = %#v, want partial assistant output", transcript.Messages)
	}

	eventsResp, err := runtimeHarness.SessionEvents(ctx, session.ID)
	if err != nil {
		t.Fatalf("SessionEvents() error = %v", err)
	}
	if !udsSessionEventsContainType(eventsResp.Events, "error") {
		t.Fatalf("UDS session events = %#v, want error projection", eventsResp.Events)
	}
}

func TestUDSTransportSettingsReadParityMatchesHTTP(t *testing.T) {
	acpmock.RequireDriver(t)
	t.Parallel()

	runtimeHarness := e2etest.StartRuntimeHarness(t, e2etest.RuntimeHarnessOptions{})

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	workspaceID := runtimeHarness.WorkspaceID

	testCases := []struct {
		name   string
		path   string
		decode func() any
	}{
		{
			name:   "general section",
			path:   "/api/settings/general",
			decode: func() any { return &aghcontract.SettingsGeneralResponse{} },
		},
		{
			name:   "providers collection",
			path:   "/api/settings/providers",
			decode: func() any { return &aghcontract.SettingsProvidersResponse{} },
		},
		{
			name: "workspace mcp servers collection",
			path: "/api/settings/mcp-servers?scope=workspace&workspace_id=" + url.QueryEscape(workspaceID),
			decode: func() any {
				return &aghcontract.SettingsMCPServersResponse{}
			},
		},
		{
			name:   "hooks and extensions section",
			path:   "/api/settings/hooks-extensions",
			decode: func() any { return &aghcontract.SettingsHooksExtensionsResponse{} },
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			httpValue := tc.decode()
			if err := runtimeHarness.HTTPJSON(ctx, http.MethodGet, tc.path, nil, httpValue); err != nil {
				t.Fatalf("HTTPJSON(%s) error = %v", tc.path, err)
			}

			udsValue := tc.decode()
			if err := runtimeHarness.UDSJSON(ctx, http.MethodGet, tc.path, nil, udsValue); err != nil {
				t.Fatalf("UDSJSON(%s) error = %v", tc.path, err)
			}

			if !reflect.DeepEqual(httpValue, udsValue) {
				t.Fatalf("%s HTTP payload = %#v, want UDS parity %#v", tc.path, httpValue, udsValue)
			}
		})
	}
}

func TestUDSTransportSettingsDependencyExtensionParityMatchesHTTP(t *testing.T) {
	acpmock.RequireDriver(t)
	t.Parallel()

	runtimeHarness := e2etest.StartRuntimeHarness(t, e2etest.RuntimeHarnessOptions{})

	clients, err := runtimeHarness.TransportClients()
	if err != nil {
		t.Fatalf("TransportClients() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	httpListResp := mustUnixRequest(
		t,
		clients.HTTPClient,
		http.MethodGet,
		transportSettingsHTTPURL(runtimeHarness, "/api/extensions"),
		nil,
		nil,
	)
	if httpListResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpListResp.Body)
		_ = httpListResp.Body.Close()
		t.Fatalf("HTTP extension list status = %d, want %d; body=%s", httpListResp.StatusCode, http.StatusOK, string(body))
	}
	var httpList aghcontract.ExtensionsResponse
	decodeHTTPJSON(t, httpListResp, &httpList)

	var udsList aghcontract.ExtensionsResponse
	if err := runtimeHarness.UDSJSON(ctx, http.MethodGet, "/api/extensions", nil, &udsList); err != nil {
		t.Fatalf("UDSJSON(/api/extensions) error = %v", err)
	}
	if !reflect.DeepEqual(httpList.Extensions, udsList.Extensions) {
		t.Fatalf("HTTP extensions = %#v, want UDS parity %#v", httpList.Extensions, udsList.Extensions)
	}
	if len(httpList.Extensions) == 0 {
		return
	}

	name := httpList.Extensions[0].Name
	httpStatusResp := mustUnixRequest(
		t,
		clients.HTTPClient,
		http.MethodGet,
		transportSettingsHTTPURL(runtimeHarness, "/api/extensions/"+url.PathEscape(name)),
		nil,
		nil,
	)
	if httpStatusResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpStatusResp.Body)
		_ = httpStatusResp.Body.Close()
		t.Fatalf("HTTP extension status = %d, want %d; body=%s", httpStatusResp.StatusCode, http.StatusOK, string(body))
	}
	var httpStatus aghcontract.ExtensionResponse
	decodeHTTPJSON(t, httpStatusResp, &httpStatus)

	var udsStatus aghcontract.ExtensionResponse
	if err := runtimeHarness.UDSJSON(ctx, http.MethodGet, "/api/extensions/"+url.PathEscape(name), nil, &udsStatus); err != nil {
		t.Fatalf("UDSJSON(/api/extensions/%s) error = %v", name, err)
	}
	if !reflect.DeepEqual(httpStatus.Extension, udsStatus.Extension) {
		t.Fatalf("HTTP extension = %#v, want UDS parity %#v", httpStatus.Extension, udsStatus.Extension)
	}
}

func TestUDSTransportSettingsMutationsRemainPrivilegedWhenHTTPIsNonLoopback(t *testing.T) {
	acpmock.RequireDriver(t)
	t.Parallel()

	runtimeHarness := e2etest.StartRuntimeHarness(t, e2etest.RuntimeHarnessOptions{
		ConfigSeed: e2etest.ConfigSeedOptions{
			Host: "0.0.0.0",
		},
	})

	clients, err := runtimeHarness.TransportClients()
	if err != nil {
		t.Fatalf("TransportClients() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	workspaceID := runtimeHarness.WorkspaceID
	putPath := "/api/settings/mcp-servers/server-a?scope=workspace&workspace_id=" +
		url.QueryEscape(workspaceID) + "&target=sidecar"
	putBody, err := json.Marshal(aghcontract.PutSettingsMCPServerRequest{
		Server: aghcontract.SettingsMCPServerPayload{
			Name:    "server-a",
			Command: "mcpd",
			Args:    []string{"serve"},
		},
	})
	if err != nil {
		t.Fatalf("json.Marshal(put settings body) error = %v", err)
	}

	httpPutResp := mustUnixRequest(
		t,
		clients.HTTPClient,
		http.MethodPut,
		transportSettingsHTTPURL(runtimeHarness, putPath),
		putBody,
		nil,
	)
	if httpPutResp.StatusCode != http.StatusForbidden {
		body, _ := io.ReadAll(httpPutResp.Body)
		_ = httpPutResp.Body.Close()
		t.Fatalf("HTTP PUT settings status = %d, want %d; body=%s", httpPutResp.StatusCode, http.StatusForbidden, string(body))
	}
	var forbidden aghcontract.ErrorPayload
	decodeHTTPJSON(t, httpPutResp, &forbidden)

	var udsMutation aghcontract.MutationResult
	if err := runtimeHarness.UDSJSON(
		ctx,
		http.MethodPut,
		putPath,
		aghcontract.PutSettingsMCPServerRequest{
			Server: aghcontract.SettingsMCPServerPayload{
				Name:    "server-a",
				Command: "mcpd",
				Args:    []string{"serve"},
			},
		},
		&udsMutation,
	); err != nil {
		t.Fatalf("UDSJSON(PUT %s) error = %v", putPath, err)
	}
	if udsMutation.Scope != aghcontract.SettingsScopeWorkspace || udsMutation.WorkspaceID != workspaceID {
		t.Fatalf("UDS mutation = %#v, want workspace-scoped result", udsMutation)
	}

	listPath := "/api/settings/mcp-servers?scope=workspace&workspace_id=" + url.QueryEscape(workspaceID)
	httpListResp := mustUnixRequest(
		t,
		clients.HTTPClient,
		http.MethodGet,
		transportSettingsHTTPURL(runtimeHarness, listPath),
		nil,
		nil,
	)
	if httpListResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpListResp.Body)
		_ = httpListResp.Body.Close()
		t.Fatalf("HTTP GET settings list status = %d, want %d; body=%s", httpListResp.StatusCode, http.StatusOK, string(body))
	}
	var httpList aghcontract.SettingsMCPServersResponse
	decodeHTTPJSON(t, httpListResp, &httpList)

	var udsList aghcontract.SettingsMCPServersResponse
	if err := runtimeHarness.UDSJSON(ctx, http.MethodGet, listPath, nil, &udsList); err != nil {
		t.Fatalf("UDSJSON(GET %s) error = %v", listPath, err)
	}
	if !reflect.DeepEqual(httpList, udsList) {
		t.Fatalf("HTTP mcp list = %#v, want UDS parity %#v", httpList, udsList)
	}

	deletePath := "/api/settings/mcp-servers/server-a?scope=workspace&workspace_id=" +
		url.QueryEscape(workspaceID) + "&target=sidecar"
	var deleteResult aghcontract.MutationResult
	if err := runtimeHarness.UDSJSON(ctx, http.MethodDelete, deletePath, nil, &deleteResult); err != nil {
		t.Fatalf("UDSJSON(DELETE %s) error = %v", deletePath, err)
	}
	if deleteResult.Scope != aghcontract.SettingsScopeWorkspace || deleteResult.WorkspaceID != workspaceID {
		t.Fatalf("deleteResult = %#v, want workspace-scoped delete result", deleteResult)
	}

	httpListResp = mustUnixRequest(
		t,
		clients.HTTPClient,
		http.MethodGet,
		transportSettingsHTTPURL(runtimeHarness, listPath),
		nil,
		nil,
	)
	if httpListResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpListResp.Body)
		_ = httpListResp.Body.Close()
		t.Fatalf("HTTP GET settings list after delete status = %d, want %d; body=%s", httpListResp.StatusCode, http.StatusOK, string(body))
	}
	var httpListAfterDelete aghcontract.SettingsMCPServersResponse
	decodeHTTPJSON(t, httpListResp, &httpListAfterDelete)

	var udsListAfterDelete aghcontract.SettingsMCPServersResponse
	if err := runtimeHarness.UDSJSON(ctx, http.MethodGet, listPath, nil, &udsListAfterDelete); err != nil {
		t.Fatalf("UDSJSON(GET %s after delete) error = %v", listPath, err)
	}
	if !reflect.DeepEqual(httpListAfterDelete, udsListAfterDelete) {
		t.Fatalf(
			"HTTP mcp list after delete = %#v, want UDS parity %#v",
			httpListAfterDelete,
			udsListAfterDelete,
		)
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

func transportSettingsHTTPURL(harness *e2etest.RuntimeHarness, path string) string {
	return fmt.Sprintf("http://127.0.0.1:%d%s", harness.Config.HTTP.Port, path)
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

func udsSSEContainsEvent(records []e2etest.SSEEvent, want string) bool {
	for _, record := range records {
		if record.Event == want {
			return true
		}
	}
	return false
}

func udsSessionEventsContainType(events []aghcontract.SessionEventPayload, want string) bool {
	for _, event := range events {
		var payload struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(event.Content, &payload); err == nil && payload.Type == want {
			return true
		}
	}
	return false
}

func joinTransportTranscript(messages []transcript.Message) string {
	parts := make([]string, 0, len(messages))
	for _, message := range messages {
		if text := strings.TrimSpace(message.Content); text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "\n")
}

func transportMockFixturePath(t testing.TB, name string) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "testutil", "acpmock", "testdata", name)
}
