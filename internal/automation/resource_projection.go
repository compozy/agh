package automation

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/pedronauck/agh/internal/resources"
	"github.com/pedronauck/agh/internal/store"
)

type jobResourceProjectionPlan struct {
	revision   int64
	operations int
	jobs       []Job
	scheduler  *Scheduler
}

func (p *jobResourceProjectionPlan) Kind() resources.ResourceKind {
	return JobResourceKind
}

func (p *jobResourceProjectionPlan) Revision() int64 {
	if p == nil {
		return 0
	}
	return p.revision
}

func (p *jobResourceProjectionPlan) OperationCount() int {
	if p == nil {
		return 0
	}
	return p.operations
}

type triggerResourceProjectionPlan struct {
	revision   int64
	operations int
	triggers   []Trigger
	engine     *TriggerEngine
}

func (p *triggerResourceProjectionPlan) Kind() resources.ResourceKind {
	return TriggerResourceKind
}

func (p *triggerResourceProjectionPlan) Revision() int64 {
	if p == nil {
		return 0
	}
	return p.revision
}

func (p *triggerResourceProjectionPlan) OperationCount() int {
	if p == nil {
		return 0
	}
	return p.operations
}

// BuildJobResourceState builds the next scheduler plan from canonical automation.job records.
func (m *Manager) BuildJobResourceState(
	ctx context.Context,
	records []resources.Record[Job],
) (resources.ProjectionPlan, error) {
	if ctx == nil {
		return nil, errors.New("automation: job resource build context is required")
	}
	if m == nil {
		return nil, errors.New("automation: manager is required")
	}

	jobs := make([]Job, 0, len(records))
	var revision int64
	for _, record := range records {
		if record.Version > revision {
			revision = record.Version
		}
		job := cloneJob(record.Spec)
		job.ID = strings.TrimSpace(record.ID)
		job.CreatedAt = record.CreatedAt.UTC()
		job.UpdatedAt = record.UpdatedAt.UTC()
		jobs = append(jobs, job)
	}
	sortJobs(jobs)

	effectiveJobs, err := m.applyJobQueryAndOverlays(ctx, jobs, JobListQuery{})
	if err != nil {
		return nil, err
	}

	scheduler, err := m.buildSchedulerRuntime(ctx)
	if err != nil {
		return nil, err
	}
	if err := m.loadSchedulerRegistrations(ctx, effectiveJobs, scheduler); err != nil {
		return nil, errors.Join(err, m.shutdownRuntimeComponent(ctx, "scheduler", scheduler))
	}

	return &jobResourceProjectionPlan{
		revision:   revision,
		operations: len(effectiveJobs),
		jobs:       cloneJobs(jobs),
		scheduler:  scheduler,
	}, nil
}

// ApplyJobResourceState atomically swaps the scheduler and desired job catalog.
func (m *Manager) ApplyJobResourceState(ctx context.Context, plan resources.ProjectionPlan) error {
	if ctx == nil {
		return errors.New("automation: job resource apply context is required")
	}
	if m == nil {
		return errors.New("automation: manager is required")
	}

	typed, ok := plan.(*jobResourceProjectionPlan)
	if !ok {
		return fmt.Errorf("automation: job resource plan has type %T", plan)
	}
	if typed.scheduler == nil {
		return errors.New("automation: job resource plan scheduler is required")
	}

	m.mu.Lock()
	running := m.running
	m.mu.Unlock()

	if running {
		if err := typed.scheduler.Start(ctx); err != nil {
			return errors.Join(err, m.shutdownRuntimeComponent(ctx, "scheduler", typed.scheduler))
		}
	}

	nextJobs := jobMapFromSlice(typed.jobs)
	if !running {
		m.mu.Lock()
		m.projectedJobs = nextJobs
		m.jobRevision = typed.revision
		m.mu.Unlock()
		return m.shutdownRuntimeComponent(ctx, "scheduler", typed.scheduler)
	}

	m.mu.Lock()
	if running && !m.running {
		m.mu.Unlock()
		return errors.Join(ErrManagerNotRunning, m.shutdownRuntimeComponent(ctx, "scheduler", typed.scheduler))
	}
	oldScheduler := m.scheduler
	m.scheduler = typed.scheduler
	m.projectedJobs = nextJobs
	m.jobRevision = typed.revision
	m.mu.Unlock()

	if oldScheduler != nil {
		if err := m.shutdownRuntimeComponent(ctx, "scheduler", oldScheduler); err != nil {
			m.logger.Warn("automation.resource.job.cleanup_failed", "error", err)
		}
	}
	return nil
}

