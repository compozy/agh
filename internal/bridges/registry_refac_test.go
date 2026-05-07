package bridges_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	"github.com/pedronauck/agh/internal/testutil"
)

func TestRegistryContextRefacs(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		call func(*bridgepkg.Service, context.Context) error
	}{
		{
			name: "Should reject canceled ListInstances before calling the store",
			call: func(registry *bridgepkg.Service, ctx context.Context) error {
				_, err := registry.ListInstances(ctx)
				return err
			},
		},
		{
			name: "Should reject canceled BuildRoutingKey before calling the store",
			call: func(registry *bridgepkg.Service, ctx context.Context) error {
				_, err := registry.BuildRoutingKey(ctx, bridgepkg.RoutingKey{BridgeInstanceID: "brg-canceled"})
				return err
			},
		},
		{
			name: "Should reject canceled ResolveDeliveryTarget before calling the store",
			call: func(registry *bridgepkg.Service, ctx context.Context) error {
				_, err := registry.ResolveDeliveryTarget(ctx, bridgepkg.ResolveDeliveryTargetRequest{
					BridgeInstanceID: "brg-canceled",
				})
				return err
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			storeCalled := false
			registry := bridgepkg.NewRegistry(stubRegistryStore{
				getBridgeInstanceFn: func(context.Context, string) (bridgepkg.BridgeInstance, error) {
					storeCalled = true
					return bridgepkg.BridgeInstance{}, errors.New("store should not be called")
				},
				listBridgeInstancesFn: func(context.Context) ([]bridgepkg.BridgeInstance, error) {
					storeCalled = true
					return nil, errors.New("store should not be called")
				},
			})
			ctx, cancel := context.WithCancel(testutil.Context(t))
			cancel()

			if err := tc.call(registry, ctx); !errors.Is(err, context.Canceled) {
				t.Fatalf("registry call error = %v, want %v", err, context.Canceled)
			}
			if storeCalled {
				t.Fatal("registry call reached store after context cancellation")
			}
		})
	}
}

func TestBridgeProviderConfigRefacs(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		config json.RawMessage
	}{
		{name: "Should reject scalar provider config", config: json.RawMessage(`"bot"`)},
		{name: "Should reject array provider config", config: json.RawMessage(`["bot"]`)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			instance := providerConfigRefacInstance(tc.config)
			requireProviderConfigShapeError(t, instance.Validate())

			req := providerConfigRefacCreateRequest(tc.config)
			requireProviderConfigShapeError(t, req.Validate())

			registry, _ := newRegistryTestHarness(t)
			_, err := registry.CreateInstance(testutil.Context(t), req)
			requireProviderConfigShapeError(t, err)

			validReq := providerConfigRefacCreateRequest(json.RawMessage(`{"mode":"bot"}`))
			validReq.ID = "brg-provider-update"
			created := createTestBridgeInstance(t, registry, validReq)
			update := bridgepkg.UpdateInstanceRequest{
				ID:             created.ID,
				ProviderConfig: &tc.config,
			}
			_, err = registry.UpdateInstance(testutil.Context(t), update)
			requireProviderConfigShapeError(t, err)
		})
	}
}

func providerConfigRefacInstance(config json.RawMessage) bridgepkg.BridgeInstance {
	return bridgepkg.BridgeInstance{
		ID:             "brg-provider-refac",
		Scope:          bridgepkg.ScopeGlobal,
		Platform:       "slack",
		ExtensionName:  "slack-adapter",
		DisplayName:    "Slack Provider",
		Enabled:        true,
		Status:         bridgepkg.BridgeStatusReady,
		RoutingPolicy:  bridgepkg.RoutingPolicy{IncludePeer: true},
		ProviderConfig: config,
	}
}

func providerConfigRefacCreateRequest(config json.RawMessage) bridgepkg.CreateInstanceRequest {
	return bridgepkg.CreateInstanceRequest{
		ID:             "brg-provider-create",
		Scope:          bridgepkg.ScopeGlobal,
		Platform:       "slack",
		ExtensionName:  "slack-adapter",
		DisplayName:    "Slack Provider",
		Enabled:        true,
		Status:         bridgepkg.BridgeStatusReady,
		RoutingPolicy:  bridgepkg.RoutingPolicy{IncludePeer: true},
		ProviderConfig: config,
	}
}

func requireProviderConfigShapeError(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("provider config validation error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "bridge instance provider config must be a JSON object or null") {
		t.Fatalf("provider config validation error = %v, want JSON object shape error", err)
	}
}
