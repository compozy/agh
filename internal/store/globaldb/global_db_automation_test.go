package globaldb

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/automation"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/testutil"
	aghworkspace "github.com/pedronauck/agh/internal/workspace"
)

type Job = automation.Job
type JobListQuery = automation.JobListQuery
type Trigger = automation.Trigger
type TriggerListQuery = automation.TriggerListQuery
type Run = automation.Run
type RunQuery = automation.RunQuery
type JobEnabledOverlay = automation.JobEnabledOverlay
type TriggerEnabledOverlay = automation.TriggerEnabledOverlay

func TestOpenGlobalDBCreatesAutomationSchemaAndIndexes(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)

	assertTablesPresent(
		t,
		globalDB.db,
		"automation_jobs",
		"automation_triggers",
		"automation_runs",
		"automation_job_overlays",
		"automation_trigger_overlays",
	)
	assertTableColumns(t, globalDB.db, "automation_jobs", []string{
		"id",
		"scope",
		"name",
		"agent_name",
		"workspace_id",
		"prompt",
		"schedule",
		"enabled",
		"retry",
		"fire_limit",
		"source",
		"created_at",
		"updated_at",
	})
	assertTableColumns(t, globalDB.db, "automation_triggers", []string{
		"id",
		"scope",
		"name",
		"agent_name",
		"workspace_id",
		"prompt",
		"event",
		"filter",
		"enabled",
		"retry",
		"fire_limit",
		"source",
		"webhook_id",
		"endpoint_slug",
		"created_at",
		"updated_at",
	})
	assertTableColumns(t, globalDB.db, "automation_runs", []string{
		"id",
		"job_id",
		"trigger_id",
		"session_id",
		"status",
		"attempt",
		"started_at",
		"ended_at",
		"error",
	})
	assertIndexesPresent(t, globalDB.db, "automation_jobs",
		"uq_automation_jobs_global_name",
		"uq_automation_jobs_workspace_name",
		"idx_automation_jobs_enabled",
	)
	assertIndexesPresent(t, globalDB.db, "automation_triggers",
		"uq_automation_triggers_global_name",
		"uq_automation_triggers_workspace_name",
		"uq_automation_triggers_webhook_id",
		"idx_automation_triggers_enabled",
		"idx_automation_triggers_event",
	)
	assertIndexesPresent(t, globalDB.db, "automation_runs",
		"idx_automation_runs_job",
		"idx_automation_runs_trigger",
		"idx_automation_runs_status",
		"idx_automation_runs_started",
	)
}

func TestGlobalDBCreateJobScopeAwareUniqueness(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	workspaceA := registerWorkspaceForGlobalTests(t, globalDB, "automation-uniqueness-a", t.TempDir())
	workspaceB := registerWorkspaceForGlobalTests(t, globalDB, "automation-uniqueness-b", t.TempDir())

	if _, err := globalDB.CreateJob(testutil.Context(t), automationJobForTest(automation.AutomationScopeGlobal, "daily-report", "", automation.JobSourceDynamic)); err != nil {
		t.Fatalf("CreateJob(global) error = %v", err)
	}
	if _, err := globalDB.CreateJob(testutil.Context(t), automationJobForTest(automation.AutomationScopeWorkspace, "daily-report", workspaceA, automation.JobSourceDynamic)); err != nil {
		t.Fatalf("CreateJob(workspace same name) error = %v", err)
	}
	if _, err := globalDB.CreateJob(testutil.Context(t), automationJobForTest(automation.AutomationScopeWorkspace, "daily-report", workspaceB, automation.JobSourceDynamic)); err != nil {
		t.Fatalf("CreateJob(second workspace same name) error = %v", err)
	}

	if _, err := globalDB.CreateJob(testutil.Context(t), automationJobForTest(automation.AutomationScopeGlobal, "daily-report", "", automation.JobSourceDynamic)); !errors.Is(err, automation.ErrJobNameTaken) {
		t.Fatalf("CreateJob(duplicate global) error = %v, want ErrJobNameTaken", err)
	}
	if _, err := globalDB.CreateJob(testutil.Context(t), automationJobForTest(automation.AutomationScopeWorkspace, "daily-report", workspaceA, automation.JobSourceDynamic)); !errors.Is(err, automation.ErrJobNameTaken) {
		t.Fatalf("CreateJob(duplicate workspace) error = %v, want ErrJobNameTaken", err)
	}
}

func TestGlobalDBGetTriggerByWebhookIDUsesStableID(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	trigger := automationWebhookTriggerForTest(automation.AutomationScopeGlobal, "deploy-review", "", automation.JobSourceDynamic)
	trigger.WebhookID = "webhook-stable-001"
	trigger.EndpointSlug = "deploy-review-v1"

	created, err := globalDB.CreateTrigger(testutil.Context(t), trigger)
	if err != nil {
		t.Fatalf("CreateTrigger() error = %v", err)
	}

	created.EndpointSlug = "deploy-review-renamed"
	updated, err := globalDB.UpdateTrigger(testutil.Context(t), created)
	if err != nil {
		t.Fatalf("UpdateTrigger() error = %v", err)
	}

	lookedUp, err := globalDB.GetTriggerByWebhookID(testutil.Context(t), "webhook-stable-001")
	if err != nil {
		t.Fatalf("GetTriggerByWebhookID() error = %v", err)
	}
	if got, want := lookedUp.ID, updated.ID; got != want {
		t.Fatalf("trigger.ID = %q, want %q", got, want)
	}
	if got, want := lookedUp.EndpointSlug, "deploy-review-renamed"; got != want {
		t.Fatalf("trigger.EndpointSlug = %q, want %q", got, want)
	}
	if got, want := lookedUp.WebhookID, "webhook-stable-001"; got != want {
		t.Fatalf("trigger.WebhookID = %q, want %q", got, want)
	}
}

