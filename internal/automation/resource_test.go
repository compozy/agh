package automation

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/resources"
	taskpkg "github.com/pedronauck/agh/internal/task"
	"github.com/pedronauck/agh/internal/testutil"
	"github.com/pedronauck/agh/internal/vault"
)

func TestAutomationResourceCodecsRejectInvalidSpecs(t *testing.T) {
	t.Parallel()

	jobCodec, err := NewJobResourceCodec()
	if err != nil {
		t.Fatalf("NewJobResourceCodec() error = %v", err)
	}
	triggerCodec, err := NewTriggerResourceCodec()
	if err != nil {
		t.Fatalf("NewTriggerResourceCodec() error = %v", err)
	}

	ctx := testutil.Context(t)
	workspaceScope := resources.ResourceScope{Kind: resources.ResourceScopeKindWorkspace, ID: "ws-resource"}

	validJob := testJob(AutomationScopeWorkspace, "resource-job", "ws-resource")
	jobWithScopeMismatch := validJob
	jobWithScopeMismatch.WorkspaceID = "ws-other"
	if _, err := jobCodec.DecodeAndValidate(
		ctx,
		workspaceScope,
		mustAutomationJSON(t, jobWithScopeMismatch),
	); !errors.Is(
		err,
		resources.ErrInvalidScopeBinding,
	) {
		t.Fatalf("job scope mismatch error = %v, want ErrInvalidScopeBinding", err)
	} else if !strings.Contains(err.Error(), "automation: bind job resource scope") {
		t.Fatalf("job scope mismatch error = %v, want bind job resource scope context", err)
	}

	malformedJob := validJob
	malformedJob.Schedule = &ScheduleSpec{Mode: ScheduleModeEvery, Interval: "0s"}
	if _, err := jobCodec.DecodeAndValidate(ctx, workspaceScope, mustAutomationJSON(t, malformedJob)); err == nil {
		t.Fatal("job codec accepted malformed schedule")
	} else if !strings.Contains(err.Error(), "automation: validate job resource spec") {
		t.Fatalf("malformed job error = %v, want validate job resource spec context", err)
	}

	validTrigger := Trigger{
		Scope:       AutomationScopeWorkspace,
		Name:        "resource-trigger",
		AgentName:   "reviewer",
		WorkspaceID: "ws-resource",
		Prompt:      `Review {{ index .Data "session_id" }}`,
		Event:       "session.stopped",
		Enabled:     true,
		Retry:       DefaultRetryConfig(),
		FireLimit:   DefaultFireLimitConfig(),
		Source:      JobSourceDynamic,
	}
	triggerWithBadFilter := validTrigger
	triggerWithBadFilter.Filter = map[string]string{"unsupported": "value"}
	if _, err := triggerCodec.DecodeAndValidate(
		ctx,
		workspaceScope,
		mustAutomationJSON(t, triggerWithBadFilter),
	); err == nil {
		t.Fatal("trigger codec accepted malformed filter")
	} else if !strings.Contains(err.Error(), "automation: validate trigger resource spec") {
		t.Fatalf("trigger filter error = %v, want validate trigger resource spec context", err)
	}

	webhookWithoutEndpoint := validTrigger
	webhookWithoutEndpoint.Event = "webhook"
	if _, err := triggerCodec.DecodeAndValidate(
		ctx,
		workspaceScope,
		mustAutomationJSON(t, webhookWithoutEndpoint),
	); err == nil {
		t.Fatal("trigger codec accepted webhook without endpoint_slug or webhook_id")
	} else if !strings.Contains(err.Error(), "automation: validate trigger resource spec") {
		t.Fatalf("webhook trigger error = %v, want validate trigger resource spec context", err)
	}
}

func TestManagerStartRegistersResourceDefinitionsAtStartup(t *testing.T) {
	t.Parallel()

	h := newManagerResourceHarness(t)
	jobRecord := h.putJobResource(t, "job-startup", "startup-job")
	triggerRecord := h.putTriggerResource(t, "trigger-startup", "startup-trigger")
	manager := h.newResourceManager(t)
	if err := manager.Start(h.ctx); err != nil {
		t.Fatalf("manager.Start() error = %v", err)
	}
	t.Cleanup(func() {
		if err := manager.Shutdown(testutil.Context(t)); err != nil {
			t.Fatalf("manager.Shutdown() error = %v", err)
		}
	})

	jobs, err := manager.Jobs(h.ctx)
	if err != nil {
		t.Fatalf("manager.Jobs() error = %v", err)
	}
	if got := findJobByID(jobs, jobRecord.ID); got == nil {
		t.Fatalf("jobs missing resource-backed job %q after Start", jobRecord.ID)
	}
	if got, want := len(manager.scheduler.States()), 1; got != want {
		t.Fatalf("len(manager.scheduler.States()) = %d, want %d", got, want)
	}

	triggers, err := manager.Triggers(h.ctx)
	if err != nil {
		t.Fatalf("manager.Triggers() error = %v", err)
	}
	if got := findTriggerByID(triggers, triggerRecord.ID); got == nil {
		t.Fatalf("triggers missing resource-backed trigger %q after Start", triggerRecord.ID)
	}
	manager.triggers.mu.RLock()
	registered := len(manager.triggers.registrations)
	manager.triggers.mu.RUnlock()
	if got, want := registered, 1; got != want {
		t.Fatalf("len(manager.triggers.registrations) = %d, want %d", got, want)
	}
}

func TestAutomationJobResourceBuildDoesNotMutateLiveRuntime(t *testing.T) {
	t.Parallel()

	h := newManagerResourceHarness(t)
	manager := h.newResourceManager(t)
	if err := manager.Start(h.ctx); err != nil {
		t.Fatalf("manager.Start() error = %v", err)
	}
	t.Cleanup(func() {
		if err := manager.Shutdown(testutil.Context(t)); err != nil {
			t.Fatalf("manager.Shutdown() error = %v", err)
		}
	})

	record := h.putJobResource(t, "job-side-effect", "side-effect-job")
	plan, err := manager.BuildJobResourceState(h.ctx, []resources.Record[Job]{record})
	if err != nil {
		t.Fatalf("BuildJobResourceState() error = %v", err)
	}
	if plan.Kind() != JobResourceKind || plan.Revision() != record.Version || plan.OperationCount() != 1 {
		t.Fatalf(
			"job plan metadata = kind:%q revision:%d operations:%d",
			plan.Kind(),
			plan.Revision(),
			plan.OperationCount(),
		)
	}
	shutdownJobResourcePlan(t, manager, plan)

	jobs, err := manager.Jobs(h.ctx)
	if err != nil {
		t.Fatalf("manager.Jobs() error = %v", err)
	}
	if len(jobs) != 0 {
		t.Fatalf("manager.Jobs() after Build = %d, want 0", len(jobs))
	}
	if got := manager.scheduler.States(); len(got) != 0 {
		t.Fatalf("scheduler states after Build = %d, want 0", len(got))
	}
}

