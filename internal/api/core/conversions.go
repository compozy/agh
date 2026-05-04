package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/api/contract"
	automationpkg "github.com/pedronauck/agh/internal/automation"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/diagnostics"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	observepkg "github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/resources"
	"github.com/pedronauck/agh/internal/session"
	settingspkg "github.com/pedronauck/agh/internal/settings"
	"github.com/pedronauck/agh/internal/skills"
	"github.com/pedronauck/agh/internal/store"
	taskpkg "github.com/pedronauck/agh/internal/task"
	"github.com/pedronauck/agh/internal/workref"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

const maxDiagnosticPayloadBytes = 2048

// SessionPayloadFromInfo converts a session info snapshot into the shared session payload.
func SessionPayloadFromInfo(info *session.Info) contract.SessionPayload {
	payload := contract.SessionPayload{}
	if info == nil {
		return payload
	}

	ref := workref.NewPath(info.WorkspaceID, info.Workspace)
	payload = contract.SessionPayload{
		ID:            info.ID,
		Name:          info.Name,
		AgentName:     info.AgentName,
		Provider:      info.Provider,
		WorkspaceID:   ref.WorkspaceID,
		WorkspacePath: ref.WorkspacePath,
		Channel:       info.Channel,
		Type:          info.Type,
		State:         info.State,
		StopReason:    info.StopReason,
		StopDetail:    info.StopDetail,
		Failure:       SessionFailurePayloadFromStore(info.Failure),
		ACPSessionID:  info.ACPSessionID,
		Lineage:       contract.SessionLineagePayloadFromStore(info.Lineage),
		CreatedAt:     info.CreatedAt,
		UpdatedAt:     info.UpdatedAt,
	}
	if caps := ACPCapsPayloadFromInfo(info.ACPCaps); caps != nil {
		payload.ACPCaps = caps
	}
	if activity := RuntimeActivityPayloadFromSessionMeta(info.Liveness, time.Now().UTC()); activity != nil {
		payload.Activity = activity
	}
	if sandbox := SessionSandboxPayloadFromMeta(info.Sandbox); sandbox != nil {
		payload.Sandbox = sandbox
	}
	return payload
}

// RuntimeActivityPayloadFromSessionMeta converts persisted session activity metadata into the shared payload.
func RuntimeActivityPayloadFromSessionMeta(
	liveness *store.SessionLivenessMeta,
	now time.Time,
) *contract.RuntimeActivityPayload {
	if liveness == nil || liveness.Activity == nil {
		return nil
	}
	activity := store.CloneSessionActivityMeta(liveness.Activity)
	payload := &contract.RuntimeActivityPayload{
		TurnID:             activity.TurnID,
		TurnSource:         activity.TurnSource,
		TurnStartedAt:      cloneTimePtr(activity.TurnStartedAt),
		LastActivityAt:     cloneTimePtr(activity.LastActivityAt),
		LastActivityKind:   activity.LastActivityKind,
		LastActivityDetail: activity.LastActivityDetail,
		CurrentTool:        activity.CurrentTool,
		ToolCallID:         activity.ToolCallID,
		LastProgressAt:     cloneTimePtr(activity.LastProgressAt),
		IterationCurrent:   activity.IterationCurrent,
		IterationMax:       activity.IterationMax,
		IdleSeconds:        store.SessionActivityIdleSeconds(activity, now),
	}
	if !now.IsZero() && activity.TurnStartedAt != nil && !activity.TurnStartedAt.IsZero() {
		elapsed := now.UTC().Sub(activity.TurnStartedAt.UTC())
		if elapsed > 0 {
			payload.ElapsedSeconds = int64(elapsed.Seconds())
		}
	}
	return payload
}

func runtimeActivityPayloadFromEvent(activity *acp.RuntimeActivity) *contract.RuntimeActivityPayload {
	if activity == nil {
		return nil
	}
	return &contract.RuntimeActivityPayload{
		TurnID:             strings.TrimSpace(activity.TurnID),
		TurnSource:         strings.TrimSpace(activity.TurnSource),
		TurnStartedAt:      cloneTimePtr(activity.TurnStartedAt),
		LastActivityAt:     cloneTimePtr(activity.LastActivityAt),
		LastActivityKind:   strings.TrimSpace(activity.LastActivityKind),
		LastActivityDetail: strings.TrimSpace(activity.LastActivityDetail),
		CurrentTool:        strings.TrimSpace(activity.CurrentTool),
		ToolCallID:         strings.TrimSpace(activity.ToolCallID),
		LastProgressAt:     cloneTimePtr(activity.LastProgressAt),
		IterationCurrent:   activity.IterationCurrent,
		IterationMax:       activity.IterationMax,
		IdleSeconds:        activity.IdleSeconds,
		ElapsedSeconds:     activity.ElapsedSeconds,
	}
}

// SessionSandboxPayloadFromMeta converts session sandbox metadata into the shared payload.
func SessionSandboxPayloadFromMeta(meta *store.SessionSandboxMeta) *contract.SessionSandboxPayload {
	if meta == nil {
		return nil
	}
	return &contract.SessionSandboxPayload{
		SandboxID:     strings.TrimSpace(meta.SandboxID),
		Backend:       strings.TrimSpace(meta.Backend),
		Profile:       strings.TrimSpace(meta.Profile),
		State:         strings.TrimSpace(meta.State),
		InstanceID:    strings.TrimSpace(meta.InstanceID),
		LastSyncError: strings.TrimSpace(meta.LastSyncError),
	}
}

// SessionPayloadsFromInfos converts a session list into response payloads.
func SessionPayloadsFromInfos(infos []*session.Info) []contract.SessionPayload {
	payload := make([]contract.SessionPayload, 0, len(infos))
	for _, info := range infos {
		if info == nil {
			continue
		}
		payload = append(payload, SessionPayloadFromInfo(info))
	}
	return payload
}

// SessionFailurePayloadFromStore converts a stored failure diagnostic into the
// shared API payload.
func SessionFailurePayloadFromStore(failure *store.SessionFailure) *contract.SessionFailurePayload {
	if failure == nil {
		return nil
	}
	normalized := failure.Normalize()
	if normalized.IsZero() {
		return nil
	}
	return &contract.SessionFailurePayload{
		Kind:            normalized.Kind,
		Summary:         diagnostics.RedactAndBound(normalized.Summary, maxDiagnosticPayloadBytes),
		CrashBundlePath: diagnostics.RedactAndBound(normalized.CrashBundlePath, maxDiagnosticPayloadBytes),
	}
}

// ACPCapsPayloadFromInfo converts ACP capability info into the shared payload.
func ACPCapsPayloadFromInfo(caps acp.Caps) *contract.ACPCapsPayload {
	if !caps.SupportsLoadSession && len(caps.SupportedModes) == 0 && len(caps.SupportedModels) == 0 {
		return nil
	}

	return &contract.ACPCapsPayload{
		SupportsLoadSession: caps.SupportsLoadSession,
		SupportedModes:      append([]string(nil), caps.SupportedModes...),
		SupportedModels:     append([]string(nil), caps.SupportedModels...),
	}
}

// SessionEventPayloadFromEvent converts a session event into the shared payload.
func SessionEventPayloadFromEvent(event store.SessionEvent, info *session.Info) contract.SessionEventPayload {
	ref := workref.NewPath(sessionWorkspaceFromInfo(info))
	payload := contract.SessionEventPayload{
		ID:            event.ID,
		SessionID:     event.SessionID,
		Sequence:      event.Sequence,
		TurnID:        event.TurnID,
		Type:          event.Type,
		AgentName:     event.AgentName,
		WorkspaceID:   ref.WorkspaceID,
		WorkspacePath: ref.WorkspacePath,
		Content:       PayloadJSON(event.Content),
		Timestamp:     event.Timestamp,
	}
	if info != nil && info.Lineage != nil {
		lineage := store.NormalizeSessionLineage(event.SessionID, info.Lineage)
		payload.ParentSessionID = lineage.ParentSessionID
		payload.RootSessionID = lineage.RootSessionID
		payload.SpawnDepth = lineage.SpawnDepth
	}
	if info != nil && event.Type == session.EventTypeSessionStopped {
		payload.StopReason = info.StopReason
		payload.StopDetail = info.StopDetail
		payload.Failure = SessionFailurePayloadFromStore(info.Failure)
	}
	return payload
}

// SessionRepairPayloadFromResult converts a session repair report into the shared payload.
func SessionRepairPayloadFromResult(result *session.RepairResult) contract.SessionRepairPayload {
	if result == nil {
		return contract.SessionRepairPayload{}
	}

	issues := make([]contract.SessionRepairIssuePayload, 0, len(result.Issues))
	for _, issue := range result.Issues {
		issues = append(issues, contract.SessionRepairIssuePayload{
			Code:     issue.Code,
			Severity: issue.Severity,
			TurnID:   issue.TurnID,
			EventID:  issue.EventID,
			Detail:   issue.Detail,
		})
	}

	actions := make([]contract.SessionRepairActionPayload, 0, len(result.Actions))
	for _, action := range result.Actions {
		actions = append(actions, contract.SessionRepairActionPayload{
			Code:       action.Code,
			TurnID:     action.TurnID,
			EventID:    action.EventID,
			ToolCallID: action.ToolCallID,
			ToolName:   action.ToolName,
			Persisted:  action.Persisted,
		})
	}

	return contract.SessionRepairPayload{
		SessionID: result.SessionID,
		Issues:    issues,
		Actions:   actions,
		Persisted: result.Persisted,
	}
}

// AgentPayloadFromDef converts an agent definition into the shared payload.
func AgentPayloadFromDef(agent aghconfig.AgentDef) contract.AgentPayload {
	mcpServers := make([]contract.AgentMCPServerJSON, 0, len(agent.MCPServers))
	for _, server := range agent.MCPServers {
		redacted := aghconfig.RedactedMCPServer(server)

		mcpServers = append(mcpServers, contract.AgentMCPServerJSON{
			Name:      redacted.Name,
			Transport: string(redacted.Transport),
			Command:   redacted.Command,
			Args:      append([]string(nil), redacted.Args...),
			Env:       redacted.Env,
			SecretEnv: redacted.SecretEnv,
			URL:       redacted.URL,
			Auth:      settingsMCPAuthConfigPayload(redacted.Auth),
		})
	}

	return contract.AgentPayload{
		Name:        agent.Name,
		Provider:    agent.Provider,
		Command:     agent.Command,
		Model:       agent.Model,
		Tools:       append([]string(nil), agent.Tools...),
		Toolsets:    append([]string(nil), agent.Toolsets...),
		DenyTools:   append([]string(nil), agent.DenyTools...),
		Permissions: agent.Permissions,
		MCPServers:  mcpServers,
		Prompt:      agent.Prompt,
	}
}

// AgentPayloadFromDiagnostic converts a malformed workspace agent diagnostic into a payload row.
func AgentPayloadFromDiagnostic(diagnostic workspacepkg.AgentDiagnostic) contract.AgentPayload {
	return contract.AgentPayload{
		Name:     diagnostic.Name,
		Provider: "",
		Prompt:   "",
		Diagnostics: []contract.AgentDiagnosticPayload{{
			Path:      diagnostic.Path,
			ErrorKind: diagnostic.ErrorKind,
			Message:   diagnostic.Message,
		}},
	}
}

// AgentPayloadsFromDefs converts a list of agent definitions into response payloads.
func AgentPayloadsFromDefs(agents []aghconfig.AgentDef) []contract.AgentPayload {
	payload := make([]contract.AgentPayload, 0, len(agents))
	for _, agent := range agents {
		payload = append(payload, AgentPayloadFromDef(agent))
	}
	return payload
}

