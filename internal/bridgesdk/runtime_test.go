package bridgesdk

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	bridgepkg "github.com/compozy/agh/internal/bridges"
	extensioncontract "github.com/compozy/agh/internal/extension/contract"
	extensionprotocol "github.com/compozy/agh/internal/extension/protocol"
	"github.com/compozy/agh/internal/subprocess"
)

func TestSessionAckDeliveryBuildsValidatedAck(t *testing.T) {
	t.Parallel()

	request := bridgepkg.DeliveryRequest{
		Event: bridgepkg.DeliveryEvent{
			DeliveryID:       "dlv-1",
			Seq:              2,
			EventType:        bridgepkg.DeliveryEventTypeDelta,
			BridgeInstanceID: "brg-1",
			RoutingKey: bridgepkg.RoutingKey{
				Scope:            bridgepkg.ScopeWorkspace,
				WorkspaceID:      "ws-1",
				BridgeInstanceID: "brg-1",
				PeerID:           "peer-1",
			},
			DeliveryTarget: bridgepkg.DeliveryTarget{
				BridgeInstanceID: "brg-1",
				PeerID:           "peer-1",
				Mode:             bridgepkg.DeliveryModeDirectSend,
			},
			Content: bridgepkg.MessageContent{
				Text: "hello",
			},
		},
	}

	session := &Session{}
	ack, err := session.AckDelivery(request, "remote-1", "")
	if err != nil {
		t.Fatalf("AckDelivery() error = %v", err)
	}
	if got, want := ack.DeliveryID, "dlv-1"; got != want {
		t.Fatalf("ack.DeliveryID = %q, want %q", got, want)
	}
	if got, want := ack.Seq, int64(2); got != want {
		t.Fatalf("ack.Seq = %d, want %d", got, want)
	}
	if got, want := ack.RemoteMessageID, "remote-1"; got != want {
		t.Fatalf("ack.RemoteMessageID = %q, want %q", got, want)
	}
}

func TestSessionReportClassifiedErrorReportsStateThroughHostAPI(t *testing.T) {
	t.Parallel()

	reported := extensioncontract.BridgesInstancesReportStateParams{}
	session := &Session{
		host: NewHostAPIClientFromCall(func(_ context.Context, method string, params any, result any) error {
			if got, want := method, "bridges/instances/report_state"; got != want {
				t.Fatalf("method = %q, want %q", got, want)
			}
			reported = params.(extensioncontract.BridgesInstancesReportStateParams)
			target := result.(*bridgepkg.BridgeInstance)
			*target = testBridgeInstance(reported.BridgeInstanceID)
			target.Status = reported.Status
			target.Degradation = reported.Degradation
			return nil
		}),
	}

	updated, recovery, err := session.ReportClassifiedError(
		context.Background(),
		"brg-1",
		ClassifyError(&RateLimitError{
			Err:        errors.New("slow down"),
			RetryAfter: time.Second,
		}),
	)
	if err != nil {
		t.Fatalf("ReportClassifiedError() error = %v", err)
	}
	if !recovery.Retry {
		t.Fatal("recovery.Retry = false, want true")
	}
	if updated == nil || updated.Status != bridgepkg.BridgeStatusDegraded {
		t.Fatalf("updated.Status = %#v, want degraded", updated)
	}
	if got, want := reported.BridgeInstanceID, "brg-1"; got != want {
		t.Fatalf("reported.BridgeInstanceID = %q, want %q", got, want)
	}
	if reported.Degradation == nil || reported.Degradation.Reason != bridgepkg.BridgeDegradationReasonRateLimited {
		t.Fatalf("reported.Degradation = %#v, want rate_limited", reported.Degradation)
	}
}

func TestNewRuntimeRejectsMissingRequiredConfig(t *testing.T) {
	t.Parallel()

	if _, err := NewRuntime(RuntimeConfig{}); err == nil {
		t.Fatal("NewRuntime(empty) error = nil, want non-nil")
	}
	if _, err := NewRuntime(RuntimeConfig{
		ExtensionInfo: subprocess.InitializeExtensionInfo{
			Name:    "telegram-adapter",
			Version: "1.0.0",
		},
	}); err == nil {
		t.Fatal("NewRuntime(missing deliver) error = nil, want non-nil")
	}
}

