package core

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/agentidentity"
	"github.com/pedronauck/agh/internal/api/contract"
	automationpkg "github.com/pedronauck/agh/internal/automation"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/modelcatalog"
	"github.com/pedronauck/agh/internal/network"
	"github.com/pedronauck/agh/internal/resources"
	"github.com/pedronauck/agh/internal/session"
	settingspkg "github.com/pedronauck/agh/internal/settings"
	skillmarketplace "github.com/pedronauck/agh/internal/skills/marketplace"
	"github.com/pedronauck/agh/internal/store"
	taskpkg "github.com/pedronauck/agh/internal/task"
	toolspkg "github.com/pedronauck/agh/internal/tools"
	"github.com/pedronauck/agh/internal/vault"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

const (
	errorsInternalServerErrorValue = "internal server error"
	errorsUnknownErrorValue        = "unknown error"
)

// ErrRequestBodyTooLarge is the shared transport sentinel for request bodies
// rejected by HTTP MaxBytesReader enforcement.
var ErrRequestBodyTooLarge = errors.New("request body too large")

// RespondError writes a transport error response, optionally masking internal error details.
func RespondError(c *gin.Context, status int, err error, maskInternalErrors bool) {
	if maxBytesErr, ok := errors.AsType[*http.MaxBytesError](err); ok && maxBytesErr != nil {
		status = http.StatusRequestEntityTooLarge
		err = ErrRequestBodyTooLarge
		maskInternalErrors = false
	}

	message := http.StatusText(status)
	switch {
	case maskInternalErrors && status >= http.StatusInternalServerError:
		if strings.TrimSpace(message) == "" {
			message = errorsInternalServerErrorValue
		}
	case err != nil && strings.TrimSpace(err.Error()) != "":
		message = err.Error()
	case strings.TrimSpace(message) == "":
		message = errorsUnknownErrorValue
	}

	message = taskpkg.RedactClaimTokens(message)
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
	return fmt.Errorf("%w: %w", memory.ErrValidation, err)
}

// ErrSettingsValidation is the sentinel for settings request validation failures.
var ErrSettingsValidation = errors.New("settings validation error")

// ErrSettingsNotFound is the sentinel for missing settings resources.
var ErrSettingsNotFound = errors.New("settings not found")

// ErrSettingsConflict is the sentinel for conflicting settings mutations or scope combinations.
var ErrSettingsConflict = errors.New("settings conflict")

// ErrSettingsForbidden is the sentinel for settings operations rejected by transport policy.
var ErrSettingsForbidden = errors.New("settings forbidden")

// NewSettingsValidationError wraps a settings validation failure with the shared sentinel.
func NewSettingsValidationError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %w", ErrSettingsValidation, err)
}

// NewSettingsNotFoundError wraps a missing settings resource failure with the shared sentinel.
func NewSettingsNotFoundError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %w", ErrSettingsNotFound, err)
}

// NewSettingsConflictError wraps a settings conflict with the shared sentinel.
func NewSettingsConflictError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %w", ErrSettingsConflict, err)
}

// NewSettingsForbiddenError wraps a settings forbidden failure with the shared sentinel.
func NewSettingsForbiddenError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %w", ErrSettingsForbidden, err)
}

