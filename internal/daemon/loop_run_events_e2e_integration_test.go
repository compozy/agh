//go:build integration && !windows

package daemon

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	aghcontract "github.com/compozy/agh/internal/api/contract"
	"github.com/compozy/agh/internal/testutil/acpmock"
	e2etest "github.com/compozy/agh/internal/testutil/e2e"
)

func TestDaemonE2ELoopRunEventsShouldStreamRichFramesAndResume(t *testing.T) {
	t.Parallel()

	t.Run("Should stream rich frames resume and isolate by workspace", func(t *testing.T) {
		t.Parallel()
		acpmock.RequireDriver(t)

		harness := e2etest.StartRuntimeHarness(t, e2etest.RuntimeHarnessOptions{
			MockAgents: []e2etest.MockAgentSpec{{
				FixturePath:  mockFixturePath(t, "loop_events_fixture.json"),
				FixtureAgent: "loop_events",
				AgentName:    "loop-events-agent",
			}},
		})

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		createLoopViaHTTP(t, ctx, harness, loopEventsDefinition())
		run := runLoopViaHTTP(t, ctx, harness, "loop-events-probe")
		waitForLoopRunStatus(t, ctx, harness, run.ID, aghcontract.LoopRunStatusDone)

		eventsPath := loopRunEventsPath(harness.WorkspaceID, run.ID, 0)
		events := readLoopRunSSEUntil(t, ctx, harness, eventsPath, func(events []loopRunSSEEvent) bool {
			return loopSSEKinds(events).Contains(
				string(aghcontract.LoopRunEventStatusChanged),
				string(aghcontract.LoopRunEventNodeRunning),
				string(aghcontract.LoopRunEventNodeSucceeded),
				string(aghcontract.LoopRunEventChannelMsg),
				string(aghcontract.LoopRunEventTokenTick),
			)
		})
		assertLoopSSEWorkspace(t, events, harness.WorkspaceID, run.ID)
		assertLoopSSEPayloadContains(t, events, aghcontract.LoopRunEventChannelMsg, "loop channel result")
		assertLoopSSEPayloadContains(t, events, aghcontract.LoopRunEventTokenTick, `"terminal":true`)

		afterSeq := firstLoopEventSeq(t, events, aghcontract.LoopRunEventNodeRunning)
		resumed := readLoopRunSSEUntil(
			t,
			ctx,
			harness,
			loopRunEventsPath(harness.WorkspaceID, run.ID, afterSeq),
			func(events []loopRunSSEEvent) bool {
				return loopSSEKinds(events).Contains(
					string(aghcontract.LoopRunEventNodeSucceeded),
					string(aghcontract.LoopRunEventTokenTick),
				)
			},
		)
		for _, event := range resumed {
			if event.Seq <= afterSeq {
				t.Fatalf("resumed event seq = %d, want > %d: %#v", event.Seq, afterSeq, resumed)
			}
		}

		foreign := readLoopRunSSEForDuration(
			t,
			harness,
			loopRunEventsPath("foreign-workspace", run.ID, 0),
			250*time.Millisecond,
		)
		if len(foreign) != 0 {
			t.Fatalf("foreign workspace stream events = %#v, want none", foreign)
		}
	})
}

func loopEventsDefinition() aghcontract.LoopDefinitionDocument {
	return aghcontract.LoopDefinitionDocument{
		APIVersion:  "agh.loop/v1",
		Kind:        "Loop",
		Concurrency: "allow",
		Meta: aghcontract.LoopDefinitionMeta{
			Name:        "loop-events-probe",
			Description: "Runtime E2E probe for rich Loop run SSE events.",
			Catalog: aghcontract.LoopCatalogMeta{
				UseWhen:  "Testing Loop run event streaming.",
				Keywords: []string{"test", "events"},
				Category: "Testing",
			},
		},
		Contract: aghcontract.LoopContract{
			Goal:             "Emit rich Loop run events.",
			DefinitionOfDone: "The probe action completes.",
			StopWhen:         "nodes.probe.status == 'succeeded'",
			IterationCap:     1,
			NoProgress: aghcontract.LoopNoProgress{
				Window:     2,
				HashFields: []string{"delivery_artifact"},
			},
			Budget: aghcontract.LoopBudget{
				Tokens:       0,
				WallClockSec: 0,
				OnExceeded:   aghcontract.LoopBudgetExceededHalt,
			},
			TerminalStates: []string{"done", "failed", "blocked", "exhausted", "stalled"},
		},
		Graph: aghcontract.LoopGraph{
			Nodes: []aghcontract.LoopGraphNode{{
				ID:    "probe",
				Class: aghcontract.LoopNodeClassAction,
				Kind:  "run-agent",
				Params: map[string]any{
					"agent":  "loop-events-agent",
					"prompt": "loop event probe",
					"output_schema": map[string]any{
						"type":     "object",
						"required": []any{"summary", "message"},
						"properties": map[string]any{
							"summary": map[string]any{"type": "string"},
							"message": map[string]any{"type": "string"},
						},
					},
				},
			}},
		},
		Start: []aghcontract.LoopStartBinding{
			{Kind: "manual"},
			{Kind: "http"},
			{Kind: "uds"},
		},
	}
}

