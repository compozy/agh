package core_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/api/core"
	"github.com/pedronauck/agh/internal/api/testutil"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/observe"
)

func TestBridgeHandlersExposeDiagnostics(t *testing.T) {
	t.Parallel()

	t.Run("Should expose diagnostics from bridge routes secrets status and health", func(t *testing.T) {
		t.Parallel()

		gin.SetMode(gin.TestMode)
		bridge := bridgepkg.BridgeInstance{
			ID:            "brg-diagnostics",
			Scope:         bridgepkg.ScopeGlobal,
			Platform:      "telegram",
			ExtensionName: "ext-telegram",
			DisplayName:   "Support",
			Enabled:       true,
			Status:        bridgepkg.BridgeStatusAuthRequired,
			RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
			Degradation: &bridgepkg.BridgeDegradation{
				Reason:  bridgepkg.BridgeDegradationReasonAuthFailed,
				Message: "provider rejected credentials",
			},
		}
		provider := bridgepkg.BridgeProvider{
			Platform:      "telegram",
			ExtensionName: "ext-telegram",
			Enabled:       false,
			HealthMessage: "provider is disabled by policy",
			SecretSlots: []bridgepkg.BridgeSecretSlot{
				{Name: "bot_token", Required: true},
			},
		}
		homePaths := testutil.NewTestHomePaths(t)
		cfg := aghconfig.DefaultWithHome(homePaths)
		handlers := core.NewBaseHandlers(&core.BaseHandlerConfig{
			TransportName:                "api-core-test",
			MaskInternalErrors:           false,
			IncludeSessionWorkspaceInSSE: true,
			Sessions:                     testutil.StubSessionManager{},
			Observer: testutil.StubObserver{
				QueryBridgeHealthFn: func(context.Context) ([]observe.BridgeInstanceHealth, error) {
					return []observe.BridgeInstanceHealth{{
						BridgeInstanceID:      bridge.ID,
						Status:                bridgepkg.BridgeStatusAuthRequired,
						RouteCount:            0,
						DeliveryFailuresTotal: 2,
						AuthFailuresTotal:     1,
						LastError:             "temporary gateway timeout",
					}}, nil
				},
			},
			Bridges: testutil.StubBridgeService{
				ListInstancesFn: func(context.Context) ([]bridgepkg.BridgeInstance, error) {
					return []bridgepkg.BridgeInstance{bridge}, nil
				},
				GetInstanceFn: func(context.Context, string) (*bridgepkg.BridgeInstance, error) {
					return &bridge, nil
				},
				ListProvidersFn: func(context.Context) ([]bridgepkg.BridgeProvider, error) {
					return []bridgepkg.BridgeProvider{provider}, nil
				},
				ListSecretBindingsFn: func(context.Context, string) ([]bridgepkg.BridgeSecretBinding, error) {
					return nil, nil
				},
			},
			Workspaces: testutil.StubWorkspaceService{},
			HomePaths:  homePaths,
			Config:     cfg,
			Logger:     testutil.DiscardLogger(),
			StartedAt:  time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC),
			Now: func() time.Time {
				return time.Date(2026, 5, 19, 12, 0, 1, 0, time.UTC)
			},
			HTTPPort: cfg.HTTP.Port,
		})
		engine := gin.New()
		engine.GET("/bridges", handlers.ListBridges)
		engine.GET("/bridges/:id", handlers.GetBridge)

		listResp := performRequest(t, engine, http.MethodGet, "/bridges", nil)
		if listResp.Code != http.StatusOK {
			t.Fatalf("list status = %d body=%s", listResp.Code, listResp.Body.String())
		}
		var listPayload contract.BridgesResponse
		testutil.DecodeJSONResponse(t, listResp, &listPayload)
		assertBridgeDiagnosticKinds(t, listPayload.BridgeHealth[bridge.ID].Diagnostics)

		getResp := performRequest(t, engine, http.MethodGet, "/bridges/"+bridge.ID, nil)
		if getResp.Code != http.StatusOK {
			t.Fatalf("get status = %d body=%s", getResp.Code, getResp.Body.String())
		}
		var getPayload contract.BridgeResponse
		testutil.DecodeJSONResponse(t, getResp, &getPayload)
		assertBridgeDiagnosticKinds(t, getPayload.Health.Diagnostics)
	})
}

func assertBridgeDiagnosticKinds(t *testing.T, diagnostics []bridgepkg.BridgeDiagnostic) {
	t.Helper()

	byKind := make(map[bridgepkg.BridgeDiagnosticKind]bridgepkg.BridgeDiagnostic, len(diagnostics))
	for _, diagnostic := range diagnostics {
		byKind[diagnostic.Kind] = diagnostic
	}
	for _, kind := range []bridgepkg.BridgeDiagnosticKind{
		bridgepkg.BridgeDiagnosticKindUnsupportedCapability,
		bridgepkg.BridgeDiagnosticKindMissingToken,
		bridgepkg.BridgeDiagnosticKindUnknownDestination,
		bridgepkg.BridgeDiagnosticKindPermissionDenied,
		bridgepkg.BridgeDiagnosticKindTransientDeliveryFailure,
	} {
		if _, ok := byKind[kind]; !ok {
			t.Fatalf("diagnostics missing kind %q: %#v", kind, diagnostics)
		}
	}
	if got := byKind[bridgepkg.BridgeDiagnosticKindMissingToken].SecretSlot; got != "bot_token" {
		t.Fatalf("missing token secret slot = %q, want bot_token", got)
	}
}
