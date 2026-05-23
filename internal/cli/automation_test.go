package cli

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/compozy/agh/internal/api/contract"
	automationpkg "github.com/compozy/agh/internal/automation"
)

func TestAutomationJobsCreateParsesWorkspaceScopeAndRetry(t *testing.T) {
	t.Parallel()

	var request AutomationJobCreateRequest
	deps := newTestDeps(t, &stubClient{
		getWorkspaceFn: func(_ context.Context, ref string) (WorkspaceDetailRecord, error) {
			if ref != "alpha" {
				t.Fatalf("GetWorkspace ref = %q, want %q", ref, "alpha")
			}
			return WorkspaceDetailRecord{Workspace: WorkspaceRecord{ID: "ws-alpha"}}, nil
		},
		createAutomationJobFn: func(_ context.Context, got AutomationJobCreateRequest) (JobRecord, error) {
			request = got
			return sampleAutomationJobRecord(), nil
		},
	})

	stdout, _, err := executeRootCommand(
		t,
		deps,
		"automation", "jobs", "create",
		"--name", "nightly-review",
		"--scope", "workspace",
		"--workspace", "alpha",
		"--schedule", "every:30m",
		"--agent", "coder",
		"--prompt", "review repo",
		"--retry", "backoff:3:2s",
		"-o", "json",
	)
	if err != nil {
		t.Fatalf("executeRootCommand(automation jobs create) error = %v", err)
	}

	if request.Scope != automationpkg.AutomationScopeWorkspace || request.WorkspaceID != "ws-alpha" {
		t.Fatalf("request scope/workspace = %#v, want workspace ws-alpha", request)
	}
	if request.Schedule.Mode != automationpkg.ScheduleModeEvery || request.Schedule.Interval != "30m" {
		t.Fatalf("request schedule = %#v, want every 30m", request.Schedule)
	}
	if request.Retry == nil || request.Retry.Strategy != automationpkg.RetryStrategyBackoff ||
		request.Retry.MaxRetries != 3 ||
		request.Retry.BaseDelay != "2s" {
		t.Fatalf("request retry = %#v, want backoff 3 2s", request.Retry)
	}

	var created JobRecord
	if err := json.Unmarshal([]byte(stdout), &created); err != nil {
		t.Fatalf("json.Unmarshal(job create) error = %v", err)
	}
	if created.ID != "job-1" {
		t.Fatalf("created.ID = %q, want %q", created.ID, "job-1")
	}
}

func TestAutomationJobsCreateRejectsMissingWorkspaceForWorkspaceScope(t *testing.T) {
	t.Parallel()

	_, _, err := executeRootCommand(
		t,
		newTestDeps(t, &stubClient{}),
		"automation", "jobs", "create",
		"--name", "nightly-review",
		"--scope", "workspace",
		"--schedule", "every:30m",
		"--agent", "coder",
		"--prompt", "review repo",
	)
	if err == nil || !strings.Contains(err.Error(), "--workspace is required when --scope is workspace") {
		t.Fatalf("missing workspace error = %v, want workspace requirement", err)
	}
}

func TestAutomationJobsCreateSupportsHumanAndJSONOutput(t *testing.T) {
	t.Parallel()

	deps := newTestDeps(t, &stubClient{
		createAutomationJobFn: func(context.Context, AutomationJobCreateRequest) (JobRecord, error) {
			return sampleAutomationJobRecord(), nil
		},
	})

	humanOut, _, err := executeRootCommand(
		t,
		deps,
		"automation", "jobs", "create",
		"--name", "nightly-review",
		"--scope", "global",
		"--schedule", "every:30m",
		"--agent", "coder",
		"--prompt", "review repo",
		"-o", "human",
	)
	if err != nil {
		t.Fatalf("automation jobs create human error = %v", err)
	}
	if !strings.Contains(humanOut, "Automation Job") || !strings.Contains(humanOut, "nightly-review") {
		t.Fatalf("human output = %q, want Automation Job details", humanOut)
	}

	jsonOut, _, err := executeRootCommand(
		t,
		deps,
		"automation", "jobs", "create",
		"--name", "nightly-review",
		"--scope", "global",
		"--schedule", "every:30m",
		"--agent", "coder",
		"--prompt", "review repo",
		"-o", "json",
	)
	if err != nil {
		t.Fatalf("automation jobs create json error = %v", err)
	}

	var created JobRecord
	if err := json.Unmarshal([]byte(jsonOut), &created); err != nil {
		t.Fatalf("json.Unmarshal(job create json) error = %v", err)
	}
	if created.ID != "job-1" {
		t.Fatalf("created.ID = %q, want %q", created.ID, "job-1")
	}
}

