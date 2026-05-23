package daemon

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/compozy/agh/internal/api/contract"
	core "github.com/compozy/agh/internal/api/core"
	automationpkg "github.com/compozy/agh/internal/automation"
	toolspkg "github.com/compozy/agh/internal/tools"
)

const (
	nativeAutomationToolsDeletedKey = "deleted"
	nativeAutomationToolsJobKey     = "job"
	nativeAutomationToolsRunsKey    = "runs"
	nativeAutomationToolsTriggerKey = "trigger"
)

func (n *daemonNativeTools) automationToolBindings(
	availability toolspkg.NativeAvailabilityFunc,
) map[toolspkg.ToolID]nativeToolBinding {
	return map[toolspkg.ToolID]nativeToolBinding{
		toolspkg.ToolIDAutomationJobsList: {
			call:         n.automationJobsList,
			availability: availability,
		},
		toolspkg.ToolIDAutomationJobsGet: {
			call:         n.automationJobsGet,
			availability: availability,
		},
		toolspkg.ToolIDAutomationJobsCreate: {
			call:         n.automationJobsCreate,
			availability: availability,
		},
		toolspkg.ToolIDAutomationJobsUpdate: {
			call:         n.automationJobsUpdate,
			availability: availability,
		},
		toolspkg.ToolIDAutomationJobsDelete: {
			call:         n.automationJobsDelete,
			availability: availability,
		},
		toolspkg.ToolIDAutomationJobsEnable: {
			call:         n.automationJobsEnable,
			availability: availability,
		},
		toolspkg.ToolIDAutomationJobsDisable: {
			call:         n.automationJobsDisable,
			availability: availability,
		},
		toolspkg.ToolIDAutomationJobsTrigger: {
			call:         n.automationJobsTrigger,
			availability: availability,
		},
		toolspkg.ToolIDAutomationJobsHistory: {
			call:         n.automationJobsHistory,
			availability: availability,
		},
		toolspkg.ToolIDAutomationTriggersList: {
			call:         n.automationTriggersList,
			availability: availability,
		},
		toolspkg.ToolIDAutomationTriggersGet: {
			call:         n.automationTriggersGet,
			availability: availability,
		},
		toolspkg.ToolIDAutomationTriggersCreate: {
			call:         n.automationTriggersCreate,
			availability: availability,
		},
		toolspkg.ToolIDAutomationTriggersUpdate: {
			call:         n.automationTriggersUpdate,
			availability: availability,
		},
		toolspkg.ToolIDAutomationTriggersDelete: {
			call:         n.automationTriggersDelete,
			availability: availability,
		},
		toolspkg.ToolIDAutomationTriggersEnable: {
			call:         n.automationTriggersEnable,
			availability: availability,
		},
		toolspkg.ToolIDAutomationTriggersDisable: {
			call:         n.automationTriggersDisable,
			availability: availability,
		},
		toolspkg.ToolIDAutomationTriggersHistory: {
			call:         n.automationTriggersHistory,
			availability: availability,
		},
		toolspkg.ToolIDAutomationRunsList: {
			call:         n.automationRunsList,
			availability: availability,
		},
		toolspkg.ToolIDAutomationRunsGet: {
			call:         n.automationRunsGet,
			availability: availability,
		},
	}
}

func (n *daemonNativeTools) automationJobsList(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input automationJobsListInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	query, err := input.query(req.ToolID)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	jobs, err := n.automationManager().ListJobs(ctx, query)
	if err != nil {
		return toolspkg.ToolResult{}, nativeAutomationToolError(req.ToolID, err)
	}
	payload, err := n.automationJobPayloads(ctx, jobs)
	if err != nil {
		return toolspkg.ToolResult{}, nativeAutomationToolError(req.ToolID, err)
	}
	return structuredResult(map[string]any{"jobs": payload}, fmt.Sprintf("%d automation jobs", len(payload)))
}

