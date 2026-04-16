//go:build integration

package udsapi

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/api/core"
	automationpkg "github.com/pedronauck/agh/internal/automation"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/resources"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store/globaldb"
	taskpkg "github.com/pedronauck/agh/internal/task"
	toolspkg "github.com/pedronauck/agh/internal/tools"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func TestUDSFullRoundTripWithRealSessionManager(t *testing.T) {
	runtime := newIntegrationRuntime(t)

	statusResp := mustUnixRequest(t, runtime.client, http.MethodGet, "http://unix/api/daemon/status", nil, nil)
	if statusResp.StatusCode != http.StatusOK {
		t.Fatalf("daemon status = %d, want %d", statusResp.StatusCode, http.StatusOK)
	}
	_ = statusResp.Body.Close()

	createResp := mustUnixRequest(t, runtime.client, http.MethodPost, "http://unix/api/sessions", []byte(`{"agent_name":"coder","name":"demo","workspace_path":"`+runtime.workspace+`"}`), nil)
	if createResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(createResp.Body)
		_ = createResp.Body.Close()
		t.Fatalf("create session status = %d, want %d; body=%s", createResp.StatusCode, http.StatusCreated, string(body))
	}
	var created struct {
		Session sessionPayload `json:"session"`
	}
	decodeHTTPJSON(t, createResp, &created)
	if created.Session.ID == "" {
		t.Fatal("expected created session id")
	}

	listResp := mustUnixRequest(t, runtime.client, http.MethodGet, "http://unix/api/sessions", nil, nil)
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("list sessions status = %d, want %d", listResp.StatusCode, http.StatusOK)
	}
	var listed struct {
		Sessions []sessionPayload `json:"sessions"`
	}
	decodeHTTPJSON(t, listResp, &listed)
	if len(listed.Sessions) != 1 || listed.Sessions[0].ID != created.Session.ID {
		t.Fatalf("listed sessions = %#v", listed.Sessions)
	}

	promptResp := mustUnixRequest(t, runtime.client, http.MethodPost, "http://unix/api/sessions/"+created.Session.ID+"/prompt", []byte(`{"message":"hello"}`), nil)
	if promptResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(promptResp.Body)
		_ = promptResp.Body.Close()
		t.Fatalf("prompt status = %d, want %d; body=%s", promptResp.StatusCode, http.StatusOK, string(body))
	}
	promptBody, err := io.ReadAll(promptResp.Body)
	_ = promptResp.Body.Close()
	if err != nil {
		t.Fatalf("io.ReadAll(prompt SSE) error = %v", err)
	}
	promptEvents := parseSSE(t, string(promptBody))
	if len(promptEvents) < 2 {
		t.Fatalf("prompt SSE events = %d, want at least 2; body=%s", len(promptEvents), string(promptBody))
	}
	if promptEvents[0].Event != "agent_message" || promptEvents[len(promptEvents)-1].Event != "done" {
		t.Fatalf("prompt SSE event types = first:%s last:%s", promptEvents[0].Event, promptEvents[len(promptEvents)-1].Event)
	}

	eventsResp := mustUnixRequest(t, runtime.client, http.MethodGet, "http://unix/api/sessions/"+created.Session.ID+"/events", nil, nil)
	if eventsResp.StatusCode != http.StatusOK {
		t.Fatalf("session events status = %d, want %d", eventsResp.StatusCode, http.StatusOK)
	}
	var events struct {
		Events []sessionEventPayload `json:"events"`
	}
	decodeHTTPJSON(t, eventsResp, &events)
	if len(events.Events) < 2 {
		t.Fatalf("persisted session events = %d, want at least 2", len(events.Events))
	}

	stopResp := mustUnixRequest(t, runtime.client, http.MethodDelete, "http://unix/api/sessions/"+created.Session.ID, nil, nil)
	if stopResp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(stopResp.Body)
		_ = stopResp.Body.Close()
		t.Fatalf("stop session status = %d, want %d; body=%s", stopResp.StatusCode, http.StatusNoContent, string(body))
	}
	_ = stopResp.Body.Close()
}

func TestUDSMemoryRoundTripAndConsolidate(t *testing.T) {
	runtime := newIntegrationRuntime(t)

	writeResp := mustUnixRequest(t, runtime.client, http.MethodPut, "http://unix/api/memory/integration.md", []byte(`{"scope":"global","content":"`+escapeJSON(memoryDocument(t, "Integration", "desc", memory.MemoryTypeUser, "hello integration"))+`"}`), nil)
	if writeResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(writeResp.Body)
		_ = writeResp.Body.Close()
		t.Fatalf("write status = %d, want %d; body=%s", writeResp.StatusCode, http.StatusOK, string(body))
	}
	_ = writeResp.Body.Close()

	readResp := mustUnixRequest(t, runtime.client, http.MethodGet, "http://unix/api/memory/integration.md?scope=global", nil, nil)
	if readResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(readResp.Body)
		_ = readResp.Body.Close()
		t.Fatalf("read status = %d, want %d; body=%s", readResp.StatusCode, http.StatusOK, string(body))
	}
	var readPayload memoryReadResponse
	decodeHTTPJSON(t, readResp, &readPayload)
	if !strings.Contains(readPayload.Content, "hello integration") {
		t.Fatalf("content = %q, want written body", readPayload.Content)
	}

	deleteResp := mustUnixRequest(t, runtime.client, http.MethodDelete, "http://unix/api/memory/integration.md?scope=global", nil, nil)
	if deleteResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(deleteResp.Body)
		_ = deleteResp.Body.Close()
		t.Fatalf("delete status = %d, want %d; body=%s", deleteResp.StatusCode, http.StatusOK, string(body))
	}
	_ = deleteResp.Body.Close()

	resp := mustUnixRequest(t, runtime.client, http.MethodPost, "http://unix/api/memory/consolidate", []byte(`{"workspace":"`+runtime.workspace+`"}`), nil)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		t.Fatalf("consolidate status = %d, want %d; body=%s", resp.StatusCode, http.StatusOK, string(body))
	}

	var payload memoryConsolidateResponse
	decodeHTTPJSON(t, resp, &payload)
	if !payload.Triggered || runtime.dream.calls != 1 {
		t.Fatalf("payload = %#v dream.calls=%d, want triggered once", payload, runtime.dream.calls)
	}
}

func TestUDSResourceCRUDRoundTrip(t *testing.T) {
	runtime := newIntegrationRuntime(t)

	createResp := mustUnixRequest(
		t,
		runtime.client,
		http.MethodPut,
		"http://unix/api/resources/bundle.activation/demo",
		[]byte(`{"scope":{"kind":"global"},"spec":{"enabled":true}}`),
		nil,
	)
	if createResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(createResp.Body)
		_ = createResp.Body.Close()
		t.Fatalf("create resource status = %d, want %d; body=%s", createResp.StatusCode, http.StatusCreated, string(body))
	}
	var created contract.ResourceResponse
	decodeHTTPJSON(t, createResp, &created)
	if created.Record.Version != 1 {
		t.Fatalf("created version = %d, want 1", created.Record.Version)
	}
	if strings.TrimSpace(string(created.Record.Spec)) != `{"enabled":true}` {
		t.Fatalf("created spec = %s, want enabled true", string(created.Record.Spec))
	}

	updateResp := mustUnixRequest(
		t,
		runtime.client,
		http.MethodPut,
		"http://unix/api/resources/bundle.activation/demo",
		[]byte(fmt.Sprintf(`{"scope":{"kind":"global"},"expected_version":%d,"spec":{"enabled":false}}`, created.Record.Version)),
		nil,
	)
	if updateResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(updateResp.Body)
		_ = updateResp.Body.Close()
		t.Fatalf("update resource status = %d, want %d; body=%s", updateResp.StatusCode, http.StatusOK, string(body))
	}
	var updated contract.ResourceResponse
	decodeHTTPJSON(t, updateResp, &updated)
	if updated.Record.Version != 2 {
		t.Fatalf("updated version = %d, want 2", updated.Record.Version)
	}
	if strings.TrimSpace(string(updated.Record.Spec)) != `{"enabled":false}` {
		t.Fatalf("updated spec = %s, want enabled false", string(updated.Record.Spec))
	}

	getResp := mustUnixRequest(t, runtime.client, http.MethodGet, "http://unix/api/resources/bundle.activation/demo", nil, nil)
	if getResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(getResp.Body)
		_ = getResp.Body.Close()
		t.Fatalf("get resource status = %d, want %d; body=%s", getResp.StatusCode, http.StatusOK, string(body))
	}
	var fetched contract.ResourceResponse
	decodeHTTPJSON(t, getResp, &fetched)
	if fetched.Record.Version != updated.Record.Version {
		t.Fatalf("fetched version = %d, want %d", fetched.Record.Version, updated.Record.Version)
	}

	listResp := mustUnixRequest(
		t,
		runtime.client,
		http.MethodGet,
		"http://unix/api/resources/bundle.activation?scope_kind=global",
		nil,
		nil,
	)
	if listResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(listResp.Body)
		_ = listResp.Body.Close()
		t.Fatalf("list resources status = %d, want %d; body=%s", listResp.StatusCode, http.StatusOK, string(body))
	}
	var listed contract.ResourcesResponse
	decodeHTTPJSON(t, listResp, &listed)
	if len(listed.Records) != 1 || listed.Records[0].ID != "demo" || listed.Records[0].Version != updated.Record.Version {
		t.Fatalf("listed records = %#v, want updated demo record", listed.Records)
	}
}