func TestGlobalDBJobEnabledOverlayDoesNotMutateDefinition(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	workspaceID := registerWorkspaceForGlobalTests(t, globalDB, "automation-overlay-workspace", t.TempDir())

	configJob := automationJobForTest(automation.AutomationScopeWorkspace, "config-job", workspaceID, automation.JobSourceConfig)
	configJob.Enabled = true
	created, err := globalDB.CreateJob(testutil.Context(t), configJob)
	if err != nil {
		t.Fatalf("CreateJob(config) error = %v", err)
	}

	overlay, err := globalDB.SetJobEnabledOverlay(testutil.Context(t), JobEnabledOverlay{
		JobID:           created.ID,
		EnabledOverride: false,
	})
	if err != nil {
		t.Fatalf("SetJobEnabledOverlay() error = %v", err)
	}

	reloaded, err := globalDB.GetJob(testutil.Context(t), created.ID)
	if err != nil {
		t.Fatalf("GetJob() error = %v", err)
	}
	if !reloaded.Enabled {
		t.Fatal("GetJob().Enabled = false, want definition payload unchanged")
	}

	storedOverlay, err := globalDB.GetJobEnabledOverlay(testutil.Context(t), created.ID)
	if err != nil {
		t.Fatalf("GetJobEnabledOverlay() error = %v", err)
	}
	if storedOverlay.EnabledOverride {
		t.Fatal("GetJobEnabledOverlay().EnabledOverride = true, want false")
	}
	if overlay.UpdatedAt.IsZero() {
		t.Fatal("SetJobEnabledOverlay().UpdatedAt = zero, want populated")
	}

	overlays, err := globalDB.ListJobEnabledOverlays(testutil.Context(t))
	if err != nil {
		t.Fatalf("ListJobEnabledOverlays() error = %v", err)
	}
	if got, want := len(overlays), 1; got != want {
		t.Fatalf("len(ListJobEnabledOverlays()) = %d, want %d", got, want)
	}

	dynamicJob, err := globalDB.CreateJob(testutil.Context(t), automationJobForTest(automation.AutomationScopeWorkspace, "dynamic-job", workspaceID, automation.JobSourceDynamic))
	if err != nil {
		t.Fatalf("CreateJob(dynamic) error = %v", err)
	}
	if _, err := globalDB.SetJobEnabledOverlay(testutil.Context(t), JobEnabledOverlay{
		JobID:           dynamicJob.ID,
		EnabledOverride: false,
	}); !errors.Is(err, automation.ErrOverlayRequiresConfigSource) {
		t.Fatalf("SetJobEnabledOverlay(dynamic) error = %v, want ErrOverlayRequiresConfigSource", err)
	}

	if err := globalDB.DeleteJobEnabledOverlay(testutil.Context(t), created.ID); err != nil {
		t.Fatalf("DeleteJobEnabledOverlay() error = %v", err)
	}
	if _, err := globalDB.GetJobEnabledOverlay(testutil.Context(t), created.ID); !errors.Is(err, automation.ErrJobOverlayNotFound) {
		t.Fatalf("GetJobEnabledOverlay(after delete) error = %v, want ErrJobOverlayNotFound", err)
	}
}