func TestAutomationTriggerResourceBuildDoesNotMutateLiveRuntime(t *testing.T) {
	t.Parallel()

	h := newManagerResourceHarness(t)
	manager := h.newResourceManager(t)
	if err := manager.Start(h.ctx); err != nil {
		t.Fatalf("manager.Start() error = %v", err)
	}
	t.Cleanup(func() {
		if err := manager.Shutdown(testutil.Context(t)); err != nil {
			t.Fatalf("manager.Shutdown() error = %v", err)
		}
	})

	record := h.putTriggerResource(t, "trigger-side-effect", "side-effect-trigger")
	plan, err := manager.BuildTriggerResourceState(h.ctx, []resources.Record[Trigger]{record})
	if err != nil {
		t.Fatalf("BuildTriggerResourceState() error = %v", err)
	}
	if plan.Kind() != TriggerResourceKind || plan.Revision() != record.Version || plan.OperationCount() != 1 {
		t.Fatalf(
			"trigger plan metadata = kind:%q revision:%d operations:%d",
			plan.Kind(),
			plan.Revision(),
			plan.OperationCount(),
		)
	}
	shutdownTriggerResourcePlan(t, manager, plan)

	triggers, err := manager.Triggers(h.ctx)
	if err != nil {
		t.Fatalf("manager.Triggers() error = %v", err)
	}
	if len(triggers) != 0 {
		t.Fatalf("manager.Triggers() after Build = %d, want 0", len(triggers))
	}
	manager.triggers.mu.RLock()
	registered := len(manager.triggers.registrations)
	manager.triggers.mu.RUnlock()
	if registered != 0 {
		t.Fatalf("trigger registrations after Build = %d, want 0", registered)
	}
}

func TestAutomationJobResourceApplyFailurePreservesPreviousRuntime(t *testing.T) {
	t.Parallel()

	h := newManagerResourceHarness(t)
	manager := h.newResourceManager(t)
	if err := manager.Start(h.ctx); err != nil {
		t.Fatalf("manager.Start() error = %v", err)
	}
	t.Cleanup(func() {
		if err := manager.Shutdown(testutil.Context(t)); err != nil {
			t.Fatalf("manager.Shutdown() error = %v", err)
		}
	})

	first := h.putJobResource(t, "job-previous", "previous-job")
	firstPlan, err := manager.BuildJobResourceState(h.ctx, []resources.Record[Job]{first})
	if err != nil {
		t.Fatalf("BuildJobResourceState(previous) error = %v", err)
	}
	if err := manager.ApplyJobResourceState(h.ctx, firstPlan); err != nil {
		t.Fatalf("ApplyJobResourceState(previous) error = %v", err)
	}

	next := h.putJobResource(t, "job-next", "next-job")
	nextPlan, err := manager.BuildJobResourceState(h.ctx, []resources.Record[Job]{next})
	if err != nil {
		t.Fatalf("BuildJobResourceState(next) error = %v", err)
	}
	canceledCtx, cancel := context.WithCancel(h.ctx)
	cancel()
	if err := manager.ApplyJobResourceState(canceledCtx, nextPlan); !errors.Is(err, context.Canceled) {
		t.Fatalf("ApplyJobResourceState(canceled) error = %v, want context.Canceled", err)
	}

	jobs, err := manager.Jobs(h.ctx)
	if err != nil {
		t.Fatalf("manager.Jobs() error = %v", err)
	}
	if got := findJobByID(jobs, first.ID); got == nil {
		t.Fatalf("previous job %q missing after failed Apply", first.ID)
	}
	if got := findJobByID(jobs, next.ID); got != nil {
		t.Fatalf("next job %q applied after failed Apply", next.ID)
	}
}

func TestAutomationTriggerResourceApplyFailurePreservesPreviousRuntime(t *testing.T) {
	t.Parallel()

	h := newManagerResourceHarness(t)
	manager := h.newResourceManager(t)
	if err := manager.Start(h.ctx); err != nil {
		t.Fatalf("manager.Start() error = %v", err)
	}
	t.Cleanup(func() {
		if err := manager.Shutdown(testutil.Context(t)); err != nil {
			t.Fatalf("manager.Shutdown() error = %v", err)
		}
	})

	first := h.putTriggerResource(t, "trigger-previous", "previous-trigger")
	firstPlan, err := manager.BuildTriggerResourceState(h.ctx, []resources.Record[Trigger]{first})
	if err != nil {
		t.Fatalf("BuildTriggerResourceState(previous) error = %v", err)
	}
	if err := manager.ApplyTriggerResourceState(h.ctx, firstPlan); err != nil {
		t.Fatalf("ApplyTriggerResourceState(previous) error = %v", err)
	}

	next := h.putTriggerResource(t, "trigger-next", "next-trigger")
	nextPlan, err := manager.BuildTriggerResourceState(h.ctx, []resources.Record[Trigger]{next})
	if err != nil {
		t.Fatalf("BuildTriggerResourceState(next) error = %v", err)
	}
	canceledCtx, cancel := context.WithCancel(h.ctx)
	cancel()
	if err := manager.ApplyTriggerResourceState(canceledCtx, nextPlan); !errors.Is(err, context.Canceled) {
		t.Fatalf("ApplyTriggerResourceState(canceled) error = %v, want context.Canceled", err)
	}

	triggers, err := manager.Triggers(h.ctx)
	if err != nil {
		t.Fatalf("manager.Triggers() error = %v", err)
	}
	if got := findTriggerByID(triggers, first.ID); got == nil {
		t.Fatalf("previous trigger %q missing after failed Apply", first.ID)
	}
	if got := findTriggerByID(triggers, next.ID); got != nil {
		t.Fatalf("next trigger %q applied after failed Apply", next.ID)
	}
}

