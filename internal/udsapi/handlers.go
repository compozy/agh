package udsapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
)

type handlerConfig struct {
	sessions     SessionManager
	observer     Observer
	homePaths    aghconfig.HomePaths
	config       aghconfig.Config
	logger       *slog.Logger
	startedAt    time.Time
	now          func() time.Time
	pollInterval time.Duration
	agentLoader  AgentLoader
}

// Handlers expose request/response and SSE endpoints for the AGH API.
type Handlers struct {
	sessions     SessionManager
	observer     Observer
	homePaths    aghconfig.HomePaths
	config       aghconfig.Config
	logger       *slog.Logger
	startedAt    time.Time
	now          func() time.Time
	pollInterval time.Duration
	agentLoader  AgentLoader
	streamDone   <-chan struct{}
}

type createSessionRequest struct {
	AgentName string `json:"agent_name"`
	Name      string `json:"name"`
	Workspace string `json:"workspace"`
}

type promptRequest struct {
	Message string `json:"message"`
}

type sessionPayload struct {
	ID           string          `json:"id"`
	Name         string          `json:"name,omitempty"`
	AgentName    string          `json:"agent_name"`
	Workspace    string          `json:"workspace"`
	State        string          `json:"state"`
	ACPSessionID string          `json:"acp_session_id,omitempty"`
	ACPCaps      *acpCapsPayload `json:"acp_caps,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type acpCapsPayload struct {
	SupportsLoadSession bool     `json:"supports_load_session"`
	SupportedModes      []string `json:"supported_modes,omitempty"`
	SupportedModels     []string `json:"supported_models,omitempty"`
}

type sessionEventPayload struct {
	ID        string          `json:"id"`
	SessionID string          `json:"session_id"`
	Sequence  int64           `json:"sequence"`
	TurnID    string          `json:"turn_id"`
	Type      string          `json:"type"`
	AgentName string          `json:"agent_name"`
	Content   json.RawMessage `json:"content"`
	Timestamp time.Time       `json:"timestamp"`
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

type daemonStatusPayload struct {
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

type errorPayload struct {
	Error string `json:"error"`
}

type sseMessage struct {
	ID   string
	Name string
	Data any
}

type observeCursor struct {
	Timestamp time.Time
	ID        string
}

type flushWriter interface {
	io.Writer
	Flush()
}

func newHandlers(cfg handlerConfig) *Handlers {
	logger := cfg.logger
	if logger == nil {
		logger = slog.Default()
	}
	now := cfg.now
	if now == nil {
		now = func() time.Time {
			return time.Now().UTC()
		}
	}
	agentLoader := cfg.agentLoader
	if agentLoader == nil {
		agentLoader = aghconfig.LoadAgentDef
	}
	if cfg.pollInterval <= 0 {
		cfg.pollInterval = defaultPollInterval
	}
	if cfg.startedAt.IsZero() {
		cfg.startedAt = now()
	}

	return &Handlers{
		sessions:     cfg.sessions,
		observer:     cfg.observer,
		homePaths:    cfg.homePaths,
		config:       cfg.config,
		logger:       logger,
		startedAt:    cfg.startedAt,
		now:          now,
		pollInterval: cfg.pollInterval,
		agentLoader:  agentLoader,
	}
}

func (h *Handlers) setStreamDone(done <-chan struct{}) {
	h.streamDone = done
}

func (h *Handlers) listSessions(c *gin.Context) {
	infos, err := h.sessions.ListAll(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}

	payload := make([]sessionPayload, 0, len(infos))
	for _, info := range infos {
		if info == nil {
			continue
		}
		payload = append(payload, sessionPayloadFromInfo(info))
	}

	c.JSON(http.StatusOK, gin.H{"sessions": payload})
}

func (h *Handlers) createSession(c *gin.Context) {
	var req createSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, fmt.Errorf("udsapi: decode create session request: %w", err))
		return
	}
	if strings.TrimSpace(req.AgentName) == "" {
		respondError(c, http.StatusBadRequest, errors.New("agent_name is required"))
		return
	}

	sess, err := h.sessions.Create(c.Request.Context(), session.CreateOpts{
		AgentName: req.AgentName,
		Name:      req.Name,
		Workspace: req.Workspace,
	})
	if err != nil {
		respondError(c, statusForSessionError(err), err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"session": sessionPayloadFromInfo(sess.Info())})
}

func (h *Handlers) getSession(c *gin.Context) {
	info, err := h.sessions.Status(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, statusForSessionError(err), err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"session": sessionPayloadFromInfo(info)})
}

func (h *Handlers) stopSession(c *gin.Context) {
	if err := h.sessions.Stop(c.Request.Context(), c.Param("id")); err != nil {
		respondError(c, statusForSessionError(err), err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "stopped"})
}

func (h *Handlers) resumeSession(c *gin.Context) {
	sess, err := h.sessions.Resume(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, statusForSessionError(err), err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"session": sessionPayloadFromInfo(sess.Info())})
}

func (h *Handlers) promptSession(c *gin.Context) {
	var req promptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, fmt.Errorf("udsapi: decode prompt request: %w", err))
		return
	}
	if strings.TrimSpace(req.Message) == "" {
		respondError(c, http.StatusBadRequest, errors.New("message is required"))
		return
	}

	events, err := h.sessions.Prompt(c.Request.Context(), c.Param("id"), req.Message)
	if err != nil {
		respondError(c, statusForSessionError(err), err)
		return
	}

	writer, err := prepareSSE(c)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-h.streamDone:
			return
		case event, ok := <-events:
			if !ok {
				return
			}
			if err := writeSSE(writer, sseMessage{
				Name: event.Type,
				Data: agentEventPayloadFromEvent(event),
			}); err != nil {
				return
			}
		}
	}
}

func (h *Handlers) sessionEvents(c *gin.Context) {
	query, err := parseSessionEventQuery(c)
	if err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}

	events, err := h.sessions.Events(c.Request.Context(), c.Param("id"), query)
	if err != nil {
		respondError(c, statusForSessionError(err), err)
		return
	}

	payload := make([]sessionEventPayload, 0, len(events))
	for _, event := range events {
		payload = append(payload, sessionEventPayloadFromEvent(event))
	}

	c.JSON(http.StatusOK, gin.H{"events": payload})
}

func (h *Handlers) sessionHistory(c *gin.Context) {
	query, err := parseSessionEventQuery(c)
	if err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}

	history, err := h.sessions.History(c.Request.Context(), c.Param("id"), query)
	if err != nil {
		respondError(c, statusForSessionError(err), err)
		return
	}

	payload := make([]turnHistoryPayload, 0, len(history))
	for _, turn := range history {
		events := make([]sessionEventPayload, 0, len(turn.Events))
		for _, event := range turn.Events {
			events = append(events, sessionEventPayloadFromEvent(event))
		}
		payload = append(payload, turnHistoryPayload{
			TurnID: turn.TurnID,
			Events: events,
		})
	}

	c.JSON(http.StatusOK, gin.H{"history": payload})
}

func (h *Handlers) streamSession(c *gin.Context) {
	if _, err := h.sessions.Status(c.Request.Context(), c.Param("id")); err != nil {
		respondError(c, statusForSessionError(err), err)
		return
	}

	query, err := parseSessionEventQuery(c)
	if err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}
	if lastEventID := strings.TrimSpace(c.GetHeader("Last-Event-ID")); lastEventID != "" {
		after, parseErr := strconv.ParseInt(lastEventID, 10, 64)
		if parseErr != nil {
			respondError(c, http.StatusBadRequest, fmt.Errorf("udsapi: invalid Last-Event-ID %q: %w", lastEventID, parseErr))
			return
		}
		query.AfterSequence = after
	}

	initial, err := h.sessions.Events(c.Request.Context(), c.Param("id"), query)
	if err != nil {
		respondError(c, statusForSessionError(err), err)
		return
	}

	writer, err := prepareSSE(c)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}

	afterSequence := query.AfterSequence
	for _, event := range initial {
		afterSequence = event.Sequence
		if err := writeSSE(writer, sseMessage{
			ID:   strconv.FormatInt(event.Sequence, 10),
			Name: event.Type,
			Data: sessionEventPayloadFromEvent(event),
		}); err != nil {
			return
		}
	}

	pollQuery := query
	pollQuery.Limit = 0
	pollQuery.AfterSequence = afterSequence

	ticker := time.NewTicker(h.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-h.streamDone:
			return
		case <-ticker.C:
			pollQuery.AfterSequence = afterSequence
			events, err := h.sessions.Events(c.Request.Context(), c.Param("id"), pollQuery)
			if err != nil {
				_ = writeSSE(writer, sseMessage{
					Name: "error",
					Data: errorPayload{Error: err.Error()},
				})
				return
			}
			for _, event := range events {
				afterSequence = event.Sequence
				if err := writeSSE(writer, sseMessage{
					ID:   strconv.FormatInt(event.Sequence, 10),
					Name: event.Type,
					Data: sessionEventPayloadFromEvent(event),
				}); err != nil {
					return
				}
			}
			if len(events) == 0 {
				info, err := h.sessions.Status(c.Request.Context(), c.Param("id"))
				if err != nil {
					_ = writeSSE(writer, sseMessage{
						Name: "error",
						Data: errorPayload{Error: err.Error()},
					})
					return
				}
				if info != nil && info.State == session.StateStopped {
					_ = writeSSE(writer, sseMessage{
						Name: session.EventTypeSessionStopped,
						Data: sessionEventPayload{
							SessionID: info.ID,
							Type:      session.EventTypeSessionStopped,
							Timestamp: info.UpdatedAt,
						},
					})
					return
				}
			}
		}
	}
}

func (h *Handlers) approveSession(c *gin.Context) {
	respondError(c, http.StatusNotImplemented, errors.New("interactive permission approval is not implemented"))
}

func (h *Handlers) listAgents(c *gin.Context) {
	entries, err := os.ReadDir(h.homePaths.AgentsDir)
	switch {
	case err == nil:
	case errors.Is(err, os.ErrNotExist):
		c.JSON(http.StatusOK, gin.H{"agents": []agentPayload{}})
		return
	default:
		respondError(c, http.StatusInternalServerError, fmt.Errorf("udsapi: read agents directory %q: %w", h.homePaths.AgentsDir, err))
		return
	}

	agents := make([]agentPayload, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := strings.TrimSpace(entry.Name())
		if name == "" {
			continue
		}

		agent, err := h.agentLoader(name, h.homePaths)
		if err != nil {
			h.logger.Warn("udsapi: skip unreadable agent definition", "agent_name", name, "error", err)
			continue
		}
		agents = append(agents, agentPayloadFromDef(agent))
	}

	sort.Slice(agents, func(i, j int) bool {
		return agents[i].Name < agents[j].Name
	})
	c.JSON(http.StatusOK, gin.H{"agents": agents})
}

func (h *Handlers) getAgent(c *gin.Context) {
	agent, err := h.agentLoader(c.Param("name"), h.homePaths)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, os.ErrNotExist) {
			status = http.StatusNotFound
		}
		respondError(c, status, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"agent": agentPayloadFromDef(agent)})
}

func (h *Handlers) observeEvents(c *gin.Context) {
	query, err := parseObserveEventQuery(c)
	if err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}

	events, err := h.observer.QueryEvents(c.Request.Context(), query)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}

	payload := make([]observeEventPayload, 0, len(events))
	for _, event := range events {
		payload = append(payload, observeEventPayloadFromEvent(event))
	}

	c.JSON(http.StatusOK, gin.H{"events": payload})
}

func (h *Handlers) streamObserveEvents(c *gin.Context) {
	query, err := parseObserveEventQuery(c)
	if err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}

	cursor, err := parseObserveCursor(c.GetHeader("Last-Event-ID"))
	if err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}
	if !cursor.Timestamp.IsZero() {
		query.Since = cursor.Timestamp
	}

	initial, err := h.observer.QueryEvents(c.Request.Context(), query)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}

	writer, err := prepareSSE(c)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}

	cursor = emitObserveEvents(writer, initial, cursor)

	pollQuery := query
	pollQuery.Limit = 0
	if !cursor.Timestamp.IsZero() {
		pollQuery.Since = cursor.Timestamp
	}

	ticker := time.NewTicker(h.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-h.streamDone:
			return
		case <-ticker.C:
			if !cursor.Timestamp.IsZero() {
				pollQuery.Since = cursor.Timestamp
			}
			events, err := h.observer.QueryEvents(c.Request.Context(), pollQuery)
			if err != nil {
				_ = writeSSE(writer, sseMessage{
					Name: "error",
					Data: errorPayload{Error: err.Error()},
				})
				return
			}
			cursor = emitObserveEvents(writer, events, cursor)
		}
	}
}

func (h *Handlers) health(c *gin.Context) {
	health, err := h.observer.Health(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"health": health})
}

func (h *Handlers) daemonStatus(c *gin.Context) {
	health, err := h.observer.Health(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	sessions, err := h.sessions.ListAll(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"daemon": daemonStatusPayload{
			Status:         "running",
			PID:            os.Getpid(),
			StartedAt:      h.startedAt,
			Socket:         h.config.Daemon.Socket,
			HTTPHost:       h.config.HTTP.Host,
			HTTPPort:       h.config.HTTP.Port,
			ActiveSessions: health.ActiveSessions,
			TotalSessions:  len(sessions),
			Version:        health.Version,
		},
	})
}

func parseSessionEventQuery(c *gin.Context) (store.EventQuery, error) {
	since, err := parseOptionalTime(c.Query("since"))
	if err != nil {
		return store.EventQuery{}, err
	}
	limit, err := parseOptionalInt(c.Query("limit"))
	if err != nil {
		return store.EventQuery{}, err
	}
	afterSequence, err := parseOptionalInt64(c.Query("after_sequence"))
	if err != nil {
		return store.EventQuery{}, err
	}

	return store.EventQuery{
		Type:          strings.TrimSpace(c.Query("type")),
		AgentName:     strings.TrimSpace(c.Query("agent_name")),
		TurnID:        strings.TrimSpace(c.Query("turn_id")),
		Since:         since,
		Limit:         limit,
		AfterSequence: afterSequence,
	}, nil
}

func parseObserveEventQuery(c *gin.Context) (observe.EventQuery, error) {
	since, err := parseOptionalTime(c.Query("since"))
	if err != nil {
		return observe.EventQuery{}, err
	}
	limit, err := parseOptionalInt(c.Query("limit"))
	if err != nil {
		return observe.EventQuery{}, err
	}

	return observe.EventQuery{
		SessionID: strings.TrimSpace(c.Query("session_id")),
		AgentName: strings.TrimSpace(c.Query("agent_name")),
		Type:      strings.TrimSpace(c.Query("type")),
		Since:     since,
		Limit:     limit,
	}, nil
}

func parseOptionalTime(raw string) (time.Time, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}, nil
	}

	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err == nil {
		return parsed.UTC(), nil
	}
	parsed, err = time.Parse(time.RFC3339, value)
	if err == nil {
		return parsed.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("udsapi: invalid time %q", value)
}

func parseOptionalInt(raw string) (int, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("udsapi: invalid integer %q: %w", value, err)
	}
	return parsed, nil
}

func parseOptionalInt64(raw string) (int64, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, nil
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("udsapi: invalid integer %q: %w", value, err)
	}
	return parsed, nil
}

func parseObserveCursor(raw string) (observeCursor, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return observeCursor{}, nil
	}

	parts := strings.SplitN(value, "|", 2)
	if len(parts) != 2 {
		return observeCursor{}, fmt.Errorf("udsapi: invalid Last-Event-ID %q", value)
	}

	timestamp, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return observeCursor{}, fmt.Errorf("udsapi: invalid Last-Event-ID timestamp %q: %w", parts[0], err)
	}

	return observeCursor{
		Timestamp: timestamp.UTC(),
		ID:        parts[1],
	}, nil
}

func emitObserveEvents(writer flushWriter, events []observe.Event, cursor observeCursor) observeCursor {
	next := cursor
	for _, event := range events {
		if !observeEventAfterCursor(event, next) {
			continue
		}
		next = observeCursor{
			Timestamp: event.Timestamp.UTC(),
			ID:        event.ID,
		}
		if err := writeSSE(writer, sseMessage{
			ID:   observeEventID(event),
			Name: event.Type,
			Data: observeEventPayloadFromEvent(event),
		}); err != nil {
			return next
		}
	}
	return next
}

func observeEventAfterCursor(event observe.Event, cursor observeCursor) bool {
	if cursor.Timestamp.IsZero() && strings.TrimSpace(cursor.ID) == "" {
		return true
	}

	timestamp := event.Timestamp.UTC()
	switch {
	case timestamp.After(cursor.Timestamp):
		return true
	case timestamp.Before(cursor.Timestamp):
		return false
	default:
		return event.ID > cursor.ID
	}
}

func observeEventID(event observe.Event) string {
	return event.Timestamp.UTC().Format(time.RFC3339Nano) + "|" + event.ID
}

func prepareSSE(c *gin.Context) (flushWriter, error) {
	writer, ok := c.Writer.(flushWriter)
	if !ok {
		return nil, errors.New("udsapi: response writer does not support flushing")
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)
	c.Writer.WriteHeaderNow()
	writer.Flush()

	return writer, nil
}

func writeSSE(writer flushWriter, msg sseMessage) error {
	if writer == nil {
		return errors.New("udsapi: sse writer is required")
	}

	payload, err := json.Marshal(msg.Data)
	if err != nil {
		return fmt.Errorf("udsapi: marshal sse payload: %w", err)
	}
	if len(payload) == 0 {
		payload = []byte("null")
	}

	if msg.ID != "" {
		if _, err := io.WriteString(writer, "id: "+msg.ID+"\n"); err != nil {
			return err
		}
	}
	if msg.Name != "" {
		if _, err := io.WriteString(writer, "event: "+msg.Name+"\n"); err != nil {
			return err
		}
	}
	if _, err := writer.Write([]byte("data: ")); err != nil {
		return err
	}
	if _, err := writer.Write(payload); err != nil {
		return err
	}
	if _, err := io.WriteString(writer, "\n\n"); err != nil {
		return err
	}
	writer.Flush()
	return nil
}

func respondError(c *gin.Context, status int, err error) {
	message := "unknown error"
	if err != nil {
		message = err.Error()
	}
	c.JSON(status, errorPayload{Error: message})
}

func statusForSessionError(err error) int {
	switch {
	case errors.Is(err, session.ErrSessionNotFound), errors.Is(err, os.ErrNotExist):
		return http.StatusNotFound
	case errors.Is(err, session.ErrMaxSessionsReached):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func sessionPayloadFromInfo(info *session.SessionInfo) sessionPayload {
	payload := sessionPayload{}
	if info == nil {
		return payload
	}

	payload = sessionPayload{
		ID:           info.ID,
		Name:         info.Name,
		AgentName:    info.AgentName,
		Workspace:    info.Workspace,
		State:        string(info.State),
		ACPSessionID: info.ACPSessionID,
		CreatedAt:    info.CreatedAt,
		UpdatedAt:    info.UpdatedAt,
	}
	if caps := acpCapsPayloadFromInfo(info.ACPCaps); caps != nil {
		payload.ACPCaps = caps
	}
	return payload
}

func acpCapsPayloadFromInfo(caps session.ACPCaps) *acpCapsPayload {
	if !caps.SupportsLoadSession && len(caps.SupportedModes) == 0 && len(caps.SupportedModels) == 0 {
		return nil
	}

	return &acpCapsPayload{
		SupportsLoadSession: caps.SupportsLoadSession,
		SupportedModes:      append([]string(nil), caps.SupportedModes...),
		SupportedModels:     append([]string(nil), caps.SupportedModels...),
	}
}

func sessionEventPayloadFromEvent(event store.SessionEvent) sessionEventPayload {
	return sessionEventPayload{
		ID:        event.ID,
		SessionID: event.SessionID,
		Sequence:  event.Sequence,
		TurnID:    event.TurnID,
		Type:      event.Type,
		AgentName: event.AgentName,
		Content:   payloadJSON(event.Content),
		Timestamp: event.Timestamp,
	}
}

func agentPayloadFromDef(agent aghconfig.AgentDef) agentPayload {
	mcpServers := make([]agentMCPServerJSON, 0, len(agent.MCPServers))
	for _, server := range agent.MCPServers {
		mcpServers = append(mcpServers, agentMCPServerJSON{
			Name:    server.Name,
			Command: server.Command,
			Args:    append([]string(nil), server.Args...),
			Env:     cloneStringMap(server.Env),
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

func agentEventPayloadFromEvent(event session.AgentEvent) agentEventPayload {
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

func tokenUsagePayloadFromUsage(usage *session.TokenUsage) *tokenUsagePayload {
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

func observeEventPayloadFromEvent(event observe.Event) observeEventPayload {
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

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}

	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