// AgentEventPayloadFromEvent converts an agent event into the shared raw-stream payload.
func AgentEventPayloadFromEvent(event acp.AgentEvent) contract.AgentEventPayload {
	return contract.AgentEventPayload{
		Type:       event.Type,
		SessionID:  event.SessionID,
		TurnID:     event.TurnID,
		RequestID:  event.RequestID,
		Timestamp:  event.Timestamp,
		Text:       event.Text,
		Title:      event.Title,
		ToolCallID: event.ToolCallID,
		StopReason: event.StopReason,
		Action:     event.Action,
		Resource:   event.Resource,
		Decision:   event.Decision,
		Error:      event.Error,
		Failure:    SessionFailurePayloadFromStore(event.Failure),
		Usage:      TokenUsagePayloadFromUsage(event.Usage),
		Runtime:    runtimeActivityPayloadFromEvent(event.Runtime),
		Raw:        payloadJSONBytes(event.Raw),
	}
}

// TokenUsagePayloadFromUsage converts token usage info into the shared payload.
func TokenUsagePayloadFromUsage(usage *acp.TokenUsage) *contract.TokenUsagePayload {
	if usage == nil {
		return nil
	}

	return &contract.TokenUsagePayload{
		TurnID:           usage.TurnID,
		InputTokens:      usage.InputTokens,
		OutputTokens:     usage.OutputTokens,
		TotalTokens:      usage.TotalTokens,
		ThoughtTokens:    usage.ThoughtTokens,
		CacheReadTokens:  usage.CacheReadTokens,
		CacheWriteTokens: usage.CacheWriteTokens,
		ContextUsed:      usage.ContextUsed,
		ContextSize:      usage.ContextSize,
		CostAmount:       usage.CostAmount,
		CostCurrency:     usage.CostCurrency,
		Timestamp:        usage.Timestamp,
	}
}

// ObserveEventPayloadFromEvent converts an observe event into the shared payload.
func ObserveEventPayloadFromEvent(event store.EventSummary) contract.ObserveEventPayload {
	return contract.ObserveEventPayload{
		ID:              event.ID,
		SessionID:       event.SessionID,
		Type:            event.Type,
		AgentName:       event.AgentName,
		ParentSessionID: event.ParentSessionID,
		RootSessionID:   event.RootSessionID,
		SpawnDepth:      event.SpawnDepth,
		Summary:         event.Summary,
		Timestamp:       event.Timestamp,
	}
}

// ObserveHealthPayloadFromHealth converts the observer health snapshot into the shared payload.
func ObserveHealthPayloadFromHealth(health *observepkg.Health) contract.ObserveHealthPayload {
	if health == nil {
		return contract.ObserveHealthPayload{}
	}
	return contract.ObserveHealthPayload{
		Status:             health.Status,
		UptimeSeconds:      health.UptimeSeconds,
		ActiveSessions:     health.ActiveSessions,
		ActiveAgents:       health.ActiveAgents,
		GlobalDBSizeBytes:  health.GlobalDBSizeBytes,
		SessionDBSizeBytes: health.SessionDBSizeBytes,
		Persistence:        ObservePersistenceHealthPayloadFromHealth(health.Persistence),
		Retention:          ObserveRetentionHealthPayloadFromHealth(health.Retention),
		Failures:           ObserveFailureHealthPayloadFromHealth(health.Failures),
		AgentProbes:        AgentProbeHealthPayloadsFromACP(health.AgentProbes),
		Bridges:            BridgeAggregateHealthPayloadFromObserve(health.Bridges),
		Activities:         SessionActivityHealthPayloadsFromObserve(health.Activities),
		Version:            health.Version,
	}
}

// ObservePersistenceHealthPayloadFromHealth converts persistence health into the shared payload.
func ObservePersistenceHealthPayloadFromHealth(
	health observepkg.PersistenceHealth,
) contract.ObservePersistenceHealthPayload {
	return contract.ObservePersistenceHealthPayload{
		Status:             strings.TrimSpace(health.Status),
		GlobalDBSizeBytes:  health.GlobalDBSizeBytes,
		SessionDBSizeBytes: health.SessionDBSizeBytes,
	}
}

// ObserveRetentionHealthPayloadFromHealth converts retention health into the shared payload.
func ObserveRetentionHealthPayloadFromHealth(
	health observepkg.RetentionHealth,
) contract.ObserveRetentionHealthPayload {
	return contract.ObserveRetentionHealthPayload{
		Enabled:                  health.Enabled,
		RetentionDays:            health.RetentionDays,
		SweepIntervalSeconds:     health.SweepIntervalSeconds,
		LastSweepStatus:          strings.TrimSpace(health.LastSweepStatus),
		LastSweepAt:              cloneTimePtr(health.LastSweepAt),
		LastCutoffAt:             cloneTimePtr(health.LastCutoffAt),
		LastSweepError:           strings.TrimSpace(health.LastSweepError),
		DeletedEventSummaries:    health.DeletedEventSummaries,
		DeletedTokenStats:        health.DeletedTokenStats,
		DeletedPermissionLogRows: health.DeletedPermissionLogRows,
	}
}

// ObserveFailureHealthPayloadFromHealth converts lifecycle failure health into
// the shared payload.
func ObserveFailureHealthPayloadFromHealth(
	health observepkg.FailureHealth,
) contract.ObserveFailureHealthPayload {
	payload := contract.ObserveFailureHealthPayload{
		Status: strings.TrimSpace(health.Status),
		Total:  health.Total,
	}
	if len(health.ByKind) > 0 {
		payload.ByKind = make(map[store.FailureKind]int, len(health.ByKind))
		maps.Copy(payload.ByKind, health.ByKind)
	}
	if len(health.Recent) > 0 {
		payload.Recent = make([]contract.SessionFailureHealthPayload, 0, len(health.Recent))
		for _, failure := range health.Recent {
			payload.Recent = append(payload.Recent, contract.SessionFailureHealthPayload{
				SessionID:       strings.TrimSpace(failure.SessionID),
				AgentName:       strings.TrimSpace(failure.AgentName),
				Provider:        strings.TrimSpace(failure.Provider),
				WorkspaceID:     strings.TrimSpace(failure.WorkspaceID),
				State:           strings.TrimSpace(failure.State),
				FailureKind:     failure.FailureKind,
				Summary:         diagnostics.RedactAndBound(failure.Summary, maxDiagnosticPayloadBytes),
				CrashBundlePath: diagnostics.RedactAndBound(failure.CrashBundlePath, maxDiagnosticPayloadBytes),
				UpdatedAt:       failure.UpdatedAt,
			})
		}
	}
	return payload
}

// AgentProbeHealthPayloadsFromACP converts downstream ACP probe results into
// the shared health payload.
func AgentProbeHealthPayloadsFromACP(probes []acp.ProbeResult) []contract.AgentProbeHealthPayload {
	if len(probes) == 0 {
		return nil
	}
	payloads := make([]contract.AgentProbeHealthPayload, 0, len(probes))
	for _, probe := range probes {
		payloads = append(payloads, contract.AgentProbeHealthPayload{
			AgentName:  strings.TrimSpace(probe.AgentName),
			Provider:   strings.TrimSpace(probe.Provider),
			Command:    diagnostics.RedactAndBound(probe.Command, maxDiagnosticPayloadBytes),
			Executable: strings.TrimSpace(probe.Executable),
			Status:     strings.TrimSpace(probe.Status),
			Error:      diagnostics.RedactAndBound(probe.Error, maxDiagnosticPayloadBytes),
			CheckedAt:  probe.CheckedAt,
			DurationMS: probe.DurationMS,
		})
	}
	return payloads
}

// SessionActivityHealthPayloadsFromObserve converts observer activity health
// rows into the shared health response payload.
func SessionActivityHealthPayloadsFromObserve(
	activities []observepkg.SessionActivityHealth,
) []contract.SessionActivityHealthPayload {
	if len(activities) == 0 {
		return nil
	}
	payloads := make([]contract.SessionActivityHealthPayload, 0, len(activities))
	for _, activity := range activities {
		payloads = append(payloads, contract.SessionActivityHealthPayload{
			SessionID:          strings.TrimSpace(activity.SessionID),
			TurnID:             strings.TrimSpace(activity.TurnID),
			TurnSource:         strings.TrimSpace(activity.TurnSource),
			TurnStartedAt:      cloneTimePtr(activity.TurnStartedAt),
			LastActivityAt:     cloneTimePtr(activity.LastActivityAt),
			LastActivityKind:   strings.TrimSpace(activity.LastActivityKind),
			LastActivityDetail: strings.TrimSpace(activity.LastActivityDetail),
			CurrentTool:        strings.TrimSpace(activity.CurrentTool),
			ToolCallID:         strings.TrimSpace(activity.ToolCallID),
			LastProgressAt:     cloneTimePtr(activity.LastProgressAt),
			IterationCurrent:   activity.IterationCurrent,
			IterationMax:       activity.IterationMax,
			IdleSeconds:        activity.IdleSeconds,
			ElapsedSeconds:     activity.ElapsedSeconds,
			Status:             strings.TrimSpace(activity.Status),
			StallState:         strings.TrimSpace(activity.StallState),
			StallReason:        strings.TrimSpace(activity.StallReason),
		})
	}
	return payloads
}

// AutomationHealthPayloadFromStatus converts manager status into the shared
// additive automation health block.
func AutomationHealthPayloadFromStatus(
	enabled bool,
	status automationpkg.ManagerStatus,
) contract.AutomationHealthPayload {
	return contract.AutomationHealthPayload{
		Enabled: enabled,
		Jobs: contract.AutomationResourceStatusPayload{
			Total:   status.Jobs.Total,
			Enabled: status.Jobs.Enabled,
		},
		Triggers: contract.AutomationResourceStatusPayload{
			Total:   status.Triggers.Total,
			Enabled: status.Triggers.Enabled,
		},
		SchedulerRunning: status.SchedulerRunning,
		NextFire:         status.NextFire,
		ScheduledJobs:    AutomationSchedulerStatePayloadsFromStates(status.ScheduledJobs),
	}
}

// AutomationSchedulerStatePayloadFromState converts durable scheduler metadata
// into the shared response payload.
func AutomationSchedulerStatePayloadFromState(
	state automationpkg.ScheduledJobState,
) contract.AutomationSchedulerStatePayload {
	payload := contract.AutomationSchedulerStatePayload{
		JobID:               state.JobID,
		Registered:          state.Registered,
		NextRunAt:           state.NextRun,
		LastRunAt:           state.LastRun,
		LastScheduledAt:     state.LastScheduledAt,
		LastFireID:          state.LastFireID,
		CatchUpPolicy:       state.CatchUpPolicy,
		MisfireGraceSeconds: state.MisfireGraceSeconds,
		LastMisfireAt:       state.LastMisfireAt,
		MisfireCount:        state.MisfireCount,
	}
	if state.Durable != nil {
		payload.ConsecutiveResumeFailures = state.Durable.ConsecutiveResumeFailures
		updatedAt := state.Durable.UpdatedAt
		if !updatedAt.IsZero() {
			payload.UpdatedAt = &updatedAt
		}
	}
	return payload
}

// AutomationSchedulerStatePayloadsFromStates converts scheduler states into response payloads.
func AutomationSchedulerStatePayloadsFromStates(
	states []automationpkg.ScheduledJobState,
) []contract.AutomationSchedulerStatePayload {
	payloads := make([]contract.AutomationSchedulerStatePayload, 0, len(states))
	for _, state := range states {
		payloads = append(payloads, AutomationSchedulerStatePayloadFromState(state))
	}
	return payloads
}