func TestGlobalDBTriggerEnabledOverlayDoesNotMutateDefinition(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	workspaceID := registerWorkspaceForGlobalTests(t, globalDB, "automation-trigger-overlay-workspace", t.TempDir())

	configTrigger := automationWebhookTriggerForTest(automation.AutomationScopeWorkspace, "config-trigger", workspaceID, automation.JobSourceConfig)
	configTrigger.Enabled = true
	created, err := globalDB.CreateTrigger(testutil.Context(t), configTrigger)
	if err != nil {
		t.Fatalf("CreateTrigger(config) error = %v", err)
	}

	if _, err := globalDB.SetTriggerEnabledOverlay(testutil.Context(t), TriggerEnabledOverlay{
		TriggerID:       created.ID,
		EnabledOverride: false,
	}); err != nil {
		t.Fatalf("SetTriggerEnabledOverlay() error = %v", err)
	}

	reloaded, err := globalDB.GetTrigger(testutil.Context(t), created.ID)
	if err != nil {
		t.Fatalf("GetTrigger() error = %v", err)
	}
	if !reloaded.Enabled {
		t.Fatal("GetTrigger().Enabled = false, want definition payload unchanged")
	}

	storedOverlay, err := globalDB.GetTriggerEnabledOverlay(testutil.Context(t), created.ID)
	if err != nil {
		t.Fatalf("GetTriggerEnabledOverlay() error = %v", err)
	}
	if storedOverlay.EnabledOverride {
		t.Fatal("GetTriggerEnabledOverlay().EnabledOverride = true, want false")
	}

	overlays, err := globalDB.ListTriggerEnabledOverlays(testutil.Context(t))
	if err != nil {
		t.Fatalf("ListTriggerEnabledOverlays() error = %v", err)
	}
	if got, want := len(overlays), 1; got != want {
		t.Fatalf("len(ListTriggerEnabledOverlays()) = %d, want %d", got, want)
	}

	dynamicTriggerDef := automationWebhookTriggerForTest(automation.AutomationScopeWorkspace, "dynamic-trigger", workspaceID, automation.JobSourceDynamic)
	dynamicTriggerDef.WebhookID = "dynamic-trigger-webhook"
	dynamicTriggerDef.EndpointSlug = "dynamic-trigger-endpoint"
	dynamicTrigger, err := globalDB.CreateTrigger(testutil.Context(t), dynamicTriggerDef)
	if err != nil {
		t.Fatalf("CreateTrigger(dynamic) error = %v", err)
	}
	if _, err := globalDB.SetTriggerEnabledOverlay(testutil.Context(t), TriggerEnabledOverlay{
		TriggerID:       dynamicTrigger.ID,
		EnabledOverride: false,
	}); !errors.Is(err, automation.ErrOverlayRequiresConfigSource) {
		t.Fatalf("SetTriggerEnabledOverlay(dynamic) error = %v, want ErrOverlayRequiresConfigSource", err)
	}

	if err := globalDB.DeleteTriggerEnabledOverlay(testutil.Context(t), created.ID); err != nil {
		t.Fatalf("DeleteTriggerEnabledOverlay() error = %v", err)
	}
	if _, err := globalDB.GetTriggerEnabledOverlay(testutil.Context(t), created.ID); !errors.Is(err, automation.ErrTriggerOverlayNotFound) {
		t.Fatalf("GetTriggerEnabledOverlay(after delete) error = %v, want ErrTriggerOverlayNotFound", err)
	}
}

func TestGlobalDBTriggerUniquenessAndWebhookIDConstraints(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	workspaceID := registerWorkspaceForGlobalTests(t, globalDB, "automation-trigger-constraints", t.TempDir())

	globalWebhook := automationWebhookTriggerForTest(automation.AutomationScopeGlobal, "deploy-review", "", automation.JobSourceDynamic)
	globalWebhook.WebhookID = "stable-webhook-a"
	if _, err := globalDB.CreateTrigger(testutil.Context(t), globalWebhook); err != nil {
		t.Fatalf("CreateTrigger(globalWebhook) error = %v", err)
	}

	workspaceTrigger := automationWebhookTriggerForTest(automation.AutomationScopeWorkspace, "deploy-review", workspaceID, automation.JobSourceDynamic)
	workspaceTrigger.WebhookID = "stable-webhook-b"
	if _, err := globalDB.CreateTrigger(testutil.Context(t), workspaceTrigger); err != nil {
		t.Fatalf("CreateTrigger(workspace same name) error = %v", err)
	}

	duplicateName := automationWebhookTriggerForTest(automation.AutomationScopeGlobal, "deploy-review", "", automation.JobSourceDynamic)
	duplicateName.WebhookID = "stable-webhook-c"
	if _, err := globalDB.CreateTrigger(testutil.Context(t), duplicateName); !errors.Is(err, automation.ErrTriggerNameTaken) {
		t.Fatalf("CreateTrigger(duplicate name) error = %v, want ErrTriggerNameTaken", err)
	}

	duplicateWebhook := automationWebhookTriggerForTest(automation.AutomationScopeWorkspace, "another-name", workspaceID, automation.JobSourceDynamic)
	duplicateWebhook.WebhookID = "stable-webhook-a"
	if _, err := globalDB.CreateTrigger(testutil.Context(t), duplicateWebhook); !errors.Is(err, automation.ErrTriggerWebhookIDTaken) {
		t.Fatalf("CreateTrigger(duplicate webhook) error = %v, want ErrTriggerWebhookIDTaken", err)
	}

	filtered, err := globalDB.ListTriggers(testutil.Context(t), TriggerListQuery{
		Scope:  automation.AutomationScopeWorkspace,
		Event:  "webhook",
		Source: automation.JobSourceDynamic,
	})
	if err != nil {
		t.Fatalf("ListTriggers(filtered) error = %v", err)
	}
	if got, want := len(filtered), 1; got != want {
		t.Fatalf("len(ListTriggers(filtered)) = %d, want %d", got, want)
	}
}

