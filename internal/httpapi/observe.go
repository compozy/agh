package httpapi

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/store"
)

type observeEventPayload struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	Type      string    `json:"type"`
	AgentName string    `json:"agent_name"`
	Summary   string    `json:"summary,omitempty"`
	Timestamp time.Time `json:"timestamp"`
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

func (h *Handlers) health(c *gin.Context) {
	health, err := h.observer.Health(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}

	memoryHealth, err := h.memoryHealth(c)
	if err != nil {
		respondError(c, statusForMemoryError(err), err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"health": health,
		"memory": memoryHealth,
	})
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
