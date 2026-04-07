package httpapi

import (
	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/api/contract"
	core "github.com/pedronauck/agh/internal/api/core"
)

type approveSessionRequest = contract.ApproveSessionRequest
type errorPayload = contract.ErrorPayload
type sseMessage = core.SSEMessage
type flushWriter = core.FlushWriter

func respondError(c *gin.Context, status int, err error) {
	core.RespondError(c, status, err, true)
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

func writeSSERaw(writer flushWriter, id string, raw string, names ...string) error {
	return core.WriteSSERaw(writer, id, raw, names...)
}