// BuildTriggerResourceState builds the next trigger-engine plan from canonical automation.trigger records.
func (m *Manager) BuildTriggerResourceState(
	ctx context.Context,
	records []resources.Record[Trigger],
) (resources.ProjectionPlan, error) {
	if ctx == nil {
		return nil, errors.New("automation: trigger resource build context is required")
	}
	if m == nil {
		return nil, errors.New("automation: manager is required")
	}

	triggers := make([]Trigger, 0, len(records))
	var revision int64
	for _, record := range records {
		if record.Version > revision {
			revision = record.Version
		}
		trigger := cloneTrigger(record.Spec)
		trigger.ID = strings.TrimSpace(record.ID)
		trigger.CreatedAt = record.CreatedAt.UTC()
		trigger.UpdatedAt = record.UpdatedAt.UTC()
		if strings.EqualFold(strings.TrimSpace(trigger.Event), "webhook") &&
			strings.TrimSpace(trigger.WebhookID) == "" {
			trigger.WebhookID = stableConfigID("wbh", trigger.ID)
		}
		triggers = append(triggers, trigger)
	}
	sortTriggers(triggers)

	effectiveTriggers, err := m.applyTriggerQueryAndOverlays(ctx, triggers, TriggerListQuery{})
	if err != nil {
		return nil, err
	}

	engine, err := m.buildTriggerRuntime(ctx)
	if err != nil {
		return nil, err
	}
	if err := m.loadTriggerRegistrations(effectiveTriggers, engine); err != nil {
		return nil, errors.Join(err, m.shutdownRuntimeComponent(ctx, "trigger engine", engine))
	}

	return &triggerResourceProjectionPlan{
		revision:   revision,
		operations: len(effectiveTriggers),
		triggers:   cloneTriggers(triggers),
		engine:     engine,
	}, nil
}

// ApplyTriggerResourceState atomically swaps the trigger engine and desired trigger catalog.
func (m *Manager) ApplyTriggerResourceState(ctx context.Context, plan resources.ProjectionPlan) error {
	if ctx == nil {
		return errors.New("automation: trigger resource apply context is required")
	}
	if m == nil {
		return errors.New("automation: manager is required")
	}

	typed, ok := plan.(*triggerResourceProjectionPlan)
	if !ok {
		return fmt.Errorf("automation: trigger resource plan has type %T", plan)
	}
	if typed.engine == nil {
		return errors.New("automation: trigger resource plan engine is required")
	}

	m.mu.Lock()
	running := m.running
	m.mu.Unlock()

	if running {
		if err := typed.engine.Start(ctx); err != nil {
			return errors.Join(err, m.shutdownRuntimeComponent(ctx, "trigger engine", typed.engine))
		}
	}

	nextTriggers := triggerMapFromSlice(typed.triggers)
	if !running {
		m.mu.Lock()
		m.projectedTriggers = nextTriggers
		m.triggerRevision = typed.revision
		m.mu.Unlock()
		return m.shutdownRuntimeComponent(ctx, "trigger engine", typed.engine)
	}

	m.mu.Lock()
	if running && !m.running {
		m.mu.Unlock()
		return errors.Join(ErrManagerNotRunning, m.shutdownRuntimeComponent(ctx, "trigger engine", typed.engine))
	}
	oldEngine := m.triggers
	m.triggers = typed.engine
	m.projectedTriggers = nextTriggers
	m.triggerRevision = typed.revision
	m.mu.Unlock()

	if oldEngine != nil {
		if err := m.shutdownRuntimeComponent(ctx, "trigger engine", oldEngine); err != nil {
			m.logger.Warn("automation.resource.trigger.cleanup_failed", "error", err)
		}
	}
	return nil
}

