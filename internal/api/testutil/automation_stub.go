package testutil

import (
	"context"

	core "github.com/pedronauck/agh/internal/api/core"
	automationpkg "github.com/pedronauck/agh/internal/automation"
)

type StubAutomationManager struct {
	ListJobsFn      func(context.Context, automationpkg.JobListQuery) ([]automationpkg.Job, error)
	JobsFn          func(context.Context) ([]automationpkg.Job, error)
	GetJobFn        func(context.Context, string) (automationpkg.Job, error)
	CreateJobFn     func(context.Context, automationpkg.Job) (automationpkg.Job, error)
	UpdateJobFn     func(context.Context, automationpkg.Job) (automationpkg.Job, error)
	DeleteJobFn     func(context.Context, string) error
	TriggerJobFn    func(context.Context, string) (automationpkg.Run, error)
	ListTriggersFn  func(context.Context, automationpkg.TriggerListQuery) ([]automationpkg.Trigger, error)
	TriggersFn      func(context.Context) ([]automationpkg.Trigger, error)
	GetTriggerFn    func(context.Context, string) (automationpkg.Trigger, error)
	CreateTriggerFn func(
		context.Context,
		automationpkg.Trigger,
		automationpkg.WebhookSecretWrite,
	) (automationpkg.Trigger, error)
	UpdateTriggerFn func(
		context.Context,
		automationpkg.Trigger,
		*automationpkg.WebhookSecretWrite,
	) (automationpkg.Trigger, error)
	DeleteTriggerFn     func(context.Context, string) error
	ListRunsFn          func(context.Context, automationpkg.RunQuery) ([]automationpkg.Run, error)
	RunsFn              func(context.Context, automationpkg.RunQuery) ([]automationpkg.Run, error)
	GetRunFn            func(context.Context, string) (automationpkg.Run, error)
	StatusFn            func(context.Context) (automationpkg.ManagerStatus, error)
	SetJobEnabledFn     func(context.Context, string, bool) (automationpkg.Job, error)
	SetTriggerEnabledFn func(context.Context, string, bool) (automationpkg.Trigger, error)
	HandleWebhookFn     func(context.Context, automationpkg.WebhookRequest) (automationpkg.TriggerResult, error)
}

func (s StubAutomationManager) ListJobs(
	ctx context.Context,
	query automationpkg.JobListQuery,
) ([]automationpkg.Job, error) {
	if s.ListJobsFn != nil {
		return s.ListJobsFn(ctx, query)
	}
	if s.JobsFn != nil {
		return s.JobsFn(ctx)
	}
	return nil, nil
}

func (s StubAutomationManager) Jobs(ctx context.Context) ([]automationpkg.Job, error) {
	if s.JobsFn != nil {
		return s.JobsFn(ctx)
	}
	return s.ListJobs(ctx, automationpkg.JobListQuery{})
}

func (s StubAutomationManager) GetJob(ctx context.Context, id string) (automationpkg.Job, error) {
	if s.GetJobFn != nil {
		return s.GetJobFn(ctx, id)
	}
	return automationpkg.Job{}, automationpkg.ErrJobNotFound
}

func (s StubAutomationManager) CreateJob(ctx context.Context, job automationpkg.Job) (automationpkg.Job, error) {
	if s.CreateJobFn != nil {
		return s.CreateJobFn(ctx, job)
	}
	return job, nil
}

func (s StubAutomationManager) UpdateJob(ctx context.Context, job automationpkg.Job) (automationpkg.Job, error) {
	if s.UpdateJobFn != nil {
		return s.UpdateJobFn(ctx, job)
	}
	return job, nil
}

func (s StubAutomationManager) DeleteJob(ctx context.Context, id string) error {
	if s.DeleteJobFn != nil {
		return s.DeleteJobFn(ctx, id)
	}
	return nil
}

func (s StubAutomationManager) TriggerJob(ctx context.Context, id string) (automationpkg.Run, error) {
	if s.TriggerJobFn != nil {
		return s.TriggerJobFn(ctx, id)
	}
	return automationpkg.Run{}, nil
}

