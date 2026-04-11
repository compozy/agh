package extensiontest

import (
	"context"
	"strings"
	"testing"
	"time"

	channelspkg "github.com/pedronauck/agh/internal/channels"
	extensionprotocol "github.com/pedronauck/agh/internal/extension/protocol"
	observepkg "github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/subprocess"
)

func TestValidateConformanceAcceptsHealthyOrderedReport(t *testing.T) {
	report := validConformanceReport()

	if err := ValidateConformance(report, ConformanceExpectation{
		InstanceID:          "chan-telegram-reference",
		ExtensionName:       "telegram-reference",
		BoundSecretNames:    []string{"bot_token"},
		RequireStateReport:  true,
		RequireDelivery:     true,
		ExpectedFinalStatus: channelspkg.ChannelStatusReady,
	}); err != nil {
		t.Fatalf("ValidateConformance() error = %v, want nil", err)
	}
}

func TestValidateConformanceFlagsMissingAck(t *testing.T) {
	report := validConformanceReport()
	report.Deliveries = []DeliveryRecord{
		{
			Request: testDeliveryRequest("delivery-1", 1, channelspkg.DeliveryEventTypeStart, false),
		},
	}

	assertConformanceIssue(t, report, ConformanceExpectation{RequireDelivery: true}, "missing_ack")
}

func TestValidateConformanceFlagsOutOfOrderDelivery(t *testing.T) {
	report := validConformanceReport()
	report.Deliveries = []DeliveryRecord{
		{
			Request: testDeliveryRequest("delivery-1", 1, channelspkg.DeliveryEventTypeStart, false),
			Ack:     testDeliveryAck("delivery-1", 1, "telegram:delivery-1:1", ""),
		},
		{
			Request: testDeliveryRequest("delivery-1", 1, channelspkg.DeliveryEventTypeDelta, false),
			Ack:     testDeliveryAck("delivery-1", 1, "telegram:delivery-1:1", ""),
		},
	}

	assertConformanceIssue(t, report, ConformanceExpectation{RequireDelivery: true}, "out_of_order_delivery")
}

func TestValidateConformanceFlagsMissingStateReporting(t *testing.T) {
	report := validConformanceReport()
	report.States = nil

	assertConformanceIssue(t, report, ConformanceExpectation{RequireStateReport: true}, "missing_state_report")
}

