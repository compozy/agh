package participation_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/compozy/agh/internal/network/participation"
)

func TestValidateShouldEnforceParticipationContract(t *testing.T) {
	t.Parallel()

	validBounds := completeBoundsRequest()
	tests := []struct {
		name    string
		request participation.Request
		wantErr error
		check   func(*testing.T, participation.Request)
	}{
		{
			name: "Should normalize a valid named live request",
			request: participation.Request{
				Mode:            new(participation.ModeLive),
				ChannelStrategy: new(participation.StrategyNamed),
				ChannelID:       new(" builders "),
				Bounds:          validBounds,
			},
			check: func(t *testing.T, got participation.Request) {
				t.Helper()
				if got.ChannelID == nil || *got.ChannelID != "builders" {
					t.Fatalf("Validate() ChannelID = %v, want builders", got.ChannelID)
				}
			},
		},
		{
			name: "Should reject local mode with a channel",
			request: participation.Request{
				Mode:      new(participation.ModeLocal),
				ChannelID: new("builders"),
			},
			wantErr: participation.ErrStrategyChannelConflict,
		},
		{
			name: "Should reject live mode without a strategy",
			request: participation.Request{
				Mode:   new(participation.ModeLive),
				Bounds: validBounds,
			},
			wantErr: participation.ErrStrategyInvalid,
		},
		{
			name: "Should reject named strategy without a channel",
			request: participation.Request{
				Mode:            new(participation.ModeLive),
				ChannelStrategy: new(participation.StrategyNamed),
				Bounds:          validBounds,
			},
			wantErr: participation.ErrStrategyInvalid,
		},
		{
			name: "Should reject a derived strategy with a channel",
			request: participation.Request{
				Mode:            new(participation.ModeLive),
				ChannelStrategy: new(participation.StrategyRun),
				ChannelID:       new("builders"),
				Bounds:          validBounds,
			},
			wantErr: participation.ErrStrategyChannelConflict,
		},
		{
			name: "Should reject live mode without finite bounds",
			request: participation.Request{
				Mode:            new(participation.ModeLive),
				ChannelStrategy: new(participation.StrategyRun),
			},
			wantErr: participation.ErrBoundsExceedCeiling,
		},
		{
			name: "Should reject an unknown mode and list allowed values",
			request: participation.Request{
				Mode: new(participation.Mode("mailbox")),
			},
			wantErr: participation.ErrStrategyInvalid,
			check: func(t *testing.T, _ participation.Request) {
				t.Helper()
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := participation.Validate(tc.request)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("Validate() error = %v, want errors.Is(%v)", err, tc.wantErr)
				}
				if tc.request.Mode != nil && *tc.request.Mode == participation.Mode("mailbox") &&
					(!strings.Contains(err.Error(), "local") || !strings.Contains(err.Error(), "live")) {
					t.Fatalf("Validate() error = %q, want allowed modes", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
			if tc.check != nil {
				tc.check(t, got)
			}
		})
	}
}

func TestValidateShouldRejectInvalidDurations(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"0s", "-1s", "250"} {
		t.Run("Should reject "+value, func(t *testing.T) {
			t.Parallel()

			bounds := completeBoundsRequest()
			bounds.CoalesceWindow = &value
			_, err := participation.Validate(participation.Request{
				Mode:            new(participation.ModeLive),
				ChannelStrategy: new(participation.StrategyRun),
				Bounds:          bounds,
			})
			if !errors.Is(err, participation.ErrBoundsExceedCeiling) {
				t.Fatalf("Validate() error = %v, want ErrBoundsExceedCeiling", err)
			}
			if !strings.Contains(err.Error(), "coalesce_window") {
				t.Fatalf("Validate() error = %q, want field name", err)
			}
		})
	}

	t.Run("Should accept a positive subsecond duration", func(t *testing.T) {
		t.Parallel()

		bounds := completeBoundsRequest()
		bounds.CoalesceWindow = new("250ms")
		if _, err := participation.Validate(participation.Request{
			Mode:            new(participation.ModeLive),
			ChannelStrategy: new(participation.StrategyRun),
			Bounds:          bounds,
		}); err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
	})
}