func TestLegacyAutomationDefinitionWritesDoNotDriveResourceManager(t *testing.T) {
	t.Parallel()

	h := newManagerResourceHarness(t)
	manager := h.newResourceManager(t)
	if err := manager.Start(h.ctx); err != nil {
		t.Fatalf("manager.Start() error = %v", err)
	}
	t.Cleanup(func() {
		if err := manager.Shutdown(testutil.Context(t)); err != nil {
			t.Fatalf("manager.Shutdown() error = %v", err)
		}
	})

	legacyJob, err := h.db.CreateJob(h.ctx, testJob(AutomationScopeGlobal, "legacy-job", ""))
	if err != nil {
		t.Fatalf("CreateJob(legacy) error = %v", err)
	}
	if _, err := manager.GetJob(h.ctx, legacyJob.ID); !errors.Is(err, ErrJobNotFound) {
		t.Fatalf("manager.GetJob(legacy) error = %v, want ErrJobNotFound", err)
	}

	resourceRecord := h.putJobResource(t, "job-resource", "resource-job")
	plan, err := manager.BuildJobResourceState(h.ctx, []resources.Record[Job]{resourceRecord})
	if err != nil {
		t.Fatalf("BuildJobResourceState(resource) error = %v", err)
	}
	if err := manager.ApplyJobResourceState(h.ctx, plan); err != nil {
		t.Fatalf("ApplyJobResourceState(resource) error = %v", err)
	}
	if _, err := manager.GetJob(h.ctx, resourceRecord.ID); err != nil {
		t.Fatalf("manager.GetJob(resource) error = %v", err)
	}
}

func TestAutomationResourceProjectionRejectsInvalidInputs(t *testing.T) {
	t.Parallel()

	h := newManagerResourceHarness(t)
	manager := h.newResourceManager(t)

	if _, err := manager.BuildJobResourceState(nilAutomationResourceContext(), nil); err == nil {
		t.Fatal("BuildJobResourceState(nil context) error = nil, want error")
	}
	var nilManager *Manager
	if _, err := nilManager.BuildJobResourceState(h.ctx, nil); err == nil {
		t.Fatal("nil manager BuildJobResourceState() error = nil, want error")
	}
	if err := manager.ApplyJobResourceState(nilAutomationResourceContext(), nil); err == nil {
		t.Fatal("ApplyJobResourceState(nil context) error = nil, want error")
	}
	if err := nilManager.ApplyJobResourceState(h.ctx, nil); err == nil {
		t.Fatal("nil manager ApplyJobResourceState() error = nil, want error")
	}
	if err := manager.ApplyJobResourceState(h.ctx, &triggerResourceProjectionPlan{}); err == nil {
		t.Fatal("ApplyJobResourceState(wrong plan type) error = nil, want error")
	}
	if err := manager.ApplyJobResourceState(h.ctx, &jobResourceProjectionPlan{}); err == nil {
		t.Fatal("ApplyJobResourceState(missing scheduler) error = nil, want error")
	}

	if _, err := manager.BuildTriggerResourceState(nilAutomationResourceContext(), nil); err == nil {
		t.Fatal("BuildTriggerResourceState(nil context) error = nil, want error")
	}
	if _, err := nilManager.BuildTriggerResourceState(h.ctx, nil); err == nil {
		t.Fatal("nil manager BuildTriggerResourceState() error = nil, want error")
	}
	if err := manager.ApplyTriggerResourceState(nilAutomationResourceContext(), nil); err == nil {
		t.Fatal("ApplyTriggerResourceState(nil context) error = nil, want error")
	}
	if err := nilManager.ApplyTriggerResourceState(h.ctx, nil); err == nil {
		t.Fatal("nil manager ApplyTriggerResourceState() error = nil, want error")
	}
	if err := manager.ApplyTriggerResourceState(h.ctx, &jobResourceProjectionPlan{}); err == nil {
		t.Fatal("ApplyTriggerResourceState(wrong plan type) error = nil, want error")
	}
	if err := manager.ApplyTriggerResourceState(h.ctx, &triggerResourceProjectionPlan{}); err == nil {
		t.Fatal("ApplyTriggerResourceState(missing engine) error = nil, want error")
	}
}

