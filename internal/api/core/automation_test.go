package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/api/contract"
	automationpkg "github.com/pedronauck/agh/internal/automation"
	aghconfig "github.com/pedronauck/agh/internal/config"
)

func TestUpdateAutomationJobConfigBackedRejectsDefinitionEditsButAllowsEnabledToggle(t *testing.T) {
	t.Parallel()

	current := automationpkg.Job{
		ID:        "job-config",
		Scope:     automationpkg.AutomationScopeGlobal,
		Name:      "config-job",
		AgentName: "coder",
		Prompt:    "do work",
		Schedule: &automationpkg.ScheduleSpec{
			Mode:     automationpkg.ScheduleModeEvery,
			Interval: "1h",
		},
		Enabled:   true,
		Retry:     automationpkg.DefaultRetryConfig(),
		FireLimit: automationpkg.DefaultFireLimitConfig(),
		Source:    automationpkg.JobSourceConfig,
	}

	setCalled := false
	router := newAutomationCoreTestRouter(t, stubAutomationManager{
		GetJobFn: func(_ context.Context, id string) (automationpkg.Job, error) {
			if id != current.ID {
				t.Fatalf("GetJob() id = %q, want %q", id, current.ID)
			}
			return current, nil
		},
		SetJobEnabledFn: func(_ context.Context, id string, enabled bool) (automationpkg.Job, error) {
			setCalled = true
			if id != current.ID || enabled {
				t.Fatalf("SetJobEnabled() = (%q, %v), want (%q, false)", id, enabled, current.ID)
			}
			next := current
			next.Enabled = false
			return next, nil
		},
		UpdateJobFn: func(context.Context, automationpkg.Job) (automationpkg.Job, error) {
			t.Fatal("UpdateJob() should not be called for config-backed job validation")
			return automationpkg.Job{}, nil
		},
	})

	invalid := performAutomationCoreRequest(t, router, http.MethodPatch, "/automation/jobs/"+current.ID, []byte(`{"prompt":"changed"}`), nil)
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid status = %d, want %d; body=%s", invalid.Code, http.StatusBadRequest, invalid.Body.String())
	}
	if setCalled {
		t.Fatal("SetJobEnabled() called for rejected config-backed definition edit")
	}

	valid := performAutomationCoreRequest(t, router, http.MethodPatch, "/automation/jobs/"+current.ID, []byte(`{"enabled":false}`), nil)
	if valid.Code != http.StatusOK {
		t.Fatalf("valid status = %d, want %d; body=%s", valid.Code, http.StatusOK, valid.Body.String())
	}
	if !setCalled {
		t.Fatal("SetJobEnabled() not called for enabled-only config-backed update")
	}

	var response struct {
		Job struct {
			Enabled bool `json:"enabled"`
		} `json:"job"`
	}
	decodeAutomationCoreJSON(t, valid, &response)
	if response.Job.Enabled {
		t.Fatalf("response.job.enabled = %v, want false", response.Job.Enabled)
	}
}

func TestUpdateAutomationTriggerConfigBackedRejectsDefinitionEditsButAllowsEnabledToggle(t *testing.T) {
	t.Parallel()

	current := automationpkg.Trigger{
		ID:        "trigger-config",
		Scope:     automationpkg.AutomationScopeGlobal,
		Name:      "config-trigger",
		AgentName: "coder",
		Prompt:    `review {{ index .Data "session_id" }}`,
		Event:     "session.stopped",
		Enabled:   true,
		Retry:     automationpkg.DefaultRetryConfig(),
		FireLimit: automationpkg.DefaultFireLimitConfig(),
		Source:    automationpkg.JobSourceConfig,
	}

	setCalled := false
	router := newAutomationCoreTestRouter(t, stubAutomationManager{
		GetTriggerFn: func(_ context.Context, id string) (automationpkg.Trigger, error) {
			if id != current.ID {
				t.Fatalf("GetTrigger() id = %q, want %q", id, current.ID)
			}
			return current, nil
		},
		SetTriggerEnabledFn: func(_ context.Context, id string, enabled bool) (automationpkg.Trigger, error) {
			setCalled = true
			if id != current.ID || enabled {
				t.Fatalf("SetTriggerEnabled() = (%q, %v), want (%q, false)", id, enabled, current.ID)
			}
			next := current
			next.Enabled = false
			return next, nil
		},
		UpdateTriggerFn: func(context.Context, automationpkg.Trigger, *string) (automationpkg.Trigger, error) {
			t.Fatal("UpdateTrigger() should not be called for config-backed trigger validation")
			return automationpkg.Trigger{}, nil
		},
	})

	invalid := performAutomationCoreRequest(t, router, http.MethodPatch, "/automation/triggers/"+current.ID, []byte(`{"prompt":"changed"}`), nil)
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid status = %d, want %d; body=%s", invalid.Code, http.StatusBadRequest, invalid.Body.String())
	}
	if setCalled {
		t.Fatal("SetTriggerEnabled() called for rejected config-backed definition edit")
	}

	valid := performAutomationCoreRequest(t, router, http.MethodPatch, "/automation/triggers/"+current.ID, []byte(`{"enabled":false}`), nil)
	if valid.Code != http.StatusOK {
		t.Fatalf("valid status = %d, want %d; body=%s", valid.Code, http.StatusOK, valid.Body.String())
	}
	if !setCalled {
		t.Fatal("SetTriggerEnabled() not called for enabled-only config-backed update")
	}
}

