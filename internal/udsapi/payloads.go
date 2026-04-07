package udsapi

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
)

type createSessionRequest struct {
	AgentName     string `json:"agent_name"`
	Name          string `json:"name"`
	Workspace     string `json:"workspace"`
	WorkspacePath string `json:"workspace_path"`
}

type sessionPayload struct {
	ID            string          `json:"id"`
	Name          string          `json:"name,omitempty"`
	AgentName     string          `json:"agent_name"`
	WorkspaceID   string          `json:"workspace_id,omitempty"`
	WorkspacePath string          `json:"workspace_path,omitempty"`
	State         string          `json:"state"`
	ACPSessionID  string          `json:"acp_session_id,omitempty"`
	ACPCaps       *acpCapsPayload `json:"acp_caps,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

type acpCapsPayload struct {
	SupportsLoadSession bool     `json:"supports_load_session"`
	SupportedModes      []string `json:"supported_modes,omitempty"`
	SupportedModels     []string `json:"supported_models,omitempty"`
}

type sessionEventPayload struct {
	ID            string          `json:"id"`
	SessionID     string          `json:"session_id"`
	Sequence      int64           `json:"sequence"`
	TurnID        string          `json:"turn_id"`
	Type          string          `json:"type"`
	AgentName     string          `json:"agent_name"`
	WorkspaceID   string          `json:"workspace_id,omitempty"`
	WorkspacePath string          `json:"workspace_path,omitempty"`
	Content       json.RawMessage `json:"content"`
	Timestamp     time.Time       `json:"timestamp"`
}

type turnHistoryPayload struct {
	TurnID string                `json:"turn_id"`
	Events []sessionEventPayload `json:"events"`
}

type agentPayload struct {
	Name        string               `json:"name"`
	Provider    string               `json:"provider"`
	Command     string               `json:"command,omitempty"`
	Model       string               `json:"model,omitempty"`
	Tools       []string             `json:"tools,omitempty"`
	Permissions string               `json:"permissions,omitempty"`
	MCPServers  []agentMCPServerJSON `json:"mcp_servers,omitempty"`
	Prompt      string               `json:"prompt"`
}

type agentMCPServerJSON struct {
	Name    string            `json:"name"`
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

type agentEventPayload struct {
	Type       string             `json:"type"`
	SessionID  string             `json:"session_id,omitempty"`
	TurnID     string             `json:"turn_id,omitempty"`
	Timestamp  time.Time          `json:"timestamp"`
	Text       string             `json:"text,omitempty"`
	Title      string             `json:"title,omitempty"`
	ToolCallID string             `json:"tool_call_id,omitempty"`
	StopReason string             `json:"stop_reason,omitempty"`
	Action     string             `json:"action,omitempty"`
	Resource   string             `json:"resource,omitempty"`
	Decision   string             `json:"decision,omitempty"`
	Error      string             `json:"error,omitempty"`
	Usage      *tokenUsagePayload `json:"usage,omitempty"`
	Raw        json.RawMessage    `json:"raw,omitempty"`
}

type tokenUsagePayload struct {
	TurnID           string    `json:"turn_id,omitempty"`
	InputTokens      *int64    `json:"input_tokens,omitempty"`
	OutputTokens     *int64    `json:"output_tokens,omitempty"`
	TotalTokens      *int64    `json:"total_tokens,omitempty"`
	ThoughtTokens    *int64    `json:"thought_tokens,omitempty"`
	CacheReadTokens  *int64    `json:"cache_read_tokens,omitempty"`
	CacheWriteTokens *int64    `json:"cache_write_tokens,omitempty"`
	ContextUsed      *int64    `json:"context_used,omitempty"`
	ContextSize      *int64    `json:"context_size,omitempty"`
	CostAmount       *float64  `json:"cost_amount,omitempty"`
	CostCurrency     *string   `json:"cost_currency,omitempty"`
	Timestamp        time.Time `json:"timestamp"`
}

type observeEventPayload struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	Type      string    `json:"type"`
	AgentName string    `json:"agent_name"`
	Summary   string    `json:"summary,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

func sessionPayloadFromInfo(info *session.SessionInfo) sessionPayload {
	payload := sessionPayload{}
	if info == nil {
		return payload
	}

	payload = sessionPayload{
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
	if caps := acpCapsPayloadFromInfo(info.ACPCaps); caps != nil {
		payload.ACPCaps = caps
	}
	return payload
}

func acpCapsPayloadFromInfo(caps acp.ACPCaps) *acpCapsPayload {
	if !caps.SupportsLoadSession && len(caps.SupportedModes) == 0 && len(caps.SupportedModels) == 0 {
		return nil
	}

	return &acpCapsPayload{
		SupportsLoadSession: caps.SupportsLoadSession,
		SupportedModes:      append([]string(nil), caps.SupportedModes...),
		SupportedModels:     append([]string(nil), caps.SupportedModels...),
	}
}

func sessionEventPayloadFromEvent(event store.SessionEvent, info *session.SessionInfo) sessionEventPayload {
	workspaceID, workspacePath := sessionWorkspaceFromInfo(info)
	return sessionEventPayload{
		ID:            event.ID,
		SessionID:     event.SessionID,
		Sequence:      event.Sequence,
		TurnID:        event.TurnID,
		Type:          event.Type,
		AgentName:     event.AgentName,
		WorkspaceID:   workspaceID,
		WorkspacePath: workspacePath,
		Content:       payloadJSON(event.Content),
		Timestamp:     event.Timestamp,
	}
}

func sessionWorkspaceFromInfo(info *session.SessionInfo) (string, string) {
	if info == nil {
		return "", ""
	}
	return strings.TrimSpace(info.WorkspaceID), strings.TrimSpace(info.Workspace)
}

func agentPayloadFromDef(agent aghconfig.AgentDef) agentPayload {
	mcpServers := make([]agentMCPServerJSON, 0, len(agent.MCPServers))
	for _, server := range agent.MCPServers {
		var env map[string]string
		if len(server.Env) > 0 {
			env = make(map[string]string, len(server.Env))
			for key, value := range server.Env {
				env[key] = value
			}
		}

		mcpServers = append(mcpServers, agentMCPServerJSON{
			Name:    server.Name,
			Command: server.Command,
			Args:    append([]string(nil), server.Args...),
			Env:     env,
		})
	}

	return agentPayload{
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

func agentEventPayloadFromEvent(event acp.AgentEvent) agentEventPayload {
	return agentEventPayload{
		Type:       event.Type,
		SessionID:  event.SessionID,
		TurnID:     event.TurnID,
		Timestamp:  event.Timestamp,
		Text:       event.Text,
		Title:      event.Title,
		ToolCallID: event.ToolCallID,
		StopReason: event.StopReason,
		Action:     event.Action,
		Resource:   event.Resource,
		Decision:   event.Decision,
		Error:      event.Error,
		Usage:      tokenUsagePayloadFromUsage(event.Usage),
		Raw:        payloadJSON(string(event.Raw)),
	}
}

func tokenUsagePayloadFromUsage(usage *acp.TokenUsage) *tokenUsagePayload {
	if usage == nil {
		return nil
	}

	return &tokenUsagePayload{
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

func observeEventPayloadFromEvent(event store.EventSummary) observeEventPayload {
	return observeEventPayload{
		ID:        event.ID,
		SessionID: event.SessionID,
		Type:      event.Type,
		AgentName: event.AgentName,
		Summary:   event.Summary,
		Timestamp: event.Timestamp,
	}
}

func payloadJSON(raw string) json.RawMessage {
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