func (m *Manager) resourceDefinitionsEnabled() bool {
	return m != nil && m.jobResources != nil && m.triggerResources != nil
}

func defaultAutomationResourceActor() resources.MutationActor {
	return resources.MutationActor{
		Kind:     resources.MutationActorKindDaemon,
		ID:       "automation-resource-sync",
		Source:   resources.ResourceSource{Kind: resources.ResourceSourceKind("daemon"), ID: "automation"},
		MaxScope: resources.ResourceScope{Kind: resources.ResourceScopeKindGlobal},
	}
}

func (m *Manager) resourceActorForSource(source JobSource) resources.MutationActor {
	actor := m.resourceActor
	if actor.Kind == "" {
		actor = defaultAutomationResourceActor()
	}
	sourceID := "automation." + strings.TrimSpace(string(source))
	actor.ID = sourceID
	actor.Source = resources.ResourceSource{Kind: resources.ResourceSourceKind("daemon"), ID: sourceID}
	return actor
}

func (m *Manager) createJobResource(ctx context.Context, job Job) (Job, error) {
	next := cloneJob(job)
	if next.Source == "" {
		next.Source = JobSourceDynamic
	}
	if next.Source != JobSourceDynamic {
		return Job{}, ErrDefinitionReadOnly
	}
	if strings.TrimSpace(next.ID) == "" {
		next.ID = store.NewID("job")
	}
	next.CreatedAt = m.now().UTC()
	next.UpdatedAt = next.CreatedAt
	if err := next.Validate("job"); err != nil {
		return Job{}, err
	}
	if _, err := m.jobResources.Put(ctx, m.resourceActorForSource(JobSourceDynamic), resources.Draft[Job]{
		ID:              next.ID,
		Scope:           ResourceScopeForAutomation(next.Scope, next.WorkspaceID),
		ExpectedVersion: 0,
		Spec:            next,
	}); err != nil {
		return Job{}, err
	}
	if err := m.applyJobResourcesFromStore(ctx); err != nil {
		return Job{}, err
	}
	return m.effectiveJob(ctx, next.ID)
}

func (m *Manager) updateJobResource(ctx context.Context, job Job) (Job, error) {
	current, err := m.jobResources.Get(ctx, m.resourceActor, strings.TrimSpace(job.ID))
	if err != nil {
		return Job{}, err
	}
	if current.Spec.Source != JobSourceDynamic {
		return Job{}, ErrDefinitionReadOnly
	}

	next := cloneJob(job)
	next.ID = current.ID
	next.Source = current.Spec.Source
	next.CreatedAt = current.CreatedAt.UTC()
	next.UpdatedAt = m.now().UTC()
	if err := next.Validate("job"); err != nil {
		return Job{}, err
	}
	if _, err := m.jobResources.Put(ctx, currentResourceActor(current.Source, m.resourceActor), resources.Draft[Job]{
		ID:              current.ID,
		Scope:           ResourceScopeForAutomation(next.Scope, next.WorkspaceID),
		ExpectedVersion: current.Version,
		Spec:            next,
	}); err != nil {
		return Job{}, err
	}
	if err := m.applyJobResourcesFromStore(ctx); err != nil {
		return Job{}, err
	}
	return m.effectiveJob(ctx, current.ID)
}

func (m *Manager) deleteJobResource(ctx context.Context, id string) error {
	current, err := m.jobResources.Get(ctx, m.resourceActor, strings.TrimSpace(id))
	if err != nil {
		return err
	}
	if current.Spec.Source != JobSourceDynamic {
		return ErrDefinitionReadOnly
	}
	if err := m.jobResources.Delete(
		ctx,
		currentResourceActor(current.Source, m.resourceActor),
		current.ID,
		current.Version,
	); err != nil {
		return err
	}
	return m.applyJobResourcesFromStore(ctx)
}

