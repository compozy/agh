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
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	aghcontract "github.com/pedronauck/agh/internal/api/contract"
	apispec "github.com/pedronauck/agh/internal/api/spec"
	automationpkg "github.com/pedronauck/agh/internal/automation"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/testutil/acpmock"
	e2etest "github.com/pedronauck/agh/internal/testutil/e2e"
	"github.com/pedronauck/agh/internal/transcript"
)

const (
	transportUDSApprovalAgent    = "transport-uds-approver"
	transportUDSAutomationAgent  = "transport-uds-automation-runner"
	transportUDSFaultyAgent      = "transport-uds-faulty-runner"
	transportUDSObserveAgent     = "transport-uds-observe"
	transportUDSOverrideProvider = "qa-transport-override"
)

func TestUDSTransportApprovalFlowMatchesHTTP(t *testing.T) {
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

			resp := mustUnixRequest(
				t,
				clients.UDSClient,
				http.MethodPost,
				clients.UDSBaseURL+"/api/workspaces/ws-workspace/sessions/"+url.PathEscape(session.ID)+"/approve",
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
			if err := e2etest.ValidateUDSApprovalResponse(resp.StatusCode, body); err != nil {
				return err
			}
			return nil
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
}

func TestUDSTransportSessionProviderCreateReadMatchesHTTP(t *testing.T) {
	acpmock.RequireDriver(t)
	t.Parallel()

	runtimeHarness := e2etest.StartRuntimeHarness(t, e2etest.RuntimeHarnessOptions{
		MockAgents: []e2etest.MockAgentSpec{{
			FixturePath:  transportMockFixturePath(t, "automation_task_fixture.json"),
			FixtureAgent: "automation-runner",
			AgentName:    transportUDSAutomationAgent,
		}},
	})
	registration, ok := runtimeHarness.MockAgentRegistration(transportUDSAutomationAgent)
	if !ok {
		t.Fatalf("MockAgentRegistration(%q) not found", transportUDSAutomationAgent)
	}
	provider := registration.Provider

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	var created aghcontract.SessionResponse
	if err := runtimeHarness.UDSJSON(ctx, http.MethodPost, "/api/sessions", aghcontract.CreateSessionRequest{
		AgentName:     transportUDSAutomationAgent,
		Provider:      provider,
		WorkspacePath: runtimeHarness.WorkspaceRoot,
	}, &created); err != nil {
		t.Fatalf("UDS create session error = %v", err)
	}
	if created.Session.ID == "" {
		t.Fatal("UDS create session id = empty, want non-empty")
	}
	if created.Session.Provider != provider {
		t.Fatalf("UDS create provider = %q, want %q", created.Session.Provider, provider)
	}

	var udsDetail aghcontract.SessionResponse
	if err := runtimeHarness.UDSJSON(
		ctx,
		http.MethodGet,
		"/api/workspaces/ws-workspace/sessions/"+url.PathEscape(created.Session.ID),
		nil,
		&udsDetail,
	); err != nil {
		t.Fatalf("UDS get session error = %v", err)
	}
	if udsDetail.Session.Provider != created.Session.Provider {
		t.Fatalf(
			"UDS detail provider = %q, want create provider %q",
			udsDetail.Session.Provider,
			created.Session.Provider,
		)
	}

	var httpDetail aghcontract.SessionResponse
	if err := runtimeHarness.HTTPJSON(
		ctx,
		http.MethodGet,
		"/api/workspaces/ws-workspace/sessions/"+url.PathEscape(created.Session.ID),
		nil,
		&httpDetail,
	); err != nil {
		t.Fatalf("HTTP get session error = %v", err)
	}
	if httpDetail.Session.Provider != created.Session.Provider {
		t.Fatalf(
			"HTTP detail provider = %q, want UDS create provider %q",
			httpDetail.Session.Provider,
			created.Session.Provider,
		)
	}
}

func TestUDSTransportResumeMissingProviderReturnsExplicitBadRequest(t *testing.T) {
	acpmock.RequireDriver(t)
	t.Parallel()

	runtimeHarness := e2etest.StartRuntimeHarness(t, e2etest.RuntimeHarnessOptions{
		MockAgents: []e2etest.MockAgentSpec{{
			FixturePath:  transportMockFixturePath(t, "automation_task_fixture.json"),
			FixtureAgent: "automation-runner",
			AgentName:    transportUDSAutomationAgent,
		}},
	})
	registration, ok := runtimeHarness.MockAgentRegistration(transportUDSAutomationAgent)
	if !ok {
		t.Fatalf("MockAgentRegistration(%q) not found", transportUDSAutomationAgent)
	}

	writeTransportProviderOverrideConfig(
		t,
		runtimeHarness.WorkspaceRoot,
		transportUDSOverrideProvider,
		registration.Command,
		true,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	var created aghcontract.SessionResponse
	if err := runtimeHarness.UDSJSON(ctx, http.MethodPost, "/api/sessions", aghcontract.CreateSessionRequest{
		AgentName:     transportUDSAutomationAgent,
		Provider:      transportUDSOverrideProvider,
		WorkspacePath: runtimeHarness.WorkspaceRoot,
	}, &created); err != nil {
		t.Fatalf("UDS create session error = %v", err)
	}

	stopResp := mustUnixRequest(
		t,
		runtimeHarness.UDSClient,
		http.MethodPost,
		runtimeHarness.UDSURL("/api/workspaces/ws-workspace/sessions/"+url.PathEscape(created.Session.ID)+"/stop"),
		nil,
		nil,
	)
	if stopResp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(stopResp.Body)
		_ = stopResp.Body.Close()
		t.Fatalf(
			"UDS stop session status = %d, want %d; body=%s",
			stopResp.StatusCode,
			http.StatusNoContent,
			string(body),
		)
	}
	_ = stopResp.Body.Close()

	writeTransportProviderOverrideConfig(
		t,
		runtimeHarness.WorkspaceRoot,
		transportUDSOverrideProvider,
		registration.Command,
		false,
	)

	resumeResp := mustUnixRequest(
		t,
		runtimeHarness.UDSClient,
		http.MethodPost,
		runtimeHarness.UDSURL("/api/workspaces/ws-workspace/sessions/"+url.PathEscape(created.Session.ID)+"/resume"),
		nil,
		nil,
	)
	body, err := io.ReadAll(resumeResp.Body)
	closeErr := resumeResp.Body.Close()
	if err != nil {
		t.Fatalf("read UDS resume body error = %v", err)
	}
	if closeErr != nil {
		t.Fatalf("close UDS resume body error = %v", closeErr)
	}
	if resumeResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("UDS resume status = %d, want %d; body=%s", resumeResp.StatusCode, http.StatusBadRequest, string(body))
	}
	if !strings.Contains(string(body), created.Session.ID) {
		t.Fatalf("UDS resume body = %s, want session id %q", string(body), created.Session.ID)
	}
	if !strings.Contains(string(body), transportUDSOverrideProvider) {
		t.Fatalf("UDS resume body = %s, want provider %q", string(body), transportUDSOverrideProvider)
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

func TestUDSTransportObserveHarnessLifecycleParityMatchesHTTP(t *testing.T) {
	acpmock.RequireDriver(t)
	t.Parallel()

	runtimeHarness := e2etest.StartRuntimeHarness(t, e2etest.RuntimeHarnessOptions{
		MockAgents: []e2etest.MockAgentSpec{{
			FixturePath:  transportMockFixturePath(t, "multi_agent_fixture.json"),
			FixtureAgent: "alpha",
			AgentName:    transportUDSObserveAgent,
		}},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	session, err := runtimeHarness.CreateSession(ctx, aghcontract.CreateSessionRequest{
		AgentName:     transportUDSObserveAgent,
		WorkspacePath: runtimeHarness.WorkspaceRoot,
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	stream, err := runtimeHarness.PromptSessionHTTP(ctx, session.ID, "hello alpha")
	if err != nil {
		t.Fatalf("PromptSessionHTTP() error = %v", err)
	}
	if len(stream) == 0 {
		t.Fatal("prompt stream = empty, want streamed events")
	}

	transcriptResp, err := runtimeHarness.SessionTranscript(ctx, session.ID)
	if err != nil {
		t.Fatalf("SessionTranscript() error = %v", err)
	}
	if !strings.Contains(joinTransportTranscript(transcriptResp.Messages), "alpha says hi") {
		t.Fatalf("transcript = %#v, want assistant reply", transcriptResp.Messages)
	}

	httpHarnessEvents := waitForTransportListLogs(
		t,
		ctx,
		"waitForHTTPListLogs",
		wantTransportObserveHarnessTypes(),
		func(fetchCtx context.Context) ([]aghcontract.LogEventPayload, error) {
			var response aghcontract.LogsListResponse
			err := runtimeHarness.HTTPJSON(
				fetchCtx,
				http.MethodGet,
				"/api/logs?workspace_id=ws-workspace?session_id="+url.QueryEscape(session.ID)+"&limit=20",
				nil,
				&response,
			)
			return response.Events, err
		},
	)
	udsHarnessEvents := waitForTransportListLogs(
		t,
		ctx,
		"waitForUDSListLogs",
		wantTransportObserveHarnessTypes(),
		func(fetchCtx context.Context) ([]aghcontract.LogEventPayload, error) {
			var response aghcontract.LogsListResponse
			err := runtimeHarness.UDSJSON(
				fetchCtx,
				http.MethodGet,
				"/api/logs?workspace_id=ws-workspace?session_id="+url.QueryEscape(session.ID)+"&limit=20",
				nil,
				&response,
			)
			return response.Events, err
		},
	)

	if !reflect.DeepEqual(httpHarnessEvents, udsHarnessEvents) {
		t.Fatalf("HTTP harness events = %#v, want UDS parity %#v", httpHarnessEvents, udsHarnessEvents)
	}
	if got, want := logEventTypes(httpHarnessEvents), wantTransportObserveHarnessTypes(); !slices.Equal(got, want) {
		t.Fatalf("harness event types = %#v, want %#v", got, want)
	}
	if !strings.Contains(httpHarnessEvents[0].Summary, "surface=startup") {
		t.Fatalf("startup summary = %q, want startup surface", httpHarnessEvents[0].Summary)
	}
	if !strings.Contains(httpHarnessEvents[1].Summary, "selected=") {
		t.Fatalf("section summary = %q, want selected sections", httpHarnessEvents[1].Summary)
	}
	if !strings.Contains(httpHarnessEvents[2].Summary, "surface=turn") {
		t.Fatalf("turn summary = %q, want turn surface", httpHarnessEvents[2].Summary)
	}
	if !strings.Contains(httpHarnessEvents[3].Summary, "augmenter=durable_memory") {
		t.Fatalf("augmenter summary = %q, want durable memory metadata", httpHarnessEvents[3].Summary)
	}
	if !strings.Contains(httpHarnessEvents[4].Summary, "augmenter=skills") {
		t.Fatalf("augmenter summary = %q, want skills metadata", httpHarnessEvents[4].Summary)
	}
	if !strings.Contains(httpHarnessEvents[5].Summary, "augmenter=situation") {
		t.Fatalf("augmenter summary = %q, want situation metadata", httpHarnessEvents[5].Summary)
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
		t.Fatalf(
			"HTTP extension list status = %d, want %d; body=%s",
			httpListResp.StatusCode,
			http.StatusOK,
			string(body),
		)
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
	if err := runtimeHarness.UDSJSON(
		ctx,
		http.MethodGet,
		"/api/extensions/"+url.PathEscape(name),
		nil,
		&udsStatus,
	); err != nil {
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
		t.Fatalf(
			"HTTP PUT settings status = %d, want %d; body=%s",
			httpPutResp.StatusCode,
			http.StatusForbidden,
			string(body),
		)
	}
	var forbidden aghcontract.ErrorPayload
	decodeHTTPJSON(t, httpPutResp, &forbidden)

	var udsMutation aghcontract.SettingsGlobalWorkspaceCollectionMutationResult
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
	if udsMutation.Scope != aghcontract.SettingsWorkspaceScopeWorkspace || udsMutation.WorkspaceID != workspaceID {
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
	if httpListResp.StatusCode != http.StatusForbidden {
		body, _ := io.ReadAll(httpListResp.Body)
		_ = httpListResp.Body.Close()
		t.Fatalf(
			"HTTP GET settings list status = %d, want %d; body=%s",
			httpListResp.StatusCode,
			http.StatusForbidden,
			string(body),
		)
	}
	var listForbidden aghcontract.ErrorPayload
	decodeHTTPJSON(t, httpListResp, &listForbidden)

	var udsList aghcontract.SettingsMCPServersResponse
	if err := runtimeHarness.UDSJSON(ctx, http.MethodGet, listPath, nil, &udsList); err != nil {
		t.Fatalf("UDSJSON(GET %s) error = %v", listPath, err)
	}
	if !settingsMCPServerPresent(udsList.MCPServers, "server-a") {
		t.Fatalf("UDS mcp list = %#v, want server-a after UDS mutation", udsList)
	}

	deletePath := "/api/settings/mcp-servers/server-a?scope=workspace&workspace_id=" +
		url.QueryEscape(workspaceID) + "&target=sidecar"
	var deleteResult aghcontract.SettingsGlobalWorkspaceCollectionMutationResult
	if err := runtimeHarness.UDSJSON(ctx, http.MethodDelete, deletePath, nil, &deleteResult); err != nil {
		t.Fatalf("UDSJSON(DELETE %s) error = %v", deletePath, err)
	}
	if deleteResult.Scope != aghcontract.SettingsWorkspaceScopeWorkspace || deleteResult.WorkspaceID != workspaceID {
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
	if httpListResp.StatusCode != http.StatusForbidden {
		body, _ := io.ReadAll(httpListResp.Body)
		_ = httpListResp.Body.Close()
		t.Fatalf(
			"HTTP GET settings list after delete status = %d, want %d; body=%s",
			httpListResp.StatusCode,
			http.StatusForbidden,
			string(body),
		)
	}
	var listAfterDeleteForbidden aghcontract.ErrorPayload
	decodeHTTPJSON(t, httpListResp, &listAfterDeleteForbidden)

	var udsListAfterDelete aghcontract.SettingsMCPServersResponse
	if err := runtimeHarness.UDSJSON(ctx, http.MethodGet, listPath, nil, &udsListAfterDelete); err != nil {
		t.Fatalf("UDSJSON(GET %s after delete) error = %v", listPath, err)
	}
	if settingsMCPServerPresent(udsListAfterDelete.MCPServers, "server-a") {
		t.Fatalf("UDS mcp list after delete = %#v, want server-a removed", udsListAfterDelete)
	}
}

func TestUDSTransportTaskSurfaceMatchesHTTPAndDocumentedSpecOperations(t *testing.T) {
	t.Parallel()

	homePaths := newTestHomePaths(t)
	udsEngine := newTestRouter(t, newTestHandlers(t, stubSessionManager{}, stubObserver{}, homePaths))

	udsRoutes := taskRoutesFromEngine(udsEngine.Routes())
	httpRoutes := documentedTaskRoutesForTransport(apispec.TransportHTTP)
	if !slices.Equal(udsRoutes, httpRoutes) {
		t.Fatalf("UDS task routes = %v, want documented HTTP task routes %v", udsRoutes, httpRoutes)
	}

	specRoutes := documentedTaskRoutesForTransport(apispec.TransportUDS)
	if !slices.Equal(udsRoutes, specRoutes) {
		t.Fatalf("UDS task routes = %v, want documented task routes %v", udsRoutes, specRoutes)
	}
}
func waitForUDSAutomationRun(
	t testing.TB,
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	runID string,
) aghcontract.RunPayload {
	t.Helper()

	return waitForTransportAutomationRun(
		t,
		ctx,
		runID,
		"waitForUDSAutomationRun",
		func(fetchCtx context.Context) (aghcontract.RunPayload, error) {
			return harness.GetAutomationRun(fetchCtx, runID)
		},
	)
}

func transportSettingsHTTPURL(harness *e2etest.RuntimeHarness, path string) string {
	return fmt.Sprintf("http://127.0.0.1:%d%s", harness.Config.HTTP.Port, path)
}

func settingsMCPServerPresent(items []aghcontract.SettingsMCPServerItemPayload, name string) bool {
	for _, item := range items {
		if item.Name == name {
			return true
		}
	}
	return false
}

func waitForCLIAutomationRun(
	t testing.TB,
	ctx context.Context,
	client *e2etest.CLIClient,
	runID string,
) aghcontract.RunPayload {
	t.Helper()

	return waitForTransportAutomationRun(
		t,
		ctx,
		runID,
		"waitForCLIAutomationRun",
		func(fetchCtx context.Context) (aghcontract.RunPayload, error) {
			var run aghcontract.RunPayload
			err := client.RunJSON(fetchCtx, &run, "automation", "runs", "get", runID, "-o", "json")
			return run, err
		},
	)
}

func seedTransportWebhookTrigger(
	t testing.TB,
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
) (aghcontract.TriggerPayload, string) {
	t.Helper()

	state, err := harness.SeedAutomationFixtures(ctx, e2etest.AutomationFixtureSeed{
		Triggers: []aghcontract.CreateTriggerRequest{{
			Scope:              automationpkg.AutomationScopeGlobal,
			Name:               "deploy-review",
			AgentName:          transportUDSAutomationAgent,
			Prompt:             `Review payload {{ index .Data "payload" }} for {{ index .Data "branch" }}`,
			Event:              "webhook",
			EndpointSlug:       "deploy-review",
			WebhookSecretValue: "shared-secret",
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

func taskRoutesFromEngine(routes gin.RoutesInfo) []string {
	filtered := make([]string, 0)
	for _, route := range routes {
		if isDocumentedTaskRoute(route.Path) {
			filtered = append(filtered, route.Method+" "+route.Path)
		}
	}
	sort.Strings(filtered)
	return filtered
}

func documentedTaskRoutesForTransport(transport apispec.Transport) []string {
	routes := make([]string, 0)
	for _, operation := range apispec.Operations() {
		if !slices.Contains(operation.Transports, transport) {
			continue
		}
		if !isDocumentedTaskRoute(operation.Path) {
			continue
		}
		routes = append(routes, operation.Method+" "+normalizeSpecRoutePath(operation.Path))
	}
	sort.Strings(routes)
	return routes
}

func isDocumentedTaskRoute(path string) bool {
	return strings.HasPrefix(path, "/api/tasks") ||
		strings.HasPrefix(path, "/api/task-runs") ||
		strings.HasPrefix(path, "/api/observe/tasks")
}

func normalizeSpecRoutePath(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") && len(part) > 2 {
			parts[i] = ":" + part[1:len(part)-1]
		}
	}
	return strings.Join(parts, "/")
}

func waitForHTTPAutomationRun(
	t testing.TB,
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	runID string,
) aghcontract.RunPayload {
	t.Helper()

	return waitForTransportAutomationRun(
		t,
		ctx,
		runID,
		"waitForHTTPAutomationRun",
		func(fetchCtx context.Context) (aghcontract.RunPayload, error) {
			var response aghcontract.RunResponse
			err := harness.HTTPJSON(
				fetchCtx,
				http.MethodGet,
				"/api/automation/runs/"+url.PathEscape(runID),
				nil,
				&response,
			)
			return response.Run, err
		},
	)
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

func waitForTransportListLogs(
	t testing.TB,
	ctx context.Context,
	label string,
	wantTypes []string,
	fetch func(context.Context) ([]aghcontract.LogEventPayload, error),
) []aghcontract.LogEventPayload {
	t.Helper()

	waitCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()

	var (
		lastErr    error
		lastEvents []aghcontract.LogEventPayload
	)
	for {
		events, err := fetch(waitCtx)
		if err == nil {
			harnessEvents := filterHarnessListLogs(events)
			if slices.Equal(logEventTypes(harnessEvents), wantTypes) {
				return harnessEvents
			}
			lastEvents = harnessEvents
		} else {
			lastErr = err
		}

		select {
		case <-waitCtx.Done():
			t.Fatalf(
				"%s timed out: %v; last error=%v last harness events=%#v",
				label,
				waitCtx.Err(),
				lastErr,
				lastEvents,
			)
		case <-ticker.C:
		}
	}
}

func wantTransportObserveHarnessTypes() []string {
	return []string{
		"harness.context_resolved",
		"harness.section_selected",
		"harness.context_resolved",
		"harness.augmenter_applied",
		"harness.augmenter_applied",
		"harness.augmenter_applied",
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

func filterHarnessListLogs(events []aghcontract.LogEventPayload) []aghcontract.LogEventPayload {
	filtered := make([]aghcontract.LogEventPayload, 0, len(events))
	for _, event := range events {
		if strings.HasPrefix(event.Type, "harness.") {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

func logEventTypes(events []aghcontract.LogEventPayload) []string {
	types := make([]string, 0, len(events))
	for _, event := range events {
		types = append(types, event.Type)
	}
	return types
}

func joinTransportTranscript(messages []transcript.UIMessage) string {
	return transcript.JoinUIMessageText(messages)
}

func transportMockFixturePath(t testing.TB, name string) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "testutil", "acpmock", "testdata", name)
}

func writeTransportProviderOverrideConfig(
	t testing.TB,
	workspaceRoot string,
	providerName string,
	command string,
	includeProvider bool,
) {
	t.Helper()

	configDir := filepath.Join(workspaceRoot, aghconfig.DirName)
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", configDir, err)
	}

	var builder strings.Builder
	if includeProvider {
		builder.WriteString("[providers." + providerName + "]\n")
		builder.WriteString(`command = "`)
		builder.WriteString(escapeTransportConfigString(command))
		builder.WriteString("\"\n")
		builder.WriteString("[providers." + providerName + ".models]\n")
		builder.WriteString(`default = "transport-override-model"` + "\n")
		builder.WriteString("[[providers." + providerName + ".credential_slots]]\n")
		builder.WriteString(`name = "api_key"` + "\n")
		builder.WriteString(`target_env = "TRANSPORT_OVERRIDE_API_KEY"` + "\n")
		builder.WriteString(`secret_ref = "env:TRANSPORT_OVERRIDE_API_KEY"` + "\n")
		builder.WriteString(`kind = "api_key"` + "\n")
		builder.WriteString(`required = false` + "\n")
	}

	configPath := filepath.Join(configDir, aghconfig.ConfigName)
	if err := os.WriteFile(configPath, []byte(builder.String()), 0o600); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", configPath, err)
	}
}

func escapeTransportConfigString(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`)
	return replacer.Replace(strings.TrimSpace(value))
}
