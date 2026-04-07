package httpapi

import (
	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/apicore"
)

type approveSessionRequest = apicore.ApproveSessionRequest
type errorPayload = apicore.ErrorPayload
type sseMessage = apicore.SSEMessage
type flushWriter = apicore.FlushWriter

func respondError(c *gin.Context, status int, err error) {
	apicore.RespondError(c, status, err, true)
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

func writeSSERaw(writer flushWriter, id string, raw string, names ...string) error {
	return apicore.WriteSSERaw(writer, id, raw, names...)
}