// JobPayloadFromJob converts an automation job into the shared response
// payload, optionally enriching it with scheduler next-run metadata.
func JobPayloadFromJob(
	job automationpkg.Job,
	nextRun *time.Time,
	schedulerState *contract.AutomationSchedulerStatePayload,
) contract.JobPayload {
	payload := contract.JobPayload{
		ID:          job.ID,
		Scope:       job.Scope,
		Name:        job.Name,
		AgentName:   job.AgentName,
		WorkspaceID: job.WorkspaceID,
		Prompt:      job.Prompt,
		Enabled:     job.Enabled,
		Retry:       job.Retry,
		FireLimit:   job.FireLimit,
		Source:      job.Source,
		CreatedAt:   job.CreatedAt,
		UpdatedAt:   job.UpdatedAt,
		NextRun:     nextRun,
		Scheduler:   schedulerState,
	}
	if job.Schedule != nil {
		schedule := *job.Schedule
		payload.Schedule = &schedule
	}
	if job.Task != nil {
		taskConfig := *job.Task
		if job.Task.Owner != nil {
			owner := *job.Task.Owner
			taskConfig.Owner = &owner
		}
		payload.Task = &taskConfig
	}
	return payload
}

// JobPayloadsFromJobs converts a slice of jobs into response payloads using
// the supplied next-run map.
func JobPayloadsFromJobs(
	jobs []automationpkg.Job,
	schedulerStateByID map[string]contract.AutomationSchedulerStatePayload,
) []contract.JobPayload {
	payloads := make([]contract.JobPayload, 0, len(jobs))
	for _, job := range jobs {
		var schedulerState *contract.AutomationSchedulerStatePayload
		if state, ok := schedulerStateByID[job.ID]; ok {
			stateCopy := state
			schedulerState = &stateCopy
		}
		payloads = append(payloads, JobPayloadFromJob(job, schedulerNextRun(schedulerState), schedulerState))
	}
	return payloads
}

func schedulerNextRun(state *contract.AutomationSchedulerStatePayload) *time.Time {
	if state == nil || state.NextRunAt == nil {
		return nil
	}
	nextRun := state.NextRunAt.UTC()
	return &nextRun
}

// TriggerPayloadFromTrigger converts an automation trigger into the shared
// response payload.
func TriggerPayloadFromTrigger(trigger automationpkg.Trigger) contract.TriggerPayload {
	return contract.TriggerPayloadFromTrigger(trigger)
}

// TriggerPayloadsFromTriggers converts a slice of triggers into response payloads.
func TriggerPayloadsFromTriggers(triggers []automationpkg.Trigger) []contract.TriggerPayload {
	return contract.TriggerPayloadsFromTriggers(triggers)
}

// RunPayloadFromRun converts an automation run into the shared response payload.
func RunPayloadFromRun(run automationpkg.Run) contract.RunPayload {
	return contract.RunPayload{
		ID:              run.ID,
		JobID:           run.JobID,
		TriggerID:       run.TriggerID,
		SessionID:       run.SessionID,
		TaskID:          run.TaskID,
		TaskRunID:       run.TaskRunID,
		FireID:          run.FireID,
		Status:          run.Status,
		Attempt:         run.Attempt,
		ScheduledAt:     run.ScheduledAt,
		StartedAt:       run.StartedAt,
		EndedAt:         run.EndedAt,
		Error:           run.Error,
		DeliveryError:   run.DeliveryError,
		DeliveryErrorAt: run.DeliveryErrorAt,
	}
}

// RunPayloadsFromRuns converts a slice of runs into response payloads.
func RunPayloadsFromRuns(runs []automationpkg.Run) []contract.RunPayload {
	payloads := make([]contract.RunPayload, 0, len(runs))
	for _, run := range runs {
		payloads = append(payloads, RunPayloadFromRun(run))
	}
	return payloads
}

// WebhookDeliveryPayloadFromResult converts a webhook trigger dispatch result
// into the shared response payload.
func WebhookDeliveryPayloadFromResult(result automationpkg.TriggerResult) contract.WebhookDeliveryPayload {
	return contract.WebhookDeliveryPayload{
		Matched: result.Matched,
		Runs:    RunPayloadsFromRuns(result.Runs),
	}
}

// BridgeAggregateHealthPayloadFromObserve converts the observer bridge
// summary into the shared payload.
func BridgeAggregateHealthPayloadFromObserve(
	summary observepkg.BridgeAggregateHealth,
) contract.BridgeAggregateHealthPayload {
	return contract.BridgeAggregateHealthPayload{
		TotalInstances:        summary.TotalInstances,
		RouteCount:            summary.RouteCount,
		DeliveryBacklog:       summary.DeliveryBacklog,
		DeliveryDroppedTotal:  summary.DeliveryDroppedTotal,
		DeliveryFailuresTotal: summary.DeliveryFailuresTotal,
		AuthFailuresTotal:     summary.AuthFailuresTotal,
		StatusCounts: contract.BridgeStatusCountsPayload{
			Disabled:     summary.StatusCounts.Disabled,
			Starting:     summary.StatusCounts.Starting,
			Ready:        summary.StatusCounts.Ready,
			Degraded:     summary.StatusCounts.Degraded,
			AuthRequired: summary.StatusCounts.AuthRequired,
			Error:        summary.StatusCounts.Error,
		},
	}
}

// BridgeHealthPayloadFromObserve converts the observer per-instance bridge
// health snapshot into the shared payload.
func BridgeHealthPayloadFromObserve(health observepkg.BridgeInstanceHealth) contract.BridgeHealthPayload {
	var lastSuccessAt *time.Time
	if !health.LastSuccessAt.IsZero() {
		timestamp := health.LastSuccessAt
		lastSuccessAt = &timestamp
	}

	var lastErrorAt *time.Time
	if !health.LastErrorAt.IsZero() {
		timestamp := health.LastErrorAt
		lastErrorAt = &timestamp
	}

	return contract.BridgeHealthPayload{
		BridgeInstanceID:        health.BridgeInstanceID,
		Status:                  health.Status,
		RouteCount:              health.RouteCount,
		DeliveryBacklog:         health.DeliveryBacklog,
		DeliveryDroppedTotal:    health.DeliveryDroppedTotal,
		DeliveryDroppedByReason: maps.Clone(health.DeliveryDroppedByReason),
		DeliveryFailuresTotal:   health.DeliveryFailuresTotal,
		AuthFailuresTotal:       health.AuthFailuresTotal,
		LastSuccessAt:           lastSuccessAt,
		LastError:               health.LastError,
		LastErrorAt:             lastErrorAt,
	}
}

// BridgePayloadFromBridgeInstance converts the daemon-owned bridge record into
// the shared bridge-management payload exposed by transports and OpenAPI.
func BridgePayloadFromBridgeInstance(instance bridgepkg.BridgeInstance) contract.BridgePayload {
	return contract.BridgePayload{
		ID:               instance.ID,
		Scope:            instance.Scope,
		WorkspaceID:      instance.WorkspaceID,
		Platform:         instance.Platform,
		ExtensionName:    instance.ExtensionName,
		DisplayName:      instance.DisplayName,
		Source:           instance.Source,
		Enabled:          instance.Enabled,
		Status:           instance.Status,
		DMPolicy:         instance.DMPolicy,
		RoutingPolicy:    instance.RoutingPolicy,
		ProviderConfig:   contract.BridgeProviderConfigPayload(cloneRawMessage(instance.ProviderConfig)),
		DeliveryDefaults: contract.BridgeDeliveryDefaultsPayload(cloneRawMessage(instance.DeliveryDefaults)),
		Degradation:      cloneBridgeDegradation(instance.Degradation),
		CreatedAt:        instance.CreatedAt,
		UpdatedAt:        instance.UpdatedAt,
	}
}

// BridgeProviderPayloadFromBridgeProvider converts installed provider metadata
// into the shared bridge-management provider catalog payload.
func BridgeProviderPayloadFromBridgeProvider(provider bridgepkg.BridgeProvider) contract.BridgeProviderPayload {
	var configSchema *bridgepkg.BridgeProviderConfigSchema
	if provider.ConfigSchema != nil {
		cloned := *provider.ConfigSchema
		configSchema = &cloned
	}

	secretSlots := make([]bridgepkg.BridgeSecretSlot, 0, len(provider.SecretSlots))
	secretSlots = append(secretSlots, provider.SecretSlots...)

	return contract.BridgeProviderPayload{
		Platform:      provider.Platform,
		ExtensionName: provider.ExtensionName,
		DisplayName:   provider.DisplayName,
		Description:   provider.Description,
		SecretSlots:   secretSlots,
		ConfigSchema:  configSchema,
		Enabled:       provider.Enabled,
		State:         provider.State,
		Health:        provider.Health,
		HealthMessage: provider.HealthMessage,
	}
}

// WorkspacePayloadFromWorkspace converts a workspace into the shared payload.
func WorkspacePayloadFromWorkspace(workspace workspacepkg.Workspace) contract.WorkspacePayload {
	addDirs := make([]string, 0, len(workspace.AdditionalDirs))
	addDirs = append(addDirs, workspace.AdditionalDirs...)

	return contract.WorkspacePayload{
		ID:           workspace.ID,
		RootDir:      workspace.RootDir,
		AddDirs:      addDirs,
		Name:         workspace.Name,
		DefaultAgent: workspace.DefaultAgent,
		SandboxRef:   workspace.SandboxRef,
		CreatedAt:    workspace.CreatedAt,
		UpdatedAt:    workspace.UpdatedAt,
	}
}

// WorkspaceSkillPayloads converts workspace skill paths into response payloads.
func WorkspaceSkillPayloads(skills []workspacepkg.SkillPath) []contract.WorkspaceSkillPayload {
	payload := make([]contract.WorkspaceSkillPayload, 0, len(skills))
	for _, skill := range skills {
		payload = append(payload, contract.WorkspaceSkillPayload{
			Name:   filepath.Base(skill.Dir),
			Dir:    skill.Dir,
			Source: skill.Source,
		})
	}
	return payload
}

// SessionProviderOptionPayloadsFromConfig converts the merged workspace config
// into a stable, UI-ready list of visible provider options.
func SessionProviderOptionPayloadsFromConfig(cfg *aghconfig.Config) []contract.SessionProviderOptionPayload {
	payloadsByName := make(map[string]contract.SessionProviderOptionPayload, len(aghconfig.BuiltinProviders()))
	for name := range aghconfig.BuiltinProviders() {
		payload, ok := sessionProviderOptionPayloadFromConfig(cfg, name)
		if !ok {
			continue
		}
		payloadsByName[payload.Name] = payload
	}
	if cfg != nil {
		for name := range cfg.Providers {
			payload, ok := sessionProviderOptionPayloadFromConfig(cfg, name)
			if !ok {
				continue
			}
			payloadsByName[payload.Name] = payload
		}
	}

	return sortSessionProviderOptionPayloads(payloadsByName)
}

func sessionProviderOptionPayloadFromConfig(
	cfg *aghconfig.Config,
	name string,
) (contract.SessionProviderOptionPayload, bool) {
	providerName := aghconfig.CanonicalProviderName(name)
	if providerName == "" {
		return contract.SessionProviderOptionPayload{}, false
	}
	var resolved aghconfig.ProviderConfig
	var err error
	if cfg == nil {
		var empty aghconfig.Config
		resolved, err = empty.ResolveProvider(providerName)
	} else {
		resolved, err = cfg.ResolveProvider(providerName)
	}
	if err != nil {
		return contract.SessionProviderOptionPayload{}, false
	}
	return contract.SessionProviderOptionPayload{
		Name:            providerName,
		DisplayName:     strings.TrimSpace(resolved.DisplayName),
		Harness:         string(resolved.EffectiveHarness()),
		RuntimeProvider: strings.TrimSpace(resolved.RuntimeProviderName(providerName)),
		DefaultModel:    strings.TrimSpace(resolved.DefaultModel),
		AuthMode:        string(resolved.EffectiveAuthMode()),
		EnvPolicy:       string(resolved.EffectiveEnvPolicy()),
		HomePolicy:      string(resolved.EffectiveHomePolicy()),
	}, true
}

