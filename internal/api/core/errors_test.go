package core

import (
	"errors"
	"net/http"
	"testing"

	"github.com/pedronauck/agh/internal/api/contract"
	channelspkg "github.com/pedronauck/agh/internal/channels"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func TestStatusForChannelError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want int
	}{
		{
			name: "Should return bad request for body path mismatch",
			err:  contract.ErrChannelInstanceMismatch,
			want: http.StatusBadRequest,
		},
		{
			name: "Should return not found for missing channel",
			err:  channelspkg.ErrChannelInstanceNotFound,
			want: http.StatusNotFound,
		},
		{
			name: "Should return not found for missing route",
			err:  channelspkg.ErrChannelRouteNotFound,
			want: http.StatusNotFound,
		},
		{
			name: "Should return not found for missing workspace",
			err:  workspacepkg.ErrWorkspaceNotFound,
			want: http.StatusNotFound,
		},
		{
			name: "Should return conflict for unavailable instance",
			err:  channelspkg.ErrChannelInstanceUnavailable,
			want: http.StatusConflict,
		},
		{
			name: "Should return conflict for invalid state transition",
			err:  channelspkg.ErrInvalidChannelStateTransition,
			want: http.StatusConflict,
		},
		{
			name: "Should return not found for missing delivery",
			err:  channelspkg.ErrDeliveryNotFound,
			want: http.StatusNotFound,
		},
		{
			name: "Should return service unavailable for saturated delivery queue",
			err:  channelspkg.ErrDeliveryQueueSaturated,
			want: http.StatusServiceUnavailable,
		},
		{
			name: "Should return service unavailable for transport outage",
			err:  channelspkg.ErrDeliveryTransportUnavailable,
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
			if got := StatusForChannelError(tt.err); got != tt.want {
				t.Fatalf("StatusForChannelError(%v) = %d, want %d", tt.err, got, tt.want)
			}
		})
	}
}