func TestAutomationTriggersCreateParsesFilters(t *testing.T) {
	t.Parallel()

	var request AutomationTriggerCreateRequest
	deps := newTestDeps(t, &stubClient{
		getWorkspaceFn: func(_ context.Context, ref string) (WorkspaceDetailRecord, error) {
			if ref != "alpha" {
				t.Fatalf("GetWorkspace ref = %q, want %q", ref, "alpha")
			}
			return WorkspaceDetailRecord{Workspace: WorkspaceRecord{ID: "ws-alpha"}}, nil
		},
		createAutomationTriggerFn: func(_ context.Context, got AutomationTriggerCreateRequest) (TriggerRecord, error) {
			request = got
			return sampleAutomationTriggerRecord(), nil
		},
	})

	stdout, _, err := executeRootCommand(
		t,
		deps,
		"automation", "triggers", "create",
		"--name", "branch-review",
		"--scope", "workspace",
		"--workspace", "alpha",
		"--event", "session.stopped",
		"--filter", "data.branch=main,data.repo=agh",
		"--agent", "coder",
		"--prompt", `review {{ index .Data "session_id" }}`,
		"-o", "json",
	)
	if err != nil {
		t.Fatalf("executeRootCommand(automation triggers create) error = %v", err)
	}

	if request.Scope != automationpkg.AutomationScopeWorkspace || request.WorkspaceID != "ws-alpha" {
		t.Fatalf("request scope/workspace = %#v, want workspace ws-alpha", request)
	}
	if request.Filter["data.branch"] != "main" || request.Filter["data.repo"] != "agh" {
		t.Fatalf("request.Filter = %#v, want parsed filter map", request.Filter)
	}

	var created TriggerRecord
	if err := json.Unmarshal([]byte(stdout), &created); err != nil {
		t.Fatalf("json.Unmarshal(trigger create) error = %v", err)
	}
	if created.ID != "trg-1" {
		t.Fatalf("created.ID = %q, want %q", created.ID, "trg-1")
	}
}

func TestAutomationTriggersCreateRejectsMalformedFilter(t *testing.T) {
	t.Parallel()

	_, _, err := executeRootCommand(
		t,
		newTestDeps(t, &stubClient{}),
		"automation", "triggers", "create",
		"--name", "branch-review",
		"--scope", "global",
		"--event", "session.stopped",
		"--filter", "data.branch",
		"--agent", "coder",
		"--prompt", `review {{ index .Data "session_id" }}`,
	)
	if err == nil || !strings.Contains(err.Error(), "invalid filter") {
		t.Fatalf("malformed filter error = %v, want invalid filter", err)
	}
}

func TestAutomationJobUpdateSurfacesConfigBackedMutationError(t *testing.T) {
	t.Parallel()

	expected := "automation validation error: config-backed automation jobs only accept enabled updates"
	deps := newTestDeps(t, &stubClient{
		updateAutomationJobFn: func(context.Context, string, AutomationJobUpdateRequest) (JobRecord, error) {
			return JobRecord{}, errors.New(expected)
		},
	})

	_, _, err := executeRootCommand(
		t,
		deps,
		"automation", "jobs", "update", "job-config",
		"--prompt", "changed",
	)
	if err == nil || !strings.Contains(err.Error(), expected) {
		t.Fatalf("config-backed update error = %v, want %q", err, expected)
	}
}

