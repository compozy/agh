package cli

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/memory"
)

const (
	baseURL              = "http://unix"
	maxSSELineBytes      = 1024 * 1024
	defaultUserAgentName = "agh-cli"
)

// DaemonClient is the CLI transport surface for talking to the AGH daemon over UDS.
type DaemonClient interface {
	DaemonStatus(ctx context.Context) (DaemonStatus, error)
	ListSessions(ctx context.Context) ([]SessionRecord, error)
	CreateSession(ctx context.Context, request CreateSessionRequest) (SessionRecord, error)
	GetSession(ctx context.Context, id string) (SessionRecord, error)
	StopSession(ctx context.Context, id string) error
	ResumeSession(ctx context.Context, id string) (SessionRecord, error)
	PromptSession(ctx context.Context, id string, message string) ([]AgentEventRecord, error)
	SessionEvents(ctx context.Context, id string, query SessionEventQuery) ([]SessionEventRecord, error)
	StreamSessionEvents(ctx context.Context, id string, query SessionEventQuery, lastEventID string, handler SSEHandler) error
	SessionHistory(ctx context.Context, id string, query SessionEventQuery) ([]TurnHistoryRecord, error)
	ListAgents(ctx context.Context) ([]AgentRecord, error)
	GetAgent(ctx context.Context, name string) (AgentRecord, error)
	ObserveEvents(ctx context.Context, query ObserveEventQuery) ([]ObserveEventRecord, error)
	StreamObserveEvents(ctx context.Context, query ObserveEventQuery, lastEventID string, handler SSEHandler) error
	ObserveHealth(ctx context.Context) (HealthStatus, error)
	ListMemory(ctx context.Context, scope memory.Scope, workspace string) ([]MemoryHeaderRecord, error)
	ReadMemory(ctx context.Context, filename string, scope memory.Scope, workspace string) (MemoryReadRecord, error)
	WriteMemory(ctx context.Context, filename string, request MemoryWriteRequest) (MemoryMutationRecord, error)
	DeleteMemory(ctx context.Context, filename string, scope memory.Scope, workspace string) (MemoryMutationRecord, error)
	ConsolidateMemory(ctx context.Context, workspace string) (MemoryConsolidateRecord, error)
}

// CreateSessionRequest captures the CLI session creation payload.
type CreateSessionRequest struct {
	AgentName string `json:"agent_name,omitempty"`
	Name      string `json:"name,omitempty"`
	Workspace string `json:"workspace"`
}