func TestAutomationResourceManagerCRUDUsesTypedResourceStores(t *testing.T) {
	t.Parallel()

	h := newManagerResourceHarness(t)
	manager := h.newResourceManager(t)
	if err := manager.Start(h.ctx); err != nil {
		t.Fatalf("manager.Start() error = %v", err)
	}
	t.Cleanup(func() {
		if err := manager.Shutdown(testutil.Context(t)); err != nil {
			t.Fatalf("manager.Shutdown() error = %v", err)
		}
	})

	createdJob, err := manager.CreateJob(h.ctx, testJob(AutomationScopeGlobal, "resource-crud-job", ""))
	if err != nil {
		t.Fatalf("CreateJob(resource) error = %v", err)
	}
	if createdJob.Source != JobSourceDynamic {
		t.Fatalf("created job source = %q, want %q", createdJob.Source, JobSourceDynamic)
	}
	jobRecord, err := h.jobStore.Get(h.ctx, h.actor, createdJob.ID)
	if err != nil {
		t.Fatalf("jobStore.Get(created) error = %v", err)
	}
	if jobRecord.Spec.Name != createdJob.Name {
		t.Fatalf("job resource name = %q, want %q", jobRecord.Spec.Name, createdJob.Name)
	}

	nextJob := createdJob
	nextJob.Prompt = "Review the resource-backed scheduler"
	updatedJob, err := manager.UpdateJob(h.ctx, nextJob)
	if err != nil {
		t.Fatalf("UpdateJob(resource) error = %v", err)
	}
	if updatedJob.Prompt != nextJob.Prompt {
		t.Fatalf("updated job prompt = %q, want %q", updatedJob.Prompt, nextJob.Prompt)
	}
	disabledJob, err := manager.SetJobEnabled(h.ctx, createdJob.ID, false)
	if err != nil {
		t.Fatalf("SetJobEnabled(resource) error = %v", err)
	}
	if disabledJob.Enabled {
		t.Fatal("SetJobEnabled(resource) returned enabled job, want disabled")
	}
	jobRecord, err = h.jobStore.Get(h.ctx, h.actor, createdJob.ID)
	if err != nil {
		t.Fatalf("jobStore.Get(disabled) error = %v", err)
	}
	if jobRecord.Spec.Enabled {
		t.Fatal("resource job spec enabled = true, want false")
	}
	if err := manager.DeleteJob(h.ctx, createdJob.ID); err != nil {
		t.Fatalf("DeleteJob(resource) error = %v", err)
	}
	if _, err := manager.GetJob(h.ctx, createdJob.ID); !errors.Is(err, ErrJobNotFound) {
		t.Fatalf("GetJob(deleted resource) error = %v, want ErrJobNotFound", err)
	}
	if _, err := h.jobStore.Get(h.ctx, h.actor, createdJob.ID); !errors.Is(err, resources.ErrNotFound) {
		t.Fatalf("jobStore.Get(deleted) error = %v, want resources.ErrNotFound", err)
	}

	trigger := testTrigger(AutomationScopeGlobal, "resource-crud-trigger", "")
	trigger.Event = "session.stopped"
	trigger.WebhookID = ""
	createdTrigger, err := manager.CreateTrigger(h.ctx, trigger, WebhookSecretWrite{})
	if err != nil {
		t.Fatalf("CreateTrigger(resource) error = %v", err)
	}
	triggerRecord, err := h.triggerStore.Get(h.ctx, h.actor, createdTrigger.ID)
	if err != nil {
		t.Fatalf("triggerStore.Get(created) error = %v", err)
	}
	if triggerRecord.Spec.Event != "session.stopped" {
		t.Fatalf("trigger resource event = %q, want session.stopped", triggerRecord.Spec.Event)
	}

	nextTrigger := createdTrigger
	nextTrigger.Prompt = `Review stopped session {{ index .Data "session_id" }}`
	updatedTrigger, err := manager.UpdateTrigger(h.ctx, nextTrigger, nil)
	if err != nil {
		t.Fatalf("UpdateTrigger(resource) error = %v", err)
	}
	if updatedTrigger.Prompt != nextTrigger.Prompt {
		t.Fatalf("updated trigger prompt = %q, want %q", updatedTrigger.Prompt, nextTrigger.Prompt)
	}
	disabledTrigger, err := manager.SetTriggerEnabled(h.ctx, createdTrigger.ID, false)
	if err != nil {
		t.Fatalf("SetTriggerEnabled(resource) error = %v", err)
	}
	if disabledTrigger.Enabled {
		t.Fatal("SetTriggerEnabled(resource) returned enabled trigger, want disabled")
	}
	triggerRecord, err = h.triggerStore.Get(h.ctx, h.actor, createdTrigger.ID)
	if err != nil {
		t.Fatalf("triggerStore.Get(disabled) error = %v", err)
	}
	if triggerRecord.Spec.Enabled {
		t.Fatal("resource trigger spec enabled = true, want false")
	}
	if err := manager.DeleteTrigger(h.ctx, createdTrigger.ID); err != nil {
		t.Fatalf("DeleteTrigger(resource) error = %v", err)
	}
	if _, err := manager.GetTrigger(h.ctx, createdTrigger.ID); !errors.Is(err, ErrTriggerNotFound) {
		t.Fatalf("GetTrigger(deleted resource) error = %v, want ErrTriggerNotFound", err)
	}
	if _, err := h.triggerStore.Get(h.ctx, h.actor, createdTrigger.ID); !errors.Is(err, resources.ErrNotFound) {
		t.Fatalf("triggerStore.Get(deleted) error = %v, want resources.ErrNotFound", err)
	}
}