func TestWebhookRequestValidationRejectsInvalidScopeAndMalformedEndpointBeforeDispatch(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/global/not-used", nil)
	req.Header.Set(WebhookTimestampHeader, time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC).Format(time.RFC3339))
	req.Header.Set(WebhookSignatureHeader, "sha256=deadbeef")
	req.Header.Set(WebhookDeliveryIDHeader, "delivery-validation")
	ctx.Request = req
	ctx.Params = gin.Params{{Key: "endpoint", Value: "deploy-review--wbh_test"}}

	if _, err := webhookRequestFromHTTP(ctx, automationpkg.AutomationScope("bogus")); !errors.Is(err, ErrAutomationValidation) {
		t.Fatalf("webhookRequestFromHTTP(invalid scope) error = %v, want ErrAutomationValidation", err)
	}

	called := false
	router := newAutomationCoreTestRouter(t, stubAutomationManager{
		HandleWebhookFn: func(context.Context, automationpkg.WebhookRequest) (automationpkg.TriggerResult, error) {
			called = true
			return automationpkg.TriggerResult{}, nil
		},
	})

	response := performAutomationCoreRequest(t, router, http.MethodPost, "/webhooks/global/not-an-endpoint", []byte(`{"payload":"deploy"}`), map[string]string{
		WebhookTimestampHeader:  time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC).Format(time.RFC3339),
		WebhookSignatureHeader:  "sha256=deadbeef",
		WebhookDeliveryIDHeader: "delivery-malformed-endpoint",
	})
	if response.Code != http.StatusBadRequest {
		t.Fatalf("malformed endpoint status = %d, want %d; body=%s", response.Code, http.StatusBadRequest, response.Body.String())
	}
	if called {
		t.Fatal("HandleWebhook() called for malformed endpoint path")
	}
}