func TestSessionReportClassifiedErrorNoActionWhenRecoveryHasNoStatus(t *testing.T) {
	t.Parallel()

	session := &Session{
		host: NewHostAPIClientFromCall(func(context.Context, string, any, any) error {
			t.Fatal("host call executed for empty recovery")
			return nil
		}),
	}

	updated, recovery, err := session.ReportClassifiedError(context.Background(), "brg-1", ClassifiedError{})
	if err != nil {
		t.Fatalf("ReportClassifiedError() error = %v", err)
	}
	if updated != nil {
		t.Fatalf("updated = %#v, want nil", updated)
	}
	if recovery.Status != "" {
		t.Fatalf("recovery.Status = %q, want empty", recovery.Status)
	}
}

func TestDecodeParamsHandlesNullAndInvalidJSON(t *testing.T) {
	t.Parallel()

	var target map[string]any
	if err := decodeParams(nil, &target); err != nil {
		t.Fatalf("decodeParams(nil) error = %v", err)
	}
	if err := decodeParams(json.RawMessage("{"), &target); err == nil {
		t.Fatal("decodeParams(invalid json) error = nil, want non-nil")
	}
}

func TestSessionAccessorsExposeConfiguredHelpers(t *testing.T) {
	t.Parallel()

	cache := NewInstanceCache(testManagedRuntime("brg-1"))
	host := NewHostAPIClientFromCall(func(context.Context, string, any, any) error { return nil })
	session := &Session{cache: cache, host: host}

	if session.BridgeRuntime() == nil {
		t.Fatal("session.BridgeRuntime() = nil, want non-nil")
	}
	if session.HostAPI() != host {
		t.Fatal("session.HostAPI() did not return configured host client")
	}
	if session.Cache() != cache {
		t.Fatal("session.Cache() did not return configured cache")
	}
}

func TestSessionInitializeAccessorsReturnClones(t *testing.T) {
	t.Parallel()

	session := &Session{
		request: subprocess.InitializeRequest{
			Capabilities: subprocess.InitializeCapabilities{
				Provides:        []string{"bridge.adapter"},
				GrantedActions:  []extensionprotocol.HostAPIMethod{extensionprotocol.HostAPIMethodBridgesInstancesList},
				GrantedSecurity: []string{"bridge.read"},
			},
			Methods: subprocess.InitializeMethods{
				DaemonRequests:    []string{"ping"},
				ExtensionServices: []string{"bridges/deliver"},
			},
			Runtime: subprocess.InitializeRuntime{
				Bridge: testManagedRuntime("brg-1"),
			},
		},
		response: subprocess.InitializeResponse{
			AcceptedCapabilities: subprocess.AcceptedCapabilities{
				Provides: []string{"bridge.adapter"},
				Actions:  []extensionprotocol.HostAPIMethod{extensionprotocol.HostAPIMethodBridgesInstancesGet},
				Security: []string{"bridge.write"},
			},
			ImplementedMethods:  []string{"bridges/deliver"},
			SupportedHookEvents: []string{"hook"},
		},
	}

	request := session.InitializeRequest()
	response := session.InitializeResponse()

	request.Capabilities.Provides[0] = "mutated"
	request.Capabilities.GrantedActions[0] = extensionprotocol.HostAPIMethodBridgesInstancesGet
	request.Capabilities.GrantedSecurity[0] = "mutated"
	request.Methods.DaemonRequests[0] = "mutated"
	request.Methods.ExtensionServices[0] = "mutated"
	request.Runtime.Bridge.ManagedInstances[0].Instance.ID = "mutated"

	response.AcceptedCapabilities.Provides[0] = "mutated"
	response.AcceptedCapabilities.Actions[0] = extensionprotocol.HostAPIMethodBridgesMessagesIngest
	response.AcceptedCapabilities.Security[0] = "mutated"
	response.ImplementedMethods[0] = "mutated"
	response.SupportedHookEvents[0] = "mutated"

	if got, want := session.request.Capabilities.Provides[0], "bridge.adapter"; got != want {
		t.Fatalf("session.request.Capabilities.Provides[0] = %q, want %q", got, want)
	}
	if got, want := session.request.Methods.DaemonRequests[0], "ping"; got != want {
		t.Fatalf("session.request.Methods.DaemonRequests[0] = %q, want %q", got, want)
	}
	if got, want := session.request.Runtime.Bridge.ManagedInstances[0].Instance.ID, "brg-1"; got != want {
		t.Fatalf("session.request.Runtime.Bridge.ManagedInstances[0].Instance.ID = %q, want %q", got, want)
	}
	if got, want := session.response.AcceptedCapabilities.Provides[0], "bridge.adapter"; got != want {
		t.Fatalf("session.response.AcceptedCapabilities.Provides[0] = %q, want %q", got, want)
	}
	if got, want := session.response.ImplementedMethods[0], "bridges/deliver"; got != want {
		t.Fatalf("session.response.ImplementedMethods[0] = %q, want %q", got, want)
	}
}