func TestAutomationResourceManagerCRUDRollsBackCommittedMutationsOnApplyFailure(t *testing.T) {
	t.Parallel()

	t.Run("Should rollback created job resources when projection apply fails", func(t *testing.T) {
		t.Parallel()

		h := newManagerResourceHarness(t)
		ctx, cancel := context.WithCancel(h.ctx)
		jobStore := &cancelAfterMutationStore[Job]{
			store:       h.jobStore,
			cancel:      cancel,
			cancelOnPut: true,
		}
		manager := h.newResourceManager(t, WithResourceDefinitions(jobStore, h.triggerStore, h.actor, nil))

		job := testJob(AutomationScopeGlobal, "rollback-created-job", "")
		_, err := manager.CreateJob(ctx, job)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("manager.CreateJob() error = %v, want context.Canceled", err)
		}
		if _, err := h.jobStore.Get(h.ctx, h.actor, job.ID); !errors.Is(err, resources.ErrNotFound) {
			t.Fatalf("jobStore.Get(created rollback) error = %v, want resources.ErrNotFound", err)
		}
		if _, err := manager.GetJob(h.ctx, job.ID); !errors.Is(err, ErrJobNotFound) {
			t.Fatalf("manager.GetJob(created rollback) error = %v, want ErrJobNotFound", err)
		}
	})

	t.Run("Should rollback updated job resources when projection apply fails", func(t *testing.T) {
		t.Parallel()

		h := newManagerResourceHarness(t)
		jobStore := &cancelAfterMutationStore[Job]{store: h.jobStore}
		manager := h.newResourceManager(t, WithResourceDefinitions(jobStore, h.triggerStore, h.actor, nil))

		current := h.putJobResource(t, "job-rollback-update", "rollback-update-job")
		if err := manager.applyJobResourcesFromStore(h.ctx); err != nil {
			t.Fatalf("manager.applyJobResourcesFromStore() error = %v", err)
		}

		ctx, cancel := context.WithCancel(h.ctx)
		jobStore.cancel = cancel
		jobStore.cancelOnPut = true
		next := cloneJob(current.Spec)
		next.ID = current.ID
		next.Prompt = "Review the updated rollback prompt"
		_, err := manager.UpdateJob(ctx, next)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("manager.UpdateJob() error = %v, want context.Canceled", err)
		}

		stored, err := h.jobStore.Get(h.ctx, h.actor, current.ID)
		if err != nil {
			t.Fatalf("jobStore.Get(updated rollback) error = %v", err)
		}
		if stored.Spec.Prompt != current.Spec.Prompt {
			t.Fatalf("stored job prompt after rollback = %q, want %q", stored.Spec.Prompt, current.Spec.Prompt)
		}
		effective, err := manager.GetJob(h.ctx, current.ID)
		if err != nil {
			t.Fatalf("manager.GetJob(updated rollback) error = %v", err)
		}
		if effective.Prompt != current.Spec.Prompt {
			t.Fatalf("effective job prompt after rollback = %q, want %q", effective.Prompt, current.Spec.Prompt)
		}
	})

	t.Run("Should rollback deleted job resources when projection apply fails", func(t *testing.T) {
		t.Parallel()

		h := newManagerResourceHarness(t)
		jobStore := &cancelAfterMutationStore[Job]{store: h.jobStore}
		manager := h.newResourceManager(t, WithResourceDefinitions(jobStore, h.triggerStore, h.actor, nil))

		current := h.putJobResource(t, "job-rollback-delete", "rollback-delete-job")
		if err := manager.applyJobResourcesFromStore(h.ctx); err != nil {
			t.Fatalf("manager.applyJobResourcesFromStore() error = %v", err)
		}

		ctx, cancel := context.WithCancel(h.ctx)
		jobStore.cancel = cancel
		jobStore.cancelOnDelete = true
		err := manager.DeleteJob(ctx, current.ID)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("manager.DeleteJob() error = %v, want context.Canceled", err)
		}

		if _, err := h.jobStore.Get(h.ctx, h.actor, current.ID); err != nil {
			t.Fatalf("jobStore.Get(deleted rollback) error = %v", err)
		}
		if _, err := manager.GetJob(h.ctx, current.ID); err != nil {
			t.Fatalf("manager.GetJob(deleted rollback) error = %v", err)
		}
	})

	t.Run(
		"Should rollback created trigger resources and owned secrets when projection apply fails",
		func(t *testing.T) {
			t.Parallel()

			h := newManagerResourceHarness(t)
			ctx, cancel := context.WithCancel(h.ctx)
			triggerStore := &cancelAfterMutationStore[Trigger]{
				store:       h.triggerStore,
				cancel:      cancel,
				cancelOnPut: true,
			}
			manager := h.newResourceManager(t, WithResourceDefinitions(h.jobStore, triggerStore, h.actor, nil))

			trigger := testWebhookTrigger(AutomationScopeGlobal, "rollback-created-trigger", "")
			trigger.WebhookSecretRef = ""
			secretRef := defaultAutomationWebhookSecretRef(trigger.ID)
			_, err := manager.CreateTrigger(ctx, trigger, webhookSecretWrite("secret-v1"))
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("manager.CreateTrigger() error = %v, want context.Canceled", err)
			}
			if _, err := h.triggerStore.Get(h.ctx, h.actor, trigger.ID); !errors.Is(err, resources.ErrNotFound) {
				t.Fatalf("triggerStore.Get(created rollback) error = %v, want resources.ErrNotFound", err)
			}
			h.assertWebhookSecretMissing(t, secretRef)
			if _, err := manager.GetTrigger(h.ctx, trigger.ID); !errors.Is(err, ErrTriggerNotFound) {
				t.Fatalf("manager.GetTrigger(created rollback) error = %v, want ErrTriggerNotFound", err)
			}
		},
	)

	t.Run(
		"Should rollback updated trigger resources and owned secret deletion when projection apply fails",
		func(t *testing.T) {
			t.Parallel()

			h := newManagerResourceHarness(t)
			secretStore := &cancelAfterWebhookSecretStore{
				store:          h.webhookSecretStore(t),
				cancelOnDelete: false,
			}
			manager := h.newResourceManager(t, WithWebhookSecretStore(secretStore))

			current := h.putOwnedWebhookTriggerResource(
				t,
				"trigger-rollback-update",
				"rollback-update-trigger",
				"secret-v1",
			)
			if err := manager.applyTriggerResourcesFromStore(h.ctx); err != nil {
				t.Fatalf("manager.applyTriggerResourcesFromStore() error = %v", err)
			}

			ctx, cancel := context.WithCancel(h.ctx)
			secretStore.cancel = cancel
			secretStore.cancelOnDelete = true
			next := cloneTrigger(current.Spec)
			next.ID = current.ID
			next.Event = "session.stopped"
			next.EndpointSlug = ""
			next.WebhookID = ""
			next.WebhookSecretRef = ""
			_, err := manager.UpdateTrigger(ctx, next, nil)
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("manager.UpdateTrigger() error = %v, want context.Canceled", err)
			}

			stored, err := h.triggerStore.Get(h.ctx, h.actor, current.ID)
			if err != nil {
				t.Fatalf("triggerStore.Get(updated rollback) error = %v", err)
			}
			if stored.Spec.Event != current.Spec.Event ||
				stored.Spec.WebhookSecretRef != current.Spec.WebhookSecretRef {
				t.Fatalf(
					"stored trigger after rollback = %#v, want event %q secret ref %q",
					stored.Spec,
					current.Spec.Event,
					current.Spec.WebhookSecretRef,
				)
			}
			if got := h.resolveWebhookSecret(t, current.Spec.WebhookSecretRef); got != "secret-v1" {
				t.Fatalf("restored webhook secret = %q, want %q", got, "secret-v1")
			}
			effective, err := manager.GetTrigger(h.ctx, current.ID)
			if err != nil {
				t.Fatalf("manager.GetTrigger(updated rollback) error = %v", err)
			}
			if effective.Event != current.Spec.Event || effective.WebhookSecretRef != current.Spec.WebhookSecretRef {
				t.Fatalf(
					"effective trigger after rollback = %#v, want event %q secret ref %q",
					effective,
					current.Spec.Event,
					current.Spec.WebhookSecretRef,
				)
			}
		},
	)

	t.Run(
		"Should rollback deleted trigger resources and owned secret deletion when projection apply fails",
		func(t *testing.T) {
			t.Parallel()

			h := newManagerResourceHarness(t)
			secretStore := &cancelAfterWebhookSecretStore{
				store: h.webhookSecretStore(t),
			}
			manager := h.newResourceManager(t, WithWebhookSecretStore(secretStore))

			current := h.putOwnedWebhookTriggerResource(
				t,
				"trigger-rollback-delete",
				"rollback-delete-trigger",
				"secret-v1",
			)
			if err := manager.applyTriggerResourcesFromStore(h.ctx); err != nil {
				t.Fatalf("manager.applyTriggerResourcesFromStore() error = %v", err)
			}

			ctx, cancel := context.WithCancel(h.ctx)
			secretStore.cancel = cancel
			secretStore.cancelOnDelete = true
			err := manager.DeleteTrigger(ctx, current.ID)
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("manager.DeleteTrigger() error = %v, want context.Canceled", err)
			}

			if _, err := h.triggerStore.Get(h.ctx, h.actor, current.ID); err != nil {
				t.Fatalf("triggerStore.Get(deleted rollback) error = %v", err)
			}
			if got := h.resolveWebhookSecret(t, current.Spec.WebhookSecretRef); got != "secret-v1" {
				t.Fatalf("restored deleted webhook secret = %q, want %q", got, "secret-v1")
			}
			if _, err := manager.GetTrigger(h.ctx, current.ID); err != nil {
				t.Fatalf("manager.GetTrigger(deleted rollback) error = %v", err)
			}
		},
	)
}

