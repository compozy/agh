package udsapi

import (
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/api/contract"
	core "github.com/pedronauck/agh/internal/api/core"
	extensionpkg "github.com/pedronauck/agh/internal/extension"
)

func (h *Handlers) ListExtensions(c *gin.Context) {
	if h == nil || h.Extensions == nil {
		core.RespondError(
			c,
			http.StatusServiceUnavailable,
			errors.New("udsapi: extension service is not configured"),
			false,
		)
		return
	}

	items, err := h.Extensions.List(c.Request.Context())
	if err != nil {
		core.RespondError(c, http.StatusInternalServerError, err, false)
		return
	}
	c.JSON(http.StatusOK, contract.ExtensionsResponse{Extensions: items})
}

func (h *Handlers) InstallExtension(c *gin.Context) {
	if h == nil || h.Extensions == nil {
		core.RespondError(
			c,
			http.StatusServiceUnavailable,
			errors.New("udsapi: extension service is not configured"),
			false,
		)
		return
	}

	var req contract.InstallExtensionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		core.RespondError(c, http.StatusBadRequest, err, false)
		return
	}
	req.Path = strings.TrimSpace(req.Path)
	req.Checksum = strings.TrimSpace(req.Checksum)
	if req.Path == "" {
		core.RespondError(c, http.StatusBadRequest, errors.New("path is required"), false)
		return
	}
	if req.Checksum == "" {
		core.RespondError(c, http.StatusBadRequest, errors.New("checksum is required"), false)
		return
	}

	item, err := h.Extensions.Install(c.Request.Context(), req)
	if err != nil {
		core.RespondError(c, extensionStatusCode(err), err, false)
		return
	}
	c.JSON(http.StatusCreated, contract.ExtensionResponse{Extension: item})
}

func (h *Handlers) EnableExtension(c *gin.Context) {
	h.mutateExtensionEnabled(c, true)
}

func (h *Handlers) DisableExtension(c *gin.Context) {
	h.mutateExtensionEnabled(c, false)
}

func (h *Handlers) ExtensionStatus(c *gin.Context) {
	if h == nil || h.Extensions == nil {
		core.RespondError(
			c,
			http.StatusServiceUnavailable,
			errors.New("udsapi: extension service is not configured"),
			false,
		)
		return
	}

	name := strings.TrimSpace(c.Param("name"))
	if name == "" {
		core.RespondError(c, http.StatusBadRequest, errors.New("name is required"), false)
		return
	}

	item, err := h.Extensions.Status(c.Request.Context(), name)
	if err != nil {
		core.RespondError(c, extensionStatusCode(err), err, false)
		return
	}
	c.JSON(http.StatusOK, contract.ExtensionResponse{Extension: item})
}

func (h *Handlers) mutateExtensionEnabled(c *gin.Context, enabled bool) {
	if h == nil || h.Extensions == nil {
		core.RespondError(
			c,
			http.StatusServiceUnavailable,
			errors.New("udsapi: extension service is not configured"),
			false,
		)
		return
	}

	name := strings.TrimSpace(c.Param("name"))
	if name == "" {
		core.RespondError(c, http.StatusBadRequest, errors.New("name is required"), false)
		return
	}

	var (
		item contract.ExtensionPayload
		err  error
	)
	if enabled {
		item, err = h.Extensions.Enable(c.Request.Context(), name)
	} else {
		item, err = h.Extensions.Disable(c.Request.Context(), name)
	}
	if err != nil {
		core.RespondError(c, extensionStatusCode(err), err, false)
		return
	}
	c.JSON(http.StatusOK, contract.ExtensionResponse{Extension: item})
}

func extensionStatusCode(err error) int {
	switch {
	case err == nil:
		return http.StatusOK
	case errors.Is(err, extensionpkg.ErrExtensionNotFound):
		return http.StatusNotFound
	case errors.Is(err, extensionpkg.ErrExtensionExists):
		return http.StatusConflict
	case errors.Is(err, extensionpkg.ErrExtensionChecksumMismatch):
		return http.StatusBadRequest
	case errors.Is(err, extensionpkg.ErrManifestInvalid):
		return http.StatusBadRequest
	case errors.Is(err, extensionpkg.ErrManifestIncompatible):
		return http.StatusBadRequest
	case errors.Is(err, extensionpkg.ErrManifestNotFound):
		return http.StatusBadRequest
	case errors.Is(err, extensionpkg.ErrExtensionHasActiveBundles):
		return http.StatusConflict
	case errors.Is(err, os.ErrNotExist):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
