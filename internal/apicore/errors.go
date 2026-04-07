package apicore

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/apisupport"
	"github.com/pedronauck/agh/internal/memory"
)

// RespondError writes a transport error response, optionally masking internal error details.
func RespondError(c *gin.Context, status int, err error, maskInternalErrors bool) {
	message := http.StatusText(status)
	switch {
	case maskInternalErrors && status >= http.StatusInternalServerError:
		if strings.TrimSpace(message) == "" {
			message = "internal server error"
		}
	case err != nil && strings.TrimSpace(err.Error()) != "":
		message = err.Error()
	case strings.TrimSpace(message) == "":
		message = "unknown error"
	}

	c.JSON(status, ErrorPayload{Error: message})
}

// StatusForSessionError maps session and workspace-domain errors to transport statuses.
func StatusForSessionError(err error) int {
	return apisupport.StatusForSessionError(err)
}

// StatusForWorkspaceError maps workspace-domain errors to transport statuses.
func StatusForWorkspaceError(err error) int {
	return apisupport.StatusForWorkspaceError(err)
}

// NewMemoryValidationError wraps a memory validation failure with the shared sentinel.
func NewMemoryValidationError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %v", memory.ErrValidation, err)
}

// StatusForMemoryError maps memory-domain errors to transport statuses.
func StatusForMemoryError(err error) int {
	switch {
	case err == nil:
		return http.StatusOK
	case errors.Is(err, os.ErrNotExist):
		return http.StatusNotFound
	case errors.Is(err, memory.ErrValidation):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
