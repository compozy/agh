package udsapi

import (
	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/api/contract"
	core "github.com/pedronauck/agh/internal/api/core"
)

type agentEventPayload = contract.AgentEventPayload
type sseMessage = core.SSEMessage
type flushWriter = core.FlushWriter

func respondError(c *gin.Context, status int, err error) {
	core.RespondError(c, status, err, false)
}

func statusForSessionError(err error) int {
	return core.StatusForSessionError(err)
}

func prepareSSE(c *gin.Context) (flushWriter, error) {
	return core.PrepareSSE(c)
}

func writeSSE(writer flushWriter, msg sseMessage) error {
	return core.WriteSSE(writer, msg)
}

func agentEventPayloadFromEvent(event acp.AgentEvent) agentEventPayload {
	return core.AgentEventPayloadFromEvent(event)
}