func (m *Manager) createTriggerResource(
	ctx context.Context,
	trigger Trigger,
	webhookSecret WebhookSecretWrite,
) (Trigger, error) {
	next := cloneTrigger(trigger)
	if next.Source == "" {
		next.Source = JobSourceDynamic
	}
	if next.Source != JobSourceDynamic {
		return Trigger{}, ErrDefinitionReadOnly
	}
	if strings.TrimSpace(next.ID) == "" {
		next.ID = store.NewID("trg")
	}
	if strings.EqualFold(strings.TrimSpace(next.Event), "webhook") &&
		strings.TrimSpace(next.WebhookID) == "" {
		next.WebhookID = stableConfigID("wbh", next.ID)
	}
	next = applyWebhookSecretRef(next, nil, &webhookSecret)
	next.CreatedAt = m.now().UTC()
	next.UpdatedAt = next.CreatedAt
	if err := requireWebhookSecretRef(next); err != nil {
		return Trigger{}, err
	}
	if err := next.Validate("trigger"); err != nil {
		return Trigger{}, err
	}
	if err := m.applyWebhookSecretWrite(ctx, next, webhookSecret); err != nil {
		return Trigger{}, err
	}
	draft := resources.Draft[Trigger]{
		ID:              next.ID,
		Scope:           ResourceScopeForAutomation(next.Scope, next.WorkspaceID),
		ExpectedVersion: 0,
		Spec:            next,
	}
	if _, err := m.triggerResources.Put(ctx, m.resourceActorForSource(JobSourceDynamic), draft); err != nil {
		return Trigger{}, errors.Join(err, m.deleteOwnedWebhookSecretIfPresent(ctx, next))
	}
	if err := m.applyTriggerResourcesFromStore(ctx); err != nil {
		return Trigger{}, err
	}
	return m.effectiveTrigger(ctx, next.ID)
}

func (m *Manager) updateTriggerResource(
	ctx context.Context,
	trigger Trigger,
	webhookSecret *WebhookSecretWrite,
) (Trigger, error) {
	current, err := m.triggerResources.Get(ctx, m.resourceActor, strings.TrimSpace(trigger.ID))
	if err != nil {
		return Trigger{}, err
	}
	if current.Spec.Source != JobSourceDynamic {
		return Trigger{}, ErrDefinitionReadOnly
	}
	next := cloneTrigger(trigger)
	next.ID = current.ID
	next.Source = current.Spec.Source
	next.CreatedAt = current.CreatedAt.UTC()
	next.UpdatedAt = m.now().UTC()
	next = applyWebhookSecretRef(next, &current.Spec, webhookSecret)
	if strings.EqualFold(strings.TrimSpace(next.Event), "webhook") &&
		strings.TrimSpace(next.WebhookID) == "" {
		next.WebhookID = stableConfigID("wbh", next.ID)
	}
	if err := requireWebhookSecretRef(next); err != nil {
		return Trigger{}, err
	}
	if err := next.Validate("trigger"); err != nil {
		return Trigger{}, err
	}
	if err := m.applyWebhookSecretWritePointer(ctx, next, webhookSecret); err != nil {
		return Trigger{}, err
	}
	if _, err := m.triggerResources.Put(
		ctx,
		currentResourceActor(current.Source, m.resourceActor),
		resources.Draft[Trigger]{
			ID:              current.ID,
			Scope:           ResourceScopeForAutomation(next.Scope, next.WorkspaceID),
			ExpectedVersion: current.Version,
			Spec:            next,
		},
	); err != nil {
		return Trigger{}, err
	}
	if err := m.deleteSupersededOwnedWebhookSecret(ctx, current.Spec, next); err != nil {
		return Trigger{}, err
	}
	if err := m.applyTriggerResourcesFromStore(ctx); err != nil {
		return Trigger{}, err
	}
	return m.effectiveTrigger(ctx, current.ID)
}