func TestUDSToolResourceCRUDRoundTripTriggersProjection(t *testing.T) {
	runtime := newIntegrationRuntime(t)

	createResp := mustUnixRequest(
		t,
		runtime.client,
		http.MethodPut,
		"http://unix/api/resources/tool/lookup",
		[]byte(`{
			"scope":{"kind":"global"},
			"spec":{
				"name":" lookup ",
				"description":" search workspace ",
				"input_schema":{"type":"object"},
				"read_only":true,
				"source":"dynamic"
			}
		}`),
		nil,
	)
	if createResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(createResp.Body)
		_ = createResp.Body.Close()
		t.Fatalf("create tool resource status = %d, want %d; body=%s", createResp.StatusCode, http.StatusCreated, string(body))
	}
	var created contract.ResourceResponse
	decodeHTTPJSON(t, createResp, &created)
	if got, want := strings.TrimSpace(string(created.Record.Spec)), `{"name":"lookup","description":"search workspace","input_schema":{"type":"object"},"read_only":true,"source":"dynamic"}`; got != want {
		t.Fatalf("created tool spec = %s, want %s", got, want)
	}

	waitForProjectedToolRevision(t, runtime, 1)
	revision, records := runtime.toolCatalog.snapshot()
	if revision != 1 || len(records) != 1 || records[0].Spec.Name != "lookup" {
		t.Fatalf("tool projection after create = revision:%d records:%#v, want lookup@1", revision, records)
	}

	updateResp := mustUnixRequest(
		t,
		runtime.client,
		http.MethodPut,
		"http://unix/api/resources/tool/lookup",
		[]byte(fmt.Sprintf(`{
			"scope":{"kind":"global"},
			"expected_version":%d,
			"spec":{
				"name":"lookup",
				"description":"search workspace v2",
				"input_schema":{"type":"object"},
				"read_only":true,
				"source":"dynamic"
			}
		}`, created.Record.Version)),
		nil,
	)
	if updateResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(updateResp.Body)
		_ = updateResp.Body.Close()
		t.Fatalf("update tool resource status = %d, want %d; body=%s", updateResp.StatusCode, http.StatusOK, string(body))
	}
	var updated contract.ResourceResponse
	decodeHTTPJSON(t, updateResp, &updated)
	waitForProjectedToolRevision(t, runtime, 2)

	_, records = runtime.toolCatalog.snapshot()
	if got, want := records[0].Spec.Description, "search workspace v2"; got != want {
		t.Fatalf("projected tool description = %q, want %q", got, want)
	}
}

func TestUDSDeleteResourceRejectsStaleVersionAndRequiresCurrentVersion(t *testing.T) {
	runtime := newIntegrationRuntime(t)

	createResp := mustUnixRequest(
		t,
		runtime.client,
		http.MethodPut,
		"http://unix/api/resources/bundle.activation/demo",
		[]byte(`{"scope":{"kind":"global"},"spec":{"enabled":true}}`),
		nil,
	)
	if createResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(createResp.Body)
		_ = createResp.Body.Close()
		t.Fatalf("create resource status = %d, want %d; body=%s", createResp.StatusCode, http.StatusCreated, string(body))
	}
	var created contract.ResourceResponse
	decodeHTTPJSON(t, createResp, &created)

	updateResp := mustUnixRequest(
		t,
		runtime.client,
		http.MethodPut,
		"http://unix/api/resources/bundle.activation/demo",
		[]byte(fmt.Sprintf(`{"scope":{"kind":"global"},"expected_version":%d,"spec":{"enabled":false}}`, created.Record.Version)),
		nil,
	)
	if updateResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(updateResp.Body)
		_ = updateResp.Body.Close()
		t.Fatalf("update resource status = %d, want %d; body=%s", updateResp.StatusCode, http.StatusOK, string(body))
	}
	var updated contract.ResourceResponse
	decodeHTTPJSON(t, updateResp, &updated)

	staleDelete := mustUnixRequest(
		t,
		runtime.client,
		http.MethodDelete,
		"http://unix/api/resources/bundle.activation/demo",
		[]byte(fmt.Sprintf(`{"expected_version":%d}`, created.Record.Version)),
		nil,
	)
	if staleDelete.StatusCode != http.StatusConflict {
		body, _ := io.ReadAll(staleDelete.Body)
		_ = staleDelete.Body.Close()
		t.Fatalf("stale delete status = %d, want %d; body=%s", staleDelete.StatusCode, http.StatusConflict, string(body))
	}
	_ = staleDelete.Body.Close()

	deleteResp := mustUnixRequest(
		t,
		runtime.client,
		http.MethodDelete,
		"http://unix/api/resources/bundle.activation/demo",
		[]byte(fmt.Sprintf(`{"expected_version":%d}`, updated.Record.Version)),
		nil,
	)
	if deleteResp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(deleteResp.Body)
		_ = deleteResp.Body.Close()
		t.Fatalf("delete resource status = %d, want %d; body=%s", deleteResp.StatusCode, http.StatusNoContent, string(body))
	}
	_ = deleteResp.Body.Close()

	getResp := mustUnixRequest(t, runtime.client, http.MethodGet, "http://unix/api/resources/bundle.activation/demo", nil, nil)
	if getResp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(getResp.Body)
		_ = getResp.Body.Close()
		t.Fatalf("get deleted resource status = %d, want %d; body=%s", getResp.StatusCode, http.StatusNotFound, string(body))
	}
}

func TestUDSAutomationJobsRoundTrip(t *testing.T) {
	runtime := newIntegrationRuntime(t)

	createResp := mustUnixRequest(t, runtime.client, http.MethodPost, "http://unix/api/automation/jobs", []byte(`{"scope":"global","name":"nightly-review","agent_name":"coder","prompt":"review repo","schedule":{"mode":"every","interval":"1h"}}`), nil)
	if createResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(createResp.Body)
		_ = createResp.Body.Close()
		t.Fatalf("create job status = %d, want %d; body=%s", createResp.StatusCode, http.StatusCreated, string(body))
	}
	var created contract.JobResponse
	decodeHTTPJSON(t, createResp, &created)
	if created.Job.ID == "" {
		t.Fatal("expected created automation job id")
	}

	updateResp := mustUnixRequest(t, runtime.client, http.MethodPatch, "http://unix/api/automation/jobs/"+created.Job.ID, []byte(`{"prompt":"review repo now"}`), nil)
	if updateResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(updateResp.Body)
		_ = updateResp.Body.Close()
		t.Fatalf("update job status = %d, want %d; body=%s", updateResp.StatusCode, http.StatusOK, string(body))
	}
	var updated contract.JobResponse
	decodeHTTPJSON(t, updateResp, &updated)
	if updated.Job.Prompt != "review repo now" {
		t.Fatalf("updated job prompt = %q, want %q", updated.Job.Prompt, "review repo now")
	}

	triggerResp := mustUnixRequest(t, runtime.client, http.MethodPost, "http://unix/api/automation/jobs/"+created.Job.ID+"/trigger", nil, nil)
	if triggerResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(triggerResp.Body)
		_ = triggerResp.Body.Close()
		t.Fatalf("trigger job status = %d, want %d; body=%s", triggerResp.StatusCode, http.StatusOK, string(body))
	}
	var run contract.RunResponse
	decodeHTTPJSON(t, triggerResp, &run)
	if run.Run.JobID != created.Job.ID {
		t.Fatalf("job run = %#v, want job_id %q", run.Run, created.Job.ID)
	}

	runsResp := mustUnixRequest(t, runtime.client, http.MethodGet, "http://unix/api/automation/jobs/"+created.Job.ID+"/runs", nil, nil)
	if runsResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(runsResp.Body)
		_ = runsResp.Body.Close()
		t.Fatalf("job runs status = %d, want %d; body=%s", runsResp.StatusCode, http.StatusOK, string(body))
	}
	var runs contract.RunsResponse
	decodeHTTPJSON(t, runsResp, &runs)
	if !containsAutomationRun(runs.Runs, run.Run.ID) {
		t.Fatalf("job runs missing %q: %#v", run.Run.ID, runs.Runs)
	}

	deleteResp := mustUnixRequest(t, runtime.client, http.MethodDelete, "http://unix/api/automation/jobs/"+created.Job.ID, nil, nil)
	if deleteResp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(deleteResp.Body)
		_ = deleteResp.Body.Close()
		t.Fatalf("delete job status = %d, want %d; body=%s", deleteResp.StatusCode, http.StatusNoContent, string(body))
	}
	_ = deleteResp.Body.Close()
}

