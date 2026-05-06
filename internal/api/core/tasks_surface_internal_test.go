package core

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/observe"
	taskpkg "github.com/pedronauck/agh/internal/task"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func newExpandedTaskHandlers(workspaceGet func(context.Context, string) (workspacepkg.Workspace, error)) *BaseHandlers {
	gin.SetMode(gin.TestMode)

	handlers := &BaseHandlers{TransportName: "api-core-test"}
	if workspaceGet != nil {
		handlers.Workspaces = workspaceServiceStub{get: workspaceGet}
	}
	return handlers
}

func TestExpandedTaskQueryParsingAndDomainConversion(t *testing.T) {
	t.Parallel()

	t.Run("Should parse and convert task list filters", func(t *testing.T) {
		t.Parallel()

		handlers := newExpandedTaskHandlers(func(_ context.Context, ref string) (workspacepkg.Workspace, error) {
			if ref != "alpha" {
				t.Fatalf("workspace ref = %q, want %q", ref, "alpha")
			}
			return workspacepkg.Workspace{ID: "ws-alpha"}, nil
		})

		recorder := httptest.NewRecorder()
		ginCtx, _ := gin.CreateTestContext(recorder)
		ginCtx.Request = httptest.NewRequestWithContext(
			context.Background(),
			http.MethodGet,
			"/tasks?scope=workspace&workspace=alpha&status=ready&priority=high&include_drafts=true&approval_state=pending&owner_kind=pool&owner_ref=reviewers&parent_task_id=task-root&network_channel=builders&query=review&limit=7",
			http.NoBody,
		)

		query, err := ParseTaskListQuery(ginCtx)
		if err != nil {
			t.Fatalf("ParseTaskListQuery() error = %v", err)
		}
		if !query.IncludeDrafts || query.Priority != taskpkg.PriorityHigh ||
			query.ApprovalState != taskpkg.ApprovalStatePending {
			t.Fatalf("ParseTaskListQuery() = %#v", query)
		}

		domainQuery, err := handlers.taskListDomainQuery(context.Background(), query)
		if err != nil {
			t.Fatalf("taskListDomainQuery() error = %v", err)
		}
		if domainQuery.WorkspaceID != "ws-alpha" ||
			domainQuery.Status != taskpkg.TaskStatusReady ||
			domainQuery.Priority != taskpkg.PriorityHigh ||
			domainQuery.ApprovalState != taskpkg.ApprovalStatePending ||
			domainQuery.OwnerKind != taskpkg.OwnerKindPool ||
			domainQuery.OwnerRef != "reviewers" ||
			domainQuery.ParentTaskID != "task-root" ||
			domainQuery.NetworkChannel != "builders" ||
			domainQuery.Search != "review" ||
			domainQuery.Limit != 7 {
			t.Fatalf("taskListDomainQuery() = %#v", domainQuery)
		}
	})

	t.Run("Should parse and convert timeline and stream filters", func(t *testing.T) {
		t.Parallel()

		handlers := newExpandedTaskHandlers(nil)

		recorder := httptest.NewRecorder()
		ginCtx, _ := gin.CreateTestContext(recorder)
		ginCtx.Request = httptest.NewRequestWithContext(
			context.Background(),
			http.MethodGet,
			"/tasks/task-1/timeline?after_sequence=42&limit=5",
			http.NoBody,
		)

		timelineQuery, err := ParseTaskTimelineQuery(ginCtx)
		if err != nil {
			t.Fatalf("ParseTaskTimelineQuery() error = %v", err)
		}
		if timelineQuery.AfterSequence != 42 || timelineQuery.Limit != 5 {
			t.Fatalf("ParseTaskTimelineQuery() = %#v", timelineQuery)
		}

		domainTimeline, err := taskTimelineDomainQuery(timelineQuery)
		if err != nil {
			t.Fatalf("taskTimelineDomainQuery() error = %v", err)
		}
		if domainTimeline.AfterSequence != 42 || domainTimeline.Limit != 5 {
			t.Fatalf("taskTimelineDomainQuery() = %#v", domainTimeline)
		}

		streamRecorder := httptest.NewRecorder()
		streamCtx, _ := gin.CreateTestContext(streamRecorder)
		streamCtx.Request = httptest.NewRequestWithContext(
			context.Background(),
			http.MethodGet,
			"/tasks/task-1/stream?after_sequence=2",
			http.NoBody,
		)
		streamCtx.Request.Header.Set("Last-Event-ID", "9")

		streamQuery, err := ParseTaskStreamQuery(streamCtx)
		if err != nil {
			t.Fatalf("ParseTaskStreamQuery() error = %v", err)
		}
		if streamQuery.AfterSequence != 2 {
			t.Fatalf("ParseTaskStreamQuery() = %#v", streamQuery)
		}

		domainStream, err := handlers.taskStreamDomainQuery(streamCtx, streamQuery)
		if err != nil {
			t.Fatalf("taskStreamDomainQuery() error = %v", err)
		}
		if domainStream.AfterSequence != 9 {
			t.Fatalf("taskStreamDomainQuery() = %#v, want after_sequence=9", domainStream)
		}

		zeroRecorder := httptest.NewRecorder()
		zeroCtx, _ := gin.CreateTestContext(zeroRecorder)
		zeroCtx.Request = httptest.NewRequestWithContext(
			context.Background(),
			http.MethodGet,
			"/tasks/task-1/stream?after_sequence=12",
			http.NoBody,
		)
		zeroCtx.Request.Header.Set("Last-Event-ID", "0")
		zeroQuery, err := ParseTaskStreamQuery(zeroCtx)
		if err != nil {
			t.Fatalf("ParseTaskStreamQuery(zero) error = %v", err)
		}
		zeroDomainStream, err := handlers.taskStreamDomainQuery(zeroCtx, zeroQuery)
		if err != nil {
			t.Fatalf("taskStreamDomainQuery(zero) error = %v", err)
		}
		if zeroDomainStream.AfterSequence != 0 {
			t.Fatalf("taskStreamDomainQuery(zero) = %#v, want after_sequence=0", zeroDomainStream)
		}
	})

	t.Run("Should parse and convert dashboard and inbox filters", func(t *testing.T) {
		t.Parallel()

		handlers := newExpandedTaskHandlers(func(_ context.Context, ref string) (workspacepkg.Workspace, error) {
			if ref != "alpha" {
				t.Fatalf("workspace ref = %q, want %q", ref, "alpha")
			}
			return workspacepkg.Workspace{ID: "ws-alpha"}, nil
		})

		dashboardRecorder := httptest.NewRecorder()
		dashboardCtx, _ := gin.CreateTestContext(dashboardRecorder)
		dashboardCtx.Request = httptest.NewRequestWithContext(
			context.Background(),
			http.MethodGet,
			"/observe/tasks/dashboard?scope=workspace&workspace=alpha&owner_kind=human&owner_ref=alice&network_channel=builders&origin_kind=http",
			http.NoBody,
		)

		dashboardQuery, err := ParseTaskDashboardQuery(dashboardCtx)
		if err != nil {
			t.Fatalf("ParseTaskDashboardQuery() error = %v", err)
		}
		if dashboardQuery.OriginKind != taskpkg.OriginKindHTTP || dashboardQuery.NetworkChannel != "builders" {
			t.Fatalf("ParseTaskDashboardQuery() = %#v", dashboardQuery)
		}

		domainDashboard, err := handlers.taskDashboardDomainQuery(context.Background(), dashboardQuery)
		if err != nil {
			t.Fatalf("taskDashboardDomainQuery() error = %v", err)
		}
		if domainDashboard.Scope != taskpkg.ScopeWorkspace ||
			domainDashboard.WorkspaceID != "ws-alpha" ||
			domainDashboard.OwnerKind != taskpkg.OwnerKindHuman ||
			domainDashboard.OwnerRef != "alice" ||
			domainDashboard.NetworkChannel != "builders" ||
			domainDashboard.OriginKind != taskpkg.OriginKindHTTP {
			t.Fatalf("taskDashboardDomainQuery() = %#v", domainDashboard)
		}

		inboxRecorder := httptest.NewRecorder()
		inboxCtx, _ := gin.CreateTestContext(inboxRecorder)
		inboxCtx.Request = httptest.NewRequestWithContext(
			context.Background(),
			http.MethodGet,
			"/observe/tasks/inbox?scope=workspace&workspace=alpha&owner_kind=human&owner_ref=alice&lane=approvals&unread=true&query=approve&limit=4",
			http.NoBody,
		)

		inboxQuery, err := ParseTaskInboxQuery(inboxCtx)
		if err != nil {
			t.Fatalf("ParseTaskInboxQuery() error = %v", err)
		}
		if inboxQuery.Lane != "approvals" || !inboxQuery.Unread || inboxQuery.Query != "approve" {
			t.Fatalf("ParseTaskInboxQuery() = %#v", inboxQuery)
		}

		domainInbox, err := handlers.taskInboxDomainQuery(context.Background(), inboxQuery)
		if err != nil {
			t.Fatalf("taskInboxDomainQuery() error = %v", err)
		}
		if domainInbox.Scope != taskpkg.ScopeWorkspace ||
			domainInbox.WorkspaceID != "ws-alpha" ||
			domainInbox.OwnerKind != taskpkg.OwnerKindHuman ||
			domainInbox.OwnerRef != "alice" ||
			domainInbox.Lane != observe.TaskInboxLaneApprovals ||
			!domainInbox.Unread ||
			domainInbox.Search != "approve" ||
			domainInbox.Limit != 4 {
			t.Fatalf("taskInboxDomainQuery() = %#v", domainInbox)
		}
	})
}