func createLoopViaHTTP(
	t testing.TB,
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	def aghcontract.LoopDefinitionDocument,
) {
	t.Helper()
	var response aghcontract.LoopResponse
	path := "/api/workspaces/" + url.PathEscape(harness.WorkspaceID) + "/loops"
	if err := harness.HTTPJSON(ctx, http.MethodPost, path, aghcontract.CreateLoopRequest{Definition: &def}, &response); err != nil {
		t.Fatalf("HTTP create loop error = %v", err)
	}
	if response.Loop.Name != def.Meta.Name {
		t.Fatalf("created loop = %#v, want %q", response.Loop, def.Meta.Name)
	}
}

func runLoopViaHTTP(
	t testing.TB,
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	name string,
) aghcontract.LoopRunPayload {
	t.Helper()
	var response aghcontract.RunLoopResponse
	path := "/api/workspaces/" + url.PathEscape(harness.WorkspaceID) + "/loops/" + url.PathEscape(name) + "/run"
	if err := harness.HTTPJSON(ctx, http.MethodPost, path, aghcontract.RunLoopRequest{}, &response); err != nil {
		t.Fatalf("HTTP run loop error = %v", err)
	}
	if response.Run == nil {
		t.Fatalf("HTTP run loop response = %#v, want run", response)
	}
	return *response.Run
}

func waitForLoopRunStatus(
	t testing.TB,
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	runID string,
	want aghcontract.LoopRunStatus,
) {
	t.Helper()
	waitForRuntimeCondition(t, "loop run status "+string(want), 20*time.Second, func() bool {
		var response aghcontract.LoopRunResponse
		path := "/api/workspaces/" + url.PathEscape(harness.WorkspaceID) + "/loop-runs/" + url.PathEscape(runID)
		if err := harness.HTTPJSON(ctx, http.MethodGet, path, nil, &response); err != nil {
			return false
		}
		return response.Run.Status == want
	})
}

func loopRunEventsPath(workspaceID string, runID string, afterSeq int64) string {
	path := "/api/workspaces/" + url.PathEscape(workspaceID) + "/loop-runs/" + url.PathEscape(runID) + "/events"
	if afterSeq > 0 {
		path += "?after_sequence=" + strconv.FormatInt(afterSeq, 10)
	}
	return path
}

type loopRunSSEEvent struct {
	ID          string
	Event       string
	LoopRunID   string
	WorkspaceID string
	Seq         int64
	Kind        aghcontract.LoopRunEventKind
	Payload     json.RawMessage
}

func readLoopRunSSEUntil(
	t testing.TB,
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	path string,
	done func([]loopRunSSEEvent) bool,
) []loopRunSSEEvent {
	t.Helper()
	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	events, err := streamLoopRunSSE(streamCtx, harness, path, func(events []loopRunSSEEvent) bool {
		if done(events) {
			cancel()
			return true
		}
		return false
	})
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("streamLoopRunSSE(%q) error = %v", path, err)
	}
	if !done(events) {
		t.Fatalf("streamLoopRunSSE(%q) events = %#v, predicate not satisfied", path, events)
	}
	return events
}

func readLoopRunSSEForDuration(
	t testing.TB,
	harness *e2etest.RuntimeHarness,
	path string,
	duration time.Duration,
) []loopRunSSEEvent {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()
	events, err := streamLoopRunSSE(ctx, harness, path, func([]loopRunSSEEvent) bool { return false })
	if err != nil && !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Fatalf("streamLoopRunSSE(%q) error = %v", path, err)
	}
	return events
}