func TestUDSAutomationTriggerRunsAndOmitsWebhookRoutes(t *testing.T) {
	runtime := newIntegrationRuntime(t)

	resolveResp := mustUnixRequest(t, runtime.client, http.MethodPost, "http://unix/api/workspaces/resolve", []byte(`{"path":"`+runtime.workspace+`"}`), nil)
	if resolveResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resolveResp.Body)
		_ = resolveResp.Body.Close()
		t.Fatalf("resolve workspace status = %d, want %d; body=%s", resolveResp.StatusCode, http.StatusOK, string(body))
	}
	var resolved contract.WorkspaceResponse
	decodeHTTPJSON(t, resolveResp, &resolved)

	createResp := mustUnixRequest(t, runtime.client, http.MethodPost, "http://unix/api/automation/triggers", []byte(`{"scope":"workspace","workspace_id":"`+resolved.Workspace.ID+`","name":"session-stop-review","agent_name":"coder","prompt":"review {{ index .Data \"session_id\" }}","event":"session.stopped","filter":{"data.session_type":"user"}}`), nil)
	if createResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(createResp.Body)
		_ = createResp.Body.Close()
		t.Fatalf("create trigger status = %d, want %d; body=%s", createResp.StatusCode, http.StatusCreated, string(body))
	}
	var created contract.TriggerResponse
	decodeHTTPJSON(t, createResp, &created)
	if created.Trigger.ID == "" {
		t.Fatal("expected created automation trigger id")
	}

	updateResp := mustUnixRequest(t, runtime.client, http.MethodPatch, "http://unix/api/automation/triggers/"+created.Trigger.ID, []byte(`{"prompt":"inspect {{ index .Data \"session_id\" }}"}`), nil)
	if updateResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(updateResp.Body)
		_ = updateResp.Body.Close()
		t.Fatalf("update trigger status = %d, want %d; body=%s", updateResp.StatusCode, http.StatusOK, string(body))
	}
	var updated contract.TriggerResponse
	decodeHTTPJSON(t, updateResp, &updated)
	if updated.Trigger.Prompt != `inspect {{ index .Data "session_id" }}` {
		t.Fatalf("updated trigger prompt = %q", updated.Trigger.Prompt)
	}

	sessionID := createIntegrationSession(t, runtime)
	stopIntegrationSession(t, runtime, sessionID)

	var runs contract.RunsResponse
	deadline := time.After(2 * time.Second)
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		runsResp := mustUnixRequest(t, runtime.client, http.MethodGet, "http://unix/api/automation/triggers/"+created.Trigger.ID+"/runs", nil, nil)
		if runsResp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(runsResp.Body)
			_ = runsResp.Body.Close()
			t.Fatalf("trigger runs status = %d, want %d; body=%s", runsResp.StatusCode, http.StatusOK, string(body))
		}

		runs = contract.RunsResponse{}
		decodeHTTPJSON(t, runsResp, &runs)
		if len(runs.Runs) > 0 {
			break
		}

		select {
		case <-deadline:
			t.Fatalf("expected trigger run history, got %#v", runs.Runs)
		case <-ticker.C:
		}
	}
	runID := runs.Runs[0].ID

	runResp := mustUnixRequest(t, runtime.client, http.MethodGet, "http://unix/api/automation/runs/"+runID, nil, nil)
	if runResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(runResp.Body)
		_ = runResp.Body.Close()
		t.Fatalf("get trigger run status = %d, want %d; body=%s", runResp.StatusCode, http.StatusOK, string(body))
	}
	var run contract.RunResponse
	decodeHTTPJSON(t, runResp, &run)
	if run.Run.TriggerID != created.Trigger.ID {
		t.Fatalf("trigger run = %#v, want trigger_id %q", run.Run, created.Trigger.ID)
	}

	webhookResp := mustUnixRequest(t, runtime.client, http.MethodPost, "http://unix/api/webhooks/global/deploy-review--wbh_test", []byte(`{"payload":"deploy"}`), nil)
	if webhookResp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(webhookResp.Body)
		_ = webhookResp.Body.Close()
		t.Fatalf("webhook route status = %d, want %d; body=%s", webhookResp.StatusCode, http.StatusNotFound, string(body))
	}
	_ = webhookResp.Body.Close()

	deleteResp := mustUnixRequest(t, runtime.client, http.MethodDelete, "http://unix/api/automation/triggers/"+created.Trigger.ID, nil, nil)
	if deleteResp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(deleteResp.Body)
		_ = deleteResp.Body.Close()
		t.Fatalf("delete trigger status = %d, want %d; body=%s", deleteResp.StatusCode, http.StatusNoContent, string(body))
	}
	_ = deleteResp.Body.Close()
}

func TestUDSSessionStreamReconnectsWithLastEventID(t *testing.T) {
	runtime := newIntegrationRuntime(t)
	sessionID := createIntegrationSession(t, runtime)
	sendPrompt(t, runtime, sessionID, "hello")
	stopIntegrationSession(t, runtime, sessionID)

	streamResp := mustUnixRequest(t, runtime.client, http.MethodGet, "http://unix/api/sessions/"+sessionID+"/stream", nil, nil)
	if streamResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(streamResp.Body)
		_ = streamResp.Body.Close()
		t.Fatalf("session stream status = %d, want %d; body=%s", streamResp.StatusCode, http.StatusOK, string(body))
	}
	initial := collectLiveSSEUntilEvent(t, streamResp.Body, session.EventTypeSessionStopped, 2*time.Second)
	_ = streamResp.Body.Close()
	if len(initial) < 3 {
		t.Fatalf("initial stream events = %d, want 3", len(initial))
	}
	if initial[len(initial)-1].Event != session.EventTypeSessionStopped {
		t.Fatalf("last event = %q, want %q", initial[len(initial)-1].Event, session.EventTypeSessionStopped)
	}

	headers := map[string]string{"Last-Event-ID": initial[0].ID}
	replayResp := mustUnixRequest(t, runtime.client, http.MethodGet, "http://unix/api/sessions/"+sessionID+"/stream", nil, headers)
	if replayResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(replayResp.Body)
		_ = replayResp.Body.Close()
		t.Fatalf("replay stream status = %d, want %d; body=%s", replayResp.StatusCode, http.StatusOK, string(body))
	}
	replayed := collectLiveSSE(t, replayResp.Body, 2, 2*time.Second)
	_ = replayResp.Body.Close()
	if len(replayed) < 2 {
		t.Fatalf("replayed events = %d, want 2", len(replayed))
	}
	if replayed[0].ID != initial[1].ID {
		t.Fatalf("replayed first id = %q, want %q", replayed[0].ID, initial[1].ID)
	}
	if replayed[0].ID == initial[0].ID {
		t.Fatalf("replayed first id = %q, should skip Last-Event-ID", replayed[0].ID)
	}
}