func TestExpandedTaskQueryValidationErrors(t *testing.T) {
	t.Parallel()

	t.Run("Should reject an invalid inbox lane", func(t *testing.T) {
		t.Parallel()

		handlers := newExpandedTaskHandlers(nil)

		recorder := httptest.NewRecorder()
		ginCtx, _ := gin.CreateTestContext(recorder)
		ginCtx.Request = httptest.NewRequestWithContext(
			context.Background(),
			http.MethodGet,
			"/observe/tasks/inbox?lane=bogus",
			http.NoBody,
		)

		_, err := ParseTaskInboxQuery(ginCtx)
		if err == nil {
			t.Fatal("ParseTaskInboxQuery(invalid lane) error = nil, want non-nil")
		}
		assertTaskValidationError(t, err, "lane")

		if _, err := handlers.taskInboxDomainQuery(
			context.Background(),
			contract.TaskInboxQuery{Lane: "bogus"},
		); err == nil {
			t.Fatal("taskInboxDomainQuery(invalid lane) error = nil, want non-nil")
		} else {
			assertTaskValidationError(t, err, "lane")
		}
	})

	t.Run("Should surface workspace lookup failures", func(t *testing.T) {
		t.Parallel()

		handlers := newExpandedTaskHandlers(func(context.Context, string) (workspacepkg.Workspace, error) {
			return workspacepkg.Workspace{}, workspacepkg.ErrWorkspaceNotFound
		})

		recorder := httptest.NewRecorder()
		ginCtx, _ := gin.CreateTestContext(recorder)
		ginCtx.Request = httptest.NewRequestWithContext(
			context.Background(),
			http.MethodGet,
			"/observe/tasks/inbox?scope=workspace&workspace=missing&lane=approvals",
			http.NoBody,
		)

		query, err := ParseTaskInboxQuery(ginCtx)
		if err != nil {
			t.Fatalf("ParseTaskInboxQuery() error = %v", err)
		}
		if _, err := handlers.taskInboxDomainQuery(
			context.Background(),
			query,
		); !errors.Is(
			err,
			workspacepkg.ErrWorkspaceNotFound,
		) {
			t.Fatalf(
				"taskInboxDomainQuery(workspace lookup) error = %v, want %v",
				err,
				workspacepkg.ErrWorkspaceNotFound,
			)
		}
	})
}