// SessionRecord is the daemon API session payload.
type SessionRecord struct {
	ID           string         `json:"id"`
	Name         string         `json:"name,omitempty"`
	AgentName    string         `json:"agent_name"`
	Workspace    string         `json:"workspace"`
	State        string         `json:"state"`
	ACPSessionID string         `json:"acp_session_id,omitempty"`
	ACPCaps      *ACPCapsRecord `json:"acp_caps,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// ACPCapsRecord captures optional runtime capabilities exposed by the daemon API.
type ACPCapsRecord struct {
	SupportsLoadSession bool     `json:"supports_load_session"`
	SupportedModes      []string `json:"supported_modes,omitempty"`
	SupportedModels     []string `json:"supported_models,omitempty"`
}

// SessionEventRecord is one persisted session event row returned by the daemon API.
type SessionEventRecord struct {
	ID        string          `json:"id"`
	SessionID string          `json:"session_id"`
	Sequence  int64           `json:"sequence"`
	TurnID    string          `json:"turn_id"`
	Type      string          `json:"type"`
	AgentName string          `json:"agent_name"`
	Content   json.RawMessage `json:"content"`
	Timestamp time.Time       `json:"timestamp"`
}

// TurnHistoryRecord groups session events by turn.
type TurnHistoryRecord struct {
	TurnID string               `json:"turn_id"`
	Events []SessionEventRecord `json:"events"`
}

// SessionEventQuery captures the CLI filters for session event/history queries.
type SessionEventQuery struct {
	Type          string
	AgentName     string
	TurnID        string
	Since         time.Time
	Last          int
	AfterSequence int64
}

// AgentRecord is the daemon API agent definition payload.
type AgentRecord struct {
	Name        string           `json:"name"`
	Provider    string           `json:"provider"`
	Command     string           `json:"command,omitempty"`
	Model       string           `json:"model,omitempty"`
	Tools       []string         `json:"tools,omitempty"`
	Permissions string           `json:"permissions,omitempty"`
	MCPServers  []AgentMCPServer `json:"mcp_servers,omitempty"`
	Prompt      string           `json:"prompt"`
}

// AgentMCPServer is one MCP server entry returned by the daemon API.
type AgentMCPServer struct {
	Name    string            `json:"name"`
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// AgentEventRecord is one prompt-stream event returned by the daemon API.
type AgentEventRecord struct {
	Type       string            `json:"type"`
	SessionID  string            `json:"session_id,omitempty"`
	TurnID     string            `json:"turn_id,omitempty"`
	Timestamp  time.Time         `json:"timestamp"`
	Text       string            `json:"text,omitempty"`
	Title      string            `json:"title,omitempty"`
	ToolCallID string            `json:"tool_call_id,omitempty"`
	StopReason string            `json:"stop_reason,omitempty"`
	Action     string            `json:"action,omitempty"`
	Resource   string            `json:"resource,omitempty"`
	Decision   string            `json:"decision,omitempty"`
	Error      string            `json:"error,omitempty"`
	Usage      *TokenUsageRecord `json:"usage,omitempty"`
	Raw        json.RawMessage   `json:"raw,omitempty"`
}

// TokenUsageRecord is the prompt usage payload returned by the daemon API.
type TokenUsageRecord struct {
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

// ObserveEventRecord is one cross-session observability event row.
type ObserveEventRecord struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	Type      string    `json:"type"`
	AgentName string    `json:"agent_name"`
	Summary   string    `json:"summary,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// ObserveEventQuery captures the CLI filters for cross-session observability queries.
type ObserveEventQuery struct {
	SessionID string
	AgentName string
	Type      string
	Since     time.Time
	Last      int
}

// MemoryHeaderRecord is one memory header returned by the daemon API.
type MemoryHeaderRecord = memory.MemoryHeader

// MemoryReadRecord is the memory document payload returned by the daemon API.
type MemoryReadRecord struct {
	Content string `json:"content"`
}

// MemoryWriteRequest captures the daemon API write payload.
type MemoryWriteRequest struct {
	Content   string `json:"content"`
	Scope     string `json:"scope,omitempty"`
	Workspace string `json:"workspace,omitempty"`
}

// MemoryMutationRecord captures the daemon API write/delete response.
type MemoryMutationRecord struct {
	OK bool `json:"ok"`
}

// MemoryConsolidateRecord captures the daemon API consolidation response.
type MemoryConsolidateRecord struct {
	Triggered bool   `json:"triggered"`
	Reason    string `json:"reason,omitempty"`
}

// HealthStatus is the daemon API observability health payload.
type HealthStatus struct {
	Status             string `json:"status"`
	UptimeSeconds      int64  `json:"uptime_seconds"`
	ActiveSessions     int    `json:"active_sessions"`
	ActiveAgents       int    `json:"active_agents"`
	GlobalDBSizeBytes  int64  `json:"global_db_size_bytes"`
	SessionDBSizeBytes int64  `json:"session_db_size_bytes"`
	Version            string `json:"version"`
}

// DaemonStatus is the daemon API status payload.
type DaemonStatus struct {
	Status         string    `json:"status"`
	PID            int       `json:"pid"`
	StartedAt      time.Time `json:"started_at"`
	Socket         string    `json:"socket"`
	HTTPHost       string    `json:"http_host"`
	HTTPPort       int       `json:"http_port"`
	ActiveSessions int       `json:"active_sessions"`
	TotalSessions  int       `json:"total_sessions"`
	Version        string    `json:"version,omitempty"`
}

// IdentityRecord is the local agent identity exposed by `agh whoami`.
type IdentityRecord struct {
	SessionID string `json:"session_id,omitempty"`
	Agent     string `json:"agent,omitempty"`
	AgentName string `json:"agent_name,omitempty"`
}

// SSEEvent is one parsed server-sent event frame.
type SSEEvent struct {
	ID    string
	Event string
	Data  json.RawMessage
}

// SSEHandler consumes parsed SSE frames.
type SSEHandler func(SSEEvent) error

type unixSocketClient struct {
	socketPath string
	httpClient *http.Client
}

var errStopSSE = errors.New("cli: stop sse stream")

// NewClient constructs a daemon client that talks HTTP over a Unix domain socket.
func NewClient(socketPath string) (DaemonClient, error) {
	path := strings.TrimSpace(socketPath)
	if path == "" {
		return nil, errors.New("cli: daemon socket path is required")
	}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var dialer net.Dialer
			return dialer.DialContext(ctx, "unix", path)
		},
	}

	return &unixSocketClient{
		socketPath: path,
		httpClient: &http.Client{Transport: transport},
	}, nil
}

