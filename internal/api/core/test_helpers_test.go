package core_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/api/core"
	"github.com/pedronauck/agh/internal/api/testutil"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/memory"
)

type stubDreamTrigger struct {
	Triggered bool
	Reason    string
	Err       error
	Last      time.Time
	LastErr   error
	EnabledFn bool
	Calls     int
	Workspace string
}

func (s *stubDreamTrigger) Trigger(_ context.Context, workspace string) (bool, string, error) {
	s.Calls++
	s.Workspace = workspace
	return s.Triggered, s.Reason, s.Err
}

func (s *stubDreamTrigger) LastConsolidatedAt() (time.Time, error) {
	return s.Last, s.LastErr
}

func (s *stubDreamTrigger) Enabled() bool {
	return s.EnabledFn
}

type handlerFixture struct {
	Handlers  *core.BaseHandlers
	Engine    *gin.Engine
	HomePaths aghconfig.HomePaths
}

func testConfigWithDisabledNetwork(homePaths aghconfig.HomePaths) aghconfig.Config {
	cfg := aghconfig.DefaultWithHome(homePaths)
	cfg.Network.Enabled = false
	return cfg
}

func newHandlerFixture(
	t *testing.T,
	manager testutil.StubSessionManager,
	observer testutil.StubObserver,
	workspaces testutil.StubWorkspaceService,
	store *memory.Store,
	dream core.DreamTrigger,
) handlerFixture {
	return newHandlerFixtureWithAutomationAndTasks(
		t,
		manager,
		observer,
		testutil.StubAutomationManager{},
		testutil.StubTaskManager{},
		workspaces,
		store,
		dream,
	)
}

func newHandlerFixtureWithAutomation(
	t *testing.T,
	manager testutil.StubSessionManager,
	observer testutil.StubObserver,
	automation testutil.StubAutomationManager,
	workspaces testutil.StubWorkspaceService,
	store *memory.Store,
	dream core.DreamTrigger,
) handlerFixture {
	return newHandlerFixtureWithAutomationAndTasks(
		t,
		manager,
		observer,
		automation,
		testutil.StubTaskManager{},
		workspaces,
		store,
		dream,
	)
}

func newHandlerFixtureWithTasks(
	t *testing.T,
	manager testutil.StubSessionManager,
	observer testutil.StubObserver,
	tasks testutil.StubTaskManager,
	workspaces testutil.StubWorkspaceService,
	store *memory.Store,
	dream core.DreamTrigger,
) handlerFixture {
	return newHandlerFixtureWithAutomationAndTasks(
		t,
		manager,
		observer,
		testutil.StubAutomationManager{},
		tasks,
		workspaces,
		store,
		dream,
	)
}