func streamLoopRunSSE(
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	path string,
	done func([]loopRunSSEEvent) bool,
) (events []loopRunSSEEvent, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, harness.HTTPURL(path), nil)
	if err != nil {
		return nil, err
	}
	resp, err := harness.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close loop SSE response body: %w", closeErr)
		}
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		payload, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("read loop SSE failure response: %w", readErr)
		}
		return nil, fmt.Errorf("loop SSE status %d: %s", resp.StatusCode, bytes.TrimSpace(payload))
	}
	return readLoopRunSSERecords(resp.Body, done)
}

func readLoopRunSSERecords(
	reader io.Reader,
	done func([]loopRunSSEEvent) bool,
) ([]loopRunSSEEvent, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	events := make([]loopRunSSEEvent, 0, 8)
	var id string
	var name string
	var data strings.Builder
	flush := func() error {
		if data.Len() == 0 {
			id = ""
			name = ""
			return nil
		}
		event, err := decodeLoopRunSSEEvent(id, name, data.String())
		if err != nil {
			return err
		}
		events = append(events, event)
		id = ""
		name = ""
		data.Reset()
		return nil
	}
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if err := flush(); err != nil {
				return events, err
			}
			if done(events) {
				return events, nil
			}
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		value = strings.TrimPrefix(value, " ")
		switch key {
		case "id":
			id = value
		case "event":
			name = value
		case "data":
			if data.Len() > 0 {
				data.WriteByte('\n')
			}
			data.WriteString(value)
		}
	}
	if err := flush(); err != nil {
		return events, err
	}
	if err := scanner.Err(); err != nil {
		return events, err
	}
	return events, nil
}

func decodeLoopRunSSEEvent(id string, name string, raw string) (loopRunSSEEvent, error) {
	var payload aghcontract.LoopRunEventPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return loopRunSSEEvent{}, err
	}
	return loopRunSSEEvent{
		ID:          id,
		Event:       name,
		LoopRunID:   payload.LoopRunID,
		WorkspaceID: payload.WorkspaceID,
		Seq:         payload.Seq,
		Kind:        payload.Kind,
		Payload:     payload.Payload,
	}, nil
}

type loopEventKindSet map[string]struct{}

func loopSSEKinds(events []loopRunSSEEvent) loopEventKindSet {
	kinds := make(loopEventKindSet, len(events))
	for _, event := range events {
		kinds[event.Event] = struct{}{}
		kinds[string(event.Kind)] = struct{}{}
	}
	return kinds
}

func (s loopEventKindSet) Contains(kinds ...string) bool {
	for _, kind := range kinds {
		if _, ok := s[kind]; !ok {
			return false
		}
	}
	return true
}

func assertLoopSSEWorkspace(t testing.TB, events []loopRunSSEEvent, workspaceID string, runID string) {
	t.Helper()
	for _, event := range events {
		if event.ID != strconv.FormatInt(event.Seq, 10) {
			t.Fatalf("event id/seq = %q/%d, want matching SSE id", event.ID, event.Seq)
		}
		if event.Event != string(event.Kind) {
			t.Fatalf("event name/kind = %q/%q, want matching named SSE kind", event.Event, event.Kind)
		}
		if event.WorkspaceID != workspaceID || event.LoopRunID != runID {
			t.Fatalf("event workspace/run = %s/%s, want %s/%s", event.WorkspaceID, event.LoopRunID, workspaceID, runID)
		}
	}
}

func assertLoopSSEPayloadContains(
	t testing.TB,
	events []loopRunSSEEvent,
	kind aghcontract.LoopRunEventKind,
	fragment string,
) {
	t.Helper()
	matched := make([]loopRunSSEEvent, 0)
	for _, event := range events {
		if event.Kind != kind {
			continue
		}
		matched = append(matched, event)
		if strings.Contains(string(event.Payload), fragment) {
			return
		}
	}
	if len(matched) > 0 {
		t.Fatalf("%s payloads = %#v, want fragment %q", kind, matched, fragment)
	}
	t.Fatalf("events = %#v, want kind %s", events, kind)
}

func firstLoopEventSeq(
	t testing.TB,
	events []loopRunSSEEvent,
	kind aghcontract.LoopRunEventKind,
) int64 {
	t.Helper()
	for _, event := range events {
		if event.Kind == kind {
			return event.Seq
		}
	}
	t.Fatalf("events = %#v, want kind %s", events, kind)
	return 0
}
