package udsapi

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handlers) approveSession(c *gin.Context) {
	respondError(c, http.StatusNotImplemented, errors.New("interactive permission approval is not implemented"))
}
