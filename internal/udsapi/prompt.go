package udsapi

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type promptRequest struct {
	Message string `json:"message"`
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
