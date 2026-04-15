package extensiontest

import (
	"context"
	"strings"
	"testing"
	"time"

	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	extensionprotocol "github.com/pedronauck/agh/internal/extension/protocol"
	observepkg "github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/subprocess"
)

func TestValidateConformanceAcceptsHealthyOrderedReport(t *testing.T) {
	report := validConformanceReport()

	if err := ValidateConformance(report, ConformanceExpectation{
		Provider:                  "telegram-reference",
		Platform:                  "telegram",
		RequireOwnedInstanceList:  true,
		RequireOwnedInstanceFetch: true,
		RequireStateReport:        true,
		RequireDelivery:           true,
		ManagedInstances: []ManagedInstanceExpectation{{
			InstanceID:          "brg-telegram-reference",
			ExtensionName:       "telegram-reference",
			BoundSecretNames:    []string{"bot_token"},
			ExpectedFinalStatus: bridgepkg.BridgeStatusReady,
		}},
	}); err != nil {
		t.Fatalf("ValidateConformance() error = %v, want nil", err)
	}
}

func TestValidateConformanceFlagsMissingAck(t *testing.T) {
	report := validConformanceReport()
	report.Deliveries = []DeliveryRecord{
		{
			Request: testDeliveryRequest("delivery-1", 1, bridgepkg.DeliveryEventTypeStart, false),
		},
	}

	assertConformanceIssue(t, report, ConformanceExpectation{RequireDelivery: true}, "missing_ack")
}

func TestValidateConformanceFlagsOutOfOrderDelivery(t *testing.T) {
	report := validConformanceReport()
	report.Deliveries = []DeliveryRecord{
		{
			Request: testDeliveryRequest("delivery-1", 1, bridgepkg.DeliveryEventTypeStart, false),
			Ack:     testDeliveryAck("delivery-1", 1, "telegram:delivery-1:1", ""),
		},
		{
			Request: testDeliveryRequest("delivery-1", 1, bridgepkg.DeliveryEventTypeDelta, false),
			Ack:     testDeliveryAck("delivery-1", 1, "telegram:delivery-1:1", ""),
		},
	}

	assertConformanceIssue(t, report, ConformanceExpectation{RequireDelivery: true}, "out_of_order_delivery")
}

func TestValidateConformanceFlagsMissingStateReporting(t *testing.T) {
	report := validConformanceReport()
	report.States = nil

	assertConformanceIssue(t, report, ConformanceExpectation{
		RequireStateReport: true,
		ManagedInstances: []ManagedInstanceExpectation{{
			InstanceID: "brg-telegram-reference",
		}},
	}, "missing_state_report")
}

func TestValidateConformanceRejectsMissingProviderScopedBridgeContext(t *testing.T) {
	report := validConformanceReport()
	report.Handshake.Request.Runtime.Bridge = nil

	assertConformanceIssue(t, report, ConformanceExpectation{
		ManagedInstances: []ManagedInstanceExpectation{{
			InstanceID: "brg-telegram-reference",
		}},
	}, "missing_bridge_runtime")
}

func TestValidateConformanceRejectsUnexpectedOwnedInstanceDelivery(t *testing.T) {
	report := validConformanceReport()
	report.Deliveries[0].Request.Event.BridgeInstanceID = "brg-unowned"
	report.Deliveries[0].Request.Event.DeliveryTarget.BridgeInstanceID = "brg-unowned"

	assertConformanceIssue(t, report, ConformanceExpectation{
		RequireDelivery: true,
		ManagedInstances: []ManagedInstanceExpectation{{
			InstanceID: "brg-telegram-reference",
		}},
	}, "unexpected_delivery_instance")
}

func TestHarnessHelperCloningAndMarkerParsingSupportManyManagedInstances(t *testing.T) {
	managed := []subprocess.InitializeBridgeManagedInstance{
		{
			Instance:     testBridgeInstanceWithID("brg-1"),
			BoundSecrets: []subprocess.InitializeBridgeBoundSecret{{BindingName: "bot_token", Kind: "token", Value: "token-1"}},
		},
		{
			Instance:     testBridgeInstanceWithID("brg-2"),
			BoundSecrets: []subprocess.InitializeBridgeBoundSecret{{BindingName: "bot_token", Kind: "token", Value: "token-2"}},
		},
	}
	cloned := cloneManagedRuntime(managed)
	cloned[0].Instance.ID = "brg-mutated"
	cloned[0].BoundSecrets[0].Value = "mutated"

	if got, want := managed[0].Instance.ID, "brg-1"; got != want {
		t.Fatalf("managed[0].Instance.ID = %q, want %q", got, want)
	}
	if got, want := managed[0].BoundSecrets[0].Value, "token-1"; got != want {
		t.Fatalf("managed[0].BoundSecrets[0].Value = %q, want %q", got, want)
	}

	root := t.TempDir()
	ownershipPath := root + "/ownership.json"
	appendJSONLine(t, ownershipPath, OwnershipRecord{
		Listed:  []bridgepkg.BridgeInstance{testBridgeInstanceWithID("brg-1"), testBridgeInstanceWithID("brg-2")},
		Fetched: []bridgepkg.BridgeInstance{testBridgeInstanceWithID("brg-1"), testBridgeInstanceWithID("brg-2")},
	})
	records, err := readJSONLinesFile[OwnershipRecord](ownershipPath)
	if err != nil {
		t.Fatalf("readJSONLinesFile(ownership) error = %v", err)
	}
	if got, want := len(records), 1; got != want {
		t.Fatalf("len(records) = %d, want %d", got, want)
	}
	if got, want := len(records[0].Fetched), 2; got != want {
		t.Fatalf("len(records[0].Fetched) = %d, want %d", got, want)
	}
}

