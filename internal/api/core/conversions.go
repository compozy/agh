package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/api/contract"
	automationpkg "github.com/pedronauck/agh/internal/automation"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	aghconfig "github.com/pedronauck/agh/internal/config"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	observepkg "github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/resources"
	"github.com/pedronauck/agh/internal/session"
	settingspkg "github.com/pedronauck/agh/internal/settings"
	"github.com/pedronauck/agh/internal/skills"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/workref"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

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
	if environment := SessionEnvironmentPayloadFromMeta(info.Environment); environment != nil {
		payload.Environment = environment
	}
	return payload
}

// SessionEnvironmentPayloadFromMeta converts session environment metadata into the shared payload.
func SessionEnvironmentPayloadFromMeta(meta *store.SessionEnvironmentMeta) *contract.SessionEnvironmentPayload {
	if meta == nil {
		return nil
	}
	return &contract.SessionEnvironmentPayload{
		EnvironmentID: strings.TrimSpace(meta.EnvironmentID),
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
			maps.Copy(env, server.Env)
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
		TaskID:    run.TaskID,
		TaskRunID: run.TaskRunID,
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
		ID:             workspace.ID,
		RootDir:        workspace.RootDir,
		AddDirs:        addDirs,
		Name:           workspace.Name,
		DefaultAgent:   workspace.DefaultAgent,
		EnvironmentRef: workspace.EnvironmentRef,
		CreatedAt:      workspace.CreatedAt,
		UpdatedAt:      workspace.UpdatedAt,
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
	case settingspkg.CollectionEnvironments:
		return contract.SettingsEnvironmentsResponse{
			SettingsCollectionResponseMetaPayload: settingsCollectionMetaPayload(envelope),
			Environments:                          settingsEnvironmentItemPayloads(envelope.Environments),
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
			Agent:       strings.TrimSpace(value.Defaults.Agent),
			Provider:    strings.TrimSpace(value.Defaults.Provider),
			Environment: strings.TrimSpace(value.Defaults.Environment),
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
		APIKeyEnvPresent: value.APIKeyEnvPresent,
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
		Command:      strings.TrimSpace(value.Command),
		DefaultModel: strings.TrimSpace(value.DefaultModel),
		APIKeyEnv:    strings.TrimSpace(value.APIKeyEnv),
	}
}

func settingsMCPServerItemPayloads(values []settingspkg.MCPServerItem) []contract.SettingsMCPServerItemPayload {
	if len(values) == 0 {
		return nil
	}
	payloads := make([]contract.SettingsMCPServerItemPayload, 0, len(values))
	for _, value := range values {
		payloads = append(payloads, contract.SettingsMCPServerItemPayload{
			Name:           strings.TrimSpace(value.Name),
			Command:        strings.TrimSpace(value.Command),
			Args:           cloneStrings(value.Args),
			Env:            cloneStringMap(value.Env),
			Scope:          contract.SettingsScopeKind(value.Scope),
			WorkspaceID:    strings.TrimSpace(value.WorkspaceID),
			SourceMetadata: settingsSourceMetadataPayload(value.SourceMetadata),
		})
	}
	return payloads
}

func settingsEnvironmentItemPayloads(values []settingspkg.EnvironmentItem) []contract.SettingsEnvironmentItemPayload {
	if len(values) == 0 {
		return nil
	}
	payloads := make([]contract.SettingsEnvironmentItemPayload, 0, len(values))
	for _, value := range values {
		payloads = append(payloads, contract.SettingsEnvironmentItemPayload{
			Name:                strings.TrimSpace(value.Name),
			Profile:             settingsEnvironmentProfilePayload(value.Profile),
			WorkspaceUsageCount: value.WorkspaceUsageCount,
			SourceMetadata:      settingsSourceMetadataPayload(value.SourceMetadata),
		})
	}
	return payloads
}

func settingsEnvironmentProfilePayload(value aghconfig.EnvironmentProfile) contract.SettingsEnvironmentProfilePayload {
	payload := contract.SettingsEnvironmentProfilePayload{
		Backend:     strings.TrimSpace(value.Backend),
		SyncMode:    strings.TrimSpace(value.SyncMode),
		Persistence: strings.TrimSpace(value.Persistence),
		RuntimeRoot: strings.TrimSpace(value.RuntimeRoot),
		Env:         cloneStringMap(value.Env),
	}
	if network := settingsEnvironmentNetworkPayload(value.Network); network != nil {
		payload.Network = network
	}
	if daytona := settingsEnvironmentDaytonaPayload(value.Daytona); daytona != nil {
		payload.Daytona = daytona
	}
	return payload
}

func settingsEnvironmentNetworkPayload(
	value aghconfig.NetworkProfile,
) *contract.SettingsEnvironmentNetworkPayload {
	if !value.AllowPublicIngress &&
		!value.AllowOutbound &&
		!value.Required &&
		len(value.AllowList) == 0 &&
		len(value.DenyList) == 0 {
		return nil
	}
	return &contract.SettingsEnvironmentNetworkPayload{
		AllowPublicIngress: value.AllowPublicIngress,
		AllowOutbound:      value.AllowOutbound,
		AllowList:          cloneStrings(value.AllowList),
		DenyList:           cloneStrings(value.DenyList),
		Required:           value.Required,
	}
}

func settingsEnvironmentDaytonaPayload(
	value aghconfig.DaytonaProfile,
) *contract.SettingsEnvironmentDaytonaPayload {
	if strings.TrimSpace(value.APIURL) == "" &&
		strings.TrimSpace(value.Target) == "" &&
		strings.TrimSpace(value.Image) == "" &&
		strings.TrimSpace(value.Snapshot) == "" &&
		strings.TrimSpace(value.Class) == "" &&
		strings.TrimSpace(value.AutoStop) == "" &&
		strings.TrimSpace(value.AutoArchive) == "" {
		return nil
	}
	return &contract.SettingsEnvironmentDaytonaPayload{
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
		Priority:     value.Priority,
		Timeout:      durationString(value.Timeout),
		Matcher:      value.Matcher,
		ExecutorKind: value.ExecutorKind,
		Command:      strings.TrimSpace(value.Command),
		Args:         cloneStrings(value.Args),
		Env:          cloneStringMap(value.Env),
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

func cloneFilter(source map[string]string) map[string]string {
	if len(source) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(source))
	maps.Copy(cloned, source)
	return cloned
}