func TestGlobalDBAutomationValidationAndDeleteBehavior(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)

	if _, err := globalDB.ListJobs(testutil.Context(t), JobListQuery{
		Scope:       automation.AutomationScopeGlobal,
		WorkspaceID: "ws-ignored",
	}); err == nil {
		t.Fatal("ListJobs(invalid scope/workspace filter) error = nil, want non-nil")
	}
	if _, err := globalDB.ListTriggers(testutil.Context(t), TriggerListQuery{
		Scope:       automation.AutomationScopeGlobal,
		WorkspaceID: "ws-ignored",
	}); err == nil {
		t.Fatal("ListTriggers(invalid scope/workspace filter) error = nil, want non-nil")
	}
	if _, err := globalDB.ListRuns(testutil.Context(t), RunQuery{
		Since: time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC),
		Until: time.Date(2026, 4, 10, 11, 0, 0, 0, time.UTC),
	}); err == nil {
		t.Fatal("ListRuns(invalid window) error = nil, want non-nil")
	}
	if _, err := globalDB.CreateRun(testutil.Context(t), Run{Status: automation.RunCompleted, Attempt: 1}); err == nil {
		t.Fatal("CreateRun(without job or trigger) error = nil, want non-nil")
	}

	if _, err := globalDB.CreateJob(testutil.Context(t), automationJobForTest(automation.AutomationScopeWorkspace, "missing-workspace-job", "ws-missing", automation.JobSourceDynamic)); !errors.Is(err, aghworkspace.ErrWorkspaceNotFound) {
		t.Fatalf("CreateJob(missing workspace) error = %v, want ErrWorkspaceNotFound", err)
	}
	if _, err := globalDB.CreateTrigger(testutil.Context(t), automationWebhookTriggerForTest(automation.AutomationScopeWorkspace, "missing-workspace-trigger", "ws-missing", automation.JobSourceDynamic)); !errors.Is(err, aghworkspace.ErrWorkspaceNotFound) {
		t.Fatalf("CreateTrigger(missing workspace) error = %v, want ErrWorkspaceNotFound", err)
	}

	workspaceID := registerWorkspaceForGlobalTests(t, globalDB, "automation-delete-workspace", t.TempDir())
	job, err := globalDB.CreateJob(testutil.Context(t), automationJobForTest(automation.AutomationScopeWorkspace, "delete-job", workspaceID, automation.JobSourceDynamic))
	if err != nil {
		t.Fatalf("CreateJob() error = %v", err)
	}
	job.Prompt = "Updated automation prompt."
	job.Schedule = &automation.ScheduleSpec{
		Mode: automation.ScheduleModeCron,
		Expr: "0 * * * *",
	}
	job, err = globalDB.UpdateJob(testutil.Context(t), job)
	if err != nil {
		t.Fatalf("UpdateJob() error = %v", err)
	}
	loadedJob, err := globalDB.GetJob(testutil.Context(t), job.ID)
	if err != nil {
		t.Fatalf("GetJob() error = %v", err)
	}
	if got, want := loadedJob.Prompt, "Updated automation prompt."; got != want {
		t.Fatalf("GetJob().Prompt = %q, want %q", got, want)
	}
	trigger, err := globalDB.CreateTrigger(testutil.Context(t), automationNonWebhookTriggerForTest(automation.AutomationScopeWorkspace, "delete-trigger", workspaceID, automation.JobSourceDynamic))
	if err != nil {
		t.Fatalf("CreateTrigger() error = %v", err)
	}
	run, err := globalDB.CreateRun(testutil.Context(t), automationRunForJob(job.ID, automation.RunRunning, 1, time.Date(2026, 4, 10, 22, 0, 0, 0, time.UTC)))
	if err != nil {
		t.Fatalf("CreateRun() error = %v", err)
	}

	run.Status = automation.RunFailed
	run.Attempt = 2
	run.SessionID = "sess-updated"
	updatedRun, err := globalDB.UpdateRun(testutil.Context(t), run)
	if err != nil {
		t.Fatalf("UpdateRun() error = %v", err)
	}
	if got, want := updatedRun.Status, automation.RunFailed; got != want {
		t.Fatalf("UpdateRun().Status = %q, want %q", got, want)
	}
	if got, want := updatedRun.SessionID, "sess-updated"; got != want {
		t.Fatalf("UpdateRun().SessionID = %q, want %q", got, want)
	}
	loadedRun, err := globalDB.GetRun(testutil.Context(t), run.ID)
	if err != nil {
		t.Fatalf("GetRun() error = %v", err)
	}
	if got, want := loadedRun.Attempt, 2; got != want {
		t.Fatalf("GetRun().Attempt = %d, want %d", got, want)
	}

	jobs, err := globalDB.ListJobs(testutil.Context(t), JobListQuery{Scope: automation.AutomationScopeWorkspace, Source: automation.JobSourceDynamic})
	if err != nil {
		t.Fatalf("ListJobs(filtered) error = %v", err)
	}
	if got, want := len(jobs), 1; got != want {
		t.Fatalf("len(ListJobs(filtered)) = %d, want %d", got, want)
	}

	if err := globalDB.DeleteRun(testutil.Context(t), run.ID); err != nil {
		t.Fatalf("DeleteRun() error = %v", err)
	}
	if _, err := globalDB.GetRun(testutil.Context(t), run.ID); !errors.Is(err, automation.ErrRunNotFound) {
		t.Fatalf("GetRun(after delete) error = %v, want ErrRunNotFound", err)
	}
	if err := globalDB.DeleteRun(testutil.Context(t), run.ID); !errors.Is(err, automation.ErrRunNotFound) {
		t.Fatalf("DeleteRun(missing) error = %v, want ErrRunNotFound", err)
	}
	if err := globalDB.DeleteJob(testutil.Context(t), job.ID); err != nil {
		t.Fatalf("DeleteJob() error = %v", err)
	}
	if _, err := globalDB.GetJob(testutil.Context(t), job.ID); !errors.Is(err, automation.ErrJobNotFound) {
		t.Fatalf("GetJob(after delete) error = %v, want ErrJobNotFound", err)
	}
	if err := globalDB.DeleteJob(testutil.Context(t), job.ID); !errors.Is(err, automation.ErrJobNotFound) {
		t.Fatalf("DeleteJob(missing) error = %v, want ErrJobNotFound", err)
	}
	if err := globalDB.DeleteTrigger(testutil.Context(t), trigger.ID); err != nil {
		t.Fatalf("DeleteTrigger() error = %v", err)
	}
	if _, err := globalDB.GetTrigger(testutil.Context(t), trigger.ID); !errors.Is(err, automation.ErrTriggerNotFound) {
		t.Fatalf("GetTrigger(after delete) error = %v, want ErrTriggerNotFound", err)
	}
	if err := globalDB.DeleteTrigger(testutil.Context(t), trigger.ID); !errors.Is(err, automation.ErrTriggerNotFound) {
		t.Fatalf("DeleteTrigger(missing) error = %v, want ErrTriggerNotFound", err)
	}
}