func (n *daemonNativeTools) automationJobsGet(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input automationJobIDInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	jobID, err := requiredNativeString(req.ToolID, "job_id", input.JobID)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	job, err := n.automationManager().GetJob(ctx, jobID)
	if err != nil {
		return toolspkg.ToolResult{}, nativeAutomationToolError(req.ToolID, err)
	}
	payload, err := n.automationJobPayload(ctx, job)
	if err != nil {
		return toolspkg.ToolResult{}, nativeAutomationToolError(req.ToolID, err)
	}
	return structuredResult(map[string]any{nativeAutomationToolsJobKey: payload}, payload.ID)
}

func (n *daemonNativeTools) automationJobsCreate(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input automationJobCreateInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	job := core.AutomationJobFromCreateRequest(input.request())
	if err := job.Validate(nativeAutomationToolsJobKey); err != nil {
		return toolspkg.ToolResult{}, nativeAutomationValidationError(req.ToolID, err)
	}
	created, err := n.automationManager().CreateJob(ctx, job)
	if err != nil {
		return toolspkg.ToolResult{}, nativeAutomationToolError(req.ToolID, err)
	}
	payload := n.automationJobPayloadBestEffort(ctx, created)
	return structuredResult(map[string]any{nativeAutomationToolsJobKey: payload}, payload.ID)
}

func (n *daemonNativeTools) automationJobsUpdate(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input automationJobUpdateInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	jobID, err := requiredNativeString(req.ToolID, "job_id", input.JobID)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	patch := input.request()
	if !patch.HasChanges() {
		return toolspkg.ToolResult{}, nativeAutomationValidationError(
			req.ToolID,
			errors.New("automation job update must include at least one field"),
		)
	}
	current, err := n.automationManager().GetJob(ctx, jobID)
	if err != nil {
		return toolspkg.ToolResult{}, nativeAutomationToolError(req.ToolID, err)
	}
	var updated automationpkg.Job
	switch current.Source {
	case automationpkg.JobSourceConfig:
		if err := core.ValidateAutomationConfigJobUpdate(patch); err != nil {
			return toolspkg.ToolResult{}, nativeAutomationValidationError(req.ToolID, err)
		}
		updated, err = n.automationManager().SetJobEnabled(ctx, current.ID, *patch.Enabled)
	default:
		next := core.ApplyAutomationJobPatch(current, patch)
		if err := next.Validate(nativeAutomationToolsJobKey); err != nil {
			return toolspkg.ToolResult{}, nativeAutomationValidationError(req.ToolID, err)
		}
		updated, err = n.automationManager().UpdateJob(ctx, next)
	}
	if err != nil {
		return toolspkg.ToolResult{}, nativeAutomationToolError(req.ToolID, err)
	}
	payload := n.automationJobPayloadBestEffort(ctx, updated)
	return structuredResult(map[string]any{nativeAutomationToolsJobKey: payload}, payload.ID)
}

func (n *daemonNativeTools) automationJobsDelete(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input automationJobIDInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	jobID, err := requiredNativeString(req.ToolID, "job_id", input.JobID)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	current, err := n.automationManager().GetJob(ctx, jobID)
	if err != nil {
		return toolspkg.ToolResult{}, nativeAutomationToolError(req.ToolID, err)
	}
	if current.Source != automationpkg.JobSourceDynamic {
		return toolspkg.ToolResult{}, nativeAutomationScopeError(
			req.ToolID,
			nativeAutomationToolsJobKey,
			current.ID,
			current.Source,
		)
	}
	if err := n.automationManager().DeleteJob(ctx, current.ID); err != nil {
		return toolspkg.ToolResult{}, nativeAutomationToolError(req.ToolID, err)
	}
	return structuredResult(map[string]any{"job_id": current.ID, nativeAutomationToolsDeletedKey: true}, current.ID)
}

func (n *daemonNativeTools) automationJobsEnable(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	return n.automationSetJobEnabled(ctx, req, true)
}

func (n *daemonNativeTools) automationJobsDisable(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	return n.automationSetJobEnabled(ctx, req, false)
}

