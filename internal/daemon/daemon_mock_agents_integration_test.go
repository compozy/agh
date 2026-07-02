//go:build integration && !windows

package daemon

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	acpsdk "github.com/coder/acp-go-sdk"
	aghcontract "github.com/compozy/agh/internal/api/contract"
	mcppkg "github.com/compozy/agh/internal/mcp"
	"github.com/compozy/agh/internal/testutil/acpmock"
	e2etest "github.com/compozy/agh/internal/testutil/e2e"
	toolspkg "github.com/compozy/agh/internal/tools"
	mcpclient "github.com/mark3labs/mcp-go/client"
	sdkmcp "github.com/mark3labs/mcp-go/mcp"
)

func TestDaemonE2EFixtureBackedMockAgentLaunchesThroughNormalAgentDefinition(t *testing.T) {
	acpmock.RequireDriver(t)
	t.Parallel()

	harness := e2etest.StartRuntimeHarness(t, e2etest.RuntimeHarnessOptions{
		MockAgents: []e2etest.MockAgentSpec{{
			FixturePath:  mockFixturePath(t, "multi_agent_fixture.json"),
			FixtureAgent: "alpha",
			AgentName:    "mock-alpha",
		}},
	})
	registration, ok := harness.MockAgentRegistration("mock-alpha")
	if !ok {
		t.Fatal("MockAgentRegistration(mock-alpha) = missing, want present")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	session := createFixtureBackedSession(t, ctx, harness, "mock-alpha", "launch-alpha")
	stream, err := harness.PromptSession(ctx, session.ID, "hello alpha")
	if err != nil {
		t.Fatalf("PromptSession() error = %v", err)
	}
	if len(stream) == 0 {
		t.Fatal("PromptSession() stream = empty, want mock agent updates")
	}

	transcriptResp, err := harness.SessionTranscript(ctx, session.ID)
	if err != nil {
		t.Fatalf("SessionTranscript() error = %v", err)
	}
	gotTranscript := joinTranscriptContent(transcriptResp.Messages)
	if !strings.Contains(gotTranscript, "alpha says hi") || !strings.Contains(gotTranscript, "bridge-alpha") {
		t.Fatalf("transcript = %q, want alpha assistant and bridge content", gotTranscript)
	}

	if err := harness.CaptureSessionTranscript(ctx, session.ID); err != nil {
		t.Fatalf("CaptureSessionTranscript() error = %v", err)
	}
	if err := harness.CaptureSessionEvents(ctx, session.ID); err != nil {
		t.Fatalf("CaptureSessionEvents() error = %v", err)
	}
	if err := harness.CaptureMockAgentDiagnostics(registration); err != nil {
		t.Fatalf("CaptureMockAgentDiagnostics() error = %v", err)
	}

	providerCallsPath, ok := harness.Artifacts.ArtifactPath(e2etest.ArtifactKindProviderCalls)
	if !ok {
		t.Fatal("ArtifactPath(provider_calls) = missing, want present")
	}
	providerCalls, err := os.ReadFile(providerCallsPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", providerCallsPath, err)
	}
	if !strings.Contains(string(providerCalls), "alpha-hello") {
		t.Fatalf("provider_calls artifact = %s, want alpha diagnostics", string(providerCalls))
	}
}

func TestDaemonE2EMockAgentsRemainIsolated(t *testing.T) {
	acpmock.RequireDriver(t)
	t.Parallel()

	fixturePath := mockFixturePath(t, "multi_agent_fixture.json")
	harness := e2etest.StartRuntimeHarness(t, e2etest.RuntimeHarnessOptions{
		MockAgents: []e2etest.MockAgentSpec{
			{
				FixturePath:  fixturePath,
				FixtureAgent: "alpha",
				AgentName:    "mock-alpha",
			},
			{
				FixturePath:  fixturePath,
				FixtureAgent: "beta",
				AgentName:    "mock-beta",
			},
		},
	})
	alphaReg, ok := harness.MockAgentRegistration("mock-alpha")
	if !ok {
		t.Fatal("MockAgentRegistration(mock-alpha) = missing, want present")
	}
	betaReg, ok := harness.MockAgentRegistration("mock-beta")
	if !ok {
		t.Fatal("MockAgentRegistration(mock-beta) = missing, want present")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	alphaSession := createFixtureBackedSession(t, ctx, harness, "mock-alpha", "alpha-session")
	if _, err := harness.PromptSession(ctx, alphaSession.ID, "hello alpha"); err != nil {
		t.Fatalf("PromptSession(alpha) error = %v", err)
	}
	betaSession := createFixtureBackedSession(t, ctx, harness, "mock-beta", "beta-session")
	if _, err := harness.PromptSession(ctx, betaSession.ID, "hello beta"); err != nil {
		t.Fatalf("PromptSession(beta) error = %v", err)
	}

	alphaTranscript, err := harness.SessionTranscript(ctx, alphaSession.ID)
	if err != nil {
		t.Fatalf("SessionTranscript(alpha) error = %v", err)
	}
	betaTranscript, err := harness.SessionTranscript(ctx, betaSession.ID)
	if err != nil {
		t.Fatalf("SessionTranscript(beta) error = %v", err)
	}

	alphaContent := joinTranscriptContent(alphaTranscript.Messages)
	betaContent := joinTranscriptContent(betaTranscript.Messages)
	if !strings.Contains(alphaContent, "alpha says hi") || strings.Contains(alphaContent, "beta only") {
		t.Fatalf("alpha transcript = %q, want only alpha content", alphaContent)
	}
	if !strings.Contains(betaContent, "beta only") || strings.Contains(betaContent, "alpha says hi") {
		t.Fatalf("beta transcript = %q, want only beta content", betaContent)
	}

	alphaDiagnostics, err := acpmock.ReadDiagnostics(alphaReg.DiagnosticsPath)
	if err != nil {
		t.Fatalf("ReadDiagnostics(alpha) error = %v", err)
	}
	betaDiagnostics, err := acpmock.ReadDiagnostics(betaReg.DiagnosticsPath)
	if err != nil {
		t.Fatalf("ReadDiagnostics(beta) error = %v", err)
	}
	alphaPromptDiagnostics := acpmock.PromptDiagnostics(alphaDiagnostics)
	betaPromptDiagnostics := acpmock.PromptDiagnostics(betaDiagnostics)
	if len(alphaPromptDiagnostics) != 1 || alphaPromptDiagnostics[0].AgentName != "alpha" {
		t.Fatalf("alpha diagnostics = %#v, want one alpha record", alphaDiagnostics)
	}
	if len(betaPromptDiagnostics) != 1 || betaPromptDiagnostics[0].AgentName != "beta" {
		t.Fatalf("beta diagnostics = %#v, want one beta record", betaDiagnostics)
	}
}

func TestDaemonE2EToolPermissionFixtureEventsSurface(t *testing.T) {
	acpmock.RequireDriver(t)
	t.Parallel()

	fixturePath := mockFixturePath(t, "tool_permission_fixture.json")

	harness := e2etest.StartRuntimeHarness(t, e2etest.RuntimeHarnessOptions{
		MockAgents: []e2etest.MockAgentSpec{
			{
				FixturePath:  fixturePath,
				FixtureAgent: "golden",
				AgentName:    "mock-golden",
			},
		},
	})
	registration, ok := harness.MockAgentRegistration("mock-golden")
	if !ok {
		t.Fatal("MockAgentRegistration(mock-golden) = missing, want present")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	session := createFixtureBackedSession(t, ctx, harness, "mock-golden", "golden-session")
	httpStream, err := harness.PromptSessionHTTPWithEvents(
		ctx,
		session.ID,
		"exercise golden",
		func(event e2etest.SSEEvent) error {
			requestID, ok := permissionRequestIDFromSSE(event)
			if !ok {
				return nil
			}
			return harness.ApproveSessionPermission(ctx, session.ID, aghcontract.ApproveSessionRequest{
				RequestID: requestID,
				Decision:  "allow-always",
			})
		},
	)
	if err != nil {
		t.Fatalf("PromptSessionHTTPWithEvents() error = %v", err)
	}
	if !streamContainsPermission(httpStream) {
		t.Fatalf("HTTP stream = %#v, want permission SSE event", httpStream)
	}

	eventsResp, err := harness.SessionEvents(ctx, session.ID)
	if err != nil {
		t.Fatalf("SessionEvents() error = %v", err)
	}
	events := decodeAgentEvents(t, eventsResp.Events)
	if !containsAgentEvent(events, aghcontract.AgentEventPayload{
		Type:       "tool_call",
		Title:      "Inspect fixture",
		ToolCallID: "tool-1",
	}) {
		t.Fatalf("events = %#v, want tool_call event", events)
	}
	if !containsAgentEvent(events, aghcontract.AgentEventPayload{
		Type:       "tool_result",
		Title:      "Inspect fixture",
		ToolCallID: "tool-1",
	}) {
		t.Fatalf("events = %#v, want tool_result event", events)
	}
	if !containsAgentEvent(events, aghcontract.AgentEventPayload{
		Type:     "permission",
		Resource: "danger.txt",
	}) {
		t.Fatalf("events = %#v, want permission event", events)
	}
	if !containsAgentEvent(events, aghcontract.AgentEventPayload{
		Type:     "permission",
		Resource: "danger.txt",
		Decision: "allow-always",
	}) {
		t.Fatalf("events = %#v, want approved permission event", events)
	}
	if !containsAgentEvent(events, aghcontract.AgentEventPayload{
		Type: "agent_message",
		Text: "allow-always",
	}) {
		t.Fatalf("events = %#v, want allow-always assistant message", events)
	}

	if err := harness.CaptureMockAgentDiagnostics(registration); err != nil {
		t.Fatalf("CaptureMockAgentDiagnostics() error = %v", err)
	}
}

func TestDaemonE2EHostedMCPProjectsAndCallsNonBootstrapNativeTool(t *testing.T) {
	acpmock.RequireDriver(t)
	t.Parallel()

	harness := e2etest.StartRuntimeHarness(t, e2etest.RuntimeHarnessOptions{
		EnableNetwork: true,
		MockAgents: []e2etest.MockAgentSpec{
			{
				FixturePath:  mockFixturePath(t, "hosted_native_tools_fixture.json"),
				FixtureAgent: "hosted-native",
				AgentName:    "mock-hosted-native",
			},
		},
	})
	registration, ok := harness.MockAgentRegistration("mock-hosted-native")
	if !ok {
		t.Fatal("MockAgentRegistration(mock-hosted-native) = missing, want present")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_ = createFixtureBackedSession(t, ctx, harness, "mock-hosted-native", "hosted-native-session")
	diagnostics, err := acpmock.ReadDiagnostics(registration.DiagnosticsPath)
	if err != nil {
		t.Fatalf("ReadDiagnostics(hosted-native) error = %v", err)
	}
	hostedServer := requireHostedMCPStdioServer(t, diagnostics)
	client := startHostedMCPClient(t, hostedServer)
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			t.Fatalf("Close(hosted MCP client) error = %v", closeErr)
		}
	}()

	var init sdkmcp.InitializeRequest
	init.Params.ProtocolVersion = sdkmcp.LATEST_PROTOCOL_VERSION
	init.Params.ClientInfo = sdkmcp.Implementation{Name: "agh-hosted-e2e", Version: "1.0.0"}
	if _, err := client.Initialize(ctx, init); err != nil {
		t.Fatalf("Initialize(hosted MCP client) error = %v", err)
	}

	list, err := client.ListTools(ctx, sdkmcp.ListToolsRequest{})
	if err != nil {
		t.Fatalf("ListTools(hosted MCP) error = %v", err)
	}
	networkToolID := toolspkg.ToolIDNetworkChannelCreate.String()
	if !sdkToolListContains(list.Tools, networkToolID) {
		t.Fatalf("hosted MCP tools = %#v, want non-bootstrap tool %s", sdkToolNames(list.Tools), networkToolID)
	}

	channelName := "hostednative"
	var call sdkmcp.CallToolRequest
	call.Params.Name = networkToolID
	call.Params.Arguments = map[string]any{
		"workspace_id": harness.WorkspaceID,
		"channel":      channelName,
		"purpose":      "Runtime E2E hosted native tool access",
	}
	result, err := client.CallTool(ctx, call)
	if err != nil {
		t.Fatalf("CallTool(%s) error = %v", networkToolID, err)
	}
	if result == nil || result.IsError {
		t.Fatalf("CallTool(%s) result = %#v, want successful result", networkToolID, result)
	}
	structured, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("Marshal(CallTool structuredContent) error = %v", err)
	}
	if !strings.Contains(string(structured), channelName) {
		t.Fatalf("CallTool structuredContent = %s, want channel %q", structured, channelName)
	}

	channel, err := harness.NetworkChannel(ctx, channelName)
	if err != nil {
		t.Fatalf("NetworkChannel(%q) error = %v", channelName, err)
	}
	if channel.Channel != channelName || channel.Purpose != "Runtime E2E hosted native tool access" {
		t.Fatalf("NetworkChannel(%q) = %#v, want hosted native purpose", channelName, channel)
	}
	if err := harness.CaptureMockAgentDiagnostics(registration); err != nil {
		t.Fatalf("CaptureMockAgentDiagnostics() error = %v", err)
	}
	if err := harness.CaptureNetworkArtifacts(ctx, channelName); err != nil {
		t.Fatalf("CaptureNetworkArtifacts(%q) error = %v", channelName, err)
	}
}

func requireHostedMCPStdioServer(
	t testing.TB,
	records []acpmock.DiagnosticsRecord,
) acpsdk.McpServerStdio {
	t.Helper()

	for _, record := range records {
		if record.LifecycleEvent != "session_new" {
			continue
		}
		for _, server := range record.MCPServers {
			if server.Stdio == nil || server.Stdio.Name != mcppkg.HostedServerName {
				continue
			}
			return *server.Stdio
		}
	}
	t.Fatalf("diagnostics = %#v, want session_new %s stdio MCP server", records, mcppkg.HostedServerName)
	return acpsdk.McpServerStdio{}
}

func startHostedMCPClient(
	t testing.TB,
	stdio acpsdk.McpServerStdio,
) *mcpclient.Client {
	t.Helper()

	if strings.TrimSpace(stdio.Command) == "" {
		t.Fatalf("hosted MCP stdio server = %#v, want command", stdio)
	}
	client, err := mcpclient.NewStdioMCPClientWithOptions(
		stdio.Command,
		hostedMCPStdioEnv(stdio),
		append([]string(nil), stdio.Args...),
	)
	if err != nil {
		t.Fatalf("NewStdioMCPClientWithOptions(%q) error = %v", stdio.Command, err)
	}
	return client
}

func hostedMCPStdioEnv(stdio acpsdk.McpServerStdio) []string {
	env := make([]string, 0, len(stdio.Env))
	for _, entry := range stdio.Env {
		name := strings.TrimSpace(entry.Name)
		if name == "" {
			continue
		}
		env = append(env, name+"="+entry.Value)
	}
	return env
}

func sdkToolListContains(tools []sdkmcp.Tool, name string) bool {
	for _, tool := range tools {
		if tool.Name == name {
			return true
		}
	}
	return false
}

func sdkToolNames(tools []sdkmcp.Tool) []string {
	names := make([]string, 0, len(tools))
	for _, tool := range tools {
		names = append(names, tool.Name)
	}
	return names
}

func permissionRequestIDFromSSE(event e2etest.SSEEvent) (string, bool) {
	if event.Event != "permission" || len(event.Data) == 0 {
		return "", false
	}

	var envelope struct {
		Type string `json:"type"`
		Data struct {
			RequestID string `json:"request_id"`
			Decision  string `json:"decision,omitempty"`
		} `json:"data"`
	}
	if err := json.Unmarshal(event.Data, &envelope); err != nil {
		return "", false
	}
	if envelope.Type != "data-agh-permission" || envelope.Data.Decision != "" || envelope.Data.RequestID == "" {
		return "", false
	}
	return envelope.Data.RequestID, true
}

func streamContainsPermission(events []e2etest.SSEEvent) bool {
	for _, event := range events {
		if event.Event == "permission" {
			return true
		}
	}
	return false
}

func decodeAgentEvents(
	t testing.TB,
	events []aghcontract.SessionEventPayload,
) []aghcontract.AgentEventPayload {
	t.Helper()

	decoded := make([]aghcontract.AgentEventPayload, 0, len(events))
	for _, event := range events {
		var payload aghcontract.AgentEventPayload
		if err := json.Unmarshal(event.Content, &payload); err != nil {
			t.Fatalf("json.Unmarshal(session event %q) error = %v", event.ID, err)
		}
		decoded = append(decoded, payload)
	}
	return decoded
}

func containsAgentEvent(events []aghcontract.AgentEventPayload, want aghcontract.AgentEventPayload) bool {
	for _, event := range events {
		if want.Type != "" && event.Type != want.Type {
			continue
		}
		if want.Text != "" && event.Text != want.Text {
			continue
		}
		if want.Title != "" && event.Title != want.Title {
			continue
		}
		if want.ToolCallID != "" && event.ToolCallID != want.ToolCallID {
			continue
		}
		if want.Resource != "" && event.Resource != want.Resource {
			continue
		}
		if want.Decision != "" && event.Decision != want.Decision {
			continue
		}
		return true
	}
	return false
}
