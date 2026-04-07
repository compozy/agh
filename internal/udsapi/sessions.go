package udsapi

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
)

func (h *Handlers) listSessions(c *gin.Context) {
	infos, err := h.sessions.ListAll(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}

	workspaceFilter := strings.TrimSpace(c.Query("workspace"))
	if workspaceFilter != "" {
		workspaceID, err := h.lookupWorkspaceID(c.Request.Context(), workspaceFilter)
		if err != nil {
			respondError(c, statusForWorkspaceError(err), err)
			return
		}
		infos = filterSessionInfosByWorkspaceID(infos, workspaceID)
	}

	c.JSON(http.StatusOK, gin.H{"sessions": sessionPayloadsFromInfos(infos)})
}

func (h *Handlers) createSession(c *gin.Context) {
	var req createSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, fmt.Errorf("udsapi: decode create session request: %w", err))
		return
	}
	if err := validateCreateSessionRequest(req); err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}

	sess, err := h.sessions.Create(c.Request.Context(), session.CreateOpts{
		AgentName:     req.AgentName,
		Name:          req.Name,
		Workspace:     strings.TrimSpace(req.Workspace),
		WorkspacePath: strings.TrimSpace(req.WorkspacePath),
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

	info, err := h.sessions.Status(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, statusForSessionError(err), err)
		return
	}

	events, err := h.sessions.Events(c.Request.Context(), c.Param("id"), query)
	if err != nil {
		respondError(c, statusForSessionError(err), err)
		return
	}

	payload := make([]sessionEventPayload, 0, len(events))
	for _, event := range events {
		payload = append(payload, sessionEventPayloadFromEvent(event, info))
	}

	c.JSON(http.StatusOK, gin.H{"events": payload})
}

func (h *Handlers) sessionHistory(c *gin.Context) {
	query, err := parseSessionEventQuery(c)
	if err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}

	info, err := h.sessions.Status(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, statusForSessionError(err), err)
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
			events = append(events, sessionEventPayloadFromEvent(event, info))
		}
		payload = append(payload, turnHistoryPayload{
			TurnID: turn.TurnID,
			Events: events,
		})
	}

	c.JSON(http.StatusOK, gin.H{"history": payload})
}

func (h *Handlers) sessionTranscript(c *gin.Context) {
	messages, err := h.sessions.Transcript(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, statusForSessionError(err), err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"messages": messages})
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