func TestAutomationJobsListAndUpdateCommands(t *testing.T) {
	t.Parallel()

	var (
		listQuery     AutomationJobQuery
		updateRequest AutomationJobUpdateRequest
	)

	deps := newTestDeps(t, &stubClient{
		getWorkspaceFn: func(_ context.Context, ref string) (WorkspaceDetailRecord, error) {
			if ref != "alpha" {
				t.Fatalf("GetWorkspace ref = %q, want %q", ref, "alpha")
			}
			return WorkspaceDetailRecord{Workspace: WorkspaceRecord{ID: "ws-alpha"}}, nil
		},
		listAutomationJobsFn: func(_ context.Context, query AutomationJobQuery) ([]JobRecord, error) {
			listQuery = query
			return []JobRecord{sampleAutomationJobRecord()}, nil
		},
		updateAutomationJobFn: func(_ context.Context, _ string, request AutomationJobUpdateRequest) (JobRecord, error) {
			updateRequest = request
			updated := sampleAutomationJobRecord()
			updated.Name = *request.Name
			updated.AgentName = *request.AgentName
			updated.WorkspaceID = *request.WorkspaceID
			updated.Prompt = *request.Prompt
			updated.Schedule = request.Schedule
			updated.Retry = *request.Retry
			updated.Enabled = *request.Enabled
			return updated, nil
		},
	})

	listJSON, _, err := executeRootCommand(
		t,
		deps,
		"automation", "jobs",
		"--scope", "workspace",
		"--workspace", "alpha",
		"--source", "dynamic",
		"--last", "3",
		"-o", "json",
	)
	if err != nil {
		t.Fatalf("automation jobs list error = %v", err)
	}

	var listed contract.JobsResponse
	if err := json.Unmarshal([]byte(listJSON), &listed); err != nil {
		t.Fatalf("json.Unmarshal(job list) error = %v", err)
	}
	if len(listed.Jobs) != 1 || listQuery.Scope != automationpkg.AutomationScopeWorkspace ||
		listQuery.WorkspaceID != "ws-alpha" ||
		listQuery.Source != automationpkg.JobSourceDynamic ||
		listQuery.Limit != 3 {
		t.Fatalf("listQuery = %#v, want resolved scope/workspace/source/limit", listQuery)
	}

	updatedJSON, _, err := executeRootCommand(
		t,
		deps,
		"automation", "jobs", "update", "job-1",
		"--name", "nightly-digest",
		"--agent", "reviewer",
		"--workspace", "alpha",
		"--prompt", "digest repo activity",
		"--schedule", "at:2026-04-15T15:00",
		"--retry", "none",
		"--enabled=false",
		"-o", "json",
	)
	if err != nil {
		t.Fatalf("automation jobs update error = %v", err)
	}

	var updated JobRecord
	if err := json.Unmarshal([]byte(updatedJSON), &updated); err != nil {
		t.Fatalf("json.Unmarshal(job update) error = %v", err)
	}
	if updated.Name != "nightly-digest" || updated.AgentName != "reviewer" || updated.WorkspaceID != "ws-alpha" ||
		updated.Schedule == nil ||
		updated.Schedule.Mode != automationpkg.ScheduleModeAt {
		t.Fatalf("updated job = %#v, want updated values", updated)
	}
	if updateRequest.Schedule == nil || updateRequest.Schedule.Mode != automationpkg.ScheduleModeAt ||
		updateRequest.Schedule.Time != "2026-04-15T15:00:00Z" {
		t.Fatalf("updateRequest.Schedule = %#v, want normalized at schedule", updateRequest.Schedule)
	}
	if updateRequest.Retry == nil || updateRequest.Retry.Strategy != automationpkg.RetryStrategyNone {
		t.Fatalf("updateRequest.Retry = %#v, want retry none", updateRequest.Retry)
	}
	if updateRequest.Enabled == nil || *updateRequest.Enabled {
		t.Fatalf("updateRequest.Enabled = %#v, want false pointer", updateRequest.Enabled)
	}
}