func (n *daemonNativeTools) automationJobsTrigger(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input automationJobIDInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	jobID, err := requiredNativeString(req.ToolID, "job_id", input.JobID)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	run, err := n.automationManager().TriggerJob(ctx, jobID)
	if err != nil {
		return toolspkg.ToolResult{}, nativeAutomationToolError(req.ToolID, err)
	}
	payload := core.RunPayloadFromRun(run)
	return structuredResult(map[string]any{"run": payload}, payload.ID)
}

func (n *daemonNativeTools) automationJobsHistory(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input automationJobHistoryInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	jobID, err := requiredNativeString(req.ToolID, "job_id", input.JobID)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	job, err := n.automationManager().GetJob(ctx, jobID)
	if err != nil {
		return toolspkg.ToolResult{}, nativeAutomationToolError(req.ToolID, err)
	}
	query, err := input.query(req.ToolID)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	query.JobID = job.ID
	query.TriggerID = ""
	return n.automationRunsForQuery(ctx, req.ToolID, query)
}

func (n *daemonNativeTools) automationTriggersList(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input automationTriggersListInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	query, err := input.query(req.ToolID)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	triggers, err := n.automationManager().ListTriggers(ctx, query)
	if err != nil {
		return toolspkg.ToolResult{}, nativeAutomationToolError(req.ToolID, err)
	}
	payload := core.TriggerPayloadsFromTriggers(triggers)
	return structuredResult(map[string]any{"triggers": payload}, fmt.Sprintf("%d automation triggers", len(payload)))
}

func (n *daemonNativeTools) automationTriggersGet(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input automationTriggerIDInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	triggerID, err := requiredNativeString(req.ToolID, "trigger_id", input.TriggerID)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	trigger, err := n.automationManager().GetTrigger(ctx, triggerID)
	if err != nil {
		return toolspkg.ToolResult{}, nativeAutomationToolError(req.ToolID, err)
	}
	payload := core.TriggerPayloadFromTrigger(trigger)
	return structuredResult(map[string]any{nativeAutomationToolsTriggerKey: payload}, payload.ID)
}

func (n *daemonNativeTools) automationTriggersCreate(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input automationTriggerCreateInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	trigger := core.AutomationTriggerFromCreateRequest(input.request())
	created, err := n.automationManager().CreateTrigger(ctx, trigger, input.webhookSecretWrite())
	if err != nil {
		return toolspkg.ToolResult{}, nativeAutomationToolError(req.ToolID, err)
	}
	payload := core.TriggerPayloadFromTrigger(created)
	return structuredResult(map[string]any{nativeAutomationToolsTriggerKey: payload}, payload.ID)
}

func (n *daemonNativeTools) automationTriggersUpdate(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input automationTriggerUpdateInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	triggerID, err := requiredNativeString(req.ToolID, "trigger_id", input.TriggerID)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	patch := input.request()
	if !patch.HasChanges() {
		return toolspkg.ToolResult{}, nativeAutomationValidationError(
			req.ToolID,
			errors.New("automation trigger update must include at least one field"),
		)
	}
	current, err := n.automationManager().GetTrigger(ctx, triggerID)
	if err != nil {
		return toolspkg.ToolResult{}, nativeAutomationToolError(req.ToolID, err)
	}
	var updated automationpkg.Trigger
	switch current.Source {
	case automationpkg.JobSourceConfig:
		if err := core.ValidateAutomationConfigTriggerUpdate(patch); err != nil {
			return toolspkg.ToolResult{}, nativeAutomationValidationError(req.ToolID, err)
		}
		updated, err = n.automationManager().SetTriggerEnabled(ctx, current.ID, *patch.Enabled)
	default:
		next := core.ApplyAutomationTriggerPatch(current, patch)
		updated, err = n.automationManager().UpdateTrigger(ctx, next, input.webhookSecretWrite())
	}
	if err != nil {
		return toolspkg.ToolResult{}, nativeAutomationToolError(req.ToolID, err)
	}
	payload := core.TriggerPayloadFromTrigger(updated)
	return structuredResult(map[string]any{nativeAutomationToolsTriggerKey: payload}, payload.ID)
}