func TestAutomationStoreHelperBranches(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	base := time.Date(2026, 4, 10, 23, 0, 0, 0, time.UTC)
	globalDB.now = func() time.Time { return base }

	job, err := globalDB.normalizeJobForCreate(Job{
		Scope:     automation.AutomationScopeGlobal,
		Name:      "helper-job",
		AgentName: "agent",
		Prompt:    "prompt",
		Schedule: &automation.ScheduleSpec{
			Mode:     automation.ScheduleModeEvery,
			Interval: "15m",
		},
		Enabled:   true,
		Retry:     automation.DefaultRetryConfig(),
		FireLimit: automation.DefaultFireLimitConfig(),
	})
	if err != nil {
		t.Fatalf("normalizeJobForCreate() error = %v", err)
	}
	if got, want := job.Source, automation.JobSourceDynamic; got != want {
		t.Fatalf("normalizeJobForCreate().Source = %q, want %q", got, want)
	}
	if job.ID == "" || !job.CreatedAt.Equal(base) || !job.UpdatedAt.Equal(base) {
		t.Fatalf("normalizeJobForCreate() = %#v, want generated id and base timestamps", job)
	}
	if _, err := globalDB.normalizeJobForUpdate(Job{}); err == nil {
		t.Fatal("normalizeJobForUpdate(empty) error = nil, want non-nil")
	}

	trigger, err := globalDB.normalizeTriggerForCreate(Trigger{
		Scope:     automation.AutomationScopeGlobal,
		Name:      "helper-trigger",
		AgentName: "agent",
		Prompt:    `{{ index .Data "payload" }}`,
		Event:     "webhook",
		Enabled:   true,
		Retry:     automation.DefaultRetryConfig(),
		FireLimit: automation.DefaultFireLimitConfig(),
		WebhookID: "helper-webhook",
	})
	if err != nil {
		t.Fatalf("normalizeTriggerForCreate() error = %v", err)
	}
	if got, want := trigger.Source, automation.JobSourceDynamic; got != want {
		t.Fatalf("normalizeTriggerForCreate().Source = %q, want %q", got, want)
	}
	if _, err := globalDB.normalizeTriggerForUpdate(Trigger{}); err == nil {
		t.Fatal("normalizeTriggerForUpdate(empty) error = nil, want non-nil")
	}

	run, err := globalDB.normalizeRunForCreate(Run{
		JobID:  "job-1",
		Status: automation.RunRunning,
	})
	if err != nil {
		t.Fatalf("normalizeRunForCreate() error = %v", err)
	}
	if got, want := run.Attempt, 1; got != want {
		t.Fatalf("normalizeRunForCreate().Attempt = %d, want %d", got, want)
	}
	if _, err := globalDB.normalizeRunForCreate(Run{
		JobID:   "job-1",
		Status:  automation.RunRunning,
		Attempt: -1,
	}); err == nil {
		t.Fatal("normalizeRunForCreate(negative attempt) error = nil, want non-nil")
	}
	if _, err := globalDB.normalizeRunForUpdate(Run{
		ID:      "run-1",
		JobID:   "job-1",
		Status:  automation.RunRunning,
		Attempt: 0,
	}); err == nil {
		t.Fatal("normalizeRunForUpdate(zero attempt) error = nil, want non-nil")
	}

	if _, err := requireAutomationID("", "automation id"); err == nil {
		t.Fatal("requireAutomationID(empty) error = nil, want non-nil")
	}
	if got, err := requireAutomationID(" job-1 ", "automation id"); err != nil || got != "job-1" {
		t.Fatalf("requireAutomationID(trimmed) = (%q, %v), want (job-1, nil)", got, err)
	}

	if err := validateAutomationRunQuery(RunQuery{Status: "bad-status"}); err == nil {
		t.Fatal("validateAutomationRunQuery(bad status) error = nil, want non-nil")
	}
	if err := validateAutomationRunRecord(Run{
		JobID:     "job-1",
		TriggerID: "trg-1",
		Status:    automation.RunRunning,
		Attempt:   1,
	}); err == nil {
		t.Fatal("validateAutomationRunRecord(both job and trigger) error = nil, want non-nil")
	}

	var schedule *automation.ScheduleSpec
	if err := decodeAutomationSchedule(sql.NullString{Valid: true, String: `{"mode":"cron","expr":"0 * * * *"}`}, &schedule); err != nil {
		t.Fatalf("decodeAutomationSchedule(valid) error = %v", err)
	}
	if schedule == nil || schedule.Mode != automation.ScheduleModeCron {
		t.Fatalf("decodeAutomationSchedule(valid) = %#v, want cron schedule", schedule)
	}
	if err := decodeAutomationSchedule(sql.NullString{Valid: true, String: `{`}, &schedule); err == nil {
		t.Fatal("decodeAutomationSchedule(invalid) error = nil, want non-nil")
	}

	var filter map[string]string
	if err := decodeAutomationFilter(sql.NullString{Valid: true, String: `{"data.kind":"ready"}`}, &filter); err != nil {
		t.Fatalf("decodeAutomationFilter(valid) error = %v", err)
	}
	if got, want := filter["data.kind"], "ready"; got != want {
		t.Fatalf("decodeAutomationFilter(valid) = %#v, want ready value", filter)
	}
	if err := decodeAutomationFilter(sql.NullString{Valid: true, String: `{`}, &filter); err == nil {
		t.Fatal("decodeAutomationFilter(invalid) error = nil, want non-nil")
	}

	if got := nullableAutomationTimestamp(nil); got != nil {
		t.Fatalf("nullableAutomationTimestamp(nil) = %#v, want nil", got)
	}
	if got, want := nullableAutomationTimestamp(timePointer(base)), any(store.FormatTimestamp(base)); got != want {
		t.Fatalf("nullableAutomationTimestamp(value) = %#v, want %#v", got, want)
	}

	if err := mapAutomationJobConstraintError(errors.New("UNIQUE constraint failed: automation_jobs.name")); !errors.Is(err, automation.ErrJobNameTaken) {
		t.Fatalf("mapAutomationJobConstraintError(name) = %v, want ErrJobNameTaken", err)
	}
	if err := mapAutomationJobConstraintError(errors.New("FOREIGN KEY constraint failed")); !errors.Is(err, aghworkspace.ErrWorkspaceNotFound) {
		t.Fatalf("mapAutomationJobConstraintError(fk) = %v, want ErrWorkspaceNotFound", err)
	}
	if err := mapAutomationTriggerConstraintError(errors.New("UNIQUE constraint failed: automation_triggers.webhook_id")); !errors.Is(err, automation.ErrTriggerWebhookIDTaken) {
		t.Fatalf("mapAutomationTriggerConstraintError(webhook) = %v, want ErrTriggerWebhookIDTaken", err)
	}
}