// StatusForSettingsError maps settings-domain failures to transport statuses.
func StatusForSettingsError(err error) int {
	switch {
	case err == nil:
		return http.StatusOK
	case errors.Is(err, ErrSettingsForbidden),
		errors.Is(err, settingspkg.ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, ErrSettingsValidation),
		errors.Is(err, settingspkg.ErrValidation):
		return http.StatusBadRequest
	case errors.Is(err, settingspkg.ErrUnprocessable):
		return http.StatusUnprocessableEntity
	case errors.Is(err, ErrSettingsNotFound),
		errors.Is(err, settingspkg.ErrNotFound),
		errors.Is(err, workspacepkg.ErrWorkspaceNotFound),
		errors.Is(err, workspacepkg.ErrWorkspaceRootMissing),
		errors.Is(err, os.ErrNotExist):
		return http.StatusNotFound
	case errors.Is(err, ErrSettingsConflict),
		errors.Is(err, settingspkg.ErrConflict),
		errors.Is(err, aghconfig.ErrUnsupportedTOMLMutation):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

// StatusForVaultError maps vault-domain failures to transport statuses.
func StatusForVaultError(err error) int {
	switch {
	case err == nil:
		return http.StatusOK
	case errors.Is(err, vault.ErrSecretNotFound):
		return http.StatusNotFound
	case errors.Is(err, vault.ErrUnsupportedSecretRef),
		errors.Is(err, vault.ErrMissingSecret):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

// StatusForMemoryError maps memory-domain errors to transport statuses.
func StatusForMemoryError(err error) int {
	switch {
	case err == nil:
		return http.StatusOK
	case errors.Is(err, ErrMemoryUnsupported):
		return http.StatusNotImplemented
	case errors.Is(err, ErrMemoryRejected):
		return http.StatusUnprocessableEntity
	case errors.Is(err, os.ErrNotExist):
		return http.StatusNotFound
	case errors.Is(err, memory.ErrValidation):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

// StatusForResourceError maps desired-state resource failures to transport statuses.
func StatusForResourceError(err error) int {
	switch {
	case err == nil:
		return http.StatusOK
	case errors.Is(err, resources.ErrPermissionDenied),
		errors.Is(err, resources.ErrDirectMutationNotAllowed):
		return http.StatusForbidden
	case errors.Is(err, resources.ErrConflict),
		errors.Is(err, resources.ErrSessionNotActive),
		errors.Is(err, resources.ErrStaleSourceVersion):
		return http.StatusConflict
	case errors.Is(err, resources.ErrPayloadTooLarge):
		return http.StatusRequestEntityTooLarge
	case errors.Is(err, resources.ErrRateLimited):
		return http.StatusTooManyRequests
	case errors.Is(err, resources.ErrNotFound), errors.Is(err, os.ErrNotExist):
		return http.StatusNotFound
	case errors.Is(err, resources.ErrValidation),
		errors.Is(err, resources.ErrInvalidScopeBinding),
		errors.Is(err, resources.ErrCodecNotFound),
		errors.Is(err, resources.ErrCodecTypeMismatch):
		return http.StatusUnprocessableEntity
	default:
		return http.StatusInternalServerError
	}
}

// StatusForToolError maps registry failures to stable transport statuses.
func StatusForToolError(err error) int {
	var toolErr *toolspkg.ToolError
	var validation *toolspkg.ValidationError
	switch {
	case err == nil:
		return http.StatusOK
	case errors.As(err, &validation):
		return http.StatusBadRequest
	case errors.As(err, &toolErr):
		return statusForToolCode(toolErr.Code, toolErr.ReasonCodes)
	case errors.Is(err, toolspkg.ErrToolInvalidInput):
		return http.StatusBadRequest
	case errors.Is(err, toolspkg.ErrToolNotFound):
		return http.StatusNotFound
	case errors.Is(err, toolspkg.ErrToolDenied):
		return http.StatusForbidden
	case errors.Is(err, toolspkg.ErrToolApprovalRequired):
		return http.StatusAccepted
	case errors.Is(err, toolspkg.ErrToolConflict):
		return http.StatusConflict
	case errors.Is(err, toolspkg.ErrToolUnavailable),
		errors.Is(err, toolspkg.ErrToolResultTooLarge):
		return http.StatusUnprocessableEntity
	case errors.Is(err, toolspkg.ErrToolBackendFailed),
		errors.Is(err, toolspkg.ErrToolCanceled),
		errors.Is(err, toolspkg.ErrToolTimedOut):
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}

func statusForToolCode(code toolspkg.ErrorCode, reasons []toolspkg.ReasonCode) int {
	switch code {
	case toolspkg.ErrorCodeInvalidInput:
		return http.StatusBadRequest
	case toolspkg.ErrorCodeNotFound:
		return http.StatusNotFound
	case toolspkg.ErrorCodeDenied:
		return http.StatusForbidden
	case toolspkg.ErrorCodeApprovalRequired:
		if hasToolReason(reasons, toolspkg.ReasonApprovalTokenExpired, toolspkg.ReasonApprovalTokenMismatch,
			toolspkg.ReasonApprovalTokenReplayed) {
			return http.StatusForbidden
		}
		return http.StatusAccepted
	case toolspkg.ErrorCodeConflict:
		return http.StatusConflict
	case toolspkg.ErrorCodeUnavailable, toolspkg.ErrorCodeResultTooLarge:
		return http.StatusUnprocessableEntity
	case toolspkg.ErrorCodeBackendFailed, toolspkg.ErrorCodeCanceled, toolspkg.ErrorCodeTimedOut:
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}

func hasToolReason(reasons []toolspkg.ReasonCode, want ...toolspkg.ReasonCode) bool {
	for _, reason := range reasons {
		if slices.Contains(want, reason) {
			return true
		}
	}
	return false
}

// NewTaskValidationError wraps a task validation failure with the shared sentinel.
func NewTaskValidationError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %w", taskpkg.ErrValidation, err)
}

// StatusForTaskError maps task-domain, workspace, and session failures to transport statuses.
func StatusForTaskError(err error) int {
	switch {
	case err == nil:
		return http.StatusOK
	case errors.Is(err, taskpkg.ErrValidation),
		errors.Is(err, taskpkg.ErrInvalidScopeBinding),
		errors.Is(err, taskpkg.ErrImmutableField):
		return http.StatusBadRequest
	case errors.Is(err, taskpkg.ErrPayloadTooLarge):
		return http.StatusRequestEntityTooLarge
	case errors.Is(err, taskpkg.ErrPermissionDenied):
		return http.StatusForbidden
	case errors.Is(err, errAgentIdentityUnavailable),
		errors.Is(err, agentidentity.ErrIdentityLookupUnavailable):
		return http.StatusServiceUnavailable
	case errors.Is(err, agentidentity.ErrIdentityUnauthorized):
		return http.StatusForbidden
	case errors.Is(err, agentidentity.ErrIdentityRequired),
		errors.Is(err, agentidentity.ErrIdentityMismatch),
		errors.Is(err, agentidentity.ErrIdentityStale):
		return http.StatusUnauthorized
	case errors.Is(err, taskpkg.ErrTaskNotFound),
		errors.Is(err, taskpkg.ErrTaskRunNotFound),
		errors.Is(err, taskpkg.ErrTaskDependencyNotFound),
		errors.Is(err, taskpkg.ErrTaskEventNotFound),
		errors.Is(err, taskpkg.ErrTaskRunIdempotencyNotFound),
		errors.Is(err, taskpkg.ErrExecutionProfileNotFound),
		errors.Is(err, taskpkg.ErrRunReviewNotFound),
		errors.Is(err, workspacepkg.ErrWorkspaceNotFound),
		errors.Is(err, session.ErrSessionNotFound),
		errors.Is(err, os.ErrNotExist):
		return http.StatusNotFound
	case errors.Is(err, workspacepkg.ErrWorkspaceRootMissing):
		return http.StatusGone
	case errors.Is(err, taskpkg.ErrInvalidStatusTransition),
		errors.Is(err, taskpkg.ErrGraphLimitExceeded),
		errors.Is(err, taskpkg.ErrCycleDetected),
		errors.Is(err, taskpkg.ErrSessionAlreadyBound),
		errors.Is(err, taskpkg.ErrSessionAttachNotAllowed),
		errors.Is(err, taskpkg.ErrStaleNetworkChannel),
		errors.Is(err, taskpkg.ErrNoClaimableRun),
		errors.Is(err, taskpkg.ErrInvalidClaimToken),
		errors.Is(err, taskpkg.ErrLeaseExpired),
		errors.Is(err, taskpkg.ErrActiveRunLease):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

// StatusForBridgeError maps bridge-domain and workspace-domain errors to transport statuses.
func StatusForBridgeError(err error) int {
	switch {
	case err == nil:
		return http.StatusOK
	case errors.Is(err, contract.ErrBridgeInstanceMismatch):
		return http.StatusBadRequest
	case errors.Is(err, bridgepkg.ErrInvalidBridgeSecretBinding):
		return http.StatusBadRequest
	case errors.Is(err, bridgepkg.ErrInvalidBridgeTaskSubscription):
		return http.StatusBadRequest
	case errors.Is(err, bridgepkg.ErrBridgeInstanceNotFound):
		return http.StatusNotFound
	case errors.Is(err, bridgepkg.ErrBridgeSecretBindingNotFound):
		return http.StatusNotFound
	case errors.Is(err, bridgepkg.ErrBridgeTaskSubscriptionNotFound):
		return http.StatusNotFound
	case errors.Is(err, bridgepkg.ErrBridgeRouteNotFound):
		return http.StatusNotFound
	case errors.Is(err, bridgepkg.ErrBridgeInstanceUnavailable):
		return http.StatusConflict
	case errors.Is(err, bridgepkg.ErrInvalidBridgeStateTransition):
		return http.StatusConflict
	case errors.Is(err, bridgepkg.ErrBridgeInstanceReadOnly):
		return http.StatusConflict
	case errors.Is(err, bridgepkg.ErrDeliveryNotFound):
		return http.StatusNotFound
	case errors.Is(err, bridgepkg.ErrDeliveryQueueSaturated):
		return http.StatusServiceUnavailable
	case errors.Is(err, bridgepkg.ErrDeliveryTransportUnavailable):
		return http.StatusServiceUnavailable
	case errors.Is(err, workspacepkg.ErrWorkspaceNotFound):
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}

// ErrSkillNotFound is the sentinel for a missing skill.
var ErrSkillNotFound = errors.New("skill not found")

// ErrSkillValidation is the sentinel for skill request validation failures.
var ErrSkillValidation = errors.New("skill validation error")

// ErrSkillUnprocessable is the sentinel for semantically invalid skill layers.
var ErrSkillUnprocessable = errors.New("skill unprocessable")

// ErrAutomationValidation is the sentinel for automation request validation failures.
var ErrAutomationValidation = errors.New("automation validation error")

// ErrNetworkValidation is the sentinel for malformed network control-plane requests.
var ErrNetworkValidation = errors.New("network validation error")

// ErrModelCatalogValidation is the sentinel for malformed model catalog requests.
var ErrModelCatalogValidation = errors.New("model catalog validation error")

// ErrModelCatalogUnavailable reports that the daemon model catalog surface is not configured.
var ErrModelCatalogUnavailable = errors.New("model catalog service unavailable")

// StatusForSkillError maps skill-domain errors to transport statuses.
func StatusForSkillError(err error) int {
	switch {
	case err == nil:
		return http.StatusOK
	case errors.Is(err, ErrSkillNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrSkillValidation):
		return http.StatusBadRequest
	case errors.Is(err, ErrSkillUnprocessable):
		return http.StatusUnprocessableEntity
	default:
		return http.StatusInternalServerError
	}
}

// StatusForSkillMarketplaceError maps skill marketplace lifecycle failures to transport statuses.
func StatusForSkillMarketplaceError(err error) int {
	switch {
	case err == nil:
		return http.StatusOK
	case errors.Is(err, skillmarketplace.ErrValidation):
		return http.StatusBadRequest
	case errors.Is(err, skillmarketplace.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, skillmarketplace.ErrNotMarketplace):
		return http.StatusUnprocessableEntity
	case errors.Is(err, skillmarketplace.ErrNotConfigured):
		return http.StatusServiceUnavailable
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
	case errors.Is(err, automationpkg.ErrWebhookSecretRequired):
		return http.StatusBadRequest
	case errors.Is(err, automationpkg.ErrWebhookEndpointInvalid):
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
		errors.Is(err, automationpkg.ErrDefinitionReadOnly),
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

// NewNetworkValidationError wraps a network request validation failure with the shared sentinel.
func NewNetworkValidationError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %w", ErrNetworkValidation, err)
}

// StatusForNetworkError maps network-domain errors to transport statuses.
func StatusForNetworkError(err error) int {
	switch {
	case err == nil:
		return http.StatusOK
	case errors.Is(err, ErrNetworkValidation):
		return http.StatusBadRequest
	case errors.Is(err, network.ErrLocalPeerNotFound),
		errors.Is(err, network.ErrTargetPeerNotFound),
		errors.Is(err, store.ErrNetworkConversationNotFound),
		errors.Is(err, sql.ErrNoRows):
		return http.StatusNotFound
	case errors.Is(err, store.ErrNetworkDirectRoomCollision),
		errors.Is(err, store.ErrNetworkWorkContainerMismatch),
		errors.Is(err, store.ErrNetworkWorkClosed):
		return http.StatusConflict
	case errors.Is(err, network.ErrMissingField),
		errors.Is(err, network.ErrInvalidField),
		errors.Is(err, network.ErrInvalidKind),
		errors.Is(err, network.ErrInvalidBody),
		errors.Is(err, network.ErrExpired),
		errors.Is(err, network.ErrReplayTooOld),
		errors.Is(err, network.ErrLegacyFieldRejected):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

// NewModelCatalogValidationError wraps a model catalog request validation failure.
func NewModelCatalogValidationError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %w", ErrModelCatalogValidation, err)
}

// StatusForModelCatalogError maps model catalog failures to transport statuses.
func StatusForModelCatalogError(err error) int {
	var maxBytesErr *http.MaxBytesError
	switch {
	case err == nil:
		return http.StatusOK
	case errors.As(err, &maxBytesErr):
		return http.StatusRequestEntityTooLarge
	case errors.Is(err, ErrModelCatalogValidation),
		errors.Is(err, modelcatalog.ErrSourceNotRegistered):
		return http.StatusBadRequest
	case errors.Is(err, ErrModelCatalogUnavailable),
		errors.Is(err, modelcatalog.ErrAllSourcesFailed):
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// RespondOpenAIError writes an OpenAI-compatible error response envelope.
func RespondOpenAIError(c *gin.Context, status int, err error, maskInternalErrors bool) {
	message := http.StatusText(status)
	switch {
	case maskInternalErrors && status >= http.StatusInternalServerError:
		if strings.TrimSpace(message) == "" {
			message = errorsInternalServerErrorValue
		}
	case err != nil && strings.TrimSpace(err.Error()) != "":
		message = err.Error()
	case strings.TrimSpace(message) == "":
		message = errorsUnknownErrorValue
	}
	message = taskpkg.RedactClaimTokens(message)
	c.JSON(status, contract.OpenAIErrorResponse{
		Error: contract.OpenAIErrorPayload{
			Message: message,
			Type:    openAIErrorTypeForStatus(status),
			Param:   nil,
			Code:    openAIErrorCodeForStatus(status),
		},
	})
}

func openAIErrorTypeForStatus(status int) string {
	if status >= http.StatusInternalServerError {
		return "server_error"
	}
	return "invalid_request_error"
}

func openAIErrorCodeForStatus(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "invalid_request"
	case http.StatusUnauthorized:
		return "unauthorized"
	case http.StatusForbidden:
		return "forbidden"
	case http.StatusNotFound:
		return "not_found"
	case http.StatusRequestEntityTooLarge:
		return "request_too_large"
	case http.StatusServiceUnavailable:
		return "service_unavailable"
	default:
		if status >= http.StatusInternalServerError {
			return "internal_error"
		}
		return "api_error"
	}
}
