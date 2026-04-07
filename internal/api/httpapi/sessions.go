package httpapi

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/api/contract"
	core "github.com/pedronauck/agh/internal/api/core"
)

func (h *Handlers) approveSession(c *gin.Context) {
	var req contract.ApproveSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		core.RespondError(c, http.StatusBadRequest, fmt.Errorf("httpapi: decode approve session request: %w", err), true)
		return
	}

	approve := acp.ApproveRequest{
		RequestID: req.RequestID,
		TurnID:    req.TurnID,
		Decision:  req.Decision,
	}
	if err := approve.Validate(); err != nil {
		core.RespondError(c, http.StatusBadRequest, err, true)
		return
	}

	if err := h.Sessions.ApprovePermission(c.Request.Context(), c.Param("id"), approve); err != nil {
		core.RespondError(c, core.StatusForSessionError(err), err, true)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "approved"})
}