func TestUDSShutdownWaitsForInflightRequests(t *testing.T) {
	homePaths := newTestHomePaths(t)
	socketPath := shortSocketPath(t)
	cfg := aghconfig.DefaultWithHome(homePaths)
	cfg.Daemon.Socket = socketPath

	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	server, err := New(
		WithHomePaths(homePaths),
		WithConfig(&cfg),
		WithSocketPath(socketPath),
		WithLogger(discardLogger()),
		WithSessionManager(stubSessionManager{
			ListAllFn: func(context.Context) ([]*session.Info, error) {
				entered <- struct{}{}
				<-release
				return []*session.Info{newSessionInfo("sess-1")}, nil
			},
		}),
		WithTaskService(stubTaskManager{}),
		WithObserver(stubObserver{
			HealthFn: func(context.Context) (observe.Health, error) { return observe.Health{Status: "ok"}, nil },
		}),
		WithWorkspaceResolver(stubWorkspaceService{}),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := server.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	released := false
	defer func() {
		if !released {
			close(release)
		}
		_ = server.Shutdown(context.Background())
	}()

	client := newUnixClient(t, socketPath)
	respCh := make(chan *http.Response, 1)
	errCh := make(chan error, 1)
	go func() {
		resp, err := client.Get("http://unix/api/sessions")
		if err != nil {
			errCh <- err
			return
		}
		respCh <- resp
	}()

	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for in-flight request")
	}

	shutdownDone := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		shutdownDone <- server.Shutdown(ctx)
	}()

	select {
	case err := <-shutdownDone:
		t.Fatalf("Shutdown() returned early: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	close(release)
	released = true

	select {
	case err := <-shutdownDone:
		if err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for shutdown")
	}

	select {
	case err := <-errCh:
		t.Fatalf("request error = %v", err)
	case resp := <-respCh:
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			t.Fatalf("request status = %d, want %d; body=%s", resp.StatusCode, http.StatusOK, string(body))
		}
		_ = resp.Body.Close()
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for request response")
	}
}

func TestUDSSessionChannelRoundTrip(t *testing.T) {
	runtime := newIntegrationRuntime(t)

	createResp := mustUnixRequest(t, runtime.client, http.MethodPost, "http://unix/api/sessions", []byte(`{"agent_name":"coder","workspace_path":"`+runtime.workspace+`","channel":"builders"}`), nil)
	if createResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(createResp.Body)
		_ = createResp.Body.Close()
		t.Fatalf("create session status = %d, want %d; body=%s", createResp.StatusCode, http.StatusCreated, string(body))
	}
	var created struct {
		Session sessionPayload `json:"session"`
	}
	decodeHTTPJSON(t, createResp, &created)
	if created.Session.Channel != "builders" {
		t.Fatalf("created.Session.Channel = %q, want %q", created.Session.Channel, "builders")
	}

	listResp := mustUnixRequest(t, runtime.client, http.MethodGet, "http://unix/api/sessions", nil, nil)
	if listResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(listResp.Body)
		_ = listResp.Body.Close()
		t.Fatalf("list sessions status = %d, want %d; body=%s", listResp.StatusCode, http.StatusOK, string(body))
	}
	var listed struct {
		Sessions []sessionPayload `json:"sessions"`
	}
	decodeHTTPJSON(t, listResp, &listed)
	if got, want := len(listed.Sessions), 1; got != want {
		t.Fatalf("len(listed.Sessions) = %d, want %d", got, want)
	}
	if listed.Sessions[0].Channel != "builders" {
		t.Fatalf("listed.Sessions[0].Channel = %q, want %q", listed.Sessions[0].Channel, "builders")
	}

	stopIntegrationSession(t, runtime, created.Session.ID)

	statusResp := mustUnixRequest(t, runtime.client, http.MethodGet, "http://unix/api/sessions/"+created.Session.ID, nil, nil)
	if statusResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(statusResp.Body)
		_ = statusResp.Body.Close()
		t.Fatalf("status after stop = %d, want %d; body=%s", statusResp.StatusCode, http.StatusOK, string(body))
	}
	var stopped struct {
		Session sessionPayload `json:"session"`
	}
	decodeHTTPJSON(t, statusResp, &stopped)
	if stopped.Session.Channel != "builders" || stopped.Session.State != session.StateStopped {
		t.Fatalf("stopped session = %#v, want stopped builders session", stopped.Session)
	}

	resumeResp := mustUnixRequest(t, runtime.client, http.MethodPost, "http://unix/api/sessions/"+created.Session.ID+"/resume", nil, nil)
	if resumeResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resumeResp.Body)
		_ = resumeResp.Body.Close()
		t.Fatalf("resume session status = %d, want %d; body=%s", resumeResp.StatusCode, http.StatusOK, string(body))
	}
	var resumed struct {
		Session sessionPayload `json:"session"`
	}
	decodeHTTPJSON(t, resumeResp, &resumed)
	if resumed.Session.Channel != "builders" || resumed.Session.State != session.StateActive {
		t.Fatalf("resumed session = %#v, want active builders session", resumed.Session)
	}
}

func TestUDSTaskRoutesRoundTrip(t *testing.T) {
	runtime := newIntegrationRuntime(t)

	created := createIntegrationTask(t, runtime, []byte(`{
		"scope":"global",
		"title":"Ship task routes",
		"description":"Expose the transport routes",
		"network_channel":"builders",
		"owner":{"kind":"pool","ref":"ops"},
		"metadata":{"priority":"high"}
	}`))
	if created.ID == "" {
		t.Fatal("expected created task id")
	}
	if created.Scope != taskpkg.ScopeGlobal {
		t.Fatalf("created scope = %q, want %q", created.Scope, taskpkg.ScopeGlobal)
	}
	if created.NetworkChannel != "builders" {
		t.Fatalf("created network_channel = %q, want %q", created.NetworkChannel, "builders")
	}
	if created.Owner == nil || created.Owner.Kind != taskpkg.OwnerKindPool || created.Owner.Ref != "ops" {
		t.Fatalf("created owner = %#v, want pool/ops", created.Owner)
	}
	if created.Origin.Kind != taskpkg.OriginKindUDS {
		t.Fatalf("created origin.kind = %q, want %q", created.Origin.Kind, taskpkg.OriginKindUDS)
	}
	if created.CreatedBy.Ref != "local-user" {
		t.Fatalf("created created_by.ref = %q, want %q", created.CreatedBy.Ref, "local-user")
	}
	if got := strings.TrimSpace(string(created.Metadata)); got != `{"priority":"high"}` {
		t.Fatalf("created metadata = %s, want %s", got, `{"priority":"high"}`)
	}

	listResp := mustUnixRequest(t, runtime.client, http.MethodGet, "http://unix/api/tasks?scope=global&status=ready&owner_kind=pool&owner_ref=ops&network_channel=builders", nil, nil)
	if listResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(listResp.Body)
		_ = listResp.Body.Close()
		t.Fatalf("list tasks status = %d, want %d; body=%s", listResp.StatusCode, http.StatusOK, string(body))
	}
	var listed contract.TasksResponse
	decodeHTTPJSON(t, listResp, &listed)
	if len(listed.Tasks) != 1 || listed.Tasks[0].ID != created.ID {
		t.Fatalf("listed tasks = %#v, want created task", listed.Tasks)
	}

	getResp := mustUnixRequest(t, runtime.client, http.MethodGet, "http://unix/api/tasks/"+created.ID, nil, nil)
	if getResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(getResp.Body)
		_ = getResp.Body.Close()
		t.Fatalf("get task status = %d, want %d; body=%s", getResp.StatusCode, http.StatusOK, string(body))
	}
	var detail contract.TaskDetailResponse
	decodeHTTPJSON(t, getResp, &detail)
	if detail.Task.Task.ID != created.ID {
		t.Fatalf("detail task id = %q, want %q", detail.Task.Task.ID, created.ID)
	}
	if len(detail.Task.Children) != 0 || len(detail.Task.Runs) != 0 {
		t.Fatalf("detail task children/runs = %#v, want empty", detail.Task)
	}

	updateResp := mustUnixRequest(t, runtime.client, http.MethodPatch, "http://unix/api/tasks/"+created.ID, []byte(`{
		"title":"Ship task routes now",
		"description":"Expose the task and run transports everywhere",
		"network_channel":"ops",
		"clear_owner":true
	}`), nil)
	if updateResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(updateResp.Body)
		_ = updateResp.Body.Close()
		t.Fatalf("update task status = %d, want %d; body=%s", updateResp.StatusCode, http.StatusOK, string(body))
	}
	var updated contract.TaskResponse
	decodeHTTPJSON(t, updateResp, &updated)
	if updated.Task.Title != "Ship task routes now" {
		t.Fatalf("updated title = %q, want %q", updated.Task.Title, "Ship task routes now")
	}
	if updated.Task.Description != "Expose the task and run transports everywhere" {
		t.Fatalf("updated description = %q", updated.Task.Description)
	}
	if updated.Task.NetworkChannel != "ops" {
		t.Fatalf("updated network_channel = %q, want %q", updated.Task.NetworkChannel, "ops")
	}
	if updated.Task.Owner != nil {
		t.Fatalf("updated owner = %#v, want nil", updated.Task.Owner)
	}

	updatedListResp := mustUnixRequest(t, runtime.client, http.MethodGet, "http://unix/api/tasks?scope=global&status=ready&network_channel=ops", nil, nil)
	if updatedListResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(updatedListResp.Body)
		_ = updatedListResp.Body.Close()
		t.Fatalf("updated list tasks status = %d, want %d; body=%s", updatedListResp.StatusCode, http.StatusOK, string(body))
	}
	var updatedList contract.TasksResponse
	decodeHTTPJSON(t, updatedListResp, &updatedList)
	if len(updatedList.Tasks) != 1 || updatedList.Tasks[0].ID != created.ID {
		t.Fatalf("updated list tasks = %#v, want created task", updatedList.Tasks)
	}
}

