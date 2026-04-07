package udsapi

import (
	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/apicore"
)

type agentEventPayload = contract.AgentEventPayload
type sseMessage = apicore.SSEMessage
type flushWriter = apicore.FlushWriter

func respondError(c *gin.Context, status int, err error) {
	apicore.RespondError(c, status, err, false)
}

func statusForSessionError(err error) int {
	return apicore.StatusForSessionError(err)
}

func prepareSSE(c *gin.Context) (flushWriter, error) {
	return apicore.PrepareSSE(c)
}

func writeSSE(writer flushWriter, msg sseMessage) error {
	return apicore.WriteSSE(writer, msg)
}

func agentEventPayloadFromEvent(event acp.AgentEvent) agentEventPayload {
	return apicore.AgentEventPayloadFromEvent(event)
}