func TestGlobalDBLookupAutomationSources(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	workspaceID := registerWorkspaceForGlobalTests(t, globalDB, "automation-source-workspace", t.TempDir())
	job, err := globalDB.CreateJob(testutil.Context(t), automationJobForTest(automation.AutomationScopeWorkspace, "source-job", workspaceID, automation.JobSourceConfig))
	if err != nil {
		t.Fatalf("CreateJob() error = %v", err)
	}
	trigger, err := globalDB.CreateTrigger(testutil.Context(t), automationWebhookTriggerForTest(automation.AutomationScopeWorkspace, "source-trigger", workspaceID, automation.JobSourceConfig))
	if err != nil {
		t.Fatalf("CreateTrigger() error = %v", err)
	}

	jobSource, err := globalDB.lookupJobSource(testutil.Context(t), job.ID)
	if err != nil {
		t.Fatalf("lookupJobSource() error = %v", err)
	}
	if got, want := jobSource, automation.JobSourceConfig; got != want {
		t.Fatalf("lookupJobSource() = %q, want %q", got, want)
	}
	triggerSource, err := globalDB.lookupTriggerSource(testutil.Context(t), trigger.ID)
	if err != nil {
		t.Fatalf("lookupTriggerSource() error = %v", err)
	}
	if got, want := triggerSource, automation.JobSourceConfig; got != want {
		t.Fatalf("lookupTriggerSource() = %q, want %q", got, want)
	}

	if _, err := globalDB.lookupJobSource(testutil.Context(t), "missing-job"); !errors.Is(err, automation.ErrJobNotFound) {
		t.Fatalf("lookupJobSource(missing) error = %v, want ErrJobNotFound", err)
	}
	if _, err := globalDB.lookupTriggerSource(testutil.Context(t), "missing-trigger"); !errors.Is(err, automation.ErrTriggerNotFound) {
		t.Fatalf("lookupTriggerSource(missing) error = %v, want ErrTriggerNotFound", err)
	}
}