func TestAutomationDynamicHandlersRoundTripAndHelperCoverage(t *testing.T) {
	t.Parallel()

	nextRun := time.Date(2026, 4, 11, 13, 0, 0, 0, time.UTC)
	startedAt := time.Date(2026, 4, 11, 12, 15, 0, 0, time.UTC)
	endedAt := time.Date(2026, 4, 11, 12, 16, 0, 0, time.UTC)

	job := automationpkg.Job{
		ID:        "job-dynamic",
		Scope:     automationpkg.AutomationScopeGlobal,
		Name:      "nightly-review",
		AgentName: "coder",
		Prompt:    "review repo",
		Schedule: &automationpkg.ScheduleSpec{
			Mode:     automationpkg.ScheduleModeEvery,
			Interval: "1h",
		},
		Enabled:   true,
		Retry:     automationpkg.DefaultRetryConfig(),
		FireLimit: automationpkg.DefaultFireLimitConfig(),
		Source:    automationpkg.JobSourceDynamic,
	}
	trigger := automationpkg.Trigger{
		ID:           "trigger-dynamic",
		Scope:        automationpkg.AutomationScopeWorkspace,
		Name:         "deploy-review",
		AgentName:    "coder",
		WorkspaceID:  "ws-alpha",
		Prompt:       `review {{ index .Data "payload" }}`,
		Event:        "webhook",
		Filter:       map[string]string{"data.branch": "main"},
		Enabled:      true,
		Retry:        automationpkg.DefaultRetryConfig(),
		FireLimit:    automationpkg.DefaultFireLimitConfig(),
		Source:       automationpkg.JobSourceDynamic,
		WebhookID:    "wbh_123",
		EndpointSlug: "deploy-review",
	}
	jobRun := automationpkg.Run{
		ID:        "run-job",
		JobID:     job.ID,
		Status:    automationpkg.RunCompleted,
		Attempt:   1,
		StartedAt: &startedAt,
		EndedAt:   &endedAt,
	}
	triggerRun := automationpkg.Run{
		ID:        "run-trigger",
		TriggerID: trigger.ID,
		Status:    automationpkg.RunCompleted,
		Attempt:   1,
		StartedAt: &startedAt,
		EndedAt:   &endedAt,
	}

	var listJobsQuery automationpkg.JobListQuery
	var listTriggersQuery automationpkg.TriggerListQuery
	var runsQueries []automationpkg.RunQuery
	var webhookRequest automationpkg.WebhookRequest
	jobDeleted := false
	triggerDeleted := false

	router := newAutomationCoreTestRouter(t, stubAutomationManager{
		ListJobsFn: func(_ context.Context, query automationpkg.JobListQuery) ([]automationpkg.Job, error) {
			listJobsQuery = query
			return []automationpkg.Job{job}, nil
		},
		GetJobFn: func(_ context.Context, id string) (automationpkg.Job, error) {
			if id != job.ID {
				t.Fatalf("GetJob() id = %q, want %q", id, job.ID)
			}
			return job, nil
		},
		CreateJobFn: func(_ context.Context, created automationpkg.Job) (automationpkg.Job, error) {
			if created.Scope != automationpkg.AutomationScopeGlobal || created.Source != automationpkg.JobSourceDynamic {
				t.Fatalf("CreateJob() job = %#v", created)
			}
			if created.Name != "nightly-review" || created.AgentName != "coder" || created.Prompt != "review repo" {
				t.Fatalf("CreateJob() trimming failed: %#v", created)
			}
			if created.Schedule == nil || created.Schedule.Interval != "1h" {
				t.Fatalf("CreateJob() schedule = %#v", created.Schedule)
			}
			if !created.Enabled {
				t.Fatalf("CreateJob() enabled = %v, want true", created.Enabled)
			}
			return job, nil
		},
		DeleteJobFn: func(_ context.Context, id string) error {
			jobDeleted = true
			if id != job.ID {
				t.Fatalf("DeleteJob() id = %q, want %q", id, job.ID)
			}
			return nil
		},
		TriggerJobFn: func(_ context.Context, id string) (automationpkg.Run, error) {
			if id != job.ID {
				t.Fatalf("TriggerJob() id = %q, want %q", id, job.ID)
			}
			return jobRun, nil
		},
		ListTriggersFn: func(_ context.Context, query automationpkg.TriggerListQuery) ([]automationpkg.Trigger, error) {
			listTriggersQuery = query
			return []automationpkg.Trigger{trigger}, nil
		},
		GetTriggerFn: func(_ context.Context, id string) (automationpkg.Trigger, error) {
			if id != trigger.ID {
				t.Fatalf("GetTrigger() id = %q, want %q", id, trigger.ID)
			}
			return trigger, nil
		},
		CreateTriggerFn: func(_ context.Context, created automationpkg.Trigger, secret string) (automationpkg.Trigger, error) {
			if secret != "shared-secret" {
				t.Fatalf("CreateTrigger() secret = %q, want %q", secret, "shared-secret")
			}
			if created.Scope != automationpkg.AutomationScopeWorkspace || created.WorkspaceID != "ws-alpha" {
				t.Fatalf("CreateTrigger() scope/workspace = %#v", created)
			}
			if created.Source != automationpkg.JobSourceDynamic || created.WebhookID != "wbh_123" || created.EndpointSlug != "deploy-review" {
				t.Fatalf("CreateTrigger() webhook fields = %#v", created)
			}
			if created.Filter["data.branch"] != "main" {
				t.Fatalf("CreateTrigger() filter = %#v", created.Filter)
			}
			return trigger, nil
		},
		DeleteTriggerFn: func(_ context.Context, id string) error {
			triggerDeleted = true
			if id != trigger.ID {
				t.Fatalf("DeleteTrigger() id = %q, want %q", id, trigger.ID)
			}
			return nil
		},
		ListRunsFn: func(_ context.Context, query automationpkg.RunQuery) ([]automationpkg.Run, error) {
			runsQueries = append(runsQueries, query)
			switch {
			case query.JobID == job.ID:
				return []automationpkg.Run{jobRun}, nil
			case query.TriggerID == trigger.ID:
				return []automationpkg.Run{triggerRun}, nil
			default:
				return []automationpkg.Run{jobRun, triggerRun}, nil
			}
		},
		GetRunFn: func(_ context.Context, id string) (automationpkg.Run, error) {
			if id != triggerRun.ID {
				t.Fatalf("GetRun() id = %q, want %q", id, triggerRun.ID)
			}
			return triggerRun, nil
		},
		StatusFn: func(context.Context) (automationpkg.ManagerStatus, error) {
			return automationpkg.ManagerStatus{
				Running:          true,
				SchedulerRunning: true,
				Jobs:             automationpkg.ResourceStatus{Total: 1, Enabled: 1},
				Triggers:         automationpkg.ResourceStatus{Total: 1, Enabled: 1},
				ScheduledJobs: []automationpkg.ScheduledJobState{{
					JobID:      job.ID,
					Registered: true,
					NextRun:    &nextRun,
				}},
			}, nil
		},
		HandleWebhookFn: func(_ context.Context, request automationpkg.WebhookRequest) (automationpkg.TriggerResult, error) {
			webhookRequest = request
			return automationpkg.TriggerResult{Matched: 1, Runs: []automationpkg.Run{triggerRun}}, nil
		},
	})

	jobList := performAutomationCoreRequest(t, router, http.MethodGet, "/automation/jobs?scope=global&source=dynamic&limit=3", nil, nil)
	if jobList.Code != http.StatusOK {
		t.Fatalf("job list status = %d, want %d; body=%s", jobList.Code, http.StatusOK, jobList.Body.String())
	}
	var jobsResponse contract.JobsResponse
	decodeAutomationCoreJSON(t, jobList, &jobsResponse)
	if len(jobsResponse.Jobs) != 1 || jobsResponse.Jobs[0].NextRun == nil {
		t.Fatalf("jobs response = %#v", jobsResponse.Jobs)
	}
	if listJobsQuery.Scope != automationpkg.AutomationScopeGlobal || listJobsQuery.Source != automationpkg.JobSourceDynamic || listJobsQuery.Limit != 3 {
		t.Fatalf("ListJobs() query = %#v", listJobsQuery)
	}

	jobCreate := performAutomationCoreRequest(t, router, http.MethodPost, "/automation/jobs", []byte(`{"scope":"global","name":" nightly-review ","agent_name":" coder ","prompt":" review repo ","schedule":{"mode":"every","interval":"1h"}}`), nil)
	if jobCreate.Code != http.StatusCreated {
		t.Fatalf("job create status = %d, want %d; body=%s", jobCreate.Code, http.StatusCreated, jobCreate.Body.String())
	}

	jobGet := performAutomationCoreRequest(t, router, http.MethodGet, "/automation/jobs/"+job.ID, nil, nil)
	if jobGet.Code != http.StatusOK {
		t.Fatalf("job get status = %d, want %d; body=%s", jobGet.Code, http.StatusOK, jobGet.Body.String())
	}

	jobTrigger := performAutomationCoreRequest(t, router, http.MethodPost, "/automation/jobs/"+job.ID+"/trigger", nil, nil)
	if jobTrigger.Code != http.StatusOK {
		t.Fatalf("job trigger status = %d, want %d; body=%s", jobTrigger.Code, http.StatusOK, jobTrigger.Body.String())
	}

	jobRuns := performAutomationCoreRequest(t, router, http.MethodGet, "/automation/jobs/"+job.ID+"/runs?status=completed&limit=2", nil, nil)
	if jobRuns.Code != http.StatusOK {
		t.Fatalf("job runs status = %d, want %d; body=%s", jobRuns.Code, http.StatusOK, jobRuns.Body.String())
	}

	jobDelete := performAutomationCoreRequest(t, router, http.MethodDelete, "/automation/jobs/"+job.ID, nil, nil)
	if jobDelete.Code != http.StatusNoContent {
		t.Fatalf("job delete status = %d, want %d; body=%s", jobDelete.Code, http.StatusNoContent, jobDelete.Body.String())
	}
	if !jobDeleted {
		t.Fatal("DeleteJob() not called")
	}

	triggerList := performAutomationCoreRequest(t, router, http.MethodGet, "/automation/triggers?scope=workspace&workspace_id=ws-alpha&source=dynamic&event=webhook&limit=2", nil, nil)
	if triggerList.Code != http.StatusOK {
		t.Fatalf("trigger list status = %d, want %d; body=%s", triggerList.Code, http.StatusOK, triggerList.Body.String())
	}
	if listTriggersQuery.Scope != automationpkg.AutomationScopeWorkspace || listTriggersQuery.WorkspaceID != "ws-alpha" || listTriggersQuery.Source != automationpkg.JobSourceDynamic || listTriggersQuery.Event != "webhook" || listTriggersQuery.Limit != 2 {
		t.Fatalf("ListTriggers() query = %#v", listTriggersQuery)
	}

	triggerCreate := performAutomationCoreRequest(t, router, http.MethodPost, "/automation/triggers", []byte(`{"scope":"workspace","workspace_id":" ws-alpha ","name":" deploy-review ","agent_name":" coder ","prompt":" review {{ index .Data \"payload\" }} ","event":"webhook","filter":{"data.branch":"main"},"webhook_id":" wbh_123 ","endpoint_slug":" deploy-review ","webhook_secret":"shared-secret"}`), nil)
	if triggerCreate.Code != http.StatusCreated {
		t.Fatalf("trigger create status = %d, want %d; body=%s", triggerCreate.Code, http.StatusCreated, triggerCreate.Body.String())
	}

	triggerGet := performAutomationCoreRequest(t, router, http.MethodGet, "/automation/triggers/"+trigger.ID, nil, nil)
	if triggerGet.Code != http.StatusOK {
		t.Fatalf("trigger get status = %d, want %d; body=%s", triggerGet.Code, http.StatusOK, triggerGet.Body.String())
	}

	triggerRuns := performAutomationCoreRequest(t, router, http.MethodGet, "/automation/triggers/"+trigger.ID+"/runs?status=completed&limit=1", nil, nil)
	if triggerRuns.Code != http.StatusOK {
		t.Fatalf("trigger runs status = %d, want %d; body=%s", triggerRuns.Code, http.StatusOK, triggerRuns.Body.String())
	}

	allRuns := performAutomationCoreRequest(t, router, http.MethodGet, "/automation/runs?status=completed&job_id="+job.ID+"&limit=5&since=2026-04-11T12:00:00Z&until=2026-04-11T13:00:00Z", nil, nil)
	if allRuns.Code != http.StatusOK {
		t.Fatalf("all runs status = %d, want %d; body=%s", allRuns.Code, http.StatusOK, allRuns.Body.String())
	}
	if len(runsQueries) != 3 {
		t.Fatalf("ListRuns() calls = %d, want 3", len(runsQueries))
	}
	if runsQueries[2].Status != automationpkg.RunCompleted || runsQueries[2].Limit != 5 || runsQueries[2].JobID != job.ID || runsQueries[2].Since.IsZero() || runsQueries[2].Until.IsZero() {
		t.Fatalf("ListRuns() final query = %#v", runsQueries[2])
	}

	runGet := performAutomationCoreRequest(t, router, http.MethodGet, "/automation/runs/"+triggerRun.ID, nil, nil)
	if runGet.Code != http.StatusOK {
		t.Fatalf("run get status = %d, want %d; body=%s", runGet.Code, http.StatusOK, runGet.Body.String())
	}

	triggerDelete := performAutomationCoreRequest(t, router, http.MethodDelete, "/automation/triggers/"+trigger.ID, nil, nil)
	if triggerDelete.Code != http.StatusNoContent {
		t.Fatalf("trigger delete status = %d, want %d; body=%s", triggerDelete.Code, http.StatusNoContent, triggerDelete.Body.String())
	}
	if !triggerDeleted {
		t.Fatal("DeleteTrigger() not called")
	}

	webhookPayload := []byte(`{"payload":"deploy"}`)
	webhookDelivery := performAutomationCoreRequest(t, router, http.MethodPost, "/webhooks/workspaces/ws-alpha/deploy-review--wbh_123", webhookPayload, map[string]string{
		WebhookTimestampHeader:  strconv.FormatInt(time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC).Unix(), 10),
		WebhookSignatureHeader:  "sha256=deadbeef",
		WebhookDeliveryIDHeader: "delivery-roundtrip",
	})
	if webhookDelivery.Code != http.StatusOK {
		t.Fatalf("workspace webhook status = %d, want %d; body=%s", webhookDelivery.Code, http.StatusOK, webhookDelivery.Body.String())
	}
	if webhookRequest.Scope != automationpkg.AutomationScopeWorkspace || webhookRequest.WorkspaceID != "ws-alpha" {
		t.Fatalf("webhook request scope/workspace = %#v", webhookRequest)
	}
	if webhookRequest.Endpoint != "deploy-review--wbh_123" || webhookRequest.Signature != "sha256=deadbeef" || webhookRequest.DeliveryID != "delivery-roundtrip" {
		t.Fatalf("webhook request routing = %#v", webhookRequest)
	}
	if payload := webhookRequest.Data["payload"]; payload != "deploy" {
		t.Fatalf("webhook request data = %#v", webhookRequest.Data)
	}
}

