package apicore

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/api/contract"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

// SessionPayloadFromInfo converts a session info snapshot into the shared session payload.
func SessionPayloadFromInfo(info *session.SessionInfo) contract.SessionPayload {
	payload := contract.SessionPayload{}
	if info == nil {
		return payload
	}

	payload = contract.SessionPayload{
		ID:            info.ID,
		Name:          info.Name,
		AgentName:     info.AgentName,
		WorkspaceID:   info.WorkspaceID,
		WorkspacePath: info.Workspace,
		State:         string(info.State),
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
	workspaceID, workspacePath := sessionWorkspaceFromInfo(info)
	return contract.SessionEventPayload{
		ID:            event.ID,
		SessionID:     event.SessionID,
		Sequence:      event.Sequence,
		TurnID:        event.TurnID,
		Type:          event.Type,
		AgentName:     event.AgentName,
		WorkspaceID:   workspaceID,
		WorkspacePath: workspacePath,
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

func sessionWorkspaceFromInfo(info *session.SessionInfo) (string, string) {
	if info == nil {
		return "", ""
	}
	return strings.TrimSpace(info.WorkspaceID), strings.TrimSpace(info.Workspace)
}