func TestAutomationCommandsSupportToonOutput(t *testing.T) {
	t.Parallel()

	deps := newTestDeps(t, &stubClient{
		listAutomationJobsFn: func(context.Context, AutomationJobQuery) ([]JobRecord, error) {
			return []JobRecord{sampleAutomationJobRecord()}, nil
		},
		getAutomationTriggerFn: func(context.Context, string) (TriggerRecord, error) {
			return sampleAutomationTriggerRecord(), nil
		},
		listAutomationRunsFn: func(context.Context, AutomationRunQuery) ([]RunRecord, error) {
			return []RunRecord{sampleAutomationRunRecord()}, nil
		},
	})

	jobsToon, _, err := executeRootCommand(t, deps, "automation", "jobs", "-o", "toon")
	if err != nil {
		t.Fatalf("automation jobs toon error = %v", err)
	}
	if !strings.Contains(
		jobsToon,
		"automation_jobs[1]{id,name,scope,workspace_id,schedule,agent_name,enabled,source,next_run}:",
	) {
		t.Fatalf("jobs toon output = %q, want automation_jobs TOON array", jobsToon)
	}

	triggerToon, _, err := executeRootCommand(t, deps, "automation", "triggers", "get", "trg-1", "-o", "toon")
	if err != nil {
		t.Fatalf("automation triggers get toon error = %v", err)
	}
	if !strings.Contains(
		triggerToon,
		"automation_trigger{id,name,scope,workspace_id,agent_name,event,enabled,source,retry,fire_limit,webhook_id,endpoint_slug,webhook_path,created_at,updated_at,prompt}:",
	) {
		t.Fatalf("trigger toon output = %q, want automation_trigger TOON object", triggerToon)
	}

	runsToon, _, err := executeRootCommand(t, deps, "automation", "runs", "-o", "toon")
	if err != nil {
		t.Fatalf("automation runs toon error = %v", err)
	}
	if !strings.Contains(
		runsToon,
		"automation_runs[1]{id,target,status,attempt,session_id,scheduled_at,started_at,ended_at,error,delivery_error}:",
	) {
		t.Fatalf("runs toon output = %q, want automation_runs TOON array", runsToon)
	}
}