func (m *Manager) deleteTriggerResource(ctx context.Context, id string) error {
	current, err := m.triggerResources.Get(ctx, m.resourceActor, strings.TrimSpace(id))
	if err != nil {
		return err
	}
	if current.Spec.Source != JobSourceDynamic {
		return ErrDefinitionReadOnly
	}
	if err := m.triggerResources.Delete(
		ctx,
		currentResourceActor(current.Source, m.resourceActor),
		current.ID,
		current.Version,
	); err != nil {
		return err
	}
	if err := m.deleteOwnedWebhookSecretIfPresent(ctx, current.Spec); err != nil {
		return err
	}
	return m.applyTriggerResourcesFromStore(ctx)
}

func (m *Manager) setJobResourceEnabled(ctx context.Context, id string, enabled bool) (Job, error) {
	current, err := m.projectedJobDefinition(id)
	if err != nil {
		return Job{}, err
	}
	if isOverlayManagedSource(current.Source) {
		if err := m.persistJobOverlay(ctx, current, enabled); err != nil {
			return Job{}, err
		}
		currentEffective, err := m.effectiveJob(ctx, current.ID)
		if err != nil {
			return Job{}, err
		}
		if err := m.applyJobToRuntime(ctx, currentEffective); err != nil {
			return Job{}, err
		}
		return currentEffective, nil
	}

	current.Enabled = enabled
	return m.updateJobResource(ctx, current)
}

func (m *Manager) setTriggerResourceEnabled(ctx context.Context, id string, enabled bool) (Trigger, error) {
	current, err := m.projectedTriggerDefinition(id)
	if err != nil {
		return Trigger{}, err
	}
	if isOverlayManagedSource(current.Source) {
		if err := m.persistTriggerOverlay(ctx, current, enabled); err != nil {
			return Trigger{}, err
		}
		currentEffective, err := m.effectiveTrigger(ctx, current.ID)
		if err != nil {
			return Trigger{}, err
		}
		if err := m.applyTriggerToRuntime(currentEffective); err != nil {
			return Trigger{}, err
		}
		return currentEffective, nil
	}

	current.Enabled = enabled
	return m.updateTriggerResource(ctx, current, nil)
}

func (m *Manager) syncManagedResourceDefinitions(
	ctx context.Context,
	source JobSource,
	desiredJobs []Job,
	desiredTriggers []Trigger,
) (SyncStats, error) {
	actor := m.resourceActorForSource(source)

	jobsSynced, jobsRemoved, err := m.syncJobResourcesForSource(ctx, actor, desiredJobs)
	if err != nil {
		return SyncStats{}, err
	}
	triggersSynced, triggersRemoved, err := m.syncTriggerResourcesForSource(
		ctx,
		actor,
		desiredTriggers,
	)
	if err != nil {
		return SyncStats{}, err
	}

	if err := m.triggerResourceReconcile(ctx, JobResourceKind); err != nil {
		return SyncStats{}, err
	}
	if err := m.triggerResourceReconcile(ctx, TriggerResourceKind); err != nil {
		return SyncStats{}, err
	}

	stats := SyncStats{
		JobsSynced:      jobsSynced,
		TriggersSynced:  triggersSynced,
		JobsRemoved:     jobsRemoved,
		TriggersRemoved: triggersRemoved,
		SyncedAt:        m.now().UTC(),
	}
	m.logger.Info(
		"automation.managed.resource_sync",
		"source", source,
		"jobs_synced", stats.JobsSynced,
		"triggers_synced", stats.TriggersSynced,
		"jobs_removed", stats.JobsRemoved,
		"triggers_removed", stats.TriggersRemoved,
	)
	return stats, nil
}

