package daemon

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/session"
	taskpkg "github.com/pedronauck/agh/internal/task"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func TestReviewRouterRoutesRunReviewRequests(t *testing.T) {
	t.Parallel()

	t.Run("Should bind an active eligible reviewer and exclude the original worker", func(t *testing.T) {
		t.Parallel()

		tasks := &reviewRouterTasksStub{
			profile: taskpkg.ExecutionProfile{
				TaskID: "task-1",
				Review: taskpkg.ReviewProfile{
					AgentName:            "reviewer",
					AllowedChannelIDs:    []string{"reviews"},
					RequiredCapabilities: []string{"review-pr"},
				},
			},
		}
		sessions := &coordinatorRuntimeSessions{
			infos: []*session.Info{
				reviewRouterSessionInfo("sess-worker", "worker", "reviews"),
				reviewRouterSessionInfo("sess-reviewer", "reviewer", "reviews"),
			},
		}
		router := newReviewRouterForTest(
			t,
			tasks,
			reviewRouterStoreForTest(),
			sessions,
			reviewRouterAgentResolverStub{
				"reviewer": reviewRouterAgentDef("reviewer", "review-pr"),
				"worker":   reviewRouterAgentDef("worker", "build"),
			},
		)

		notification := reviewRouterNotificationForTest()
		router.OnRunReviewRequested(context.Background(), &notification)

		if got, want := len(tasks.binds), 1; got != want {
			t.Fatalf("bind calls = %d, want %d", got, want)
		}
		bind := tasks.binds[0]
		if got, want := bind.SessionID, "sess-reviewer"; got != want {
			t.Fatalf("BindRunReviewSession.SessionID = %q, want %q", got, want)
		}
		if got, want := bind.ReviewerPeerID, "reviewer.sess-reviewer"; got != want {
			t.Fatalf("BindRunReviewSession.ReviewerPeerID = %q, want %q", got, want)
		}
		if len(tasks.records) != 0 {
			t.Fatalf("RecordRunReview calls = %#v, want none", tasks.records)
		}
	})

	t.Run("Should create and bind a reviewer session when no active reviewer exists", func(t *testing.T) {
		t.Parallel()

		tasks := &reviewRouterTasksStub{
			profile: taskpkg.ExecutionProfile{
				TaskID: "task-1",
				Review: taskpkg.ReviewProfile{
					AgentName:           "reviewer",
					PreferredChannelIDs: []string{"reviews"},
				},
			},
		}
		sessions := &coordinatorRuntimeSessions{}
		router := newReviewRouterForTest(
			t,
			tasks,
			reviewRouterStoreForTest(),
			sessions,
			reviewRouterAgentResolverStub{"reviewer": reviewRouterAgentDef("reviewer")},
		)

		notification := reviewRouterNotificationForTest()
		router.OnRunReviewRequested(context.Background(), &notification)

		if got, want := sessions.createCount(), 1; got != want {
			t.Fatalf("session create calls = %d, want %d", got, want)
		}
		create := sessions.createCall(0)
		if create.Type != session.SessionTypeSystem {
			t.Fatalf("CreateOpts.Type = %q, want system", create.Type)
		}
		if create.AgentName != "reviewer" || create.Channel != "reviews" {
			t.Fatalf("CreateOpts agent/channel = %q/%q, want reviewer/reviews", create.AgentName, create.Channel)
		}
		if !strings.Contains(create.PromptOverlay, "agh-task-reviewer") ||
			!strings.Contains(create.PromptOverlay, "submit_run_review") {
			t.Fatalf("CreateOpts.PromptOverlay = %q, want reviewer instructions", create.PromptOverlay)
		}
		if got, want := len(tasks.binds), 1; got != want {
			t.Fatalf("bind calls = %d, want %d", got, want)
		}
		if got, want := tasks.binds[0].ReviewerAgentName, "reviewer"; got != want {
			t.Fatalf("bind reviewer agent = %q, want %q", got, want)
		}
	})

	t.Run("Should include task context bundle overlay when creating a reviewer session", func(t *testing.T) {
		t.Parallel()

		tasks := &reviewRouterTasksStub{
			profile: taskpkg.ExecutionProfile{
				TaskID: "task-1",
				Review: taskpkg.ReviewProfile{
					AgentName:           "reviewer",
					PreferredChannelIDs: []string{"reviews"},
				},
			},
		}
		sessions := &coordinatorRuntimeSessions{}
		router := newReviewRouterForTest(
			t,
			tasks,
			reviewRouterStoreForTest(),
			sessions,
			reviewRouterAgentResolverStub{"reviewer": reviewRouterAgentDef("reviewer")},
		)
		overlay := &taskContextOverlayStub{overlay: "review task context bundle"}
		router.contextOverlay = overlay

		notification := reviewRouterNotificationForTest()
		router.OnRunReviewRequested(context.Background(), &notification)

		create := sessions.createCall(0)
		if !strings.Contains(create.PromptOverlay, "review task context bundle") ||
			!strings.Contains(create.PromptOverlay, "submit_run_review") {
			t.Fatalf("CreateOpts.PromptOverlay = %q, want task bundle plus reviewer instructions", create.PromptOverlay)
		}
		if len(overlay.calls) != 1 ||
			overlay.calls[0].taskID != "task-1" ||
			overlay.calls[0].runID != "run-1" {
			t.Fatalf("overlay calls = %#v, want reviewed task/run", overlay.calls)
		}
	})

	t.Run(
		"Should record a deterministic no-route diagnostic instead of binding the original worker",
		func(t *testing.T) {
			t.Parallel()

			tasks := &reviewRouterTasksStub{
				profile: taskpkg.ExecutionProfile{
					TaskID: "task-1",
					Review: taskpkg.ReviewProfile{
						AgentName: "worker",
					},
				},
			}
			sessions := &coordinatorRuntimeSessions{
				infos: []*session.Info{
					reviewRouterSessionInfo("sess-worker", "worker", "reviews"),
				},
			}
			router := newReviewRouterForTest(
				t,
				tasks,
				reviewRouterStoreForTest(),
				sessions,
				reviewRouterAgentResolverStub{"worker": reviewRouterAgentDef("worker")},
			)

			notification := reviewRouterNotificationForTest()
			router.OnRunReviewRequested(context.Background(), &notification)

			if len(tasks.binds) != 0 {
				t.Fatalf("BindRunReviewSession calls = %#v, want none", tasks.binds)
			}
			if got, want := len(tasks.records), 1; got != want {
				t.Fatalf("RecordRunReview calls = %d, want %d", got, want)
			}
			record := tasks.records[0]
			if got, want := record.Verdict.Outcome, taskpkg.RunReviewOutcomeBlocked; got != want {
				t.Fatalf("RecordRunReview outcome = %q, want %q", got, want)
			}
			if !strings.HasPrefix(record.Verdict.DeliveryID, reviewRouterNoRouteDeliveryPrefix) {
				t.Fatalf(
					"RecordRunReview delivery_id = %q, want review-router no-route prefix",
					record.Verdict.DeliveryID,
				)
			}
			if !strings.Contains(record.Verdict.Reason, "review profile agent selectors exclude") {
				t.Fatalf("RecordRunReview reason = %q, want deterministic selector diagnostic", record.Verdict.Reason)
			}
		},
	)
}

