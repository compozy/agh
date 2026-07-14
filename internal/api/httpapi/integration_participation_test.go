//go:build integration

package httpapi

import (
	"context"
	"strings"
	"testing"

	"github.com/compozy/agh/internal/network/participation"
	"github.com/compozy/agh/internal/store/globaldb"
)

func assertResolvedParticipationChannel(
	t *testing.T,
	spec *participation.Spec,
	wantChannel string,
) {
	t.Helper()
	if spec == nil {
		t.Fatalf("resolved_network_participation = nil, want channel %q", wantChannel)
	}
	if got, want := spec.Mode, participation.ModeLive; got != want {
		t.Fatalf("resolved_network_participation.mode = %q, want %q", got, want)
	}
	if got, want := strings.TrimSpace(spec.ChannelID), strings.TrimSpace(wantChannel); got != want {
		t.Fatalf("resolved_network_participation.channel_id = %q, want %q", got, want)
	}
}

func assertLocalResolvedParticipation(t *testing.T, spec *participation.Spec) {
	t.Helper()
	if spec == nil {
		return
	}
	if spec.Mode != participation.ModeLocal && strings.TrimSpace(spec.ChannelID) != "" {
		t.Fatalf("resolved_network_participation = %#v, want local/empty channel", spec)
	}
}

func newIntegrationParticipationResolver(t *testing.T, db *globaldb.GlobalDB) participation.Resolver {
	t.Helper()

	defaults := participation.Bounds{
		MaxWakes:         4,
		MaxWakeWallTime:  "30s",
		MaxTotalWallTime: "2m",
		MaxInputTokens:   4096,
		MaxOutputTokens:  4096,
		MaxWakeDepth:     4,
		CoalesceWindow:   "250ms",
	}
	resolver, err := participation.NewResolver(participation.ResolverOptions{
		Defaults: defaults,
		Limits: participation.Limits{
			MaxWakes:          16,
			MaxWakeWallTime:   "2m",
			MaxTotalWallTime:  "10m",
			MaxInputTokens:    65536,
			MaxOutputTokens:   65536,
			MaxWakeDepth:      16,
			MinCoalesceWindow: "100ms",
			MaxCoalesceWindow: "5s",
		},
		Availability: func(ctx context.Context) (bool, error) {
			state, readErr := db.GetNetworkAvailability(ctx)
			if readErr != nil {
				return false, readErr
			}
			return state.Enabled, nil
		},
		ChannelExists: func(context.Context, string, string) (bool, error) {
			return true, nil
		},
		LiveSupport: func(context.Context, participation.ResolveInput) (bool, error) {
			return true, nil
		},
	})
	if err != nil {
		t.Fatalf("participation.NewResolver() error = %v", err)
	}
	return resolver
}

func enableIntegrationLiveNetwork(t *testing.T, registry *globaldb.GlobalDB) {
	t.Helper()

	if _, err := registry.SetNetworkAvailability(context.Background(), true, "test:enable"); err != nil {
		t.Fatalf("SetNetworkAvailability(true) error = %v", err)
	}
}
