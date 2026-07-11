//go:build integration && !windows

package daemon

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	aghcontract "github.com/compozy/agh/internal/api/contract"
	e2etest "github.com/compozy/agh/internal/testutil/e2e"
	"github.com/compozy/agh/internal/transcript"
)

func mockFixturePath(t testing.TB, name string) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed")
	}
	return filepath.Join(filepath.Dir(file), "..", "testutil", "acpmock", "testdata", name)
}

func createFixtureBackedSession(
	t testing.TB,
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	agentName string,
	name string,
) aghcontract.SessionPayload {
	t.Helper()

	session, err := harness.CreateSession(ctx, aghcontract.CreateSessionRequest{
		AgentName:     agentName,
		Name:          name,
		WorkspacePath: harness.WorkspaceRoot,
	})
	if err != nil {
		t.Fatalf("CreateSession(%q) error = %v", agentName, err)
	}
	waitForRuntimeCondition(t, "fixture-backed session visible", 5*time.Second, func() bool {
		current, err := harness.GetSession(ctx, session.ID)
		return err == nil && current.ID == session.ID
	})
	return session
}

func createSessionHTTPFailure(
	t testing.TB,
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	request aghcontract.CreateSessionRequest,
) (int, aghcontract.ErrorPayload) {
	t.Helper()

	body, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("json.Marshal(create session request) error = %v", err)
	}
	httpRequest, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		harness.HTTPURL("/api/sessions"),
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("http.NewRequestWithContext(create session) error = %v", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	response, err := harness.HTTPClient.Do(httpRequest)
	if err != nil {
		t.Fatalf("HTTP create session error = %v", err)
	}
	var payload aghcontract.ErrorPayload
	decodeErr := json.NewDecoder(response.Body).Decode(&payload)
	closeErr := response.Body.Close()
	if decodeErr != nil {
		t.Fatalf("decode HTTP create session failure error = %v", decodeErr)
	}
	if closeErr != nil {
		t.Fatalf("close HTTP create session failure body error = %v", closeErr)
	}
	return response.StatusCode, payload
}

func providerModelListHTTP(
	t testing.TB,
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	providerID string,
	view string,
) (int, aghcontract.ProviderModelListResponse) {
	t.Helper()

	path := "/api/model-catalog/providers/" + url.PathEscape(providerID) + "/models"
	if view != "" {
		path += "?view=" + url.QueryEscape(view)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, harness.HTTPURL(path), nil)
	if err != nil {
		t.Fatalf("http.NewRequestWithContext(provider models) error = %v", err)
	}
	response, err := harness.HTTPClient.Do(request)
	if err != nil {
		t.Fatalf("HTTP provider models error = %v", err)
	}
	var payload aghcontract.ProviderModelListResponse
	decodeErr := json.NewDecoder(response.Body).Decode(&payload)
	closeErr := response.Body.Close()
	if decodeErr != nil {
		t.Fatalf("decode HTTP provider models error = %v", decodeErr)
	}
	if closeErr != nil {
		t.Fatalf("close HTTP provider models body error = %v", closeErr)
	}
	return response.StatusCode, payload
}

func joinTranscriptContent(messages []transcript.UIMessage) string {
	return transcript.JoinUIMessageText(messages)
}

func sessionTranscriptMessages(response aghcontract.SessionTranscriptResponse) []transcript.UIMessage {
	return transcript.MessagesFromEntries(response.Entries)
}