func (c *unixSocketClient) DaemonStatus(ctx context.Context) (DaemonStatus, error) {
	var response struct {
		Daemon DaemonStatus `json:"daemon"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/daemon/status", nil, nil, &response); err != nil {
		return DaemonStatus{}, err
	}
	return response.Daemon, nil
}

func (c *unixSocketClient) ListSessions(ctx context.Context) ([]SessionRecord, error) {
	var response struct {
		Sessions []SessionRecord `json:"sessions"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/sessions", nil, nil, &response); err != nil {
		return nil, err
	}
	return response.Sessions, nil
}

func (c *unixSocketClient) CreateSession(ctx context.Context, request CreateSessionRequest) (SessionRecord, error) {
	var response struct {
		Session SessionRecord `json:"session"`
	}
	if err := c.doJSON(ctx, http.MethodPost, "/api/sessions", nil, request, &response); err != nil {
		return SessionRecord{}, err
	}
	return response.Session, nil
}

func (c *unixSocketClient) GetSession(ctx context.Context, id string) (SessionRecord, error) {
	var response struct {
		Session SessionRecord `json:"session"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/sessions/"+url.PathEscape(strings.TrimSpace(id)), nil, nil, &response); err != nil {
		return SessionRecord{}, err
	}
	return response.Session, nil
}

func (c *unixSocketClient) StopSession(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, "/api/sessions/"+url.PathEscape(strings.TrimSpace(id)), nil, nil, nil)
}

func (c *unixSocketClient) ResumeSession(ctx context.Context, id string) (SessionRecord, error) {
	var response struct {
		Session SessionRecord `json:"session"`
	}
	if err := c.doJSON(ctx, http.MethodPost, "/api/sessions/"+url.PathEscape(strings.TrimSpace(id))+"/resume", nil, nil, &response); err != nil {
		return SessionRecord{}, err
	}
	return response.Session, nil
}

func (c *unixSocketClient) PromptSession(ctx context.Context, id string, message string) ([]AgentEventRecord, error) {
	path := "/api/sessions/" + url.PathEscape(strings.TrimSpace(id)) + "/prompt"
	var events []AgentEventRecord
	err := c.doSSE(ctx, http.MethodPost, path, nil, map[string]string{"message": message}, "", func(event SSEEvent) error {
		var payload AgentEventRecord
		if len(event.Data) > 0 {
			if err := json.Unmarshal(event.Data, &payload); err != nil {
				return fmt.Errorf("cli: decode prompt event: %w", err)
			}
		}
		if payload.Type == "" {
			payload.Type = event.Event
		}
		events = append(events, payload)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return events, nil
}

func (c *unixSocketClient) SessionEvents(ctx context.Context, id string, query SessionEventQuery) ([]SessionEventRecord, error) {
	var response struct {
		Events []SessionEventRecord `json:"events"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/sessions/"+url.PathEscape(strings.TrimSpace(id))+"/events", sessionEventValues(query), nil, &response); err != nil {
		return nil, err
	}
	return response.Events, nil
}

func (c *unixSocketClient) StreamSessionEvents(ctx context.Context, id string, query SessionEventQuery, lastEventID string, handler SSEHandler) error {
	return c.doSSE(ctx, http.MethodGet, "/api/sessions/"+url.PathEscape(strings.TrimSpace(id))+"/stream", sessionEventValues(query), nil, lastEventID, handler)
}

func (c *unixSocketClient) SessionHistory(ctx context.Context, id string, query SessionEventQuery) ([]TurnHistoryRecord, error) {
	var response struct {
		History []TurnHistoryRecord `json:"history"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/sessions/"+url.PathEscape(strings.TrimSpace(id))+"/history", sessionEventValues(query), nil, &response); err != nil {
		return nil, err
	}
	return response.History, nil
}

func (c *unixSocketClient) ListAgents(ctx context.Context) ([]AgentRecord, error) {
	var response struct {
		Agents []AgentRecord `json:"agents"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/agents", nil, nil, &response); err != nil {
		return nil, err
	}
	return response.Agents, nil
}