func (m *Manager) syncJobResourcesForSource(
	ctx context.Context,
	actor resources.MutationActor,
	desired []Job,
) (int, int, error) {
	source := actor.Source
	current, err := m.jobResources.List(ctx, actor, resources.ResourceFilter{
		Kind:   JobResourceKind,
		Source: &source,
	})
	if err != nil {
		return 0, 0, err
	}
	currentByID := make(map[string]resources.Record[Job], len(current))
	for _, record := range current {
		currentByID[record.ID] = record
	}

	synced := 0
	for _, job := range desired {
		next := cloneJob(job)
		next.Source = JobSource(strings.TrimSpace(string(next.Source)))
		if strings.TrimSpace(next.ID) == "" {
			return 0, 0, errors.New("automation: managed job id is required")
		}
		currentRecord, exists := currentByID[next.ID]
		if exists && currentRecord.Scope == ResourceScopeForAutomation(next.Scope, next.WorkspaceID) &&
			sameJobDefinition(currentRecord.Spec, next) {
			delete(currentByID, next.ID)
			synced++
			continue
		}

		expectedVersion := int64(0)
		if exists {
			expectedVersion = currentRecord.Version
		}
		if _, err := m.jobResources.Put(ctx, actor, resources.Draft[Job]{
			ID:              next.ID,
			Scope:           ResourceScopeForAutomation(next.Scope, next.WorkspaceID),
			ExpectedVersion: expectedVersion,
			Spec:            next,
		}); err != nil {
			return 0, 0, err
		}
		delete(currentByID, next.ID)
		synced++
	}

	removed := 0
	for _, stale := range currentByID {
		if err := m.jobResources.Delete(ctx, actor, stale.ID, stale.Version); err != nil {
			return 0, 0, err
		}
		removed++
	}
	return synced, removed, nil
}

func (m *Manager) syncTriggerResourcesForSource(
	ctx context.Context,
	actor resources.MutationActor,
	desired []Trigger,
) (int, int, error) {
	source := actor.Source
	current, err := m.triggerResources.List(ctx, actor, resources.ResourceFilter{
		Kind:   TriggerResourceKind,
		Source: &source,
	})
	if err != nil {
		return 0, 0, err
	}
	currentByID := make(map[string]resources.Record[Trigger], len(current))
	for _, record := range current {
		currentByID[record.ID] = record
	}

	synced := 0
	for _, trigger := range desired {
		next := cloneTrigger(trigger)
		if strings.TrimSpace(next.ID) == "" {
			return 0, 0, errors.New("automation: managed trigger id is required")
		}
		if strings.EqualFold(strings.TrimSpace(next.Event), "webhook") && strings.TrimSpace(next.WebhookID) == "" {
			next.WebhookID = stableConfigID("wbh", next.ID)
		}

		currentRecord, exists := currentByID[next.ID]
		if exists && currentRecord.Scope == ResourceScopeForAutomation(next.Scope, next.WorkspaceID) &&
			sameTriggerDefinition(currentRecord.Spec, next) {
			delete(currentByID, next.ID)
			synced++
			continue
		}

		expectedVersion := int64(0)
		if exists {
			expectedVersion = currentRecord.Version
		}
		if _, err := m.triggerResources.Put(ctx, actor, resources.Draft[Trigger]{
			ID:              next.ID,
			Scope:           ResourceScopeForAutomation(next.Scope, next.WorkspaceID),
			ExpectedVersion: expectedVersion,
			Spec:            next,
		}); err != nil {
			return 0, 0, err
		}
		delete(currentByID, next.ID)
		synced++
	}

	removed := 0
	for _, stale := range currentByID {
		if err := m.triggerResources.Delete(ctx, actor, stale.ID, stale.Version); err != nil {
			return 0, 0, err
		}
		if err := m.deleteOwnedWebhookSecretIfPresent(ctx, stale.Spec); err != nil {
			return 0, 0, err
		}
		removed++
	}
	return synced, removed, nil
}

func (m *Manager) applyJobResourcesFromStore(ctx context.Context) error {
	records, err := m.jobResources.List(ctx, m.resourceActor, resources.ResourceFilter{Kind: JobResourceKind})
	if err != nil {
		return err
	}
	plan, err := m.BuildJobResourceState(ctx, records)
	if err != nil {
		return err
	}
	return m.ApplyJobResourceState(ctx, plan)
}

