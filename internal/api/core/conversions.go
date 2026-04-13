package core

import (
	"encoding/json"
	"maps"
	"path/filepath"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/api/contract"
	automationpkg "github.com/pedronauck/agh/internal/automation"
	aghconfig "github.com/pedronauck/agh/internal/config"
	observepkg "github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/skills"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/workref"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

// SessionPayloadFromInfo converts a session info snapshot into the shared session payload.
func SessionPayloadFromInfo(info *session.SessionInfo) contract.SessionPayload {
	payload := contract.SessionPayload{}
	if info == nil {
		return payload
	}

	ref := workref.NewPath(info.WorkspaceID, info.Workspace)
	payload = contract.SessionPayload{
		ID:            info.ID,
		Name:          info.Name,
		AgentName:     info.AgentName,
		WorkspaceID:   ref.WorkspaceID,
		WorkspacePath: ref.WorkspacePath,
		Channel:       info.Channel,
		State:         info.State,
		StopReason:    info.StopReason,
		StopDetail:    info.StopDetail,
		ACPSessionID:  info.ACPSessionID,
		CreatedAt:     info.CreatedAt,
		UpdatedAt:     info.UpdatedAt,
	}
	if caps := ACPCapsPayloadFromInfo(info.ACPCaps); caps != nil {
		payload.ACPCaps = caps
	}
	return payload
}

// SessionPayloadsFromInfos converts a session list into response payloads.
func SessionPayloadsFromInfos(infos []*session.SessionInfo) []contract.SessionPayload {
	payload := make([]contract.SessionPayload, 0, len(infos))
	for _, info := range infos {
		if info == nil {
			continue
		}
		payload = append(payload, SessionPayloadFromInfo(info))
	}
	return payload
}

// ACPCapsPayloadFromInfo converts ACP capability info into the shared payload.
func ACPCapsPayloadFromInfo(caps acp.ACPCaps) *contract.ACPCapsPayload {
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
func SessionEventPayloadFromEvent(event store.SessionEvent, info *session.SessionInfo) contract.SessionEventPayload {
	ref := workref.NewPath(sessionWorkspaceFromInfo(info))
	return contract.SessionEventPayload{
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
}

// AgentPayloadFromDef converts an agent definition into the shared payload.
func AgentPayloadFromDef(agent aghconfig.AgentDef) contract.AgentPayload {
	mcpServers := make([]contract.AgentMCPServerJSON, 0, len(agent.MCPServers))
	for _, server := range agent.MCPServers {
		var env map[string]string
		if len(server.Env) > 0 {
			env = make(map[string]string, len(server.Env))
			for key, value := range server.Env {
				env[key] = value
			}
		}

		mcpServers = append(mcpServers, contract.AgentMCPServerJSON{
			Name:    server.Name,
			Command: server.Command,
			Args:    append([]string(nil), server.Args...),
			Env:     env,
		})
	}

	return contract.AgentPayload{
		Name:        agent.Name,
		Provider:    agent.Provider,
		Command:     agent.Command,
		Model:       agent.Model,
		Tools:       append([]string(nil), agent.Tools...),
		Permissions: agent.Permissions,
		MCPServers:  mcpServers,
		Prompt:      agent.Prompt,
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
		Usage:      TokenUsagePayloadFromUsage(event.Usage),
		Raw:        PayloadJSON(string(event.Raw)),
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
		ID:        event.ID,
		SessionID: event.SessionID,
		Type:      event.Type,
		AgentName: event.AgentName,
		Summary:   event.Summary,
		Timestamp: event.Timestamp,
	}
}

// ObserveHealthPayloadFromHealth converts the observer health snapshot into the shared payload.
func ObserveHealthPayloadFromHealth(health observepkg.Health) contract.ObserveHealthPayload {
	return contract.ObserveHealthPayload{
		Status:             health.Status,
		UptimeSeconds:      health.UptimeSeconds,
		ActiveSessions:     health.ActiveSessions,
		ActiveAgents:       health.ActiveAgents,
		GlobalDBSizeBytes:  health.GlobalDBSizeBytes,
		SessionDBSizeBytes: health.SessionDBSizeBytes,
		Bridges:            BridgeAggregateHealthPayloadFromObserve(health.Bridges),
		Version:            health.Version,
	}
}

// AutomationHealthPayloadFromStatus converts manager status into the shared
// additive automation health block.
func AutomationHealthPayloadFromStatus(enabled bool, status automationpkg.ManagerStatus) contract.AutomationHealthPayload {
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
	}
}

// JobPayloadFromJob converts an automation job into the shared response
// payload, optionally enriching it with scheduler next-run metadata.
func JobPayloadFromJob(job automationpkg.Job, nextRun *time.Time) contract.JobPayload {
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
	}
	if job.Schedule != nil {
		schedule := *job.Schedule
		payload.Schedule = &schedule
	}
	return payload
}

// JobPayloadsFromJobs converts a slice of jobs into response payloads using
// the supplied next-run map.
func JobPayloadsFromJobs(jobs []automationpkg.Job, nextRunByID map[string]*time.Time) []contract.JobPayload {
	payloads := make([]contract.JobPayload, 0, len(jobs))
	for _, job := range jobs {
		payloads = append(payloads, JobPayloadFromJob(job, timePointerFromMap(nextRunByID, job.ID)))
	}
	return payloads
}

// TriggerPayloadFromTrigger converts an automation trigger into the shared
// response payload.
func TriggerPayloadFromTrigger(trigger automationpkg.Trigger) contract.TriggerPayload {
	return contract.TriggerPayload{
		ID:           trigger.ID,
		Scope:        trigger.Scope,
		Name:         trigger.Name,
		AgentName:    trigger.AgentName,
		WorkspaceID:  trigger.WorkspaceID,
		Prompt:       trigger.Prompt,
		Event:        trigger.Event,
		Filter:       cloneFilter(trigger.Filter),
		Enabled:      trigger.Enabled,
		Retry:        trigger.Retry,
		FireLimit:    trigger.FireLimit,
		Source:       trigger.Source,
		WebhookID:    trigger.WebhookID,
		EndpointSlug: trigger.EndpointSlug,
		CreatedAt:    trigger.CreatedAt,
		UpdatedAt:    trigger.UpdatedAt,
	}
}

// TriggerPayloadsFromTriggers converts a slice of triggers into response payloads.
func TriggerPayloadsFromTriggers(triggers []automationpkg.Trigger) []contract.TriggerPayload {
	payloads := make([]contract.TriggerPayload, 0, len(triggers))
	for _, trigger := range triggers {
		payloads = append(payloads, TriggerPayloadFromTrigger(trigger))
	}
	return payloads
}

// RunPayloadFromRun converts an automation run into the shared response payload.
func RunPayloadFromRun(run automationpkg.Run) contract.RunPayload {
	return contract.RunPayload{
		ID:        run.ID,
		JobID:     run.JobID,
		TriggerID: run.TriggerID,
		SessionID: run.SessionID,
		Status:    run.Status,
		Attempt:   run.Attempt,
		StartedAt: run.StartedAt,
		EndedAt:   run.EndedAt,
		Error:     run.Error,
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
func BridgeAggregateHealthPayloadFromObserve(summary observepkg.BridgeAggregateHealth) contract.BridgeAggregateHealthPayload {
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
		LastError:               health.LastError,
		LastErrorAt:             lastErrorAt,
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

func sessionWorkspaceFromInfo(info *session.SessionInfo) (string, string) {
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

func cloneFilter(source map[string]string) map[string]string {
	if len(source) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}