func sessionProviderOptionPayloads(names []string) []contract.SessionProviderOptionPayload {
	values := make(map[string]contract.SessionProviderOptionPayload, len(names))
	for _, name := range names {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue
		}
		values[trimmed] = contract.SessionProviderOptionPayload{Name: trimmed}
	}
	return sortSessionProviderOptionPayloads(values)
}

func sortSessionProviderOptionPayloads(
	values map[string]contract.SessionProviderOptionPayload,
) []contract.SessionProviderOptionPayload {
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	payloads := make([]contract.SessionProviderOptionPayload, 0, len(names))
	for _, name := range names {
		payloads = append(payloads, values[name])
	}
	return payloads
}

// PayloadJSON coerces raw strings into valid JSON response bodies.
func PayloadJSON(raw string) json.RawMessage {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return json.RawMessage("null")
	}
	if json.Valid([]byte(trimmed)) {
		return json.RawMessage(trimmed)
	}

	encoded, err := json.Marshal(trimmed)
	if err != nil {
		return json.RawMessage("null")
	}
	return json.RawMessage(encoded)
}

func payloadJSONBytes(raw []byte) json.RawMessage {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return json.RawMessage("null")
	}
	if json.Valid(trimmed) {
		return append(json.RawMessage(nil), trimmed...)
	}

	encoded, err := json.Marshal(string(trimmed))
	if err != nil {
		return json.RawMessage("null")
	}
	return json.RawMessage(encoded)
}