func TestAutomationResourceSyncManagedDefinitionsPublishesAndPrunesSourceRecords(t *testing.T) {
	t.Parallel()

	h := newManagerResourceHarness(t)
	var triggered []resources.ResourceKind
	manager := h.newResourceManager(t, WithResourceDefinitions(
		h.jobStore,
		h.triggerStore,
		h.actor,
		func(_ context.Context, kind resources.ResourceKind, _ resources.ReconcileReason) error {
			triggered = append(triggered, kind.Normalize())
			return nil
		},
	))

	job := testJob(AutomationScopeGlobal, "managed-resource-job", "")
	trigger := testTrigger(AutomationScopeGlobal, "managed-resource-trigger", "")
	trigger.Event = "session.stopped"
	trigger.WebhookID = ""
	trigger.WebhookSecretRef = ""

	stats, err := manager.SyncManagedDefinitions(
		h.ctx,
		JobSourceConfig,
		[]Job{job},
		[]Trigger{trigger},
	)
	if err != nil {
		t.Fatalf("SyncManagedDefinitions(create) error = %v", err)
	}
	if stats.JobsSynced != 1 || stats.TriggersSynced != 1 || stats.JobsRemoved != 0 || stats.TriggersRemoved != 0 {
		t.Fatalf("create stats = %#v", stats)
	}

	configActor := manager.resourceActorForSource(JobSourceConfig)
	jobRecord, err := h.jobStore.Get(h.ctx, configActor, job.ID)
	if err != nil {
		t.Fatalf("jobStore.Get(config) error = %v", err)
	}
	if jobRecord.Spec.Source != JobSourceConfig || jobRecord.Spec.Name != job.Name {
		t.Fatalf("config job resource = %#v", jobRecord.Spec)
	}
	triggerRecord, err := h.triggerStore.Get(h.ctx, configActor, trigger.ID)
	if err != nil {
		t.Fatalf("triggerStore.Get(config) error = %v", err)
	}
	if triggerRecord.Spec.Source != JobSourceConfig || triggerRecord.Spec.Event != "session.stopped" {
		t.Fatalf("config trigger resource = %#v", triggerRecord.Spec)
	}

	job.Prompt = "Review the updated config resource"
	stats, err = manager.SyncManagedDefinitions(h.ctx, JobSourceConfig, []Job{job}, nil)
	if err != nil {
		t.Fatalf("SyncManagedDefinitions(update) error = %v", err)
	}
	if stats.JobsSynced != 1 || stats.TriggersSynced != 0 || stats.JobsRemoved != 0 || stats.TriggersRemoved != 1 {
		t.Fatalf("update stats = %#v", stats)
	}
	jobRecord, err = h.jobStore.Get(h.ctx, configActor, job.ID)
	if err != nil {
		t.Fatalf("jobStore.Get(updated config) error = %v", err)
	}
	if jobRecord.Spec.Prompt != job.Prompt {
		t.Fatalf("updated config job prompt = %q, want %q", jobRecord.Spec.Prompt, job.Prompt)
	}
	if _, err := h.triggerStore.Get(h.ctx, configActor, trigger.ID); !errors.Is(err, resources.ErrNotFound) {
		t.Fatalf("triggerStore.Get(pruned config) error = %v, want resources.ErrNotFound", err)
	}
	if len(triggered) != 4 {
		t.Fatalf("resource reconcile triggers = %#v, want two kinds for each sync", triggered)
	}
}

func TestAutomationResourceSyncManagedDefinitionsSkipsUnchangedTaskBackedJob(t *testing.T) {
	t.Parallel()

	h := newManagerResourceHarness(t)
	manager := h.newResourceManager(t)

	job := testJob(AutomationScopeGlobal, "task-backed-resource-job", "")
	job.AgentName = ""
	job.Prompt = ""
	job.Retry = RetryConfig{Strategy: RetryStrategyNone}
	job.Task = &JobTaskConfig{
		Title:          "Run task-backed automation",
		Description:    "Exercise task equality in managed resource sync",
		NetworkChannel: "builders",
		Owner:          &taskpkg.Ownership{Kind: taskpkg.OwnerKindPool, Ref: "ops"},
	}

	if _, err := manager.SyncManagedDefinitions(h.ctx, JobSourceConfig, []Job{job}, nil); err != nil {
		t.Fatalf("SyncManagedDefinitions(first) error = %v", err)
	}
	configActor := manager.resourceActorForSource(JobSourceConfig)
	firstRecord, err := h.jobStore.Get(h.ctx, configActor, job.ID)
	if err != nil {
		t.Fatalf("jobStore.Get(first) error = %v", err)
	}

	stats, err := manager.SyncManagedDefinitions(h.ctx, JobSourceConfig, []Job{job}, nil)
	if err != nil {
		t.Fatalf("SyncManagedDefinitions(second) error = %v", err)
	}
	if stats.JobsSynced != 1 || stats.JobsRemoved != 0 || stats.TriggersSynced != 0 || stats.TriggersRemoved != 0 {
		t.Fatalf("second sync stats = %#v", stats)
	}
	secondRecord, err := h.jobStore.Get(h.ctx, configActor, job.ID)
	if err != nil {
		t.Fatalf("jobStore.Get(second) error = %v", err)
	}
	if secondRecord.Version != firstRecord.Version {
		t.Fatalf("unchanged task-backed job version = %d, want %d", secondRecord.Version, firstRecord.Version)
	}
	if secondRecord.Spec.Task == nil || secondRecord.Spec.Task.Owner == nil {
		t.Fatalf("task-backed job resource lost task owner: %#v", secondRecord.Spec.Task)
	}
}

