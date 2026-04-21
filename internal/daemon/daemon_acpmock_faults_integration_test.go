//go:build integration && !windows

package daemon

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	aghcontract "github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/testutil/acpmock"
	e2etest "github.com/pedronauck/agh/internal/testutil/e2e"
)

const faultyMockAgentName = "mock-faulty"

func TestDaemonE2EACPmockCrashMidStreamProjectsRuntimeFailure(t *testing.T) {
	acpmock.RequireDriver(t)
	t.Parallel()

	harness, session := startFaultyMockSession(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	stream, err := harness.PromptSessionHTTP(ctx, session.ID, "trigger crash mid-stream")
	if err != nil {
		t.Fatalf("PromptSessionHTTP() error = %v", err)
	}
	assertFaultPromptProjection(
		t,
		ctx,
		harness,
		session.ID,
		stream,
		"partial before crash",
		false,
	)
}

func TestDaemonE2EACPmockInvalidFrameProjectsRuntimeFailure(t *testing.T) {
	acpmock.RequireDriver(t)
	t.Parallel()

	harness, session := startFaultyMockSession(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	stream, err := harness.PromptSessionHTTP(ctx, session.ID, "trigger invalid frame")
	if err != nil {
		t.Fatalf("PromptSessionHTTP() error = %v", err)
	}
	assertFaultPromptProjection(
		t,
		ctx,
		harness,
		session.ID,
		stream,
		"partial before invalid frame",
		false,
	)
}

func TestDaemonE2EACPmockPermissionDisconnectProjectsRuntimeFailure(t *testing.T) {
	acpmock.RequireDriver(t)
	t.Parallel()

	harness, session := startFaultyMockSession(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	stream, err := harness.PromptSessionHTTP(ctx, session.ID, "trigger permission disconnect")
	if err != nil {
		t.Fatalf("PromptSessionHTTP() error = %v", err)
	}
	assertFaultPromptProjection(
		t,
		ctx,
		harness,
		session.ID,
		stream,
		"",
		true,
	)
}

func TestDaemonE2EACPmockBlockedCancelStopsPromptWithoutOrphaning(t *testing.T) {
	acpmock.RequireDriver(t)
	t.Parallel()

	harness, session := startFaultyMockSession(t)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	type promptResult struct {
		stream []e2etest.SSEEvent
		err    error
	}

	resultCh := make(chan promptResult, 1)
	go func() {
		stream, err := harness.PromptSessionHTTP(ctx, session.ID, "block until canceled")
		resultCh <- promptResult{stream: stream, err: err}
	}()

	waitForRuntimeCondition(t, "blocked ACP prompt to be recorded", 10*time.Second, func() bool {
		eventsResp, err := harness.SessionEvents(ctx, session.ID)
		if err != nil {
			return false
		}
		for _, event := range eventsResp.Events {
			if event.Type == "user_message" {
				return true
			}
		}
		return false
	})

	if err := harness.StopSession(ctx, session.ID); err != nil {
		t.Fatalf("StopSession(%q) error = %v", session.ID, err)
	}

	var result promptResult
	select {
	case result = <-resultCh:
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for blocked ACP prompt to finish after stop")
	}
	if result.err != nil {
		t.Fatalf("PromptSessionHTTP(blocked cancel) error = %v", result.err)
	}
	if !sseStreamContainsEvent(result.stream, "error") {
		t.Fatalf("prompt stream = %#v, want transport error when the blocked peer is stopped", result.stream)
	}
	if !sseStreamContainsEvent(result.stream, "done") {
		t.Fatalf("prompt stream = %#v, want terminal done marker after cancellation", result.stream)
	}

	waitForRuntimeCondition(t, "blocked ACP session stopped", 10*time.Second, func() bool {
		current, err := harness.GetSession(ctx, session.ID)
		return err == nil && string(current.State) == "stopped"
	})

	sessionInfo, err := harness.GetSession(ctx, session.ID)
	if err != nil {
		t.Fatalf("GetSession(%q) error = %v", session.ID, err)
	}
	if got, want := string(sessionInfo.State), "stopped"; got != want {
		t.Fatalf("sessionInfo.State = %q, want %q", got, want)
	}
	if got, want := sessionInfo.StopReason, store.StopUserCanceled; got != want {
		t.Fatalf("sessionInfo.StopReason = %q, want %q", got, want)
	}

	meta := mustReadSessionMeta(t, harness, session.ID)
	if meta.Liveness == nil {
		t.Fatal("meta.Liveness = nil, want persisted liveness metadata")
	}
	if got := meta.Liveness.SubprocessPID; got != 0 {
		t.Fatalf("meta.Liveness.SubprocessPID = %d, want 0 after clean stop", got)
	}
	if got := strings.TrimSpace(meta.Liveness.StallState); got != "" {
		t.Fatalf("meta.Liveness.StallState = %q, want empty after clean stop", got)
	}
	if got := strings.TrimSpace(meta.Liveness.StallReason); got != "" {
		t.Fatalf("meta.Liveness.StallReason = %q, want empty after clean stop", got)
	}

	eventsResp := mustSessionEvents(t, ctx, harness, session.ID)
	events := decodeAgentEvents(t, eventsResp.Events)
	if !containsAgentEvent(events, aghcontract.AgentEventPayload{Type: "error"}) {
		t.Fatalf("events = %#v, want error event after blocked peer disconnect", events)
	}

	if err := harness.CaptureSessionTranscript(ctx, session.ID); err != nil {
		t.Fatalf("CaptureSessionTranscript() error = %v", err)
	}
	if err := harness.CaptureSessionEvents(ctx, session.ID); err != nil {
		t.Fatalf("CaptureSessionEvents() error = %v", err)
	}
	if err := harness.CaptureSessionEnvironment(ctx, session.ID); err != nil {
		t.Fatalf("CaptureSessionEnvironment() error = %v", err)
	}
}

func startFaultyMockSession(t testing.TB) (*e2etest.RuntimeHarness, aghcontract.SessionPayload) {
	t.Helper()

	harness := e2etest.StartRuntimeHarness(t, e2etest.RuntimeHarnessOptions{
		MockAgents: []e2etest.MockAgentSpec{{
			FixturePath:  mockFixturePath(t, "driver_fault_fixture.json"),
			FixtureAgent: "faulty",
			AgentName:    faultyMockAgentName,
		}},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	session := createFixtureBackedSession(t, ctx, harness, faultyMockAgentName, "faulty-session")
	return harness, session
}

func assertFaultPromptProjection(
	t testing.TB,
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	sessionID string,
	stream []e2etest.SSEEvent,
	wantTranscriptFragment string,
	wantPermission bool,
) {
	t.Helper()

	if !sseStreamContainsEvent(stream, "error") {
		t.Fatalf("prompt stream = %#v, want error event", stream)
	}
	if wantTranscriptFragment != "" && !e2etest.RecordsContainTextDelta(stream, wantTranscriptFragment) {
		t.Fatalf("prompt stream = %#v, want assistant text delta %q", stream, wantTranscriptFragment)
	}
	if wantPermission && !sseStreamContainsEvent(stream, "permission") {
		t.Fatalf("prompt stream = %#v, want permission event before failure", stream)
	}

	httpSession := mustHTTPSession(t, ctx, harness, sessionID)
	udsSession, err := harness.GetSession(ctx, sessionID)
	if err != nil {
		t.Fatalf("GetSession(%q) error = %v", sessionID, err)
	}
	if got, want := httpSession.ID, sessionID; got != want {
		t.Fatalf("HTTP session ID = %q, want %q", got, want)
	}
	if got, want := udsSession.ID, sessionID; got != want {
		t.Fatalf("UDS session ID = %q, want %q", got, want)
	}

	transcript := mustSessionTranscript(t, ctx, harness, sessionID)
	content := joinTranscriptContent(transcript.Messages)
	if wantTranscriptFragment != "" && !strings.Contains(content, wantTranscriptFragment) {
		t.Fatalf("transcript = %q, want fragment %q", content, wantTranscriptFragment)
	}
	if wantPermission && strings.Contains(content, "allow-always") {
		t.Fatalf("transcript = %q, want no approval decision after disconnect", content)
	}

	eventsResp := mustSessionEvents(t, ctx, harness, sessionID)
	events := decodeAgentEvents(t, eventsResp.Events)
	if wantTranscriptFragment != "" && !containsAgentEvent(events, aghcontract.AgentEventPayload{
		Type: "agent_message",
		Text: wantTranscriptFragment,
	}) {
		t.Fatalf("events = %#v, want agent_message %q", events, wantTranscriptFragment)
	}
	if !containsAgentEvent(events, aghcontract.AgentEventPayload{Type: "error"}) {
		t.Fatalf("events = %#v, want error event", events)
	}
	if containsAgentEvent(events, aghcontract.AgentEventPayload{Type: "done"}) {
		t.Fatalf("events = %#v, want no done event after runtime failure", events)
	}
	if wantPermission && !containsAgentEvent(events, aghcontract.AgentEventPayload{Type: "permission"}) {
		t.Fatalf("events = %#v, want permission event", events)
	}

	if err := harness.CaptureSessionTranscript(ctx, sessionID); err != nil {
		t.Fatalf("CaptureSessionTranscript() error = %v", err)
	}
	if err := harness.CaptureSessionEvents(ctx, sessionID); err != nil {
		t.Fatalf("CaptureSessionEvents() error = %v", err)
	}
	if err := harness.CaptureSessionEnvironment(ctx, sessionID); err != nil {
		t.Fatalf("CaptureSessionEnvironment() error = %v", err)
	}

	assertArtifactExists(t, harness, e2etest.ArtifactKindTranscript)
	assertArtifactExists(t, harness, e2etest.ArtifactKindEvents)
	assertArtifactExists(t, harness, e2etest.ArtifactKindSessionEnvironment)
}

func mustHTTPSession(
	t testing.TB,
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	sessionID string,
) aghcontract.SessionPayload {
	t.Helper()

	var response aghcontract.SessionResponse
	if err := harness.HTTPJSON(
		ctx,
		http.MethodGet,
		"/api/sessions/"+url.PathEscape(sessionID),
		nil,
		&response,
	); err != nil {
		t.Fatalf("HTTP session %q error = %v", sessionID, err)
	}
	return response.Session
}

func sseStreamContainsEvent(records []e2etest.SSEEvent, want string) bool {
	for _, record := range records {
		if record.Event == want {
			return true
		}
	}
	return false
}

func assertArtifactExists(
	t testing.TB,
	harness *e2etest.RuntimeHarness,
	kind e2etest.ArtifactKind,
) {
	t.Helper()

	path, ok := harness.Artifacts.ArtifactPath(kind)
	if !ok {
		t.Fatalf("ArtifactPath(%s) = missing, want present", kind)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("os.Stat(%q) error = %v", path, err)
	}
}