func newHandlerFixtureWithAutomationAndTasks(
	t *testing.T,
	manager testutil.StubSessionManager,
	observer testutil.StubObserver,
	automation testutil.StubAutomationManager,
	tasks testutil.StubTaskManager,
	workspaces testutil.StubWorkspaceService,
	store *memory.Store,
	dream core.DreamTrigger,
) handlerFixture {
	t.Helper()

	gin.SetMode(gin.TestMode)
	homePaths := testutil.NewTestHomePaths(t)
	cfg := testConfigWithDisabledNetwork(homePaths)
	cfg.HTTP.Host = "127.0.0.1"
	cfg.HTTP.Port = 2123
	cfg.Daemon.Socket = "/tmp/api-core-test.sock"

	handlers := core.NewBaseHandlers(&core.BaseHandlerConfig{
		TransportName:                "api-core-test",
		MaskInternalErrors:           false,
		IncludeSessionWorkspaceInSSE: true,
		Sessions:                     manager,
		Observer:                     observer,
		Automation:                   automation,
		Tasks:                        tasks,
		Workspaces:                   workspaces,
		MemoryStore:                  store,
		DreamTrigger:                 dream,
		HomePaths:                    homePaths,
		Config:                       cfg,
		Logger:                       testutil.DiscardLogger(),
		StartedAt:                    time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
		Now: func() time.Time {
			return time.Date(2026, 4, 3, 12, 0, 1, 0, time.UTC)
		},
		PollInterval: 5 * time.Millisecond,
		HTTPPort:     cfg.HTTP.Port,
	})

	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.GET("/sessions", handlers.ListSessions)
	engine.POST("/sessions", handlers.CreateSession)
	engine.GET("/sessions/:id", handlers.GetSession)
	engine.DELETE("/sessions/:id", handlers.StopSession)
	engine.POST("/sessions/:id/resume", handlers.ResumeSession)
	engine.GET("/sessions/:id/events", handlers.SessionEvents)
	engine.GET("/sessions/:id/history", handlers.SessionHistory)
	engine.GET("/sessions/:id/transcript", handlers.SessionTranscript)
	engine.GET("/sessions/:id/stream", handlers.StreamSession)
	engine.GET("/agents", handlers.ListAgents)
	engine.GET("/agents/:name", handlers.GetAgent)
	engine.GET("/hooks/catalog", handlers.HookCatalog)
	engine.GET("/hooks/runs", handlers.HookRuns)
	engine.GET("/hooks/events", handlers.HookEvents)
	engine.GET("/observe/events", handlers.ObserveEvents)
	engine.GET("/observe/events/stream", handlers.StreamObserveEvents)
	engine.GET("/observe/health", handlers.Health)
	engine.GET("/automation/jobs", handlers.ListAutomationJobs)
	engine.POST("/automation/jobs", handlers.CreateAutomationJob)
	engine.GET("/automation/jobs/:id", handlers.GetAutomationJob)
	engine.PATCH("/automation/jobs/:id", handlers.UpdateAutomationJob)
	engine.DELETE("/automation/jobs/:id", handlers.DeleteAutomationJob)
	engine.POST("/automation/jobs/:id/trigger", handlers.TriggerAutomationJob)
	engine.GET("/automation/jobs/:id/runs", handlers.AutomationJobRuns)
	engine.GET("/automation/triggers", handlers.ListAutomationTriggers)
	engine.POST("/automation/triggers", handlers.CreateAutomationTrigger)
	engine.GET("/automation/triggers/:id", handlers.GetAutomationTrigger)
	engine.PATCH("/automation/triggers/:id", handlers.UpdateAutomationTrigger)
	engine.DELETE("/automation/triggers/:id", handlers.DeleteAutomationTrigger)
	engine.GET("/automation/triggers/:id/runs", handlers.AutomationTriggerRuns)
	engine.GET("/automation/runs", handlers.ListAutomationRuns)
	engine.GET("/automation/runs/:id", handlers.GetAutomationRun)
	engine.POST("/webhooks/global/:endpoint", handlers.DeliverGlobalWebhook)
	engine.POST("/webhooks/workspaces/:workspace_id/:endpoint", handlers.DeliverWorkspaceWebhook)
	engine.GET("/daemon/status", handlers.DaemonStatus)
	engine.GET("/network/status", handlers.NetworkStatus)
	engine.GET("/network/peers", handlers.NetworkPeers)
	engine.GET("/network/peers/:peer_id", handlers.NetworkPeer)
	engine.GET("/network/channels", handlers.NetworkChannels)
	engine.POST("/network/channels", handlers.CreateNetworkChannel)
	engine.GET("/network/channels/:channel", handlers.NetworkChannel)
	engine.GET("/network/channels/:channel/messages", handlers.NetworkChannelMessages)
	engine.POST("/network/send", handlers.NetworkSend)
	engine.GET("/network/inbox", handlers.NetworkInbox)
	engine.GET("/tasks", handlers.ListTasks)
	engine.POST("/tasks", handlers.CreateTask)
	engine.GET("/tasks/:id", handlers.GetTask)
	engine.PATCH("/tasks/:id", handlers.UpdateTask)
	engine.POST("/tasks/:id/publish", handlers.PublishTask)
	engine.POST("/tasks/:id/cancel", handlers.CancelTask)
	engine.POST("/tasks/:id/children", handlers.CreateChildTask)
	engine.POST("/tasks/:id/dependencies", handlers.AddTaskDependency)
	engine.DELETE("/tasks/:id/dependencies/:depends_on_id", handlers.RemoveTaskDependency)
	engine.GET("/tasks/:id/runs", handlers.ListTaskRuns)
	engine.GET("/tasks/:id/timeline", handlers.TaskTimeline)
	engine.GET("/tasks/:id/stream", handlers.StreamTask)
	engine.GET("/tasks/:id/tree", handlers.TaskTree)
	engine.POST("/tasks/:id/approve", handlers.ApproveTask)
	engine.POST("/tasks/:id/reject", handlers.RejectTask)
	engine.POST("/tasks/:id/triage/read", handlers.MarkTaskRead)
	engine.POST("/tasks/:id/triage/archive", handlers.ArchiveTask)
	engine.POST("/tasks/:id/triage/dismiss", handlers.DismissTask)
	engine.POST("/tasks/:id/runs", handlers.EnqueueTaskRun)
	engine.GET("/task-runs/:id", handlers.GetTaskRun)
	engine.POST("/task-runs/:id/claim", handlers.ClaimTaskRun)
	engine.POST("/task-runs/:id/start", handlers.StartTaskRun)
	engine.POST("/task-runs/:id/attach-session", handlers.AttachTaskRunSession)
	engine.POST("/task-runs/:id/complete", handlers.CompleteTaskRun)
	engine.POST("/task-runs/:id/fail", handlers.FailTaskRun)
	engine.POST("/task-runs/:id/cancel", handlers.CancelTaskRun)
	engine.GET("/observe/tasks/dashboard", handlers.TaskDashboard)
	engine.GET("/observe/tasks/inbox", handlers.TaskInbox)
	engine.GET("/memory", handlers.ListMemory)
	engine.GET("/memory/:filename", handlers.ReadMemory)
	engine.PUT("/memory/:filename", handlers.WriteMemory)
	engine.DELETE("/memory/:filename", handlers.DeleteMemory)
	engine.POST("/memory/consolidate", handlers.ConsolidateMemory)
	engine.POST("/workspaces", handlers.CreateWorkspace)
	engine.GET("/workspaces", handlers.ListWorkspaces)
	engine.GET("/workspaces/:id", handlers.GetWorkspace)
	engine.PATCH("/workspaces/:id", handlers.UpdateWorkspace)
	engine.DELETE("/workspaces/:id", handlers.DeleteWorkspace)
	engine.POST("/workspaces/resolve", handlers.ResolveWorkspace)

	return handlerFixture{
		Handlers:  handlers,
		Engine:    engine,
		HomePaths: homePaths,
	}
}

func performRequest(t *testing.T, engine http.Handler, method, path string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	return testutil.PerformRequest(t, engine, method, path, body)
}