func TestAutomationResourceConfigEnabledChangesUseOperationalOverlays(t *testing.T) {
	t.Parallel()

	h := newManagerResourceHarness(t)
	manager := h.newResourceManager(t)
	if err := manager.Start(h.ctx); err != nil {
		t.Fatalf("manager.Start() error = %v", err)
	}
	t.Cleanup(func() {
		if err := manager.Shutdown(testutil.Context(t)); err != nil {
			t.Fatalf("manager.Shutdown() error = %v", err)
		}
	})

	job := testJob(AutomationScopeGlobal, "config-resource-job", "")
	trigger := testTrigger(AutomationScopeGlobal, "config-resource-trigger", "")
	trigger.Event = "session.stopped"
	trigger.WebhookID = ""
	trigger.WebhookSecretRef = ""
	if _, err := manager.SyncManagedDefinitions(
		h.ctx,
		JobSourceConfig,
		[]Job{job},
		[]Trigger{trigger},
	); err != nil {
		t.Fatalf("SyncManagedDefinitions(config) error = %v", err)
	}
	if err := manager.applyJobResourcesFromStore(h.ctx); err != nil {
		t.Fatalf("applyJobResourcesFromStore() error = %v", err)
	}
	if err := manager.applyTriggerResourcesFromStore(h.ctx); err != nil {
		t.Fatalf("applyTriggerResourcesFromStore() error = %v", err)
	}

	disabledJob, err := manager.SetJobEnabled(h.ctx, job.ID, false)
	if err != nil {
		t.Fatalf("SetJobEnabled(config resource) error = %v", err)
	}
	if disabledJob.Enabled {
		t.Fatal("config resource job effective enabled = true, want false overlay")
	}
	configActor := manager.resourceActorForSource(JobSourceConfig)
	jobRecord, err := h.jobStore.Get(h.ctx, configActor, job.ID)
	if err != nil {
		t.Fatalf("jobStore.Get(config resource) error = %v", err)
	}
	if !jobRecord.Spec.Enabled {
		t.Fatal("config resource job spec enabled = false, want unchanged true")
	}
	jobOverlay, err := h.db.GetJobEnabledOverlay(h.ctx, job.ID)
	if err != nil {
		t.Fatalf("GetJobEnabledOverlay(config resource) error = %v", err)
	}
	if jobOverlay.EnabledOverride {
		t.Fatal("config job overlay enabled_override = true, want false")
	}

	disabledTrigger, err := manager.SetTriggerEnabled(h.ctx, trigger.ID, false)
	if err != nil {
		t.Fatalf("SetTriggerEnabled(config resource) error = %v", err)
	}
	if disabledTrigger.Enabled {
		t.Fatal("config resource trigger effective enabled = true, want false overlay")
	}
	triggerRecord, err := h.triggerStore.Get(h.ctx, configActor, trigger.ID)
	if err != nil {
		t.Fatalf("triggerStore.Get(config resource) error = %v", err)
	}
	if !triggerRecord.Spec.Enabled {
		t.Fatal("config resource trigger spec enabled = false, want unchanged true")
	}
	triggerOverlay, err := h.db.GetTriggerEnabledOverlay(h.ctx, trigger.ID)
	if err != nil {
		t.Fatalf("GetTriggerEnabledOverlay(config resource) error = %v", err)
	}
	if triggerOverlay.EnabledOverride {
		t.Fatal("config trigger overlay enabled_override = true, want false")
	}
}

type managerResourceHarness struct {
	*managerHarness
	jobStore     resources.Store[Job]
	triggerStore resources.Store[Trigger]
	actor        resources.MutationActor
}

func newManagerResourceHarness(t *testing.T) *managerResourceHarness {
	t.Helper()

	base := newManagerHarness(t)
	kernel, err := resources.NewKernel(base.db.DB())
	if err != nil {
		t.Fatalf("resources.NewKernel() error = %v", err)
	}
	jobCodec, err := NewJobResourceCodec()
	if err != nil {
		t.Fatalf("NewJobResourceCodec() error = %v", err)
	}
	jobStore, err := resources.NewStore(kernel, jobCodec)
	if err != nil {
		t.Fatalf("resources.NewStore(job) error = %v", err)
	}
	triggerCodec, err := NewTriggerResourceCodec()
	if err != nil {
		t.Fatalf("NewTriggerResourceCodec() error = %v", err)
	}
	triggerStore, err := resources.NewStore(kernel, triggerCodec)
	if err != nil {
		t.Fatalf("resources.NewStore(trigger) error = %v", err)
	}

	return &managerResourceHarness{
		managerHarness: base,
		jobStore:       jobStore,
		triggerStore:   triggerStore,
		actor: resources.MutationActor{
			Kind: resources.MutationActorKindDaemon,
			ID:   "automation-resource-test",
			Source: resources.ResourceSource{
				Kind: resources.ResourceSourceKind("daemon"),
				ID:   "automation-resource-test",
			},
			MaxScope: resources.ResourceScope{Kind: resources.ResourceScopeKindGlobal},
		},
	}
}

func (h *managerResourceHarness) newResourceManager(t *testing.T, opts ...Option) *Manager {
	t.Helper()

	resourceOpts := []Option{
		WithResourceDefinitions(h.jobStore, h.triggerStore, h.actor, nil),
	}
	resourceOpts = append(resourceOpts, opts...)
	return h.newManager(t, defaultAutomationTestConfig(), resourceOpts...)
}

