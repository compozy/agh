package core_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/api/testutil"
	automationpkg "github.com/pedronauck/agh/internal/automation"
)

func TestAutomationEndpointsAdditionalCoverage(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)
	job := automationpkg.Job{
		ID:        "job-1",
		Scope:     automationpkg.AutomationScopeWorkspace,
		Name:      "nightly-review",
		AgentName: "coder",
		Prompt:    "review repo",
		Enabled:   true,
		Retry:     automationpkg.DefaultRetryConfig(),
		FireLimit: automationpkg.DefaultFireLimitConfig(),
		Source:    automationpkg.JobSourceDynamic,
		CreatedAt: now,
		UpdatedAt: now,
	}
	trigger := automationpkg.Trigger{
		ID:        "trigger-1",
		Scope:     automationpkg.AutomationScopeWorkspace,
		Name:      "deploy-review",
		AgentName: "coder",
		Prompt:    "review deploy",
		Event:     "webhook",
		Enabled:   true,
		Retry:     automationpkg.DefaultRetryConfig(),
		FireLimit: automationpkg.DefaultFireLimitConfig(),
		Source:    automationpkg.JobSourceDynamic,
		CreatedAt: now,
		UpdatedAt: now,
	}
	run := automationpkg.Run{
		ID:        "run-1",
		JobID:     job.ID,
		TriggerID: trigger.ID,
		Status:    automationpkg.RunCompleted,
		Attempt:   1,
		StartedAt: &now,
		EndedAt:   &now,
	}

	automation := testutil.StubAutomationManager{
		ListTriggersFn: func(context.Context, automationpkg.TriggerListQuery) ([]automationpkg.Trigger, error) {
			return []automationpkg.Trigger{trigger}, nil
		},
		GetJobFn: func(context.Context, string) (automationpkg.Job, error) {
			return job, nil
		},
		DeleteJobFn: func(context.Context, string) error {
			return nil
		},
		GetTriggerFn: func(context.Context, string) (automationpkg.Trigger, error) {
			return trigger, nil
		},
		DeleteTriggerFn: func(context.Context, string) error {
			return nil
		},
		ListRunsFn: func(_ context.Context, query automationpkg.RunQuery) ([]automationpkg.Run, error) {
			switch {
			case query.JobID == job.ID:
				return []automationpkg.Run{run}, nil
			case query.TriggerID == trigger.ID:
				return []automationpkg.Run{run}, nil
			default:
				return []automationpkg.Run{run}, nil
			}
		},
		GetRunFn: func(context.Context, string) (automationpkg.Run, error) {
			return run, nil
		},
	}

	fixture := newHandlerFixtureWithAutomation(t, testutil.StubSessionManager{}, testutil.StubObserver{}, automation, testutil.StubWorkspaceService{}, nil, nil)

	for _, request := range []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/automation/jobs/job-1"},
		{method: http.MethodDelete, path: "/automation/jobs/job-1"},
		{method: http.MethodGet, path: "/automation/triggers"},
		{method: http.MethodGet, path: "/automation/triggers/trigger-1"},
		{method: http.MethodDelete, path: "/automation/triggers/trigger-1"},
		{method: http.MethodGet, path: "/automation/jobs/job-1/runs"},
		{method: http.MethodGet, path: "/automation/triggers/trigger-1/runs"},
		{method: http.MethodGet, path: "/automation/runs?limit=1"},
		{method: http.MethodGet, path: "/automation/runs/run-1"},
	} {
		request := request
		t.Run(request.method+" "+request.path, func(t *testing.T) {
			resp := performRequest(t, fixture.Engine, request.method, request.path, nil)
			if request.method == http.MethodDelete {
				if resp.Code != http.StatusNoContent {
					t.Fatalf("%s %s status = %d, want %d; body=%s", request.method, request.path, resp.Code, http.StatusNoContent, resp.Body.String())
				}
				return
			}
			if resp.Code != http.StatusOK {
				t.Fatalf("%s %s status = %d, want %d; body=%s", request.method, request.path, resp.Code, http.StatusOK, resp.Body.String())
			}
		})
	}
}