func (s StubAutomationManager) ListTriggers(
	ctx context.Context,
	query automationpkg.TriggerListQuery,
) ([]automationpkg.Trigger, error) {
	if s.ListTriggersFn != nil {
		return s.ListTriggersFn(ctx, query)
	}
	if s.TriggersFn != nil {
		return s.TriggersFn(ctx)
	}
	return nil, nil
}

func (s StubAutomationManager) Triggers(ctx context.Context) ([]automationpkg.Trigger, error) {
	if s.TriggersFn != nil {
		return s.TriggersFn(ctx)
	}
	return s.ListTriggers(ctx, automationpkg.TriggerListQuery{})
}

func (s StubAutomationManager) GetTrigger(ctx context.Context, id string) (automationpkg.Trigger, error) {
	if s.GetTriggerFn != nil {
		return s.GetTriggerFn(ctx, id)
	}
	return automationpkg.Trigger{}, automationpkg.ErrTriggerNotFound
}

func (s StubAutomationManager) CreateTrigger(
	ctx context.Context,
	trigger automationpkg.Trigger,
	secret automationpkg.WebhookSecretWrite,
) (automationpkg.Trigger, error) {
	if s.CreateTriggerFn != nil {
		return s.CreateTriggerFn(ctx, trigger, secret)
	}
	return trigger, nil
}

func (s StubAutomationManager) UpdateTrigger(
	ctx context.Context,
	trigger automationpkg.Trigger,
	secret *automationpkg.WebhookSecretWrite,
) (automationpkg.Trigger, error) {
	if s.UpdateTriggerFn != nil {
		return s.UpdateTriggerFn(ctx, trigger, secret)
	}
	return trigger, nil
}

func (s StubAutomationManager) DeleteTrigger(ctx context.Context, id string) error {
	if s.DeleteTriggerFn != nil {
		return s.DeleteTriggerFn(ctx, id)
	}
	return nil
}

func (s StubAutomationManager) ListRuns(
	ctx context.Context,
	query automationpkg.RunQuery,
) ([]automationpkg.Run, error) {
	if s.ListRunsFn != nil {
		return s.ListRunsFn(ctx, query)
	}
	if s.RunsFn != nil {
		return s.RunsFn(ctx, query)
	}
	return nil, nil
}

func (s StubAutomationManager) Runs(ctx context.Context, query automationpkg.RunQuery) ([]automationpkg.Run, error) {
	if s.RunsFn != nil {
		return s.RunsFn(ctx, query)
	}
	return s.ListRuns(ctx, query)
}

func (s StubAutomationManager) GetRun(ctx context.Context, id string) (automationpkg.Run, error) {
	if s.GetRunFn != nil {
		return s.GetRunFn(ctx, id)
	}
	return automationpkg.Run{}, automationpkg.ErrRunNotFound
}

func (s StubAutomationManager) Status(ctx context.Context) (automationpkg.ManagerStatus, error) {
	if s.StatusFn != nil {
		return s.StatusFn(ctx)
	}
	return automationpkg.ManagerStatus{}, nil
}

func (s StubAutomationManager) SetJobEnabled(ctx context.Context, id string, enabled bool) (automationpkg.Job, error) {
	if s.SetJobEnabledFn != nil {
		return s.SetJobEnabledFn(ctx, id, enabled)
	}
	return automationpkg.Job{}, automationpkg.ErrJobNotFound
}

func (s StubAutomationManager) SetTriggerEnabled(
	ctx context.Context,
	id string,
	enabled bool,
) (automationpkg.Trigger, error) {
	if s.SetTriggerEnabledFn != nil {
		return s.SetTriggerEnabledFn(ctx, id, enabled)
	}
	return automationpkg.Trigger{}, automationpkg.ErrTriggerNotFound
}

func (s StubAutomationManager) HandleWebhook(
	ctx context.Context,
	request automationpkg.WebhookRequest,
) (automationpkg.TriggerResult, error) {
	if s.HandleWebhookFn != nil {
		return s.HandleWebhookFn(ctx, request)
	}
	return automationpkg.TriggerResult{}, nil
}

var _ core.AutomationManager = (*StubAutomationManager)(nil)
