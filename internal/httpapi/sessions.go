package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
)

type createSessionRequest struct {
	AgentName string `json:"agent_name"`
	Name      string `json:"name"`
	Workspace string `json:"workspace"`
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
		respondError(c, http.StatusBadRequest, fmt.Errorf("httpapi: decode create session request: %w", err))
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

func (h *Handlers) approveSession(c *gin.Context) {
	respondError(c, http.StatusNotImplemented, errors.New("interactive permission approval is not implemented"))
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
	return time.Time{}, fmt.Errorf("httpapi: invalid time %q", value)
}

func parseOptionalInt(raw string) (int, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("httpapi: invalid integer %q: %w", value, err)
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
		return 0, fmt.Errorf("httpapi: invalid integer %q: %w", value, err)
	}
	return parsed, nil
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