func newReviewRouterForTest(
	t *testing.T,
	tasks *reviewRouterTasksStub,
	store *reviewRouterStoreStub,
	sessions *coordinatorRuntimeSessions,
	agents reviewRouterAgentResolverStub,
) *reviewRouter {
	t.Helper()
	router, err := newReviewRouter(
		tasks,
		store,
		sessions,
		nil,
		agents,
		discardLogger(),
		func() time.Time { return time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC) },
	)
	if err != nil {
		t.Fatalf("newReviewRouter() error = %v", err)
	}
	return router
}

func reviewRouterStoreForTest() *reviewRouterStoreStub {
	return &reviewRouterStoreStub{
		taskRecord: taskpkg.Task{
			ID:          "task-1",
			Scope:       taskpkg.ScopeWorkspace,
			WorkspaceID: "ws-1",
			Status:      taskpkg.TaskStatusReady,
			Title:       "Review routed task",
		},
		run: taskpkg.Run{
			ID:                    "run-1",
			TaskID:                "task-1",
			Status:                taskpkg.TaskRunStatusCompleted,
			SessionID:             "sess-worker",
			CoordinationChannelID: "reviews",
			NetworkChannel:        "reviews",
		},
	}
}

func reviewRouterNotificationForTest() taskpkg.RunReviewRequestedNotification {
	return taskpkg.RunReviewRequestedNotification{
		Review: taskpkg.RunReview{
			ReviewID:    "review-1",
			TaskID:      "task-1",
			RunID:       "run-1",
			Policy:      taskpkg.ReviewPolicyAlways,
			ReviewRound: 1,
			Attempt:     1,
			Status:      taskpkg.RunReviewStatusRequested,
		},
	}
}

