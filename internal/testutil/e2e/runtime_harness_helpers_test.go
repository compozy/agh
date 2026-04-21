package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	aghcontract "github.com/pedronauck/agh/internal/api/contract"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	sessionpkg "github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
	taskpkg "github.com/pedronauck/agh/internal/task"
)

func TestRuntimeHarnessCaptureHelpersPersistArtifacts(t *testing.T) {
	t.Parallel()

	server := newHarnessTestServer(t)

	homePaths := NewHomePaths(t)
	auditTime := time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)
	auditEntry, err := json.Marshal(store.NetworkAuditEntry{
		ID:        "naud-1",
		SessionID: "sess-1",
		Direction: "sent",
		Kind:      "say",
		Channel:   "builders",
		PeerFrom:  "coder.sess-1",
		MessageID: "msg-1",
		Size:      42,
		Timestamp: auditTime,
	})
	if err != nil {
		t.Fatalf("json.Marshal(auditEntry) error = %v", err)
	}
	if err := os.WriteFile(homePaths.NetworkAuditFile, append(auditEntry, '\n'), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", homePaths.NetworkAuditFile, err)
	}

	harness := &RuntimeHarness{
		HomePaths:   homePaths,
		Artifacts:   NewArtifactCollector(t),
		HTTPBaseURL: server.URL,
		HTTPClient:  server.Client(),
		UDSBaseURL:  server.URL,
		UDSClient:   server.Client(),
	}

	metaPath := store.SessionMetaFile(filepath.Join(homePaths.SessionsDir, "sess-1"))
	if err := store.WriteSessionMeta(metaPath, store.SessionMeta{
		ID:          "sess-1",
		AgentName:   "coder",
		WorkspaceID: "ws-1",
		State:       "stopped",
		Environment: &store.SessionEnvironmentMeta{
			EnvironmentID:  "env-1",
			Backend:        "local",
			Profile:        "local-sandbox",
			State:          "stopped",
			RuntimeRootDir: "/workspace",
			RuntimeAdditionalDirs: []string{
				"/workspace/shared",
			},
			ProviderState: json.RawMessage(`{"sandbox":"local"}`),
		},
	}); err != nil {
		t.Fatalf("WriteSessionMeta(%q) error = %v", metaPath, err)
	}

	workspace, err := harness.ResolveWorkspace(testContext(t), "/workspace")
	if err != nil {
		t.Fatalf("ResolveWorkspace() error = %v", err)
	}
	if got, want := workspace.ID, "ws-1"; got != want {
		t.Fatalf("workspace.ID = %q, want %q", got, want)
	}

	gotWorkspace, err := harness.GetWorkspace(testContext(t), "ws-1")
	if err != nil {
		t.Fatalf("GetWorkspace() error = %v", err)
	}
	if got, want := gotWorkspace.RootDir, "/workspace"; got != want {
		t.Fatalf("gotWorkspace.RootDir = %q, want %q", got, want)
	}

	extensions, err := harness.ListExtensions(testContext(t))
	if err != nil {
		t.Fatalf("ListExtensions() error = %v", err)
	}
	if got, want := len(extensions), 1; got != want {
		t.Fatalf("len(extensions) = %d, want %d", got, want)
	}

	installedExtension, err := harness.InstallExtension(testContext(t), aghcontract.InstallExtensionRequest{
		Path:     "/extensions/telegram-reference",
		Checksum: "sha256-demo",
	})
	if err != nil {
		t.Fatalf("InstallExtension() error = %v", err)
	}
	if got, want := installedExtension.Name, "telegram-reference"; got != want {
		t.Fatalf("installedExtension.Name = %q, want %q", got, want)
	}

	gotExtension, err := harness.GetExtension(testContext(t), "telegram-reference")
	if err != nil {
		t.Fatalf("GetExtension() error = %v", err)
	}
	if got, want := gotExtension.State, "registered"; got != want {
		t.Fatalf("gotExtension.State = %q, want %q", got, want)
	}

	enabledExtension, err := harness.EnableExtension(testContext(t), "telegram-reference")
	if err != nil {
		t.Fatalf("EnableExtension() error = %v", err)
	}
	if got, want := enabledExtension.Health, "healthy"; got != want {
		t.Fatalf("enabledExtension.Health = %q, want %q", got, want)
	}

	disabledExtension, err := harness.DisableExtension(testContext(t), "telegram-reference")
	if err != nil {
		t.Fatalf("DisableExtension() error = %v", err)
	}
	if disabledExtension.Enabled {
		t.Fatalf("disabledExtension.Enabled = %v, want false", disabledExtension.Enabled)
	}

	session, err := harness.CreateSession(testContext(t), aghcontract.CreateSessionRequest{
		AgentName:     "coder",
		WorkspacePath: "/workspace",
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if got, want := session.ID, "sess-1"; got != want {
		t.Fatalf("session.ID = %q, want %q", got, want)
	}

	gotSession, err := harness.GetSession(testContext(t), "sess-1")
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	if got, want := gotSession.Environment.Profile, "local"; got != want {
		t.Fatalf("gotSession.Environment.Profile = %q, want %q", got, want)
	}
	resumedSession, err := harness.ResumeSession(testContext(t), "sess-1")
	if err != nil {
		t.Fatalf("ResumeSession() error = %v", err)
	}
	if got, want := resumedSession.State, sessionpkg.StateActive; got != want {
		t.Fatalf("resumedSession.State = %q, want %q", got, want)
	}
	if got, want := resumedSession.Channel, "builders"; got != want {
		t.Fatalf("resumedSession.Channel = %q, want %q", got, want)
	}

	stream, err := harness.PromptSession(testContext(t), "sess-1", "hello world")
	if err != nil {
		t.Fatalf("PromptSession() error = %v", err)
	}
	if got, want := len(stream), 2; got != want {
		t.Fatalf("len(stream) = %d, want %d", got, want)
	}
	if got, want := stream[0].Event, "agent_message"; got != want {
		t.Fatalf("stream[0].Event = %q, want %q", got, want)
	}

	if _, err := harness.SessionTranscript(testContext(t), "sess-1"); err != nil {
		t.Fatalf("SessionTranscript() error = %v", err)
	}
	if _, err := harness.SessionEvents(testContext(t), "sess-1"); err != nil {
		t.Fatalf("SessionEvents() error = %v", err)
	}
	if _, err := harness.NetworkStatus(testContext(t)); err != nil {
		t.Fatalf("NetworkStatus() error = %v", err)
	}
	if _, err := harness.NetworkPeers(testContext(t), "builders"); err != nil {
		t.Fatalf("NetworkPeers() error = %v", err)
	}
	if _, err := harness.NetworkChannels(testContext(t)); err != nil {
		t.Fatalf("NetworkChannels() error = %v", err)
	}
	if _, err := harness.NetworkChannel(testContext(t), "builders"); err != nil {
		t.Fatalf("NetworkChannel() error = %v", err)
	}
	if _, err := harness.NetworkChannelMessages(testContext(t), "builders"); err != nil {
		t.Fatalf("NetworkChannelMessages() error = %v", err)
	}
	if _, err := harness.NetworkInbox(testContext(t), "sess-1"); err != nil {
		t.Fatalf("NetworkInbox() error = %v", err)
	}
	if _, err := harness.NetworkSend(testContext(t), aghcontract.NetworkSendRequest{
		SessionID: "sess-1",
		Channel:   "builders",
		Kind:      "say",
		Body:      json.RawMessage(`{"text":"hello"}`),
	}); err != nil {
		t.Fatalf("NetworkSend() error = %v", err)
	}
	if _, err := harness.CreateNetworkChannel(testContext(t), aghcontract.CreateNetworkChannelRequest{
		Channel:     "builders",
		WorkspaceID: "ws-1",
		AgentNames:  []string{"coder"},
	}); err != nil {
		t.Fatalf("CreateNetworkChannel() error = %v", err)
	}

	createdBridge, err := harness.CreateBridge(testContext(t), aghcontract.CreateBridgeRequest{
		Scope:         bridgepkg.ScopeWorkspace,
		WorkspaceID:   "ws-1",
		Platform:      "telegram",
		ExtensionName: "telegram-reference",
		DisplayName:   "Telegram Runtime",
		Enabled:       false,
		Status:        bridgepkg.BridgeStatusDisabled,
	})
	if err != nil {
		t.Fatalf("CreateBridge() error = %v", err)
	}
	if got, want := createdBridge.Bridge.ID, "brg-1"; got != want {
		t.Fatalf("createdBridge.Bridge.ID = %q, want %q", got, want)
	}

	gotBridge, err := harness.GetBridge(testContext(t), "brg-1")
	if err != nil {
		t.Fatalf("GetBridge() error = %v", err)
	}
	if got, want := gotBridge.Health.RouteCount, 1; got != want {
		t.Fatalf("gotBridge.Health.RouteCount = %d, want %d", got, want)
	}

	enabledBridge, err := harness.EnableBridge(testContext(t), "brg-1")
	if err != nil {
		t.Fatalf("EnableBridge() error = %v", err)
	}
	if !enabledBridge.Bridge.Enabled {
		t.Fatal("enabledBridge.Bridge.Enabled = false, want true")
	}

	restartedBridge, err := harness.RestartBridge(testContext(t), "brg-1")
	if err != nil {
		t.Fatalf("RestartBridge() error = %v", err)
	}
	if got, want := restartedBridge.Health.DeliveryBacklog, 0; got != want {
		t.Fatalf("restartedBridge.Health.DeliveryBacklog = %d, want %d", got, want)
	}

	routes, err := harness.ListBridgeRoutes(testContext(t), "brg-1")
	if err != nil {
		t.Fatalf("ListBridgeRoutes() error = %v", err)
	}
	if got, want := len(routes), 1; got != want {
		t.Fatalf("len(routes) = %d, want %d", got, want)
	}

	binding, err := harness.PutBridgeSecretBinding(
		testContext(t),
		"brg-1",
		"bot_token",
		aghcontract.PutBridgeSecretBindingRequest{
			VaultRef: "env:AGH_TEST_TELEGRAM_TOKEN",
			Kind:     "token",
		},
	)
	if err != nil {
		t.Fatalf("PutBridgeSecretBinding() error = %v", err)
	}
	if got, want := binding.BindingName, "bot_token"; got != want {
		t.Fatalf("binding.BindingName = %q, want %q", got, want)
	}

	bindings, err := harness.ListBridgeSecretBindings(testContext(t), "brg-1")
	if err != nil {
		t.Fatalf("ListBridgeSecretBindings() error = %v", err)
	}
	if got, want := len(bindings), 1; got != want {
		t.Fatalf("len(bindings) = %d, want %d", got, want)
	}

	if err := harness.CaptureSessionTranscript(testContext(t), "sess-1"); err != nil {
		t.Fatalf("CaptureSessionTranscript() error = %v", err)
	}
	if err := harness.CaptureSessionEvents(testContext(t), "sess-1"); err != nil {
		t.Fatalf("CaptureSessionEvents() error = %v", err)
	}
	if err := harness.CaptureSessionEnvironment(testContext(t), "sess-1"); err != nil {
		t.Fatalf("CaptureSessionEnvironment() error = %v", err)
	}
	if err := harness.CaptureNetworkArtifacts(testContext(t), "builders"); err != nil {
		t.Fatalf("CaptureNetworkArtifacts() error = %v", err)
	}
	if err := harness.CaptureAutomationRuns(testContext(t), url.Values{"status": {"completed"}}); err != nil {
		t.Fatalf("CaptureAutomationRuns() error = %v", err)
	}
	if err := harness.CaptureTasks(testContext(t), url.Values{"limit": {"10"}}); err != nil {
		t.Fatalf("CaptureTasks() error = %v", err)
	}
	if err := harness.CaptureTaskRuns(testContext(t), "task-1", url.Values{"status": {"completed"}}); err != nil {
		t.Fatalf("CaptureTaskRuns() error = %v", err)
	}
	if err := harness.CaptureBridgeHealth(testContext(t)); err != nil {
		t.Fatalf("CaptureBridgeHealth() error = %v", err)
	}
	if err := harness.CaptureBridgeRoutes(testContext(t), "brg-1"); err != nil {
		t.Fatalf("CaptureBridgeRoutes() error = %v", err)
	}
	if err := harness.CaptureBridgeDeliveryState(testContext(t), "brg-1"); err != nil {
		t.Fatalf("CaptureBridgeDeliveryState() error = %v", err)
	}
	if err := harness.CaptureBridgeSecretBindings(testContext(t), "brg-1"); err != nil {
		t.Fatalf("CaptureBridgeSecretBindings() error = %v", err)
	}
	if err := harness.CaptureToolHostDiagnosticsJSON(ToolHostDiagnosticsArtifact{
		SessionID: "sess-1",
		Operations: []ToolHostOperationDiagnostic{{
			Operation:        "write_text_file",
			Path:             "toolhost/output.txt",
			Outcome:          ToolHostOutcomeAllowed,
			SideEffectPath:   "/workspace/toolhost/output.txt",
			SideEffectExists: true,
		}},
	}); err != nil {
		t.Fatalf("CaptureToolHostDiagnosticsJSON() error = %v", err)
	}
	if err := harness.CaptureCombinedFlowJSON(CombinedFlowArtifact{
		Scenario:          "bridge-environment-network",
		SessionID:         "sess-1",
		Channel:           "builders",
		AutomationRunID:   "run-1",
		TaskID:            "task-1",
		TaskRunID:         "task-run-1",
		BridgeID:          "brg-1",
		NetworkMessageIDs: []string{"msg-1"},
		SideEffectPaths:   []string{"/workspace/toolhost/output.txt"},
	}); err != nil {
		t.Fatalf("CaptureCombinedFlowJSON() error = %v", err)
	}

	providerLog := filepath.Join(t.TempDir(), "provider.log")
	if err := os.WriteFile(providerLog, []byte("provider call"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", providerLog, err)
	}
	if err := harness.CaptureProviderCallsFile(providerLog, "text/plain"); err != nil {
		t.Fatalf("CaptureProviderCallsFile() error = %v", err)
	}

	tracePath := filepath.Join(t.TempDir(), "trace.zip")
	if err := os.WriteFile(tracePath, []byte("trace-bytes"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", tracePath, err)
	}
	if err := harness.CaptureBrowserTraceFile(tracePath); err != nil {
		t.Fatalf("CaptureBrowserTraceFile() error = %v", err)
	}

	screenshotOne := filepath.Join(t.TempDir(), "screen-1.png")
	screenshotTwo := filepath.Join(t.TempDir(), "screen-2.png")
	for _, item := range []struct {
		path string
		data string
	}{
		{path: screenshotOne, data: "one"},
		{path: screenshotTwo, data: "two"},
	} {
		if err := os.WriteFile(item.path, []byte(item.data), 0o644); err != nil {
			t.Fatalf("os.WriteFile(%q) error = %v", item.path, err)
		}
	}
	if err := harness.CaptureBrowserScreenshots([]string{screenshotOne, screenshotTwo}); err != nil {
		t.Fatalf("CaptureBrowserScreenshots() error = %v", err)
	}
	if err := harness.CaptureBrowserConsoleJSON([]map[string]string{{"level": "error"}}); err != nil {
		t.Fatalf("CaptureBrowserConsoleJSON() error = %v", err)
	}
	if err := harness.CaptureBrowserNetworkJSON([]map[string]string{{"url": "/api/demo"}}); err != nil {
		t.Fatalf("CaptureBrowserNetworkJSON() error = %v", err)
	}
	if err := harness.StopSession(testContext(t), "sess-1"); err != nil {
		t.Fatalf("StopSession() error = %v", err)
	}

	manifest := harness.Artifacts.Manifest()
	if got, wantMin := len(manifest.Artifacts), 19; got < wantMin {
		t.Fatalf("len(manifest.Artifacts) = %d, want at least %d", got, wantMin)
	}

	manifestPath := harness.Artifacts.ManifestPath()
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", manifestPath, err)
	}
	if !strings.Contains(string(manifestBytes), "bridge_health.json") {
		t.Fatalf("manifest = %s, want bridge_health.json entry", string(manifestBytes))
	}
	if !strings.Contains(string(manifestBytes), "bridge_routes.json") {
		t.Fatalf("manifest = %s, want bridge_routes.json entry", string(manifestBytes))
	}
	if !strings.Contains(string(manifestBytes), "bridge_delivery_state.json") {
		t.Fatalf("manifest = %s, want bridge_delivery_state.json entry", string(manifestBytes))
	}
	if !strings.Contains(string(manifestBytes), "bridge_secret_bindings.json") {
		t.Fatalf("manifest = %s, want bridge_secret_bindings.json entry", string(manifestBytes))
	}

	networkAuditPath, ok := harness.Artifacts.ArtifactPath(ArtifactKindNetworkAudit)
	if !ok {
		t.Fatal("ArtifactPath(network_audit) = missing, want present")
	}
	networkAuditBytes, err := os.ReadFile(networkAuditPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", networkAuditPath, err)
	}
	if strings.Contains(string(networkAuditBytes), "\n{\"id\":") {
		t.Fatalf(
			"network audit artifact = %s, want stable JSON snapshot instead of raw ndjson",
			string(networkAuditBytes),
		)
	}
	if !strings.Contains(string(networkAuditBytes), "\"Direction\": \"sent\"") {
		t.Fatalf("network audit artifact = %s, want decoded audit entry", string(networkAuditBytes))
	}

	automationRunsPath, ok := harness.Artifacts.ArtifactPath(ArtifactKindAutomationRuns)
	if !ok {
		t.Fatal("ArtifactPath(automation_runs) = missing, want present")
	}
	automationRunsBytes, err := os.ReadFile(automationRunsPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", automationRunsPath, err)
	}
	if !strings.Contains(string(automationRunsBytes), `"session_id": "sess-1"`) {
		t.Fatalf("automation runs artifact = %s, want linked session id", string(automationRunsBytes))
	}

	tasksPath, ok := harness.Artifacts.ArtifactPath(ArtifactKindTasks)
	if !ok {
		t.Fatal("ArtifactPath(tasks) = missing, want present")
	}
	tasksBytes, err := os.ReadFile(tasksPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", tasksPath, err)
	}
	if !strings.Contains(string(tasksBytes), `"ref": "run:run-1"`) {
		t.Fatalf("tasks artifact = %s, want automation origin linkage", string(tasksBytes))
	}

	taskRunsPath, ok := harness.Artifacts.ArtifactPath(ArtifactKindTaskRuns)
	if !ok {
		t.Fatal("ArtifactPath(task_runs) = missing, want present")
	}
	taskRunsBytes, err := os.ReadFile(taskRunsPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", taskRunsPath, err)
	}
	if !strings.Contains(string(taskRunsBytes), `"session_id": "sess-1"`) ||
		!strings.Contains(string(taskRunsBytes), `"idempotency_key": "automation-run:run-1"`) {
		t.Fatalf("task runs artifact = %s, want session linkage and idempotency key", string(taskRunsBytes))
	}

	sessionEnvironmentPath, ok := harness.Artifacts.ArtifactPath(ArtifactKindSessionEnvironment)
	if !ok {
		t.Fatal("ArtifactPath(session_environment) = missing, want present")
	}
	sessionEnvironmentBytes, err := os.ReadFile(sessionEnvironmentPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", sessionEnvironmentPath, err)
	}
	if !strings.Contains(string(sessionEnvironmentBytes), `"runtime_root_dir": "/workspace"`) {
		t.Fatalf(
			"session environment artifact = %s, want persisted runtime root metadata",
			string(sessionEnvironmentBytes),
		)
	}
	if !strings.Contains(string(sessionEnvironmentBytes), `"session_state": "stopped"`) {
		t.Fatalf(
			"session environment artifact = %s, want session-level stop visibility",
			string(sessionEnvironmentBytes),
		)
	}

	bridgeRoutesPath, ok := harness.Artifacts.ArtifactPath(ArtifactKindBridgeRoutes)
	if !ok {
		t.Fatal("ArtifactPath(bridge_routes) = missing, want present")
	}
	bridgeRoutesBytes, err := os.ReadFile(bridgeRoutesPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", bridgeRoutesPath, err)
	}
	if !strings.Contains(string(bridgeRoutesBytes), `"session_id": "sess-1"`) {
		t.Fatalf("bridge routes artifact = %s, want route session linkage", string(bridgeRoutesBytes))
	}

	bridgeStatePath, ok := harness.Artifacts.ArtifactPath(ArtifactKindBridgeDeliveryState)
	if !ok {
		t.Fatal("ArtifactPath(bridge_delivery_state) = missing, want present")
	}
	bridgeStateBytes, err := os.ReadFile(bridgeStatePath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", bridgeStatePath, err)
	}
	if !strings.Contains(string(bridgeStateBytes), `"delivery_backlog": 0`) {
		t.Fatalf("bridge delivery state artifact = %s, want delivery health snapshot", string(bridgeStateBytes))
	}

	bridgeBindingsPath, ok := harness.Artifacts.ArtifactPath(ArtifactKindBridgeSecretBindings)
	if !ok {
		t.Fatal("ArtifactPath(bridge_secret_bindings) = missing, want present")
	}
	bridgeBindingsBytes, err := os.ReadFile(bridgeBindingsPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", bridgeBindingsPath, err)
	}
	if !strings.Contains(string(bridgeBindingsBytes), `"vault_ref": "env:AGH_TEST_TELEGRAM_TOKEN"`) {
		t.Fatalf("bridge secret bindings artifact = %s, want stable binding snapshot", string(bridgeBindingsBytes))
	}

	combinedFlowPath, ok := harness.Artifacts.ArtifactPath(ArtifactKindCombinedFlow)
	if !ok {
		t.Fatal("ArtifactPath(combined_flow) = missing, want present")
	}
	combinedFlowBytes, err := os.ReadFile(combinedFlowPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", combinedFlowPath, err)
	}
	if !strings.Contains(string(combinedFlowBytes), `"scenario": "bridge-environment-network"`) ||
		!strings.Contains(string(combinedFlowBytes), `"bridge_id": "brg-1"`) {
		t.Fatalf("combined flow artifact = %s, want cross-domain scenario summary", string(combinedFlowBytes))
	}

	toolHostDiagnosticsPath, ok := harness.Artifacts.ArtifactPath(ArtifactKindToolHostDiagnostics)
	if !ok {
		t.Fatal("ArtifactPath(tool_host_diagnostics) = missing, want present")
	}
	toolHostDiagnosticsBytes, err := os.ReadFile(toolHostDiagnosticsPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", toolHostDiagnosticsPath, err)
	}
	if !strings.Contains(string(toolHostDiagnosticsBytes), `"operation": "write_text_file"`) {
		t.Fatalf(
			"tool host diagnostics artifact = %s, want tool-host operation diagnostics",
			string(toolHostDiagnosticsBytes),
		)
	}

	providerCallsPath, ok := harness.Artifacts.ArtifactPath(ArtifactKindProviderCalls)
	if !ok {
		t.Fatal("ArtifactPath(provider_calls) = missing, want present")
	}
	providerCallsBytes, err := os.ReadFile(providerCallsPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", providerCallsPath, err)
	}
	if !strings.Contains(string(providerCallsBytes), "provider call") {
		t.Fatalf("provider calls artifact = %s, want provider log capture", string(providerCallsBytes))
	}
}

func TestRuntimeHarnessBridgeAndExtensionHelpersSurfaceTransportErrors(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/bridges/health/stream":
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = fmt.Fprint(w, "event: bridge_health\ndata: not-json\n\n")
		default:
			http.Error(w, "boom", http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	harness := &RuntimeHarness{
		Artifacts:   NewArtifactCollector(t),
		HTTPBaseURL: server.URL,
		HTTPClient:  server.Client(),
		UDSBaseURL:  server.URL,
		UDSClient:   server.Client(),
	}

	ctx := testContext(t)

	_, err := harness.GetExtension(ctx, "telegram-reference")
	assertErrorContains(t, err, "/api/extensions/telegram-reference status 500: boom")
	if _, err := harness.CreateBridge(ctx, aghcontract.CreateBridgeRequest{
		Scope:         bridgepkg.ScopeWorkspace,
		WorkspaceID:   "ws-1",
		Platform:      "telegram",
		ExtensionName: "telegram-reference",
		DisplayName:   "Telegram Runtime",
		Enabled:       false,
		Status:        bridgepkg.BridgeStatusDisabled,
	}); err != nil {
		assertErrorContains(t, err, "/api/bridges status 500: boom")
	} else {
		t.Fatal("CreateBridge() error = nil, want transport error")
	}
	if _, err := harness.PutBridgeSecretBinding(
		ctx,
		"brg-1",
		"bot_token",
		aghcontract.PutBridgeSecretBindingRequest{
			VaultRef: "env:AGH_TEST_TELEGRAM_TOKEN",
			Kind:     "token",
		},
	); err != nil {
		assertErrorContains(t, err, "/api/bridges/brg-1/secret-bindings/bot_token status 500: boom")
	} else {
		t.Fatal("PutBridgeSecretBinding() error = nil, want transport error")
	}
	assertErrorContains(t, harness.CaptureBridgeRoutes(ctx, "brg-1"), "/api/bridges/brg-1/routes status 500: boom")
	assertErrorContains(
		t,
		harness.CaptureBridgeSecretBindings(ctx, "brg-1"),
		"/api/bridges/brg-1/secret-bindings status 500: boom",
	)
	assertErrorContains(t, harness.CaptureBridgeHealth(ctx), "decode bridge health snapshot")
}

func TestRuntimeHarnessBridgeHelpersRejectBlankBridgeID(t *testing.T) {
	t.Parallel()

	harness := &RuntimeHarness{}
	ctx := testContext(t)

	tests := []struct {
		name string
		call func() error
	}{
		{
			name: "get bridge",
			call: func() error {
				_, err := harness.GetBridge(ctx, "   ")
				return err
			},
		},
		{
			name: "enable bridge",
			call: func() error {
				_, err := harness.EnableBridge(ctx, "")
				return err
			},
		},
		{
			name: "restart bridge",
			call: func() error {
				_, err := harness.RestartBridge(ctx, "")
				return err
			},
		},
		{
			name: "list routes",
			call: func() error {
				_, err := harness.ListBridgeRoutes(ctx, "")
				return err
			},
		},
		{
			name: "put secret binding",
			call: func() error {
				_, err := harness.PutBridgeSecretBinding(
					ctx,
					"",
					"bot_token",
					aghcontract.PutBridgeSecretBindingRequest{},
				)
				return err
			},
		},
		{
			name: "list secret bindings",
			call: func() error {
				_, err := harness.ListBridgeSecretBindings(ctx, "")
				return err
			},
		},
		{
			name: "capture routes",
			call: func() error {
				return harness.CaptureBridgeRoutes(ctx, "")
			},
		},
		{
			name: "capture delivery state",
			call: func() error {
				return harness.CaptureBridgeDeliveryState(ctx, "")
			},
		},
		{
			name: "capture secret bindings",
			call: func() error {
				return harness.CaptureBridgeSecretBindings(ctx, "")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assertErrorContains(t, tt.call(), "bridge id is required")
		})
	}
}

type harnessTestServer struct {
	*httptest.Server
	handlerErrs chan error
}

func newHarnessTestServer(t testing.TB) *harnessTestServer {
	t.Helper()

	now := time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)
	routeTime := now.Add(5 * time.Minute)
	handlerErrs := make(chan error, 32)
	mux := http.NewServeMux()

	mux.HandleFunc("/api/workspaces/resolve", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, aghcontract.WorkspaceResponse{
			Workspace: aghcontract.WorkspacePayload{
				ID:        "ws-1",
				RootDir:   "/workspace",
				Name:      "workspace",
				CreatedAt: now,
				UpdatedAt: now,
			},
		})
	})
	mux.HandleFunc("/api/workspaces/ws-1", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, aghcontract.WorkspaceResponse{
			Workspace: aghcontract.WorkspacePayload{
				ID:        "ws-1",
				RootDir:   "/workspace",
				Name:      "workspace",
				CreatedAt: now,
				UpdatedAt: now,
			},
		})
	})
	mux.HandleFunc("/api/sessions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusCreated)
		writeJSON(w, aghcontract.SessionResponse{
			Session: aghcontract.SessionPayload{
				ID:            "sess-1",
				AgentName:     "coder",
				WorkspaceID:   "ws-1",
				WorkspacePath: "/workspace",
				CreatedAt:     now,
				UpdatedAt:     now,
			},
		})
	})
	mux.HandleFunc("/api/sessions/sess-1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeJSON(w, aghcontract.SessionResponse{
			Session: aghcontract.SessionPayload{
				ID:            "sess-1",
				AgentName:     "coder",
				WorkspaceID:   "ws-1",
				WorkspacePath: "/workspace",
				State:         "stopped",
				StopReason:    store.StopCompleted,
				Environment: &aghcontract.SessionEnvironmentPayload{
					EnvironmentID: "env-1",
					Backend:       "local",
					Profile:       "local",
					State:         "ready",
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
		})
	})
	mux.HandleFunc("/api/sessions/sess-1/resume", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, aghcontract.SessionResponse{
			Session: aghcontract.SessionPayload{
				ID:            "sess-1",
				AgentName:     "coder",
				WorkspaceID:   "ws-1",
				WorkspacePath: "/workspace",
				Channel:       "builders",
				State:         "active",
				Environment: &aghcontract.SessionEnvironmentPayload{
					EnvironmentID: "env-1",
					Backend:       "local",
					Profile:       "local",
					State:         "ready",
				},
				CreatedAt: now,
				UpdatedAt: routeTime,
			},
		})
	})
	mux.HandleFunc("/api/sessions/sess-1/transcript", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{
			"messages": []map[string]any{
				{"role": "user", "content": "hello world"},
				{"role": "assistant", "content": "echo: hello world"},
			},
		})
	})
	mux.HandleFunc("/api/sessions/sess-1/events", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{
			"events": []map[string]any{
				{"id": "evt-1", "type": "agent_message"},
				{"id": "evt-2", "type": "session_stopped"},
			},
		})
	})
	mux.HandleFunc("/api/sessions/sess-1/prompt", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(
			w,
			"event: agent_message\n"+
				"data: {\"type\":\"text-delta\"}\n\n"+
				"event: done\n"+
				"data: [DONE]\n\n",
		)
	})
	mux.HandleFunc("/api/network/channels/builders/messages", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, aghcontract.NetworkChannelMessagesResponse{
			Messages: []aghcontract.NetworkChannelMessagePayload{{
				MessageID: "msg-1",
				Channel:   "builders",
				PeerID:    "peer-1",
				Text:      "hello",
				Timestamp: now,
			}},
		})
	})
	mux.HandleFunc("/api/network/status", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, aghcontract.NetworkStatusResponse{
			Network: aghcontract.NetworkStatusPayload{
				Enabled:           true,
				Status:            "running",
				LocalPeers:        1,
				RemotePeers:       1,
				Channels:          1,
				MessagesSent:      3,
				MessagesDelivered: 2,
			},
		})
	})
	mux.HandleFunc("/api/network/peers", func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Query().Get("channel"), "builders"; got != want {
			reportHarnessHandlerError(
				w,
				handlerErrs,
				http.StatusBadRequest,
				"network peers query channel = %q, want %q",
				got,
				want,
			)
			return
		}
		writeJSON(w, aghcontract.NetworkPeersResponse{
			Peers: []aghcontract.NetworkPeerPayload{{
				SessionID:   stringPtr("sess-1"),
				PeerID:      "coder.sess-1",
				DisplayName: "coder",
				Channel:     "builders",
				Local:       true,
				PeerCard: aghcontract.NetworkPeerCardPayload{
					PeerID:            "coder.sess-1",
					ProfilesSupported: []string{"agh-network/v0"},
					Capabilities: []aghcontract.NetworkCapabilityBriefPayload{{
						ID:      "chat.review",
						Summary: "Reviews chat requests.",
					}},
					ArtifactsSupported: []string{"capability"},
				},
			}},
		})
	})
	mux.HandleFunc("/api/network/channels", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
			writeJSON(w, aghcontract.CreateNetworkChannelResponse{
				Channel: aghcontract.NetworkChannelDetailPayload{
					Channel:   "builders",
					PeerCount: 1,
					Sessions: []aghcontract.SessionPayload{{
						ID:        "sess-1",
						AgentName: "coder",
						Channel:   "builders",
						State:     "active",
					}},
				},
			})
			return
		}
		writeJSON(w, aghcontract.NetworkChannelsResponse{
			Channels: []aghcontract.NetworkChannelPayload{{
				Channel:         "builders",
				PeerCount:       1,
				LocalPeerCount:  1,
				RemotePeerCount: 0,
				SessionCount:    1,
				MessageCount:    1,
				LastMessageAt:   &now,
			}},
		})
	})
	mux.HandleFunc("/api/network/channels/builders", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, aghcontract.NetworkChannelResponse{
			Channel: aghcontract.NetworkChannelDetailPayload{
				Channel:   "builders",
				PeerCount: 1,
				Sessions: []aghcontract.SessionPayload{{
					ID:        "sess-1",
					AgentName: "coder",
					Channel:   "builders",
					State:     "active",
				}},
				Peers: []aghcontract.NetworkPeerPayload{{
					SessionID:   stringPtr("sess-1"),
					PeerID:      "coder.sess-1",
					DisplayName: "coder",
					Channel:     "builders",
					Local:       true,
					PeerCard: aghcontract.NetworkPeerCardPayload{
						PeerID:            "coder.sess-1",
						ProfilesSupported: []string{"agh-network/v0"},
						Capabilities: []aghcontract.NetworkCapabilityBriefPayload{{
							ID:      "chat.review",
							Summary: "Reviews chat requests.",
						}},
						ArtifactsSupported: []string{"capability"},
					},
				}},
			},
		})
	})
	mux.HandleFunc("/api/network/inbox", func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Query().Get("session_id"), "sess-1"; got != want {
			reportHarnessHandlerError(
				w,
				handlerErrs,
				http.StatusBadRequest,
				"network inbox query session_id = %q, want %q",
				got,
				want,
			)
			return
		}
		writeJSON(w, aghcontract.NetworkInboxResponse{
			Messages: []aghcontract.NetworkEnvelopePayload{{
				Protocol: "agh-network/v0",
				ID:       "msg-inbox-1",
				Kind:     "direct",
				Channel:  "builders",
				From:     "peer-1",
				TS:       now.Unix(),
				Body:     json.RawMessage(`{"text":"review this"}`),
			}},
		})
	})
	mux.HandleFunc("/api/network/send", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, aghcontract.NetworkSendResponse{
			Message: aghcontract.NetworkSendPayload{
				ID:        "msg-send-1",
				SessionID: "sess-1",
				Channel:   "builders",
				Kind:      "say",
			},
		})
	})
	mux.HandleFunc("/api/automation/runs", func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Query().Get("status"), "completed"; got != want {
			reportHarnessHandlerError(
				w,
				handlerErrs,
				http.StatusBadRequest,
				"automation runs query status = %q, want %q",
				got,
				want,
			)
			return
		}
		writeJSON(w, aghcontract.RunsResponse{
			Runs: []aghcontract.RunPayload{{
				ID:        "run-1",
				SessionID: "sess-1",
				Status:    "completed",
				Attempt:   1,
			}},
		})
	})
	mux.HandleFunc("/api/tasks", func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Query().Get("limit"), "10"; got != want {
			reportHarnessHandlerError(
				w,
				handlerErrs,
				http.StatusBadRequest,
				"tasks query limit = %q, want %q",
				got,
				want,
			)
			return
		}
		writeJSON(w, aghcontract.TasksResponse{
			Tasks: []aghcontract.TaskSummaryPayload{{
				ID:        "task-1",
				Scope:     "workspace",
				Title:     "demo",
				Status:    "in_progress",
				CreatedBy: taskpkg.ActorIdentity{Kind: taskpkg.ActorKindAutomation, Ref: "job-1"},
				Origin:    taskpkg.Origin{Kind: taskpkg.OriginKindAutomation, Ref: "run:run-1"},
				CreatedAt: now,
				UpdatedAt: now,
			}},
		})
	})
	mux.HandleFunc("/api/tasks/task-1/runs", func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Query().Get("status"), "completed"; got != want {
			reportHarnessHandlerError(
				w,
				handlerErrs,
				http.StatusBadRequest,
				"task runs query status = %q, want %q",
				got,
				want,
			)
			return
		}
		writeJSON(w, aghcontract.TaskRunsResponse{
			Runs: []aghcontract.TaskRunPayload{{
				ID:             "task-run-1",
				TaskID:         "task-1",
				Status:         "completed",
				Attempt:        1,
				SessionID:      "sess-1",
				Origin:         taskpkg.Origin{Kind: taskpkg.OriginKindAutomation, Ref: "run:run-1"},
				IdempotencyKey: "automation-run:run-1",
				QueuedAt:       now,
			}},
		})
	})
	mux.HandleFunc("/api/extensions", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, aghcontract.ExtensionsResponse{
				Extensions: []aghcontract.ExtensionPayload{{
					Name:          "telegram-reference",
					Version:       "0.1.0",
					Type:          "local",
					Source:        "user",
					Enabled:       true,
					State:         "registered",
					Capabilities:  []string{"bridge.adapter"},
					Actions:       []string{"bridges/messages/ingest", "bridges/instances/report_state"},
					Health:        "healthy",
					DaemonRunning: true,
				}},
			})
		case http.MethodPost:
			var request aghcontract.InstallExtensionRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				reportHarnessHandlerError(
					w,
					handlerErrs,
					http.StatusBadRequest,
					"json.Decode(install extension) error = %v",
					err,
				)
				return
			}
			if got, want := request.Path, "/extensions/telegram-reference"; got != want {
				reportHarnessHandlerError(
					w,
					handlerErrs,
					http.StatusBadRequest,
					"install extension path = %q, want %q",
					got,
					want,
				)
				return
			}
			if got, want := request.Checksum, "sha256-demo"; got != want {
				reportHarnessHandlerError(
					w,
					handlerErrs,
					http.StatusBadRequest,
					"install extension checksum = %q, want %q",
					got,
					want,
				)
				return
			}
			w.WriteHeader(http.StatusCreated)
			writeJSON(w, aghcontract.ExtensionResponse{
				Extension: aghcontract.ExtensionPayload{
					Name:          "telegram-reference",
					Version:       "0.1.0",
					Type:          "local",
					Source:        "user",
					Enabled:       true,
					State:         "registered",
					Capabilities:  []string{"bridge.adapter"},
					Actions:       []string{"bridges/messages/ingest", "bridges/instances/report_state"},
					Health:        "healthy",
					DaemonRunning: true,
				},
			})
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/extensions/telegram-reference", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, aghcontract.ExtensionResponse{
			Extension: aghcontract.ExtensionPayload{
				Name:          "telegram-reference",
				Version:       "0.1.0",
				Type:          "local",
				Source:        "user",
				Enabled:       true,
				State:         "registered",
				Capabilities:  []string{"bridge.adapter"},
				Actions:       []string{"bridges/messages/ingest", "bridges/instances/report_state"},
				Health:        "healthy",
				DaemonRunning: true,
			},
		})
	})
	mux.HandleFunc("/api/extensions/telegram-reference/enable", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, aghcontract.ExtensionResponse{
			Extension: aghcontract.ExtensionPayload{
				Name:          "telegram-reference",
				Version:       "0.1.0",
				Type:          "local",
				Source:        "user",
				Enabled:       true,
				State:         "active",
				Capabilities:  []string{"bridge.adapter"},
				Actions:       []string{"bridges/messages/ingest", "bridges/instances/report_state"},
				Health:        "healthy",
				DaemonRunning: true,
			},
		})
	})
	mux.HandleFunc("/api/extensions/telegram-reference/disable", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, aghcontract.ExtensionResponse{
			Extension: aghcontract.ExtensionPayload{
				Name:          "telegram-reference",
				Version:       "0.1.0",
				Type:          "local",
				Source:        "user",
				Enabled:       false,
				State:         "registered",
				Capabilities:  []string{"bridge.adapter"},
				Actions:       []string{"bridges/messages/ingest", "bridges/instances/report_state"},
				Health:        "idle",
				DaemonRunning: true,
			},
		})
	})
	mux.HandleFunc("/api/bridges", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var request aghcontract.CreateBridgeRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			reportHarnessHandlerError(
				w,
				handlerErrs,
				http.StatusBadRequest,
				"json.Decode(create bridge) error = %v",
				err,
			)
			return
		}
		if got, want := request.ExtensionName, "telegram-reference"; got != want {
			reportHarnessHandlerError(
				w,
				handlerErrs,
				http.StatusBadRequest,
				"create bridge extension = %q, want %q",
				got,
				want,
			)
			return
		}
		w.WriteHeader(http.StatusCreated)
		writeJSON(w, aghcontract.BridgeResponse{
			Bridge: aghcontract.BridgePayload{
				ID:            "brg-1",
				Scope:         bridgepkg.ScopeWorkspace,
				WorkspaceID:   "ws-1",
				Platform:      "telegram",
				ExtensionName: "telegram-reference",
				DisplayName:   "Telegram Runtime",
				Enabled:       false,
				Status:        bridgepkg.BridgeStatusDisabled,
				CreatedAt:     now,
				UpdatedAt:     now,
			},
			Health: aghcontract.BridgeHealthPayload{
				BridgeInstanceID: "brg-1",
				Status:           bridgepkg.BridgeStatusDisabled,
			},
		})
	})
	mux.HandleFunc("/api/bridges/brg-1", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, aghcontract.BridgeResponse{
			Bridge: aghcontract.BridgePayload{
				ID:            "brg-1",
				Scope:         bridgepkg.ScopeWorkspace,
				WorkspaceID:   "ws-1",
				Platform:      "telegram",
				ExtensionName: "telegram-reference",
				DisplayName:   "Telegram Runtime",
				Enabled:       true,
				Status:        bridgepkg.BridgeStatusReady,
				CreatedAt:     now,
				UpdatedAt:     routeTime,
			},
			Health: aghcontract.BridgeHealthPayload{
				BridgeInstanceID: "brg-1",
				Status:           bridgepkg.BridgeStatusReady,
				RouteCount:       1,
				DeliveryBacklog:  0,
			},
		})
	})
	mux.HandleFunc("/api/bridges/brg-1/enable", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, aghcontract.BridgeResponse{
			Bridge: aghcontract.BridgePayload{
				ID:            "brg-1",
				Scope:         bridgepkg.ScopeWorkspace,
				WorkspaceID:   "ws-1",
				Platform:      "telegram",
				ExtensionName: "telegram-reference",
				DisplayName:   "Telegram Runtime",
				Enabled:       true,
				Status:        bridgepkg.BridgeStatusStarting,
				CreatedAt:     now,
				UpdatedAt:     routeTime,
			},
			Health: aghcontract.BridgeHealthPayload{
				BridgeInstanceID: "brg-1",
				Status:           bridgepkg.BridgeStatusStarting,
			},
		})
	})
	mux.HandleFunc("/api/bridges/brg-1/restart", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, aghcontract.BridgeResponse{
			Bridge: aghcontract.BridgePayload{
				ID:            "brg-1",
				Scope:         bridgepkg.ScopeWorkspace,
				WorkspaceID:   "ws-1",
				Platform:      "telegram",
				ExtensionName: "telegram-reference",
				DisplayName:   "Telegram Runtime",
				Enabled:       true,
				Status:        bridgepkg.BridgeStatusReady,
				CreatedAt:     now,
				UpdatedAt:     routeTime,
			},
			Health: aghcontract.BridgeHealthPayload{
				BridgeInstanceID: "brg-1",
				Status:           bridgepkg.BridgeStatusReady,
				RouteCount:       1,
				DeliveryBacklog:  0,
			},
		})
	})
	mux.HandleFunc("/api/bridges/brg-1/routes", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, aghcontract.BridgeRoutesResponse{
			Routes: []bridgepkg.BridgeRoute{{
				Scope:            bridgepkg.ScopeWorkspace,
				WorkspaceID:      "ws-1",
				BridgeInstanceID: "brg-1",
				PeerID:           "telegram:chat:777:user:888",
				ThreadID:         "654",
				SessionID:        "sess-1",
				AgentName:        "coder",
				LastActivityAt:   routeTime,
				CreatedAt:        routeTime,
				UpdatedAt:        routeTime,
			}},
		})
	})
	mux.HandleFunc("/api/bridges/brg-1/secret-bindings", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, aghcontract.BridgeSecretBindingsResponse{
			Bindings: []bridgepkg.BridgeSecretBinding{{
				BridgeInstanceID: "brg-1",
				BindingName:      "bot_token",
				VaultRef:         "env:AGH_TEST_TELEGRAM_TOKEN",
				Kind:             "token",
				CreatedAt:        now,
				UpdatedAt:        routeTime,
			}},
		})
	})
	mux.HandleFunc("/api/bridges/brg-1/secret-bindings/bot_token", func(w http.ResponseWriter, r *http.Request) {
		var request aghcontract.PutBridgeSecretBindingRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			reportHarnessHandlerError(
				w,
				handlerErrs,
				http.StatusBadRequest,
				"json.Decode(put bridge secret binding) error = %v",
				err,
			)
			return
		}
		if got, want := request.VaultRef, "env:AGH_TEST_TELEGRAM_TOKEN"; got != want {
			reportHarnessHandlerError(
				w,
				handlerErrs,
				http.StatusBadRequest,
				"secret binding vault_ref = %q, want %q",
				got,
				want,
			)
			return
		}
		if got, want := request.Kind, "token"; got != want {
			reportHarnessHandlerError(
				w,
				handlerErrs,
				http.StatusBadRequest,
				"secret binding kind = %q, want %q",
				got,
				want,
			)
			return
		}
		writeJSON(w, aghcontract.BridgeSecretBindingResponse{
			Binding: bridgepkg.BridgeSecretBinding{
				BridgeInstanceID: "brg-1",
				BindingName:      "bot_token",
				VaultRef:         request.VaultRef,
				Kind:             request.Kind,
				CreatedAt:        now,
				UpdatedAt:        routeTime,
			},
		})
	})
	mux.HandleFunc("/api/bridges/health/stream", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		payload, err := json.Marshal(aghcontract.BridgeHealthStreamPayload{
			GeneratedAt: now,
			BridgeHealth: map[string]aghcontract.BridgeHealthPayload{
				"brg-1": {
					BridgeInstanceID: "brg-1",
					Status:           "ready",
					RouteCount:       1,
					DeliveryBacklog:  0,
				},
			},
		})
		if err != nil {
			reportHarnessHandlerError(
				w,
				handlerErrs,
				http.StatusInternalServerError,
				"json.Marshal(bridge health) error = %v",
				err,
			)
			return
		}
		_, _ = fmt.Fprintf(w, "event: bridge_health\ndata: %s\n\n", payload)
	})

	server := &harnessTestServer{
		Server:      httptest.NewServer(mux),
		handlerErrs: handlerErrs,
	}
	t.Cleanup(func() {
		server.assertNoHandlerErrors(t)
	})
	t.Cleanup(server.Close)
	return server
}

func writeJSON(w http.ResponseWriter, value any) {
	if err := writeJSONResponse(w, value); err != nil {
		panic(err)
	}
}

func writeJSONResponse(w http.ResponseWriter, value any) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(value)
}

func testContext(t testing.TB) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func stringPtr(value string) *string {
	return &value
}

func (s *harnessTestServer) assertNoHandlerErrors(t testing.TB) {
	t.Helper()

	close(s.handlerErrs)

	messages := make([]string, 0)
	for err := range s.handlerErrs {
		messages = append(messages, err.Error())
	}
	if len(messages) > 0 {
		t.Fatalf("handler validation errors:\n%s", strings.Join(messages, "\n"))
	}
}

func reportHarnessHandlerError(
	w http.ResponseWriter,
	errCh chan<- error,
	status int,
	format string,
	args ...any,
) {
	err := fmt.Errorf(format, args...)
	errCh <- err
	http.Error(w, err.Error(), status)
}

func assertErrorContains(t testing.TB, err error, want string) {
	t.Helper()

	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}
