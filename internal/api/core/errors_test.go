package core

import (
	"errors"
	"net/http"
	"testing"

	"github.com/pedronauck/agh/internal/api/contract"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
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