func TestUDSTaskRunLifecycleRoutesRoundTrip(t *testing.T) {
	runtime := newIntegrationRuntime(t)
	created := createIntegrationTask(t, runtime, []byte(`{"scope":"global","title":"Run task routes"}`))

	queued := enqueueIntegrationTaskRun(t, runtime, created.ID, `{"idempotency_key":"enqueue-1","network_channel":"builders"}`)
	if queued.Status != taskpkg.TaskRunStatusQueued {
		t.Fatalf("queued status = %q, want %q", queued.Status, taskpkg.TaskRunStatusQueued)
	}
	if queued.NetworkChannel != "builders" {
		t.Fatalf("queued network_channel = %q, want %q", queued.NetworkChannel, "builders")
	}

	listQueuedResp := mustUnixRequest(t, runtime.client, http.MethodGet, "http://unix/api/tasks/"+created.ID+"/runs?status=queued&limit=1", nil, nil)
	if listQueuedResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(listQueuedResp.Body)
		_ = listQueuedResp.Body.Close()
		t.Fatalf("list queued runs status = %d, want %d; body=%s", listQueuedResp.StatusCode, http.StatusOK, string(body))
	}
	var queuedList contract.TaskRunsResponse
	decodeHTTPJSON(t, listQueuedResp, &queuedList)
	if len(queuedList.Runs) != 1 || queuedList.Runs[0].ID != queued.ID {
		t.Fatalf("queued runs = %#v, want queued run", queuedList.Runs)
	}

	claimed := claimIntegrationTaskRun(t, runtime, queued.ID, `{"idempotency_key":"claim-1"}`)
	if claimed.Status != taskpkg.TaskRunStatusClaimed {
		t.Fatalf("claimed status = %q, want %q", claimed.Status, taskpkg.TaskRunStatusClaimed)
	}
	if claimed.ClaimedBy == nil || claimed.ClaimedBy.Ref != "local-user" {
		t.Fatalf("claimed claimed_by = %#v, want local-user", claimed.ClaimedBy)
	}

	startResp := mustUnixRequest(t, runtime.client, http.MethodPost, "http://unix/api/task-runs/"+queued.ID+"/start", []byte(`{"idempotency_key":"start-1"}`), nil)
	if startResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(startResp.Body)
		_ = startResp.Body.Close()
		t.Fatalf("start run status = %d, want %d; body=%s", startResp.StatusCode, http.StatusOK, string(body))
	}
	var started contract.TaskRunResponse
	decodeHTTPJSON(t, startResp, &started)
	if started.Run.Status != taskpkg.TaskRunStatusRunning {
		t.Fatalf("started status = %q, want %q", started.Run.Status, taskpkg.TaskRunStatusRunning)
	}
	if started.Run.SessionID == "" {
		t.Fatal("expected started run session id")
	}

	completeResp := mustUnixRequest(t, runtime.client, http.MethodPost, "http://unix/api/task-runs/"+queued.ID+"/complete", []byte(`{"result":{"ok":true}}`), nil)
	if completeResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(completeResp.Body)
		_ = completeResp.Body.Close()
		t.Fatalf("complete run status = %d, want %d; body=%s", completeResp.StatusCode, http.StatusOK, string(body))
	}
	var completed contract.TaskRunResponse
	decodeHTTPJSON(t, completeResp, &completed)
	if completed.Run.Status != taskpkg.TaskRunStatusCompleted {
		t.Fatalf("completed status = %q, want %q", completed.Run.Status, taskpkg.TaskRunStatusCompleted)
	}

	secondRun := enqueueIntegrationTaskRun(t, runtime, created.ID, `{"idempotency_key":"enqueue-2"}`)
	claimIntegrationTaskRun(t, runtime, secondRun.ID, `{"idempotency_key":"claim-2"}`)
	attachResp := mustUnixRequest(t, runtime.client, http.MethodPost, "http://unix/api/task-runs/"+secondRun.ID+"/attach-session", []byte(`{"session_id":"sess-resume-1"}`), nil)
	if attachResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(attachResp.Body)
		_ = attachResp.Body.Close()
		t.Fatalf("attach run session status = %d, want %d; body=%s", attachResp.StatusCode, http.StatusOK, string(body))
	}
	var attached contract.TaskRunResponse
	decodeHTTPJSON(t, attachResp, &attached)
	if attached.Run.Status != taskpkg.TaskRunStatusStarting {
		t.Fatalf("attached status = %q, want %q", attached.Run.Status, taskpkg.TaskRunStatusStarting)
	}
	if attached.Run.SessionID != "sess-resume-1" {
		t.Fatalf("attached session_id = %q, want %q", attached.Run.SessionID, "sess-resume-1")
	}

	failResp := mustUnixRequest(t, runtime.client, http.MethodPost, "http://unix/api/task-runs/"+secondRun.ID+"/fail", []byte(`{"error":"boom","metadata":{"step":"attach"}}`), nil)
	if failResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(failResp.Body)
		_ = failResp.Body.Close()
		t.Fatalf("fail run status = %d, want %d; body=%s", failResp.StatusCode, http.StatusOK, string(body))
	}
	var failed contract.TaskRunResponse
	decodeHTTPJSON(t, failResp, &failed)
	if failed.Run.Status != taskpkg.TaskRunStatusFailed {
		t.Fatalf("failed status = %q, want %q", failed.Run.Status, taskpkg.TaskRunStatusFailed)
	}

	thirdRun := enqueueIntegrationTaskRun(t, runtime, created.ID, `{"idempotency_key":"enqueue-3"}`)
	cancelResp := mustUnixRequest(t, runtime.client, http.MethodPost, "http://unix/api/task-runs/"+thirdRun.ID+"/cancel", []byte(`{"reason":"operator cancelled"}`), nil)
	if cancelResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(cancelResp.Body)
		_ = cancelResp.Body.Close()
		t.Fatalf("cancel run status = %d, want %d; body=%s", cancelResp.StatusCode, http.StatusOK, string(body))
	}
	var cancelled contract.TaskRunResponse
	decodeHTTPJSON(t, cancelResp, &cancelled)
	if cancelled.Run.Status != taskpkg.TaskRunStatusCanceled {
		t.Fatalf("cancelled status = %q, want %q", cancelled.Run.Status, taskpkg.TaskRunStatusCanceled)
	}

	finalRunsResp := mustUnixRequest(t, runtime.client, http.MethodGet, "http://unix/api/tasks/"+created.ID+"/runs", nil, nil)
	if finalRunsResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(finalRunsResp.Body)
		_ = finalRunsResp.Body.Close()
		t.Fatalf("final list runs status = %d, want %d; body=%s", finalRunsResp.StatusCode, http.StatusOK, string(body))
	}
	var finalRuns contract.TaskRunsResponse
	decodeHTTPJSON(t, finalRunsResp, &finalRuns)
	if len(finalRuns.Runs) != 3 {
		t.Fatalf("len(final runs) = %d, want 3", len(finalRuns.Runs))
	}

	seenStatuses := map[taskpkg.RunStatus]int{}
	for _, run := range finalRuns.Runs {
		seenStatuses[run.Status]++
	}
	if seenStatuses[taskpkg.TaskRunStatusCompleted] != 1 || seenStatuses[taskpkg.TaskRunStatusFailed] != 1 || seenStatuses[taskpkg.TaskRunStatusCanceled] != 1 {
		t.Fatalf("final run statuses = %#v, want one completed, failed, cancelled", seenStatuses)
	}
}

type integrationRuntime struct {
	client         *http.Client
	server         *Server
	manager        *session.Manager
	tasks          *taskpkg.Service
	observer       *observe.Observer
	registry       *globaldb.GlobalDB
	bridges        *integrationBridgeService
	memory         *memory.Store
	dream          *integrationDreamTrigger
	resourceDriver resources.ReconcileDriver
	toolCatalog    *integrationToolCatalog
	socket         string
	workspace      string
}

type integrationToolCatalog struct {
	mu       sync.Mutex
	revision int64
	records  []resources.Record[toolspkg.Tool]
}