func (m *Manager) applyTriggerResourcesFromStore(ctx context.Context) error {
	records, err := m.triggerResources.List(ctx, m.resourceActor, resources.ResourceFilter{Kind: TriggerResourceKind})
	if err != nil {
		return err
	}
	plan, err := m.BuildTriggerResourceState(ctx, records)
	if err != nil {
		return err
	}
	return m.ApplyTriggerResourceState(ctx, plan)
}

func (m *Manager) loadProjectedJobDefinitionsFromStore(ctx context.Context) ([]Job, int64, error) {
	records, err := m.jobResources.List(ctx, m.resourceActor, resources.ResourceFilter{Kind: JobResourceKind})
	if err != nil {
		return nil, 0, err
	}

	jobs := make([]Job, 0, len(records))
	var revision int64
	for _, record := range records {
		if record.Version > revision {
			revision = record.Version
		}
		job := cloneJob(record.Spec)
		job.ID = strings.TrimSpace(record.ID)
		job.CreatedAt = record.CreatedAt.UTC()
		job.UpdatedAt = record.UpdatedAt.UTC()
		jobs = append(jobs, job)
	}
	sortJobs(jobs)
	return jobs, revision, nil
}

func (m *Manager) loadProjectedTriggerDefinitionsFromStore(ctx context.Context) ([]Trigger, int64, error) {
	records, err := m.triggerResources.List(ctx, m.resourceActor, resources.ResourceFilter{Kind: TriggerResourceKind})
	if err != nil {
		return nil, 0, err
	}

	triggers := make([]Trigger, 0, len(records))
	var revision int64
	for _, record := range records {
		if record.Version > revision {
			revision = record.Version
		}
		trigger := cloneTrigger(record.Spec)
		trigger.ID = strings.TrimSpace(record.ID)
		trigger.CreatedAt = record.CreatedAt.UTC()
		trigger.UpdatedAt = record.UpdatedAt.UTC()
		if strings.EqualFold(strings.TrimSpace(trigger.Event), "webhook") &&
			strings.TrimSpace(trigger.WebhookID) == "" {
			trigger.WebhookID = stableConfigID("wbh", trigger.ID)
		}
		triggers = append(triggers, trigger)
	}
	sortTriggers(triggers)
	return triggers, revision, nil
}

func (m *Manager) triggerResourceReconcile(ctx context.Context, kind resources.ResourceKind) error {
	if m == nil || m.resourceTrigger == nil {
		return nil
	}
	return m.resourceTrigger(ctx, kind, resources.ReconcileReasonWrite)
}

func currentResourceActor(source resources.ResourceSource, fallback resources.MutationActor) resources.MutationActor {
	actor := fallback
	if actor.Kind == "" {
		actor = defaultAutomationResourceActor()
	}
	actor.Source = source.Normalize()
	actor.ID = source.ID
	if strings.TrimSpace(actor.ID) == "" {
		actor.ID = "automation-resource"
	}
	return actor
}

func (m *Manager) projectedJobDefinitions() []Job {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.projectedJobDefinitionsLocked()
}

func (m *Manager) projectedJobDefinitionsLocked() []Job {
	jobs := make([]Job, 0, len(m.projectedJobs))
	for _, job := range m.projectedJobs {
		jobs = append(jobs, cloneJob(job))
	}
	sortJobs(jobs)
	return jobs
}

func (m *Manager) projectedTriggerDefinitions() []Trigger {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.projectedTriggerDefinitionsLocked()
}

func (m *Manager) projectedTriggerDefinitionsLocked() []Trigger {
	triggers := make([]Trigger, 0, len(m.projectedTriggers))
	for _, trigger := range m.projectedTriggers {
		triggers = append(triggers, cloneTrigger(trigger))
	}
	sortTriggers(triggers)
	return triggers
}

func (m *Manager) projectedJobDefinition(id string) (Job, error) {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return Job{}, ErrJobNotFound
	}
	m.mu.RLock()
	job, ok := m.projectedJobs[trimmedID]
	m.mu.RUnlock()
	if !ok {
		return Job{}, ErrJobNotFound
	}
	return cloneJob(job), nil
}

