//go:build integration

package core_test

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/compozy/agh/internal/api/contract"
	"github.com/compozy/agh/internal/api/testutil"
	taskpkg "github.com/compozy/agh/internal/task"
	"github.com/gin-gonic/gin"
)

type taskRunTerminalIntegrationCapture struct {
	runID        string
	failure      taskpkg.RunFailure
	cancel       taskpkg.CancelRun
	actorContext taskpkg.ActorContext
}

func TestTaskRunTerminalHandlersPreserveHistoricalChannelBindingsIntegration(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		path             string
		body             []byte
		buildTaskManager func(*testing.T, *taskRunTerminalIntegrationCapture, time.Time) testutil.StubTaskManager
		assertCapture    func(*testing.T, *taskRunTerminalIntegrationCapture)
		wantStatus       taskpkg.RunStatus
		wantOriginRef    string
		wantError        string
		wantMetadataJSON string
		wantResultJSON   string
	}{
		{
			name: "Should preserve historical channel bindings in complete responses",
			path: "/task-runs/run-2/complete",
			body: []byte(`{"result":{"ok":true,"path":"historical-http-complete"}}`),
			buildTaskManager: func(t *testing.T, capture *taskRunTerminalIntegrationCapture, now time.Time) testutil.StubTaskManager {
				t.Helper()
				return testutil.StubTaskManager{
					CompleteRunFn: func(_ context.Context, runID string, result taskpkg.RunResult, actor taskpkg.ActorContext) (*taskpkg.Run, error) {
						capture.runID = runID
						capture.actorContext = actor
						return &taskpkg.Run{
							ID:                    runID,
							TaskID:                "task-1",
							Status:                taskpkg.TaskRunStatusCompleted,
							Attempt:               2,
							Origin:                actor.Origin,
							QueuedAt:              now,
							EndedAt:               now.Add(time.Minute),
							Result:                result.Value,
							NetworkChannel:        "builders",
							CoordinationChannelID: "builders",
						}, nil
					},
				}
			},
			assertCapture: func(t *testing.T, capture *taskRunTerminalIntegrationCapture) {
				t.Helper()
				if capture.runID != "run-2" {
					t.Fatalf("runID = %q, want %q", capture.runID, "run-2")
				}
				if capture.actorContext.Actor.Ref != "user-1" ||
					capture.actorContext.Origin.Ref != "tasks.complete_run" {
					t.Fatalf("actorContext = %#v", capture.actorContext)
				}
			},
			wantStatus:       taskpkg.TaskRunStatusCompleted,
			wantOriginRef:    "tasks.complete_run",
			wantMetadataJSON: "",
			wantResultJSON:   `{"ok":true,"path":"historical-http-complete"}`,
		},
		{
			name: "Should preserve historical channel bindings in fail responses",
			path: "/task-runs/run-2/fail",
			body: []byte(`{"error":"boom","metadata":{"step":"claim","mode":"historical-http"}}`),
			buildTaskManager: func(t *testing.T, capture *taskRunTerminalIntegrationCapture, now time.Time) testutil.StubTaskManager {
				t.Helper()
				return testutil.StubTaskManager{
					FailRunFn: func(_ context.Context, runID string, failure taskpkg.RunFailure, actor taskpkg.ActorContext) (*taskpkg.Run, error) {
						capture.runID = runID
						capture.failure = failure
						capture.actorContext = actor
						return &taskpkg.Run{
							ID:                    runID,
							TaskID:                "task-1",
							Status:                taskpkg.TaskRunStatusFailed,
							Attempt:               2,
							Origin:                actor.Origin,
							QueuedAt:              now,
							EndedAt:               now.Add(time.Minute),
							Error:                 failure.Error,
							Metadata:              failure.Metadata,
							NetworkChannel:        "builders",
							CoordinationChannelID: "builders",
						}, nil
					},
				}
			},
			assertCapture: func(t *testing.T, capture *taskRunTerminalIntegrationCapture) {
				t.Helper()
				if capture.runID != "run-2" {
					t.Fatalf("runID = %q, want %q", capture.runID, "run-2")
				}
				if capture.failure.Error != "boom" {
					t.Fatalf("failure = %#v, want error %q", capture.failure, "boom")
				}
				assertRawJSONEqual(
					t,
					"failure metadata",
					capture.failure.Metadata,
					`{"step":"claim","mode":"historical-http"}`,
				)
				if capture.actorContext.Actor.Ref != "user-1" || capture.actorContext.Origin.Ref != "tasks.fail_run" {
					t.Fatalf("actorContext = %#v", capture.actorContext)
				}
			},
			wantStatus:       taskpkg.TaskRunStatusFailed,
			wantOriginRef:    "tasks.fail_run",
			wantError:        "boom",
			wantMetadataJSON: `{"step":"claim","mode":"historical-http"}`,
		},
		{
			name: "Should preserve historical channel bindings in cancel responses",
			path: "/task-runs/run-2/cancel",
			body: []byte(`{"reason":"operator canceled","metadata":{"step":"cancel","mode":"historical-http"}}`),
			buildTaskManager: func(t *testing.T, capture *taskRunTerminalIntegrationCapture, now time.Time) testutil.StubTaskManager {
				t.Helper()
				return testutil.StubTaskManager{
					CancelRunFn: func(_ context.Context, runID string, req taskpkg.CancelRun, actor taskpkg.ActorContext) (*taskpkg.Run, error) {
						capture.runID = runID
						capture.cancel = req
						capture.actorContext = actor
						return &taskpkg.Run{
							ID:                    runID,
							TaskID:                "task-1",
							Status:                taskpkg.TaskRunStatusCanceled,
							Attempt:               2,
							Origin:                actor.Origin,
							QueuedAt:              now,
							EndedAt:               now.Add(time.Minute),
							Metadata:              req.Metadata,
							NetworkChannel:        "builders",
							CoordinationChannelID: "builders",
						}, nil
					},
				}
			},
			assertCapture: func(t *testing.T, capture *taskRunTerminalIntegrationCapture) {
				t.Helper()
				if capture.runID != "run-2" {
					t.Fatalf("runID = %q, want %q", capture.runID, "run-2")
				}
				if capture.cancel.Reason != "operator canceled" {
					t.Fatalf("cancel = %#v, want reason %q", capture.cancel, "operator canceled")
				}
				assertRawJSONEqual(
					t,
					"cancel metadata",
					capture.cancel.Metadata,
					`{"step":"cancel","mode":"historical-http"}`,
				)
				if capture.actorContext.Actor.Ref != "user-1" || capture.actorContext.Origin.Ref != "tasks.cancel_run" {
					t.Fatalf("actorContext = %#v", capture.actorContext)
				}
			},
			wantStatus:       taskpkg.TaskRunStatusCanceled,
			wantOriginRef:    "tasks.cancel_run",
			wantMetadataJSON: `{"step":"cancel","mode":"historical-http"}`,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			now := time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC)
			capture := &taskRunTerminalIntegrationCapture{}
			fixture := newHandlerFixtureWithTasks(
				t,
				testutil.StubSessionManager{},
				testutil.StubObserver{},
				tc.buildTaskManager(t, capture, now),
				testutil.StubWorkspaceService{},
				nil,
				nil,
			)
			fixture.Handlers.TaskActorContextResolver = func(_ *gin.Context, action string) (taskpkg.ActorContext, error) {
				return taskpkg.DeriveHumanActorContext("user-1", taskpkg.OriginKindHTTP, "tasks."+action)
			}

			resp := performRequest(t, fixture.Engine, http.MethodPost, tc.path, tc.body)
			if resp.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body=%s", resp.Code, http.StatusOK, resp.Body.String())
			}

			var payload contract.TaskRunResponse
			testutil.DecodeJSONResponse(t, resp, &payload)

			if payload.Run.Status != tc.wantStatus {
				t.Fatalf("payload.Run.Status = %q, want %q", payload.Run.Status, tc.wantStatus)
			}
			if payload.Run.NetworkChannel != "builders" || payload.Run.CoordinationChannelID != "builders" {
				t.Fatalf("payload.Run = %#v, want preserved historical channel bindings", payload.Run)
			}
			if payload.Run.Origin.Ref != tc.wantOriginRef {
				t.Fatalf("payload.Run.Origin.Ref = %q, want %q", payload.Run.Origin.Ref, tc.wantOriginRef)
			}
			if payload.Run.Error != tc.wantError {
				t.Fatalf("payload.Run.Error = %q, want %q", payload.Run.Error, tc.wantError)
			}
			if tc.wantMetadataJSON == "" {
				if len(payload.Run.Metadata) != 0 {
					t.Fatalf("payload.Run.Metadata = %s, want empty metadata", string(payload.Run.Metadata))
				}
			} else {
				assertRawJSONEqual(t, "payload.Run.Metadata", payload.Run.Metadata, tc.wantMetadataJSON)
			}
			if tc.wantResultJSON == "" {
				if len(payload.Run.Result) != 0 {
					t.Fatalf("payload.Run.Result = %s, want empty result", string(payload.Run.Result))
				}
			} else {
				assertRawJSONEqual(t, "payload.Run.Result", payload.Run.Result, tc.wantResultJSON)
			}

			tc.assertCapture(t, capture)
		})
	}
}

func assertRawJSONEqual(t *testing.T, label string, got json.RawMessage, want string) {
	t.Helper()

	var gotValue any
	if err := json.Unmarshal(got, &gotValue); err != nil {
		t.Fatalf("%s json.Unmarshal(got) error = %v; got=%s", label, err, string(got))
	}
	var wantValue any
	if err := json.Unmarshal([]byte(want), &wantValue); err != nil {
		t.Fatalf("%s json.Unmarshal(want) error = %v; want=%s", label, err, want)
	}
	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Fatalf("%s = %#v, want %#v", label, gotValue, wantValue)
	}
}
