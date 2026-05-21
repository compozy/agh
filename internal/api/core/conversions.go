package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
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
	eventspkg "github.com/pedronauck/agh/internal/events"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/notifications"
	observepkg "github.com/pedronauck/agh/internal/observe"
	registrypkg "github.com/pedronauck/agh/internal/registry"
	"github.com/pedronauck/agh/internal/resources"
	"github.com/pedronauck/agh/internal/session"
	settingspkg "github.com/pedronauck/agh/internal/settings"
	"github.com/pedronauck/agh/internal/skills"
	skillmarketplace "github.com/pedronauck/agh/internal/skills/marketplace"
	ssepkg "github.com/pedronauck/agh/internal/sse"
	"github.com/pedronauck/agh/internal/store"
	taskpkg "github.com/pedronauck/agh/internal/task"
	"github.com/pedronauck/agh/internal/workref"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

const (
	maxDiagnosticPayloadBytes = 2048
	skillWarningSeverityInfo  = "info"
	skillWarningSeverityWarn  = "warning"
	skillWarningSeverityCrit  = "critical"
)

// SessionPayloadFromInfo converts a session info snapshot into the shared session payload.
func SessionPayloadFromInfo(info *session.Info) contract.SessionPayload {
	payload := contract.SessionPayload{}
	if info == nil {
		return payload
	}

	ref := workref.NewPath(info.WorkspaceID, info.Workspace)
	payload = contract.SessionPayload{
		ID:              info.ID,
		Name:            info.Name,
		AgentName:       info.AgentName,
		Provider:        info.Provider,
		Model:           strings.TrimSpace(info.Model),
		ReasoningEffort: strings.TrimSpace(info.ReasoningEffort),
		WorkspaceID:     ref.WorkspaceID,
		WorkspacePath:   ref.WorkspacePath,
		Channel:         info.Channel,
		Type:            info.Type,
		State:           info.State,
		StopReason:      info.StopReason,
		StopDetail:      info.StopDetail,
		Failure:         SessionFailurePayloadFromStore(info.Failure),
		ACPSessionID:    info.ACPSessionID,
		Lineage:         contract.SessionLineagePayloadFromStore(info.Lineage),
		CreatedAt:       info.CreatedAt,
		UpdatedAt:       info.UpdatedAt,
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
		DeadlineAt:         nil,
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
			payload.ElapsedMS = elapsed.Milliseconds()
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
		DeadlineAt:         cloneTimePtr(activity.DeadlineAt),
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
		ElapsedMS:          activity.ElapsedMS,
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
	if !caps.SupportsLoadSession &&
		len(caps.SupportedModes) == 0 &&
		len(caps.SupportedModels) == 0 &&
		len(caps.ConfigOptions) == 0 {
		return nil
	}

	return &contract.ACPCapsPayload{
		SupportsLoadSession: caps.SupportsLoadSession,
		SupportedModes:      append([]string(nil), caps.SupportedModes...),
		SupportedModels:     append([]string(nil), caps.SupportedModels...),
		ConfigOptions:       SessionConfigOptionPayloadsFromInfo(caps.ConfigOptions),
	}
}

// SessionConfigOptionPayloadsFromInfo converts active ACP config options into the shared payload.
func SessionConfigOptionPayloadsFromInfo(options []acp.SessionConfigOption) []contract.SessionConfigOptionPayload {
	if len(options) == 0 {
		return nil
	}
	payloads := make([]contract.SessionConfigOptionPayload, 0, len(options))
	for _, option := range options {
		payloads = append(payloads, contract.SessionConfigOptionPayload{
			ID:          strings.TrimSpace(option.ID),
			Label:       strings.TrimSpace(option.Label),
			Description: strings.TrimSpace(option.Description),
			Kind:        string(option.Kind),
			Current:     strings.TrimSpace(option.Current),
			Values:      sessionConfigOptionValuePayloads(option.Values),
		})
	}
	return payloads
}

func sessionConfigOptionValuePayloads(
	values []acp.SessionConfigOptionValue,
) []contract.SessionConfigOptionValuePayload {
	if len(values) == 0 {
		return nil
	}
	payloads := make([]contract.SessionConfigOptionValuePayload, 0, len(values))
	for _, value := range values {
		payloads = append(payloads, contract.SessionConfigOptionValuePayload{
			Value:       strings.TrimSpace(value.Value),
			Label:       strings.TrimSpace(value.Label),
			Description: strings.TrimSpace(value.Description),
		})
	}
	return payloads
}

// SessionEventPayloadFromEvent converts a session event into the shared payload.
func SessionEventPayloadFromEvent(event store.SessionEvent, info *session.Info) contract.SessionEventPayload {
	ref := workref.NewPath(sessionWorkspaceFromInfo(info))
	payload := contract.SessionEventPayload{
		ID:               event.ID,
		SessionID:        event.SessionID,
		Sequence:         event.Sequence,
		TurnID:           event.TurnID,
		Type:             event.Type,
		AgentName:        event.AgentName,
		WorkspaceID:      ref.WorkspaceID,
		WorkspacePath:    ref.WorkspacePath,
		EventCorrelation: sessionEventCorrelation(event),
		Content:          PayloadJSON(event.Content),
		Timestamp:        event.Timestamp,
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
		Name:         agent.Name,
		Provider:     agent.Provider,
		Command:      agent.Command,
		Model:        agent.Model,
		Tools:        append([]string(nil), agent.Tools...),
		Toolsets:     append([]string(nil), agent.Toolsets...),
		DenyTools:    append([]string(nil), agent.DenyTools...),
		Permissions:  agent.Permissions,
		CategoryPath: append([]string(nil), agent.CategoryPath...),
		MCPServers:   mcpServers,
		Prompt:       agent.Prompt,
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

// LogEventPayloadFromSummary converts an event summary into the shared logs payload.
func LogEventPayloadFromSummary(event store.EventSummary) contract.LogEventPayload {
	return contract.LogEventPayload{
		ID:               event.ID,
		SessionID:        event.SessionID,
		WorkspaceID:      event.WorkspaceID,
		Type:             event.Type,
		AgentName:        event.AgentName,
		Provider:         event.Provider,
		Component:        eventspkg.ComponentFor(event.Type),
		Outcome:          logEventOutcome(event),
		Content:          ssepkg.ScrubMemoryContextBytes(append([]byte(nil), event.Content...)),
		EventCorrelation: event.Normalize(),
		ParentSessionID:  event.ParentSessionID,
		RootSessionID:    event.RootSessionID,
		SpawnDepth:       event.SpawnDepth,
		Summary:          ssepkg.ScrubMemoryContextString(event.Summary),
		Timestamp:        event.Timestamp,
	}
}

func logEventOutcome(event store.EventSummary) string {
	outcome := strings.TrimSpace(event.Outcome)
	if outcome != "" {
		return outcome
	}
	return string(eventspkg.OutcomeFor(event.Type))
}

func sessionEventCorrelation(event store.SessionEvent) store.EventCorrelation {
	decoded, err := decodeSessionEventCorrelation(event.Content)
	if err != nil {
		slog.Warn(
			"api: decode session event correlation failed",
			"event_id",
			strings.TrimSpace(event.ID),
			"session_id",
			strings.TrimSpace(event.SessionID),
			"type",
			strings.TrimSpace(event.Type),
			"turn_id",
			strings.TrimSpace(event.TurnID),
			"error",
			err,
		)
		return store.EventCorrelation{}
	}
	return decoded.Normalize()
}

type sessionEventCorrelationPayload struct {
	TaskID               string `json:"task_id"`
	RunID                string `json:"run_id"`
	WorkflowID           string `json:"workflow_id"`
	ClaimTokenHash       string `json:"claim_token_hash"`
	LeaseUntil           string `json:"lease_until"`
	CoordinatorSessionID string `json:"coordinator_session_id"`
	SchedulerReason      string `json:"scheduler_reason"`
	HookEvent            string `json:"hook_event"`
	HookName             string `json:"hook_name"`
	ActorKind            string `json:"actor_kind"`
	ActorID              string `json:"actor_id"`
	ReleaseReason        string `json:"release_reason"`
}

func decodeSessionEventCorrelation(payload string) (store.EventCorrelation, error) {
	trimmed := strings.TrimSpace(payload)
	if trimmed == "" {
		return store.EventCorrelation{}, nil
	}

	var decoded sessionEventCorrelationPayload
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		return store.EventCorrelation{}, fmt.Errorf("api: unmarshal session event correlation: %w", err)
	}

	leaseUntil, err := parseSessionEventCorrelationTimestamp(decoded.LeaseUntil)
	if err != nil {
		return store.EventCorrelation{}, fmt.Errorf("api: parse session event lease_until: %w", err)
	}

	return store.EventCorrelation{
		TaskID:               decoded.TaskID,
		RunID:                decoded.RunID,
		WorkflowID:           decoded.WorkflowID,
		ClaimTokenHash:       decoded.ClaimTokenHash,
		LeaseUntil:           leaseUntil,
		CoordinatorSessionID: decoded.CoordinatorSessionID,
		SchedulerReason:      decoded.SchedulerReason,
		HookEvent:            decoded.HookEvent,
		HookName:             decoded.HookName,
		ActorKind:            decoded.ActorKind,
		ActorID:              decoded.ActorID,
		ReleaseReason:        decoded.ReleaseReason,
	}, nil
}

func parseSessionEventCorrelationTimestamp(value string) (*time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, trimmed)
	if err != nil {
		return nil, err
	}
	normalized := parsed.UTC()
	return &normalized, nil
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

// TaskHealthPayloadFromObserve converts observer task health into the shared status payload.
func TaskHealthPayloadFromObserve(health observepkg.TaskHealth) contract.TaskHealthPayload {
	return contract.TaskHealthPayload{
		Status:                     strings.TrimSpace(health.Status),
		QueueDepthTotal:            health.QueueDepthTotal,
		OldestQueuedAt:             optionalTime(health.OldestQueuedAt),
		OldestQueueAgeMilli:        health.OldestQueueAgeMilli,
		QueueDepth:                 TaskQueueDepthPayloadsFromObserve(health.QueueDepth),
		StuckRuns:                  StuckTaskRunPayloadsFromObserve(health.StuckRuns),
		ActiveOrphanRuns:           health.ActiveOrphanRuns,
		TaskTotals:                 TaskStatusTotalPayloadsFromObserve(health.TaskTotals),
		RunTotals:                  TaskRunTotalPayloadsFromObserve(health.RunTotals),
		OwnerTotals:                TaskOwnerTotalPayloadsFromObserve(health.OwnerTotals),
		ForcedStopsSinceStart:      health.ForcedStopsSinceStart,
		DuplicateIngressSinceStart: health.DuplicateIngressSinceStart,
		ChannelMismatchSinceStart:  health.ChannelMismatchSinceStart,
		RecoverySinceStart: contract.TaskRecoveryTotalsPayload{
			Requeued:      health.RecoverySinceStart.Requeued,
			MarkedRunning: health.RecoverySinceStart.MarkedRunning,
			Failed:        health.RecoverySinceStart.Failed,
		},
	}
}

// TaskQueueDepthPayloadsFromObserve converts task queue-depth rows.
func TaskQueueDepthPayloadsFromObserve(rows []observepkg.TaskQueueDepth) []contract.TaskQueueDepthPayload {
	if len(rows) == 0 {
		return nil
	}
	payloads := make([]contract.TaskQueueDepthPayload, 0, len(rows))
	for _, row := range rows {
		payloads = append(payloads, contract.TaskQueueDepthPayload{
			NetworkChannel:      strings.TrimSpace(row.NetworkChannel),
			Count:               row.Count,
			OldestQueuedAt:      optionalTime(row.OldestQueuedAt),
			OldestQueueAgeMilli: row.OldestQueueAgeMilli,
		})
	}
	return payloads
}

// StuckTaskRunPayloadsFromObserve converts stuck task-run diagnostics.
func StuckTaskRunPayloadsFromObserve(rows []observepkg.StuckTaskRun) []contract.StuckTaskRunPayload {
	if len(rows) == 0 {
		return nil
	}
	payloads := make([]contract.StuckTaskRunPayload, 0, len(rows))
	for _, row := range rows {
		payloads = append(payloads, contract.StuckTaskRunPayload{
			TaskID:         strings.TrimSpace(row.TaskID),
			RunID:          strings.TrimSpace(row.RunID),
			Status:         strings.TrimSpace(string(row.Status)),
			OriginKind:     strings.TrimSpace(string(row.OriginKind)),
			NetworkChannel: strings.TrimSpace(row.NetworkChannel),
			SessionID:      strings.TrimSpace(row.SessionID),
			AgeMillis:      row.AgeMillis,
		})
	}
	return payloads
}

// TaskStatusTotalPayloadsFromObserve converts task status buckets.
func TaskStatusTotalPayloadsFromObserve(rows []observepkg.TaskStatusTotal) []contract.TaskStatusTotalPayload {
	if len(rows) == 0 {
		return nil
	}
	payloads := make([]contract.TaskStatusTotalPayload, 0, len(rows))
	for _, row := range rows {
		payloads = append(payloads, contract.TaskStatusTotalPayload{
			Scope:          strings.TrimSpace(string(row.Scope)),
			Status:         strings.TrimSpace(string(row.Status)),
			NetworkChannel: strings.TrimSpace(row.NetworkChannel),
			Count:          row.Count,
		})
	}
	return payloads
}

// TaskRunTotalPayloadsFromObserve converts task-run status buckets.
func TaskRunTotalPayloadsFromObserve(rows []observepkg.TaskRunTotal) []contract.TaskRunTotalPayload {
	if len(rows) == 0 {
		return nil
	}
	payloads := make([]contract.TaskRunTotalPayload, 0, len(rows))
	for _, row := range rows {
		payloads = append(payloads, contract.TaskRunTotalPayload{
			Status:         strings.TrimSpace(string(row.Status)),
			OriginKind:     strings.TrimSpace(string(row.OriginKind)),
			NetworkChannel: strings.TrimSpace(row.NetworkChannel),
			Count:          row.Count,
		})
	}
	return payloads
}

// TaskOwnerTotalPayloadsFromObserve converts task ownership buckets.
func TaskOwnerTotalPayloadsFromObserve(rows []observepkg.TaskOwnerTotal) []contract.TaskOwnerTotalPayload {
	if len(rows) == 0 {
		return nil
	}
	payloads := make([]contract.TaskOwnerTotalPayload, 0, len(rows))
	for _, row := range rows {
		payloads = append(payloads, contract.TaskOwnerTotalPayload{
			OwnerKind: strings.TrimSpace(string(row.OwnerKind)),
			OwnerRef:  strings.TrimSpace(row.OwnerRef),
			Count:     row.Count,
		})
	}
	return payloads
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

// TaskBridgeNotificationSubscriptionPayloadFromSubscription converts one
// bridge task subscription into the shared task-scoped transport payload.
func TaskBridgeNotificationSubscriptionPayloadFromSubscription(
	subscription bridgepkg.BridgeTaskSubscription,
) contract.TaskBridgeNotificationSubscriptionPayload {
	normalized := subscription.Normalize()
	return contract.TaskBridgeNotificationSubscriptionPayload{
		SubscriptionID:   normalized.SubscriptionID,
		TaskID:           normalized.TaskID,
		BridgeInstanceID: normalized.BridgeInstanceID,
		Scope:            normalized.Scope,
		WorkspaceID:      normalized.WorkspaceID,
		PeerID:           normalized.PeerID,
		ThreadID:         normalized.ThreadID,
		GroupID:          normalized.GroupID,
		DeliveryMode:     normalized.DeliveryMode,
		Cursor:           TaskBridgeNotificationCursorPayloadFromKey(normalized.CursorKey()),
		CreatedBy:        normalized.CreatedBy,
		CreatedAt:        normalized.CreatedAt,
		UpdatedAt:        normalized.UpdatedAt,
	}
}

// TaskBridgeNotificationSubscriptionPayloadFromSubscriptionAndCursor converts
// one bridge task subscription with its persisted cursor diagnostics.
func TaskBridgeNotificationSubscriptionPayloadFromSubscriptionAndCursor(
	subscription bridgepkg.BridgeTaskSubscription,
	cursor notifications.Cursor,
) contract.TaskBridgeNotificationSubscriptionPayload {
	payload := TaskBridgeNotificationSubscriptionPayloadFromSubscription(subscription)
	payload.Cursor = TaskBridgeNotificationCursorPayloadFromCursor(cursor)
	return payload
}

// TaskBridgeNotificationSubscriptionPayloadsFromSubscriptions converts
// bridge task subscriptions into shared task-scoped transport payloads.
func TaskBridgeNotificationSubscriptionPayloadsFromSubscriptions(
	subscriptions []bridgepkg.BridgeTaskSubscription,
) []contract.TaskBridgeNotificationSubscriptionPayload {
	payloads := make([]contract.TaskBridgeNotificationSubscriptionPayload, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		payloads = append(payloads, TaskBridgeNotificationSubscriptionPayloadFromSubscription(subscription))
	}
	return payloads
}

// TaskBridgeNotificationCursorPayloadFromKey converts a durable cursor identity
// into the transport diagnostics shape before any delivery has been persisted.
func TaskBridgeNotificationCursorPayloadFromKey(
	key notifications.CursorKey,
) contract.TaskBridgeNotificationCursorPayload {
	normalized, err := key.Normalize()
	if err != nil {
		normalized = notifications.CursorKey{
			ConsumerID: strings.TrimSpace(key.ConsumerID),
			StreamName: strings.TrimSpace(key.StreamName),
			SubjectID:  strings.TrimSpace(key.SubjectID),
		}
	}
	return contract.TaskBridgeNotificationCursorPayload{
		ConsumerID:   normalized.ConsumerID,
		StreamName:   normalized.StreamName,
		SubjectID:    normalized.SubjectID,
		LastSequence: 0,
	}
}

// TaskBridgeNotificationCursorPayloadFromCursor converts persisted cursor
// diagnostics into the transport payload used by HTTP, UDS, CLI, and web.
func TaskBridgeNotificationCursorPayloadFromCursor(
	cursor notifications.Cursor,
) contract.TaskBridgeNotificationCursorPayload {
	payload := TaskBridgeNotificationCursorPayloadFromKey(cursor.Key)
	payload.LastSequence = cursor.LastSequence
	payload.LastDeliveryID = cursor.LastDeliveryID
	payload.LastError = cursor.LastError
	if !cursor.LastDeliveredAt.IsZero() {
		lastDeliveredAt := cursor.LastDeliveredAt.UTC()
		payload.LastDeliveredAt = &lastDeliveredAt
	}
	if !cursor.UpdatedAt.IsZero() {
		updatedAt := cursor.UpdatedAt.UTC()
		payload.UpdatedAt = &updatedAt
	}
	return payload
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

	defaultProvider := ""
	if cfg != nil {
		defaultProvider = aghconfig.CanonicalProviderName(cfg.Defaults.Provider)
	}
	return sortSessionProviderOptionPayloads(payloadsByName, defaultProvider)
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
	return sortSessionProviderOptionPayloads(values, "")
}

func sortSessionProviderOptionPayloads(
	values map[string]contract.SessionProviderOptionPayload,
	defaultProvider string,
) []contract.SessionProviderOptionPayload {
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	defaultProvider = aghconfig.CanonicalProviderName(defaultProvider)
	if defaultProvider != "" {
		for i, name := range names {
			if name != defaultProvider {
				continue
			}
			copy(names[1:i+1], names[:i])
			names[0] = defaultProvider
			break
		}
	}
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
		Diagnostics: SkillDiagnosticPayloadsFromDiagnostics(skills.DiagnosticsForSkill(skill)),
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

// SkillDiagnosticPayloadsFromDiagnostics converts skill registry diagnostics for API payloads.
func SkillDiagnosticPayloadsFromDiagnostics(
	diagnostics []skills.SkillDiagnostic,
) []contract.SkillDiagnosticPayload {
	if len(diagnostics) == 0 {
		return nil
	}
	payloads := make([]contract.SkillDiagnosticPayload, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		payloads = append(payloads, skillDiagnosticPayloadFromDiagnostic(diagnostic))
	}
	return payloads
}

func skillDiagnosticPayloadFromDiagnostic(
	diagnostic skills.SkillDiagnostic,
) contract.SkillDiagnosticPayload {
	verificationStatus := diagnostic.VerificationStatus
	if verificationStatus == "" {
		verificationStatus = skills.SkillVerificationStatusPassed
	}
	return contract.SkillDiagnosticPayload{
		Name:               diagnostic.Name,
		State:              contract.SkillDiagnosticState(diagnostic.State),
		Source:             diagnostic.Source,
		Path:               diagnostic.Path,
		WinningSource:      diagnostic.WinningSource,
		WinningPath:        diagnostic.WinningPath,
		VerificationStatus: contract.SkillVerificationStatus(verificationStatus),
		Warnings:           skillVerificationWarningPayloads(diagnostic.Warnings),
		Failure:            skillVerificationFailurePayload(diagnostic.Failure),
	}
}

func skillVerificationWarningPayloads(
	warnings []skills.Warning,
) []contract.SkillVerificationWarningPayload {
	if len(warnings) == 0 {
		return nil
	}
	payloads := make([]contract.SkillVerificationWarningPayload, 0, len(warnings))
	for _, warning := range warnings {
		payloads = append(payloads, contract.SkillVerificationWarningPayload{
			Severity: skillWarningSeverityName(warning.Severity),
			Pattern:  warning.Pattern,
			Message:  warning.Message,
		})
	}
	return payloads
}

func skillWarningSeverityName(severity skills.WarningSeverity) string {
	switch severity {
	case skills.SeverityCritical:
		return skillWarningSeverityCrit
	case skills.SeverityWarning:
		return skillWarningSeverityWarn
	default:
		return skillWarningSeverityInfo
	}
}

func skillVerificationFailurePayload(
	failure *skills.SkillVerificationFailure,
) *contract.SkillVerificationFailurePayload {
	if failure == nil {
		return nil
	}
	return &contract.SkillVerificationFailurePayload{
		Code:         failure.Code,
		Message:      failure.Message,
		ExpectedHash: failure.ExpectedHash,
		ActualHash:   failure.ActualHash,
	}
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

// SkillMarketplaceListingPayloadFromListing converts a remote listing into the shared payload.
func SkillMarketplaceListingPayloadFromListing(
	listing registrypkg.Listing,
) contract.SkillMarketplaceListingPayload {
	return contract.SkillMarketplaceListingPayload{
		Slug:        listing.Slug,
		Name:        listing.Name,
		Description: listing.Description,
		Author:      listing.Author,
		Version:     listing.Version,
		Downloads:   listing.Downloads,
		Source:      listing.Source,
	}
}

// SkillMarketplaceListingPayloadsFromListings converts remote listings into shared payloads.
func SkillMarketplaceListingPayloadsFromListings(
	listings []registrypkg.Listing,
) []contract.SkillMarketplaceListingPayload {
	payload := make([]contract.SkillMarketplaceListingPayload, 0, len(listings))
	for _, listing := range listings {
		payload = append(payload, SkillMarketplaceListingPayloadFromListing(listing))
	}
	return payload
}

// SkillMarketplaceDetailPayloadFromDetail converts a remote detail into the shared payload.
func SkillMarketplaceDetailPayloadFromDetail(
	detail *registrypkg.Detail,
) contract.SkillMarketplaceDetailPayload {
	if detail == nil {
		return contract.SkillMarketplaceDetailPayload{}
	}
	return contract.SkillMarketplaceDetailPayload{
		Slug:        detail.Slug,
		Name:        detail.Name,
		Description: detail.Description,
		Author:      detail.Author,
		Version:     detail.Version,
		Downloads:   detail.Downloads,
		Source:      detail.Source,
		Readme:      detail.Readme,
		MCPServers:  append([]string(nil), detail.MCPServers...),
		Tags:        append([]string(nil), detail.Tags...),
		License:     detail.License,
		Repository:  detail.Repository,
		Versions:    append([]string(nil), detail.Versions...),
	}
}

// SkillMarketplaceInstallPayloadFromResult converts an install result into the shared payload.
func SkillMarketplaceInstallPayloadFromResult(
	result skillmarketplace.InstallResult,
) contract.SkillMarketplaceInstallPayload {
	return contract.SkillMarketplaceInstallPayload{
		Name:     result.Name,
		Slug:     result.Slug,
		Version:  result.Version,
		Registry: result.Registry,
		Path:     result.Path,
		Hash:     result.Hash,
		Status:   result.Status,
	}
}

// SkillMarketplaceUpdatePayloadsFromResults converts update results into shared payloads.
func SkillMarketplaceUpdatePayloadsFromResults(
	results []skillmarketplace.UpdateResult,
) []contract.SkillMarketplaceUpdatePayload {
	payload := make([]contract.SkillMarketplaceUpdatePayload, 0, len(results))
	for _, result := range results {
		payload = append(payload, contract.SkillMarketplaceUpdatePayload{
			Name:           result.Name,
			Slug:           result.Slug,
			CurrentVersion: result.CurrentVersion,
			LatestVersion:  result.LatestVersion,
			Path:           result.Path,
			Status:         result.Status,
		})
	}
	return payload
}

// SkillMarketplaceRemovePayloadFromResult converts a removal result into the shared payload.
func SkillMarketplaceRemovePayloadFromResult(
	result skillmarketplace.RemoveResult,
) contract.SkillMarketplaceRemovePayload {
	return contract.SkillMarketplaceRemovePayload{
		Name:   result.Name,
		Slug:   result.Slug,
		Path:   result.Path,
		Status: result.Status,
	}
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
		SettingsGlobalSectionResponseMetaPayload: settingsGlobalSectionMetaPayload(envelope),
		ConfigPaths:                              settingsConfigPathsPayload(envelope.General.ConfigPaths),
		Config:                                   settingsGeneralConfigPayload(envelope.General.Settings),
		Runtime:                                  settingsDaemonRuntimePayload(envelope.General.Runtime),
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
		SettingsGlobalSectionResponseMetaPayload: settingsGlobalSectionMetaPayload(envelope),
		Config:                                   settingsMemoryConfigPayload(&envelope.Memory.Config),
		Health:                                   settingsMemoryHealthPayload(envelope.Memory.Health),
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
		SettingsSkillsSectionResponseMetaPayload: settingsSkillsSectionMetaPayload(envelope),
		Config:                                   settingsSkillsConfigPayload(envelope.Skills.Config),
		DiscoveredCount:                          envelope.Skills.DiscoveredCount,
		DisabledCount:                            envelope.Skills.DisabledCount,
		RuntimeAvailable:                         envelope.Skills.RuntimeAvailable,
		Diagnostics:                              SkillDiagnosticPayloadsFromDiagnostics(envelope.Skills.Diagnostics),
		Links:                                    settingsOperationalLinkPayloads(envelope.Skills.Links),
	}, nil
}

func settingsAutomationSectionResponse(envelope settingspkg.SectionEnvelope) (any, error) {
	if envelope.Automation == nil {
		return nil, errors.New("settings automation section is required")
	}
	return contract.SettingsAutomationResponse{
		SettingsGlobalSectionResponseMetaPayload: settingsGlobalSectionMetaPayload(envelope),
		Config:                                   settingsAutomationConfigPayload(envelope.Automation.Config),
		Runtime:                                  settingsAutomationRuntimePayload(envelope.Automation.Runtime),
		Links:                                    settingsOperationalLinkPayloads(envelope.Automation.Links),
	}, nil
}

func settingsNetworkSectionResponse(envelope settingspkg.SectionEnvelope) (any, error) {
	if envelope.Network == nil {
		return nil, errors.New("settings network section is required")
	}
	return contract.SettingsNetworkResponse{
		SettingsGlobalSectionResponseMetaPayload: settingsGlobalSectionMetaPayload(envelope),
		Config:                                   settingsNetworkConfigPayload(envelope.Network.Config),
		Runtime:                                  settingsNetworkRuntimePayload(envelope.Network.Runtime),
		Links:                                    settingsOperationalLinkPayloads(envelope.Network.Links),
	}, nil
}

func settingsObservabilitySectionResponse(envelope settingspkg.SectionEnvelope) (any, error) {
	if envelope.Observability == nil {
		return nil, errors.New("settings observability section is required")
	}
	return contract.SettingsObservabilityResponse{
		SettingsGlobalSectionResponseMetaPayload: settingsGlobalSectionMetaPayload(envelope),
		Config:                                   settingsObservabilityConfigPayload(envelope.Observability.Config),
		Runtime:                                  settingsObservabilityRuntimePayload(envelope.Observability.Runtime),
		LogTail: settingsLogTailCapabilityPayload(
			envelope.Observability.LogTailSupport,
		),
	}, nil
}

func settingsHooksExtensionsSectionResponse(envelope settingspkg.SectionEnvelope) (any, error) {
	if envelope.HooksExtensions == nil {
		return nil, errors.New("settings hooks-extensions section is required")
	}
	return contract.SettingsHooksExtensionsResponse{
		SettingsGlobalSectionResponseMetaPayload: settingsGlobalSectionMetaPayload(envelope),
		Hooks:                                    settingsHookItemPayloads(envelope.HooksExtensions.Hooks),
		Config:                                   settingsExtensionsConfigPayload(envelope.HooksExtensions.Extensions),
		Installed: settingsInstalledExtensionPayloads(
			envelope.HooksExtensions.Installed,
		),
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
			SettingsGlobalCollectionResponseMetaPayload: settingsGlobalCollectionMetaPayload(envelope),
			Providers: settingsProviderItemPayloads(envelope.Providers),
		}, nil
	case settingspkg.CollectionMCPServers:
		return contract.SettingsMCPServersResponse{
			SettingsGlobalWorkspaceCollectionResponseMetaPayload: settingsGlobalWorkspaceCollectionMetaPayload(
				envelope,
			),
			MCPServers: settingsMCPServerItemPayloads(envelope.MCPServers),
		}, nil
	case settingspkg.CollectionSandboxes:
		return contract.SettingsSandboxesResponse{
			SettingsGlobalCollectionResponseMetaPayload: settingsGlobalCollectionMetaPayload(envelope),
			Sandboxes: settingsSandboxItemPayloads(envelope.Sandboxes),
		}, nil
	case settingspkg.CollectionHooks:
		return contract.SettingsHooksResponse{
			SettingsGlobalCollectionResponseMetaPayload: settingsGlobalCollectionMetaPayload(envelope),
			Hooks: settingsHookItemPayloads(envelope.Hooks),
		}, nil
	default:
		return nil, fmt.Errorf("unknown settings collection %q", envelope.Collection)
	}
}

// SettingsSectionMutationResultPayloadFromResult converts one settings section mutation result into the shared payload.
func SettingsSectionMutationResultPayloadFromResult(result settingspkg.MutationResult) (any, error) {
	switch result.Section {
	case settingspkg.SectionGeneral,
		settingspkg.SectionMemory,
		settingspkg.SectionAutomation,
		settingspkg.SectionNetwork,
		settingspkg.SectionObservability,
		settingspkg.SectionHooksExtensions:
		return contract.SettingsGlobalSectionMutationResult{
			Section:         contract.SettingsSectionName(result.Section),
			Scope:           contract.SettingsGlobalScopeKind(result.Scope),
			WriteTarget:     contract.SettingsWriteTargetKind(result.WriteTarget),
			Behavior:        contract.SettingsMutationBehavior(result.Behavior),
			Applied:         result.Applied,
			RestartRequired: result.RestartRequired,
			RestartScope:    strings.TrimSpace(result.RestartScope),
			Warnings:        cloneStrings(result.Warnings),
		}, nil
	case settingspkg.SectionSkills:
		return contract.SettingsSkillsMutationResult{
			Section:         contract.SettingsSectionName(result.Section),
			Scope:           contract.SettingsAgentScopeKind(result.Scope),
			WriteTarget:     contract.SettingsWriteTargetKind(result.WriteTarget),
			WorkspaceID:     strings.TrimSpace(result.WorkspaceID),
			AgentName:       strings.TrimSpace(result.AgentName),
			Behavior:        contract.SettingsMutationBehavior(result.Behavior),
			Applied:         result.Applied,
			RestartRequired: result.RestartRequired,
			RestartScope:    strings.TrimSpace(result.RestartScope),
			Warnings:        cloneStrings(result.Warnings),
		}, nil
	default:
		return nil, fmt.Errorf("unknown settings section mutation %q", result.Section)
	}
}

// SettingsCollectionMutationResultPayloadFromResult converts one settings
// collection mutation result into the shared payload.
func SettingsCollectionMutationResultPayloadFromResult(result settingspkg.MutationResult) (any, error) {
	collection := contract.SettingsCollectionName(result.Section)
	switch collection {
	case contract.SettingsCollectionProviders,
		contract.SettingsCollectionSandboxes,
		contract.SettingsCollectionHooks:
		return contract.SettingsGlobalCollectionMutationResult{
			Section:         collection,
			Scope:           contract.SettingsGlobalScopeKind(result.Scope),
			WriteTarget:     contract.SettingsWriteTargetKind(result.WriteTarget),
			Behavior:        contract.SettingsMutationBehavior(result.Behavior),
			Applied:         result.Applied,
			RestartRequired: result.RestartRequired,
			RestartScope:    strings.TrimSpace(result.RestartScope),
			Warnings:        cloneStrings(result.Warnings),
		}, nil
	case contract.SettingsCollectionMCPServers:
		return contract.SettingsGlobalWorkspaceCollectionMutationResult{
			Section:         collection,
			Scope:           contract.SettingsWorkspaceScopeKind(result.Scope),
			WriteTarget:     contract.SettingsWriteTargetKind(result.WriteTarget),
			WorkspaceID:     strings.TrimSpace(result.WorkspaceID),
			Behavior:        contract.SettingsMutationBehavior(result.Behavior),
			Applied:         result.Applied,
			RestartRequired: result.RestartRequired,
			RestartScope:    strings.TrimSpace(result.RestartScope),
			Warnings:        cloneStrings(result.Warnings),
		}, nil
	default:
		return nil, fmt.Errorf("unknown settings collection mutation %q", result.Section)
	}
}

// SettingsApplyResponseFromResult converts one settings apply result into the public payload.
func SettingsApplyResponseFromResult(result settingspkg.ApplyResult) contract.SettingsApplyResponse {
	return contract.SettingsApplyResponse{
		Section:          contract.SettingsApplyTargetName(strings.TrimSpace(string(result.Section))),
		Scope:            contract.SettingsScopeKind(strings.TrimSpace(string(result.Scope))),
		WriteTarget:      contract.SettingsWriteTargetKind(result.WriteTarget),
		WorkspaceID:      strings.TrimSpace(result.WorkspaceID),
		AgentName:        strings.TrimSpace(result.AgentName),
		Applied:          result.Applied,
		Lifecycle:        contract.SettingsApplyLifecycle(result.Record.Lifecycle),
		ApplyRecordID:    strings.TrimSpace(result.Record.ID),
		ActiveGeneration: result.Record.Generation,
		ActiveConfigHash: strings.TrimSpace(result.Record.ActiveHash),
		NextAction:       contract.SettingsApplyNextAction(result.NextAction),
		RestartRequired:  result.RestartRequired,
		RestartScope:     strings.TrimSpace(result.RestartScope),
		Warnings:         cloneStrings(result.Warnings),
		PartialFailures:  settingsApplyFailurePayloads(result.PartialFailures),
		Skipped:          result.Skipped,
		SkippedReason:    strings.TrimSpace(result.SkippedReason),
	}
}

// ConfigApplyRecordsResponseFromRecords converts apply history rows into the public payload.
func ConfigApplyRecordsResponseFromRecords(
	records []settingspkg.ApplyRecord,
) contract.ConfigApplyRecordsResponse {
	entries := make([]contract.ConfigApplyRecordPayload, 0, len(records))
	for _, record := range records {
		entries = append(entries, configApplyRecordPayload(record))
	}
	return contract.ConfigApplyRecordsResponse{Entries: entries}
}

func configApplyRecordPayload(record settingspkg.ApplyRecord) contract.ConfigApplyRecordPayload {
	return contract.ConfigApplyRecordPayload{
		ID:                strings.TrimSpace(record.ID),
		DesiredConfigHash: strings.TrimSpace(record.DesiredHash),
		ActiveConfigHash:  strings.TrimSpace(record.ActiveHash),
		Generation:        record.Generation,
		Actor:             strings.TrimSpace(record.Actor),
		DiffClass:         contract.SettingsApplyLifecycle(record.DiffClass),
		Status:            contract.ConfigApplyStatus(record.Status),
		Lifecycle:         contract.SettingsApplyLifecycle(record.Lifecycle),
		NextAction:        contract.SettingsApplyNextAction(record.NextAction),
		Diagnostics:       append([]contract.DiagnosticItem(nil), record.Diagnostics...),
		CreatedAt:         record.CreatedAt,
		AppliedAt:         record.AppliedAt,
		UpdatedAt:         record.UpdatedAt,
	}
}

func settingsApplyFailurePayloads(
	failures []settingspkg.ApplyFailure,
) []contract.SettingsApplyFailurePayload {
	if len(failures) == 0 {
		return nil
	}
	payloads := make([]contract.SettingsApplyFailurePayload, 0, len(failures))
	for _, failure := range failures {
		payloads = append(payloads, contract.SettingsApplyFailurePayload{
			Subsystem:  strings.TrimSpace(failure.Subsystem),
			Diagnostic: failure.Diagnostic,
		})
	}
	return payloads
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

func settingsGlobalSectionMetaPayload(
	envelope settingspkg.SectionEnvelope,
) contract.SettingsGlobalSectionResponseMetaPayload {
	return contract.SettingsGlobalSectionResponseMetaPayload{
		Section:         contract.SettingsSectionName(envelope.Section),
		Scope:           contract.SettingsGlobalScopeKind(envelope.Scope),
		AvailableScopes: settingsGlobalScopeKindsPayload(envelope.AvailableScopes),
	}
}

func settingsSkillsSectionMetaPayload(
	envelope settingspkg.SectionEnvelope,
) contract.SettingsSkillsSectionResponseMetaPayload {
	return contract.SettingsSkillsSectionResponseMetaPayload{
		Section:         contract.SettingsSectionName(envelope.Section),
		Scope:           contract.SettingsAgentScopeKind(envelope.Scope),
		WorkspaceID:     strings.TrimSpace(envelope.WorkspaceID),
		AgentName:       strings.TrimSpace(envelope.AgentName),
		AvailableScopes: settingsAgentScopeKindsPayload(envelope.AvailableScopes),
	}
}

func settingsGlobalCollectionMetaPayload(
	envelope settingspkg.CollectionEnvelope,
) contract.SettingsGlobalCollectionResponseMetaPayload {
	return contract.SettingsGlobalCollectionResponseMetaPayload{
		Collection:      contract.SettingsCollectionName(envelope.Collection),
		Scope:           contract.SettingsGlobalScopeKind(envelope.Scope),
		AvailableScopes: settingsGlobalScopeKindsPayload(envelope.AvailableScopes),
	}
}

func settingsGlobalWorkspaceCollectionMetaPayload(
	envelope settingspkg.CollectionEnvelope,
) contract.SettingsGlobalWorkspaceCollectionResponseMetaPayload {
	return contract.SettingsGlobalWorkspaceCollectionResponseMetaPayload{
		Collection:      contract.SettingsCollectionName(envelope.Collection),
		Scope:           contract.SettingsWorkspaceScopeKind(envelope.Scope),
		WorkspaceID:     strings.TrimSpace(envelope.WorkspaceID),
		AvailableScopes: settingsWorkspaceScopeKindsPayload(envelope.AvailableScopes),
	}
}

func settingsGlobalScopeKindsPayload(scopes []settingspkg.ScopeKind) []contract.SettingsGlobalScopeKind {
	if len(scopes) == 0 {
		return nil
	}
	payloads := make([]contract.SettingsGlobalScopeKind, 0, len(scopes))
	for _, scope := range scopes {
		payloads = append(payloads, contract.SettingsGlobalScopeKind(scope))
	}
	return payloads
}

func settingsAgentScopeKindsPayload(scopes []settingspkg.ScopeKind) []contract.SettingsAgentScopeKind {
	if len(scopes) == 0 {
		return nil
	}
	payloads := make([]contract.SettingsAgentScopeKind, 0, len(scopes))
	for _, scope := range scopes {
		payloads = append(payloads, contract.SettingsAgentScopeKind(scope))
	}
	return payloads
}

func settingsWorkspaceScopeKindsPayload(
	scopes []settingspkg.ScopeKind,
) []contract.SettingsWorkspaceScopeKind {
	if len(scopes) == 0 {
		return nil
	}
	payloads := make([]contract.SettingsWorkspaceScopeKind, 0, len(scopes))
	for _, scope := range scopes {
		payloads = append(payloads, contract.SettingsWorkspaceScopeKind(scope))
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
			ReloadTimeouts: contract.SettingsDaemonReloadTimeoutsPayload{
				Providers: value.Daemon.ReloadTimeouts.Providers.String(),
				MCP:       value.Daemon.ReloadTimeouts.MCP.String(),
				Bridges:   value.Daemon.ReloadTimeouts.Bridges.String(),
			},
		},
	}
}

func settingsMemoryConfigPayload(value *aghconfig.MemoryConfig) contract.SettingsMemoryConfigPayload {
	if value == nil {
		return contract.SettingsMemoryConfigPayload{}
	}
	return contract.SettingsMemoryConfigPayload{
		Enabled:    value.Enabled,
		GlobalDir:  strings.TrimSpace(value.GlobalDir),
		Controller: settingsMemoryControllerPayload(value.Controller),
		Recall:     settingsMemoryRecallPayload(value.Recall),
		Decisions:  settingsMemoryDecisionsPayload(value.Decisions),
		Extractor:  settingsMemoryExtractorPayload(value.Extractor),
		Dream:      settingsMemoryDreamPayload(value.Dream),
		Session:    settingsMemorySessionPayload(value.Session),
		Daily:      settingsMemoryDailyPayload(value.Daily),
		File:       contract.SettingsMemoryFilePayload{MaxLines: value.File.MaxLines, MaxBytes: value.File.MaxBytes},
		Provider:   settingsMemoryProviderPayload(value.Provider),
		Workspace: contract.SettingsMemoryWorkspacePayload{
			TOMLPath:   strings.TrimSpace(value.Workspace.TOMLPath),
			AutoCreate: value.Workspace.AutoCreate,
		},
	}
}

func settingsMemoryControllerPayload(value aghconfig.MemoryControllerConfig) contract.SettingsMemoryControllerPayload {
	return contract.SettingsMemoryControllerPayload{
		Mode:            strings.TrimSpace(value.Mode),
		MaxLatency:      value.MaxLatency.String(),
		DefaultOpOnFail: strings.TrimSpace(value.DefaultOpOnFail),
		LLM: contract.SettingsMemoryControllerLLMPayload{
			Enabled:       value.LLM.Enabled,
			Model:         strings.TrimSpace(value.LLM.Model),
			TopK:          value.LLM.TopK,
			PromptVersion: strings.TrimSpace(value.LLM.PromptVersion),
			Timeout:       value.LLM.Timeout.String(),
			MaxTokensOut:  value.LLM.MaxTokensOut,
		},
		Policy: contract.SettingsMemoryControllerPolicyPayload{
			MaxContentChars: value.Policy.MaxContentChars,
			MaxWritesPerMin: value.Policy.MaxWritesPerMin,
			AllowOrigins:    cloneStrings(value.Policy.AllowOrigins),
		},
	}
}

func settingsMemoryRecallPayload(value aghconfig.MemoryRecallConfig) contract.SettingsMemoryRecallPayload {
	return contract.SettingsMemoryRecallPayload{
		TopK:                   value.TopK,
		RawCandidates:          value.RawCandidates,
		Fusion:                 strings.TrimSpace(value.Fusion),
		IncludeAlreadySurfaced: value.IncludeAlreadySurfaced,
		IncludeSystem:          value.IncludeSystem,
		Weights: contract.SettingsMemoryRecallWeightsPayload{
			BM25Unicode:  value.Weights.BM25Unicode,
			BM25Trigram:  value.Weights.BM25Trigram,
			Recency:      value.Weights.Recency,
			RecallSignal: value.Weights.RecallSignal,
		},
		Freshness: contract.SettingsMemoryRecallFreshnessPayload{
			BannerAfterDays: value.Freshness.BannerAfterDays,
		},
		Signals: contract.SettingsMemoryRecallSignalsPayload{
			QueueCapacity:  value.Signals.QueueCapacity,
			WorkerRetryMax: value.Signals.WorkerRetryMax,
			MetricsEnabled: value.Signals.MetricsEnabled,
		},
	}
}

func settingsMemoryDecisionsPayload(value aghconfig.MemoryDecisionsConfig) contract.SettingsMemoryDecisionsPayload {
	return contract.SettingsMemoryDecisionsPayload{
		PruneAfterAppliedDays: value.PruneAfterAppliedDays,
		KeepAuditSummary:      value.KeepAuditSummary,
		MaxPostContentBytes:   value.MaxPostContentBytes,
	}
}

func settingsMemoryExtractorPayload(value aghconfig.MemoryExtractorConfig) contract.SettingsMemoryExtractorPayload {
	return contract.SettingsMemoryExtractorPayload{
		Enabled:          value.Enabled,
		Mode:             strings.TrimSpace(value.Mode),
		ThrottleTurns:    value.ThrottleTurns,
		Deadline:         value.Deadline.String(),
		SandboxInboxOnly: value.SandboxInboxOnly,
		InboxPath:        strings.TrimSpace(value.InboxPath),
		DLQPath:          strings.TrimSpace(value.DLQPath),
		Model:            strings.TrimSpace(value.Model),
		Queue: contract.SettingsMemoryExtractorQueuePayload{
			Capacity:    value.Queue.Capacity,
			CoalesceMax: value.Queue.CoalesceMax,
		},
	}
}

func settingsMemoryDreamPayload(value aghconfig.DreamConfig) contract.SettingsMemoryDreamPayload {
	return contract.SettingsMemoryDreamPayload{
		Enabled:       value.Enabled,
		Agent:         strings.TrimSpace(value.Agent),
		MinHours:      value.MinHours,
		MinSessions:   value.MinSessions,
		Debounce:      value.Debounce.String(),
		PromptVersion: strings.TrimSpace(value.PromptVersion),
		CheckInterval: value.CheckInterval.String(),
		Gates: contract.SettingsMemoryDreamGatesPayload{
			MinUnpromoted:  value.Gates.MinUnpromoted,
			MinRecallCount: value.Gates.MinRecallCount,
			MinScore:       value.Gates.MinScore,
		},
		Scoring: settingsMemoryDreamScoringPayload(value.Scoring),
	}
}

func settingsMemoryDreamScoringPayload(
	value aghconfig.MemoryDreamScoringConfig,
) contract.SettingsMemoryDreamScoringPayload {
	return contract.SettingsMemoryDreamScoringPayload{
		RecencyHalfLifeDays: value.RecencyHalfLifeDays,
		Weights: contract.SettingsMemoryDreamScoringWeightsPayload{
			Frequency: value.Weights.Frequency,
			Relevance: value.Weights.Relevance,
			Recency:   value.Weights.Recency,
			Freshness: value.Weights.Freshness,
		},
	}
}

func settingsMemorySessionPayload(value aghconfig.MemorySessionConfig) contract.SettingsMemorySessionPayload {
	return contract.SettingsMemorySessionPayload{
		LedgerFormat:     strings.TrimSpace(value.LedgerFormat),
		LedgerRoot:       strings.TrimSpace(value.LedgerRoot),
		EventsPurgeGrace: value.EventsPurgeGrace.String(),
		ColdArchiveDays:  value.ColdArchiveDays,
		HardDeleteDays:   value.HardDeleteDays,
		MaxArchiveBytes:  value.MaxArchiveBytes,
		UnboundPartition: strings.TrimSpace(value.UnboundPartition),
	}
}

func settingsMemoryDailyPayload(value aghconfig.MemoryDailyConfig) contract.SettingsMemoryDailyPayload {
	return contract.SettingsMemoryDailyPayload{
		MaxBytes:        value.MaxBytes,
		MaxLines:        value.MaxLines,
		RotateFormat:    strings.TrimSpace(value.RotateFormat),
		DreamingWindow:  value.DreamingWindow,
		ColdArchiveDays: value.ColdArchiveDays,
		HardDeleteDays:  value.HardDeleteDays,
		MaxArchiveBytes: value.MaxArchiveBytes,
		SweepHour:       value.SweepHour,
		ArchivePath:     strings.TrimSpace(value.ArchivePath),
	}
}

func settingsMemoryProviderPayload(value aghconfig.MemoryProviderConfig) contract.SettingsMemoryProviderPayload {
	return contract.SettingsMemoryProviderPayload{
		Name:             strings.TrimSpace(value.Name),
		Timeout:          value.Timeout.String(),
		FailureThreshold: value.FailureThreshold,
		Cooldown:         value.Cooldown.String(),
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
	for idx := range values {
		payloads = append(payloads, settingsProviderItemPayload(&values[idx]))
	}
	return payloads
}

func settingsProviderItemPayload(value *settingspkg.ProviderItem) contract.SettingsProviderItemPayload {
	if value == nil {
		return contract.SettingsProviderItemPayload{}
	}
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
		Models:          settingsProviderModelsPayload(value.Models),
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

func settingsProviderModelsPayload(
	value aghconfig.ProviderModelsConfig,
) *contract.SettingsProviderModelsPayload {
	if providerModelsConfigIsEmpty(value) {
		return nil
	}
	return &contract.SettingsProviderModelsPayload{
		Default:   strings.TrimSpace(value.Default),
		Curated:   settingsProviderModelPayloads(value.Curated),
		Discovery: settingsProviderModelsDiscoveryPayload(value.Discovery),
	}
}

func settingsProviderModelsDiscoveryPayload(
	value aghconfig.ProviderModelsDiscoveryConfig,
) *contract.SettingsProviderModelsDiscoveryPayload {
	if value.Enabled == nil &&
		strings.TrimSpace(value.Command) == "" &&
		strings.TrimSpace(value.Endpoint) == "" &&
		strings.TrimSpace(value.Timeout) == "" {
		return nil
	}
	return &contract.SettingsProviderModelsDiscoveryPayload{
		Enabled:  cloneBoolPtr(value.Enabled),
		Command:  strings.TrimSpace(value.Command),
		Endpoint: strings.TrimSpace(value.Endpoint),
		Timeout:  strings.TrimSpace(value.Timeout),
	}
}

func settingsProviderModelPayloads(
	values []aghconfig.ProviderModelConfig,
) []contract.SettingsProviderModelPayload {
	if values == nil {
		return nil
	}
	payloads := make([]contract.SettingsProviderModelPayload, 0, len(values))
	for _, value := range values {
		payloads = append(payloads, contract.SettingsProviderModelPayload{
			ID:                     strings.TrimSpace(value.ID),
			DisplayName:            strings.TrimSpace(value.DisplayName),
			ContextWindow:          cloneInt64Ptr(value.ContextWindow),
			MaxInputTokens:         cloneInt64Ptr(value.MaxInputTokens),
			MaxOutputTokens:        cloneInt64Ptr(value.MaxOutputTokens),
			SupportsTools:          cloneBoolPtr(value.SupportsTools),
			SupportsReasoning:      cloneBoolPtr(value.SupportsReasoning),
			ReasoningEfforts:       cloneStrings(value.ReasoningEfforts),
			DefaultReasoningEffort: strings.TrimSpace(value.DefaultReasoningEffort),
			CostInputPerMillion:    cloneFloat64Ptr(value.CostInputPerMillion),
			CostOutputPerMillion:   cloneFloat64Ptr(value.CostOutputPerMillion),
		})
	}
	return payloads
}

func providerModelsConfigIsEmpty(value aghconfig.ProviderModelsConfig) bool {
	return strings.TrimSpace(value.Default) == "" &&
		value.Curated == nil &&
		value.Discovery.Enabled == nil &&
		strings.TrimSpace(value.Discovery.Command) == "" &&
		strings.TrimSpace(value.Discovery.Endpoint) == "" &&
		strings.TrimSpace(value.Discovery.Timeout) == ""
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
		Code:       strings.TrimSpace(value.Code),
		Message:    strings.TrimSpace(value.Message),
		StatusCmd:  strings.TrimSpace(value.StatusCmd),
		LoginCmd:   strings.TrimSpace(value.LoginCmd),
		LoginEnv:   cloneStrings(value.LoginEnv),
	}
	if value.NativeCLI != nil {
		payload.NativeCLI = &contract.SettingsProviderNativeCLIStatusPayload{
			Command: strings.TrimSpace(value.NativeCLI.Command),
			Present: value.NativeCLI.Present,
			Path:    strings.TrimSpace(value.NativeCLI.Path),
			Source:  strings.TrimSpace(value.NativeCLI.Source),
			Error:   strings.TrimSpace(value.NativeCLI.Error),
		}
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
			RuntimeStatus:  settingsMCPServerRuntimeStatusPayload(value.RuntimeStatus),
			Scope:          contract.SettingsScopeKind(value.Scope),
			WorkspaceID:    strings.TrimSpace(value.WorkspaceID),
			SourceMetadata: settingsSourceMetadataPayload(value.SourceMetadata),
		})
	}
	return payloads
}

func settingsMCPServerRuntimeStatusPayload(
	value *settingspkg.MCPServerRuntimeStatus,
) *contract.SettingsMCPServerRuntimeStatusPayload {
	if value == nil {
		return nil
	}
	return &contract.SettingsMCPServerRuntimeStatusPayload{
		Configured:  value.Configured,
		Initialized: value.Initialized,
		State:       strings.TrimSpace(string(value.State)),
		Probe:       strings.TrimSpace(string(value.Probe)),
		ToolCount:   value.ToolCount,
		Reason:      strings.TrimSpace(value.Reason),
		Diagnostic:  strings.TrimSpace(value.Diagnostic),
	}
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
		AgentName:   strings.TrimSpace(value.AgentName),
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

func cloneBoolPtr(src *bool) *bool {
	if src == nil {
		return nil
	}
	value := *src
	return &value
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
		ID:             record.ID,
		Identifier:     record.Identifier,
		Title:          record.Title,
		Status:         record.Status,
		Priority:       record.Priority,
		Owner:          cloneOwnership(record.Owner),
		Scope:          record.Scope,
		WorkspaceID:    record.WorkspaceID,
		LatestEventSeq: record.LatestEventSeq,
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
			LatestEventSeq: item.LatestEventSeq,
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
