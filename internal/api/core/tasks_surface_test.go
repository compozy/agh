package core_test

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/api/core"
	"github.com/pedronauck/agh/internal/api/testutil"
	"github.com/pedronauck/agh/internal/observe"
	taskpkg "github.com/pedronauck/agh/internal/task"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func TestExpandedTaskPayloadBuildersPreserveLiveAndAggregateFields(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)
	lastActivity := now.Add(-time.Minute)
	lastSeen := now.Add(-2 * time.Minute)
	toolCalls := int64(3)
	totalCost := 1.75
	currency := "USD"

	summary := taskpkg.Summary{
		ID:              "task-1",
		Identifier:      "TASK-1",
		Scope:           taskpkg.ScopeWorkspace,
		WorkspaceID:     "ws-alpha",
		ParentTaskID:    "task-root",
		NetworkChannel:  "builders",
		Title:           "Review handlers",
		Priority:        taskpkg.PriorityHigh,
		MaxAttempts:     4,
		Status:          taskpkg.TaskStatusBlocked,
		ApprovalPolicy:  taskpkg.ApprovalPolicyManual,
		ApprovalState:   taskpkg.ApprovalStatePending,
		Draft:           false,
		Owner:           &taskpkg.Ownership{Kind: taskpkg.OwnerKindPool, Ref: "reviewers"},
		CreatedBy:       taskpkg.ActorIdentity{Kind: taskpkg.ActorKindHuman, Ref: "user-1"},
		Origin:          taskpkg.Origin{Kind: taskpkg.OriginKindHTTP, Ref: "tasks.create"},
		CreatedAt:       now.Add(-2 * time.Hour),
		UpdatedAt:       now,
		ChildCount:      2,
		DependencyCount: 1,
		Dependencies: []taskpkg.DependencyReference{{
			TaskID:          "task-1",
			DependsOnTaskID: "task-blocker",
			Kind:            taskpkg.DependencyKindBlocks,
			CreatedAt:       now.Add(-time.Hour),
			DependsOn: taskpkg.Reference{
				ID:          "task-blocker",
				Identifier:  "TASK-2",
				Title:       "Blocked task",
				Status:      taskpkg.TaskStatusReady,
				Priority:    taskpkg.PriorityUrgent,
				Owner:       &taskpkg.Ownership{Kind: taskpkg.OwnerKindHuman, Ref: "alice"},
				Scope:       taskpkg.ScopeWorkspace,
				WorkspaceID: "ws-alpha",
			},
		}},
		ActiveRun: &taskpkg.RunSummary{
			ID:          "run-1",
			TaskID:      "task-1",
			Status:      taskpkg.TaskRunStatusRunning,
			Attempt:     2,
			MaxAttempts: 4,
			SessionID:   "sess-1",
			ClaimedBy:   &taskpkg.ActorIdentity{Kind: taskpkg.ActorKindHuman, Ref: "user-1"},
			QueuedAt:    now.Add(-10 * time.Minute),
			StartedAt:   now.Add(-5 * time.Minute),
		},
		LastActivityAt: lastActivity,
	}

	summaryPayload := core.TaskSummaryPayloadFromSummary(summary)
	if summaryPayload.Priority != taskpkg.PriorityHigh ||
		summaryPayload.MaxAttempts != 4 ||
		summaryPayload.ApprovalState != taskpkg.ApprovalStatePending ||
		summaryPayload.DependencyCount != 1 ||
		summaryPayload.ActiveRun == nil ||
		summaryPayload.LastActivityAt == nil ||
		summaryPayload.Dependencies[0].DependsOn.Identifier != "TASK-2" {
		t.Fatalf("TaskSummaryPayloadFromSummary() = %#v", summaryPayload)
	}

	detailPayload := core.TaskDetailPayloadFromView(&taskpkg.View{
		Summary: summary,
		Task: taskpkg.Task{
			ID:             "task-1",
			Identifier:     "TASK-1",
			Scope:          taskpkg.ScopeWorkspace,
			WorkspaceID:    "ws-alpha",
			ParentTaskID:   "task-root",
			NetworkChannel: "builders",
			Title:          "Review handlers",
			Description:    "Deep detail",
			Priority:       taskpkg.PriorityHigh,
			MaxAttempts:    4,
			Status:         taskpkg.TaskStatusBlocked,
			ApprovalPolicy: taskpkg.ApprovalPolicyManual,
			ApprovalState:  taskpkg.ApprovalStatePending,
			Owner:          &taskpkg.Ownership{Kind: taskpkg.OwnerKindPool, Ref: "reviewers"},
			CreatedBy:      taskpkg.ActorIdentity{Kind: taskpkg.ActorKindHuman, Ref: "user-1"},
			Origin:         taskpkg.Origin{Kind: taskpkg.OriginKindHTTP, Ref: "tasks.create"},
			CreatedAt:      now.Add(-2 * time.Hour),
			UpdatedAt:      now,
			Metadata:       json.RawMessage(`{"priority":"high"}`),
		},
		DependencyReferences: summary.Dependencies,
	})
	if detailPayload.Summary.Priority != taskpkg.PriorityHigh ||
		detailPayload.Task.MaxAttempts != 4 ||
		detailPayload.Task.ApprovalPolicy != taskpkg.ApprovalPolicyManual ||
		len(detailPayload.DependencyReferences) != 1 {
		t.Fatalf("TaskDetailPayloadFromView() = %#v", detailPayload)
	}

	runDetailPayload := core.TaskRunDetailPayloadFromView(&taskpkg.RunDetailView{
		Run: taskpkg.Run{
			ID:       "run-1",
			TaskID:   "task-1",
			Status:   taskpkg.TaskRunStatusRunning,
			Attempt:  2,
			Origin:   taskpkg.Origin{Kind: taskpkg.OriginKindHTTP, Ref: "tasks.start_run"},
			QueuedAt: now.Add(-10 * time.Minute),
		},
		Task: taskpkg.Reference{
			ID:          "task-1",
			Identifier:  "TASK-1",
			Title:       "Review handlers",
			Status:      taskpkg.TaskStatusInProgress,
			Priority:    taskpkg.PriorityHigh,
			Owner:       &taskpkg.Ownership{Kind: taskpkg.OwnerKindPool, Ref: "reviewers"},
			Scope:       taskpkg.ScopeWorkspace,
			WorkspaceID: "ws-alpha",
		},
		Session: &taskpkg.RunSessionRef{
			SessionID:   "sess-1",
			WorkspaceID: "ws-alpha",
			AgentName:   "coder",
			Name:        "Task Runner",
			Channel:     "builders",
			State:       "active",
			CreatedAt:   now.Add(-9 * time.Minute),
			UpdatedAt:   now,
		},
		Summary: taskpkg.RunOperationalSummary{
			LastActivityAt: lastActivity,
			LastEventType:  "task.run.started",
			ToolCallCount:  &toolCalls,
			TotalCost:      &totalCost,
			CostCurrency:   &currency,
		},
	})
	if runDetailPayload.Session == nil ||
		runDetailPayload.Session.AgentName != "coder" ||
		runDetailPayload.Summary.ToolCallCount == nil ||
		*runDetailPayload.Summary.ToolCallCount != 3 ||
		runDetailPayload.Task.Priority != taskpkg.PriorityHigh {
		t.Fatalf("TaskRunDetailPayloadFromView() = %#v", runDetailPayload)
	}
	if nilSession := core.TaskRunSessionPayloadFromSession(nil); nilSession != nil {
		t.Fatalf("TaskRunSessionPayloadFromSession(nil) = %#v, want nil", nilSession)
	}
	if nilRunDetail := core.TaskRunDetailPayloadFromView(
		nil,
	); !reflect.DeepEqual(
		nilRunDetail,
		contract.TaskRunDetailPayload{},
	) {
		t.Fatalf("TaskRunDetailPayloadFromView(nil) = %#v, want zero value", nilRunDetail)
	}

	treePayload := core.TaskTreePayloadFromView(&taskpkg.TreeView{
		Root: taskpkg.TreeNode{
			Task: taskpkg.Reference{
				ID:          "task-root",
				Identifier:  "TASK-ROOT",
				Title:       "Root task",
				Status:      taskpkg.TaskStatusInProgress,
				Priority:    taskpkg.PriorityUrgent,
				Scope:       taskpkg.ScopeWorkspace,
				WorkspaceID: "ws-alpha",
			},
			Depth:          0,
			ChildCount:     1,
			ActiveRun:      summary.ActiveRun,
			LastActivityAt: now,
		},
		Descendants: []taskpkg.TreeNode{{
			Task: taskpkg.Reference{
				ID:          "task-1",
				Identifier:  "TASK-1",
				Title:       "Review handlers",
				Status:      taskpkg.TaskStatusBlocked,
				Priority:    taskpkg.PriorityHigh,
				Scope:       taskpkg.ScopeWorkspace,
				WorkspaceID: "ws-alpha",
			},
			ParentTaskID:   "task-root",
			Depth:          1,
			ChildCount:     0,
			LastActivityAt: lastActivity,
		}},
	})
	if treePayload.Root.ActiveRun == nil || len(treePayload.Descendants) != 1 ||
		treePayload.Descendants[0].ParentTaskID != "task-root" {
		t.Fatalf("TaskTreePayloadFromView() = %#v", treePayload)
	}

	dashboardPayload := core.TaskDashboardPayloadFromView(&observe.TaskDashboardView{
		Totals: observe.TaskDashboardTotals{TasksTotal: 3, RunsTotal: 2, ActiveRuns: 1, AwaitingApprovalTasks: 1},
		Cards: observe.TaskDashboardCards{
			InProgress: observe.TaskDashboardInProgressCard{Tasks: 1, ActiveRuns: 1, HealthStatus: "healthy"},
			Blocked:    observe.TaskDashboardBlockedCard{Tasks: 1, AwaitingApproval: 1, HealthStatus: "warning"},
			Failed:     observe.TaskDashboardFailedCard{Tasks: 1, FailedRuns: 1, HealthStatus: "warning"},
			Latency: observe.TaskDashboardLatencyCard{
				ClaimLatencyMillis: observe.LatencyMetric{Samples: 2, AverageMillis: 50, MaximumMillis: 75},
				StartLatencyMillis: observe.LatencyMetric{Samples: 2, AverageMillis: 60, MaximumMillis: 90},
			},
		},
		Queue: observe.TaskDashboardQueue{
			Total: 1,
			Depth: []observe.TaskQueueDepth{
				{
					NetworkChannel:      "builders",
					Count:               1,
					OldestQueuedAt:      now.Add(-5 * time.Minute),
					OldestQueueAgeMilli: 300000,
				},
			},
			OldestQueuedAt:        now.Add(-5 * time.Minute),
			OldestQueueAgeMilli:   300000,
			BacklogWarning:        true,
			BacklogStatus:         "warning",
			BacklogThresholdMilli: 120000,
		},
		Health: observe.TaskDashboardHealth{Status: "warning", StuckRuns: 1, QueueBacklog: true},
		StatusBreakdown: []observe.TaskDashboardStatusBreakdown{{
			Status:       taskpkg.TaskStatusInProgress,
			Count:        1,
			SharePercent: 33,
		}},
		ActiveRuns: observe.TaskDashboardActiveRuns{
			Total: 1, Running: 1,
			Items: []observe.TaskDashboardActiveRun{{
				TaskID:         "task-1",
				TaskIdentifier: "TASK-1",
				TaskTitle:      "Review handlers",
				TaskStatus:     taskpkg.TaskStatusInProgress,
				TaskPriority:   taskpkg.PriorityHigh,
				TaskOwner:      &taskpkg.Ownership{Kind: taskpkg.OwnerKindPool, Ref: "reviewers"},
				Scope:          taskpkg.ScopeWorkspace,
				WorkspaceID:    "ws-alpha",
				RunID:          "run-1",
				RunStatus:      taskpkg.TaskRunStatusRunning,
				Attempt:        2,
				MaxAttempts:    4,
				SessionID:      "sess-1",
				NetworkChannel: "builders",
				LastActivityAt: lastActivity,
				AgeMilli:       60000,
				HealthStatus:   "healthy",
			}},
		},
		Freshness: observe.TaskDashboardFreshness{
			ObservedAt:       now,
			LatestActivityAt: lastActivity,
			AgeMilli:         60000,
			StaleAfterMilli:  120000,
			HasLiveWork:      true,
			Status:           "live",
		},
	})
	if dashboardPayload.Queue.Depth[0].NetworkChannel != "builders" ||
		dashboardPayload.ActiveRuns.Items[0].RunID != "run-1" ||
		len(dashboardPayload.StatusBreakdown) != 1 ||
		dashboardPayload.StatusBreakdown[0].Status != taskpkg.TaskStatusInProgress ||
		dashboardPayload.Freshness.Status != "live" {
		t.Fatalf("TaskDashboardPayloadFromView() = %#v", dashboardPayload)
	}

	inboxPayload := core.TaskInboxPayloadFromView(observe.TaskInboxView{
		Total:         1,
		UnreadTotal:   1,
		ArchivedTotal: 0,
		Groups: []observe.TaskInboxLaneGroup{{
			Lane:        observe.TaskInboxLaneApprovals,
			Count:       1,
			UnreadCount: 1,
			Items: []observe.TaskInboxItem{{
				Task: taskpkg.Reference{
					ID:          "task-1",
					Identifier:  "TASK-1",
					Title:       "Review handlers",
					Status:      taskpkg.TaskStatusBlocked,
					Priority:    taskpkg.PriorityHigh,
					Scope:       taskpkg.ScopeWorkspace,
					WorkspaceID: "ws-alpha",
				},
				Lane:             observe.TaskInboxLaneApprovals,
				ApprovalPolicy:   taskpkg.ApprovalPolicyManual,
				ApprovalState:    taskpkg.ApprovalStatePending,
				BlockingReason:   "awaiting_approval",
				LatestActivityAt: lastActivity,
				Run:              summary.ActiveRun,
				Triage: taskpkg.TriageState{
					TaskID:             "task-1",
					Actor:              taskpkg.ActorIdentity{Kind: taskpkg.ActorKindHuman, Ref: "user-1"},
					LastSeenActivityAt: lastSeen,
					UpdatedAt:          now,
				},
			}},
		}},
	})
	if inboxPayload.Groups[0].Items[0].Run == nil ||
		inboxPayload.Groups[0].Items[0].Triage.LastSeenActivityAt == nil ||
		inboxPayload.Groups[0].Items[0].BlockingReason != "awaiting_approval" {
		t.Fatalf("TaskInboxPayloadFromView() = %#v", inboxPayload)
	}
}