func reviewRouterSessionInfo(id string, agent string, channel string) *session.Info {
	return &session.Info{
		ID:          id,
		AgentName:   agent,
		WorkspaceID: "ws-1",
		Workspace:   "ws-1",
		Channel:     channel,
		Type:        session.SessionTypeSystem,
		State:       session.StateActive,
	}
}

func reviewRouterAgentDef(name string, capabilities ...string) aghconfig.AgentDef {
	agent := aghconfig.AgentDef{Name: name, Provider: "codex", Model: "gpt-5"}
	if len(capabilities) == 0 {
		return agent
	}
	agent.Capabilities = &aghconfig.CapabilityCatalog{
		Capabilities: make([]aghconfig.CapabilityDef, 0, len(capabilities)),
	}
	for _, capability := range capabilities {
		agent.Capabilities.Capabilities = append(agent.Capabilities.Capabilities, aghconfig.CapabilityDef{
			ID:      capability,
			Summary: capability,
			Outcome: capability,
		})
	}
	return agent
}

type reviewRouterStoreStub struct {
	taskRecord taskpkg.Task
	run        taskpkg.Run
}

func (s *reviewRouterStoreStub) GetTask(_ context.Context, id string) (taskpkg.Task, error) {
	if id != s.taskRecord.ID {
		return taskpkg.Task{}, taskpkg.ErrTaskNotFound
	}
	return s.taskRecord, nil
}

func (s *reviewRouterStoreStub) GetTaskRun(_ context.Context, id string) (taskpkg.Run, error) {
	if id != s.run.ID {
		return taskpkg.Run{}, taskpkg.ErrTaskRunNotFound
	}
	return s.run, nil
}

type reviewRouterTasksStub struct {
	mu      sync.Mutex
	profile taskpkg.ExecutionProfile
	binds   []taskpkg.BindRunReviewSessionRequest
	records []taskpkg.RecordRunReviewRequest
}

func (s *reviewRouterTasksStub) GetExecutionProfile(
	context.Context,
	string,
	taskpkg.ActorContext,
) (taskpkg.ExecutionProfile, error) {
	return s.profile, nil
}

func (s *reviewRouterTasksStub) BindRunReviewSession(
	_ context.Context,
	req taskpkg.BindRunReviewSessionRequest,
	_ taskpkg.ActorContext,
) (taskpkg.RunReviewBinding, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.binds = append(s.binds, req)
	return taskpkg.RunReviewBinding{SessionID: req.SessionID}, nil
}

func (s *reviewRouterTasksStub) RecordRunReview(
	_ context.Context,
	req taskpkg.RecordRunReviewRequest,
	_ taskpkg.ActorContext,
) (taskpkg.RunReviewResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = append(s.records, req)
	return taskpkg.RunReviewResult{
		Review: taskpkg.RunReview{
			ReviewID: req.ReviewID,
			RunID:    req.RunID,
			Status:   taskpkg.RunReviewStatusRecorded,
			Outcome:  req.Verdict.Outcome,
		},
	}, nil
}

type reviewRouterAgentResolverStub map[string]aghconfig.AgentDef

func (s reviewRouterAgentResolverStub) ResolveAgent(
	name string,
	_ *workspacepkg.ResolvedWorkspace,
) (aghconfig.AgentDef, error) {
	agent, ok := s[strings.TrimSpace(name)]
	if !ok {
		return aghconfig.AgentDef{}, workspacepkg.ErrAgentNotAvailable
	}
	return agent, nil
}