func TestAutomationHelperFunctionsAndErrors(t *testing.T) {
	t.Parallel()

	rootCause := errors.New("bad request")
	validationErr := NewAutomationValidationError(rootCause)
	if !errors.Is(validationErr, ErrAutomationValidation) {
		t.Fatalf("NewAutomationValidationError() = %v, want ErrAutomationValidation", validationErr)
	}
	if !errors.Is(validationErr, rootCause) {
		t.Fatalf("NewAutomationValidationError() = %v, want wrapped root cause", validationErr)
	}

	if _, err := parseWebhookTimestampHeader(""); err == nil {
		t.Fatal("parseWebhookTimestampHeader(empty) error = nil, want error")
	}

	seconds := time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC).Unix()
	parsed, err := parseWebhookTimestampHeader(strconv.FormatInt(seconds, 10))
	if err != nil {
		t.Fatalf("parseWebhookTimestampHeader(unix) error = %v", err)
	}
	if parsed.Unix() != seconds {
		t.Fatalf("parseWebhookTimestampHeader(unix) = %v, want unix %d", parsed, seconds)
	}
	if _, err := parseWebhookTimestampHeader("not-a-time"); err == nil {
		t.Fatal("parseWebhookTimestampHeader(invalid) error = nil, want error")
	}

	if data := decodeWebhookPayloadData(nil); data != nil {
		t.Fatalf("decodeWebhookPayloadData(nil) = %#v, want nil", data)
	}
	if data := decodeWebhookPayloadData([]byte("not-json")); data != nil {
		t.Fatalf("decodeWebhookPayloadData(invalid) = %#v, want nil", data)
	}
	data := decodeWebhookPayloadData([]byte(`{"payload":"deploy"}`))
	if data["payload"] != "deploy" {
		t.Fatalf("decodeWebhookPayloadData(json) = %#v", data)
	}

	createdJob := jobFromCreateRequest(contract.CreateJobRequest{
		Scope:       automationpkg.AutomationScopeWorkspace,
		Name:        " build review ",
		AgentName:   " coder ",
		WorkspaceID: " ws-alpha ",
		Prompt:      " inspect repo ",
		Schedule: automationpkg.ScheduleSpec{
			Mode:     automationpkg.ScheduleModeEvery,
			Interval: "2h",
		},
	})
	if createdJob.Scope != automationpkg.AutomationScopeWorkspace || createdJob.Name != "build review" || createdJob.AgentName != "coder" || createdJob.WorkspaceID != "ws-alpha" || createdJob.Prompt != "inspect repo" || createdJob.Schedule == nil || createdJob.Schedule.Interval != "2h" {
		t.Fatalf("jobFromCreateRequest() = %#v", createdJob)
	}

	jobName := " renamed "
	jobAgent := " reviewer "
	jobWorkspace := " ws-beta "
	jobPrompt := " next prompt "
	jobEnabled := false
	jobSchedule := automationpkg.ScheduleSpec{Mode: automationpkg.ScheduleModeCron, Expr: "0 * * * *"}
	jobRetry := automationpkg.RetryConfig{Strategy: automationpkg.RetryStrategyBackoff, MaxRetries: 3, BaseDelay: "2m"}
	jobFireLimit := automationpkg.FireLimitConfig{Max: 4, Window: "24h"}
	updatedJob := applyJobPatch(automationpkg.Job{
		ID:          "job-1",
		Name:        "before",
		AgentName:   "old-agent",
		WorkspaceID: "ws-alpha",
		Prompt:      "old",
		Enabled:     true,
		Schedule:    &automationpkg.ScheduleSpec{Mode: automationpkg.ScheduleModeEvery, Interval: "1h"},
		Source:      automationpkg.JobSourceDynamic,
		Retry:       automationpkg.DefaultRetryConfig(),
		FireLimit:   automationpkg.DefaultFireLimitConfig(),
	}, contract.UpdateJobRequest{
		Name:        &jobName,
		AgentName:   &jobAgent,
		WorkspaceID: &jobWorkspace,
		Prompt:      &jobPrompt,
		Schedule:    &jobSchedule,
		Enabled:     &jobEnabled,
		Retry:       &jobRetry,
		FireLimit:   &jobFireLimit,
	})
	if updatedJob.Name != "renamed" || updatedJob.AgentName != "reviewer" || updatedJob.WorkspaceID != "ws-beta" || updatedJob.Prompt != "next prompt" || updatedJob.Enabled || updatedJob.Schedule == nil || updatedJob.Schedule.Expr != "0 * * * *" || updatedJob.Retry.MaxRetries != 3 || updatedJob.FireLimit.Max != 4 {
		t.Fatalf("applyJobPatch() = %#v", updatedJob)
	}

	createdTrigger := triggerFromCreateRequest(contract.CreateTriggerRequest{
		Scope:        automationpkg.AutomationScopeWorkspace,
		Name:         " deploy-review ",
		AgentName:    " coder ",
		WorkspaceID:  " ws-alpha ",
		Prompt:       ` review {{ index .Data "payload" }} `,
		Event:        " webhook ",
		Filter:       map[string]string{"data.branch": "main"},
		WebhookID:    " wbh_456 ",
		EndpointSlug: " deploy-review ",
	})
	if createdTrigger.Scope != automationpkg.AutomationScopeWorkspace || createdTrigger.Name != "deploy-review" || createdTrigger.AgentName != "coder" || createdTrigger.WorkspaceID != "ws-alpha" || createdTrigger.Event != "webhook" || createdTrigger.WebhookID != "wbh_456" || createdTrigger.EndpointSlug != "deploy-review" || createdTrigger.Filter["data.branch"] != "main" {
		t.Fatalf("triggerFromCreateRequest() = %#v", createdTrigger)
	}

	triggerEvent := "session.stopped"
	triggerFilter := map[string]string{"kind": "session"}
	triggerEnabled := false
	triggerRetry := automationpkg.RetryConfig{Strategy: automationpkg.RetryStrategyBackoff, MaxRetries: 2, BaseDelay: "30s"}
	triggerFireLimit := automationpkg.FireLimitConfig{Max: 2, Window: "1h"}
	updatedTrigger := applyTriggerPatch(automationpkg.Trigger{
		ID:           "trigger-1",
		Name:         "before",
		AgentName:    "old-agent",
		WorkspaceID:  "ws-alpha",
		Prompt:       "old",
		Event:        "webhook",
		Filter:       map[string]string{"branch": "main"},
		Enabled:      true,
		WebhookID:    "wbh_123",
		EndpointSlug: "deploy-review",
		Source:       automationpkg.JobSourceDynamic,
		Retry:        automationpkg.DefaultRetryConfig(),
		FireLimit:    automationpkg.DefaultFireLimitConfig(),
	}, contract.UpdateTriggerRequest{
		Name:        &jobName,
		AgentName:   &jobAgent,
		WorkspaceID: &jobWorkspace,
		Prompt:      &jobPrompt,
		Event:       &triggerEvent,
		Filter:      triggerFilter,
		Enabled:     &triggerEnabled,
		Retry:       &triggerRetry,
		FireLimit:   &triggerFireLimit,
	})
	if updatedTrigger.Name != "renamed" || updatedTrigger.AgentName != "reviewer" || updatedTrigger.WorkspaceID != "ws-beta" || updatedTrigger.Prompt != "next prompt" || updatedTrigger.Event != "session.stopped" || updatedTrigger.WebhookID != "" || updatedTrigger.EndpointSlug != "" || updatedTrigger.Enabled || updatedTrigger.Retry.MaxRetries != 2 || updatedTrigger.FireLimit.Max != 2 {
		t.Fatalf("applyTriggerPatch() = %#v", updatedTrigger)
	}
	triggerFilter["kind"] = "mutated"
	if updatedTrigger.Filter["kind"] != "session" {
		t.Fatalf("applyTriggerPatch() filter clone = %#v", updatedTrigger.Filter)
	}
	if clone := cloneAutomationFilter(nil); clone != nil {
		t.Fatalf("cloneAutomationFilter(nil) = %#v, want nil", clone)
	}

	if status := StatusForAutomationError(validationErr); status != http.StatusBadRequest {
		t.Fatalf("StatusForAutomationError(validation) = %d, want %d", status, http.StatusBadRequest)
	}
	if status := StatusForAutomationError(&http.MaxBytesError{Limit: maxWebhookPayloadSize}); status != http.StatusRequestEntityTooLarge {
		t.Fatalf("StatusForAutomationError(max bytes) = %d, want %d", status, http.StatusRequestEntityTooLarge)
	}
	if status := StatusForAutomationError(automationpkg.ErrWebhookSignatureInvalid); status != http.StatusUnauthorized {
		t.Fatalf("StatusForAutomationError(signature) = %d, want %d", status, http.StatusUnauthorized)
	}
	if status := StatusForAutomationError(automationpkg.ErrRunNotFound); status != http.StatusNotFound {
		t.Fatalf("StatusForAutomationError(not found) = %d, want %d", status, http.StatusNotFound)
	}
	if status := StatusForAutomationError(automationpkg.ErrJobOverlayNotFound); status != http.StatusNotFound {
		t.Fatalf("StatusForAutomationError(job overlay not found) = %d, want %d", status, http.StatusNotFound)
	}
	if status := StatusForAutomationError(automationpkg.ErrFireLimitReached); status != http.StatusConflict {
		t.Fatalf("StatusForAutomationError(conflict) = %d, want %d", status, http.StatusConflict)
	}
	if status := StatusForAutomationError(automationpkg.ErrOverlayRequiresConfigSource); status != http.StatusConflict {
		t.Fatalf("StatusForAutomationError(overlay requires config source) = %d, want %d", status, http.StatusConflict)
	}
	if status := StatusForAutomationError(automationpkg.ErrWebhookReplayDetected); status != http.StatusConflict {
		t.Fatalf("StatusForAutomationError(webhook replay) = %d, want %d", status, http.StatusConflict)
	}
	if status := StatusForAutomationError(automationpkg.ErrManagerNotRunning); status != http.StatusServiceUnavailable {
		t.Fatalf("StatusForAutomationError(unavailable) = %d, want %d", status, http.StatusServiceUnavailable)
	}
}