func cloneBridgeDegradation(value *bridgepkg.BridgeDegradation) *bridgepkg.BridgeDegradation {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

// SkillPayloadFromSkill converts a skills.Skill into the shared HTTP payload.
func SkillPayloadFromSkill(skill *skills.Skill) contract.SkillPayload {
	if skill == nil {
		return contract.SkillPayload{}
	}

	payload := contract.SkillPayload{
		Name:        skill.Meta.Name,
		Description: skill.Meta.Description,
		Version:     skill.Meta.Version,
		Source:      skills.SkillSourceName(skill.Source),
		Enabled:     skill.Enabled,
		Dir:         skill.Dir,
		Metadata:    skill.Meta.Metadata,
	}
	if skill.Provenance != nil {
		payload.Provenance = &contract.ProvenancePayload{
			Slug:        skill.Provenance.Slug,
			Registry:    skill.Provenance.Registry,
			Version:     skill.Provenance.Version,
			InstalledAt: skill.Provenance.InstalledAt,
		}
	}

	return payload
}

// SkillPayloadsFromSkills converts a slice of skills into response payloads.
func SkillPayloadsFromSkills(skillList []*skills.Skill) []contract.SkillPayload {
	payload := make([]contract.SkillPayload, 0, len(skillList))
	for _, skill := range skillList {
		if skill == nil {
			continue
		}
		payload = append(payload, SkillPayloadFromSkill(skill))
	}
	return payload
}

func sessionWorkspaceFromInfo(info *session.Info) (string, string) {
	if info == nil {
		return "", ""
	}
	return strings.TrimSpace(info.WorkspaceID), strings.TrimSpace(info.Workspace)
}

func timePointerFromMap(values map[string]*time.Time, id string) *time.Time {
	if len(values) == 0 {
		return nil
	}
	value, ok := values[id]
	if !ok || value == nil {
		return nil
	}
	next := value.UTC()
	return &next
}

// SettingsSectionResponseFromEnvelope converts one settings section envelope into the shared response payload.
func SettingsSectionResponseFromEnvelope(envelope settingspkg.SectionEnvelope) (any, error) {
	switch envelope.Section {
	case settingspkg.SectionGeneral:
		return settingsGeneralSectionResponse(envelope)
	case settingspkg.SectionMemory:
		return settingsMemorySectionResponse(envelope)
	case settingspkg.SectionSkills:
		return settingsSkillsSectionResponse(envelope)
	case settingspkg.SectionAutomation:
		return settingsAutomationSectionResponse(envelope)
	case settingspkg.SectionNetwork:
		return settingsNetworkSectionResponse(envelope)
	case settingspkg.SectionObservability:
		return settingsObservabilitySectionResponse(envelope)
	case settingspkg.SectionHooksExtensions:
		return settingsHooksExtensionsSectionResponse(envelope)
	default:
		return nil, fmt.Errorf("unknown settings section %q", envelope.Section)
	}
}

func settingsGeneralSectionResponse(envelope settingspkg.SectionEnvelope) (any, error) {
	if envelope.General == nil {
		return nil, errors.New("settings general section is required")
	}
	return contract.SettingsGeneralResponse{
		SettingsSectionResponseMetaPayload: settingsSectionMetaPayload(envelope),
		ConfigPaths:                        settingsConfigPathsPayload(envelope.General.ConfigPaths),
		Config:                             settingsGeneralConfigPayload(envelope.General.Settings),
		Runtime:                            settingsDaemonRuntimePayload(envelope.General.Runtime),
		Actions: contract.SettingsGeneralActionsPayload{
			Restart: settingsActionMetadataPayload(envelope.General.Actions.Restart),
		},
	}, nil
}

func settingsMemorySectionResponse(envelope settingspkg.SectionEnvelope) (any, error) {
	if envelope.Memory == nil {
		return nil, errors.New("settings memory section is required")
	}
	return contract.SettingsMemoryResponse{
		SettingsSectionResponseMetaPayload: settingsSectionMetaPayload(envelope),
		Config:                             settingsMemoryConfigPayload(envelope.Memory.Config),
		Health:                             settingsMemoryHealthPayload(envelope.Memory.Health),
		Actions: contract.SettingsMemoryActionsPayload{
			Consolidate: settingsActionMetadataPayload(envelope.Memory.Actions.Consolidate),
		},
	}, nil
}

func settingsSkillsSectionResponse(envelope settingspkg.SectionEnvelope) (any, error) {
	if envelope.Skills == nil {
		return nil, errors.New("settings skills section is required")
	}
	return contract.SettingsSkillsResponse{
		SettingsSectionResponseMetaPayload: settingsSectionMetaPayload(envelope),
		Config:                             settingsSkillsConfigPayload(envelope.Skills.Config),
		DiscoveredCount:                    envelope.Skills.DiscoveredCount,
		DisabledCount:                      envelope.Skills.DisabledCount,
		RuntimeAvailable:                   envelope.Skills.RuntimeAvailable,
		Links:                              settingsOperationalLinkPayloads(envelope.Skills.Links),
	}, nil
}

func settingsAutomationSectionResponse(envelope settingspkg.SectionEnvelope) (any, error) {
	if envelope.Automation == nil {
		return nil, errors.New("settings automation section is required")
	}
	return contract.SettingsAutomationResponse{
		SettingsSectionResponseMetaPayload: settingsSectionMetaPayload(envelope),
		Config:                             settingsAutomationConfigPayload(envelope.Automation.Config),
		Runtime:                            settingsAutomationRuntimePayload(envelope.Automation.Runtime),
		Links:                              settingsOperationalLinkPayloads(envelope.Automation.Links),
	}, nil
}

func settingsNetworkSectionResponse(envelope settingspkg.SectionEnvelope) (any, error) {
	if envelope.Network == nil {
		return nil, errors.New("settings network section is required")
	}
	return contract.SettingsNetworkResponse{
		SettingsSectionResponseMetaPayload: settingsSectionMetaPayload(envelope),
		Config:                             settingsNetworkConfigPayload(envelope.Network.Config),
		Runtime:                            settingsNetworkRuntimePayload(envelope.Network.Runtime),
		Links:                              settingsOperationalLinkPayloads(envelope.Network.Links),
	}, nil
}

func settingsObservabilitySectionResponse(envelope settingspkg.SectionEnvelope) (any, error) {
	if envelope.Observability == nil {
		return nil, errors.New("settings observability section is required")
	}
	return contract.SettingsObservabilityResponse{
		SettingsSectionResponseMetaPayload: settingsSectionMetaPayload(envelope),
		Config:                             settingsObservabilityConfigPayload(envelope.Observability.Config),
		Runtime:                            settingsObservabilityRuntimePayload(envelope.Observability.Runtime),
		LogTail:                            settingsLogTailCapabilityPayload(envelope.Observability.LogTailSupport),
	}, nil
}

func settingsHooksExtensionsSectionResponse(envelope settingspkg.SectionEnvelope) (any, error) {
	if envelope.HooksExtensions == nil {
		return nil, errors.New("settings hooks-extensions section is required")
	}
	return contract.SettingsHooksExtensionsResponse{
		SettingsSectionResponseMetaPayload: settingsSectionMetaPayload(envelope),
		Hooks:                              settingsHookItemPayloads(envelope.HooksExtensions.Hooks),
		Config:                             settingsExtensionsConfigPayload(envelope.HooksExtensions.Extensions),
		Installed:                          settingsInstalledExtensionPayloads(envelope.HooksExtensions.Installed),
		TransportParity: settingsTransportParityPayload(
			envelope.HooksExtensions.TransportParity,
		),
	}, nil
}

// SettingsCollectionResponseFromEnvelope converts one settings collection envelope into the shared response payload.
func SettingsCollectionResponseFromEnvelope(envelope settingspkg.CollectionEnvelope) (any, error) {
	switch envelope.Collection {
	case settingspkg.CollectionProviders:
		return contract.SettingsProvidersResponse{
			SettingsCollectionResponseMetaPayload: settingsCollectionMetaPayload(envelope),
			Providers:                             settingsProviderItemPayloads(envelope.Providers),
		}, nil
	case settingspkg.CollectionMCPServers:
		return contract.SettingsMCPServersResponse{
			SettingsCollectionResponseMetaPayload: settingsCollectionMetaPayload(envelope),
			MCPServers:                            settingsMCPServerItemPayloads(envelope.MCPServers),
		}, nil
	case settingspkg.CollectionSandboxes:
		return contract.SettingsSandboxesResponse{
			SettingsCollectionResponseMetaPayload: settingsCollectionMetaPayload(envelope),
			Sandboxes:                             settingsSandboxItemPayloads(envelope.Sandboxes),
		}, nil
	case settingspkg.CollectionHooks:
		return contract.SettingsHooksResponse{
			SettingsCollectionResponseMetaPayload: settingsCollectionMetaPayload(envelope),
			Hooks:                                 settingsHookItemPayloads(envelope.Hooks),
		}, nil
	default:
		return nil, fmt.Errorf("unknown settings collection %q", envelope.Collection)
	}
}

// SettingsMutationResultPayloadFromResult converts one settings mutation result into the shared payload.
func SettingsMutationResultPayloadFromResult(result settingspkg.MutationResult) contract.MutationResult {
	return contract.MutationResult{
		Section:         contract.SettingsSectionName(result.Section),
		Scope:           contract.SettingsScopeKind(result.Scope),
		WriteTarget:     contract.SettingsWriteTargetKind(result.WriteTarget),
		WorkspaceID:     strings.TrimSpace(result.WorkspaceID),
		Behavior:        contract.SettingsMutationBehavior(result.Behavior),
		Applied:         result.Applied,
		RestartRequired: result.RestartRequired,
		RestartScope:    strings.TrimSpace(result.RestartScope),
		Warnings:        cloneStrings(result.Warnings),
	}
}

// SettingsRestartActionResponseFromOperation converts one daemon restart operation into the action response payload.
func SettingsRestartActionResponseFromOperation(operation SettingsRestartOperation) contract.RestartActionResponse {
	return contract.RestartActionResponse{
		OperationID:        strings.TrimSpace(operation.OperationID),
		Status:             contract.RestartOperationStatus(operation.Status),
		StatusURL:          settingsRestartStatusURL(operation.OperationID),
		ActiveSessionCount: operation.ActiveSessionCount,
	}
}

// SettingsRestartActionStatusFromOperation converts one daemon restart operation into the polling payload.
func SettingsRestartActionStatusFromOperation(operation SettingsRestartOperation) contract.RestartActionStatus {
	return contract.RestartActionStatus{
		OperationID:        strings.TrimSpace(operation.OperationID),
		Status:             contract.RestartOperationStatus(operation.Status),
		OldPID:             operation.OldPID,
		OldStartedAt:       operation.OldStartedAt,
		OldSocketPath:      strings.TrimSpace(operation.OldSocketPath),
		NewPID:             operation.NewPID,
		ActiveSessionCount: operation.ActiveSessionCount,
		FailureReason:      strings.TrimSpace(operation.FailureReason),
		StartedAt:          operation.StartedAt,
		UpdatedAt:          operation.UpdatedAt,
		CompletedAt:        cloneTimePointer(operation.CompletedAt),
	}
}

// SettingsUpdateResponseFromStatus converts the daemon-owned update snapshot into the transport payload.
func SettingsUpdateResponseFromStatus(status SettingsUpdateStatus) contract.SettingsUpdateResponse {
	return contract.SettingsUpdateResponse{
		Supported:      status.Supported,
		Managed:        status.Managed,
		InstallMethod:  strings.TrimSpace(status.InstallMethod),
		CurrentVersion: strings.TrimSpace(status.CurrentVersion),
		LatestVersion:  strings.TrimSpace(status.LatestVersion),
		Available:      status.Available,
		Status:         contract.SettingsUpdateStatusKind(strings.TrimSpace(status.Status)),
		Recommendation: strings.TrimSpace(status.Recommendation),
		ReleaseURL:     strings.TrimSpace(status.ReleaseURL),
		CheckedAt:      cloneTimePointer(status.CheckedAt),
		LastError:      strings.TrimSpace(status.LastError),
	}
}

func settingsSectionMetaPayload(envelope settingspkg.SectionEnvelope) contract.SettingsSectionResponseMetaPayload {
	return contract.SettingsSectionResponseMetaPayload{
		Section:         contract.SettingsSectionName(envelope.Section),
		Scope:           contract.SettingsScopeKind(envelope.Scope),
		WorkspaceID:     strings.TrimSpace(envelope.WorkspaceID),
		AvailableScopes: settingsScopeKindsPayload(envelope.AvailableScopes),
	}
}

func settingsCollectionMetaPayload(
	envelope settingspkg.CollectionEnvelope,
) contract.SettingsCollectionResponseMetaPayload {
	return contract.SettingsCollectionResponseMetaPayload{
		Collection:      contract.SettingsCollectionName(envelope.Collection),
		Scope:           contract.SettingsScopeKind(envelope.Scope),
		WorkspaceID:     strings.TrimSpace(envelope.WorkspaceID),
		AvailableScopes: settingsScopeKindsPayload(envelope.AvailableScopes),
	}
}

func settingsScopeKindsPayload(scopes []settingspkg.ScopeKind) []contract.SettingsScopeKind {
	if len(scopes) == 0 {
		return nil
	}
	payloads := make([]contract.SettingsScopeKind, 0, len(scopes))
	for _, scope := range scopes {
		payloads = append(payloads, contract.SettingsScopeKind(scope))
	}
	return payloads
}

func settingsConfigPathsPayload(paths settingspkg.ConfigPaths) contract.SettingsConfigPathsPayload {
	return contract.SettingsConfigPathsPayload{
		HomeDir:          strings.TrimSpace(paths.HomeDir),
		GlobalConfig:     strings.TrimSpace(paths.GlobalConfig),
		GlobalMCPSidecar: strings.TrimSpace(paths.GlobalMCPSidecar),
		LogFile:          strings.TrimSpace(paths.LogFile),
		DaemonInfo:       strings.TrimSpace(paths.DaemonInfo),
	}
}

func settingsGeneralConfigPayload(value settingspkg.GeneralSettings) contract.SettingsGeneralConfigPayload {
	return contract.SettingsGeneralConfigPayload{
		Defaults: contract.SettingsDefaultsPayload{
			Agent:    strings.TrimSpace(value.Defaults.Agent),
			Provider: strings.TrimSpace(value.Defaults.Provider),
			Sandbox:  strings.TrimSpace(value.Defaults.Sandbox),
		},
		Limits: contract.SettingsLimitsPayload{
			MaxSessions:         value.Limits.MaxSessions,
			MaxConcurrentAgents: value.Limits.MaxConcurrentAgents,
		},
		Permissions: contract.SettingsPermissionsPayload{
			Mode: contract.SettingsPermissionMode(value.Permissions.Mode),
		},
		SessionTimeout: value.SessionTimeout.String(),
		HTTP: contract.SettingsHTTPPayload{
			Host: strings.TrimSpace(value.HTTP.Host),
			Port: value.HTTP.Port,
		},
		Daemon: contract.SettingsDaemonPayload{
			Socket: strings.TrimSpace(value.Daemon.Socket),
		},
	}
}

func settingsMemoryConfigPayload(value aghconfig.MemoryConfig) contract.SettingsMemoryConfigPayload {
	return contract.SettingsMemoryConfigPayload{
		Enabled:   value.Enabled,
		GlobalDir: strings.TrimSpace(value.GlobalDir),
		Dream: contract.SettingsMemoryDreamPayload{
			Enabled:       value.Dream.Enabled,
			Agent:         strings.TrimSpace(value.Dream.Agent),
			MinHours:      value.Dream.MinHours,
			MinSessions:   value.Dream.MinSessions,
			CheckInterval: value.Dream.CheckInterval.String(),
		},
	}
}

func settingsSkillsConfigPayload(value aghconfig.SkillsConfig) contract.SettingsSkillsConfigPayload {
	return contract.SettingsSkillsConfigPayload{
		Enabled:                 value.Enabled,
		DisabledSkills:          cloneStrings(value.DisabledSkills),
		PollInterval:            value.PollInterval.String(),
		AllowedMarketplaceMCP:   cloneStrings(value.AllowedMarketplaceMCP),
		AllowedMarketplaceHooks: cloneStrings(value.AllowedMarketplaceHooks),
		Marketplace: contract.SettingsMarketplacePayload{
			Registry: strings.TrimSpace(value.Marketplace.Registry),
			BaseURL:  strings.TrimSpace(value.Marketplace.BaseURL),
		},
	}
}

func settingsAutomationConfigPayload(value settingspkg.AutomationSettings) contract.SettingsAutomationConfigPayload {
	return contract.SettingsAutomationConfigPayload{
		Enabled:           value.Enabled,
		Timezone:          strings.TrimSpace(value.Timezone),
		MaxConcurrentJobs: value.MaxConcurrentJobs,
		DefaultFireLimit:  value.DefaultFireLimit,
	}
}

func settingsNetworkConfigPayload(value aghconfig.NetworkConfig) contract.SettingsNetworkConfigPayload {
	return contract.SettingsNetworkConfigPayload{
		Enabled:        value.Enabled,
		DefaultChannel: strings.TrimSpace(value.DefaultChannel),
		Port:           value.Port,
		MaxPayload:     value.MaxPayload,
		GreetInterval:  value.GreetInterval,
		MaxReplayAge:   value.MaxReplayAge,
		MaxQueueDepth:  value.MaxQueueDepth,
	}
}

func settingsObservabilityConfigPayload(
	value aghconfig.ObservabilityConfig,
) contract.SettingsObservabilityConfigPayload {
	return contract.SettingsObservabilityConfigPayload{
		Enabled:        value.Enabled,
		RetentionDays:  value.RetentionDays,
		MaxGlobalBytes: value.MaxGlobalBytes,
		Transcripts: contract.SettingsObservabilityTranscriptPayload{
			Enabled:            value.Transcripts.Enabled,
			SegmentBytes:       value.Transcripts.SegmentBytes,
			MaxBytesPerSession: value.Transcripts.MaxBytesPerSession,
		},
	}
}

func settingsExtensionsConfigPayload(value aghconfig.ExtensionsConfig) contract.SettingsExtensionsConfigPayload {
	return contract.SettingsExtensionsConfigPayload{
		Marketplace: contract.SettingsMarketplacePayload{
			Registry: strings.TrimSpace(value.Marketplace.Registry),
			BaseURL:  strings.TrimSpace(value.Marketplace.BaseURL),
		},
		Resources: contract.SettingsExtensionResourcesPayload{
			AllowedKinds:           resourceKindsToStrings(value.Resources.AllowedKinds),
			MaxScope:               value.Resources.MaxScope,
			SnapshotRateLimit:      settingsExtensionRateLimitPayload(value.Resources.SnapshotRateLimit),
			OperatorWriteRateLimit: settingsExtensionRateLimitPayload(value.Resources.OperatorWriteRateLimit),
		},
	}
}

func settingsExtensionRateLimitPayload(
	value aghconfig.ExtensionsResourceRateLimitConfig,
) contract.SettingsExtensionRateLimitPayload {
	return contract.SettingsExtensionRateLimitPayload{
		Requests: value.Requests,
		Window:   value.Window.String(),
		Queue:    value.Queue,
	}
}

func settingsDaemonRuntimePayload(value settingspkg.DaemonRuntimeStatus) contract.SettingsDaemonRuntimePayload {
	payload := contract.SettingsDaemonRuntimePayload{
		Available:      value.Available,
		Status:         strings.TrimSpace(value.Status),
		PID:            value.PID,
		UptimeSeconds:  value.UptimeSeconds,
		Socket:         strings.TrimSpace(value.Socket),
		HTTPHost:       strings.TrimSpace(value.HTTPHost),
		HTTPPort:       value.HTTPPort,
		ActiveSessions: value.ActiveSessions,
		ActiveAgents:   value.ActiveAgents,
		TotalSessions:  value.TotalSessions,
		Version:        strings.TrimSpace(value.Version),
	}
	if startedAt := optionalTime(value.StartedAt); startedAt != nil {
		payload.StartedAt = startedAt
	}
	return payload
}

func settingsMemoryHealthPayload(value settingspkg.MemoryHealthStatus) contract.SettingsMemoryHealthPayload {
	return contract.SettingsMemoryHealthPayload{
		Available:          value.Available,
		FileCount:          value.FileCount,
		DreamEnabled:       value.DreamEnabled,
		LastConsolidatedAt: cloneTimePointer(value.LastConsolidatedAt),
	}
}

func settingsAutomationRuntimePayload(
	value settingspkg.AutomationRuntimeStatus,
) contract.SettingsAutomationRuntimePayload {
	return contract.SettingsAutomationRuntimePayload{
		Available:        value.Available,
		Running:          value.Running,
		SchedulerRunning: value.SchedulerRunning,
		JobTotal:         value.JobTotal,
		JobEnabled:       value.JobEnabled,
		TriggerTotal:     value.TriggerTotal,
		TriggerEnabled:   value.TriggerEnabled,
		NextFire:         cloneTimePointer(value.NextFire),
		LastSyncedAt:     cloneTimePointer(value.LastSyncedAt),
	}
}

func settingsNetworkRuntimePayload(value settingspkg.NetworkRuntimeStatus) contract.SettingsNetworkRuntimePayload {
	return contract.SettingsNetworkRuntimePayload{
		Available:       value.Available,
		Enabled:         value.Enabled,
		Status:          strings.TrimSpace(value.Status),
		ListenerHost:    strings.TrimSpace(value.ListenerHost),
		ListenerPort:    value.ListenerPort,
		LocalPeers:      value.LocalPeers,
		RemotePeers:     value.RemotePeers,
		Channels:        value.Channels,
		QueuedMessages:  value.QueuedMessages,
		QueuedSessions:  value.QueuedSessions,
		DeliveryWorkers: value.DeliveryWorkers,
	}
}

func settingsObservabilityRuntimePayload(
	value settingspkg.ObservabilityRuntimeStatus,
) contract.SettingsObservabilityRuntimePayload {
	return contract.SettingsObservabilityRuntimePayload{
		Available:          value.Available,
		Status:             strings.TrimSpace(value.Status),
		GlobalDBSizeBytes:  value.GlobalDBSizeBytes,
		SessionDBSizeBytes: value.SessionDBSizeBytes,
		ActiveSessions:     value.ActiveSessions,
		ActiveAgents:       value.ActiveAgents,
		UptimeSeconds:      value.UptimeSeconds,
	}
}

func settingsLogTailCapabilityPayload(value settingspkg.CapabilityStatus) contract.SettingsLogTailCapabilityPayload {
	payload := contract.SettingsLogTailCapabilityPayload{Available: value.Available}
	if value.Available {
		payload.StreamURL = settingsObservabilityLogTailPath
		payload.Transport = contract.SettingsStreamTransportSSE
	}
	return payload
}

func settingsActionMetadataPayload(value settingspkg.ActionMetadata) contract.SettingsActionMetadataPayload {
	return contract.SettingsActionMetadataPayload{
		Name:      strings.TrimSpace(value.Name),
		Available: value.Available,
		Behavior:  contract.SettingsMutationBehavior(value.Behavior),
	}
}

func settingsOperationalLinkPayloads(values []settingspkg.OperationalLink) []contract.SettingsOperationalLinkPayload {
	if len(values) == 0 {
		return nil
	}
	payloads := make([]contract.SettingsOperationalLinkPayload, 0, len(values))
	for _, value := range values {
		payloads = append(payloads, contract.SettingsOperationalLinkPayload{
			Label: strings.TrimSpace(value.Label),
			Path:  strings.TrimSpace(value.Path),
		})
	}
	return payloads
}

func settingsTransportParityPayload(value settingspkg.TransportParityStatus) contract.SettingsTransportParityPayload {
	return contract.SettingsTransportParityPayload{
		Known:          value.Known,
		SettingsHTTP:   value.SettingsHTTP,
		SettingsUDS:    value.SettingsUDS,
		ExtensionsHTTP: value.ExtensionsHTTP,
		ExtensionsUDS:  value.ExtensionsUDS,
	}
}

func settingsInstalledExtensionPayloads(
	values []settingspkg.InstalledExtension,
) []contract.SettingsInstalledExtensionPayload {
	if len(values) == 0 {
		return nil
	}
	payloads := make([]contract.SettingsInstalledExtensionPayload, 0, len(values))
	for _, value := range values {
		payloads = append(payloads, contract.SettingsInstalledExtensionPayload{
			Name:          strings.TrimSpace(value.Name),
			Version:       strings.TrimSpace(value.Version),
			Enabled:       value.Enabled,
			State:         strings.TrimSpace(value.State),
			Health:        strings.TrimSpace(value.Health),
			HealthMessage: strings.TrimSpace(value.HealthMessage),
			LastError:     strings.TrimSpace(value.LastError),
			RequiresEnv:   append([]string(nil), value.RequiresEnv...),
			MissingEnv:    append([]string(nil), value.MissingEnv...),
		})
	}
	return payloads
}

func settingsProviderItemPayloads(values []settingspkg.ProviderItem) []contract.SettingsProviderItemPayload {
	if len(values) == 0 {
		return nil
	}
	payloads := make([]contract.SettingsProviderItemPayload, 0, len(values))
	for _, value := range values {
		payloads = append(payloads, settingsProviderItemPayload(value))
	}
	return payloads
}

func settingsProviderItemPayload(value settingspkg.ProviderItem) contract.SettingsProviderItemPayload {
	payload := contract.SettingsProviderItemPayload{
		Name:             strings.TrimSpace(value.Name),
		Settings:         settingsProviderSettingsPayload(value.Settings),
		Default:          value.Default,
		CommandAvailable: value.CommandAvailable,
		Credentials:      settingsProviderCredentialStatusPayloads(value.Credentials),
		AuthStatus:       settingsProviderAuthStatusPayload(value.AuthStatus),
		SourceMetadata:   settingsSourceMetadataPayload(value.SourceMetadata),
	}
	if value.Fallback != nil {
		payload.Fallback = &contract.SettingsProviderFallbackPayload{
			Source:   settingsSourceRefPayload(value.Fallback.Source),
			Settings: settingsProviderSettingsPayload(value.Fallback.Settings),
		}
	}
	return payload
}

func settingsProviderSettingsPayload(value settingspkg.ProviderSettings) contract.SettingsProviderSettingsPayload {
	return contract.SettingsProviderSettingsPayload{
		Command:         strings.TrimSpace(value.Command),
		DisplayName:     strings.TrimSpace(value.DisplayName),
		DefaultModel:    strings.TrimSpace(value.DefaultModel),
		Harness:         string(value.Harness),
		RuntimeProvider: strings.TrimSpace(value.RuntimeProvider),
		Transport:       strings.TrimSpace(value.Transport),
		BaseURL:         strings.TrimSpace(value.BaseURL),
		AuthMode:        string(value.AuthMode),
		EnvPolicy:       string(value.EnvPolicy),
		HomePolicy:      string(value.HomePolicy),
		AuthStatusCmd:   strings.TrimSpace(value.AuthStatusCmd),
		AuthLoginCmd:    strings.TrimSpace(value.AuthLoginCmd),
		CredentialSlots: settingsProviderCredentialSlotPayloads(value.CredentialSlots),
	}
}

func settingsProviderCredentialSlotPayloads(
	values []aghconfig.ProviderCredentialSlot,
) []contract.SettingsProviderCredentialSlotPayload {
	if len(values) == 0 {
		return nil
	}
	payloads := make([]contract.SettingsProviderCredentialSlotPayload, 0, len(values))
	for _, value := range values {
		payloads = append(payloads, contract.SettingsProviderCredentialSlotPayload{
			Name:      strings.TrimSpace(value.Name),
			TargetEnv: strings.TrimSpace(value.TargetEnv),
			SecretRef: strings.TrimSpace(value.SecretRef),
			Kind:      strings.TrimSpace(value.Kind),
			Required:  value.Required,
		})
	}
	return payloads
}

func settingsProviderAuthStatusPayload(
	value settingspkg.ProviderAuthStatus,
) *contract.SettingsProviderAuthStatusPayload {
	payload := contract.SettingsProviderAuthStatusPayload{
		Mode:       string(value.Mode),
		EnvPolicy:  string(value.EnvPolicy),
		HomePolicy: string(value.HomePolicy),
		State:      strings.TrimSpace(value.State),
		Message:    strings.TrimSpace(value.Message),
		StatusCmd:  strings.TrimSpace(value.StatusCmd),
		LoginCmd:   strings.TrimSpace(value.LoginCmd),
	}
	return &payload
}

func settingsProviderCredentialStatusPayloads(
	values []settingspkg.ProviderCredentialStatus,
) []contract.SettingsProviderCredentialStatusPayload {
	if len(values) == 0 {
		return nil
	}
	payloads := make([]contract.SettingsProviderCredentialStatusPayload, 0, len(values))
	for _, value := range values {
		payloads = append(payloads, contract.SettingsProviderCredentialStatusPayload{
			Name:      strings.TrimSpace(value.Name),
			TargetEnv: strings.TrimSpace(value.TargetEnv),
			SecretRef: strings.TrimSpace(value.SecretRef),
			Kind:      strings.TrimSpace(value.Kind),
			Required:  value.Required,
			Present:   value.Present,
			Source:    strings.TrimSpace(value.Source),
		})
	}
	return payloads
}

func settingsMCPServerItemPayloads(values []settingspkg.MCPServerItem) []contract.SettingsMCPServerItemPayload {
	if len(values) == 0 {
		return nil
	}
	payloads := make([]contract.SettingsMCPServerItemPayload, 0, len(values))
	for _, value := range values {
		payloads = append(payloads, contract.SettingsMCPServerItemPayload{
			Name:           strings.TrimSpace(value.Name),
			Transport:      strings.TrimSpace(string(value.Transport)),
			Command:        strings.TrimSpace(value.Command),
			Args:           cloneStrings(value.Args),
			Env:            cloneStringMap(value.Env),
			SecretEnv:      cloneStringMap(value.SecretEnv),
			URL:            strings.TrimSpace(value.URL),
			Auth:           settingsMCPAuthConfigPayload(value.Auth),
			AuthStatus:     settingsMCPAuthStatusPayload(value.AuthStatus),
			Scope:          contract.SettingsScopeKind(value.Scope),
			WorkspaceID:    strings.TrimSpace(value.WorkspaceID),
			SourceMetadata: settingsSourceMetadataPayload(value.SourceMetadata),
		})
	}
	return payloads
}

func settingsMCPAuthConfigPayload(value aghconfig.MCPAuthConfig) *contract.SettingsMCPAuthConfigPayload {
	if value.IsZero() {
		return nil
	}
	return &contract.SettingsMCPAuthConfigPayload{
		Type:             strings.TrimSpace(string(value.Type)),
		IssuerURL:        strings.TrimSpace(value.IssuerURL),
		MetadataURL:      strings.TrimSpace(value.MetadataURL),
		AuthorizationURL: strings.TrimSpace(value.AuthorizationURL),
		TokenURL:         strings.TrimSpace(value.TokenURL),
		RevocationURL:    strings.TrimSpace(value.RevocationURL),
		ClientID:         strings.TrimSpace(value.ClientID),
		ClientSecretRef:  strings.TrimSpace(value.ClientSecretRef),
		Scopes:           cloneStrings(value.Scopes),
	}
}

func settingsMCPAuthStatusPayload(value *settingspkg.MCPAuthStatus) *contract.SettingsMCPAuthStatusPayload {
	if value == nil {
		return nil
	}
	return &contract.SettingsMCPAuthStatusPayload{
		ServerName:       strings.TrimSpace(value.ServerName),
		Status:           strings.TrimSpace(string(value.Status)),
		RemoteURL:        strings.TrimSpace(value.RemoteURL),
		AuthType:         strings.TrimSpace(value.AuthType),
		ClientID:         strings.TrimSpace(value.ClientID),
		Issuer:           strings.TrimSpace(value.Issuer),
		Scopes:           cloneStrings(value.Scopes),
		ExpiresAt:        cloneTimePtr(value.ExpiresAt),
		UpdatedAt:        cloneTimePtr(value.UpdatedAt),
		Refreshable:      value.Refreshable,
		TokenPresent:     value.TokenPresent,
		RevocationURL:    strings.TrimSpace(value.RevocationURL),
		Diagnostic:       strings.TrimSpace(value.Diagnostic),
		AuthorizationURL: strings.TrimSpace(value.AuthorizationURL),
	}
}

func settingsSandboxItemPayloads(values []settingspkg.SandboxItem) []contract.SettingsSandboxItemPayload {
	if len(values) == 0 {
		return nil
	}
	payloads := make([]contract.SettingsSandboxItemPayload, 0, len(values))
	for _, value := range values {
		payloads = append(payloads, contract.SettingsSandboxItemPayload{
			Name:                strings.TrimSpace(value.Name),
			Profile:             settingsSandboxProfilePayload(value.Profile),
			WorkspaceUsageCount: value.WorkspaceUsageCount,
			SourceMetadata:      settingsSourceMetadataPayload(value.SourceMetadata),
		})
	}
	return payloads
}

func settingsSandboxProfilePayload(value aghconfig.SandboxProfile) contract.SettingsSandboxProfilePayload {
	payload := contract.SettingsSandboxProfilePayload{
		Backend:     strings.TrimSpace(value.Backend),
		SyncMode:    strings.TrimSpace(value.SyncMode),
		Persistence: strings.TrimSpace(value.Persistence),
		RuntimeRoot: strings.TrimSpace(value.RuntimeRoot),
		Env:         cloneStringMap(value.Env),
		SecretEnv:   cloneStringMap(value.SecretEnv),
	}
	if network := settingsSandboxNetworkPayload(value.Network); network != nil {
		payload.Network = network
	}
	if daytona := settingsSandboxDaytonaPayload(value.Daytona); daytona != nil {
		payload.Daytona = daytona
	}
	return payload
}

func settingsSandboxNetworkPayload(
	value aghconfig.NetworkProfile,
) *contract.SettingsSandboxNetworkPayload {
	if !value.AllowPublicIngress &&
		!value.AllowOutbound &&
		!value.Required &&
		len(value.AllowList) == 0 &&
		len(value.DenyList) == 0 {
		return nil
	}
	return &contract.SettingsSandboxNetworkPayload{
		AllowPublicIngress: value.AllowPublicIngress,
		AllowOutbound:      value.AllowOutbound,
		AllowList:          cloneStrings(value.AllowList),
		DenyList:           cloneStrings(value.DenyList),
		Required:           value.Required,
	}
}

func settingsSandboxDaytonaPayload(
	value aghconfig.DaytonaProfile,
) *contract.SettingsSandboxDaytonaPayload {
	if strings.TrimSpace(value.APIURL) == "" &&
		strings.TrimSpace(value.Target) == "" &&
		strings.TrimSpace(value.Image) == "" &&
		strings.TrimSpace(value.Snapshot) == "" &&
		strings.TrimSpace(value.Class) == "" &&
		strings.TrimSpace(value.AutoStop) == "" &&
		strings.TrimSpace(value.AutoArchive) == "" {
		return nil
	}
	return &contract.SettingsSandboxDaytonaPayload{
		APIURL:      strings.TrimSpace(value.APIURL),
		Target:      strings.TrimSpace(value.Target),
		Image:       strings.TrimSpace(value.Image),
		Snapshot:    strings.TrimSpace(value.Snapshot),
		Class:       strings.TrimSpace(value.Class),
		AutoStop:    strings.TrimSpace(value.AutoStop),
		AutoArchive: strings.TrimSpace(value.AutoArchive),
	}
}

func settingsHookItemPayloads(values []settingspkg.HookItem) []contract.SettingsHookItemPayload {
	if len(values) == 0 {
		return nil
	}
	payloads := make([]contract.SettingsHookItemPayload, 0, len(values))
	for i := range values {
		value := &values[i]
		payloads = append(payloads, contract.SettingsHookItemPayload{
			Name:           strings.TrimSpace(value.Name),
			Declaration:    settingsHookDeclarationPayload(value.Declaration),
			SourceMetadata: settingsSourceMetadataPayload(value.SourceMetadata),
		})
	}
	return payloads
}

func settingsHookDeclarationPayload(value hookspkg.HookDecl) contract.SettingsHookDeclarationPayload {
	return contract.SettingsHookDeclarationPayload{
		Name:         strings.TrimSpace(value.Name),
		Event:        value.Event,
		Mode:         value.Mode,
		Required:     value.Required,
		Priority:     int(value.Priority),
		Timeout:      durationString(value.Timeout),
		Matcher:      value.Matcher,
		ExecutorKind: value.ExecutorKind,
		Command:      strings.TrimSpace(value.Command),
		Args:         cloneStrings(value.Args),
		Env:          cloneStringMap(value.Env),
		SecretEnv:    cloneStringMap(value.SecretEnv),
		Metadata:     cloneStringMap(value.Metadata),
	}
}

func settingsSourceMetadataPayload(value settingspkg.SourceMetadata) contract.SettingsSourceMetadataPayload {
	return contract.SettingsSourceMetadataPayload{
		EffectiveSource:  settingsSourceRefPayload(value.EffectiveSource),
		ShadowedSources:  settingsSourceRefPayloads(value.ShadowedSources),
		AvailableTargets: settingsWriteTargetKindsPayload(value.AvailableTargets),
	}
}

func settingsSourceRefPayload(value settingspkg.SourceRef) contract.SettingsSourceRefPayload {
	return contract.SettingsSourceRefPayload{
		Kind:        contract.SettingsSourceKind(value.Kind),
		Scope:       contract.SettingsScopeKind(value.Scope),
		WorkspaceID: strings.TrimSpace(value.WorkspaceID),
	}
}

func settingsSourceRefPayloads(values []settingspkg.SourceRef) []contract.SettingsSourceRefPayload {
	if len(values) == 0 {
		return nil
	}
	payloads := make([]contract.SettingsSourceRefPayload, 0, len(values))
	for _, value := range values {
		payloads = append(payloads, settingsSourceRefPayload(value))
	}
	return payloads
}

func settingsWriteTargetKindsPayload(values []settingspkg.WriteTargetKind) []contract.SettingsWriteTargetKind {
	if len(values) == 0 {
		return nil
	}
	payloads := make([]contract.SettingsWriteTargetKind, 0, len(values))
	for _, value := range values {
		payloads = append(payloads, contract.SettingsWriteTargetKind(value))
	}
	return payloads
}

func cloneStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]string, len(src))
	maps.Copy(dst, src)
	return dst
}