func TestResolverShouldResolveLocalWithoutChannelLookup(t *testing.T) {
	t.Parallel()

	lookups := 0
	resolver := newTestResolver(t, participation.ResolverOptions{
		ChannelExists: func(context.Context, string, string) (bool, error) {
			lookups++
			return true, nil
		},
	})
	spec, err := resolver.Resolve(context.Background(), participation.ResolveInput{})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if spec.Mode != participation.ModeLocal || spec.Source != participation.SourceBuiltInLocal {
		t.Fatalf("Resolve() spec = %#v, want built-in local", spec)
	}
	if spec.ChannelID != "" || !spec.Bounds.IsZero() {
		t.Fatalf("Resolve() spec = %#v, want no channel or bounds", spec)
	}
	if lookups != 0 {
		t.Fatalf("Resolve() channel lookups = %d, want 0", lookups)
	}
}

func TestResolverShouldRejectRunStrategyWithoutRunID(t *testing.T) {
	t.Parallel()

	resolver := newTestResolver(t, participation.ResolverOptions{})
	_, err := resolver.Resolve(context.Background(), participation.ResolveInput{
		Owner: participation.OwnerRef{Kind: participation.OwnerKindTaskRun, ID: "owner-1"},
		Request: &participation.Request{
			Mode:            new(participation.ModeLive),
			ChannelStrategy: new(participation.StrategyRun),
		},
	})
	if !errors.Is(err, participation.ErrStrategyInvalid) {
		t.Fatalf("Resolve() error = %v, want ErrStrategyInvalid", err)
	}
	if !strings.Contains(err.Error(), "task_run") {
		t.Fatalf("Resolve() error = %q, want owner kind", err)
	}
}

func TestResolverShouldDeriveUniqueChannelIDs(t *testing.T) {
	t.Parallel()

	resolver := newTestResolver(t, participation.ResolverOptions{})
	resolve := func(runID string) participation.Spec {
		t.Helper()
		spec, err := resolver.Resolve(context.Background(), participation.ResolveInput{
			Owner: participation.OwnerRef{Kind: participation.OwnerKindTaskRun, ID: runID},
			RunID: runID,
			Request: &participation.Request{
				Mode:            new(participation.ModeLive),
				ChannelStrategy: new(participation.StrategyRun),
			},
		})
		if err != nil {
			t.Fatalf("Resolve(%q) error = %v", runID, err)
		}
		if len(spec.ChannelID) > 64 {
			t.Fatalf("Resolve(%q) ChannelID length = %d, want <= 64", runID, len(spec.ChannelID))
		}
		return spec
	}

	caseVariant := resolve("Run-Release")
	caseVariantLower := resolve("run-release")
	if caseVariant.ChannelID == caseVariantLower.ChannelID {
		t.Fatalf("case-variant run IDs derived the same channel %q", caseVariant.ChannelID)
	}

	sharedPrefix := strings.Repeat("release-segment-", 5)
	longA := resolve(sharedPrefix + "alpha")
	longB := resolve(sharedPrefix + "bravo")
	if longA.ChannelID == longB.ChannelID {
		t.Fatalf("long run IDs derived the same channel %q", longA.ChannelID)
	}
	if repeated := resolve("Run-Release"); repeated.ChannelID != caseVariant.ChannelID {
		t.Fatalf("same run ID derived %q then %q", caseVariant.ChannelID, repeated.ChannelID)
	}
}

func TestSpecShouldRoundTripLosslessly(t *testing.T) {
	t.Parallel()

	want := participation.Spec{
		Version:         participation.SpecVersion,
		Mode:            participation.ModeLive,
		WorkspaceID:     "ws-1",
		ChannelStrategy: participation.StrategyNamed,
		ChannelID:       "builders",
		Source:          participation.SourceExplicitRequest,
		Bounds:          completeBounds(),
	}
	payload, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if !strings.Contains(string(payload), `"version":"network-participation/v1"`) {
		t.Fatalf("json.Marshal() = %s, want canonical version", payload)
	}
	var got participation.Spec
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got != want {
		t.Fatalf("round trip = %#v, want %#v", got, want)
	}
	if err := participation.ValidateSpec(got); err != nil {
		t.Fatalf("ValidateSpec() error = %v", err)
	}
}