func TestAutomationAdditionalCommandsAndQueries(t *testing.T) {
	t.Parallel()

	var (
		listTriggerQuery     AutomationTriggerQuery
		jobHistoryQuery      AutomationRunQuery
		updateTriggerRequest AutomationTriggerUpdateRequest
		runsQuery            AutomationRunQuery
	)

	deps := newTestDeps(t, &stubClient{
		getWorkspaceFn: func(_ context.Context, ref string) (WorkspaceDetailRecord, error) {
			if ref != "alpha" {
				t.Fatalf("GetWorkspace ref = %q, want %q", ref, "alpha")
			}
			return WorkspaceDetailRecord{Workspace: WorkspaceRecord{ID: "ws-alpha"}}, nil
		},
		listAutomationTriggersFn: func(_ context.Context, query AutomationTriggerQuery) ([]TriggerRecord, error) {
			listTriggerQuery = query
			return []TriggerRecord{sampleAutomationTriggerRecord()}, nil
		},
		getAutomationJobFn: func(context.Context, string) (JobRecord, error) {
			return sampleAutomationJobRecord(), nil
		},
		deleteAutomationJobFn: func(context.Context, string) error {
			return nil
		},
		triggerAutomationJobFn: func(context.Context, string) (RunRecord, error) {
			return sampleAutomationRunRecord(), nil
		},
		automationJobRunsFn: func(_ context.Context, _ string, query AutomationRunQuery) ([]RunRecord, error) {
			jobHistoryQuery = query
			return []RunRecord{sampleAutomationRunRecord()}, nil
		},
		getAutomationTriggerFn: func(context.Context, string) (TriggerRecord, error) {
			return sampleAutomationTriggerRecord(), nil
		},
		updateAutomationTriggerFn: func(_ context.Context, _ string, request AutomationTriggerUpdateRequest) (TriggerRecord, error) {
			updateTriggerRequest = request
			updated := sampleAutomationTriggerRecord()
			updated.Prompt = *request.Prompt
			updated.Enabled = *request.Enabled
			return updated, nil
		},
		deleteAutomationTriggerFn: func(context.Context, string) error {
			return nil
		},
		automationTriggerRunsFn: func(context.Context, string, AutomationRunQuery) ([]RunRecord, error) {
			return []RunRecord{sampleAutomationRunRecord()}, nil
		},
		listAutomationRunsFn: func(_ context.Context, query AutomationRunQuery) ([]RunRecord, error) {
			runsQuery = query
			return []RunRecord{sampleAutomationRunRecord()}, nil
		},
		getAutomationRunFn: func(context.Context, string) (RunRecord, error) {
			return sampleAutomationRunRecord(), nil
		},
	})

	stdout, _, err := executeRootCommand(
		t,
		deps,
		"automation",
		"triggers",
		"--scope",
		"workspace",
		"--workspace",
		"alpha",
		"--event",
		"webhook",
		"--source",
		"dynamic",
		"--last",
		"2",
		"-o",
		"json",
	)
	if err != nil {
		t.Fatalf("automation triggers list error = %v", err)
	}
	var listed contract.TriggersResponse
	if err := json.Unmarshal([]byte(stdout), &listed); err != nil {
		t.Fatalf("json.Unmarshal(trigger list) error = %v", err)
	}
	if len(listed.Triggers) != 1 || listTriggerQuery.WorkspaceID != "ws-alpha" ||
		listTriggerQuery.Scope != automationpkg.AutomationScopeWorkspace ||
		listTriggerQuery.Event != "webhook" ||
		listTriggerQuery.Source != automationpkg.JobSourceDynamic ||
		listTriggerQuery.Limit != 2 {
		t.Fatalf("listTriggerQuery = %#v, want resolved workspace/event/source/limit", listTriggerQuery)
	}

	jobHuman, _, err := executeRootCommand(t, deps, "automation", "jobs", "get", "job-1", "-o", "human")
	if err != nil {
		t.Fatalf("automation jobs get human error = %v", err)
	}
	if !strings.Contains(jobHuman, "Automation Job") {
		t.Fatalf("job human output = %q, want Automation Job section", jobHuman)
	}

	deletedJob, _, err := executeRootCommand(t, deps, "automation", "jobs", "delete", "job-1", "-o", "json")
	if err != nil {
		t.Fatalf("automation jobs delete error = %v", err)
	}
	var deleted JobRecord
	if err := json.Unmarshal([]byte(deletedJob), &deleted); err != nil {
		t.Fatalf("json.Unmarshal(job delete) error = %v", err)
	}
	if deleted.ID != "job-1" {
		t.Fatalf("deleted job = %#v, want job-1", deleted)
	}

	triggeredRun, _, err := executeRootCommand(t, deps, "automation", "jobs", "trigger", "job-1", "-o", "json")
	if err != nil {
		t.Fatalf("automation jobs trigger error = %v", err)
	}
	var run RunRecord
	if err := json.Unmarshal([]byte(triggeredRun), &run); err != nil {
		t.Fatalf("json.Unmarshal(job trigger) error = %v", err)
	}
	if run.ID != "run-1" {
		t.Fatalf("triggered run = %#v, want run-1", run)
	}

	_, _, err = executeRootCommand(
		t,
		deps,
		"automation",
		"jobs",
		"history",
		"job-1",
		"--status",
		"completed",
		"--since",
		"1h",
		"--until",
		"30m",
		"--last",
		"2",
		"-o",
		"json",
	)
	if err != nil {
		t.Fatalf("automation jobs history error = %v", err)
	}
	if jobHistoryQuery.Status != automationpkg.RunCompleted || jobHistoryQuery.Limit != 2 ||
		jobHistoryQuery.Since.IsZero() ||
		jobHistoryQuery.Until.IsZero() {
		t.Fatalf("jobHistoryQuery = %#v, want parsed run filters", jobHistoryQuery)
	}

	updatedTriggerJSON, _, err := executeRootCommand(
		t,
		deps,
		"automation", "triggers", "update", "trg-1",
		"--prompt", `inspect {{ index .Data "session_id" }}`,
		"--event", "webhook",
		"--filter", "data.branch=main",
		"--retry", "backoff",
		"--enabled=false",
		"--webhook-id", "wbh_123",
		"--endpoint-slug", "branch-review",
		"--webhook-secret-value", "shared-secret",
		"-o", "json",
	)
	if err != nil {
		t.Fatalf("automation triggers update error = %v", err)
	}
	var updated TriggerRecord
	if err := json.Unmarshal([]byte(updatedTriggerJSON), &updated); err != nil {
		t.Fatalf("json.Unmarshal(trigger update) error = %v", err)
	}
	if updated.ID != "trg-1" ||
		updateTriggerRequest.Event == nil ||
		*updateTriggerRequest.Event != "webhook" ||
		updateTriggerRequest.Retry == nil ||
		updateTriggerRequest.Retry.Strategy != automationpkg.RetryStrategyBackoff ||
		updateTriggerRequest.Filter["data.branch"] != "main" ||
		updateTriggerRequest.WebhookID == nil ||
		*updateTriggerRequest.WebhookID != "wbh_123" ||
		updateTriggerRequest.EndpointSlug == nil ||
		*updateTriggerRequest.EndpointSlug != "branch-review" ||
		updateTriggerRequest.WebhookSecretValue == nil ||
		*updateTriggerRequest.WebhookSecretValue != "shared-secret" {
		t.Fatalf("updateTriggerRequest = %#v, want parsed trigger update request", updateTriggerRequest)
	}

	deletedTriggerJSON, _, err := executeRootCommand(t, deps, "automation", "triggers", "delete", "trg-1", "-o", "json")
	if err != nil {
		t.Fatalf("automation triggers delete error = %v", err)
	}
	var deletedTrigger TriggerRecord
	if err := json.Unmarshal([]byte(deletedTriggerJSON), &deletedTrigger); err != nil {
		t.Fatalf("json.Unmarshal(trigger delete) error = %v", err)
	}
	if deletedTrigger.ID != "trg-1" {
		t.Fatalf("deleted trigger = %#v, want trg-1", deletedTrigger)
	}

	triggerHistoryHuman, _, err := executeRootCommand(
		t,
		deps,
		"automation",
		"triggers",
		"history",
		"trg-1",
		"--status",
		"completed",
		"--last",
		"1",
		"-o",
		"human",
	)
	if err != nil {
		t.Fatalf("automation triggers history human error = %v", err)
	}
	if !strings.Contains(triggerHistoryHuman, "Automation Runs") {
		t.Fatalf("trigger history human output = %q, want Automation Runs table", triggerHistoryHuman)
	}

	_, _, err = executeRootCommand(
		t,
		deps,
		"automation",
		"runs",
		"--job-id",
		"job-1",
		"--trigger-id",
		"trg-1",
		"--status",
		"completed",
		"--since",
		"1h",
		"--until",
		"30m",
		"--last",
		"4",
		"-o",
		"json",
	)
	if err != nil {
		t.Fatalf("automation runs list error = %v", err)
	}
	if runsQuery.JobID != "job-1" || runsQuery.TriggerID != "trg-1" || runsQuery.Status != automationpkg.RunCompleted ||
		runsQuery.Limit != 4 {
		t.Fatalf("runsQuery = %#v, want parsed runs query", runsQuery)
	}

	runHuman, _, err := executeRootCommand(t, deps, "automation", "runs", "get", "run-1", "-o", "human")
	if err != nil {
		t.Fatalf("automation runs get human error = %v", err)
	}
	if !strings.Contains(runHuman, "Automation Run") {
		t.Fatalf("run human output = %q, want Automation Run section", runHuman)
	}
}