func TestHarnessHelperUtilities(t *testing.T) {
	driver := NewScriptedPromptDriver(time.Date(2026, 4, 11, 5, 15, 0, 0, time.UTC), nil)
	if err := driver.Cancel(context.Background(), nil); err != nil {
		t.Fatalf("ScriptedPromptDriver.Cancel() error = %v", err)
	}

	workspace := defaultResolvedWorkspace("/tmp/channel-adapter-workspace", time.Date(2026, 4, 11, 5, 15, 0, 0, time.UTC))
	resolver := staticWorkspaceResolver{resolved: workspace}
	resolved, err := resolver.ResolveOrRegister(context.Background(), "")
	if err != nil {
		t.Fatalf("staticWorkspaceResolver.ResolveOrRegister() error = %v", err)
	}
	if got, want := resolved.ID, workspace.ID; got != want {
		t.Fatalf("ResolveOrRegister().ID = %q, want %q", got, want)
	}

	var nilRuntimeResolver *stubChannelRuntimeResolver
	runtime, err := nilRuntimeResolver.ResolveChannelRuntime(context.Background(), "telegram-reference")
	if err != nil {
		t.Fatalf("stubChannelRuntimeResolver.ResolveChannelRuntime() error = %v", err)
	}
	if runtime != nil {
		t.Fatalf("ResolveChannelRuntime() = %#v, want nil", runtime)
	}

	sink := &deferredChannelTelemetrySink{}
	sink.RecordChannelAuthFailure("chan-1")
	sink.RecordChannelRuntimeIssue("chan-1", channelspkg.ChannelStatusError, "adapter failed")
	sink.ClearChannelRuntimeIssue("chan-1")

	observer, err := observepkg.New(
		context.Background(),
		observepkg.WithStartTime(time.Date(2026, 4, 11, 5, 15, 0, 0, time.UTC)),
	)
	if err != nil {
		t.Fatalf("observe.New() error = %v", err)
	}
	sink.observer = observer
	sink.RecordChannelAuthFailure("chan-1")
	sink.RecordChannelRuntimeIssue("chan-1", channelspkg.ChannelStatusError, "adapter failed")
	sink.ClearChannelRuntimeIssue("chan-1")

	source := harnessChannelSource{}
	instances, err := source.ListInstances(context.Background())
	if err != nil {
		t.Fatalf("harnessChannelSource.ListInstances() error = %v", err)
	}
	if instances != nil {
		t.Fatalf("harnessChannelSource.ListInstances() = %#v, want nil", instances)
	}
	routes, err := source.ListRoutes(context.Background(), "chan-1")
	if err != nil {
		t.Fatalf("harnessChannelSource.ListRoutes() error = %v", err)
	}
	if routes != nil {
		t.Fatalf("harnessChannelSource.ListRoutes() = %#v, want nil", routes)
	}
	if metrics := source.DeliveryMetrics(); metrics != nil {
		t.Fatalf("harnessChannelSource.DeliveryMetrics() = %#v, want nil", metrics)
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
					Provides: []string{"channel.adapter"},
					GrantedActions: []extensionprotocol.HostAPIMethod{
						extensionprotocol.HostAPIMethodChannelsMessagesIngest,
						extensionprotocol.HostAPIMethodChannelsInstancesGet,
						extensionprotocol.HostAPIMethodChannelsInstancesReportState,
					},
					GrantedSecurity: []string{"channel.read", "channel.write"},
				},
				Methods: subprocess.InitializeMethods{
					ExtensionServices: []string{"channels/deliver", "health_check", "shutdown"},
				},
				Runtime: subprocess.InitializeRuntime{
					HealthCheckIntervalMS: 30_000,
					HealthCheckTimeoutMS:  5_000,
					ShutdownTimeoutMS:     10_000,
					DefaultHookTimeoutMS:  5_000,
					Channel: &subprocess.InitializeChannelRuntime{
						Instance: testChannelInstance(),
						BoundSecrets: []subprocess.InitializeChannelBoundSecret{
							{BindingName: "bot_token", Kind: "token", Value: "telegram-token"},
						},
					},
				},
			},
		},
		Instance: &channelspkg.ChannelInstance{
			ID:            "chan-telegram-reference",
			Scope:         channelspkg.ScopeWorkspace,
			WorkspaceID:   "ws-telegram",
			Platform:      "telegram",
			ExtensionName: "telegram-reference",
			DisplayName:   "Telegram Reference",
			Enabled:       true,
			Status:        channelspkg.ChannelStatusReady,
			RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true, IncludeThread: true},
			CreatedAt:     time.Date(2026, 4, 11, 5, 0, 0, 0, time.UTC),
			UpdatedAt:     time.Date(2026, 4, 11, 5, 0, 0, 0, time.UTC),
		},
		States: []StateRecord{
			{
				Status:   channelspkg.ChannelStatusReady,
				Instance: testChannelInstance(),
			},
		},
		Deliveries: []DeliveryRecord{
			{
				Request: testDeliveryRequest("delivery-1", 1, channelspkg.DeliveryEventTypeStart, false),
				Ack:     testDeliveryAck("delivery-1", 1, "telegram:delivery-1:1", ""),
			},
			{
				Request: testDeliveryRequest("delivery-1", 2, channelspkg.DeliveryEventTypeDelta, false),
				Ack:     testDeliveryAck("delivery-1", 2, "telegram:delivery-1:2", "telegram:delivery-1:1"),
			},
			{
				Request: testDeliveryRequest("delivery-1", 3, channelspkg.DeliveryEventTypeFinal, true),
				Ack:     testDeliveryAck("delivery-1", 3, "telegram:delivery-1:3", "telegram:delivery-1:2"),
			},
		},
	}
}

func testChannelInstance() channelspkg.ChannelInstance {
	now := time.Date(2026, 4, 11, 5, 0, 0, 0, time.UTC)
	return channelspkg.ChannelInstance{
		ID:            "chan-telegram-reference",
		Scope:         channelspkg.ScopeWorkspace,
		WorkspaceID:   "ws-telegram",
		Platform:      "telegram",
		ExtensionName: "telegram-reference",
		DisplayName:   "Telegram Reference",
		Enabled:       true,
		Status:        channelspkg.ChannelStatusReady,
		RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true, IncludeThread: true},
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func testDeliveryRequest(deliveryID string, seq int64, eventType string, final bool) channelspkg.DeliveryRequest {
	return channelspkg.DeliveryRequest{
		Event: channelspkg.DeliveryEvent{
			DeliveryID:        deliveryID,
			ChannelInstanceID: "chan-telegram-reference",
			RoutingKey: channelspkg.RoutingKey{
				Scope:             channelspkg.ScopeWorkspace,
				WorkspaceID:       "ws-telegram",
				ChannelInstanceID: "chan-telegram-reference",
				PeerID:            "peer-1",
				ThreadID:          "thread-1",
			},
			DeliveryTarget: channelspkg.DeliveryTarget{
				ChannelInstanceID: "chan-telegram-reference",
				PeerID:            "peer-1",
				ThreadID:          "thread-1",
				Mode:              channelspkg.DeliveryModeReply,
			},
			Seq:       seq,
			EventType: eventType,
			Content:   channelspkg.MessageContent{Text: "hello"},
			Final:     final,
		},
	}
}

func testDeliveryAck(deliveryID string, seq int64, remoteID string, replaceID string) *channelspkg.DeliveryAck {
	return &channelspkg.DeliveryAck{
		DeliveryID:             deliveryID,
		Seq:                    seq,
		RemoteMessageID:        remoteID,
		ReplaceRemoteMessageID: replaceID,
	}
}
