package subprocess

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/bridges"
)

func TestInitializeBridgeRuntimeValidateRejectsInvalidProviderScopedPayload(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	validManaged := InitializeBridgeManagedInstance{
		Instance: bridges.BridgeInstance{
			ID:            "brg-1",
			Scope:         bridges.ScopeGlobal,
			Platform:      "telegram",
			ExtensionName: "telegram-reference",
			DisplayName:   "Telegram",
			Enabled:       true,
			Status:        bridges.BridgeStatusReady,
			RoutingPolicy: bridges.RoutingPolicy{IncludePeer: true},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		BoundSecrets: []InitializeBridgeBoundSecret{
			{BindingName: "bot_token", Kind: "token", Value: "secret"},
		},
	}

	tests := []struct {
		name    string
		runtime InitializeBridgeRuntime
		want    string
	}{
		{
			name: "missing provider identity",
			runtime: InitializeBridgeRuntime{
				RuntimeVersion:   InitializeBridgeRuntimeVersion1,
				Platform:         "telegram",
				ManagedInstances: []InitializeBridgeManagedInstance{validManaged},
			},
			want: "provider is required",
		},
		{
			name: "invalid managed instance snapshot",
			runtime: InitializeBridgeRuntime{
				RuntimeVersion: InitializeBridgeRuntimeVersion1,
				Provider:       "telegram-reference",
				Platform:       "telegram",
				ManagedInstances: []InitializeBridgeManagedInstance{{
					Instance: bridges.BridgeInstance{
						ID:            "brg-invalid",
						Scope:         bridges.ScopeGlobal,
						Platform:      "telegram",
						DisplayName:   "Telegram",
						Enabled:       true,
						Status:        bridges.BridgeStatusReady,
						RoutingPolicy: bridges.RoutingPolicy{IncludePeer: true},
						CreatedAt:     now,
						UpdatedAt:     now,
					},
				}},
			},
			want: "bridge instance extension name",
		},
		{
			name: "platform mismatch",
			runtime: InitializeBridgeRuntime{
				RuntimeVersion:   InitializeBridgeRuntimeVersion1,
				Provider:         "telegram-reference",
				Platform:         "slack",
				ManagedInstances: []InitializeBridgeManagedInstance{validManaged},
			},
			want: "does not match runtime platform",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.runtime.Validate()
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Validate() error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestCloneInitializeBridgeRuntimeDoesNotAliasManagedInstanceState(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 15, 12, 5, 0, 0, time.UTC)
	src := &InitializeBridgeRuntime{
		RuntimeVersion: InitializeBridgeRuntimeVersion1,
		Provider:       "telegram-reference",
		Platform:       "telegram",
		ManagedInstances: []InitializeBridgeManagedInstance{{
			Instance: bridges.BridgeInstance{
				ID:               "brg-1",
				Scope:            bridges.ScopeWorkspace,
				WorkspaceID:      "ws-1",
				Platform:         "telegram",
				ExtensionName:    "telegram-reference",
				DisplayName:      "Telegram",
				Enabled:          true,
				Status:           bridges.BridgeStatusDegraded,
				RoutingPolicy:    bridges.RoutingPolicy{IncludePeer: true},
				ProviderConfig:   json.RawMessage(`{"mode":"bot"}`),
				DeliveryDefaults: json.RawMessage(`{"peer_id":"peer-1"}`),
				Degradation: &bridges.BridgeDegradation{
					Reason:  bridges.BridgeDegradationReasonAuthFailed,
					Message: "token expired",
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
			BoundSecrets: []InitializeBridgeBoundSecret{
				{BindingName: "bot_token", Kind: "token", Value: "secret-1"},
			},
		}},
	}

	cloned := CloneInitializeBridgeRuntime(src)
	if cloned == nil {
		t.Fatal("CloneInitializeBridgeRuntime() = nil, want non-nil")
	}

	src.ManagedInstances[0].Instance.ID = "mutated"
	src.ManagedInstances[0].Instance.ProviderConfig[0] = '['
	src.ManagedInstances[0].Instance.DeliveryDefaults[0] = '['
	src.ManagedInstances[0].Instance.Degradation.Message = "mutated"
	src.ManagedInstances[0].BoundSecrets[0].Value = "mutated"
	src.ManagedInstances = append(src.ManagedInstances, InitializeBridgeManagedInstance{
		Instance: bridges.BridgeInstance{
			ID:            "brg-2",
			Scope:         bridges.ScopeGlobal,
			Platform:      "telegram",
			ExtensionName: "telegram-reference",
			DisplayName:   "Telegram 2",
			Enabled:       true,
			Status:        bridges.BridgeStatusReady,
			RoutingPolicy: bridges.RoutingPolicy{IncludePeer: true},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
	})

	if got, want := cloned.ManagedInstances[0].Instance.ID, "brg-1"; got != want {
		t.Fatalf("cloned managed instance id = %q, want %q", got, want)
	}
	if got, want := string(cloned.ManagedInstances[0].Instance.ProviderConfig), `{"mode":"bot"}`; got != want {
		t.Fatalf("cloned provider config = %q, want %q", got, want)
	}
	if got, want := string(cloned.ManagedInstances[0].Instance.DeliveryDefaults), `{"peer_id":"peer-1"}`; got != want {
		t.Fatalf("cloned delivery defaults = %q, want %q", got, want)
	}
	if got, want := cloned.ManagedInstances[0].Instance.Degradation.Message, "token expired"; got != want {
		t.Fatalf("cloned degradation message = %q, want %q", got, want)
	}
	if got, want := cloned.ManagedInstances[0].BoundSecrets[0].Value, "secret-1"; got != want {
		t.Fatalf("cloned bound secret value = %q, want %q", got, want)
	}
	if got, want := len(cloned.ManagedInstances), 1; got != want {
		t.Fatalf("len(cloned.ManagedInstances) = %d, want %d", got, want)
	}
}
