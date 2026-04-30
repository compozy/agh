package udsapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	core "github.com/pedronauck/agh/internal/api/core"
	mcppkg "github.com/pedronauck/agh/internal/mcp"
)

func registerHostedMCPRoutes(api gin.IRouter, handlers *Handlers) {
	hosted := api.Group("/internal/hosted-mcp")
	{
		hosted.POST("/bind", handlers.bindHostedMCP)
		hosted.GET("/projection", handlers.hostedMCPProjection)
		hosted.GET("/projection/stream", handlers.streamHostedMCPProjection)
		hosted.POST("/tools/call", handlers.callHostedMCP)
		hosted.POST("/release", handlers.releaseHostedMCP)
	}
}

func (h *Handlers) bindHostedMCP(c *gin.Context) {
	if h == nil || h.HostedMCP == nil {
		core.RespondError(c, http.StatusServiceUnavailable, mcppkg.ErrHostedDisabled, false)
		return
	}
	var req mcppkg.HostedBindRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		core.RespondError(c, http.StatusBadRequest, err, false)
		return
	}
	peer, err := mcppkg.PeerInfoFromContext(c.Request.Context())
	if err != nil {
		core.RespondError(c, http.StatusForbidden, err, false)
		return
	}
	response, err := h.HostedMCP.Bind(c.Request.Context(), req, peer)
	if err != nil {
		core.RespondError(c, hostedMCPStatus(err), err, false)
		return
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handlers) hostedMCPProjection(c *gin.Context) {
	if h == nil || h.HostedMCP == nil {
		core.RespondError(c, http.StatusServiceUnavailable, mcppkg.ErrHostedDisabled, false)
		return
	}
	peer, err := mcppkg.PeerInfoFromContext(c.Request.Context())
	if err != nil {
		core.RespondError(c, http.StatusForbidden, err, false)
		return
	}
	response, err := h.HostedMCP.Projection(c.Request.Context(), c.Query("bind_id"), peer)
	if err != nil {
		core.RespondError(c, hostedMCPStatus(err), err, false)
		return
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handlers) streamHostedMCPProjection(c *gin.Context) {
	if h == nil || h.HostedMCP == nil {
		core.RespondError(c, http.StatusServiceUnavailable, mcppkg.ErrHostedDisabled, false)
		return
	}
	peer, err := mcppkg.PeerInfoFromContext(c.Request.Context())
	if err != nil {
		core.RespondError(c, http.StatusForbidden, err, false)
		return
	}
	bindID := strings.TrimSpace(c.Query("bind_id"))
	lastDigest := strings.TrimSpace(c.Query("last_digest"))
	response, err := h.HostedMCP.Projection(c.Request.Context(), bindID, peer)
	if err != nil {
		core.RespondError(c, hostedMCPStatus(err), err, false)
		return
	}
	writer, err := core.PrepareSSE(c)
	if err != nil {
		core.RespondError(c, http.StatusInternalServerError, err, false)
		return
	}
	interval := h.PollInterval
	if interval <= 0 {
		interval = defaultPollInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if response.Digest != "" && response.Digest != lastDigest {
			if writeErr := core.WriteSSE(writer, core.SSEMessage{
				ID:   response.Digest,
				Name: "projection",
				Data: response,
			}); writeErr != nil {
				if h.Logger != nil {
					h.Logger.Warn("udsapi: failed to emit hosted MCP projection", "error", writeErr)
				}
				return
			}
			lastDigest = response.Digest
		}
		select {
		case <-c.Request.Context().Done():
			return
		case <-h.StreamDoneChannel():
			return
		case <-ticker.C:
		}
		response, err = h.HostedMCP.Projection(c.Request.Context(), bindID, peer)
		if err != nil {
			status := hostedMCPStatus(err)
			if h.Logger != nil {
				h.Logger.Warn("udsapi: hosted MCP projection failed", "status", status, "error", err)
			}
			if writeErr := core.WriteSSE(writer, core.SSEMessage{
				Name: "error",
				Data: hostedMCPStreamErrorData(err),
			}); writeErr != nil && h.Logger != nil {
				h.Logger.Warn("udsapi: failed to emit hosted MCP error", "error", writeErr)
			}
			return
		}
	}
}

// hostedMCPStreamErrorData keeps stream failures stable without exposing backend error text.
func hostedMCPStreamErrorData(err error) map[string]any {
	status := hostedMCPStatus(err)
	return map[string]any{
		"error":   "hosted_mcp_projection_failed",
		"status":  status,
		"message": http.StatusText(status),
	}
}

func (h *Handlers) callHostedMCP(c *gin.Context) {
	if h == nil || h.HostedMCP == nil {
		core.RespondError(c, http.StatusServiceUnavailable, mcppkg.ErrHostedDisabled, false)
		return
	}
	var req mcppkg.HostedCallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		core.RespondError(c, http.StatusBadRequest, err, false)
		return
	}
	peer, err := mcppkg.PeerInfoFromContext(c.Request.Context())
	if err != nil {
		core.RespondError(c, http.StatusForbidden, err, false)
		return
	}
	response, err := h.HostedMCP.Call(c.Request.Context(), req, peer)
	if err != nil {
		core.RespondError(c, hostedMCPStatus(err), err, false)
		return
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handlers) releaseHostedMCP(c *gin.Context) {
	if h == nil || h.HostedMCP == nil {
		core.RespondError(c, http.StatusServiceUnavailable, mcppkg.ErrHostedDisabled, false)
		return
	}
	var req mcppkg.HostedReleaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		core.RespondError(c, http.StatusBadRequest, err, false)
		return
	}
	peer, err := mcppkg.PeerInfoFromContext(c.Request.Context())
	if err != nil {
		core.RespondError(c, http.StatusForbidden, err, false)
		return
	}
	if err := h.HostedMCP.ReleaseBindForPeer(c.Request.Context(), req.BindID, peer); err != nil {
		core.RespondError(c, hostedMCPStatus(err), err, false)
		return
	}
	c.Status(http.StatusNoContent)
}

func hostedMCPStatus(err error) int {
	switch {
	case err == nil:
		return http.StatusOK
	case errors.Is(err, mcppkg.ErrHostedDisabled), errors.Is(err, mcppkg.ErrHostedRegistryRequired):
		return http.StatusServiceUnavailable
	case errors.Is(err, mcppkg.ErrHostedSessionRequired),
		errors.Is(err, mcppkg.ErrHostedNonceRequired),
		errors.Is(err, mcppkg.ErrHostedBindRequired):
		return http.StatusBadRequest
	case errors.Is(err, mcppkg.ErrHostedNonceInvalid),
		errors.Is(err, mcppkg.ErrHostedNonceExpired),
		errors.Is(err, mcppkg.ErrHostedPeerInvalid),
		errors.Is(err, mcppkg.ErrHostedBinaryInvalid),
		errors.Is(err, mcppkg.ErrHostedBindNotFound):
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}