func TestWebhookRequestValidationRejectsBodiesThatExceedTheConfiguredLimit(t *testing.T) {
	t.Parallel()

	called := false
	router := newAutomationCoreTestRouter(t, stubAutomationManager{
		HandleWebhookFn: func(context.Context, automationpkg.WebhookRequest) (automationpkg.TriggerResult, error) {
			called = true
			return automationpkg.TriggerResult{}, nil
		},
	})

	tooLargeBody := bytes.Repeat([]byte("a"), maxWebhookPayloadSize+1)
	response := performAutomationCoreRequest(t, router, http.MethodPost, "/webhooks/global/deploy-review--wbh_123", tooLargeBody, map[string]string{
		WebhookTimestampHeader:  time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC).Format(time.RFC3339),
		WebhookSignatureHeader:  "sha256=deadbeef",
		WebhookDeliveryIDHeader: "delivery-too-large",
	})
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized webhook status = %d, want %d; body=%s", response.Code, http.StatusRequestEntityTooLarge, response.Body.String())
	}
	if called {
		t.Fatal("HandleWebhook() called for oversized webhook body")
	}
}

func TestAutomationHandlersRequireConfiguredManager(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	homePaths, err := aghconfig.ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	cfg := aghconfig.DefaultWithHome(homePaths)

	handlers := NewBaseHandlers(BaseHandlerConfig{
		TransportName: "core-automation-test",
		HomePaths:     homePaths,
		Config:        cfg,
		Logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	engine := gin.New()
	engine.GET("/automation/jobs", handlers.ListAutomationJobs)

	response := performAutomationCoreRequest(t, engine, http.MethodGet, "/automation/jobs", nil, nil)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusServiceUnavailable, response.Body.String())
	}
}