func cloneStrings(src []string) []string {
	if len(src) == 0 {
		return nil
	}
	return append([]string(nil), src...)
}

func resourceKindsToStrings(values []resources.ResourceKind) []string {
	if len(values) == 0 {
		return nil
	}
	payloads := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(string(value)); trimmed != "" {
			payloads = append(payloads, trimmed)
		}
	}
	return payloads
}

func cloneTimePointer(src *time.Time) *time.Time {
	if src == nil {
		return nil
	}
	cloned := src.UTC()
	return &cloned
}

func durationString(value time.Duration) string {
	if value <= 0 {
		return ""
	}
	return value.String()
}

// TaskReferencePayloadFromReference converts one task reference into the shared payload.
func TaskReferencePayloadFromReference(record taskpkg.Reference) contract.TaskReferencePayload {
	return contract.TaskReferencePayload{
		ID:          record.ID,
		Identifier:  record.Identifier,
		Title:       record.Title,
		Status:      record.Status,
		Priority:    record.Priority,
		Owner:       cloneOwnership(record.Owner),
		Scope:       record.Scope,
		WorkspaceID: record.WorkspaceID,
	}
}

// TaskRunSummaryPayloadFromSummary converts one operator-facing run summary into the shared payload.
func TaskRunSummaryPayloadFromSummary(summary *taskpkg.RunSummary) *contract.TaskRunSummaryPayload {
	if summary == nil {
		return nil
	}

	return &contract.TaskRunSummaryPayload{
		ID:                    summary.ID,
		TaskID:                summary.TaskID,
		Status:                summary.Status,
		Attempt:               summary.Attempt,
		MaxAttempts:           summary.MaxAttempts,
		SessionID:             summary.SessionID,
		ClaimedBy:             cloneActorIdentity(summary.ClaimedBy),
		ClaimTokenHash:        summary.ClaimTokenHash,
		LeaseUntil:            optionalTime(summary.LeaseUntil),
		HeartbeatAt:           optionalTime(summary.HeartbeatAt),
		CoordinationChannelID: summary.CoordinationChannelID,
		QueuedAt:              summary.QueuedAt,
		ClaimedAt:             optionalTime(summary.ClaimedAt),
		StartedAt:             optionalTime(summary.StartedAt),
		EndedAt:               optionalTime(summary.EndedAt),
		Error:                 summary.Error,
	}
}