func (n *daemonNativeTools) automationTriggersDelete(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input automationTriggerIDInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	triggerID, err := requiredNativeString(req.ToolID, "trigger_id", input.TriggerID)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	current, err := n.automationManager().GetTrigger(ctx, triggerID)
	if err != nil {
		return toolspkg.ToolResult{}, nativeAutomationToolError(req.ToolID, err)
	}
	if current.Source != automationpkg.JobSourceDynamic {
		return toolspkg.ToolResult{}, nativeAutomationScopeError(
			req.ToolID,
			nativeAutomationToolsTriggerKey,
			current.ID,
			current.Source,
		)
	}
	if err := n.automationManager().DeleteTrigger(ctx, current.ID); err != nil {
		return toolspkg.ToolResult{}, nativeAutomationToolError(req.ToolID, err)
	}
	return structuredResult(map[string]any{"trigger_id": current.ID, nativeAutomationToolsDeletedKey: true}, current.ID)
}

func (n *daemonNativeTools) automationTriggersEnable(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	return n.automationSetTriggerEnabled(ctx, req, true)
}

func (n *daemonNativeTools) automationTriggersDisable(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	return n.automationSetTriggerEnabled(ctx, req, false)
}

func (n *daemonNativeTools) automationTriggersHistory(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input automationTriggerHistoryInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	triggerID, err := requiredNativeString(req.ToolID, "trigger_id", input.TriggerID)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	trigger, err := n.automationManager().GetTrigger(ctx, triggerID)
	if err != nil {
		return toolspkg.ToolResult{}, nativeAutomationToolError(req.ToolID, err)
	}
	query, err := input.query(req.ToolID)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	query.JobID = ""
	query.TriggerID = trigger.ID
	return n.automationRunsForQuery(ctx, req.ToolID, query)
}

func (n *daemonNativeTools) automationRunsList(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input automationRunQueryInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	query, err := input.query(req.ToolID)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	return n.automationRunsForQuery(ctx, req.ToolID, query)
}

func (n *daemonNativeTools) automationRunsGet(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input automationRunIDInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	runID, err := requiredNativeString(req.ToolID, "run_id", input.RunID)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	run, err := n.automationManager().GetRun(ctx, runID)
	if err != nil {
		return toolspkg.ToolResult{}, nativeAutomationToolError(req.ToolID, err)
	}
	payload := core.RunPayloadFromRun(run)
	return structuredResult(map[string]any{"run": payload}, payload.ID)
}

func (n *daemonNativeTools) automationSetJobEnabled(
	ctx context.Context,
	req toolspkg.CallRequest,
	enabled bool,
) (toolspkg.ToolResult, error) {
	var input automationJobIDInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	jobID, err := requiredNativeString(req.ToolID, "job_id", input.JobID)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	updated, err := n.automationManager().SetJobEnabled(ctx, jobID, enabled)
	if err != nil {
		return toolspkg.ToolResult{}, nativeAutomationToolError(req.ToolID, err)
	}
	payload := n.automationJobPayloadBestEffort(ctx, updated)
	return structuredResult(map[string]any{nativeAutomationToolsJobKey: payload}, payload.ID)
}

func (n *daemonNativeTools) automationSetTriggerEnabled(
	ctx context.Context,
	req toolspkg.CallRequest,
	enabled bool,
) (toolspkg.ToolResult, error) {
	var input automationTriggerIDInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	triggerID, err := requiredNativeString(req.ToolID, "trigger_id", input.TriggerID)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	updated, err := n.automationManager().SetTriggerEnabled(ctx, triggerID, enabled)
	if err != nil {
		return toolspkg.ToolResult{}, nativeAutomationToolError(req.ToolID, err)
	}
	payload := core.TriggerPayloadFromTrigger(updated)
	return structuredResult(map[string]any{nativeAutomationToolsTriggerKey: payload}, payload.ID)
}