func TestBaseHandlersExpandedTaskEndpoints(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 17, 13, 0, 0, 0, time.UTC)
	var publishActor taskpkg.ActorContext
	var runDetailActor taskpkg.ActorContext
	var timelineActor taskpkg.ActorContext
	var timelineQuery taskpkg.TimelineQuery
	var treeActor taskpkg.ActorContext
	var dashboardQuery observe.TaskDashboardQuery
	var inboxQuery observe.TaskInboxQuery
	var inboxActor taskpkg.ActorIdentity
	var approveActor taskpkg.ActorContext
	var readActor taskpkg.ActorContext
	var archiveActor taskpkg.ActorContext
	var dismissActor taskpkg.ActorContext
	var streamActor taskpkg.ActorContext
	var streamQuery taskpkg.StreamQuery

	tasks := testutil.StubTaskManager{
		PublishTaskFn: func(_ context.Context, id string, actor taskpkg.ActorContext) (*taskpkg.Task, error) {
			publishActor = actor
			return &taskpkg.Task{
				ID:             id,
				Identifier:     "TASK-1",
				Scope:          taskpkg.ScopeWorkspace,
				WorkspaceID:    "ws-alpha",
				Title:          "Published task",
				Priority:       taskpkg.PriorityHigh,
				MaxAttempts:    3,
				Status:         taskpkg.TaskStatusReady,
				ApprovalPolicy: taskpkg.ApprovalPolicyNone,
				ApprovalState:  taskpkg.ApprovalStateNotRequired,
				CreatedBy:      actor.Actor,
				Origin:         actor.Origin,
				CreatedAt:      now.Add(-time.Hour),
				UpdatedAt:      now,
			}, nil
		},
		RunDetailFn: func(_ context.Context, id string, actor taskpkg.ActorContext) (*taskpkg.RunDetailView, error) {
			runDetailActor = actor
			return &taskpkg.RunDetailView{
				Run: taskpkg.Run{
					ID:       id,
					TaskID:   "task-1",
					Status:   taskpkg.TaskRunStatusRunning,
					Attempt:  2,
					Origin:   actor.Origin,
					QueuedAt: now.Add(-10 * time.Minute),
				},
				Task: taskpkg.Reference{
					ID:          "task-1",
					Identifier:  "TASK-1",
					Title:       "Published task",
					Status:      taskpkg.TaskStatusInProgress,
					Priority:    taskpkg.PriorityHigh,
					Scope:       taskpkg.ScopeWorkspace,
					WorkspaceID: "ws-alpha",
				},
				Session: &taskpkg.RunSessionRef{
					SessionID:   "sess-1",
					WorkspaceID: "ws-alpha",
					AgentName:   "coder",
					Name:        "Runner",
					Channel:     "builders",
					State:       "active",
					CreatedAt:   now.Add(-9 * time.Minute),
					UpdatedAt:   now,
				},
			}, nil
		},
		TimelineFn: func(_ context.Context, id string, query taskpkg.TimelineQuery, actor taskpkg.ActorContext) ([]taskpkg.TimelineItem, error) {
			timelineActor = actor
			timelineQuery = query
			return []taskpkg.TimelineItem{{
				Sequence: 11,
				EventID:  "evt-11",
				Task: taskpkg.Reference{
					ID:          id,
					Identifier:  "TASK-1",
					Title:       "Published task",
					Status:      taskpkg.TaskStatusInProgress,
					Priority:    taskpkg.PriorityHigh,
					Scope:       taskpkg.ScopeWorkspace,
					WorkspaceID: "ws-alpha",
				},
				Run: &taskpkg.RunSummary{
					ID:          "run-1",
					TaskID:      id,
					Status:      taskpkg.TaskRunStatusRunning,
					Attempt:     2,
					MaxAttempts: 3,
					SessionID:   "sess-1",
					QueuedAt:    now.Add(-10 * time.Minute),
				},
				EventType: "task.run.started",
				Actor:     actor.Actor,
				Origin:    actor.Origin,
				Payload:   json.RawMessage(`{"status":"running"}`),
				Timestamp: now,
			}}, nil
		},
		StreamFn: func(_ context.Context, _ string, query taskpkg.StreamQuery, actor taskpkg.ActorContext) (<-chan taskpkg.StreamEvent, error) {
			streamActor = actor
			streamQuery = query
			ch := make(chan taskpkg.StreamEvent, 1)
			ch <- taskpkg.StreamEvent{
				Sequence: 13,
				Type:     "task.run.started",
				Timeline: taskpkg.TimelineItem{
					Sequence: 13,
					EventID:  "evt-13",
					Task: taskpkg.Reference{
						ID:          "task-1",
						Identifier:  "TASK-1",
						Title:       "Published task",
						Status:      taskpkg.TaskStatusInProgress,
						Priority:    taskpkg.PriorityHigh,
						Scope:       taskpkg.ScopeWorkspace,
						WorkspaceID: "ws-alpha",
					},
					EventType: "task.run.started",
					Actor:     actor.Actor,
					Origin:    actor.Origin,
					Timestamp: now,
				},
			}
			close(ch)
			return ch, nil
		},
		TreeFn: func(_ context.Context, _ string, actor taskpkg.ActorContext) (*taskpkg.TreeView, error) {
			treeActor = actor
			return &taskpkg.TreeView{
				Root: taskpkg.TreeNode{
					Task: taskpkg.Reference{
						ID:          "task-1",
						Identifier:  "TASK-1",
						Title:       "Published task",
						Status:      taskpkg.TaskStatusInProgress,
						Priority:    taskpkg.PriorityHigh,
						Scope:       taskpkg.ScopeWorkspace,
						WorkspaceID: "ws-alpha",
					},
					Depth:          0,
					ChildCount:     1,
					LastActivityAt: now,
				},
			}, nil
		},
		ApproveTaskFn: func(_ context.Context, id string, actor taskpkg.ActorContext) (*taskpkg.Task, error) {
			approveActor = actor
			return &taskpkg.Task{
				ID:             id,
				Scope:          taskpkg.ScopeWorkspace,
				WorkspaceID:    "ws-alpha",
				Title:          "Approval task",
				Status:         taskpkg.TaskStatusReady,
				ApprovalPolicy: taskpkg.ApprovalPolicyManual,
				ApprovalState:  taskpkg.ApprovalStateApproved,
				CreatedBy:      actor.Actor,
				Origin:         actor.Origin,
				CreatedAt:      now.Add(-time.Hour),
				UpdatedAt:      now,
			}, nil
		},
		MarkTaskReadFn: func(_ context.Context, id string, actor taskpkg.ActorContext) (taskpkg.TriageState, error) {
			readActor = actor
			return taskpkg.TriageState{TaskID: id, Actor: actor.Actor, Read: true, UpdatedAt: now}, nil
		},
		ArchiveTaskFn: func(_ context.Context, id string, actor taskpkg.ActorContext) (taskpkg.TriageState, error) {
			archiveActor = actor
			return taskpkg.TriageState{TaskID: id, Actor: actor.Actor, Read: true, Archived: true, UpdatedAt: now}, nil
		},
		DismissTaskFn: func(_ context.Context, id string, actor taskpkg.ActorContext) (taskpkg.TriageState, error) {
			dismissActor = actor
			return taskpkg.TriageState{TaskID: id, Actor: actor.Actor, Dismissed: true, UpdatedAt: now}, nil
		},
	}
	observer := testutil.StubObserver{
		QueryTaskDashboardFn: func(_ context.Context, query observe.TaskDashboardQuery) (observe.TaskDashboardView, error) {
			dashboardQuery = query
			return observe.TaskDashboardView{
				Totals:     observe.TaskDashboardTotals{TasksTotal: 3, RunsTotal: 1, ActiveRuns: 1},
				Cards:      observe.TaskDashboardCards{},
				Queue:      observe.TaskDashboardQueue{Total: 1, OldestQueuedAt: now, BacklogStatus: "ok"},
				Health:     observe.TaskDashboardHealth{Status: "ok"},
				ActiveRuns: observe.TaskDashboardActiveRuns{Total: 1, Running: 1},
				Freshness:  observe.TaskDashboardFreshness{ObservedAt: now, LatestActivityAt: now, Status: "live"},
			}, nil
		},
		QueryTaskInboxFn: func(_ context.Context, query observe.TaskInboxQuery, actor taskpkg.ActorIdentity) (observe.TaskInboxView, error) {
			inboxQuery = query
			inboxActor = actor
			return observe.TaskInboxView{
				Total:       1,
				UnreadTotal: 1,
				Groups: []observe.TaskInboxLaneGroup{{
					Lane:        observe.TaskInboxLaneApprovals,
					Count:       1,
					UnreadCount: 1,
					Items: []observe.TaskInboxItem{{
						Task: taskpkg.Reference{
							ID:          "task-1",
							Identifier:  "TASK-1",
							Title:       "Approval task",
							Status:      taskpkg.TaskStatusBlocked,
							Priority:    taskpkg.PriorityHigh,
							Scope:       taskpkg.ScopeWorkspace,
							WorkspaceID: "ws-alpha",
						},
						Lane:             observe.TaskInboxLaneApprovals,
						ApprovalPolicy:   taskpkg.ApprovalPolicyManual,
						ApprovalState:    taskpkg.ApprovalStatePending,
						BlockingReason:   "awaiting_approval",
						LatestActivityAt: now,
						Triage: taskpkg.TriageState{
							TaskID:    "task-1",
							Actor:     actor,
							UpdatedAt: now,
						},
					}},
				}},
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

	fixture := newHandlerFixtureWithTasks(t, testutil.StubSessionManager{}, observer, tasks, workspaces, nil, nil)
	fixture.Handlers.TaskActorContextResolver = func(_ *gin.Context, action string) (taskpkg.ActorContext, error) {
		return taskpkg.DeriveHumanActorContext("user-1", taskpkg.OriginKindHTTP, "tasks."+action)
	}

	t.Run("happy path json endpoints", func(t *testing.T) {
		t.Parallel()

		publishResp := performRequest(t, fixture.Engine, http.MethodPost, "/tasks/task-1/publish", nil)
		if publishResp.Code != http.StatusOK {
			t.Fatalf(
				"publish status = %d, want %d; body=%s",
				publishResp.Code,
				http.StatusOK,
				publishResp.Body.String(),
			)
		}
		var publishPayload contract.TaskResponse
		testutil.DecodeJSONResponse(t, publishResp, &publishPayload)
		if publishPayload.Task.Status != taskpkg.TaskStatusReady || publishActor.Origin.Ref != "tasks.publish" {
			t.Fatalf("publish payload/actor = %#v / %#v", publishPayload, publishActor)
		}

		runResp := performRequest(t, fixture.Engine, http.MethodGet, "/task-runs/run-1", nil)
		if runResp.Code != http.StatusOK {
			t.Fatalf("run detail status = %d, want %d; body=%s", runResp.Code, http.StatusOK, runResp.Body.String())
		}
		var runPayload contract.TaskRunDetailResponse
		testutil.DecodeJSONResponse(t, runResp, &runPayload)
		if runPayload.Run.Run.ID != "run-1" || runPayload.Run.Session == nil ||
			runDetailActor.Origin.Ref != "tasks.get_run" {
			t.Fatalf("run detail payload/actor = %#v / %#v", runPayload, runDetailActor)
		}

		timelineResp := performRequest(
			t,
			fixture.Engine,
			http.MethodGet,
			"/tasks/task-1/timeline?after_sequence=4&limit=3",
			nil,
		)
		if timelineResp.Code != http.StatusOK {
			t.Fatalf(
				"timeline status = %d, want %d; body=%s",
				timelineResp.Code,
				http.StatusOK,
				timelineResp.Body.String(),
			)
		}
		var timelinePayload contract.TaskTimelineResponse
		testutil.DecodeJSONResponse(t, timelineResp, &timelinePayload)
		if len(timelinePayload.Timeline) != 1 || timelinePayload.Timeline[0].Sequence != 11 {
			t.Fatalf("timeline payload = %#v", timelinePayload)
		}
		if timelineActor.Origin.Ref != "tasks.timeline" || timelineQuery.AfterSequence != 4 ||
			timelineQuery.Limit != 3 {
			t.Fatalf("timeline actor/query = %#v / %#v", timelineActor, timelineQuery)
		}

		treeResp := performRequest(t, fixture.Engine, http.MethodGet, "/tasks/task-1/tree", nil)
		if treeResp.Code != http.StatusOK {
			t.Fatalf("tree status = %d, want %d; body=%s", treeResp.Code, http.StatusOK, treeResp.Body.String())
		}
		var treePayload contract.TaskTreeResponse
		testutil.DecodeJSONResponse(t, treeResp, &treePayload)
		if treePayload.Tree.Root.Task.ID != "task-1" || treeActor.Origin.Ref != "tasks.tree" {
			t.Fatalf("tree payload/actor = %#v / %#v", treePayload, treeActor)
		}

		dashboardResp := performRequest(
			t,
			fixture.Engine,
			http.MethodGet,
			"/observe/tasks/dashboard?scope=workspace&workspace=alpha&owner_kind=human&owner_ref=alice&network_channel=builders&origin_kind=http",
			nil,
		)
		if dashboardResp.Code != http.StatusOK {
			t.Fatalf(
				"dashboard status = %d, want %d; body=%s",
				dashboardResp.Code,
				http.StatusOK,
				dashboardResp.Body.String(),
			)
		}
		var dashboardPayload contract.TaskDashboardResponse
		testutil.DecodeJSONResponse(t, dashboardResp, &dashboardPayload)
		if dashboardPayload.Dashboard.Totals.TasksTotal != 3 || dashboardQuery.WorkspaceID != "ws-alpha" ||
			dashboardQuery.OriginKind != taskpkg.OriginKindHTTP {
			t.Fatalf("dashboard payload/query = %#v / %#v", dashboardPayload, dashboardQuery)
		}

		inboxResp := performRequest(
			t,
			fixture.Engine,
			http.MethodGet,
			"/observe/tasks/inbox?scope=workspace&workspace=alpha&owner_kind=human&owner_ref=alice&lane=approvals&unread=true&query=approve&limit=2",
			nil,
		)
		if inboxResp.Code != http.StatusOK {
			t.Fatalf("inbox status = %d, want %d; body=%s", inboxResp.Code, http.StatusOK, inboxResp.Body.String())
		}
		var inboxPayload contract.TaskInboxResponse
		testutil.DecodeJSONResponse(t, inboxResp, &inboxPayload)
		if inboxPayload.Inbox.Total != 1 || len(inboxPayload.Inbox.Groups) != 1 {
			t.Fatalf("inbox payload = %#v", inboxPayload)
		}
		if inboxActor.Ref != "user-1" || inboxActor.Kind != taskpkg.ActorKindHuman ||
			inboxQuery.Lane != observe.TaskInboxLaneApprovals ||
			!inboxQuery.Unread {
			t.Fatalf("inbox actor/query = %#v / %#v", inboxActor, inboxQuery)
		}

		approveResp := performRequest(t, fixture.Engine, http.MethodPost, "/tasks/task-1/approve", nil)
		if approveResp.Code != http.StatusOK {
			t.Fatalf(
				"approve status = %d, want %d; body=%s",
				approveResp.Code,
				http.StatusOK,
				approveResp.Body.String(),
			)
		}
		if approveActor.Origin.Ref != "tasks.approve" {
			t.Fatalf("approve actor = %#v", approveActor)
		}

		readResp := performRequest(t, fixture.Engine, http.MethodPost, "/tasks/task-1/triage/read", nil)
		if readResp.Code != http.StatusOK {
			t.Fatalf("triage read status = %d, want %d; body=%s", readResp.Code, http.StatusOK, readResp.Body.String())
		}
		archiveResp := performRequest(t, fixture.Engine, http.MethodPost, "/tasks/task-1/triage/archive", nil)
		if archiveResp.Code != http.StatusOK {
			t.Fatalf(
				"triage archive status = %d, want %d; body=%s",
				archiveResp.Code,
				http.StatusOK,
				archiveResp.Body.String(),
			)
		}
		dismissResp := performRequest(t, fixture.Engine, http.MethodPost, "/tasks/task-1/triage/dismiss", nil)
		if dismissResp.Code != http.StatusOK {
			t.Fatalf(
				"triage dismiss status = %d, want %d; body=%s",
				dismissResp.Code,
				http.StatusOK,
				dismissResp.Body.String(),
			)
		}
		if readActor.Origin.Ref != "tasks.triage_read" ||
			archiveActor.Origin.Ref != "tasks.triage_archive" ||
			dismissActor.Origin.Ref != "tasks.triage_dismiss" {
			t.Fatalf("triage actors = %#v / %#v / %#v", readActor, archiveActor, dismissActor)
		}
	})

	t.Run("task stream sse", func(t *testing.T) {
		t.Parallel()

		streamResp := testutil.PerformRequestWithHeaders(
			t,
			fixture.Engine,
			http.MethodGet,
			"/tasks/task-1/stream?after_sequence=2",
			nil,
			map[string]string{"Last-Event-ID": "9"},
		)
		if streamResp.Code != http.StatusOK {
			t.Fatalf("stream status = %d, want %d; body=%s", streamResp.Code, http.StatusOK, streamResp.Body.String())
		}
		if got := streamResp.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/event-stream") {
			t.Fatalf("stream content-type = %q, want prefix %q", got, "text/event-stream")
		}
		if !streamResp.Flushed {
			t.Fatal("stream recorder was not flushed")
		}
		records := testutil.ParseSSE(t, streamResp.Body.String())
		if len(records) != 1 {
			t.Fatalf("stream records = %d, want 1; body=%s", len(records), streamResp.Body.String())
		}
		if records[0].ID != "13" || records[0].Event != "task.run.started" {
			t.Fatalf("stream record = %#v", records[0])
		}
		var payload contract.TaskStreamEventPayload
		testutil.DecodeSSEData(t, records[0], &payload)
		if payload.Sequence != 13 || payload.Timeline.EventID != "evt-13" {
			t.Fatalf("stream payload = %#v", payload)
		}
		if streamActor.Origin.Ref != "tasks.stream" || streamQuery.AfterSequence != 9 {
			t.Fatalf("stream actor/query = %#v / %#v", streamActor, streamQuery)
		}
	})
}

func TestBaseHandlersExpandedTaskEndpointErrorPaths(t *testing.T) {
	t.Parallel()

	workspaces := testutil.StubWorkspaceService{
		GetFn: func(_ context.Context, ref string) (workspacepkg.Workspace, error) {
			if ref == "missing" {
				return workspacepkg.Workspace{}, workspacepkg.ErrWorkspaceNotFound
			}
			return workspacepkg.Workspace{ID: "ws-alpha", Name: ref}, nil
		},
	}

	fixture := newHandlerFixtureWithTasks(
		t,
		testutil.StubSessionManager{},
		testutil.StubObserver{},
		testutil.StubTaskManager{
			PublishTaskFn: func(context.Context, string, taskpkg.ActorContext) (*taskpkg.Task, error) {
				return nil, taskpkg.ErrInvalidStatusTransition
			},
			ApproveTaskFn: func(context.Context, string, taskpkg.ActorContext) (*taskpkg.Task, error) {
				return nil, taskpkg.ErrInvalidStatusTransition
			},
			RejectTaskFn: func(context.Context, string, taskpkg.ActorContext) (*taskpkg.Task, error) {
				return nil, taskpkg.ErrTaskNotFound
			},
			MarkTaskReadFn: func(context.Context, string, taskpkg.ActorContext) (taskpkg.TriageState, error) {
				return taskpkg.TriageState{}, taskpkg.ErrTaskNotFound
			},
			RunDetailFn: func(context.Context, string, taskpkg.ActorContext) (*taskpkg.RunDetailView, error) {
				return nil, taskpkg.ErrTaskRunNotFound
			},
			TimelineFn: func(context.Context, string, taskpkg.TimelineQuery, taskpkg.ActorContext) ([]taskpkg.TimelineItem, error) {
				return nil, taskpkg.ErrTaskNotFound
			},
			StreamFn: func(context.Context, string, taskpkg.StreamQuery, taskpkg.ActorContext) (<-chan taskpkg.StreamEvent, error) {
				return nil, taskpkg.ErrTaskNotFound
			},
			TreeFn: func(context.Context, string, taskpkg.ActorContext) (*taskpkg.TreeView, error) {
				return nil, taskpkg.ErrTaskNotFound
			},
		},
		workspaces,
		nil,
		nil,
	)

	if resp := performRequest(
		t,
		fixture.Engine,
		http.MethodPost,
		"/tasks/task-1/publish",
		nil,
	); resp.Code != http.StatusConflict {
		t.Fatalf("publish conflict status = %d, want %d; body=%s", resp.Code, http.StatusConflict, resp.Body.String())
	}
	if resp := performRequest(
		t,
		fixture.Engine,
		http.MethodPost,
		"/tasks/task-1/approve",
		nil,
	); resp.Code != http.StatusConflict {
		t.Fatalf("approve conflict status = %d, want %d; body=%s", resp.Code, http.StatusConflict, resp.Body.String())
	}
	if resp := performRequest(
		t,
		fixture.Engine,
		http.MethodPost,
		"/tasks/task-1/reject",
		nil,
	); resp.Code != http.StatusNotFound {
		t.Fatalf("reject not found status = %d, want %d; body=%s", resp.Code, http.StatusNotFound, resp.Body.String())
	}
	if resp := performRequest(
		t,
		fixture.Engine,
		http.MethodPost,
		"/tasks/task-1/triage/read",
		nil,
	); resp.Code != http.StatusNotFound {
		t.Fatalf(
			"triage read not found status = %d, want %d; body=%s",
			resp.Code,
			http.StatusNotFound,
			resp.Body.String(),
		)
	}
	if resp := performRequest(
		t,
		fixture.Engine,
		http.MethodGet,
		"/task-runs/run-1",
		nil,
	); resp.Code != http.StatusNotFound {
		t.Fatalf(
			"run detail not found status = %d, want %d; body=%s",
			resp.Code,
			http.StatusNotFound,
			resp.Body.String(),
		)
	}
	if resp := performRequest(
		t,
		fixture.Engine,
		http.MethodGet,
		"/tasks/task-1/timeline?after_sequence=1",
		nil,
	); resp.Code != http.StatusNotFound {
		t.Fatalf("timeline not found status = %d, want %d; body=%s", resp.Code, http.StatusNotFound, resp.Body.String())
	}
	if resp := performRequest(
		t,
		fixture.Engine,
		http.MethodGet,
		"/tasks/task-1/stream",
		nil,
	); resp.Code != http.StatusNotFound {
		t.Fatalf("stream not found status = %d, want %d; body=%s", resp.Code, http.StatusNotFound, resp.Body.String())
	}
	if resp := performRequest(
		t,
		fixture.Engine,
		http.MethodGet,
		"/tasks/task-1/tree",
		nil,
	); resp.Code != http.StatusNotFound {
		t.Fatalf("tree not found status = %d, want %d; body=%s", resp.Code, http.StatusNotFound, resp.Body.String())
	}
	if resp := performRequest(
		t,
		fixture.Engine,
		http.MethodGet,
		"/observe/tasks/dashboard?scope=workspace&workspace=missing",
		nil,
	); resp.Code != http.StatusNotFound {
		t.Fatalf(
			"dashboard missing workspace status = %d, want %d; body=%s",
			resp.Code,
			http.StatusNotFound,
			resp.Body.String(),
		)
	}
	if resp := performRequest(
		t,
		fixture.Engine,
		http.MethodGet,
		"/observe/tasks/inbox?lane=bogus",
		nil,
	); resp.Code != http.StatusBadRequest {
		t.Fatalf(
			"inbox invalid lane status = %d, want %d; body=%s",
			resp.Code,
			http.StatusBadRequest,
			resp.Body.String(),
		)
	}
	if resp := testutil.PerformRequestWithHeaders(
		t,
		fixture.Engine,
		http.MethodGet,
		"/tasks/task-1/stream",
		nil,
		map[string]string{"Last-Event-ID": "bogus"},
	); resp.Code != http.StatusBadRequest {
		t.Fatalf(
			"stream invalid Last-Event-ID status = %d, want %d; body=%s",
			resp.Code,
			http.StatusBadRequest,
			resp.Body.String(),
		)
	}

	taskless := newHandlerFixture(t, testutil.StubSessionManager{}, testutil.StubObserver{}, workspaces, nil, nil)
	taskless.Handlers.Tasks = nil
	if resp := performRequest(
		t,
		taskless.Engine,
		http.MethodPost,
		"/tasks/task-1/publish",
		nil,
	); resp.Code != http.StatusServiceUnavailable {
		t.Fatalf(
			"publish without task service status = %d, want %d; body=%s",
			resp.Code,
			http.StatusServiceUnavailable,
			resp.Body.String(),
		)
	}

	taskless.Handlers.Observer = nil
	if resp := performRequest(
		t,
		taskless.Engine,
		http.MethodGet,
		"/observe/tasks/dashboard",
		nil,
	); resp.Code != http.StatusServiceUnavailable {
		t.Fatalf(
			"dashboard without observer status = %d, want %d; body=%s",
			resp.Code,
			http.StatusServiceUnavailable,
			resp.Body.String(),
		)
	}
}
