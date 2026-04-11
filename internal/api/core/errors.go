package core

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/api/contract"
	automationpkg "github.com/pedronauck/agh/internal/automation"
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

	c.JSON(status, contract.ErrorPayload{Error: message})
}

// StatusForSessionError maps session and workspace-domain errors to transport statuses.
func StatusForSessionError(err error) int {
	return statusForSessionError(err)
}

// StatusForWorkspaceError maps workspace-domain errors to transport statuses.
func StatusForWorkspaceError(err error) int {
	return statusForWorkspaceError(err)
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

// ErrSkillNotFound is the sentinel for a missing skill.
var ErrSkillNotFound = errors.New("skill not found")

// ErrSkillValidation is the sentinel for skill request validation failures.
var ErrSkillValidation = errors.New("skill validation error")

// ErrAutomationValidation is the sentinel for automation request validation failures.
var ErrAutomationValidation = errors.New("automation validation error")

// StatusForSkillError maps skill-domain errors to transport statuses.
func StatusForSkillError(err error) int {
	switch {
	case err == nil:
		return http.StatusOK
	case errors.Is(err, ErrSkillNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrSkillValidation):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

// NewAutomationValidationError wraps an automation validation failure with the shared sentinel.
func NewAutomationValidationError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %w", ErrAutomationValidation, err)
}

// StatusForAutomationError maps automation-domain failures to transport statuses.
func StatusForAutomationError(err error) int {
	var maxBytesErr *http.MaxBytesError
	switch {
	case err == nil:
		return http.StatusOK
	case errors.As(err, &maxBytesErr):
		return http.StatusRequestEntityTooLarge
	case errors.Is(err, ErrAutomationValidation):
		return http.StatusBadRequest
	case errors.Is(err, automationpkg.ErrJobNotFound),
		errors.Is(err, automationpkg.ErrTriggerNotFound),
		errors.Is(err, automationpkg.ErrRunNotFound),
		errors.Is(err, automationpkg.ErrWebhookTriggerNotRegistered),
		errors.Is(err, automationpkg.ErrJobOverlayNotFound),
		errors.Is(err, automationpkg.ErrTriggerOverlayNotFound):
		return http.StatusNotFound
	case errors.Is(err, automationpkg.ErrJobNameTaken),
		errors.Is(err, automationpkg.ErrTriggerNameTaken),
		errors.Is(err, automationpkg.ErrTriggerWebhookIDTaken),
		errors.Is(err, automationpkg.ErrConcurrencyLimitReached),
		errors.Is(err, automationpkg.ErrFireLimitReached),
		errors.Is(err, automationpkg.ErrOverlayRequiresConfigSource),
		errors.Is(err, automationpkg.ErrWebhookReplayDetected):
		return http.StatusConflict
	case errors.Is(err, automationpkg.ErrWebhookSignatureInvalid),
		errors.Is(err, automationpkg.ErrWebhookTimestampInvalid):
		return http.StatusUnauthorized
	case errors.Is(err, automationpkg.ErrManagerNotRunning):
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