// TaskDependencyReferencePayloadsFromReferences converts enriched dependency references into shared payloads.
func TaskDependencyReferencePayloadsFromReferences(
	dependencies []taskpkg.DependencyReference,
) []contract.TaskDependencyReferencePayload {
	payloads := make([]contract.TaskDependencyReferencePayload, 0, len(dependencies))
	for _, dependency := range dependencies {
		payloads = append(payloads, contract.TaskDependencyReferencePayload{
			TaskID:          dependency.TaskID,
			DependsOnTaskID: dependency.DependsOnTaskID,
			Kind:            dependency.Kind,
			CreatedAt:       dependency.CreatedAt,
			DependsOn:       TaskReferencePayloadFromReference(dependency.DependsOn),
		})
	}
	return payloads
}

// TaskTimelineItemPayloadsFromItems converts task timeline items into shared payloads.
func TaskTimelineItemPayloadsFromItems(items []taskpkg.TimelineItem) []contract.TaskTimelineItemPayload {
	payloads := make([]contract.TaskTimelineItemPayload, 0, len(items))
	for _, item := range items {
		payloads = append(payloads, TaskTimelineItemPayloadFromItem(item))
	}
	return payloads
}

// TaskTimelineItemPayloadFromItem converts one task timeline item into the shared payload.
func TaskTimelineItemPayloadFromItem(item taskpkg.TimelineItem) contract.TaskTimelineItemPayload {
	return contract.TaskTimelineItemPayload{
		Sequence:  item.Sequence,
		EventID:   item.EventID,
		Task:      TaskReferencePayloadFromReference(item.Task),
		Run:       TaskRunSummaryPayloadFromSummary(item.Run),
		EventType: item.EventType,
		Actor:     item.Actor,
		Origin:    item.Origin,
		Payload:   cloneRawMessage(item.Payload),
		Timestamp: item.Timestamp,
	}
}

// TaskStreamEventPayloadFromEvent converts one task live-stream event into the shared payload.
func TaskStreamEventPayloadFromEvent(event taskpkg.StreamEvent) contract.TaskStreamEventPayload {
	return contract.TaskStreamEventPayload{
		Sequence: event.Sequence,
		Type:     event.Type,
		Timeline: TaskTimelineItemPayloadFromItem(event.Timeline),
	}
}

// TaskTreePayloadFromView converts one task tree snapshot into the shared payload.
func TaskTreePayloadFromView(view *taskpkg.TreeView) contract.TaskTreePayload {
	if view == nil {
		return contract.TaskTreePayload{}
	}

	payload := contract.TaskTreePayload{
		Root: TaskTreeNodePayloadFromNode(view.Root),
	}
	if len(view.Descendants) > 0 {
		payload.Descendants = make([]contract.TaskTreeNodePayload, 0, len(view.Descendants))
		for _, node := range view.Descendants {
			payload.Descendants = append(payload.Descendants, TaskTreeNodePayloadFromNode(node))
		}
	}
	return payload
}

// TaskTreeNodePayloadFromNode converts one task tree node into the shared payload.
func TaskTreeNodePayloadFromNode(node taskpkg.TreeNode) contract.TaskTreeNodePayload {
	return contract.TaskTreeNodePayload{
		Task:           TaskReferencePayloadFromReference(node.Task),
		ParentTaskID:   node.ParentTaskID,
		Depth:          node.Depth,
		ChildCount:     node.ChildCount,
		ActiveRun:      TaskRunSummaryPayloadFromSummary(node.ActiveRun),
		LastActivityAt: node.LastActivityAt,
	}
}

// TaskRunSessionPayloadFromSession converts one task run session link into the shared payload.
func TaskRunSessionPayloadFromSession(session *taskpkg.RunSessionRef) *contract.TaskRunSessionPayload {
	if session == nil {
		return nil
	}

	return &contract.TaskRunSessionPayload{
		SessionID:   session.SessionID,
		WorkspaceID: session.WorkspaceID,
		AgentName:   session.AgentName,
		Name:        session.Name,
		Channel:     session.Channel,
		State:       session.State,
		CreatedAt:   session.CreatedAt,
		UpdatedAt:   session.UpdatedAt,
	}
}