func TestTaskDraftFilteringAndNormalizationHelpers(t *testing.T) {
	t.Parallel()

	t.Run("Should filter draft tasks from default list queries", func(t *testing.T) {
		t.Parallel()

		tasks := []taskpkg.Summary{
			{ID: "task-draft", Status: taskpkg.TaskStatusDraft, Draft: true},
			{ID: "task-ready", Status: taskpkg.TaskStatusReady},
			{ID: "task-blocked", Status: taskpkg.TaskStatusBlocked},
		}

		filtered := filterTaskListDrafts(tasks, contract.TaskListQuery{Limit: 1})
		if len(filtered) != 1 || filtered[0].ID != "task-ready" {
			t.Fatalf("filterTaskListDrafts(default) = %#v", filtered)
		}

		withDrafts := filterTaskListDrafts(tasks, contract.TaskListQuery{IncludeDrafts: true})
		if len(withDrafts) != len(tasks) {
			t.Fatalf("filterTaskListDrafts(include drafts) len = %d, want %d", len(withDrafts), len(tasks))
		}

		withExplicitStatus := filterTaskListDrafts(tasks, contract.TaskListQuery{Status: taskpkg.TaskStatusDraft})
		if len(withExplicitStatus) != len(tasks) {
			t.Fatalf("filterTaskListDrafts(explicit status) len = %d, want %d", len(withExplicitStatus), len(tasks))
		}
	})

	t.Run("Should compensate for draft filtering when a limited page is under-filled", func(t *testing.T) {
		t.Parallel()

		allTasks := []taskpkg.Summary{
			{ID: "task-draft-1", Status: taskpkg.TaskStatusDraft, Draft: true},
			{ID: "task-draft-2", Status: taskpkg.TaskStatusDraft, Draft: true},
			{ID: "task-ready-1", Status: taskpkg.TaskStatusReady},
			{ID: "task-ready-2", Status: taskpkg.TaskStatusReady},
		}
		var limits []int

		filtered, err := listTasksWithDraftCompensation(
			context.Background(),
			taskpkg.Query{Limit: 2},
			contract.TaskListQuery{Limit: 2},
			func(_ context.Context, query taskpkg.Query) ([]taskpkg.Summary, error) {
				limits = append(limits, query.Limit)
				if query.Limit >= len(allTasks) {
					return allTasks, nil
				}
				return allTasks[:query.Limit], nil
			},
		)
		if err != nil {
			t.Fatalf("listTasksWithDraftCompensation() error = %v", err)
		}
		if got, want := limits, []int{2, 4}; !slices.Equal(got, want) {
			t.Fatalf("fetch limits = %#v, want %#v", got, want)
		}
		if got, want := len(filtered), 2; got != want {
			t.Fatalf("len(filtered) = %d, want %d", got, want)
		}
		if filtered[0].ID != "task-ready-1" || filtered[1].ID != "task-ready-2" {
			t.Fatalf("filtered = %#v, want ready tasks only", filtered)
		}
	})

	t.Run("Should cap draft compensation overfetch", func(t *testing.T) {
		t.Parallel()

		var limits []int
		_, err := listTasksWithDraftCompensation(
			context.Background(),
			taskpkg.Query{Limit: 200},
			contract.TaskListQuery{Limit: 200},
			func(_ context.Context, query taskpkg.Query) ([]taskpkg.Summary, error) {
				limits = append(limits, query.Limit)
				tasks := make([]taskpkg.Summary, query.Limit)
				for idx := range tasks {
					tasks[idx] = taskpkg.Summary{
						ID:     "task-draft",
						Status: taskpkg.TaskStatusDraft,
						Draft:  true,
					}
				}
				return tasks, nil
			},
		)
		if err != nil {
			t.Fatalf("listTasksWithDraftCompensation() error = %v", err)
		}
		if len(limits) == 0 {
			t.Fatal("fetch limits = empty, want bounded compensation attempts")
		}
		if got := limits[len(limits)-1]; got != taskDraftOverfetchMaxLimit {
			t.Fatalf("last fetch limit = %d, want %d", got, taskDraftOverfetchMaxLimit)
		}
	})

	t.Run("Should normalize optional pointer helpers", func(t *testing.T) {
		t.Parallel()

		if got := normalizePriorityPtr(nil); got != nil {
			t.Fatalf("normalizePriorityPtr(nil) = %#v, want nil", got)
		}
		if got := normalizeApprovalPolicyPtr(nil); got != nil {
			t.Fatalf("normalizeApprovalPolicyPtr(nil) = %#v, want nil", got)
		}

		priority := taskpkg.Priority(" high ")
		policy := taskpkg.ApprovalPolicy(" manual ")

		normalizedPriority := normalizePriorityPtr(&priority)
		if normalizedPriority == nil || *normalizedPriority != taskpkg.PriorityHigh {
			t.Fatalf("normalizePriorityPtr() = %#v", normalizedPriority)
		}

		normalizedPolicy := normalizeApprovalPolicyPtr(&policy)
		if normalizedPolicy == nil || *normalizedPolicy != taskpkg.ApprovalPolicyManual {
			t.Fatalf("normalizeApprovalPolicyPtr() = %#v", normalizedPolicy)
		}
	})
}