func newAutomationCoreTestRouter(t *testing.T, automation stubAutomationManager) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)
	homePaths, err := aghconfig.ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	cfg := aghconfig.DefaultWithHome(homePaths)

	handlers := NewBaseHandlers(BaseHandlerConfig{
		TransportName: "core-automation-test",
		Automation:    automation,
		HomePaths:     homePaths,
		Config:        cfg,
		Logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		StartedAt:     time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC),
		Now: func() time.Time {
			return time.Date(2026, 4, 11, 12, 0, 1, 0, time.UTC)
		},
	})

	engine := gin.New()
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
	return engine
}

func performAutomationCoreRequest(t *testing.T, engine http.Handler, method, path string, body []byte, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	if len(body) > 0 {
		request.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		request.Header.Set(key, value)
	}

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	return recorder
}

func decodeAutomationCoreJSON(t *testing.T, recorder *httptest.ResponseRecorder, dest any) {
	t.Helper()
	if err := json.Unmarshal(recorder.Body.Bytes(), dest); err != nil {
		t.Fatalf("json.Unmarshal() error = %v; body=%s", err, recorder.Body.String())
	}
}

type stubAutomationManager struct {
	ListJobsFn          func(context.Context, automationpkg.JobListQuery) ([]automationpkg.Job, error)
	GetJobFn            func(context.Context, string) (automationpkg.Job, error)
	CreateJobFn         func(context.Context, automationpkg.Job) (automationpkg.Job, error)
	UpdateJobFn         func(context.Context, automationpkg.Job) (automationpkg.Job, error)
	DeleteJobFn         func(context.Context, string) error
	TriggerJobFn        func(context.Context, string) (automationpkg.Run, error)
	ListTriggersFn      func(context.Context, automationpkg.TriggerListQuery) ([]automationpkg.Trigger, error)
	GetTriggerFn        func(context.Context, string) (automationpkg.Trigger, error)
	CreateTriggerFn     func(context.Context, automationpkg.Trigger, string) (automationpkg.Trigger, error)
	UpdateTriggerFn     func(context.Context, automationpkg.Trigger, *string) (automationpkg.Trigger, error)
	DeleteTriggerFn     func(context.Context, string) error
	ListRunsFn          func(context.Context, automationpkg.RunQuery) ([]automationpkg.Run, error)
	GetRunFn            func(context.Context, string) (automationpkg.Run, error)
	StatusFn            func(context.Context) (automationpkg.ManagerStatus, error)
	SetJobEnabledFn     func(context.Context, string, bool) (automationpkg.Job, error)
	SetTriggerEnabledFn func(context.Context, string, bool) (automationpkg.Trigger, error)
	HandleWebhookFn     func(context.Context, automationpkg.WebhookRequest) (automationpkg.TriggerResult, error)
}

