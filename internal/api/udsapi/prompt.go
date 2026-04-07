package udsapi

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	core "github.com/pedronauck/agh/internal/api/core"
)

type promptRequest struct {
	Message string `json:"message"`
}

func (h *Handlers) promptSession(c *gin.Context) {
	var req promptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		core.RespondError(c, http.StatusBadRequest, fmt.Errorf("udsapi: decode prompt request: %w", err), false)
		return
	}
	if strings.TrimSpace(req.Message) == "" {
		core.RespondError(c, http.StatusBadRequest, errors.New("message is required"), false)
		return
	}

	events, err := h.Sessions.Prompt(c.Request.Context(), c.Param("id"), req.Message)
	if err != nil {
		core.RespondError(c, core.StatusForSessionError(err), err, false)
		return
	}

	writer, err := core.PrepareSSE(c)
	if err != nil {
		core.RespondError(c, http.StatusInternalServerError, err, false)
		return
	}

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-h.StreamDoneChannel():
			return
		case event, ok := <-events:
			if !ok {
				return
			}
			if err := core.WriteSSE(writer, core.SSEMessage{
				Name: event.Type,
				Data: core.AgentEventPayloadFromEvent(event),
			}); err != nil {
				return
			}
		}
	}
}