type integrationToolPlan struct {
	revision   int64
	operations int
	records    []resources.Record[toolspkg.Tool]
}

func (p *integrationToolPlan) Kind() resources.ResourceKind { return toolspkg.ToolResourceKind }
func (p *integrationToolPlan) Revision() int64              { return p.revision }
func (p *integrationToolPlan) OperationCount() int          { return p.operations }

type integrationToolProjector struct {
	catalog *integrationToolCatalog
}

func (p *integrationToolProjector) Kind() resources.ResourceKind {
	return toolspkg.ToolResourceKind
}

func (p *integrationToolProjector) DependsOn() []resources.ResourceKind {
	return nil
}

func (p *integrationToolProjector) Build(
	_ context.Context,
	records []resources.Record[toolspkg.Tool],
) (resources.ProjectionPlan, error) {
	var revision int64
	cloned := make([]resources.Record[toolspkg.Tool], 0, len(records))
	for _, record := range records {
		if record.Version > revision {
			revision = record.Version
		}
		next := record
		next.Spec = cloneIntegrationTool(record.Spec)
		cloned = append(cloned, next)
	}
	return &integrationToolPlan{
		revision:   revision,
		operations: len(records),
		records:    cloned,
	}, nil
}

func (p *integrationToolProjector) Apply(_ context.Context, plan resources.ProjectionPlan) error {
	typed, ok := plan.(*integrationToolPlan)
	if !ok {
		return fmt.Errorf("integration tool plan type = %T, want *integrationToolPlan", plan)
	}
	p.catalog.replace(typed.revision, typed.records)
	return nil
}

func (c *integrationToolCatalog) replace(revision int64, records []resources.Record[toolspkg.Tool]) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.revision = revision
	c.records = cloneIntegrationToolRecords(records)
}

func (c *integrationToolCatalog) snapshot() (int64, []resources.Record[toolspkg.Tool]) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.revision, cloneIntegrationToolRecords(c.records)
}

func cloneIntegrationToolRecords(records []resources.Record[toolspkg.Tool]) []resources.Record[toolspkg.Tool] {
	if len(records) == 0 {
		return nil
	}
	cloned := make([]resources.Record[toolspkg.Tool], 0, len(records))
	for _, record := range records {
		next := record
		next.Spec = cloneIntegrationTool(record.Spec)
		cloned = append(cloned, next)
	}
	return cloned
}

func cloneIntegrationTool(spec toolspkg.Tool) toolspkg.Tool {
	cloned := spec
	if len(spec.InputSchema) > 0 {
		cloned.InputSchema = append([]byte(nil), spec.InputSchema...)
	}
	return cloned
}

func waitForProjectedToolRevision(t *testing.T, runtime integrationRuntime, want int64) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		revision, _ := runtime.toolCatalog.snapshot()
		if revision >= want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	revision, records := runtime.toolCatalog.snapshot()
	t.Fatalf("timed out waiting for projected tool revision %d (got %d, records=%#v)", want, revision, records)
}

type integrationTaskSessionExecutor struct {
	started int
}

func (e *integrationTaskSessionExecutor) StartTaskSession(
	_ context.Context,
	_ *taskpkg.StartTaskSession,
) (*taskpkg.SessionRef, error) {
	e.started++
	return &taskpkg.SessionRef{SessionID: fmt.Sprintf("task-sess-%d", e.started)}, nil
}

func (*integrationTaskSessionExecutor) AttachTaskSession(_ context.Context, _ string, sessionID string) (*taskpkg.SessionRef, error) {
	return &taskpkg.SessionRef{SessionID: sessionID}, nil
}

func (*integrationTaskSessionExecutor) RequestTaskStop(context.Context, string, taskpkg.StopReason) error {
	return nil
}

func (*integrationTaskSessionExecutor) ForceTaskStop(context.Context, string, taskpkg.StopReason) error {
	return nil
}

type integrationDreamTrigger struct {
	enabled   bool
	triggered bool
	reason    string
	last      time.Time
	calls     int
}

type integrationBridgeSecretStore interface {
	ListBridgeSecretBindings(context.Context, string) ([]bridgepkg.BridgeSecretBinding, error)
	PutBridgeSecretBinding(context.Context, bridgepkg.BridgeSecretBinding) error
	DeleteBridgeSecretBinding(context.Context, string, string) error
}

type integrationBridgeService struct {
	*bridgepkg.Service
	store     integrationBridgeSecretStore
	providers []bridgepkg.BridgeProvider
}

var _ core.BridgeService = (*integrationBridgeService)(nil)

func newIntegrationBridgeService(store bridgepkg.RegistryStore) *integrationBridgeService {
	secretStore, _ := store.(integrationBridgeSecretStore)
	return &integrationBridgeService{
		Service: bridgepkg.NewRegistry(store),
		store:   secretStore,
	}
}

func (s *integrationBridgeService) StartInstance(ctx context.Context, id string) (*bridgepkg.BridgeInstance, error) {
	if _, err := s.UpdateInstanceState(ctx, bridgepkg.UpdateInstanceStateRequest{
		ID:      id,
		Enabled: true,
		Status:  bridgepkg.BridgeStatusStarting,
	}); err != nil {
		return nil, fmt.Errorf("start bridge instance %q: %w", id, err)
	}
	instance, err := s.UpdateInstanceState(ctx, bridgepkg.UpdateInstanceStateRequest{
		ID:      id,
		Enabled: true,
		Status:  bridgepkg.BridgeStatusReady,
	})
	if err != nil {
		return nil, fmt.Errorf("mark bridge instance %q ready: %w", id, err)
	}
	return instance, nil
}

func (s *integrationBridgeService) StopInstance(ctx context.Context, id string) (*bridgepkg.BridgeInstance, error) {
	instance, err := s.UpdateInstanceState(ctx, bridgepkg.UpdateInstanceStateRequest{
		ID:      id,
		Enabled: false,
		Status:  bridgepkg.BridgeStatusDisabled,
	})
	if err != nil {
		return nil, fmt.Errorf("stop bridge instance %q: %w", id, err)
	}
	return instance, nil
}

func (s *integrationBridgeService) RestartInstance(ctx context.Context, id string) (*bridgepkg.BridgeInstance, error) {
	if _, err := s.UpdateInstanceState(ctx, bridgepkg.UpdateInstanceStateRequest{
		ID:      id,
		Enabled: true,
		Status:  bridgepkg.BridgeStatusStarting,
	}); err != nil {
		return nil, fmt.Errorf("restart bridge instance %q: %w", id, err)
	}
	instance, err := s.UpdateInstanceState(ctx, bridgepkg.UpdateInstanceStateRequest{
		ID:      id,
		Enabled: true,
		Status:  bridgepkg.BridgeStatusReady,
	})
	if err != nil {
		return nil, fmt.Errorf("mark restarted bridge instance %q ready: %w", id, err)
	}
	return instance, nil
}

func (s *integrationBridgeService) ListProviders(context.Context) ([]bridgepkg.BridgeProvider, error) {
	providers := make([]bridgepkg.BridgeProvider, 0, len(s.providers))
	providers = append(providers, s.providers...)
	return providers, nil
}

func (s *integrationBridgeService) ListSecretBindings(ctx context.Context, bridgeInstanceID string) ([]bridgepkg.BridgeSecretBinding, error) {
	if s == nil || s.store == nil {
		return nil, errors.New("integration bridge secret store is not configured")
	}
	return s.store.ListBridgeSecretBindings(ctx, bridgeInstanceID)
}

func (s *integrationBridgeService) PutSecretBinding(ctx context.Context, binding bridgepkg.BridgeSecretBinding) error {
	if s == nil || s.store == nil {
		return errors.New("integration bridge secret store is not configured")
	}
	return s.store.PutBridgeSecretBinding(ctx, binding)
}

func (s *integrationBridgeService) DeleteSecretBinding(ctx context.Context, bridgeInstanceID string, bindingName string) error {
	if s == nil || s.store == nil {
		return errors.New("integration bridge secret store is not configured")
	}
	return s.store.DeleteBridgeSecretBinding(ctx, bridgeInstanceID, bindingName)
}

func (t *integrationDreamTrigger) Trigger(context.Context, string) (bool, string, error) {
	t.calls++
	return t.triggered, t.reason, nil
}

func (t *integrationDreamTrigger) LastConsolidatedAt() (time.Time, error) {
	return t.last, nil
}

func (t *integrationDreamTrigger) Enabled() bool {
	return t.enabled
}

type integrationNotifierFanout struct {
	notifiers []session.Notifier
}