func TestHarnessHelperUtilities(t *testing.T) {
	driver := NewScriptedPromptDriver(time.Date(2026, 4, 11, 5, 15, 0, 0, time.UTC), nil)
	if err := driver.Cancel(context.Background(), nil); err != nil {
		t.Fatalf("ScriptedPromptDriver.Cancel() error = %v", err)
	}

	workspace := defaultResolvedWorkspace("/tmp/bridge-adapter-workspace", time.Date(2026, 4, 11, 5, 15, 0, 0, time.UTC))
	resolver := staticWorkspaceResolver{resolved: workspace}
	resolved, err := resolver.ResolveOrRegister(context.Background(), "")
	if err != nil {
		t.Fatalf("staticWorkspaceResolver.ResolveOrRegister() error = %v", err)
	}
	if got, want := resolved.ID, workspace.ID; got != want {
		t.Fatalf("ResolveOrRegister().ID = %q, want %q", got, want)
	}

	var nilRuntimeResolver *stubBridgeRuntimeResolver
	runtime, err := nilRuntimeResolver.ResolveBridgeRuntime(context.Background(), "telegram-reference")
	if err != nil {
		t.Fatalf("stubBridgeRuntimeResolver.ResolveBridgeRuntime() error = %v", err)
	}
	if runtime != nil {
		t.Fatalf("ResolveBridgeRuntime() = %#v, want nil", runtime)
	}

	sink := &deferredBridgeTelemetrySink{}
	sink.RecordBridgeAuthFailure("brg-1")
	sink.RecordBridgeRuntimeIssue("brg-1", bridgepkg.BridgeStatusError, "adapter failed")
	sink.ClearBridgeRuntimeIssue("brg-1")

	observer, err := observepkg.New(
		context.Background(),
		observepkg.WithStartTime(time.Date(2026, 4, 11, 5, 15, 0, 0, time.UTC)),
	)
	if err != nil {
		t.Fatalf("observe.New() error = %v", err)
	}
	sink.observer = observer
	sink.RecordBridgeAuthFailure("brg-1")
	sink.RecordBridgeRuntimeIssue("brg-1", bridgepkg.BridgeStatusError, "adapter failed")
	sink.ClearBridgeRuntimeIssue("brg-1")

	source := harnessBridgeSource{}
	instances, err := source.ListInstances(context.Background())
	if err != nil {
		t.Fatalf("harnessBridgeSource.ListInstances() error = %v", err)
	}
	if instances != nil {
		t.Fatalf("harnessBridgeSource.ListInstances() = %#v, want nil", instances)
	}
	routes, err := source.ListRoutes(context.Background(), "brg-1")
	if err != nil {
		t.Fatalf("harnessBridgeSource.ListRoutes() error = %v", err)
	}
	if routes != nil {
		t.Fatalf("harnessBridgeSource.ListRoutes() = %#v, want nil", routes)
	}
	if metrics := source.DeliveryMetrics(); metrics != nil {
		t.Fatalf("harnessBridgeSource.DeliveryMetrics() = %#v, want nil", metrics)
	}
}

func assertConformanceIssue(
	t *testing.T,
	report ConformanceReport,
	expect ConformanceExpectation,
	code string,
) {
	t.Helper()

	err := ValidateConformance(report, expect)
	if err == nil {
		t.Fatalf("ValidateConformance() error = nil, want %q", code)
	}

	var confErr *ConformanceError
	if !strings.Contains(err.Error(), code) {
		t.Fatalf("ValidateConformance() error = %v, want code %q", err, code)
	}
	if !asConformanceError(err, &confErr) {
		t.Fatalf("ValidateConformance() error type = %T, want *ConformanceError", err)
	}
}

func asConformanceError(err error, target **ConformanceError) bool {
	if err == nil {
		return false
	}
	confErr, ok := err.(*ConformanceError)
	if !ok {
		return false
	}
	*target = confErr
	return true
}