func TestAutomationHelperFormattingAndParsing(t *testing.T) {
	t.Parallel()

	normalized, err := normalizeAutomationAtTime("2026-04-15T15:00")
	if err != nil {
		t.Fatalf("normalizeAutomationAtTime() error = %v", err)
	}
	if normalized != "2026-04-15T15:00:00Z" {
		t.Fatalf("normalizeAutomationAtTime() = %q, want %q", normalized, "2026-04-15T15:00:00Z")
	}

	if _, err := normalizeAutomationAtTime("bad"); err == nil {
		t.Fatal("normalizeAutomationAtTime(bad) error = nil, want non-nil")
	}

	since, err := parseAutomationOptionalTimeFlag("1h", "since", func() time.Time { return fixedTestNow })
	if err != nil {
		t.Fatalf("parseAutomationOptionalTimeFlag() error = %v", err)
	}
	if want := fixedTestNow.Add(-time.Hour); !since.Equal(want) {
		t.Fatalf("parseAutomationOptionalTimeFlag() = %v, want %v", since, want)
	}

	if _, err := parseAutomationOptionalTimeFlag("-1h", "since", func() time.Time { return fixedTestNow }); err == nil {
		t.Fatal("parseAutomationOptionalTimeFlag(-1h) error = nil, want non-nil")
	}

	if source, err := parseOptionalAutomationSource("dynamic"); err != nil || source != automationpkg.JobSourceDynamic {
		t.Fatalf("parseOptionalAutomationSource() = %q, %v", source, err)
	}
	if status, err := parseOptionalAutomationRunStatus(
		"completed",
	); err != nil ||
		status != automationpkg.RunCompleted {
		t.Fatalf("parseOptionalAutomationRunStatus() = %q, %v", status, err)
	}

	triggerListHuman, err := automationTriggerListBundle([]TriggerRecord{sampleAutomationTriggerRecord()}).human()
	if err != nil {
		t.Fatalf("automationTriggerListBundle().human() error = %v", err)
	}
	if !strings.Contains(triggerListHuman, "Automation Triggers") {
		t.Fatalf("trigger list human output = %q, want Automation Triggers title", triggerListHuman)
	}

	runBundleHuman, err := automationRunBundle(sampleAutomationRunRecord()).human()
	if err != nil {
		t.Fatalf("automationRunBundle().human() error = %v", err)
	}
	if !strings.Contains(runBundleHuman, "Automation Run") || !strings.Contains(runBundleHuman, "job:job-1") {
		t.Fatalf("run bundle human output = %q, want Automation Run details", runBundleHuman)
	}

	if got := formatAutomationSchedule(
		&automationpkg.ScheduleSpec{Mode: automationpkg.ScheduleModeCron, Expr: "0 9 * * *"},
	); got != "cron:0 9 * * *" {
		t.Fatalf("formatAutomationSchedule(cron) = %q, want cron prefix", got)
	}
	if got := displayTriggerEndpoint(
		sampleAutomationTriggerRecord(),
	); !strings.Contains(
		got,
		"/api/webhooks/workspaces/ws-alpha/",
	) {
		t.Fatalf("displayTriggerEndpoint() = %q, want workspace webhook path", got)
	}
	if got := displayRunTarget(sampleAutomationRunRecord()); got != "job:job-1" {
		t.Fatalf("displayRunTarget() = %q, want job target", got)
	}
	if ptr := new(true); ptr == nil || !*ptr {
		t.Fatalf("new(true) = %#v, want true pointer", ptr)
	}
}