// TaskRunOperationalSummaryPayloadFromSummary converts run-detail operational metrics into the shared payload.
func TaskRunOperationalSummaryPayloadFromSummary(
	summary taskpkg.RunOperationalSummary,
) contract.TaskRunOperationalSummaryPayload {
	return contract.TaskRunOperationalSummaryPayload{
		LastActivityAt: summary.LastActivityAt,
		LastEventType:  summary.LastEventType,
		ToolCallCount:  summary.ToolCallCount,
		TurnCount:      summary.TurnCount,
		InputTokens:    summary.InputTokens,
		OutputTokens:   summary.OutputTokens,
		TotalTokens:    summary.TotalTokens,
		TotalCost:      summary.TotalCost,
		CostCurrency:   summary.CostCurrency,
	}
}

// TaskRunDetailPayloadFromView converts one run-detail view into the shared payload.
func TaskRunDetailPayloadFromView(view *taskpkg.RunDetailView) contract.TaskRunDetailPayload {
	if view == nil {
		return contract.TaskRunDetailPayload{}
	}

	return contract.TaskRunDetailPayload{
		Run:     TaskRunPayloadFromRun(&view.Run),
		Task:    TaskReferencePayloadFromReference(view.Task),
		Session: TaskRunSessionPayloadFromSession(view.Session),
		Summary: TaskRunOperationalSummaryPayloadFromSummary(view.Summary),
	}
}

// TaskTriageStatePayloadFromState converts one triage-state record into the shared payload.
func TaskTriageStatePayloadFromState(state taskpkg.TriageState) contract.TaskTriageStatePayload {
	return contract.TaskTriageStatePayload{
		TaskID:             state.TaskID,
		Actor:              state.Actor,
		Read:               state.Read,
		Archived:           state.Archived,
		Dismissed:          state.Dismissed,
		LastSeenActivityAt: optionalTime(state.LastSeenActivityAt),
		UpdatedAt:          state.UpdatedAt,
	}
}

// TaskDashboardPayloadFromView converts one observer-backed dashboard view into the shared payload.
func TaskDashboardPayloadFromView(view *observepkg.TaskDashboardView) contract.TaskDashboardPayload {
	if view == nil {
		return contract.TaskDashboardPayload{}
	}

	return contract.TaskDashboardPayload{
		Totals:          taskDashboardTotalsPayload(view.Totals),
		Cards:           taskDashboardCardsPayload(view.Cards),
		Queue:           taskDashboardQueuePayload(view.Queue),
		Health:          taskDashboardHealthPayload(view.Health),
		StatusBreakdown: taskDashboardStatusBreakdownPayloads(view.StatusBreakdown),
		ActiveRuns:      taskDashboardActiveRunsPayload(view.ActiveRuns),
		Freshness:       taskDashboardFreshnessPayload(view.Freshness),
	}
}

func taskDashboardTotalsPayload(totals observepkg.TaskDashboardTotals) contract.TaskDashboardTotalsPayload {
	return contract.TaskDashboardTotalsPayload{
		TasksTotal:             totals.TasksTotal,
		RunsTotal:              totals.RunsTotal,
		DraftTasks:             totals.DraftTasks,
		PendingTasks:           totals.PendingTasks,
		ReadyTasks:             totals.ReadyTasks,
		InProgressTasks:        totals.InProgressTasks,
		BlockedTasks:           totals.BlockedTasks,
		CompletedTasks:         totals.CompletedTasks,
		FailedTasks:            totals.FailedTasks,
		CanceledTasks:          totals.CanceledTasks,
		AwaitingApprovalTasks:  totals.AwaitingApprovalTasks,
		DependencyBlockedTasks: totals.DependencyBlockedTasks,
		QueuedRuns:             totals.QueuedRuns,
		ClaimedRuns:            totals.ClaimedRuns,
		StartingRuns:           totals.StartingRuns,
		RunningRuns:            totals.RunningRuns,
		CompletedRuns:          totals.CompletedRuns,
		FailedRuns:             totals.FailedRuns,
		CanceledRuns:           totals.CanceledRuns,
		ActiveRuns:             totals.ActiveRuns,
	}
}

func taskDashboardCardsPayload(cards observepkg.TaskDashboardCards) contract.TaskDashboardCardsPayload {
	return contract.TaskDashboardCardsPayload{
		InProgress: contract.TaskDashboardInProgressCardPayload{
			Tasks:        cards.InProgress.Tasks,
			ActiveRuns:   cards.InProgress.ActiveRuns,
			RunningRuns:  cards.InProgress.RunningRuns,
			StartingRuns: cards.InProgress.StartingRuns,
			ClaimedRuns:  cards.InProgress.ClaimedRuns,
			QueuedRuns:   cards.InProgress.QueuedRuns,
			HealthStatus: cards.InProgress.HealthStatus,
		},
		Blocked: contract.TaskDashboardBlockedCardPayload{
			Tasks:                cards.Blocked.Tasks,
			AwaitingApproval:     cards.Blocked.AwaitingApproval,
			AwaitingDependencies: cards.Blocked.AwaitingDependencies,
			HealthStatus:         cards.Blocked.HealthStatus,
		},
		Failed: contract.TaskDashboardFailedCardPayload{
			Tasks:        cards.Failed.Tasks,
			FailedRuns:   cards.Failed.FailedRuns,
			ForcedStops:  cards.Failed.ForcedStops,
			HealthStatus: cards.Failed.HealthStatus,
		},
		Latency: contract.TaskDashboardLatencyCardPayload{
			ClaimLatencyMillis: taskLatencyMetricPayload(cards.Latency.ClaimLatencyMillis),
			StartLatencyMillis: taskLatencyMetricPayload(cards.Latency.StartLatencyMillis),
		},
	}
}

func taskLatencyMetricPayload(metric observepkg.LatencyMetric) contract.TaskLatencyMetricPayload {
	return contract.TaskLatencyMetricPayload{
		Samples:       metric.Samples,
		AverageMillis: metric.AverageMillis,
		MaximumMillis: metric.MaximumMillis,
	}
}

func taskDashboardQueuePayload(queue observepkg.TaskDashboardQueue) contract.TaskDashboardQueuePayload {
	payload := contract.TaskDashboardQueuePayload{
		Total:                 queue.Total,
		OldestQueuedAt:        queue.OldestQueuedAt,
		OldestQueueAgeMilli:   queue.OldestQueueAgeMilli,
		BacklogWarning:        queue.BacklogWarning,
		BacklogStatus:         queue.BacklogStatus,
		BacklogThresholdMilli: queue.BacklogThresholdMilli,
	}
	if len(queue.Depth) == 0 {
		return payload
	}

	payload.Depth = make([]contract.TaskDashboardQueueDepthPayload, 0, len(queue.Depth))
	for _, item := range queue.Depth {
		payload.Depth = append(payload.Depth, contract.TaskDashboardQueueDepthPayload{
			NetworkChannel:      item.NetworkChannel,
			Count:               item.Count,
			OldestQueuedAt:      item.OldestQueuedAt,
			OldestQueueAgeMilli: item.OldestQueueAgeMilli,
		})
	}
	return payload
}

func taskDashboardHealthPayload(health observepkg.TaskDashboardHealth) contract.TaskDashboardHealthPayload {
	return contract.TaskDashboardHealthPayload{
		Status:           health.Status,
		StuckRuns:        health.StuckRuns,
		ActiveOrphanRuns: health.ActiveOrphanRuns,
		QueueBacklog:     health.QueueBacklog,
	}
}

func taskDashboardStatusBreakdownPayloads(
	items []observepkg.TaskDashboardStatusBreakdown,
) []contract.TaskDashboardStatusBreakdownPayload {
	if len(items) == 0 {
		return nil
	}

	payloads := make([]contract.TaskDashboardStatusBreakdownPayload, 0, len(items))
	for _, item := range items {
		payloads = append(payloads, contract.TaskDashboardStatusBreakdownPayload{
			Status:       item.Status,
			Count:        item.Count,
			SharePercent: item.SharePercent,
		})
	}
	return payloads
}

func taskDashboardActiveRunsPayload(
	activeRuns observepkg.TaskDashboardActiveRuns,
) contract.TaskDashboardActiveRunsPayload {
	payload := contract.TaskDashboardActiveRunsPayload{
		Total:    activeRuns.Total,
		Running:  activeRuns.Running,
		Starting: activeRuns.Starting,
		Claimed:  activeRuns.Claimed,
		Queued:   activeRuns.Queued,
	}
	if len(activeRuns.Items) == 0 {
		return payload
	}

	payload.Items = make([]contract.TaskDashboardActiveRunPayload, 0, len(activeRuns.Items))
	for _, item := range activeRuns.Items {
		payload.Items = append(payload.Items, contract.TaskDashboardActiveRunPayload{
			TaskID:         item.TaskID,
			TaskIdentifier: item.TaskIdentifier,
			TaskTitle:      item.TaskTitle,
			TaskStatus:     item.TaskStatus,
			TaskPriority:   item.TaskPriority,
			TaskOwner:      cloneOwnership(item.TaskOwner),
			Scope:          item.Scope,
			WorkspaceID:    item.WorkspaceID,
			RunID:          item.RunID,
			RunStatus:      item.RunStatus,
			Attempt:        item.Attempt,
			MaxAttempts:    item.MaxAttempts,
			SessionID:      item.SessionID,
			NetworkChannel: item.NetworkChannel,
			LastActivityAt: item.LastActivityAt,
			AgeMilli:       item.AgeMilli,
			HealthStatus:   item.HealthStatus,
			Stuck:          item.Stuck,
			Error:          item.Error,
		})
	}
	return payload
}

func taskDashboardFreshnessPayload(
	freshness observepkg.TaskDashboardFreshness,
) contract.TaskDashboardFreshnessPayload {
	return contract.TaskDashboardFreshnessPayload{
		ObservedAt:       freshness.ObservedAt,
		LatestActivityAt: freshness.LatestActivityAt,
		AgeMilli:         freshness.AgeMilli,
		StaleAfterMilli:  freshness.StaleAfterMilli,
		HasLiveWork:      freshness.HasLiveWork,
		Status:           freshness.Status,
		Stale:            freshness.Stale,
	}
}

// TaskInboxPayloadFromView converts one observer-backed inbox view into the shared payload.
func TaskInboxPayloadFromView(view observepkg.TaskInboxView) contract.TaskInboxPayload {
	payload := contract.TaskInboxPayload{
		Total:         view.Total,
		UnreadTotal:   view.UnreadTotal,
		ArchivedTotal: view.ArchivedTotal,
	}

	if len(view.Groups) == 0 {
		return payload
	}

	payload.Groups = make([]contract.TaskInboxLaneGroupPayload, 0, len(view.Groups))
	for _, group := range view.Groups {
		groupPayload := contract.TaskInboxLaneGroupPayload{
			Lane:        contract.TaskInboxLane(group.Lane),
			Count:       group.Count,
			UnreadCount: group.UnreadCount,
		}
		if len(group.Items) > 0 {
			groupPayload.Items = make([]contract.TaskInboxItemPayload, 0, len(group.Items))
			for _, item := range group.Items {
				groupPayload.Items = append(groupPayload.Items, contract.TaskInboxItemPayload{
					Task:             TaskReferencePayloadFromReference(item.Task),
					Lane:             contract.TaskInboxLane(item.Lane),
					ApprovalPolicy:   item.ApprovalPolicy,
					ApprovalState:    item.ApprovalState,
					BlockingReason:   item.BlockingReason,
					LatestActivityAt: item.LatestActivityAt,
					Run:              TaskRunSummaryPayloadFromSummary(item.Run),
					Triage:           TaskTriageStatePayloadFromState(item.Triage),
				})
			}
		}
		payload.Groups = append(payload.Groups, groupPayload)
	}

	return payload
}