func (n *daemonNativeTools) automationRunsForQuery(
	ctx context.Context,
	toolID toolspkg.ToolID,
	query automationpkg.RunQuery,
) (toolspkg.ToolResult, error) {
	runs, err := n.automationManager().ListRuns(ctx, query)
	if err != nil {
		return toolspkg.ToolResult{}, nativeAutomationToolError(toolID, err)
	}
	payload := core.RunPayloadsFromRuns(runs)
	return structuredResult(
		map[string]any{nativeAutomationToolsRunsKey: payload},
		fmt.Sprintf("%d automation runs", len(payload)),
	)
}

func (n *daemonNativeTools) automationJobPayloads(
	ctx context.Context,
	jobs []automationpkg.Job,
) ([]contract.JobPayload, error) {
	stateByID, err := n.automationSchedulerStateByJobID(ctx)
	if err != nil {
		return nil, err
	}
	return core.JobPayloadsFromJobs(jobs, stateByID), nil
}

func (n *daemonNativeTools) automationJobPayload(
	ctx context.Context,
	job automationpkg.Job,
) (contract.JobPayload, error) {
	payloads, err := n.automationJobPayloads(ctx, []automationpkg.Job{job})
	if err != nil {
		return contract.JobPayload{}, err
	}
	if len(payloads) == 0 {
		return contract.JobPayload{}, errors.New("daemon: automation job payload missing")
	}
	return payloads[0], nil
}

func (n *daemonNativeTools) automationJobPayloadBestEffort(
	ctx context.Context,
	job automationpkg.Job,
) contract.JobPayload {
	payload, err := n.automationJobPayload(ctx, job)
	if err == nil {
		return payload
	}
	return core.JobPayloadFromJob(job, nil, nil)
}

func (n *daemonNativeTools) automationSchedulerStateByJobID(
	ctx context.Context,
) (map[string]contract.AutomationSchedulerStatePayload, error) {
	status, err := n.automationManager().Status(ctx)
	if err != nil {
		return nil, err
	}
	stateByID := make(map[string]contract.AutomationSchedulerStatePayload, len(status.ScheduledJobs))
	for _, scheduled := range status.ScheduledJobs {
		stateByID[scheduled.JobID] = core.AutomationSchedulerStatePayloadFromState(scheduled)
	}
	return stateByID, nil
}

type automationJobsListInput struct {
	Scope       string `json:"scope,omitempty"`
	WorkspaceID string `json:"workspace_id,omitempty"`
	Source      string `json:"source,omitempty"`
	Limit       int    `json:"limit,omitempty"`
}

func (i automationJobsListInput) query(id toolspkg.ToolID) (automationpkg.JobListQuery, error) {
	query := automationpkg.JobListQuery{
		WorkspaceID: strings.TrimSpace(i.WorkspaceID),
		Limit:       i.Limit,
	}
	if scope := strings.TrimSpace(i.Scope); scope != "" {
		query.Scope = automationpkg.Scope(scope)
		if err := query.Scope.Validate("scope"); err != nil {
			return automationpkg.JobListQuery{}, nativeAutomationValidationError(id, err)
		}
	}
	if source := strings.TrimSpace(i.Source); source != "" {
		query.Source = automationpkg.JobSource(source)
		if err := query.Source.Validate("source"); err != nil {
			return automationpkg.JobListQuery{}, nativeAutomationValidationError(id, err)
		}
	}
	return query, nil
}

type automationTriggersListInput struct {
	Scope       string `json:"scope,omitempty"`
	WorkspaceID string `json:"workspace_id,omitempty"`
	Event       string `json:"event,omitempty"`
	Source      string `json:"source,omitempty"`
	Limit       int    `json:"limit,omitempty"`
}

