package core_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/api/core"
	"github.com/pedronauck/agh/internal/api/testutil"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	"github.com/pedronauck/agh/internal/notifications"
	taskpkg "github.com/pedronauck/agh/internal/task"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func TestTaskPayloadBuildersPreserveIdentityOwnershipAndRunBindings(t *testing.T) {
	t.Parallel()

	taskMetadata := json.RawMessage(`{"priority":"high"}`)
	runResult := json.RawMessage(`{"ok":true}`)
	eventPayload := json.RawMessage(`{"action":"claim"}`)

	view := &taskpkg.View{
		Task: taskpkg.Task{
			ID:             "task-1",
			Identifier:     "TASK-1",
			Scope:          taskpkg.ScopeWorkspace,
			WorkspaceID:    "ws-alpha",
			ParentTaskID:   "task-root",
			NetworkChannel: "builders",
			Title:          "Review task API",
			Description:    "Check handler wiring",
			Status:         taskpkg.TaskStatusInProgress,
			Owner:          &taskpkg.Ownership{Kind: taskpkg.OwnerKindPool, Ref: "reviewers"},
			CreatedBy:      taskpkg.ActorIdentity{Kind: taskpkg.ActorKindHuman, Ref: "local-user"},
			Origin:         taskpkg.Origin{Kind: taskpkg.OriginKindHTTP, Ref: "tasks.create"},
			CreatedAt:      time.Date(2026, 4, 14, 10, 0, 0, 0, time.UTC),
			UpdatedAt:      time.Date(2026, 4, 14, 10, 1, 0, 0, time.UTC),
			Metadata:       taskMetadata,
		},
		Children: []taskpkg.Summary{{
			ID:        "task-child",
			Title:     "Follow up",
			Scope:     taskpkg.ScopeWorkspace,
			Status:    taskpkg.TaskStatusReady,
			CreatedBy: taskpkg.ActorIdentity{Kind: taskpkg.ActorKindHuman, Ref: "local-user"},
			Origin:    taskpkg.Origin{Kind: taskpkg.OriginKindHTTP, Ref: "tasks.create_child"},
			CreatedAt: time.Date(2026, 4, 14, 10, 2, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 4, 14, 10, 2, 0, 0, time.UTC),
		}},
		Dependencies: []taskpkg.Dependency{{
			TaskID:          "task-1",
			DependsOnTaskID: "task-blocker",
			Kind:            taskpkg.DependencyKindBlocks,
			CreatedAt:       time.Date(2026, 4, 14, 10, 3, 0, 0, time.UTC),
		}},
		Runs: []taskpkg.Run{{
			ID:             "run-1",
			TaskID:         "task-1",
			Status:         taskpkg.TaskRunStatusRunning,
			Attempt:        2,
			ClaimedBy:      &taskpkg.ActorIdentity{Kind: taskpkg.ActorKindHuman, Ref: "local-user"},
			SessionID:      "sess-1",
			Origin:         taskpkg.Origin{Kind: taskpkg.OriginKindHTTP, Ref: "tasks.start_run"},
			IdempotencyKey: "key-1",
			NetworkChannel: "builders",
			QueuedAt:       time.Date(2026, 4, 14, 10, 0, 0, 0, time.UTC),
			StartedAt:      time.Date(2026, 4, 14, 10, 4, 0, 0, time.UTC),
			Result:         runResult,
		}},
		Events: []taskpkg.Event{{
			ID:        "evt-1",
			TaskID:    "task-1",
			RunID:     "run-1",
			EventType: "task.run.started",
			Actor:     taskpkg.ActorIdentity{Kind: taskpkg.ActorKindHuman, Ref: "local-user"},
			Origin:    taskpkg.Origin{Kind: taskpkg.OriginKindHTTP, Ref: "tasks.start_run"},
			Payload:   eventPayload,
			Timestamp: time.Date(2026, 4, 14, 10, 4, 0, 0, time.UTC),
		}},
	}

	payload := core.TaskDetailPayloadFromView(view)
	if payload.Task.CreatedBy.Ref != "local-user" || payload.Task.Origin.Ref != "tasks.create" {
		t.Fatalf("task payload identity = %#v", payload.Task)
	}
	if payload.Task.Owner == nil || payload.Task.Owner.Ref != "reviewers" {
		t.Fatalf("task payload owner = %#v", payload.Task.Owner)
	}
	if got := payload.Runs[0].SessionID; got != "sess-1" {
		t.Fatalf("run payload session_id = %q, want %q", got, "sess-1")
	}
	if got := payload.Runs[0].ClaimedBy; got == nil || got.Ref != "local-user" {
		t.Fatalf("run payload claimed_by = %#v", got)
	}
	if string(payload.Task.Metadata) != `{"priority":"high"}` {
		t.Fatalf("task payload metadata = %s", string(payload.Task.Metadata))
	}
	if string(payload.Runs[0].Result) != `{"ok":true}` {
		t.Fatalf("run payload result = %s", string(payload.Runs[0].Result))
	}

	taskMetadata[2] = 'X'
	runResult[2] = 'Y'
	eventPayload[2] = 'Z'
	if string(payload.Task.Metadata) != `{"priority":"high"}` {
		t.Fatalf("task payload metadata mutated = %s", string(payload.Task.Metadata))
	}
	if string(payload.Runs[0].Result) != `{"ok":true}` {
		t.Fatalf("run payload result mutated = %s", string(payload.Runs[0].Result))
	}
	if string(payload.Events[0].Payload) != `{"action":"claim"}` {
		t.Fatalf("event payload mutated = %s", string(payload.Events[0].Payload))
	}
}

func TestStatusForTaskError(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		err  error
		want int
	}{
		{name: "validation", err: core.NewTaskValidationError(context.Canceled), want: http.StatusBadRequest},
		{name: "payload too large", err: taskpkg.ErrPayloadTooLarge, want: http.StatusRequestEntityTooLarge},
		{name: "permission denied", err: taskpkg.ErrPermissionDenied, want: http.StatusForbidden},
		{name: "task not found", err: taskpkg.ErrTaskNotFound, want: http.StatusNotFound},
		{name: "workspace missing", err: workspacepkg.ErrWorkspaceNotFound, want: http.StatusNotFound},
		{name: "invalid transition", err: taskpkg.ErrInvalidStatusTransition, want: http.StatusConflict},
		{name: "stale network channel", err: taskpkg.ErrStaleNetworkChannel, want: http.StatusConflict},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := core.StatusForTaskError(tc.err); got != tc.want {
				t.Fatalf("StatusForTaskError(%v) = %d, want %d", tc.err, got, tc.want)
			}
		})
	}
}