func TestAutomationJSONHelperFunctions(t *testing.T) {
	t.Parallel()

	encoded, err := encodeAutomationJSON(automation.DefaultFireLimitConfig(), "fire_limit")
	if err != nil {
		t.Fatalf("encodeAutomationJSON() error = %v", err)
	}
	if encoded == "" {
		t.Fatal("encodeAutomationJSON() = empty, want JSON")
	}

	optional, err := encodeOptionalAutomationJSON(map[string]string{"key": "value"}, false, "filter")
	if err != nil {
		t.Fatalf("encodeOptionalAutomationJSON(non-empty) error = %v", err)
	}
	if optional == nil {
		t.Fatal("encodeOptionalAutomationJSON(non-empty) = nil, want JSON payload")
	}
	optional, err = encodeOptionalAutomationJSON(map[string]string{}, true, "filter")
	if err != nil {
		t.Fatalf("encodeOptionalAutomationJSON(empty) error = %v", err)
	}
	if optional != nil {
		t.Fatalf("encodeOptionalAutomationJSON(empty) = %#v, want nil", optional)
	}

	var decoded automation.FireLimitConfig
	if err := decodeAutomationJSON(encoded, &decoded, "fire_limit"); err != nil {
		t.Fatalf("decodeAutomationJSON(valid) error = %v", err)
	}
	if got, want := decoded.Window, automation.DefaultFireLimitConfig().Window; got != want {
		t.Fatalf("decodeAutomationJSON(valid).Window = %q, want %q", got, want)
	}
	if err := decodeAutomationJSON("", &decoded, "fire_limit"); err == nil {
		t.Fatal("decodeAutomationJSON(empty) error = nil, want non-nil")
	}
}

func TestAutomationOverlayNormalizersAndQueryValidators(t *testing.T) {
	t.Parallel()

	if _, err := normalizeJobOverlay(JobEnabledOverlay{}, time.Date(2026, 4, 10, 23, 30, 0, 0, time.UTC)); err == nil {
		t.Fatal("normalizeJobOverlay(empty) error = nil, want non-nil")
	}
	if _, err := normalizeTriggerOverlay(TriggerEnabledOverlay{}, time.Date(2026, 4, 10, 23, 30, 0, 0, time.UTC)); err == nil {
		t.Fatal("normalizeTriggerOverlay(empty) error = nil, want non-nil")
	}
	if err := validateAutomationJobListQuery(JobListQuery{Source: "bad-source"}); err == nil {
		t.Fatal("validateAutomationJobListQuery(bad source) error = nil, want non-nil")
	}
	if err := validateAutomationTriggerListQuery(TriggerListQuery{Source: "bad-source"}); err == nil {
		t.Fatal("validateAutomationTriggerListQuery(bad source) error = nil, want non-nil")
	}
}

func TestGlobalDBListRunsFiltersByJobTriggerStatusAndTimeWindow(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	workspaceID := registerWorkspaceForGlobalTests(t, globalDB, "automation-runs-workspace", t.TempDir())
	jobA, err := globalDB.CreateJob(testutil.Context(t), automationJobForTest(automation.AutomationScopeWorkspace, "job-a", workspaceID, automation.JobSourceDynamic))
	if err != nil {
		t.Fatalf("CreateJob(jobA) error = %v", err)
	}
	jobB, err := globalDB.CreateJob(testutil.Context(t), automationJobForTest(automation.AutomationScopeWorkspace, "job-b", workspaceID, automation.JobSourceDynamic))
	if err != nil {
		t.Fatalf("CreateJob(jobB) error = %v", err)
	}
	trigger, err := globalDB.CreateTrigger(testutil.Context(t), automationNonWebhookTriggerForTest(automation.AutomationScopeWorkspace, "trigger-a", workspaceID, automation.JobSourceDynamic))
	if err != nil {
		t.Fatalf("CreateTrigger() error = %v", err)
	}

	base := time.Date(2026, 4, 10, 20, 0, 0, 0, time.UTC)
	for _, run := range []Run{
		automationRunForJob(jobA.ID, automation.RunCompleted, 1, base.Add(-10*time.Minute)),
		automationRunForJob(jobA.ID, automation.RunFailed, 2, base.Add(-5*time.Minute)),
		automationRunForTrigger(trigger.ID, automation.RunCompleted, 1, base.Add(-2*time.Minute)),
		automationRunForJob(jobB.ID, automation.RunCompleted, 1, base.Add(-90*time.Minute)),
	} {
		if _, err := globalDB.CreateRun(testutil.Context(t), run); err != nil {
			t.Fatalf("CreateRun(%q) error = %v", run.Status, err)
		}
	}

	jobRuns, err := globalDB.ListRuns(testutil.Context(t), RunQuery{
		JobID:  jobA.ID,
		Status: automation.RunFailed,
		Since:  base.Add(-6 * time.Minute),
		Until:  base,
	})
	if err != nil {
		t.Fatalf("ListRuns(job filter) error = %v", err)
	}
	if got, want := len(jobRuns), 1; got != want {
		t.Fatalf("len(jobRuns) = %d, want %d", got, want)
	}
	if got, want := jobRuns[0].Attempt, 2; got != want {
		t.Fatalf("jobRuns[0].Attempt = %d, want %d", got, want)
	}

	triggerRuns, err := globalDB.ListRuns(testutil.Context(t), RunQuery{
		TriggerID: trigger.ID,
		Status:    automation.RunCompleted,
		Since:     base.Add(-5 * time.Minute),
	})
	if err != nil {
		t.Fatalf("ListRuns(trigger filter) error = %v", err)
	}
	if got, want := len(triggerRuns), 1; got != want {
		t.Fatalf("len(triggerRuns) = %d, want %d", got, want)
	}
	if got, want := triggerRuns[0].TriggerID, trigger.ID; got != want {
		t.Fatalf("triggerRuns[0].TriggerID = %q, want %q", got, want)
	}

	count, err := globalDB.CountRuns(testutil.Context(t), RunQuery{
		JobID: jobA.ID,
		Since: base.Add(-30 * time.Minute),
	})
	if err != nil {
		t.Fatalf("CountRuns() error = %v", err)
	}
	if got, want := count, int64(2); got != want {
		t.Fatalf("CountRuns() = %d, want %d", got, want)
	}
}