func (f *integrationNotifierFanout) OnSessionCreated(ctx context.Context, sess *session.Session) {
	for _, notifier := range f.notifiers {
		notifier.OnSessionCreated(ctx, sess)
	}
}

func (f *integrationNotifierFanout) OnSessionStopped(ctx context.Context, sess *session.Session) {
	for _, notifier := range f.notifiers {
		notifier.OnSessionStopped(ctx, sess)
	}
}

func (f *integrationNotifierFanout) OnAgentEvent(ctx context.Context, sessionID string, event any) {
	for _, notifier := range f.notifiers {
		notifier.OnAgentEvent(ctx, sessionID, event)
	}
}

type integrationDriver struct {
	mu       sync.Mutex
	nextPID  int
	nextSess int
	states   map[*session.AgentProcess]chan struct{}
}

func newIntegrationDriver() *integrationDriver {
	return &integrationDriver{
		nextPID:  2000,
		nextSess: 1,
		states:   make(map[*session.AgentProcess]chan struct{}),
	}
}

func (d *integrationDriver) Start(_ context.Context, opts acp.StartOpts) (*session.AgentProcess, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.nextPID++
	d.nextSess++
	done := make(chan struct{})
	sessionID := strings.TrimSpace(opts.ResumeSessionID)
	if sessionID == "" {
		sessionID = fmt.Sprintf("acp-session-%d", d.nextSess)
	}

	proc := session.NewAgentProcess(session.AgentProcessOptions{
		PID:       d.nextPID,
		AgentName: opts.AgentName,
		Command:   opts.Command,
		Cwd:       opts.Cwd,
		SessionID: sessionID,
		Caps: acp.Caps{
			SupportsLoadSession: true,
			SupportedModels:     []string{"fake-model"},
		},
		StartedAt: time.Now().UTC(),
		Done:      done,
		Wait: func() error {
			<-done
			return nil
		},
	})
	d.states[proc] = done
	return proc, nil
}

func (d *integrationDriver) Prompt(_ context.Context, proc *session.AgentProcess, req acp.PromptRequest) (<-chan acp.AgentEvent, error) {
	ch := make(chan acp.AgentEvent, 2)
	ch <- acp.AgentEvent{
		Type:      "agent_message",
		SessionID: proc.SessionID,
		TurnID:    req.TurnID,
		Timestamp: time.Now().UTC(),
		Text:      req.Message,
	}
	ch <- acp.AgentEvent{
		Type:       "done",
		SessionID:  proc.SessionID,
		TurnID:     req.TurnID,
		Timestamp:  time.Now().UTC(),
		StopReason: "end_turn",
	}
	close(ch)
	return ch, nil
}

func (d *integrationDriver) Cancel(context.Context, *session.AgentProcess) error {
	return nil
}

func (d *integrationDriver) Stop(_ context.Context, proc *session.AgentProcess) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	done, ok := d.states[proc]
	if !ok {
		return nil
	}
	select {
	case <-done:
	default:
		close(done)
	}
	delete(d.states, proc)
	return nil
}

func newIntegrationRuntime(t *testing.T) integrationRuntime {
	t.Helper()

	homePaths := newTestHomePaths(t)
	socketPath := shortSocketPath(t)
	writeAgentDef(t, homePaths, "coder")

	workspace := t.TempDir()
	cfg := aghconfig.DefaultWithHome(homePaths)
	cfg.Daemon.Socket = socketPath
	cfg.Providers = map[string]aghconfig.ProviderConfig{
		"fake": {Command: "fake-agent"},
	}

	registry, err := globaldb.OpenGlobalDB(context.Background(), homePaths.DatabaseFile)
	if err != nil {
		t.Fatalf("OpenGlobalDB() error = %v", err)
	}
	t.Cleanup(func() {
		if err := registry.Close(context.Background()); err != nil {
			t.Fatalf("registry.Close() error = %v", err)
		}
	})

	fanout := &integrationNotifierFanout{}
	resolver, err := workspacepkg.NewResolver(
		registry,
		workspacepkg.WithHomePaths(homePaths),
		workspacepkg.WithLogger(discardLogger()),
		workspacepkg.WithConfigLoader(func(string) (aghconfig.Config, error) { return cfg, nil }),
	)
	if err != nil {
		t.Fatalf("workspace.NewResolver() error = %v", err)
	}
	manager, err := session.NewManager(
		session.WithHomePaths(homePaths),
		session.WithWorkspaceResolver(resolver),
		session.WithLogger(discardLogger()),
		session.WithDriver(newIntegrationDriver()),
		session.WithNotifier(fanout),
	)
	if err != nil {
		t.Fatalf("session.NewManager() error = %v", err)
	}

	observer, err := observe.New(
		context.Background(),
		observe.WithHomePaths(homePaths),
		observe.WithRegistry(registry),
		observe.WithSessionSource(manager),
		observe.WithLogger(discardLogger()),
	)
	if err != nil {
		t.Fatalf("observe.New() error = %v", err)
	}
	fanout.notifiers = append(fanout.notifiers, observer)

	memoryStore := memory.NewStore(homePaths.MemoryDir)
	if err := memoryStore.EnsureDirs(); err != nil {
		t.Fatalf("memoryStore.EnsureDirs() error = %v", err)
	}
	bridgeService := newIntegrationBridgeService(registry)
	dreamTrigger := &integrationDreamTrigger{
		enabled:   true,
		triggered: true,
		last:      time.Date(2026, 4, 4, 3, 30, 0, 0, time.UTC),
	}

	automationManager, err := automationpkg.New(
		automationpkg.WithStore(registry),
		automationpkg.WithSessions(manager),
		automationpkg.WithWorkspaceResolver(resolver),
		automationpkg.WithConfig(cfg.Automation),
		automationpkg.WithLogger(discardLogger()),
		automationpkg.WithGlobalWorkspacePath(homePaths.HomeDir),
	)
	if err != nil {
		t.Fatalf("automation.New() error = %v", err)
	}
	if err := automationManager.Start(context.Background()); err != nil {
		t.Fatalf("automationManager.Start() error = %v", err)
	}
	fanout.notifiers = append(fanout.notifiers, automationManager.SessionObserver())
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := automationManager.Shutdown(ctx); err != nil {
			t.Fatalf("automationManager.Shutdown() error = %v", err)
		}
	})

	taskExecutor := &integrationTaskSessionExecutor{}
	taskManager, err := taskpkg.NewManager(
		taskpkg.WithStore(registry),
		taskpkg.WithSessionExecutor(taskExecutor),
	)
	if err != nil {
		t.Fatalf("task.NewManager() error = %v", err)
	}

	resourceKernel, err := resources.NewKernel(registry.DB())
	if err != nil {
		t.Fatalf("resources.NewKernel() error = %v", err)
	}
	resourceCodecs := resources.NewCodecRegistry()
	toolCodec, err := toolspkg.NewResourceCodec()
	if err != nil {
		t.Fatalf("toolspkg.NewResourceCodec() error = %v", err)
	}
	if err := resources.RegisterCodec(resourceCodecs, toolCodec); err != nil {
		t.Fatalf("resources.RegisterCodec(tool) error = %v", err)
	}
	toolCatalog := &integrationToolCatalog{}
	toolRegistration, err := resources.NewTypedProjectorRegistration(toolCodec, &integrationToolProjector{catalog: toolCatalog})
	if err != nil {
		t.Fatalf("resources.NewTypedProjectorRegistration(tool) error = %v", err)
	}
	resourceDriver, err := resources.NewReconcileDriver(
		resourceKernel,
		resources.MutationActor{
			Kind: resources.MutationActorKindDaemon,
			ID:   "uds-integration",
			Source: resources.ResourceSource{
				Kind: resources.ResourceSourceKind("daemon"),
				ID:   "uds-integration",
			},
			MaxScope: resources.ResourceScope{Kind: resources.ResourceScopeKindGlobal},
		},
		[]resources.ProjectorRegistration{toolRegistration},
		resources.WithReconcileLogger(discardLogger()),
	)
	if err != nil {
		t.Fatalf("resources.NewReconcileDriver() error = %v", err)
	}
	t.Cleanup(func() {
		if err := resourceDriver.Close(context.Background()); err != nil {
			t.Fatalf("resourceDriver.Close() error = %v", err)
		}
	})
	resourceService, err := core.NewOperatorResourceService(&core.ResourceServiceConfig{
		RawStore:      resourceKernel,
		CodecRegistry: resourceCodecs,
		Trigger: func(ctx context.Context, kind resources.ResourceKind, reason resources.ReconcileReason) error {
			if kind.Normalize() != toolspkg.ToolResourceKind {
				return nil
			}
			return resourceDriver.Trigger(ctx, kind, reason)
		},
	})
	if err != nil {
		t.Fatalf("core.NewOperatorResourceService() error = %v", err)
	}

	server, err := New(
		WithHomePaths(homePaths),
		WithConfig(&cfg),
		WithSocketPath(socketPath),
		WithLogger(discardLogger()),
		WithSessionManager(manager),
		WithTaskService(taskManager),
		WithObserver(observer),
		WithResourceService(resourceService),
		WithAutomation(automationManager),
		WithBridgeService(bridgeService),
		WithWorkspaceResolver(resolver),
		WithMemoryStore(memoryStore),
		WithDreamTrigger(dreamTrigger),
		WithPollInterval(10*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("udsapi.New() error = %v", err)
	}
	if err := server.Start(context.Background()); err != nil {
		t.Fatalf("server.Start() error = %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			t.Fatalf("server.Shutdown() error = %v", err)
		}
	})

	return integrationRuntime{
		client:         newUnixClient(t, socketPath),
		server:         server,
		manager:        manager,
		tasks:          taskManager,
		observer:       observer,
		registry:       registry,
		bridges:        bridgeService,
		memory:         memoryStore,
		dream:          dreamTrigger,
		resourceDriver: resourceDriver,
		toolCatalog:    toolCatalog,
		socket:         socketPath,
		workspace:      workspace,
	}
}