func (h *managerResourceHarness) putJobResource(t *testing.T, id string, name string) resources.Record[Job] {
	t.Helper()

	runAt := time.Now().UTC().Add(time.Hour).Format(time.RFC3339)
	job := Job{
		Scope:     AutomationScopeGlobal,
		Name:      name,
		AgentName: "reviewer",
		Prompt:    "Review repository",
		Schedule:  &ScheduleSpec{Mode: ScheduleModeAt, Time: runAt},
		Enabled:   true,
		Retry:     DefaultRetryConfig(),
		FireLimit: DefaultFireLimitConfig(),
		Source:    JobSourceDynamic,
	}
	record, err := h.jobStore.Put(testutil.Context(t), h.actor, resources.Draft[Job]{
		ID:              id,
		Scope:           ResourceScopeForAutomation(job.Scope, job.WorkspaceID),
		ExpectedVersion: 0,
		Spec:            job,
	})
	if err != nil {
		t.Fatalf("jobStore.Put(%q) error = %v", id, err)
	}
	return record
}

func (h *managerResourceHarness) putTriggerResource(t *testing.T, id string, name string) resources.Record[Trigger] {
	t.Helper()

	trigger := Trigger{
		Scope:     AutomationScopeGlobal,
		Name:      name,
		AgentName: "reviewer",
		Prompt:    `Review {{ index .Data "session_id" }}`,
		Event:     "session.stopped",
		Enabled:   true,
		Retry:     DefaultRetryConfig(),
		FireLimit: DefaultFireLimitConfig(),
		Source:    JobSourceDynamic,
	}
	record, err := h.triggerStore.Put(testutil.Context(t), h.actor, resources.Draft[Trigger]{
		ID:              id,
		Scope:           ResourceScopeForAutomation(trigger.Scope, trigger.WorkspaceID),
		ExpectedVersion: 0,
		Spec:            trigger,
	})
	if err != nil {
		t.Fatalf("triggerStore.Put(%q) error = %v", id, err)
	}
	return record
}

func (h *managerResourceHarness) putOwnedWebhookTriggerResource(
	t *testing.T,
	id string,
	name string,
	secret string,
) resources.Record[Trigger] {
	t.Helper()

	trigger := testWebhookTrigger(AutomationScopeGlobal, name, "")
	trigger.ID = id
	trigger.Name = name
	trigger.WebhookID = "wbh_" + name
	trigger.WebhookSecretRef = defaultAutomationWebhookSecretRef(id)
	record, err := h.triggerStore.Put(testutil.Context(t), h.actor, resources.Draft[Trigger]{
		ID:              trigger.ID,
		Scope:           ResourceScopeForAutomation(trigger.Scope, trigger.WorkspaceID),
		ExpectedVersion: 0,
		Spec:            trigger,
	})
	if err != nil {
		t.Fatalf("triggerStore.Put(%q) error = %v", id, err)
	}
	h.putWebhookSecret(t, trigger.WebhookSecretRef, secret)
	return record
}

func defaultAutomationTestConfig() aghconfig.AutomationConfig {
	return aghconfig.AutomationConfig{
		Enabled:           true,
		Timezone:          DefaultTimezone,
		MaxConcurrentJobs: DefaultMaxConcurrentJobs,
		DefaultFireLimit:  DefaultFireLimitConfig(),
	}
}

func mustAutomationJSON(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return raw
}

func nilAutomationResourceContext() context.Context {
	return nil
}

func shutdownJobResourcePlan(t *testing.T, manager *Manager, plan resources.ProjectionPlan) {
	t.Helper()
	typed, ok := plan.(*jobResourceProjectionPlan)
	if !ok || typed.scheduler == nil {
		return
	}
	if err := manager.shutdownRuntimeComponent(testutil.Context(t), "scheduler", typed.scheduler); err != nil {
		t.Fatalf("shutdown job resource plan scheduler error = %v", err)
	}
}

func shutdownTriggerResourcePlan(t *testing.T, manager *Manager, plan resources.ProjectionPlan) {
	t.Helper()
	typed, ok := plan.(*triggerResourceProjectionPlan)
	if !ok || typed.engine == nil {
		return
	}
	if err := manager.shutdownRuntimeComponent(testutil.Context(t), "trigger engine", typed.engine); err != nil {
		t.Fatalf("shutdown trigger resource plan engine error = %v", err)
	}
}

type cancelAfterMutationStore[T any] struct {
	store          resources.Store[T]
	cancel         context.CancelFunc
	cancelOnPut    bool
	cancelOnDelete bool
}

func (s *cancelAfterMutationStore[T]) Put(
	ctx context.Context,
	actor resources.MutationActor,
	draft resources.Draft[T],
) (resources.Record[T], error) {
	record, err := s.store.Put(ctx, actor, draft)
	if err == nil && s.cancelOnPut && s.cancel != nil {
		s.cancel()
	}
	return record, err
}

func (s *cancelAfterMutationStore[T]) Delete(
	ctx context.Context,
	actor resources.MutationActor,
	id string,
	expectedVersion int64,
) error {
	err := s.store.Delete(ctx, actor, id, expectedVersion)
	if err == nil && s.cancelOnDelete && s.cancel != nil {
		s.cancel()
	}
	return err
}

func (s *cancelAfterMutationStore[T]) Get(
	ctx context.Context,
	actor resources.MutationActor,
	id string,
) (resources.Record[T], error) {
	return s.store.Get(ctx, actor, id)
}

func (s *cancelAfterMutationStore[T]) List(
	ctx context.Context,
	actor resources.MutationActor,
	filter resources.ResourceFilter,
) ([]resources.Record[T], error) {
	return s.store.List(ctx, actor, filter)
}

type cancelAfterWebhookSecretStore struct {
	store          WebhookSecretStore
	cancel         context.CancelFunc
	cancelOnPut    bool
	cancelOnDelete bool
}

func (s *cancelAfterWebhookSecretStore) ResolveRef(ctx context.Context, ref string) (string, error) {
	return s.store.ResolveRef(ctx, ref)
}

func (s *cancelAfterWebhookSecretStore) PutSecret(
	ctx context.Context,
	ref string,
	kind string,
	value string,
) (vault.Metadata, error) {
	metadata, err := s.store.PutSecret(ctx, ref, kind, value)
	if err == nil && s.cancelOnPut && s.cancel != nil {
		s.cancel()
	}
	return metadata, err
}

func (s *cancelAfterWebhookSecretStore) DeleteSecret(ctx context.Context, ref string) error {
	err := s.store.DeleteSecret(ctx, ref)
	if err == nil && s.cancelOnDelete && s.cancel != nil {
		s.cancel()
	}
	return err
}
