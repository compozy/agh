//go:build integration && !windows

package daemon

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	aghcontract "github.com/compozy/agh/internal/api/contract"
	"github.com/compozy/agh/internal/testutil/acpmock"
	e2etest "github.com/compozy/agh/internal/testutil/e2e"
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