func (s stubAutomationManager) ListJobs(ctx context.Context, query automationpkg.JobListQuery) ([]automationpkg.Job, error) {
	if s.ListJobsFn == nil {
		return nil, nil
	}
	return s.ListJobsFn(ctx, query)
}

func (s stubAutomationManager) GetJob(ctx context.Context, id string) (automationpkg.Job, error) {
	if s.GetJobFn == nil {
		return automationpkg.Job{}, nil
	}
	return s.GetJobFn(ctx, id)
}

func (s stubAutomationManager) CreateJob(ctx context.Context, job automationpkg.Job) (automationpkg.Job, error) {
	if s.CreateJobFn == nil {
		return automationpkg.Job{}, nil
	}
	return s.CreateJobFn(ctx, job)
}

func (s stubAutomationManager) UpdateJob(ctx context.Context, job automationpkg.Job) (automationpkg.Job, error) {
	if s.UpdateJobFn == nil {
		return automationpkg.Job{}, nil
	}
	return s.UpdateJobFn(ctx, job)
}

func (s stubAutomationManager) DeleteJob(ctx context.Context, id string) error {
	if s.DeleteJobFn == nil {
		return nil
	}
	return s.DeleteJobFn(ctx, id)
}

func (s stubAutomationManager) TriggerJob(ctx context.Context, id string) (automationpkg.Run, error) {
	if s.TriggerJobFn == nil {
		return automationpkg.Run{}, nil
	}
	return s.TriggerJobFn(ctx, id)
}

