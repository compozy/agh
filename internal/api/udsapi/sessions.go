package udsapi

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
		core.RespondError(
			c,
			http.StatusBadRequest,
			fmt.Errorf("udsapi: decode approve session request: %w", err),
			false,
		)
		return
	}

	approve := acp.ApproveRequest{
		RequestID: req.RequestID,
		TurnID:    req.TurnID,
		Decision:  req.Decision,
	}
	if err := approve.Validate(); err != nil {
		core.RespondError(c, http.StatusBadRequest, err, false)
		return
	}

	if err := h.Sessions.ApprovePermission(c.Request.Context(), c.Param("id"), approve); err != nil {
		core.RespondError(c, core.StatusForSessionError(err), err, false)
		return
	}

	c.JSON(http.StatusOK, contract.SessionApprovalResponse{Status: "approved"})
}

func (h *Handlers) cancelSessionPrompt(c *gin.Context) {
	if err := h.Sessions.CancelPrompt(c.Request.Context(), c.Param("id")); err != nil {
		core.RespondError(c, core.StatusForSessionError(err), err, false)
		return
	}

	c.Status(http.StatusOK)
}