func createIntegrationSession(t *testing.T, runtime integrationRuntime) string {
	t.Helper()

	resp := mustUnixRequest(t, runtime.client, http.MethodPost, "http://unix/api/sessions", []byte(`{"agent_name":"coder","workspace_path":"`+runtime.workspace+`"}`), nil)
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		t.Fatalf("create session status = %d, want %d; body=%s", resp.StatusCode, http.StatusCreated, string(body))
	}
	var created struct {
		Session sessionPayload `json:"session"`
	}
	decodeHTTPJSON(t, resp, &created)
	return created.Session.ID
}

func createIntegrationTask(t *testing.T, runtime integrationRuntime, body []byte) contract.TaskPayload {
	t.Helper()

	resp := mustUnixRequest(t, runtime.client, http.MethodPost, "http://unix/api/tasks", body, nil)
	if resp.StatusCode != http.StatusCreated {
		payload, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		t.Fatalf("create task status = %d, want %d; body=%s", resp.StatusCode, http.StatusCreated, string(payload))
	}
	var created contract.TaskResponse
	decodeHTTPJSON(t, resp, &created)
	return created.Task
}

func enqueueIntegrationTaskRun(t *testing.T, runtime integrationRuntime, taskID string, body string) contract.TaskRunPayload {
	t.Helper()

	resp := mustUnixRequest(t, runtime.client, http.MethodPost, "http://unix/api/tasks/"+taskID+"/runs", []byte(body), nil)
	if resp.StatusCode != http.StatusCreated {
		payload, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		t.Fatalf("enqueue run status = %d, want %d; body=%s", resp.StatusCode, http.StatusCreated, string(payload))
	}
	var created contract.TaskRunResponse
	decodeHTTPJSON(t, resp, &created)
	return created.Run
}

func claimIntegrationTaskRun(t *testing.T, runtime integrationRuntime, runID string, body string) contract.TaskRunPayload {
	t.Helper()

	resp := mustUnixRequest(t, runtime.client, http.MethodPost, "http://unix/api/task-runs/"+runID+"/claim", []byte(body), nil)
	if resp.StatusCode != http.StatusOK {
		payload, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		t.Fatalf("claim run status = %d, want %d; body=%s", resp.StatusCode, http.StatusOK, string(payload))
	}
	var claimed contract.TaskRunResponse
	decodeHTTPJSON(t, resp, &claimed)
	return claimed.Run
}

func sendPrompt(t *testing.T, runtime integrationRuntime, sessionID string, message string) {
	t.Helper()

	resp := mustUnixRequest(t, runtime.client, http.MethodPost, "http://unix/api/sessions/"+sessionID+"/prompt", []byte(`{"message":"`+message+`"}`), nil)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		t.Fatalf("prompt status = %d, want %d; body=%s", resp.StatusCode, http.StatusOK, string(body))
	}
	_, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()
}

func stopIntegrationSession(t *testing.T, runtime integrationRuntime, sessionID string) {
	t.Helper()

	resp := mustUnixRequest(t, runtime.client, http.MethodDelete, "http://unix/api/sessions/"+sessionID, nil, nil)
	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		t.Fatalf("stop status = %d, want %d; body=%s", resp.StatusCode, http.StatusNoContent, string(body))
	}
	_ = resp.Body.Close()
}

func mustUnixRequest(t *testing.T, client *http.Client, method, url string, body []byte, headers map[string]string) *http.Response {
	t.Helper()

	var reader io.Reader
	if len(body) > 0 {
		reader = strings.NewReader(string(body))
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("client.Do() error = %v", err)
	}
	return resp
}

func decodeHTTPJSON(t *testing.T, resp *http.Response, dest any) {
	t.Helper()

	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		t.Fatalf("io.ReadAll(response) error = %v", err)
	}
	if err := json.Unmarshal(body, dest); err != nil {
		t.Fatalf("json.Unmarshal(response) error = %v; body=%s", err, string(body))
	}
}

func containsAutomationRun(runs []contract.RunPayload, id string) bool {
	for _, run := range runs {
		if run.ID == id {
			return true
		}
	}
	return false
}

func collectLiveSSE(t *testing.T, body io.ReadCloser, want int, timeout time.Duration) []sseRecord {
	t.Helper()

	records := make([]sseRecord, 0, want)
	recordCh := make(chan sseRecord, want+1)
	errCh := make(chan error, 1)

	go func() {
		scanner := bufio.NewScanner(body)
		current := sseRecord{}
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				recordCh <- current
				current = sseRecord{}
				continue
			}
			switch {
			case strings.HasPrefix(line, "id: "):
				current.ID = strings.TrimPrefix(line, "id: ")
			case strings.HasPrefix(line, "event: "):
				current.Event = strings.TrimPrefix(line, "event: ")
			case strings.HasPrefix(line, "data: "):
				current.Data = append(current.Data, []byte(strings.TrimPrefix(line, "data: "))...)
			}
		}
		if current.Event != "" || current.ID != "" || len(current.Data) > 0 {
			recordCh <- current
		}
		if err := scanner.Err(); err != nil && !errors.Is(err, os.ErrClosed) {
			errCh <- err
			return
		}
		close(recordCh)
	}()

	deadline := time.NewTimer(timeout)
	defer deadline.Stop()

	for len(records) < want {
		select {
		case record, ok := <-recordCh:
			if !ok {
				return records
			}
			records = append(records, record)
		case err := <-errCh:
			t.Fatalf("SSE scanner error = %v", err)
		case <-deadline.C:
			t.Fatalf("timed out waiting for %d SSE events; got %d", want, len(records))
		}
	}

	return records
}

func collectLiveSSEUntilEvent(t *testing.T, body io.ReadCloser, wantEvent string, timeout time.Duration) []sseRecord {
	t.Helper()

	records := make([]sseRecord, 0, 4)
	recordCh := make(chan sseRecord, 8)
	errCh := make(chan error, 1)

	go func() {
		scanner := bufio.NewScanner(body)
		current := sseRecord{}
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				recordCh <- current
				current = sseRecord{}
				continue
			}
			switch {
			case strings.HasPrefix(line, "id: "):
				current.ID = strings.TrimPrefix(line, "id: ")
			case strings.HasPrefix(line, "event: "):
				current.Event = strings.TrimPrefix(line, "event: ")
			case strings.HasPrefix(line, "data: "):
				current.Data = append(current.Data, []byte(strings.TrimPrefix(line, "data: "))...)
			}
		}
		if current.Event != "" || current.ID != "" || len(current.Data) > 0 {
			recordCh <- current
		}
		if err := scanner.Err(); err != nil && !errors.Is(err, os.ErrClosed) {
			errCh <- err
			return
		}
		close(recordCh)
	}()

	deadline := time.NewTimer(timeout)
	defer deadline.Stop()

	for {
		select {
		case record, ok := <-recordCh:
			if !ok {
				t.Fatalf("stream ended before event %q; got %d records", wantEvent, len(records))
			}
			records = append(records, record)
			if record.Event == wantEvent {
				return records
			}
		case err := <-errCh:
			t.Fatalf("SSE scanner error = %v", err)
		case <-deadline.C:
			t.Fatalf("timed out waiting for SSE event %q; got %d records", wantEvent, len(records))
		}
	}
}