func (i automationTriggersListInput) query(id toolspkg.ToolID) (automationpkg.TriggerListQuery, error) {
	query := automationpkg.TriggerListQuery{
		WorkspaceID: strings.TrimSpace(i.WorkspaceID),
		Event:       strings.TrimSpace(i.Event),
		Limit:       i.Limit,
	}
	if scope := strings.TrimSpace(i.Scope); scope != "" {
		query.Scope = automationpkg.Scope(scope)
		if err := query.Scope.Validate("scope"); err != nil {
			return automationpkg.TriggerListQuery{}, nativeAutomationValidationError(id, err)
		}
	}
	if source := strings.TrimSpace(i.Source); source != "" {
		query.Source = automationpkg.JobSource(source)
		if err := query.Source.Validate("source"); err != nil {
			return automationpkg.TriggerListQuery{}, nativeAutomationValidationError(id, err)
		}
	}
	return query, nil
}

type automationJobIDInput struct {
	JobID string `json:"job_id"`
}

type automationTriggerIDInput struct {
	TriggerID string `json:"trigger_id"`
}

type automationRunIDInput struct {
	RunID string `json:"run_id"`
}

type automationJobCreateInput struct {
	Scope       automationpkg.Scope            `json:"scope"`
	Name        string                         `json:"name"`
	AgentName   string                         `json:"agent_name"`
	WorkspaceID string                         `json:"workspace_id,omitempty"`
	Prompt      string                         `json:"prompt"`
	Schedule    automationpkg.ScheduleSpec     `json:"schedule"`
	Task        *automationpkg.JobTaskConfig   `json:"task,omitempty"`
	Enabled     *bool                          `json:"enabled,omitempty"`
	Retry       *automationpkg.RetryConfig     `json:"retry,omitempty"`
	FireLimit   *automationpkg.FireLimitConfig `json:"fire_limit,omitempty"`
}

func (i automationJobCreateInput) request() contract.CreateJobRequest {
	return contract.CreateJobRequest{
		Scope:       i.Scope,
		Name:        i.Name,
		AgentName:   i.AgentName,
		WorkspaceID: i.WorkspaceID,
		Prompt:      i.Prompt,
		Schedule:    i.Schedule,
		Task:        i.Task,
		Enabled:     i.Enabled,
		Retry:       i.Retry,
		FireLimit:   i.FireLimit,
	}
}

type automationJobUpdateInput struct {
	JobID       string                         `json:"job_id"`
	Name        *string                        `json:"name,omitempty"`
	AgentName   *string                        `json:"agent_name,omitempty"`
	WorkspaceID *string                        `json:"workspace_id,omitempty"`
	Prompt      *string                        `json:"prompt,omitempty"`
	Schedule    *automationpkg.ScheduleSpec    `json:"schedule,omitempty"`
	Task        *automationpkg.JobTaskConfig   `json:"task,omitempty"`
	Enabled     *bool                          `json:"enabled,omitempty"`
	Retry       *automationpkg.RetryConfig     `json:"retry,omitempty"`
	FireLimit   *automationpkg.FireLimitConfig `json:"fire_limit,omitempty"`
}

func (i automationJobUpdateInput) request() contract.UpdateJobRequest {
	return contract.UpdateJobRequest{
		Name:        i.Name,
		AgentName:   i.AgentName,
		WorkspaceID: i.WorkspaceID,
		Prompt:      i.Prompt,
		Schedule:    i.Schedule,
		Task:        i.Task,
		Enabled:     i.Enabled,
		Retry:       i.Retry,
		FireLimit:   i.FireLimit,
	}
}

type automationTriggerCreateInput struct {
	Scope              automationpkg.Scope            `json:"scope"`
	Name               string                         `json:"name"`
	AgentName          string                         `json:"agent_name"`
	WorkspaceID        string                         `json:"workspace_id,omitempty"`
	Prompt             string                         `json:"prompt"`
	Event              string                         `json:"event"`
	Filter             map[string]string              `json:"filter,omitempty"`
	Enabled            *bool                          `json:"enabled,omitempty"`
	Retry              *automationpkg.RetryConfig     `json:"retry,omitempty"`
	FireLimit          *automationpkg.FireLimitConfig `json:"fire_limit,omitempty"`
	WebhookID          string                         `json:"webhook_id,omitempty"`
	EndpointSlug       string                         `json:"endpoint_slug,omitempty"`
	WebhookSecretValue *string                        `json:"webhook_secret_value,omitempty"`
}