func TestValidateSpecShouldRejectProjectionConflicts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		spec    participation.Spec
		wantErr error
	}{
		{
			name: "Should reject a local snapshot carrying a channel",
			spec: participation.Spec{
				Version:   participation.SpecVersion,
				Mode:      participation.ModeLocal,
				ChannelID: "builders",
				Source:    participation.SourceBuiltInLocal,
			},
			wantErr: participation.ErrStrategyChannelConflict,
		},
		{
			name: "Should reject a snapshot with an unknown version",
			spec: participation.Spec{
				Version: "network-participation/v2",
				Mode:    participation.ModeLocal,
				Source:  participation.SourceBuiltInLocal,
			},
			wantErr: participation.ErrStrategyInvalid,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if err := participation.ValidateSpec(tc.spec); !errors.Is(err, tc.wantErr) {
				t.Fatalf("ValidateSpec() error = %v, want errors.Is(%v)", err, tc.wantErr)
			}
		})
	}
}

func TestResolveBoundsShouldTreatAdministrativeCeilingsAsInclusive(t *testing.T) {
	t.Parallel()

	limits := participation.Limits{
		MaxWakes:          64,
		MaxWakeWallTime:   "15m",
		MaxTotalWallTime:  "2h",
		MaxInputTokens:    1_000_000,
		MaxOutputTokens:   200_000,
		MaxWakeDepth:      5,
		MinCoalesceWindow: "100ms",
		MaxCoalesceWindow: "5s",
	}
	equal := 64
	resolved, err := participation.ResolveBounds(
		&participation.BoundsRequest{MaxWakes: &equal},
		completeBounds(),
		limits,
	)
	if err != nil {
		t.Fatalf("ResolveBounds(equal ceiling) error = %v", err)
	}
	if resolved.MaxWakes != equal {
		t.Fatalf("ResolveBounds(equal ceiling).MaxWakes = %d, want %d", resolved.MaxWakes, equal)
	}

	above := 65
	_, err = participation.ResolveBounds(
		&participation.BoundsRequest{MaxWakes: &above},
		completeBounds(),
		limits,
	)
	if !errors.Is(err, participation.ErrBoundsExceedCeiling) {
		t.Fatalf("ResolveBounds(above ceiling) error = %v, want ErrBoundsExceedCeiling", err)
	}
	if !strings.Contains(err.Error(), "network.live.limits.max_wakes") {
		t.Fatalf("ResolveBounds(above ceiling) error = %q, want ceiling key", err)
	}
}

func newTestResolver(t *testing.T, options participation.ResolverOptions) participation.Resolver {
	t.Helper()
	options.Defaults = completeBounds()
	options.Limits = participation.Limits{
		MaxWakes:          64,
		MaxWakeWallTime:   "15m",
		MaxTotalWallTime:  "2h",
		MaxInputTokens:    1_000_000,
		MaxOutputTokens:   200_000,
		MaxWakeDepth:      5,
		MinCoalesceWindow: "100ms",
		MaxCoalesceWindow: "5s",
	}
	resolver, err := participation.NewResolver(options)
	if err != nil {
		t.Fatalf("NewResolver() error = %v", err)
	}
	return resolver
}

func completeBoundsRequest() *participation.BoundsRequest {
	bounds := completeBounds()
	return &participation.BoundsRequest{
		MaxWakes:         new(bounds.MaxWakes),
		MaxWakeWallTime:  new(bounds.MaxWakeWallTime),
		MaxTotalWallTime: new(bounds.MaxTotalWallTime),
		MaxInputTokens:   new(bounds.MaxInputTokens),
		MaxOutputTokens:  new(bounds.MaxOutputTokens),
		MaxWakeDepth:     new(bounds.MaxWakeDepth),
		CoalesceWindow:   new(bounds.CoalesceWindow),
	}
}

func completeBounds() participation.Bounds {
	return participation.Bounds{
		MaxWakes:         8,
		MaxWakeWallTime:  "5m",
		MaxTotalWallTime: "30m",
		MaxInputTokens:   200_000,
		MaxOutputTokens:  50_000,
		MaxWakeDepth:     3,
		CoalesceWindow:   "500ms",
	}
}