func sampleAutomationJobRecord() JobRecord {
	nextRun := fixedTestNow.Add(time.Hour)
	return JobRecord{
		ID:          "job-1",
		Scope:       automationpkg.AutomationScopeWorkspace,
		Name:        "nightly-review",
		AgentName:   "coder",
		WorkspaceID: "ws-alpha",
		Prompt:      "review repo",
		Schedule: &automationpkg.ScheduleSpec{
			Mode:     automationpkg.ScheduleModeEvery,
			Interval: "30m",
		},
		Enabled:   true,
		Retry:     automationpkg.DefaultRetryConfig(),
		FireLimit: automationpkg.DefaultFireLimitConfig(),
		Source:    automationpkg.JobSourceDynamic,
		CreatedAt: fixedTestNow,
		UpdatedAt: fixedTestNow,
		NextRun:   &nextRun,
	}
}

func sampleAutomationTriggerRecord() TriggerRecord {
	return TriggerRecord{
		ID:           "trg-1",
		Scope:        automationpkg.AutomationScopeWorkspace,
		Name:         "branch-review",
		AgentName:    "coder",
		WorkspaceID:  "ws-alpha",
		Prompt:       `review {{ index .Data "session_id" }}`,
		Event:        "webhook",
		Filter:       map[string]string{"data.branch": "main"},
		Enabled:      true,
		Retry:        automationpkg.DefaultRetryConfig(),
		FireLimit:    automationpkg.DefaultFireLimitConfig(),
		Source:       automationpkg.JobSourceDynamic,
		WebhookID:    "wbh_123",
		EndpointSlug: "branch-review",
		CreatedAt:    fixedTestNow,
		UpdatedAt:    fixedTestNow,
	}
}

func sampleAutomationRunRecord() RunRecord {
	started := fixedTestNow
	ended := fixedTestNow.Add(2 * time.Minute)
	return RunRecord{
		ID:        "run-1",
		JobID:     "job-1",
		SessionID: "sess-1",
		Status:    automationpkg.RunCompleted,
		Attempt:   1,
		StartedAt: &started,
		EndedAt:   &ended,
	}
}