func (i automationTriggerCreateInput) request() contract.CreateTriggerRequest {
	return contract.CreateTriggerRequest{
		Scope:        i.Scope,
		Name:         i.Name,
		AgentName:    i.AgentName,
		WorkspaceID:  i.WorkspaceID,
		Prompt:       i.Prompt,
		Event:        i.Event,
		Filter:       i.Filter,
		Enabled:      i.Enabled,
		Retry:        i.Retry,
		FireLimit:    i.FireLimit,
		WebhookID:    i.WebhookID,
		EndpointSlug: i.EndpointSlug,
	}
}

func (i automationTriggerCreateInput) webhookSecretWrite() automationpkg.WebhookSecretWrite {
	write := automationpkg.WebhookSecretWrite{}
	if i.WebhookSecretValue != nil {
		value := strings.TrimSpace(*i.WebhookSecretValue)
		write.Value = &value
	}
	return write
}

type automationTriggerUpdateInput struct {
	TriggerID          string                         `json:"trigger_id"`
	Name               *string                        `json:"name,omitempty"`
	AgentName          *string                        `json:"agent_name,omitempty"`
	WorkspaceID        *string                        `json:"workspace_id,omitempty"`
	Prompt             *string                        `json:"prompt,omitempty"`
	Event              *string                        `json:"event,omitempty"`
	Filter             map[string]string              `json:"filter,omitempty"`
	Enabled            *bool                          `json:"enabled,omitempty"`
	Retry              *automationpkg.RetryConfig     `json:"retry,omitempty"`
	FireLimit          *automationpkg.FireLimitConfig `json:"fire_limit,omitempty"`
	WebhookID          *string                        `json:"webhook_id,omitempty"`
	EndpointSlug       *string                        `json:"endpoint_slug,omitempty"`
	WebhookSecretValue *string                        `json:"webhook_secret_value,omitempty"`
}

func (i automationTriggerUpdateInput) request() contract.UpdateTriggerRequest {
	return contract.UpdateTriggerRequest{
		Name:               i.Name,
		AgentName:          i.AgentName,
		WorkspaceID:        i.WorkspaceID,
		Prompt:             i.Prompt,
		Event:              i.Event,
		Filter:             i.Filter,
		Enabled:            i.Enabled,
		Retry:              i.Retry,
		FireLimit:          i.FireLimit,
		WebhookID:          i.WebhookID,
		EndpointSlug:       i.EndpointSlug,
		WebhookSecretValue: i.WebhookSecretValue,
	}
}

func (i automationTriggerUpdateInput) webhookSecretWrite() *automationpkg.WebhookSecretWrite {
	if i.WebhookSecretValue == nil {
		return nil
	}
	write := automationpkg.WebhookSecretWrite{}
	value := strings.TrimSpace(*i.WebhookSecretValue)
	write.Value = &value
	return &write
}

type automationJobHistoryInput struct {
	JobID string `json:"job_id"`
	automationRunQueryInput
}

type automationTriggerHistoryInput struct {
	TriggerID string `json:"trigger_id"`
	automationRunQueryInput
}

