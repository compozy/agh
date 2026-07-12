package bridges_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	bridgepkg "github.com/compozy/agh/internal/bridges"
	"github.com/compozy/agh/internal/testutil"
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

func TestBridgeProviderConfigRejectsOperatorOwnedDestinations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  string
		raw  json.RawMessage
	}{
		{
			name: "Should reject an API base URL",
			key:  "api_base_url",
			raw:  json.RawMessage(`{"api_base_url":"https://attacker.example/api"}`),
		},
		{
			name: "Should reject an OAuth token URL",
			key:  "oauth_token_url",
			raw:  json.RawMessage(`{"oauth_token_url":"https://attacker.example/token"}`),
		},
		{
			name: "Should reject a service URL",
			key:  "service_url",
			raw:  json.RawMessage(`{"service_url":"https://attacker.example/service"}`),
		},
		{
			name: "Should reject a nested OpenID metadata URL",
			key:  "openid_metadata_url",
			raw:  json.RawMessage(`{"auth":{"openid_metadata_url":"https://attacker.example/openid"}}`),
		},
		{
			name: "Should reject a nested token URL",
			key:  "token_url",
			raw:  json.RawMessage(`{"auth":{"token_url":"https://attacker.example/token"}}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			request := providerConfigRefacCreateRequest(tt.raw)
			err := request.Validate()
			if err == nil {
				t.Fatalf("CreateInstanceRequest.Validate() error = nil, want rejection for %q", tt.key)
			}
			if !strings.Contains(err.Error(), tt.key) || !strings.Contains(err.Error(), "operator-owned") {
				t.Fatalf("CreateInstanceRequest.Validate() error = %q, want %q operator-owned rejection", err, tt.key)
			}

			registry, store := newRegistryTestHarness(t)
			_, err = registry.CreateInstance(testutil.Context(t), request)
			if err == nil {
				t.Fatalf("CreateInstance() error = nil, want rejection for %q", tt.key)
			}
			if len(store.instances) != 0 {
				t.Fatalf("persisted instances = %d, want 0", len(store.instances))
			}
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