func TestBaseHandlersTaskExecutionProfileEndpoints(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC)
	storedProfile := taskpkg.ExecutionProfile{
		TaskID: "task-1",
		Coordinator: taskpkg.CoordinatorProfile{
			Mode: taskpkg.CoordinatorModeGuided,
		},
		Worker: taskpkg.WorkerProfile{
			Mode:      taskpkg.WorkerModeSelect,
			AgentName: "worker-a",
			Provider:  "openai",
			Model:     "gpt-5.4",
		},
		Sandbox: taskpkg.SandboxPolicy{
			Mode:       taskpkg.SandboxModeRef,
			SandboxRef: "macos-lab",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	var (
		gotSetProfile taskpkg.ExecutionProfile
		deletedTaskID string
	)
	tasks := testutil.StubTaskManager{
		GetExecutionProfileFn: func(_ context.Context, id string, actor taskpkg.ActorContext) (taskpkg.ExecutionProfile, error) {
			if id != "task-1" || actor.Origin.Ref != "tasks.get_profile" {
				t.Fatalf("GetExecutionProfile(id, actor) = %q, %#v", id, actor)
			}
			return storedProfile, nil
		},
		SetExecutionProfileFn: func(
			_ context.Context,
			id string,
			profile *taskpkg.ExecutionProfile,
			actor taskpkg.ActorContext,
		) (taskpkg.ExecutionProfile, error) {
			if id != "task-1" || actor.Origin.Ref != "tasks.set_profile" {
				t.Fatalf("SetExecutionProfile(id, actor) = %q, %#v", id, actor)
			}
			if profile == nil {
				t.Fatal("SetExecutionProfile profile = nil")
			}
			gotSetProfile = *profile
			stored := *profile
			stored.UpdatedAt = now
			return stored, nil
		},
		DeleteExecutionProfileFn: func(_ context.Context, id string, actor taskpkg.ActorContext) error {
			if actor.Origin.Ref != "tasks.delete_profile" {
				t.Fatalf("DeleteExecutionProfile actor = %#v", actor)
			}
			deletedTaskID = id
			return nil
		},
	}
	fixture := newHandlerFixtureWithTasks(
		t,
		testutil.StubSessionManager{},
		testutil.StubObserver{},
		tasks,
		testutil.StubWorkspaceService{},
		nil,
		nil,
	)

	resp := performRequest(t, fixture.Engine, http.MethodGet, "/tasks/task-1/execution-profile", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("get profile status = %d, want %d; body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}
	var getPayload contract.TaskExecutionProfileResponse
	testutil.DecodeJSONResponse(t, resp, &getPayload)
	if getPayload.Profile.Worker.AgentName != "worker-a" ||
		getPayload.Profile.Sandbox.SandboxRef != "macos-lab" {
		t.Fatalf("get profile payload = %#v", getPayload.Profile)
	}

	resp = performRequest(
		t,
		fixture.Engine,
		http.MethodPut,
		"/tasks/task-1/execution-profile",
		[]byte(`{"worker":{"mode":"select","agent_name":"worker-b"},"sandbox":{"mode":"none"}}`),
	)
	if resp.Code != http.StatusOK {
		t.Fatalf("set profile status = %d, want %d; body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}
	if gotSetProfile.TaskID != "task-1" ||
		gotSetProfile.Worker.Mode != taskpkg.WorkerModeSelect ||
		gotSetProfile.Worker.AgentName != "worker-b" ||
		gotSetProfile.Sandbox.Mode != taskpkg.SandboxModeNone {
		t.Fatalf("set profile request = %#v", gotSetProfile)
	}

	resp = performRequest(t, fixture.Engine, http.MethodDelete, "/tasks/task-1/execution-profile", nil)
	if resp.Code != http.StatusNoContent {
		t.Fatalf("delete profile status = %d, want %d; body=%s", resp.Code, http.StatusNoContent, resp.Body.String())
	}
	if deletedTaskID != "task-1" {
		t.Fatalf("deleted task id = %q, want task-1", deletedTaskID)
	}

	resp = performRequest(
		t,
		fixture.Engine,
		http.MethodPut,
		"/tasks/task-1/execution-profile",
		[]byte(`{"task_id":"other"}`),
	)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf(
			"set mismatched profile status = %d, want %d; body=%s",
			resp.Code,
			http.StatusBadRequest,
			resp.Body.String(),
		)
	}
}

func TestBaseHandlersTaskBridgeNotificationSubscriptionEndpoints(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 5, 11, 0, 0, 0, time.UTC)
	subscription := bridgepkg.BridgeTaskSubscription{
		SubscriptionID:   "sub-1",
		TaskID:           "task-1",
		BridgeInstanceID: "brg-1",
		Scope:            bridgepkg.ScopeWorkspace,
		WorkspaceID:      "ws-1",
		PeerID:           "peer-1",
		ThreadID:         "thread-1",
		DeliveryMode:     bridgepkg.DeliveryModeReply,
		CreatedBy:        taskpkg.ActorIdentity{Kind: taskpkg.ActorKindHuman, Ref: "local-user"},
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	var (
		putSubscription bridgepkg.BridgeTaskSubscription
		listQuery       bridgepkg.BridgeTaskSubscriptionQuery
		deleteID        string
		deleted         bool
	)
	tasks := testutil.StubTaskManager{
		GetTaskFn: func(_ context.Context, id string, actor taskpkg.ActorContext) (*taskpkg.View, error) {
			if id != "task-1" {
				t.Fatalf("GetTask id = %q, want task-1", id)
			}
			switch actor.Origin.Ref {
			case "tasks.create_bridge_notification_subscription",
				"tasks.list_bridge_notification_subscriptions",
				"tasks.get_bridge_notification_subscription",
				"tasks.delete_bridge_notification_subscription":
			default:
				t.Fatalf("GetTask actor origin = %#v", actor.Origin)
			}
			return &taskpkg.View{Task: taskpkg.Task{ID: "task-1"}}, nil
		},
	}
	bridges := testutil.StubBridgeService{
		GetInstanceFn: func(_ context.Context, id string) (*bridgepkg.BridgeInstance, error) {
			if id != "brg-1" {
				t.Fatalf("GetInstance id = %q, want brg-1", id)
			}
			return &bridgepkg.BridgeInstance{ID: "brg-1"}, nil
		},
		PutTaskSubscriptionFn: func(_ context.Context, item bridgepkg.BridgeTaskSubscription) error {
			putSubscription = item
			return nil
		},
		ListTaskSubscriptionsFn: func(
			_ context.Context,
			query bridgepkg.BridgeTaskSubscriptionQuery,
		) ([]bridgepkg.BridgeTaskSubscription, error) {
			listQuery = query
			return []bridgepkg.BridgeTaskSubscription{subscription}, nil
		},
		GetTaskSubscriptionFn: func(_ context.Context, id string) (bridgepkg.BridgeTaskSubscription, error) {
			if id != "sub-1" {
				t.Fatalf("GetBridgeTaskSubscription id = %q, want sub-1", id)
			}
			if deleted {
				return bridgepkg.BridgeTaskSubscription{}, bridgepkg.ErrBridgeTaskSubscriptionNotFound
			}
			return subscription, nil
		},
		DeleteTaskSubscriptionFn: func(_ context.Context, id string) error {
			deleteID = id
			deleted = true
			return nil
		},
		GetCursorFn: func(_ context.Context, key notifications.CursorKey) (notifications.Cursor, error) {
			if key.ConsumerID != "bridge_task_subscription:sub-1" ||
				key.StreamName != "task_events" ||
				key.SubjectID != "task-1" {
				t.Fatalf("GetCursor key = %#v, want subscription cursor", key)
			}
			return notifications.Cursor{
				Key:             key,
				LastSequence:    7,
				LastDeliveryID:  "notif:sub-1:7",
				LastDeliveredAt: now.Add(time.Minute),
				LastError:       "bridge adapter rejected send",
				UpdatedAt:       now.Add(2 * time.Minute),
			}, nil
		},
	}
	fixture := newHandlerFixtureWithTasksAndBridges(
		t,
		testutil.StubSessionManager{},
		testutil.StubObserver{},
		tasks,
		bridges,
		testutil.StubWorkspaceService{},
		nil,
		nil,
	)

	resp := performRequest(
		t,
		fixture.Engine,
		http.MethodPost,
		"/tasks/task-1/notifications/bridges",
		[]byte(
			`{"subscription_id":"sub-1","bridge_instance_id":"brg-1","scope":"workspace","workspace_id":"ws-1","peer_id":"peer-1","thread_id":"thread-1","delivery_mode":"reply"}`,
		),
	)
	if resp.Code != http.StatusCreated {
		t.Fatalf("create subscription status = %d, want %d; body=%s", resp.Code, http.StatusCreated, resp.Body.String())
	}
	if putSubscription.SubscriptionID != "sub-1" ||
		putSubscription.TaskID != "task-1" ||
		putSubscription.BridgeInstanceID != "brg-1" ||
		putSubscription.CreatedBy.Kind != taskpkg.ActorKindHuman ||
		putSubscription.CreatedAt.IsZero() {
		t.Fatalf("put subscription = %#v", putSubscription)
	}
	var createPayload contract.TaskBridgeNotificationSubscriptionResponse
	testutil.DecodeJSONResponse(t, resp, &createPayload)
	if createPayload.Subscription.Cursor.ConsumerID != "bridge_task_subscription:sub-1" ||
		createPayload.Subscription.Cursor.StreamName != "task_events" ||
		createPayload.Subscription.Cursor.SubjectID != "task-1" ||
		createPayload.Subscription.Cursor.LastSequence != 7 ||
		createPayload.Subscription.Cursor.LastDeliveryID != "notif:sub-1:7" ||
		createPayload.Subscription.Cursor.LastError != "bridge adapter rejected send" {
		t.Fatalf("create payload cursor = %#v", createPayload.Subscription.Cursor)
	}

	resp = performRequest(
		t,
		fixture.Engine,
		http.MethodGet,
		"/tasks/task-1/notifications/bridges?bridge_instance_id=brg-1&scope=workspace&workspace_id=ws-1&limit=2",
		nil,
	)
	if resp.Code != http.StatusOK {
		t.Fatalf("list subscription status = %d, want %d; body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}
	if listQuery.TaskID != "task-1" ||
		listQuery.BridgeInstanceID != "brg-1" ||
		listQuery.Scope != bridgepkg.ScopeWorkspace ||
		listQuery.WorkspaceID != "ws-1" ||
		listQuery.Limit != 2 {
		t.Fatalf("list query = %#v", listQuery)
	}
	var listPayload contract.TaskBridgeNotificationSubscriptionsResponse
	testutil.DecodeJSONResponse(t, resp, &listPayload)
	if len(listPayload.Subscriptions) != 1 || listPayload.Subscriptions[0].SubscriptionID != "sub-1" {
		t.Fatalf("list payload = %#v", listPayload)
	}
	if listPayload.Subscriptions[0].Cursor.LastSequence != 7 ||
		listPayload.Subscriptions[0].Cursor.LastError != "bridge adapter rejected send" {
		t.Fatalf("list payload cursor = %#v", listPayload.Subscriptions[0].Cursor)
	}

	resp = performRequest(
		t,
		fixture.Engine,
		http.MethodGet,
		"/tasks/task-1/notifications/bridges/sub-1",
		nil,
	)
	if resp.Code != http.StatusOK {
		t.Fatalf("get subscription status = %d, want %d; body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}
	var getPayload contract.TaskBridgeNotificationSubscriptionResponse
	testutil.DecodeJSONResponse(t, resp, &getPayload)
	if getPayload.Subscription.SubscriptionID != "sub-1" ||
		getPayload.Subscription.Cursor.ConsumerID != "bridge_task_subscription:sub-1" ||
		getPayload.Subscription.Cursor.LastSequence != 7 ||
		getPayload.Subscription.Cursor.UpdatedAt == nil {
		t.Fatalf("get payload = %#v", getPayload)
	}

	resp = performRequest(
		t,
		fixture.Engine,
		http.MethodDelete,
		"/tasks/task-1/notifications/bridges/sub-1",
		nil,
	)
	if resp.Code != http.StatusNoContent {
		t.Fatalf(
			"delete subscription status = %d, want %d; body=%s",
			resp.Code,
			http.StatusNoContent,
			resp.Body.String(),
		)
	}
	if deleteID != "sub-1" {
		t.Fatalf("delete id = %q, want sub-1", deleteID)
	}

	resp = performRequest(
		t,
		fixture.Engine,
		http.MethodGet,
		"/tasks/task-1/notifications/bridges/sub-1",
		nil,
	)
	if resp.Code != http.StatusNotFound {
		t.Fatalf(
			"get deleted subscription status = %d, want %d; body=%s",
			resp.Code,
			http.StatusNotFound,
			resp.Body.String(),
		)
	}

	resp = performRequest(
		t,
		fixture.Engine,
		http.MethodDelete,
		"/tasks/task-1/notifications/bridges/sub-1",
		nil,
	)
	if resp.Code != http.StatusNotFound {
		t.Fatalf(
			"delete deleted subscription status = %d, want %d; body=%s",
			resp.Code,
			http.StatusNotFound,
			resp.Body.String(),
		)
	}

	resp = performRequest(
		t,
		fixture.Engine,
		http.MethodPost,
		"/tasks/task-1/notifications/bridges",
		[]byte(`{"bridge_instance_id":"brg-1","scope":"global","workspace_id":"ws-1","delivery_mode":"reply"}`),
	)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf(
			"invalid subscription status = %d, want %d; body=%s",
			resp.Code,
			http.StatusBadRequest,
			resp.Body.String(),
		)
	}
}

func TestBaseHandlersTaskBridgeNotificationSubscriptionValidation(t *testing.T) {
	t.Parallel()

	t.Run("Should reject subscriptions for missing bridge instances before persistence", func(t *testing.T) {
		t.Parallel()

		tasks := testutil.StubTaskManager{
			GetTaskFn: func(_ context.Context, id string, actor taskpkg.ActorContext) (*taskpkg.View, error) {
				if id != "task-1" {
					t.Fatalf("GetTask id = %q, want task-1", id)
				}
				if actor.Origin.Ref != "tasks.create_bridge_notification_subscription" {
					t.Fatalf("GetTask actor origin = %#v", actor.Origin)
				}
				return &taskpkg.View{Task: taskpkg.Task{ID: "task-1"}}, nil
			},
		}
		bridges := testutil.StubBridgeService{
			GetInstanceFn: func(_ context.Context, id string) (*bridgepkg.BridgeInstance, error) {
				if id != "missing-bridge" {
					t.Fatalf("GetInstance id = %q, want missing-bridge", id)
				}
				return nil, bridgepkg.ErrBridgeInstanceNotFound
			},
			PutTaskSubscriptionFn: func(context.Context, bridgepkg.BridgeTaskSubscription) error {
				t.Fatal("PutBridgeTaskSubscription should not be called for a missing bridge instance")
				return nil
			},
		}
		fixture := newHandlerFixtureWithTasksAndBridges(
			t,
			testutil.StubSessionManager{},
			testutil.StubObserver{},
			tasks,
			bridges,
			testutil.StubWorkspaceService{},
			nil,
			nil,
		)

		resp := performRequest(
			t,
			fixture.Engine,
			http.MethodPost,
			"/tasks/task-1/notifications/bridges",
			[]byte(
				`{"subscription_id":"sub-missing","bridge_instance_id":"missing-bridge","scope":"global","peer_id":"peer-1","delivery_mode":"reply"}`,
			),
		)
		if resp.Code != http.StatusNotFound {
			t.Fatalf(
				"create subscription missing bridge status = %d, want %d; body=%s",
				resp.Code,
				http.StatusNotFound,
				resp.Body.String(),
			)
		}
		var errorPayload contract.ErrorPayload
		testutil.DecodeJSONResponse(t, resp, &errorPayload)
		if errorPayload.Error != bridgepkg.ErrBridgeInstanceNotFound.Error() {
			t.Fatalf("missing bridge error = %q, want %q", errorPayload.Error, bridgepkg.ErrBridgeInstanceNotFound)
		}
	})
}

func TestBaseHandlersTaskRunReviewEndpoints(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 5, 10, 30, 0, 0, time.UTC)
	review := taskpkg.RunReview{
		ReviewID:          "review-1",
		TaskID:            "task-1",
		RunID:             "run-1",
		Policy:            taskpkg.ReviewPolicyAlways,
		ReviewRound:       1,
		Attempt:           1,
		Status:            taskpkg.RunReviewStatusRequested,
		Reason:            "ready for review",
		MissingWork:       json.RawMessage(`[]`),
		ReviewerSessionID: "sess-review",
		RequestedAt:       now,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	var (
		requestCalls int
		listQueries  []taskpkg.RunReviewQuery
		gotReviewID  string
		gotVerdict   taskpkg.RecordRunReviewRequest
	)
	tasks := testutil.StubTaskManager{
		RunDetailFn: func(_ context.Context, runID string, actor taskpkg.ActorContext) (*taskpkg.RunDetailView, error) {
			if runID != "run-1" || actor.Origin.Ref != "tasks.request_review" {
				t.Fatalf("RunDetail(runID, actor) = %q, %#v", runID, actor)
			}
			return &taskpkg.RunDetailView{
				Run: taskpkg.Run{
					ID:      "run-1",
					TaskID:  "task-1",
					Status:  taskpkg.TaskRunStatusCompleted,
					Attempt: 1,
					Origin:  taskpkg.Origin{Kind: taskpkg.OriginKindHTTP, Ref: "tasks.run.start"},
				},
			}, nil
		},
		RequestRunReviewFn: func(
			_ context.Context,
			req taskpkg.RunReviewRequest,
			actor taskpkg.ActorContext,
		) (taskpkg.RunReview, bool, error) {
			requestCalls++
			if actor.Origin.Ref != "tasks.request_review" ||
				req.TaskID != "task-1" ||
				req.RunID != "run-1" ||
				req.Policy != taskpkg.ReviewPolicyAlways ||
				req.Reason != "ready for review" {
				t.Fatalf("RequestRunReview(req, actor) = %#v, %#v", req, actor)
			}
			return review, true, nil
		},
		ListRunReviewsFn: func(
			_ context.Context,
			query taskpkg.RunReviewQuery,
			actor taskpkg.ActorContext,
		) ([]taskpkg.RunReview, error) {
			if actor.Origin.Ref != "tasks.list_reviews" {
				t.Fatalf("ListRunReviews actor = %#v", actor)
			}
			listQueries = append(listQueries, query)
			return []taskpkg.RunReview{review}, nil
		},
		GetRunReviewFn: func(_ context.Context, reviewID string, actor taskpkg.ActorContext) (taskpkg.RunReview, error) {
			if actor.Origin.Ref != "tasks.get_review" {
				t.Fatalf("GetRunReview actor = %#v", actor)
			}
			gotReviewID = reviewID
			return review, nil
		},
		RecordRunReviewFn: func(
			_ context.Context,
			req taskpkg.RecordRunReviewRequest,
			actor taskpkg.ActorContext,
		) (taskpkg.RunReviewResult, error) {
			if actor.Origin.Ref != "tasks.submit_review" {
				t.Fatalf("RecordRunReview actor = %#v", actor)
			}
			gotVerdict = req
			recorded := review
			recorded.Status = taskpkg.RunReviewStatusRecorded
			recorded.Outcome = taskpkg.RunReviewOutcomeRejected
			continuation := &taskpkg.Run{
				ID:      "run-2",
				TaskID:  "task-1",
				Status:  taskpkg.TaskRunStatusQueued,
				Attempt: 2,
				Origin:  taskpkg.Origin{Kind: taskpkg.OriginKindHTTP, Ref: "tasks.run.review_retry"},
			}
			return taskpkg.RunReviewResult{Review: recorded, ContinuationRun: continuation}, nil
		},
	}
	fixture := newHandlerFixtureWithTasks(
		t,
		testutil.StubSessionManager{},
		testutil.StubObserver{},
		tasks,
		testutil.StubWorkspaceService{},
		nil,
		nil,
	)

	resp := performRequest(
		t,
		fixture.Engine,
		http.MethodPost,
		"/task-runs/run-1/reviews",
		[]byte(`{"reason":"ready for review"}`),
	)
	if resp.Code != http.StatusCreated {
		t.Fatalf("request review status = %d, want %d; body=%s", resp.Code, http.StatusCreated, resp.Body.String())
	}
	var requestPayload contract.TaskRunReviewRequestResponse
	testutil.DecodeJSONResponse(t, resp, &requestPayload)
	if !requestPayload.Created || requestPayload.Review.ReviewID != "review-1" {
		t.Fatalf("request review payload = %#v", requestPayload)
	}
	if requestCalls != 1 {
		t.Fatalf("request calls = %d, want 1", requestCalls)
	}

	resp = performRequest(
		t,
		fixture.Engine,
		http.MethodGet,
		"/task-runs/run-1/reviews?status=requested&reviewer_session_id=sess-review&limit=2",
		nil,
	)
	if resp.Code != http.StatusOK {
		t.Fatalf("list run reviews status = %d, want %d; body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}
	resp = performRequest(t, fixture.Engine, http.MethodGet, "/tasks/task-1/reviews?status=requested", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("list task reviews status = %d, want %d; body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}
	if len(listQueries) != 2 ||
		listQueries[0].RunID != "run-1" ||
		listQueries[0].ReviewerSessionID != "sess-review" ||
		listQueries[0].Limit != 2 ||
		listQueries[1].TaskID != "task-1" {
		t.Fatalf("review list queries = %#v", listQueries)
	}

	resp = performRequest(t, fixture.Engine, http.MethodGet, "/task-reviews/review-1", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("get review status = %d, want %d; body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}
	if gotReviewID != "review-1" {
		t.Fatalf("review id = %q, want review-1", gotReviewID)
	}

	resp = performRequest(
		t,
		fixture.Engine,
		http.MethodPost,
		"/task-reviews/review-1/verdict",
		[]byte(
			`{"run_id":"run-1","verdict":{"outcome":"rejected","confidence":0.8,"reason":"missing tests","delivery_id":"delivery-1","missing_work":["tests"],"next_round_guidance":"add tests"}}`,
		),
	)
	if resp.Code != http.StatusOK {
		t.Fatalf("submit review status = %d, want %d; body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}
	if gotVerdict.ReviewID != "review-1" ||
		gotVerdict.RunID != "run-1" ||
		gotVerdict.Verdict.Outcome != taskpkg.RunReviewOutcomeRejected ||
		gotVerdict.Verdict.Confidence == nil ||
		*gotVerdict.Verdict.Confidence != 0.8 ||
		string(gotVerdict.Verdict.MissingWork) != `["tests"]` {
		t.Fatalf("record review request = %#v", gotVerdict)
	}
	var verdictPayload contract.TaskRunReviewVerdictResponse
	testutil.DecodeJSONResponse(t, resp, &verdictPayload)
	if verdictPayload.Review.Status != taskpkg.RunReviewStatusRecorded ||
		verdictPayload.ContinuationRun == nil ||
		verdictPayload.ContinuationRun.ID != "run-2" {
		t.Fatalf("verdict payload = %#v", verdictPayload)
	}

	resp = performRequest(
		t,
		fixture.Engine,
		http.MethodPost,
		"/task-runs/run-1/reviews",
		[]byte(`{"run_id":"other"}`),
	)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf(
			"mismatched review status = %d, want %d; body=%s",
			resp.Code,
			http.StatusBadRequest,
			resp.Body.String(),
		)
	}
	if requestCalls != 1 {
		t.Fatalf("request calls after mismatch = %d, want 1", requestCalls)
	}
}

func TestBaseHandlersTaskValidationAndErrorMapping(t *testing.T) {
	t.Parallel()

	t.Run("ShouldRejectInvalidScopeWorkspaceAndChannelInputs", func(t *testing.T) {
		t.Parallel()

		tasks := testutil.StubTaskManager{
			CreateTaskFn: func(context.Context, taskpkg.CreateTask, taskpkg.ActorContext) (*taskpkg.Task, error) {
				t.Fatal("CreateTask should not be called for invalid task input")
				return nil, nil
			},
		}
		fixture := newHandlerFixtureWithTasks(
			t,
			testutil.StubSessionManager{},
			testutil.StubObserver{},
			tasks,
			testutil.StubWorkspaceService{},
			nil,
			nil,
		)

		resp := performRequest(
			t,
			fixture.Engine,
			http.MethodPost,
			"/tasks",
			[]byte(`{"scope":"workspace","title":"Broken"}`),
		)
		if resp.Code != http.StatusBadRequest {
			t.Fatalf(
				"workspace create status = %d, want %d; body=%s",
				resp.Code,
				http.StatusBadRequest,
				resp.Body.String(),
			)
		}

		resp = performRequest(
			t,
			fixture.Engine,
			http.MethodPost,
			"/tasks",
			[]byte(`{"scope":"global","title":"Broken","network_channel":"bad.channel"}`),
		)
		if resp.Code != http.StatusBadRequest {
			t.Fatalf(
				"channel create status = %d, want %d; body=%s",
				resp.Code,
				http.StatusBadRequest,
				resp.Body.String(),
			)
		}
	})

	t.Run("ShouldRejectUnknownWorkspaceAndInvalidOwnerInput", func(t *testing.T) {
		t.Parallel()

		tasks := testutil.StubTaskManager{
			CreateTaskFn: func(context.Context, taskpkg.CreateTask, taskpkg.ActorContext) (*taskpkg.Task, error) {
				t.Fatal("CreateTask should not be called when workspace lookup fails")
				return nil, nil
			},
			UpdateTaskFn: func(context.Context, string, taskpkg.Patch, taskpkg.ActorContext) (*taskpkg.Task, error) {
				t.Fatal("UpdateTask should not be called for invalid owner input")
				return nil, nil
			},
		}
		workspaces := testutil.StubWorkspaceService{
			GetFn: func(context.Context, string) (workspacepkg.Workspace, error) {
				return workspacepkg.Workspace{}, workspacepkg.ErrWorkspaceNotFound
			},
		}
		fixture := newHandlerFixtureWithTasks(
			t,
			testutil.StubSessionManager{},
			testutil.StubObserver{},
			tasks,
			workspaces,
			nil,
			nil,
		)

		resp := performRequest(
			t,
			fixture.Engine,
			http.MethodPost,
			"/tasks",
			[]byte(`{"scope":"workspace","workspace":"missing","title":"Broken"}`),
		)
		if resp.Code != http.StatusNotFound {
			t.Fatalf(
				"workspace lookup status = %d, want %d; body=%s",
				resp.Code,
				http.StatusNotFound,
				resp.Body.String(),
			)
		}

		resp = performRequest(
			t,
			fixture.Engine,
			http.MethodPatch,
			"/tasks/task-1",
			[]byte(`{"owner":{"kind":"bogus","ref":"ops"}}`),
		)
		if resp.Code != http.StatusBadRequest {
			t.Fatalf(
				"invalid owner status = %d, want %d; body=%s",
				resp.Code,
				http.StatusBadRequest,
				resp.Body.String(),
			)
		}
	})

	t.Run("ShouldRejectGlobalWorkspaceBindingsWithoutWorkspaceLookup", func(t *testing.T) {
		t.Parallel()

		workspaceLookups := 0
		tasks := testutil.StubTaskManager{
			ListTasksFn: func(context.Context, taskpkg.Query, taskpkg.ActorContext) ([]taskpkg.Summary, error) {
				t.Fatal("ListTasks should not be called when global scope includes workspace filter")
				return nil, nil
			},
			CreateTaskFn: func(context.Context, taskpkg.CreateTask, taskpkg.ActorContext) (*taskpkg.Task, error) {
				t.Fatal("CreateTask should not be called when global scope includes workspace binding")
				return nil, nil
			},
			CreateChildTaskFn: func(context.Context, string, taskpkg.CreateTask, taskpkg.ActorContext) (*taskpkg.Task, error) {
				t.Fatal("CreateChildTask should not be called when global scope includes workspace binding")
				return nil, nil
			},
		}
		workspaces := testutil.StubWorkspaceService{
			GetFn: func(context.Context, string) (workspacepkg.Workspace, error) {
				workspaceLookups++
				return workspacepkg.Workspace{}, workspacepkg.ErrWorkspaceNotFound
			},
		}
		fixture := newHandlerFixtureWithTasks(
			t,
			testutil.StubSessionManager{},
			testutil.StubObserver{},
			tasks,
			workspaces,
			nil,
			nil,
		)

		resp := performRequest(t, fixture.Engine, http.MethodGet, "/tasks?scope=global&workspace=missing", nil)
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("global list status = %d, want %d; body=%s", resp.Code, http.StatusBadRequest, resp.Body.String())
		}

		resp = performRequest(
			t,
			fixture.Engine,
			http.MethodPost,
			"/tasks",
			[]byte(`{"scope":"global","workspace":"missing","title":"Broken"}`),
		)
		if resp.Code != http.StatusBadRequest {
			t.Fatalf(
				"global create status = %d, want %d; body=%s",
				resp.Code,
				http.StatusBadRequest,
				resp.Body.String(),
			)
		}

		resp = performRequest(
			t,
			fixture.Engine,
			http.MethodPost,
			"/tasks/task-root/children",
			[]byte(`{"scope":"global","workspace":"missing","title":"Broken child"}`),
		)
		if resp.Code != http.StatusBadRequest {
			t.Fatalf(
				"global child create status = %d, want %d; body=%s",
				resp.Code,
				http.StatusBadRequest,
				resp.Body.String(),
			)
		}

		if workspaceLookups != 0 {
			t.Fatalf("workspace lookup count = %d, want 0", workspaceLookups)
		}
	})

	t.Run("ShouldMapTaskDomainErrorsToStableStatuses", func(t *testing.T) {
		t.Parallel()

		tasks := testutil.StubTaskManager{
			GetTaskFn: func(context.Context, string, taskpkg.ActorContext) (*taskpkg.View, error) {
				return nil, taskpkg.ErrTaskNotFound
			},
			UpdateTaskFn: func(context.Context, string, taskpkg.Patch, taskpkg.ActorContext) (*taskpkg.Task, error) {
				return nil, taskpkg.ErrPermissionDenied
			},
			StartRunFn: func(context.Context, string, taskpkg.StartRun, taskpkg.ActorContext) (*taskpkg.Run, error) {
				return nil, taskpkg.ErrInvalidStatusTransition
			},
		}
		fixture := newHandlerFixtureWithTasks(
			t,
			testutil.StubSessionManager{},
			testutil.StubObserver{},
			tasks,
			testutil.StubWorkspaceService{},
			nil,
			nil,
		)

		resp := performRequest(t, fixture.Engine, http.MethodGet, "/tasks/missing", nil)
		if resp.Code != http.StatusNotFound {
			t.Fatalf("get missing status = %d, want %d; body=%s", resp.Code, http.StatusNotFound, resp.Body.String())
		}

		resp = performRequest(t, fixture.Engine, http.MethodPatch, "/tasks/task-1", []byte(`{"title":"rename"}`))
		if resp.Code != http.StatusForbidden {
			t.Fatalf(
				"update forbidden status = %d, want %d; body=%s",
				resp.Code,
				http.StatusForbidden,
				resp.Body.String(),
			)
		}

		resp = performRequest(t, fixture.Engine, http.MethodPost, "/task-runs/run-1/start", []byte(`{}`))
		if resp.Code != http.StatusConflict {
			t.Fatalf("start conflict status = %d, want %d; body=%s", resp.Code, http.StatusConflict, resp.Body.String())
		}
	})
}

func TestBaseHandlersTaskHappyPathEndpoints(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)

	var listedQuery taskpkg.Query
	var createdSpec taskpkg.CreateTask
	var childSpec taskpkg.CreateTask
	var deletedTaskID string
	var updatedPatch taskpkg.Patch
	var cancelledTask taskpkg.CancelTask
	var addedDependency taskpkg.AddDependency
	var removedTaskID string
	var removedDependsOnID string
	var enqueuedRun taskpkg.EnqueueRun
	var listedRunTaskID string
	var listedRunQuery taskpkg.RunQuery
	var claimedRun taskpkg.ClaimRun
	var startedRun taskpkg.StartRun
	var attachedRunID string
	var attachedSessionID string
	var completedRun taskpkg.RunResult
	var failedRun taskpkg.RunFailure
	var cancelledRun taskpkg.CancelRun

	taskView := &taskpkg.View{
		Task: taskpkg.Task{
			ID:             "task-1",
			Scope:          taskpkg.ScopeWorkspace,
			WorkspaceID:    "ws-alpha",
			NetworkChannel: "builders",
			Title:          "Review task API",
			Description:    "Check handler wiring",
			Status:         taskpkg.TaskStatusInProgress,
			Owner:          &taskpkg.Ownership{Kind: taskpkg.OwnerKindPool, Ref: "reviewers"},
			CreatedBy:      taskpkg.ActorIdentity{Kind: taskpkg.ActorKindHuman, Ref: "local-user"},
			Origin:         taskpkg.Origin{Kind: taskpkg.OriginKindHTTP, Ref: "tasks.create"},
			CreatedAt:      now,
			UpdatedAt:      now,
			Metadata:       json.RawMessage(`{"priority":"high"}`),
		},
		Dependencies: []taskpkg.Dependency{{
			TaskID:          "task-1",
			DependsOnTaskID: "task-blocker",
			Kind:            taskpkg.DependencyKindBlocks,
			CreatedAt:       now,
		}},
		Runs: []taskpkg.Run{
			{
				ID:        "run-1",
				TaskID:    "task-1",
				Status:    taskpkg.TaskRunStatusRunning,
				Attempt:   1,
				SessionID: "sess-1",
				Origin:    taskpkg.Origin{Kind: taskpkg.OriginKindHTTP, Ref: "tasks.start_run"},
				QueuedAt:  now,
				StartedAt: now,
			},
			{
				ID:       "run-2",
				TaskID:   "task-1",
				Status:   taskpkg.TaskRunStatusQueued,
				Attempt:  2,
				Origin:   taskpkg.Origin{Kind: taskpkg.OriginKindHTTP, Ref: "tasks.enqueue_run"},
				QueuedAt: now,
			},
		},
		Events: []taskpkg.Event{{
			ID:        "evt-1",
			TaskID:    "task-1",
			EventType: "task.created",
			Actor:     taskpkg.ActorIdentity{Kind: taskpkg.ActorKindHuman, Ref: "local-user"},
			Origin:    taskpkg.Origin{Kind: taskpkg.OriginKindHTTP, Ref: "tasks.create"},
			Timestamp: now,
		}},
	}

	getTaskCalls := 0
	tasks := testutil.StubTaskManager{
		ListTasksFn: func(_ context.Context, query taskpkg.Query, _ taskpkg.ActorContext) ([]taskpkg.Summary, error) {
			listedQuery = query
			return []taskpkg.Summary{{
				ID:             "task-1",
				Scope:          query.Scope,
				WorkspaceID:    query.WorkspaceID,
				ParentTaskID:   query.ParentTaskID,
				NetworkChannel: query.NetworkChannel,
				Title:          "Review task API",
				Status:         query.Status,
				Owner:          &taskpkg.Ownership{Kind: query.OwnerKind, Ref: query.OwnerRef},
				CreatedBy:      taskpkg.ActorIdentity{Kind: taskpkg.ActorKindHuman, Ref: "local-user"},
				Origin:         taskpkg.Origin{Kind: taskpkg.OriginKindHTTP, Ref: "tasks.list"},
				CreatedAt:      now,
				UpdatedAt:      now,
			}}, nil
		},
		CreateTaskFn: func(_ context.Context, spec taskpkg.CreateTask, actor taskpkg.ActorContext) (*taskpkg.Task, error) {
			createdSpec = spec
			record := taskView.Task
			record.CreatedBy = actor.Actor
			record.Origin = actor.Origin
			record.Title = spec.Title
			record.Description = spec.Description
			record.Metadata = spec.Metadata
			return &record, nil
		},
		GetTaskFn: func(_ context.Context, id string, _ taskpkg.ActorContext) (*taskpkg.View, error) {
			getTaskCalls++
			if id != "task-1" {
				t.Fatalf("GetTask id = %q, want %q", id, "task-1")
			}
			return taskView, nil
		},
		ListTaskRunsFn: func(_ context.Context, taskID string, query taskpkg.RunQuery, _ taskpkg.ActorContext) ([]taskpkg.Run, error) {
			listedRunTaskID = taskID
			listedRunQuery = query
			return []taskpkg.Run{taskView.Runs[0]}, nil
		},
		UpdateTaskFn: func(_ context.Context, _ string, patch taskpkg.Patch, _ taskpkg.ActorContext) (*taskpkg.Task, error) {
			updatedPatch = patch
			record := taskView.Task
			if patch.Title != nil {
				record.Title = *patch.Title
			}
			return &record, nil
		},
		CancelTaskFn: func(_ context.Context, _ string, req taskpkg.CancelTask, _ taskpkg.ActorContext) (*taskpkg.Task, error) {
			cancelledTask = req
			record := taskView.Task
			record.Status = taskpkg.TaskStatusCanceled
			return &record, nil
		},
		CreateChildTaskFn: func(_ context.Context, parentTaskID string, spec taskpkg.CreateTask, actor taskpkg.ActorContext) (*taskpkg.Task, error) {
			if parentTaskID != "task-root" {
				t.Fatalf("CreateChildTask parentTaskID = %q, want %q", parentTaskID, "task-root")
			}
			childSpec = spec
			return &taskpkg.Task{
				ID:           "task-child",
				Scope:        spec.Scope,
				WorkspaceID:  spec.WorkspaceID,
				Title:        spec.Title,
				Description:  spec.Description,
				Status:       taskpkg.TaskStatusReady,
				CreatedBy:    actor.Actor,
				Origin:       actor.Origin,
				CreatedAt:    now,
				UpdatedAt:    now,
				ParentTaskID: parentTaskID,
			}, nil
		},
		DeleteTaskFn: func(_ context.Context, id string, _ taskpkg.ActorContext) error {
			deletedTaskID = id
			return nil
		},
		AddDependencyFn: func(_ context.Context, spec taskpkg.AddDependency, _ taskpkg.ActorContext) error {
			addedDependency = spec
			return nil
		},
		RemoveDependencyFn: func(_ context.Context, taskID string, dependsOnID string, _ taskpkg.ActorContext) error {
			removedTaskID = taskID
			removedDependsOnID = dependsOnID
			return nil
		},
		EnqueueRunFn: func(_ context.Context, spec taskpkg.EnqueueRun, actor taskpkg.ActorContext) (*taskpkg.Run, error) {
			enqueuedRun = spec
			return &taskpkg.Run{
				ID:             "run-3",
				TaskID:         spec.TaskID,
				Status:         taskpkg.TaskRunStatusQueued,
				Attempt:        3,
				Origin:         actor.Origin,
				IdempotencyKey: spec.IdempotencyKey,
				NetworkChannel: spec.NetworkChannel,
				Metadata:       spec.Metadata,
				QueuedAt:       now,
			}, nil
		},
		ClaimRunFn: func(_ context.Context, _ string, claim taskpkg.ClaimRun, actor taskpkg.ActorContext) (*taskpkg.Run, error) {
			claimedRun = claim
			return &taskpkg.Run{
				ID:        "run-1",
				TaskID:    "task-1",
				Status:    taskpkg.TaskRunStatusClaimed,
				Attempt:   1,
				ClaimedBy: &actor.Actor,
				Origin:    actor.Origin,
				QueuedAt:  now,
				ClaimedAt: now,
			}, nil
		},
		StartRunFn: func(_ context.Context, _ string, req taskpkg.StartRun, actor taskpkg.ActorContext) (*taskpkg.Run, error) {
			startedRun = req
			return &taskpkg.Run{
				ID:        "run-1",
				TaskID:    "task-1",
				Status:    taskpkg.TaskRunStatusRunning,
				Attempt:   1,
				SessionID: "sess-1",
				Origin:    actor.Origin,
				QueuedAt:  now,
				StartedAt: now,
			}, nil
		},
		AttachRunSessionFn: func(_ context.Context, runID string, sessionID string, actor taskpkg.ActorContext) (*taskpkg.Run, error) {
			attachedRunID = runID
			attachedSessionID = sessionID
			return &taskpkg.Run{
				ID:        runID,
				TaskID:    "task-1",
				Status:    taskpkg.TaskRunStatusStarting,
				Attempt:   1,
				SessionID: sessionID,
				Origin:    actor.Origin,
				QueuedAt:  now,
			}, nil
		},
		CompleteRunFn: func(_ context.Context, _ string, result taskpkg.RunResult, actor taskpkg.ActorContext) (*taskpkg.Run, error) {
			completedRun = result
			return &taskpkg.Run{
				ID:                    "run-1",
				TaskID:                "task-1",
				Status:                taskpkg.TaskRunStatusCompleted,
				Attempt:               1,
				Origin:                actor.Origin,
				QueuedAt:              now,
				EndedAt:               now,
				Result:                result.Value,
				NetworkChannel:        "builders",
				CoordinationChannelID: "builders",
			}, nil
		},
		FailRunFn: func(_ context.Context, _ string, failure taskpkg.RunFailure, actor taskpkg.ActorContext) (*taskpkg.Run, error) {
			failedRun = failure
			return &taskpkg.Run{
				ID:                    "run-2",
				TaskID:                "task-1",
				Status:                taskpkg.TaskRunStatusFailed,
				Attempt:               2,
				Origin:                actor.Origin,
				QueuedAt:              now,
				EndedAt:               now,
				Error:                 failure.Error,
				NetworkChannel:        "builders",
				CoordinationChannelID: "builders",
			}, nil
		},
		CancelRunFn: func(_ context.Context, _ string, req taskpkg.CancelRun, actor taskpkg.ActorContext) (*taskpkg.Run, error) {
			cancelledRun = req
			return &taskpkg.Run{
				ID:                    "run-2",
				TaskID:                "task-1",
				Status:                taskpkg.TaskRunStatusCanceled,
				Attempt:               2,
				Origin:                actor.Origin,
				QueuedAt:              now,
				EndedAt:               now,
				NetworkChannel:        "builders",
				CoordinationChannelID: "builders",
			}, nil
		},
	}
	workspaces := testutil.StubWorkspaceService{
		GetFn: func(_ context.Context, ref string) (workspacepkg.Workspace, error) {
			if ref != "alpha" {
				t.Fatalf("workspace ref = %q, want %q", ref, "alpha")
			}
			return workspacepkg.Workspace{ID: "ws-alpha", Name: "alpha"}, nil
		},
	}

	fixture := newHandlerFixtureWithTasks(
		t,
		testutil.StubSessionManager{},
		testutil.StubObserver{},
		tasks,
		workspaces,
		nil,
		nil,
	)

	resp := performRequest(
		t,
		fixture.Engine,
		http.MethodGet,
		"/tasks?scope=workspace&workspace=alpha&status=ready&owner_kind=pool&owner_ref=reviewers&parent_task_id=task-root&network_channel=builders&limit=2",
		nil,
	)
	if resp.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d; body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}

	resp = performRequest(
		t,
		fixture.Engine,
		http.MethodPost,
		"/tasks",
		[]byte(
			`{"scope":"workspace","workspace":"alpha","title":"Review task API","description":"Check handler wiring","network_channel":"builders","owner":{"kind":"pool","ref":"reviewers"},"metadata":{"priority":"high"}}`,
		),
	)
	if resp.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d; body=%s", resp.Code, http.StatusCreated, resp.Body.String())
	}

	resp = performRequest(t, fixture.Engine, http.MethodGet, "/tasks/task-1", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d; body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}

	resp = performRequest(t, fixture.Engine, http.MethodDelete, "/tasks/task-1", nil)
	if resp.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want %d; body=%s", resp.Code, http.StatusNoContent, resp.Body.String())
	}

	resp = performRequest(
		t,
		fixture.Engine,
		http.MethodPatch,
		"/tasks/task-1",
		[]byte(`{"title":"Renamed task","network_channel":"builders","metadata":{"priority":"medium"}}`),
	)
	if resp.Code != http.StatusOK {
		t.Fatalf("update status = %d, want %d; body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}

	resp = performRequest(
		t,
		fixture.Engine,
		http.MethodPost,
		"/tasks/task-1/cancel",
		[]byte(`{"reason":"no longer needed","metadata":{"source":"test"}}`),
	)
	if resp.Code != http.StatusOK {
		t.Fatalf("cancel task status = %d, want %d; body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}

	resp = performRequest(
		t,
		fixture.Engine,
		http.MethodPost,
		"/tasks/task-root/children",
		[]byte(`{"scope":"workspace","workspace":"alpha","title":"Child task","description":"Follow-up work"}`),
	)
	if resp.Code != http.StatusCreated {
		t.Fatalf("create child status = %d, want %d; body=%s", resp.Code, http.StatusCreated, resp.Body.String())
	}

	resp = performRequest(
		t,
		fixture.Engine,
		http.MethodPost,
		"/tasks/task-1/dependencies",
		[]byte(`{"depends_on_task_id":"task-blocker"}`),
	)
	if resp.Code != http.StatusOK {
		t.Fatalf("add dependency status = %d, want %d; body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}

	resp = performRequest(t, fixture.Engine, http.MethodDelete, "/tasks/task-1/dependencies/task-blocker", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("remove dependency status = %d, want %d; body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}

	resp = performRequest(
		t,
		fixture.Engine,
		http.MethodGet,
		"/tasks/task-1/runs?status=running&session_id=sess-1&limit=1",
		nil,
	)
	if resp.Code != http.StatusOK {
		t.Fatalf("list runs status = %d, want %d; body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}

	resp = performRequest(
		t,
		fixture.Engine,
		http.MethodPost,
		"/tasks/task-1/runs",
		[]byte(
			`{"idempotency_key":"key-3","network_channel":"builders","metadata":{"schema":"agh.harness.detached.v1","kind":"harness_detached_run"}}`,
		),
	)
	if resp.Code != http.StatusCreated {
		t.Fatalf("enqueue status = %d, want %d; body=%s", resp.Code, http.StatusCreated, resp.Body.String())
	}

	resp = performRequest(
		t,
		fixture.Engine,
		http.MethodPost,
		"/task-runs/run-1/claim",
		[]byte(`{"idempotency_key":"claim-1"}`),
	)
	if resp.Code != http.StatusOK {
		t.Fatalf("claim status = %d, want %d; body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}

	resp = performRequest(
		t,
		fixture.Engine,
		http.MethodPost,
		"/task-runs/run-1/start",
		[]byte(`{"idempotency_key":"start-1"}`),
	)
	if resp.Code != http.StatusOK {
		t.Fatalf("start status = %d, want %d; body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}

	resp = performRequest(
		t,
		fixture.Engine,
		http.MethodPost,
		"/task-runs/run-1/attach-session",
		[]byte(`{"session_id":"sess-1"}`),
	)
	if resp.Code != http.StatusOK {
		t.Fatalf("attach status = %d, want %d; body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}

	resp = performRequest(
		t,
		fixture.Engine,
		http.MethodPost,
		"/task-runs/run-1/complete",
		[]byte(`{"result":{"ok":true}}`),
	)
	if resp.Code != http.StatusOK {
		t.Fatalf("complete status = %d, want %d; body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}
	var completedResp contract.TaskRunResponse
	testutil.DecodeJSONResponse(t, resp, &completedResp)
	if completedResp.Run.NetworkChannel != "builders" || completedResp.Run.CoordinationChannelID != "builders" {
		t.Fatalf("completed response = %#v, want preserved network/coordination channel", completedResp.Run)
	}

	resp = performRequest(
		t,
		fixture.Engine,
		http.MethodPost,
		"/task-runs/run-2/fail",
		[]byte(`{"error":"boom","metadata":{"step":"claim"}}`),
	)
	if resp.Code != http.StatusOK {
		t.Fatalf("fail status = %d, want %d; body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}
	var failedResp contract.TaskRunResponse
	testutil.DecodeJSONResponse(t, resp, &failedResp)
	if failedResp.Run.NetworkChannel != "builders" || failedResp.Run.CoordinationChannelID != "builders" {
		t.Fatalf("failed response = %#v, want preserved network/coordination channel", failedResp.Run)
	}

	resp = performRequest(
		t,
		fixture.Engine,
		http.MethodPost,
		"/task-runs/run-2/cancel",
		[]byte(`{"reason":"operator canceled","metadata":{"step":"cancel"}}`),
	)
	if resp.Code != http.StatusOK {
		t.Fatalf("cancel run status = %d, want %d; body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}
	var cancelledResp contract.TaskRunResponse
	testutil.DecodeJSONResponse(t, resp, &cancelledResp)
	if cancelledResp.Run.NetworkChannel != "builders" || cancelledResp.Run.CoordinationChannelID != "builders" {
		t.Fatalf("canceled response = %#v, want preserved network/coordination channel", cancelledResp.Run)
	}

	if listedQuery.WorkspaceID != "ws-alpha" || listedQuery.Scope != taskpkg.ScopeWorkspace ||
		listedQuery.NetworkChannel != "builders" {
		t.Fatalf("listed query = %#v", listedQuery)
	}
	if listedQuery.Status != taskpkg.TaskStatusReady || listedQuery.OwnerKind != taskpkg.OwnerKindPool ||
		listedQuery.OwnerRef != "reviewers" ||
		listedQuery.Limit != 2 {
		t.Fatalf("listed query = %#v", listedQuery)
	}
	if listedRunTaskID != "task-1" {
		t.Fatalf("listed run task id = %q, want %q", listedRunTaskID, "task-1")
	}
	if listedRunQuery.Status != taskpkg.TaskRunStatusRunning || listedRunQuery.SessionID != "sess-1" ||
		listedRunQuery.Limit != 1 {
		t.Fatalf("listed run query = %#v", listedRunQuery)
	}
	if getTaskCalls != 3 {
		t.Fatalf("GetTask() calls = %d, want 3 detail reads without extra run-list fetch", getTaskCalls)
	}
	if createdSpec.WorkspaceID != "ws-alpha" || createdSpec.NetworkChannel != "builders" || createdSpec.Owner == nil ||
		createdSpec.Owner.Ref != "reviewers" {
		t.Fatalf("created spec = %#v", createdSpec)
	}
	if childSpec.WorkspaceID != "ws-alpha" || childSpec.Title != "Child task" {
		t.Fatalf("child spec = %#v", childSpec)
	}
	if deletedTaskID != "task-1" {
		t.Fatalf("deleted task id = %q, want %q", deletedTaskID, "task-1")
	}
	if updatedPatch.Title == nil || *updatedPatch.Title != "Renamed task" {
		t.Fatalf("updated patch = %#v", updatedPatch)
	}
	if cancelledTask.Reason != "no longer needed" {
		t.Fatalf("canceled task = %#v", cancelledTask)
	}
	if addedDependency.Kind != taskpkg.DependencyKindBlocks || addedDependency.DependsOnTaskID != "task-blocker" {
		t.Fatalf("added dependency = %#v", addedDependency)
	}
	if removedTaskID != "task-1" || removedDependsOnID != "task-blocker" {
		t.Fatalf("removed dependency = task=%q dependsOn=%q", removedTaskID, removedDependsOnID)
	}
	if enqueuedRun.IdempotencyKey != "key-3" || enqueuedRun.NetworkChannel != "builders" {
		t.Fatalf("enqueued run = %#v", enqueuedRun)
	}
	if got, want := string(
		enqueuedRun.Metadata,
	), `{"schema":"agh.harness.detached.v1","kind":"harness_detached_run"}`; got != want {
		t.Fatalf("enqueued run metadata = %q, want %q", got, want)
	}
	if claimedRun.IdempotencyKey != "claim-1" {
		t.Fatalf("claimed run = %#v", claimedRun)
	}
	if startedRun.IdempotencyKey != "start-1" {
		t.Fatalf("started run = %#v", startedRun)
	}
	if attachedRunID != "run-1" || attachedSessionID != "sess-1" {
		t.Fatalf("attached run = %q session = %q", attachedRunID, attachedSessionID)
	}
	if string(completedRun.Value) != `{"ok":true}` {
		t.Fatalf("completed run = %#v", completedRun)
	}
	if failedRun.Error != "boom" {
		t.Fatalf("failed run = %#v", failedRun)
	}
	if cancelledRun.Reason != "operator canceled" {
		t.Fatalf("canceled run = %#v", cancelledRun)
	}
}

func TestBaseHandlersTaskActorResolverErrors(t *testing.T) {
	t.Parallel()

	fixture := newHandlerFixtureWithTasks(
		t,
		testutil.StubSessionManager{},
		testutil.StubObserver{},
		testutil.StubTaskManager{},
		testutil.StubWorkspaceService{},
		nil,
		nil,
	)
	fixture.Handlers.TaskActorContextResolver = func(*gin.Context, string) (taskpkg.ActorContext, error) {
		return taskpkg.ActorContext{}, errors.New("resolver failed")
	}

	requests := []struct {
		method string
		path   string
		body   []byte
	}{
		{method: http.MethodGet, path: "/tasks"},
		{method: http.MethodPost, path: "/tasks", body: []byte(`{"scope":"global","title":"Review task API"}`)},
		{method: http.MethodGet, path: "/tasks/task-1"},
		{method: http.MethodDelete, path: "/tasks/task-1"},
		{method: http.MethodPatch, path: "/tasks/task-1", body: []byte(`{"title":"Renamed task"}`)},
		{method: http.MethodPost, path: "/tasks/task-1/cancel", body: []byte(`{}`)},
		{
			method: http.MethodPost,
			path:   "/tasks/task-root/children",
			body:   []byte(`{"scope":"global","title":"Child task"}`),
		},
		{
			method: http.MethodPost,
			path:   "/tasks/task-1/dependencies",
			body:   []byte(`{"depends_on_task_id":"task-blocker"}`),
		},
		{method: http.MethodDelete, path: "/tasks/task-1/dependencies/task-blocker"},
		{method: http.MethodGet, path: "/tasks/task-1/runs"},
		{method: http.MethodPost, path: "/tasks/task-1/runs", body: []byte(`{}`)},
		{method: http.MethodPost, path: "/task-runs/run-1/claim", body: []byte(`{}`)},
		{method: http.MethodPost, path: "/task-runs/run-1/start", body: []byte(`{}`)},
		{method: http.MethodPost, path: "/task-runs/run-1/attach-session", body: []byte(`{"session_id":"sess-1"}`)},
		{method: http.MethodPost, path: "/task-runs/run-1/complete", body: []byte(`{}`)},
		{method: http.MethodPost, path: "/task-runs/run-1/fail", body: []byte(`{"error":"boom"}`)},
		{method: http.MethodPost, path: "/task-runs/run-1/cancel", body: []byte(`{}`)},
	}

	for _, request := range requests {
		t.Run(request.method+" "+request.path, func(t *testing.T) {
			resp := performRequest(t, fixture.Engine, request.method, request.path, request.body)
			if resp.Code != http.StatusInternalServerError {
				t.Fatalf(
					"%s %s status = %d, want %d; body=%s",
					request.method,
					request.path,
					resp.Code,
					http.StatusInternalServerError,
					resp.Body.String(),
				)
			}
		})
	}
}

func TestBaseHandlersTaskServiceUnavailable(t *testing.T) {
	t.Parallel()

	fixture := newHandlerFixtureWithTasks(
		t,
		testutil.StubSessionManager{},
		testutil.StubObserver{},
		testutil.StubTaskManager{},
		testutil.StubWorkspaceService{},
		nil,
		nil,
	)
	fixture.Handlers.Tasks = nil

	requests := []struct {
		method string
		path   string
		body   []byte
	}{
		{method: http.MethodGet, path: "/tasks"},
		{method: http.MethodPost, path: "/tasks", body: []byte(`{"scope":"global","title":"Review task API"}`)},
		{method: http.MethodGet, path: "/tasks/task-1"},
		{method: http.MethodDelete, path: "/tasks/task-1"},
		{method: http.MethodPatch, path: "/tasks/task-1", body: []byte(`{"title":"Renamed task"}`)},
		{method: http.MethodPost, path: "/tasks/task-1/cancel", body: []byte(`{}`)},
		{
			method: http.MethodPost,
			path:   "/tasks/task-root/children",
			body:   []byte(`{"scope":"global","title":"Child task"}`),
		},
		{
			method: http.MethodPost,
			path:   "/tasks/task-1/dependencies",
			body:   []byte(`{"depends_on_task_id":"task-blocker"}`),
		},
		{method: http.MethodDelete, path: "/tasks/task-1/dependencies/task-blocker"},
		{method: http.MethodGet, path: "/tasks/task-1/runs"},
		{method: http.MethodPost, path: "/tasks/task-1/runs", body: []byte(`{}`)},
		{method: http.MethodPost, path: "/task-runs/run-1/claim", body: []byte(`{}`)},
		{method: http.MethodPost, path: "/task-runs/run-1/start", body: []byte(`{}`)},
		{method: http.MethodPost, path: "/task-runs/run-1/attach-session", body: []byte(`{"session_id":"sess-1"}`)},
		{method: http.MethodPost, path: "/task-runs/run-1/complete", body: []byte(`{}`)},
		{method: http.MethodPost, path: "/task-runs/run-1/fail", body: []byte(`{"error":"boom"}`)},
		{method: http.MethodPost, path: "/task-runs/run-1/cancel", body: []byte(`{}`)},
	}

	for _, request := range requests {
		t.Run(request.method+" "+request.path, func(t *testing.T) {
			resp := performRequest(t, fixture.Engine, request.method, request.path, request.body)
			if resp.Code != http.StatusServiceUnavailable {
				t.Fatalf(
					"%s %s status = %d, want %d; body=%s",
					request.method,
					request.path,
					resp.Code,
					http.StatusServiceUnavailable,
					resp.Body.String(),
				)
			}
		})
	}
}

func TestBaseHandlersTaskManagerErrors(t *testing.T) {
	t.Parallel()

	fixture := newHandlerFixtureWithTasks(
		t,
		testutil.StubSessionManager{},
		testutil.StubObserver{},
		testutil.StubTaskManager{
			ListTasksFn: func(context.Context, taskpkg.Query, taskpkg.ActorContext) ([]taskpkg.Summary, error) {
				return nil, taskpkg.ErrPermissionDenied
			},
			CreateTaskFn: func(context.Context, taskpkg.CreateTask, taskpkg.ActorContext) (*taskpkg.Task, error) {
				return nil, taskpkg.ErrPermissionDenied
			},
			GetTaskFn: func(context.Context, string, taskpkg.ActorContext) (*taskpkg.View, error) {
				return nil, taskpkg.ErrTaskNotFound
			},
			DeleteTaskFn: func(context.Context, string, taskpkg.ActorContext) error {
				return taskpkg.ErrValidation
			},
			ListTaskRunsFn: func(context.Context, string, taskpkg.RunQuery, taskpkg.ActorContext) ([]taskpkg.Run, error) {
				return nil, taskpkg.ErrTaskNotFound
			},
			UpdateTaskFn: func(context.Context, string, taskpkg.Patch, taskpkg.ActorContext) (*taskpkg.Task, error) {
				return nil, taskpkg.ErrPermissionDenied
			},
			CancelTaskFn: func(context.Context, string, taskpkg.CancelTask, taskpkg.ActorContext) (*taskpkg.Task, error) {
				return nil, taskpkg.ErrInvalidStatusTransition
			},
			CreateChildTaskFn: func(context.Context, string, taskpkg.CreateTask, taskpkg.ActorContext) (*taskpkg.Task, error) {
				return nil, taskpkg.ErrPermissionDenied
			},
			AddDependencyFn: func(context.Context, taskpkg.AddDependency, taskpkg.ActorContext) error {
				return taskpkg.ErrCycleDetected
			},
			RemoveDependencyFn: func(context.Context, string, string, taskpkg.ActorContext) error {
				return taskpkg.ErrTaskDependencyNotFound
			},
			EnqueueRunFn: func(context.Context, taskpkg.EnqueueRun, taskpkg.ActorContext) (*taskpkg.Run, error) {
				return nil, taskpkg.ErrInvalidStatusTransition
			},
			ClaimRunFn: func(context.Context, string, taskpkg.ClaimRun, taskpkg.ActorContext) (*taskpkg.Run, error) {
				return nil, taskpkg.ErrTaskRunNotFound
			},
			StartRunFn: func(context.Context, string, taskpkg.StartRun, taskpkg.ActorContext) (*taskpkg.Run, error) {
				return nil, taskpkg.ErrInvalidStatusTransition
			},
			AttachRunSessionFn: func(context.Context, string, string, taskpkg.ActorContext) (*taskpkg.Run, error) {
				return nil, taskpkg.ErrSessionAlreadyBound
			},
			CompleteRunFn: func(context.Context, string, taskpkg.RunResult, taskpkg.ActorContext) (*taskpkg.Run, error) {
				return nil, taskpkg.ErrTaskRunNotFound
			},
			FailRunFn: func(context.Context, string, taskpkg.RunFailure, taskpkg.ActorContext) (*taskpkg.Run, error) {
				return nil, taskpkg.ErrTaskRunNotFound
			},
			CancelRunFn: func(context.Context, string, taskpkg.CancelRun, taskpkg.ActorContext) (*taskpkg.Run, error) {
				return nil, taskpkg.ErrTaskRunNotFound
			},
		},
		testutil.StubWorkspaceService{},
		nil,
		nil,
	)

	requests := []struct {
		method string
		path   string
		body   []byte
		want   int
	}{
		{method: http.MethodGet, path: "/tasks", want: http.StatusForbidden},
		{
			method: http.MethodPost,
			path:   "/tasks",
			body:   []byte(`{"scope":"global","title":"Review task API"}`),
			want:   http.StatusForbidden,
		},
		{method: http.MethodGet, path: "/tasks/task-1", want: http.StatusNotFound},
		{method: http.MethodDelete, path: "/tasks/task-1", want: http.StatusBadRequest},
		{
			method: http.MethodPatch,
			path:   "/tasks/task-1",
			body:   []byte(`{"title":"Renamed task"}`),
			want:   http.StatusForbidden,
		},
		{method: http.MethodPost, path: "/tasks/task-1/cancel", body: []byte(`{}`), want: http.StatusConflict},
		{
			method: http.MethodPost,
			path:   "/tasks/task-root/children",
			body:   []byte(`{"scope":"global","title":"Child task"}`),
			want:   http.StatusForbidden,
		},
		{
			method: http.MethodPost,
			path:   "/tasks/task-1/dependencies",
			body:   []byte(`{"depends_on_task_id":"task-blocker"}`),
			want:   http.StatusConflict,
		},
		{method: http.MethodDelete, path: "/tasks/task-1/dependencies/task-blocker", want: http.StatusNotFound},
		{method: http.MethodGet, path: "/tasks/task-1/runs", want: http.StatusNotFound},
		{method: http.MethodPost, path: "/tasks/task-1/runs", body: []byte(`{}`), want: http.StatusConflict},
		{method: http.MethodPost, path: "/task-runs/run-1/claim", body: []byte(`{}`), want: http.StatusNotFound},
		{method: http.MethodPost, path: "/task-runs/run-1/start", body: []byte(`{}`), want: http.StatusConflict},
		{
			method: http.MethodPost,
			path:   "/task-runs/run-1/attach-session",
			body:   []byte(`{"session_id":"sess-1"}`),
			want:   http.StatusConflict,
		},
		{method: http.MethodPost, path: "/task-runs/run-1/complete", body: []byte(`{}`), want: http.StatusNotFound},
		{
			method: http.MethodPost,
			path:   "/task-runs/run-1/fail",
			body:   []byte(`{"error":"boom"}`),
			want:   http.StatusNotFound,
		},
		{method: http.MethodPost, path: "/task-runs/run-1/cancel", body: []byte(`{}`), want: http.StatusNotFound},
	}

	for _, request := range requests {
		t.Run(request.method+" "+request.path, func(t *testing.T) {
			resp := performRequest(t, fixture.Engine, request.method, request.path, request.body)
			if resp.Code != request.want {
				t.Fatalf(
					"%s %s status = %d, want %d; body=%s",
					request.method,
					request.path,
					resp.Code,
					request.want,
					resp.Body.String(),
				)
			}
		})
	}
}

func TestBaseHandlersTaskDecodeErrors(t *testing.T) {
	t.Parallel()

	fixture := newHandlerFixtureWithTasks(
		t,
		testutil.StubSessionManager{},
		testutil.StubObserver{},
		testutil.StubTaskManager{},
		testutil.StubWorkspaceService{},
		nil,
		nil,
	)

	requests := []struct {
		method string
		path   string
		body   []byte
	}{
		{method: http.MethodPost, path: "/tasks", body: []byte(`{"scope":`)},
		{method: http.MethodPatch, path: "/tasks/task-1", body: []byte(`{"title":`)},
		{method: http.MethodPost, path: "/tasks/task-1/cancel", body: []byte(`{"reason":`)},
		{method: http.MethodPost, path: "/tasks/task-root/children", body: []byte(`{"scope":`)},
		{method: http.MethodPost, path: "/tasks/task-1/dependencies", body: []byte(`{"depends_on_task_id":`)},
		{method: http.MethodPost, path: "/tasks/task-1/runs", body: []byte(`{"idempotency_key":`)},
		{method: http.MethodPost, path: "/task-runs/run-1/claim", body: []byte(`{"idempotency_key":`)},
		{method: http.MethodPost, path: "/task-runs/run-1/start", body: []byte(`{"idempotency_key":`)},
		{method: http.MethodPost, path: "/task-runs/run-1/attach-session", body: []byte(`{"session_id":`)},
		{method: http.MethodPost, path: "/task-runs/run-1/complete", body: []byte(`{"result":`)},
		{method: http.MethodPost, path: "/task-runs/run-1/fail", body: []byte(`{"error":`)},
		{method: http.MethodPost, path: "/task-runs/run-1/cancel", body: []byte(`{"reason":`)},
	}

	for _, request := range requests {
		t.Run(request.method+" "+request.path, func(t *testing.T) {
			resp := performRequest(t, fixture.Engine, request.method, request.path, request.body)
			if resp.Code != http.StatusBadRequest {
				t.Fatalf(
					"%s %s status = %d, want %d; body=%s",
					request.method,
					request.path,
					resp.Code,
					http.StatusBadRequest,
					resp.Body.String(),
				)
			}
		})
	}
}
