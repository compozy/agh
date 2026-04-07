package udsapi

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	core "github.com/pedronauck/agh/internal/api/core"
)

func (h *Handlers) approveSession(c *gin.Context) {
	core.RespondError(c, http.StatusNotImplemented, errors.New("interactive permission approval is not implemented"), false)
}