func (c *unixSocketClient) GetAgent(ctx context.Context, name string) (AgentRecord, error) {
	var response struct {
		Agent AgentRecord `json:"agent"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/agents/"+url.PathEscape(strings.TrimSpace(name)), nil, nil, &response); err != nil {
		return AgentRecord{}, err
	}
	return response.Agent, nil
}

func (c *unixSocketClient) ObserveEvents(ctx context.Context, query ObserveEventQuery) ([]ObserveEventRecord, error) {
	var response struct {
		Events []ObserveEventRecord `json:"events"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/observe/events", observeEventValues(query), nil, &response); err != nil {
		return nil, err
	}
	return response.Events, nil
}

func (c *unixSocketClient) StreamObserveEvents(ctx context.Context, query ObserveEventQuery, lastEventID string, handler SSEHandler) error {
	return c.doSSE(ctx, http.MethodGet, "/api/observe/events/stream", observeEventValues(query), nil, lastEventID, handler)
}

func (c *unixSocketClient) ObserveHealth(ctx context.Context) (HealthStatus, error) {
	var response struct {
		Health HealthStatus `json:"health"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/observe/health", nil, nil, &response); err != nil {
		return HealthStatus{}, err
	}
	return response.Health, nil
}

func (c *unixSocketClient) ListMemory(ctx context.Context, scope memory.Scope, workspace string) ([]MemoryHeaderRecord, error) {
	var response []MemoryHeaderRecord
	if err := c.doJSON(ctx, http.MethodGet, "/api/memory", memoryValues(scope, workspace), nil, &response); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *unixSocketClient) ReadMemory(ctx context.Context, filename string, scope memory.Scope, workspace string) (MemoryReadRecord, error) {
	var response MemoryReadRecord
	if err := c.doJSON(ctx, http.MethodGet, "/api/memory/"+url.PathEscape(strings.TrimSpace(filename)), memoryValues(scope, workspace), nil, &response); err != nil {
		return MemoryReadRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) WriteMemory(ctx context.Context, filename string, request MemoryWriteRequest) (MemoryMutationRecord, error) {
	var response MemoryMutationRecord
	if err := c.doJSON(ctx, http.MethodPut, "/api/memory/"+url.PathEscape(strings.TrimSpace(filename)), nil, request, &response); err != nil {
		return MemoryMutationRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) DeleteMemory(ctx context.Context, filename string, scope memory.Scope, workspace string) (MemoryMutationRecord, error) {
	var response MemoryMutationRecord
	if err := c.doJSON(ctx, http.MethodDelete, "/api/memory/"+url.PathEscape(strings.TrimSpace(filename)), memoryValues(scope, workspace), nil, &response); err != nil {
		return MemoryMutationRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) ConsolidateMemory(ctx context.Context, workspace string) (MemoryConsolidateRecord, error) {
	var response MemoryConsolidateRecord
	if err := c.doJSON(ctx, http.MethodPost, "/api/memory/consolidate", nil, map[string]string{"workspace": strings.TrimSpace(workspace)}, &response); err != nil {
		return MemoryConsolidateRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) doJSON(ctx context.Context, method string, path string, query url.Values, requestBody any, responseBody any) error {
	response, err := c.doRequest(ctx, method, path, query, requestBody, "")
	if err != nil {
		return err
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return readAPIError(response)
	}
	if responseBody == nil {
		_, _ = io.Copy(io.Discard, response.Body)
		return nil
	}
	if err := json.NewDecoder(response.Body).Decode(responseBody); err != nil {
		return fmt.Errorf("cli: decode %s %s response: %w", method, path, err)
	}
	return nil
}

func (c *unixSocketClient) doSSE(ctx context.Context, method string, path string, query url.Values, requestBody any, lastEventID string, handler SSEHandler) error {
	response, err := c.doRequest(ctx, method, path, query, requestBody, lastEventID)
	if err != nil {
		return err
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return readAPIError(response)
	}

	if handler == nil {
		_, _ = io.Copy(io.Discard, response.Body)
		return nil
	}
	return decodeSSE(ctx, response.Body, handler)
}

func (c *unixSocketClient) doRequest(ctx context.Context, method string, path string, query url.Values, requestBody any, lastEventID string) (*http.Response, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	target := baseURL + path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}

	var body io.Reader
	if requestBody != nil {
		payload, err := json.Marshal(requestBody)
		if err != nil {
			return nil, fmt.Errorf("cli: encode %s %s request: %w", method, path, err)
		}
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		return nil, fmt.Errorf("cli: build %s %s request: %w", method, path, err)
	}
	req.Header.Set("User-Agent", defaultUserAgentName)
	if requestBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if strings.TrimSpace(lastEventID) != "" {
		req.Header.Set("Last-Event-ID", strings.TrimSpace(lastEventID))
	}

	response, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cli: %s %s via %s: %w", method, path, c.socketPath, err)
	}
	return response, nil
}

func decodeSSE(ctx context.Context, body io.Reader, handler SSEHandler) error {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), maxSSELineBytes)

	event := SSEEvent{}
	dataLines := make([]string, 0, 4)
	emit := func() error {
		if event.ID == "" && event.Event == "" && len(dataLines) == 0 {
			return nil
		}
		if len(dataLines) > 0 {
			event.Data = json.RawMessage(strings.Join(dataLines, "\n"))
		}
		err := handler(event)
		event = SSEEvent{}
		dataLines = dataLines[:0]
		return err
	}

	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return err
		}

		line := scanner.Text()
		if line == "" {
			if err := emit(); err != nil {
				if errors.Is(err, errStopSSE) {
					return nil
				}
				return err
			}
			continue
		}
		if strings.HasPrefix(line, ":") {
			continue
		}

		field, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		value = strings.TrimPrefix(value, " ")

		switch field {
		case "id":
			event.ID = value
		case "event":
			event.Event = value
		case "data":
			dataLines = append(dataLines, value)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("cli: read sse stream: %w", err)
	}
	if err := emit(); err != nil {
		if errors.Is(err, errStopSSE) {
			return nil
		}
		return err
	}
	return nil
}

func sessionEventValues(query SessionEventQuery) url.Values {
	values := url.Values{}
	if trimmed := strings.TrimSpace(query.Type); trimmed != "" {
		values.Set("type", trimmed)
	}
	if trimmed := strings.TrimSpace(query.AgentName); trimmed != "" {
		values.Set("agent_name", trimmed)
	}
	if trimmed := strings.TrimSpace(query.TurnID); trimmed != "" {
		values.Set("turn_id", trimmed)
	}
	if !query.Since.IsZero() {
		values.Set("since", query.Since.UTC().Format(time.RFC3339Nano))
	}
	if query.Last > 0 {
		values.Set("limit", strconv.Itoa(query.Last))
	}
	if query.AfterSequence > 0 {
		values.Set("after_sequence", strconv.FormatInt(query.AfterSequence, 10))
	}
	return values
}

func observeEventValues(query ObserveEventQuery) url.Values {
	values := url.Values{}
	if trimmed := strings.TrimSpace(query.SessionID); trimmed != "" {
		values.Set("session_id", trimmed)
	}
	if trimmed := strings.TrimSpace(query.AgentName); trimmed != "" {
		values.Set("agent_name", trimmed)
	}
	if trimmed := strings.TrimSpace(query.Type); trimmed != "" {
		values.Set("type", trimmed)
	}
	if !query.Since.IsZero() {
		values.Set("since", query.Since.UTC().Format(time.RFC3339Nano))
	}
	if query.Last > 0 {
		values.Set("limit", strconv.Itoa(query.Last))
	}
	return values
}

func memoryValues(scope memory.Scope, workspace string) url.Values {
	values := url.Values{}
	if trimmed := strings.TrimSpace(string(scope)); trimmed != "" {
		values.Set("scope", trimmed)
	}
	if trimmed := strings.TrimSpace(workspace); trimmed != "" {
		values.Set("workspace", trimmed)
	}
	return values
}

func readAPIError(response *http.Response) error {
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("cli: read api error response: %w", err)
	}

	var payload struct {
		Error string `json:"error"`
	}
	if len(body) > 0 && json.Unmarshal(body, &payload) == nil && strings.TrimSpace(payload.Error) != "" {
		return errors.New(strings.TrimSpace(payload.Error))
	}

	message := strings.TrimSpace(string(body))
	if message == "" {
		message = response.Status
	}
	return fmt.Errorf("daemon api %s: %s", response.Status, message)
}