func (s stubAutomationManager) Jobs(ctx context.Context) ([]automationpkg.Job, error) {
	return s.ListJobs(ctx, automationpkg.JobListQuery{})
}

func (s stubAutomationManager) ListTriggers(ctx context.Context, query automationpkg.TriggerListQuery) ([]automationpkg.Trigger, error) {
	if s.ListTriggersFn == nil {
		return nil, nil
	}
	return s.ListTriggersFn(ctx, query)
}

func (s stubAutomationManager) GetTrigger(ctx context.Context, id string) (automationpkg.Trigger, error) {
	if s.GetTriggerFn == nil {
		return automationpkg.Trigger{}, nil
	}
	return s.GetTriggerFn(ctx, id)
}

func (s stubAutomationManager) CreateTrigger(ctx context.Context, trigger automationpkg.Trigger, secret string) (automationpkg.Trigger, error) {
	if s.CreateTriggerFn == nil {
		return automationpkg.Trigger{}, nil
	}
	return s.CreateTriggerFn(ctx, trigger, secret)
}

func (s stubAutomationManager) UpdateTrigger(ctx context.Context, trigger automationpkg.Trigger, secret *string) (automationpkg.Trigger, error) {
	if s.UpdateTriggerFn == nil {
		return automationpkg.Trigger{}, nil
	}
	return s.UpdateTriggerFn(ctx, trigger, secret)
}

func (s stubAutomationManager) DeleteTrigger(ctx context.Context, id string) error {
	if s.DeleteTriggerFn == nil {
		return nil
	}
	return s.DeleteTriggerFn(ctx, id)
}

func (s stubAutomationManager) Triggers(ctx context.Context) ([]automationpkg.Trigger, error) {
	return s.ListTriggers(ctx, automationpkg.TriggerListQuery{})
}

func (s stubAutomationManager) ListRuns(ctx context.Context, query automationpkg.RunQuery) ([]automationpkg.Run, error) {
	if s.ListRunsFn == nil {
		return nil, nil
	}
	return s.ListRunsFn(ctx, query)
}

func (s stubAutomationManager) GetRun(ctx context.Context, id string) (automationpkg.Run, error) {
	if s.GetRunFn == nil {
		return automationpkg.Run{}, nil
	}
	return s.GetRunFn(ctx, id)
}

func (s stubAutomationManager) Runs(ctx context.Context, query automationpkg.RunQuery) ([]automationpkg.Run, error) {
	return s.ListRuns(ctx, query)
}

func (s stubAutomationManager) Status(ctx context.Context) (automationpkg.ManagerStatus, error) {
	if s.StatusFn == nil {
		return automationpkg.ManagerStatus{}, nil
	}
	return s.StatusFn(ctx)
}

func (s stubAutomationManager) SetJobEnabled(ctx context.Context, id string, enabled bool) (automationpkg.Job, error) {
	if s.SetJobEnabledFn == nil {
		return automationpkg.Job{}, nil
	}
	return s.SetJobEnabledFn(ctx, id, enabled)
}

func (s stubAutomationManager) SetTriggerEnabled(ctx context.Context, id string, enabled bool) (automationpkg.Trigger, error) {
	if s.SetTriggerEnabledFn == nil {
		return automationpkg.Trigger{}, nil
	}
	return s.SetTriggerEnabledFn(ctx, id, enabled)
}

func (s stubAutomationManager) HandleWebhook(ctx context.Context, request automationpkg.WebhookRequest) (automationpkg.TriggerResult, error) {
	if s.HandleWebhookFn == nil {
		return automationpkg.TriggerResult{}, nil
	}
	return s.HandleWebhookFn(ctx, request)
}