func (m *Manager) projectedTriggerDefinition(id string) (Trigger, error) {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return Trigger{}, ErrTriggerNotFound
	}
	m.mu.RLock()
	trigger, ok := m.projectedTriggers[trimmedID]
	m.mu.RUnlock()
	if !ok {
		return Trigger{}, ErrTriggerNotFound
	}
	return cloneTrigger(trigger), nil
}

func (m *Manager) applyJobQueryAndOverlays(
	ctx context.Context,
	jobs []Job,
	query JobListQuery,
) ([]Job, error) {
	overlays, err := m.store.ListJobEnabledOverlays(ctx)
	if err != nil {
		return nil, err
	}
	overlayByID := make(map[string]bool, len(overlays))
	for _, overlay := range overlays {
		overlayByID[overlay.JobID] = overlay.EnabledOverride
	}

	effective := make([]Job, 0, len(jobs))
	for _, job := range jobs {
		if query.Scope != "" && job.Scope != query.Scope {
			continue
		}
		if query.WorkspaceID != "" && job.WorkspaceID != strings.TrimSpace(query.WorkspaceID) {
			continue
		}
		if query.Source != "" && job.Source != query.Source {
			continue
		}
		next := cloneJob(job)
		if isOverlayManagedSource(next.Source) {
			if enabled, ok := overlayByID[next.ID]; ok {
				next.Enabled = enabled
			}
		}
		effective = append(effective, next)
	}
	sortJobs(effective)
	if query.Limit > 0 && len(effective) > query.Limit {
		effective = effective[:query.Limit]
	}
	return effective, nil
}

func (m *Manager) applyTriggerQueryAndOverlays(
	ctx context.Context,
	triggers []Trigger,
	query TriggerListQuery,
) ([]Trigger, error) {
	overlays, err := m.store.ListTriggerEnabledOverlays(ctx)
	if err != nil {
		return nil, err
	}
	overlayByID := make(map[string]bool, len(overlays))
	for _, overlay := range overlays {
		overlayByID[overlay.TriggerID] = overlay.EnabledOverride
	}

	effective := make([]Trigger, 0, len(triggers))
	for _, trigger := range triggers {
		if query.Scope != "" && trigger.Scope != query.Scope {
			continue
		}
		if query.WorkspaceID != "" && trigger.WorkspaceID != strings.TrimSpace(query.WorkspaceID) {
			continue
		}
		if query.Event != "" && trigger.Event != strings.TrimSpace(query.Event) {
			continue
		}
		if query.Source != "" && trigger.Source != query.Source {
			continue
		}
		next := cloneTrigger(trigger)
		if isOverlayManagedSource(next.Source) {
			if enabled, ok := overlayByID[next.ID]; ok {
				next.Enabled = enabled
			}
		}
		effective = append(effective, next)
	}
	sortTriggers(effective)
	if query.Limit > 0 && len(effective) > query.Limit {
		effective = effective[:query.Limit]
	}
	return effective, nil
}

func jobMapFromSlice(jobs []Job) map[string]Job {
	byID := make(map[string]Job, len(jobs))
	for _, job := range jobs {
		byID[job.ID] = cloneJob(job)
	}
	return byID
}

func triggerMapFromSlice(triggers []Trigger) map[string]Trigger {
	byID := make(map[string]Trigger, len(triggers))
	for _, trigger := range triggers {
		byID[trigger.ID] = cloneTrigger(trigger)
	}
	return byID
}

func cloneJobs(jobs []Job) []Job {
	if len(jobs) == 0 {
		return nil
	}
	cloned := make([]Job, 0, len(jobs))
	for _, job := range jobs {
		cloned = append(cloned, cloneJob(job))
	}
	return cloned
}

func cloneTriggers(triggers []Trigger) []Trigger {
	if len(triggers) == 0 {
		return nil
	}
	cloned := make([]Trigger, 0, len(triggers))
	for _, trigger := range triggers {
		cloned = append(cloned, cloneTrigger(trigger))
	}
	return cloned
}
