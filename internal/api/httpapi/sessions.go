package httpapi

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/acp"
)

func (h *Handlers) approveSession(c *gin.Context) {
	var req approveSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, fmt.Errorf("httpapi: decode approve session request: %w", err))
		return
	}

	approve := acp.ApproveRequest{
		RequestID: req.RequestID,
		TurnID:    req.TurnID,
		Decision:  req.Decision,
	}
	if err := approve.Validate(); err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}

	if err := h.Sessions.ApprovePermission(c.Request.Context(), c.Param("id"), approve); err != nil {
		respondError(c, statusForSessionError(err), err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "approved"})
}