type automationRunQueryInput struct {
	JobID     string `json:"job_id,omitempty"`
	TriggerID string `json:"trigger_id,omitempty"`
	Status    string `json:"status,omitempty"`
	Since     string `json:"since,omitempty"`
	Until     string `json:"until,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}

func (i automationRunQueryInput) query(id toolspkg.ToolID) (automationpkg.RunQuery, error) {
	since, err := parseNativeAutomationOptionalRFC3339(id, "since", i.Since)
	if err != nil {
		return automationpkg.RunQuery{}, err
	}
	until, err := parseNativeAutomationOptionalRFC3339(id, "until", i.Until)
	if err != nil {
		return automationpkg.RunQuery{}, err
	}
	query := automationpkg.RunQuery{
		JobID:     strings.TrimSpace(i.JobID),
		TriggerID: strings.TrimSpace(i.TriggerID),
		Since:     since,
		Until:     until,
		Limit:     i.Limit,
	}
	if rawStatus := strings.TrimSpace(i.Status); rawStatus != "" {
		query.Status = automationpkg.RunStatus(rawStatus)
		if err := query.Status.Validate("status"); err != nil {
			return automationpkg.RunQuery{}, nativeAutomationValidationError(id, err)
		}
	}
	return query, nil
}

func parseNativeAutomationOptionalRFC3339(
	id toolspkg.ToolID,
	field string,
	raw string,
) (time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return time.Time{}, nil
	}
	timestamp, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return time.Time{}, nativeAutomationValidationError(
			id,
			fmt.Errorf("%s must be an RFC3339 timestamp: %w", field, err),
		)
	}
	return timestamp, nil
}

func nativeAutomationToolError(id toolspkg.ToolID, err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, automationpkg.ErrJobNotFound),
		errors.Is(err, automationpkg.ErrTriggerNotFound),
		errors.Is(err, automationpkg.ErrRunNotFound),
		errors.Is(err, automationpkg.ErrJobOverlayNotFound),
		errors.Is(err, automationpkg.ErrTriggerOverlayNotFound):
		return toolspkg.NewToolError(
			toolspkg.ErrorCodeNotFound,
			id,
			err.Error(),
			fmt.Errorf("%w: %w", toolspkg.ErrToolNotFound, err),
			toolspkg.ReasonToolUnknown,
		)
	case errors.Is(err, automationpkg.ErrDefinitionReadOnly):
		return toolspkg.NewToolError(
			toolspkg.ErrorCodeDenied,
			id,
			err.Error(),
			fmt.Errorf("%w: %w", toolspkg.ErrToolDenied, err),
			toolspkg.ReasonAutomationScopeForbidden,
		)
	case errors.Is(err, automationpkg.ErrManagerNotRunning):
		return toolspkg.NewToolError(
			toolspkg.ErrorCodeUnavailable,
			id,
			err.Error(),
			fmt.Errorf("%w: %w", toolspkg.ErrToolUnavailable, err),
			toolspkg.ReasonDependencyMissing,
		)
	case errors.Is(err, automationpkg.ErrFireLimitReached),
		errors.Is(err, automationpkg.ErrConcurrencyLimitReached),
		errors.Is(err, automationpkg.ErrScheduledFireAlreadyClaimed):
		return toolspkg.NewToolError(
			toolspkg.ErrorCodeConflict,
			id,
			err.Error(),
			fmt.Errorf("%w: %w", toolspkg.ErrToolConflict, err),
			toolspkg.ReasonAutomationValidationFailed,
		)
	case errors.Is(err, automationpkg.ErrJobNameTaken),
		errors.Is(err, automationpkg.ErrTriggerNameTaken),
		errors.Is(err, automationpkg.ErrTriggerWebhookIDTaken),
		errors.Is(err, automationpkg.ErrWebhookSecretRequired),
		errors.Is(err, automationpkg.ErrOverlayRequiresConfigSource):
		return nativeAutomationValidationError(id, err)
	default:
		return err
	}
}

func nativeAutomationValidationError(id toolspkg.ToolID, err error) error {
	return toolspkg.NewToolError(
		toolspkg.ErrorCodeInvalidInput,
		id,
		"automation validation failed",
		fmt.Errorf("%w: %w", toolspkg.ErrToolInvalidInput, err),
		toolspkg.ReasonAutomationValidationFailed,
	)
}

func nativeAutomationScopeError(
	id toolspkg.ToolID,
	resource string,
	resourceID string,
	source automationpkg.JobSource,
) error {
	return toolspkg.NewToolError(
		toolspkg.ErrorCodeDenied,
		id,
		fmt.Sprintf("automation %s %q source %q is immutable by tools", resource, resourceID, source),
		toolspkg.ErrToolDenied,
		toolspkg.ReasonAutomationScopeForbidden,
	)
}
