package core

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/api/contract"
	automationpkg "github.com/pedronauck/agh/internal/automation"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	"github.com/pedronauck/agh/internal/network"
	"github.com/pedronauck/agh/internal/session"
	taskpkg "github.com/pedronauck/agh/internal/task"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func TestStatusForBridgeError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want int
	}{
		{
			name: "Should return bad request for body path mismatch",
			err:  contract.ErrBridgeInstanceMismatch,
			want: http.StatusBadRequest,
		},
		{
			name: "Should return not found for missing bridge",
			err:  bridgepkg.ErrBridgeInstanceNotFound,
			want: http.StatusNotFound,
		},
		{
			name: "Should return not found for missing route",
			err:  bridgepkg.ErrBridgeRouteNotFound,
			want: http.StatusNotFound,
		},
		{
			name: "Should return not found for missing workspace",
			err:  workspacepkg.ErrWorkspaceNotFound,
			want: http.StatusNotFound,
		},
		{
			name: "Should return conflict for unavailable instance",
			err:  bridgepkg.ErrBridgeInstanceUnavailable,
			want: http.StatusConflict,
		},
		{
			name: "Should return conflict for invalid state transition",
			err:  bridgepkg.ErrInvalidBridgeStateTransition,
			want: http.StatusConflict,
		},
		{
			name: "Should return not found for missing delivery",
			err:  bridgepkg.ErrDeliveryNotFound,
			want: http.StatusNotFound,
		},
		{
			name: "Should return service unavailable for saturated delivery queue",
			err:  bridgepkg.ErrDeliveryQueueSaturated,
			want: http.StatusServiceUnavailable,
		},
		{
			name: "Should return service unavailable for transport outage",
			err:  bridgepkg.ErrDeliveryTransportUnavailable,
			want: http.StatusServiceUnavailable,
		},
		{
			name: "Should return internal server error for unknown failures",
			err:  errors.New("boom"),
			want: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := StatusForBridgeError(tt.err); got != tt.want {
				t.Fatalf("StatusForBridgeError(%v) = %d, want %d", tt.err, got, tt.want)
			}
		})
	}
}

func TestTaskErrorHelpers(t *testing.T) {
	t.Parallel()

	wrapped := NewTaskValidationError(errors.New("bad input"))
	if !errors.Is(wrapped, taskpkg.ErrValidation) {
		t.Fatalf("NewTaskValidationError() = %v, want wrapped task validation sentinel", wrapped)
	}
	if got := NewTaskValidationError(nil); got != nil {
		t.Fatalf("NewTaskValidationError(nil) = %v, want nil", got)
	}

	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "nil", err: nil, want: http.StatusOK},
		{name: "invalid scope", err: taskpkg.ErrInvalidScopeBinding, want: http.StatusBadRequest},
		{name: "immutable field", err: taskpkg.ErrImmutableField, want: http.StatusBadRequest},
		{name: "run missing", err: taskpkg.ErrTaskRunNotFound, want: http.StatusNotFound},
		{name: "session missing", err: session.ErrSessionNotFound, want: http.StatusNotFound},
		{name: "os not exist", err: os.ErrNotExist, want: http.StatusNotFound},
		{name: "workspace root missing", err: workspacepkg.ErrWorkspaceRootMissing, want: http.StatusGone},
		{name: "attach forbidden", err: taskpkg.ErrSessionAttachNotAllowed, want: http.StatusConflict},
		{name: "stale network channel", err: taskpkg.ErrStaleNetworkChannel, want: http.StatusConflict},
		{name: "default", err: errors.New("boom"), want: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := StatusForTaskError(tt.err); got != tt.want {
				t.Fatalf("StatusForTaskError(%v) = %d, want %d", tt.err, got, tt.want)
			}
		})
	}
}

func TestAutomationAndNetworkErrorHelpers(t *testing.T) {
	t.Parallel()

	automationErr := NewAutomationValidationError(errors.New("bad automation request"))
	if !errors.Is(automationErr, ErrAutomationValidation) {
		t.Fatalf("NewAutomationValidationError() = %v, want ErrAutomationValidation", automationErr)
	}
	if got := NewAutomationValidationError(nil); got != nil {
		t.Fatalf("NewAutomationValidationError(nil) = %v, want nil", got)
	}

	networkErr := NewNetworkValidationError(errors.New("bad network request"))
	if !errors.Is(networkErr, ErrNetworkValidation) {
		t.Fatalf("NewNetworkValidationError() = %v, want ErrNetworkValidation", networkErr)
	}
	if got := NewNetworkValidationError(nil); got != nil {
		t.Fatalf("NewNetworkValidationError(nil) = %v, want nil", got)
	}

	if got := StatusForAutomationError(nil); got != http.StatusOK {
		t.Fatalf("StatusForAutomationError(nil) = %d, want %d", got, http.StatusOK)
	}
	if got := StatusForAutomationError(automationpkg.ErrManagerNotRunning); got != http.StatusServiceUnavailable {
		t.Fatalf("StatusForAutomationError(manager not running) = %d, want %d", got, http.StatusServiceUnavailable)
	}
	if got := StatusForAutomationError(automationpkg.ErrWebhookSignatureInvalid); got != http.StatusUnauthorized {
		t.Fatalf("StatusForAutomationError(signature invalid) = %d, want %d", got, http.StatusUnauthorized)
	}

	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "nil", err: nil, want: http.StatusOK},
		{name: "validation", err: ErrNetworkValidation, want: http.StatusBadRequest},
		{name: "local peer missing", err: network.ErrLocalPeerNotFound, want: http.StatusNotFound},
		{name: "missing field", err: network.ErrMissingField, want: http.StatusBadRequest},
		{name: "default", err: errors.New("boom"), want: http.StatusInternalServerError},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := StatusForNetworkError(tt.err); got != tt.want {
				t.Fatalf("StatusForNetworkError(%v) = %d, want %d", tt.err, got, tt.want)
			}
		})
	}
}

func TestRespondErrorFallbackBranches(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		status     int
		err        error
		mask       bool
		wantErr    string
		wantStatus int
	}{
		{name: "unknown error fallback", status: 0, err: nil, mask: false, wantErr: "unknown error", wantStatus: 200},
		{name: "masked internal fallback", status: 599, err: nil, mask: true, wantErr: "internal server error", wantStatus: 599},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			RespondError(ctx, tt.status, tt.err, tt.mask)

			var payload contract.ErrorPayload
			if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}
			if payload.Error != tt.wantErr {
				t.Fatalf("payload.Error = %q, want %q", payload.Error, tt.wantErr)
			}
			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.wantStatus)
			}
		})
	}
}
