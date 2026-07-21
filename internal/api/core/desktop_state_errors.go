package core

import (
	"errors"
	"net/http"

	"github.com/compozy/agh/internal/api/contract"
	"github.com/compozy/agh/internal/clientstate"
	"github.com/gin-gonic/gin"
)

func (h *BaseHandlers) respondDesktopStateError(c *gin.Context, err error, key string) {
	status, code := desktopStateErrorStatus(err)
	if status >= http.StatusInternalServerError && h.Logger != nil {
		h.Logger.Error("desktop-state request failed", "error", err, "key", key)
	}
	c.JSON(status, contract.DesktopStateErrorPayload{
		Error: string(code),
		Code:  code,
		Key:   key,
	})
}

func desktopStateErrorStatus(err error) (int, contract.DesktopStateErrorCode) {
	switch {
	case errors.Is(err, clientstate.ErrNotFound):
		return http.StatusNotFound, contract.DesktopStateErrorNotFound
	case errors.Is(err, clientstate.ErrWorkspaceNotFound):
		return http.StatusNotFound, contract.DesktopStateErrorWorkspace
	case errors.Is(err, clientstate.ErrRevConflict):
		return http.StatusConflict, contract.DesktopStateErrorRevConflict
	case errors.Is(err, clientstate.ErrValueTooLarge):
		return http.StatusRequestEntityTooLarge, contract.DesktopStateErrorValueTooLarge
	case errors.Is(err, clientstate.ErrKeyQuota):
		return http.StatusUnprocessableEntity, contract.DesktopStateErrorKeyQuota
	case errors.Is(err, clientstate.ErrInvalidKey):
		return http.StatusUnprocessableEntity, contract.DesktopStateErrorInvalidKey
	case errors.Is(err, clientstate.ErrInvalidValue),
		errors.Is(err, clientstate.ErrInvalidDomain),
		errors.Is(err, clientstate.ErrEmptyApply):
		return http.StatusUnprocessableEntity, contract.DesktopStateErrorInvalidValue
	case errors.Is(err, clientstate.ErrSlowConsumer):
		return http.StatusConflict, contract.DesktopStateErrorSlowConsumer
	default:
		return http.StatusInternalServerError, contract.DesktopStateErrorInvalidValue
	}
}