func automationJobForTest(scope automation.AutomationScope, name string, workspaceID string, source automation.JobSource) Job {
	createdAt := time.Date(2026, 4, 10, 18, 0, 0, 0, time.UTC)
	return Job{
		Scope:       scope,
		Name:        name,
		AgentName:   "researcher",
		WorkspaceID: workspaceID,
		Prompt:      "Summarize the latest automation state.",
		Schedule: &automation.ScheduleSpec{
			Mode:     automation.ScheduleModeEvery,
			Interval: "30m",
		},
		Enabled:   true,
		Retry:     automation.DefaultRetryConfig(),
		FireLimit: automation.DefaultFireLimitConfig(),
		Source:    source,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}
}

func automationWebhookTriggerForTest(scope automation.AutomationScope, name string, workspaceID string, source automation.JobSource) Trigger {
	createdAt := time.Date(2026, 4, 10, 18, 5, 0, 0, time.UTC)
	return Trigger{
		Scope:        scope,
		Name:         name,
		AgentName:    "reviewer",
		WorkspaceID:  workspaceID,
		Prompt:       `Review webhook payload {{ index .Data "payload" }}`,
		Event:        "webhook",
		Enabled:      true,
		Retry:        automation.DefaultRetryConfig(),
		FireLimit:    automation.DefaultFireLimitConfig(),
		Source:       source,
		WebhookID:    "webhook-default",
		EndpointSlug: "endpoint-default",
		CreatedAt:    createdAt,
		UpdatedAt:    createdAt,
	}
}

func automationNonWebhookTriggerForTest(scope automation.AutomationScope, name string, workspaceID string, source automation.JobSource) Trigger {
	createdAt := time.Date(2026, 4, 10, 18, 10, 0, 0, time.UTC)
	return Trigger{
		Scope:       scope,
		Name:        name,
		AgentName:   "reviewer",
		WorkspaceID: workspaceID,
		Prompt:      `Summarize session {{ index .Data "session_id" }}`,
		Event:       "session.stopped",
		Filter: map[string]string{
			"data.stop_reason": "completed",
		},
		Enabled:   true,
		Retry:     automation.DefaultRetryConfig(),
		FireLimit: automation.DefaultFireLimitConfig(),
		Source:    source,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}
}

func automationRunForJob(jobID string, status automation.RunStatus, attempt int, startedAt time.Time) Run {
	endedAt := startedAt.Add(time.Minute)
	return Run{
		JobID:     jobID,
		Status:    status,
		Attempt:   attempt,
		StartedAt: timePointer(startedAt),
		EndedAt:   timePointer(endedAt),
	}
}

func automationRunForTrigger(triggerID string, status automation.RunStatus, attempt int, startedAt time.Time) Run {
	endedAt := startedAt.Add(time.Minute)
	return Run{
		TriggerID: triggerID,
		Status:    status,
		Attempt:   attempt,
		StartedAt: timePointer(startedAt),
		EndedAt:   timePointer(endedAt),
	}
}

func timePointer(value time.Time) *time.Time {
	timestamp := value
	return &timestamp
}

func assertIndexesPresent(t *testing.T, db *sql.DB, table string, want ...string) {
	t.Helper()

	rows, err := db.QueryContext(testutil.Context(t), "PRAGMA index_list("+table+")")
	if err != nil {
		t.Fatalf("QueryContext(index_list %q) error = %v", table, err)
	}
	defer func() {
		_ = rows.Close()
	}()

	got := make(map[string]struct{})
	for rows.Next() {
		var (
			seq     int
			name    string
			unique  int
			origin  string
			partial int
		)
		if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			t.Fatalf("Scan(index_list %q) error = %v", table, err)
		}
		got[name] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows.Err(index_list %q) error = %v", table, err)
	}

	for _, indexName := range want {
		if _, ok := got[indexName]; !ok {
			t.Fatalf("index %q missing from %s indexes %#v", indexName, table, got)
		}
	}
}