func validConformanceReport() ConformanceReport {
	return ConformanceReport{
		Handshake: &HandshakeRecord{
			Request: subprocess.InitializeRequest{
				Capabilities: subprocess.InitializeCapabilities{
					Provides: []string{"bridge.adapter"},
					GrantedActions: []extensionprotocol.HostAPIMethod{
						extensionprotocol.HostAPIMethodBridgesInstancesList,
						extensionprotocol.HostAPIMethodBridgesMessagesIngest,
						extensionprotocol.HostAPIMethodBridgesInstancesGet,
						extensionprotocol.HostAPIMethodBridgesInstancesReportState,
					},
					GrantedSecurity: []string{"bridge.read", "bridge.write"},
				},
				Methods: subprocess.InitializeMethods{
					ExtensionServices: []string{"bridges/deliver", "health_check", "shutdown"},
				},
				Runtime: subprocess.InitializeRuntime{
					HealthCheckIntervalMS: 30_000,
					HealthCheckTimeoutMS:  5_000,
					ShutdownTimeoutMS:     10_000,
					DefaultHookTimeoutMS:  5_000,
					Bridge: &subprocess.InitializeBridgeRuntime{
						RuntimeVersion: subprocess.InitializeBridgeRuntimeVersion1,
						Provider:       "telegram-reference",
						Platform:       "telegram",
						ManagedInstances: []subprocess.InitializeBridgeManagedInstance{{
							Instance: testBridgeInstance(),
							BoundSecrets: []subprocess.InitializeBridgeBoundSecret{
								{BindingName: "bot_token", Kind: "token", Value: "telegram-token"},
							},
						}},
					},
				},
			},
		},
		Ownership: &OwnershipRecord{
			Listed:  []bridgepkg.BridgeInstance{testBridgeInstance()},
			Fetched: []bridgepkg.BridgeInstance{testBridgeInstance()},
		},
		States: []StateRecord{
			{
				BridgeInstanceID: "brg-telegram-reference",
				Status:           bridgepkg.BridgeStatusReady,
				Instance:         testBridgeInstance(),
			},
		},
		Deliveries: []DeliveryRecord{
			{
				Request: testDeliveryRequest("delivery-1", 1, bridgepkg.DeliveryEventTypeStart, false),
				Ack:     testDeliveryAck("delivery-1", 1, "telegram:delivery-1:1", ""),
			},
			{
				Request: testDeliveryRequest("delivery-1", 2, bridgepkg.DeliveryEventTypeDelta, false),
				Ack:     testDeliveryAck("delivery-1", 2, "telegram:delivery-1:2", "telegram:delivery-1:1"),
			},
			{
				Request: testDeliveryRequest("delivery-1", 3, bridgepkg.DeliveryEventTypeFinal, true),
				Ack:     testDeliveryAck("delivery-1", 3, "telegram:delivery-1:3", "telegram:delivery-1:2"),
			},
		},
	}
}

func testBridgeInstance() bridgepkg.BridgeInstance {
	return testBridgeInstanceWithID("brg-telegram-reference")
}

func testBridgeInstanceWithID(instanceID string) bridgepkg.BridgeInstance {
	now := time.Date(2026, 4, 11, 5, 0, 0, 0, time.UTC)
	return bridgepkg.BridgeInstance{
		ID:            instanceID,
		Scope:         bridgepkg.ScopeWorkspace,
		WorkspaceID:   "ws-telegram",
		Platform:      "telegram",
		ExtensionName: "telegram-reference",
		DisplayName:   "Telegram Reference",
		Enabled:       true,
		Status:        bridgepkg.BridgeStatusReady,
		RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true, IncludeThread: true},
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func testDeliveryRequest(deliveryID string, seq int64, eventType string, final bool) bridgepkg.DeliveryRequest {
	return bridgepkg.DeliveryRequest{
		Event: bridgepkg.DeliveryEvent{
			DeliveryID:       deliveryID,
			BridgeInstanceID: "brg-telegram-reference",
			RoutingKey: bridgepkg.RoutingKey{
				Scope:            bridgepkg.ScopeWorkspace,
				WorkspaceID:      "ws-telegram",
				BridgeInstanceID: "brg-telegram-reference",
				PeerID:           "peer-1",
				ThreadID:         "thread-1",
			},
			DeliveryTarget: bridgepkg.DeliveryTarget{
				BridgeInstanceID: "brg-telegram-reference",
				PeerID:           "peer-1",
				ThreadID:         "thread-1",
				Mode:             bridgepkg.DeliveryModeReply,
			},
			Seq:       seq,
			EventType: eventType,
			Content:   bridgepkg.MessageContent{Text: "hello"},
			Final:     final,
		},
	}
}

func testDeliveryAck(deliveryID string, seq int64, remoteID string, replaceID string) *bridgepkg.DeliveryAck {
	return &bridgepkg.DeliveryAck{
		DeliveryID:             deliveryID,
		Seq:                    seq,
		RemoteMessageID:        remoteID,
		ReplaceRemoteMessageID: replaceID,
	}
}
