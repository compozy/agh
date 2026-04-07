package udsapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

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
