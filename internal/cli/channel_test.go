package cli

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	channelspkg "github.com/pedronauck/agh/internal/channels"
)

func TestChannelListRendersScopePlatformAndStatusInHumanOutput(t *testing.T) {
	t.Parallel()

	deps := newTestDeps(t, stubClient{
		listChannelsFn: func(context.Context) ([]ChannelRecord, error) {
			return []ChannelRecord{testChannelRecord(t)}, nil
		},
	})

	stdout, _, err := executeRootCommand(t, deps, "channel", "list", "-o", "human")
	if err != nil {
		t.Fatalf("channel list human error = %v", err)
	}

	for _, token := range []string{"Channels", "Platform", "Scope", "Status", "telegram", "workspace", "ready", "peer, thread"} {
		if !strings.Contains(stdout, token) {
			t.Fatalf("channel list human output missing %q: %s", token, stdout)
		}
	}
}

func TestChannelGetReturnsStructuredJSONOutput(t *testing.T) {
	t.Parallel()

	expected := testChannelRecord(t)
	deps := newTestDeps(t, stubClient{
		getChannelFn: func(_ context.Context, id string) (ChannelRecord, error) {
			if id != expected.ID {
				t.Fatalf("GetChannel() id = %q, want %q", id, expected.ID)
			}
			return expected, nil
		},
	})

	stdout, _, err := executeRootCommand(t, deps, "channel", "get", expected.ID, "-o", "json")
	if err != nil {
		t.Fatalf("channel get json error = %v", err)
	}

	var decoded ChannelRecord
	if err := json.Unmarshal([]byte(stdout), &decoded); err != nil {
		t.Fatalf("json.Unmarshal(channel get) error = %v", err)
	}
	if decoded.ID != expected.ID || decoded.Scope != expected.Scope || decoded.Status != expected.Status || decoded.WorkspaceID != expected.WorkspaceID {
		t.Fatalf("decoded = %#v, want %#v", decoded, expected)
	}
}

func TestChannelCreateBuildsSharedRequestAndDerivesDisabledStatus(t *testing.T) {
	t.Parallel()

	var captured CreateChannelRequest
	deps := newTestDeps(t, stubClient{
		createChannelFn: func(_ context.Context, request CreateChannelRequest) (ChannelRecord, error) {
			captured = request
			record := testChannelRecord(t)
			record.Enabled = request.Enabled
			record.Status = request.Status
			record.Scope = request.Scope
			record.WorkspaceID = request.WorkspaceID
			record.Platform = request.Platform
			record.ExtensionName = request.ExtensionName
			record.DisplayName = request.DisplayName
			record.RoutingPolicy = request.RoutingPolicy
			record.DeliveryDefaults = request.DeliveryDefaults
			return record, nil
		},
	})

	stdout, _, err := executeRootCommand(
		t,
		deps,
		"channel", "create",
		"--scope", "workspace",
		"--workspace-id", "ws-alpha",
		"--platform", "telegram",
		"--extension", "ext-telegram",
		"--display-name", "Support",
		"--enabled=false",
		"--include-peer",
		"--include-group",
		"--delivery-defaults", `{"mode":"reply","group_id":"group-1"}`,
		"-o", "json",
	)
	if err != nil {
		t.Fatalf("channel create error = %v", err)
	}

	if captured.Scope != channelspkg.ScopeWorkspace || captured.WorkspaceID != "ws-alpha" {
		t.Fatalf("captured scope payload = %#v", captured)
	}
	if captured.Status != channelspkg.ChannelStatusDisabled || captured.Enabled {
		t.Fatalf("captured lifecycle = enabled:%t status:%q, want false/disabled", captured.Enabled, captured.Status)
	}
	if !captured.RoutingPolicy.IncludePeer || !captured.RoutingPolicy.IncludeGroup || captured.RoutingPolicy.IncludeThread {
		t.Fatalf("captured routing policy = %#v", captured.RoutingPolicy)
	}
	if string(captured.DeliveryDefaults) != `{"mode":"reply","group_id":"group-1"}` {
		t.Fatalf("captured delivery defaults = %s", string(captured.DeliveryDefaults))
	}

	var decoded ChannelRecord
	if err := json.Unmarshal([]byte(stdout), &decoded); err != nil {
		t.Fatalf("json.Unmarshal(channel create) error = %v", err)
	}
	if decoded.Status != channelspkg.ChannelStatusDisabled {
		t.Fatalf("decoded.Status = %q, want disabled", decoded.Status)
	}
}

func TestChannelUpdateMergesRoutingPolicyAndAllowsNullDeliveryDefaults(t *testing.T) {
	t.Parallel()

	current := testChannelRecord(t)
	current.RoutingPolicy = channelspkg.RoutingPolicy{
		IncludePeer:   true,
		IncludeThread: false,
		IncludeGroup:  true,
	}

	var (
		getCalls int
		captured UpdateChannelRequest
		updateID string
	)
	deps := newTestDeps(t, stubClient{
		getChannelFn: func(_ context.Context, id string) (ChannelRecord, error) {
			getCalls++
			if id != current.ID {
				t.Fatalf("GetChannel() id = %q, want %q", id, current.ID)
			}
			return current, nil
		},
		updateChannelFn: func(_ context.Context, id string, request UpdateChannelRequest) (ChannelRecord, error) {
			updateID = id
			captured = request
			updated := current
			updated.DisplayName = *request.DisplayName
			updated.RoutingPolicy = *request.RoutingPolicy
			updated.DeliveryDefaults = *request.DeliveryDefaults
			return updated, nil
		},
	})

	stdout, _, err := executeRootCommand(
		t,
		deps,
		"channel", "update", current.ID,
		"--display-name", "Support Ops",
		"--include-thread",
		"--delivery-defaults", "null",
		"-o", "json",
	)
	if err != nil {
		t.Fatalf("channel update error = %v", err)
	}

	if getCalls != 1 || updateID != current.ID {
		t.Fatalf("getCalls/updateID = %d/%q, want 1/%q", getCalls, updateID, current.ID)
	}
	if captured.DisplayName == nil || *captured.DisplayName != "Support Ops" {
		t.Fatalf("captured display name = %#v", captured.DisplayName)
	}
	if captured.RoutingPolicy == nil || !captured.RoutingPolicy.IncludePeer || !captured.RoutingPolicy.IncludeThread || !captured.RoutingPolicy.IncludeGroup {
		t.Fatalf("captured routing policy = %#v", captured.RoutingPolicy)
	}
	if captured.DeliveryDefaults == nil || string(*captured.DeliveryDefaults) != "null" {
		t.Fatalf("captured delivery defaults = %#v", captured.DeliveryDefaults)
	}

	var decoded ChannelRecord
	if err := json.Unmarshal([]byte(stdout), &decoded); err != nil {
		t.Fatalf("json.Unmarshal(channel update) error = %v", err)
	}
	if decoded.DisplayName != "Support Ops" || !decoded.RoutingPolicy.IncludeThread {
		t.Fatalf("decoded = %#v, want updated display name and thread routing", decoded)
	}
}

func TestChannelLifecycleCommandsUseDaemonClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		args      []string
		status    channelspkg.ChannelStatus
		enableFn  func(context.Context, string) (ChannelRecord, error)
		disableFn func(context.Context, string) (ChannelRecord, error)
		restartFn func(context.Context, string) (ChannelRecord, error)
	}{
		{
			name:   "enable",
			args:   []string{"channel", "enable", "chan-1", "-o", "json"},
			status: channelspkg.ChannelStatusStarting,
			enableFn: func(_ context.Context, id string) (ChannelRecord, error) {
				record := testChannelRecord(t)
				record.ID = id
				record.Enabled = true
				record.Status = channelspkg.ChannelStatusStarting
				return record, nil
			},
		},
		{
			name:   "disable",
			args:   []string{"channel", "disable", "chan-1", "-o", "json"},
			status: channelspkg.ChannelStatusDisabled,
			disableFn: func(_ context.Context, id string) (ChannelRecord, error) {
				record := testChannelRecord(t)
				record.ID = id
				record.Enabled = false
				record.Status = channelspkg.ChannelStatusDisabled
				return record, nil
			},
		},
		{
			name:   "restart",
			args:   []string{"channel", "restart", "chan-1", "-o", "json"},
			status: channelspkg.ChannelStatusStarting,
			restartFn: func(_ context.Context, id string) (ChannelRecord, error) {
				record := testChannelRecord(t)
				record.ID = id
				record.Enabled = true
				record.Status = channelspkg.ChannelStatusStarting
				return record, nil
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			deps := newTestDeps(t, stubClient{
				enableChannelFn:  tt.enableFn,
				disableChannelFn: tt.disableFn,
				restartChannelFn: tt.restartFn,
			})

			stdout, _, err := executeRootCommand(t, deps, tt.args...)
			if err != nil {
				t.Fatalf("executeRootCommand(%v) error = %v", tt.args, err)
			}

			var decoded ChannelRecord
			if err := json.Unmarshal([]byte(stdout), &decoded); err != nil {
				t.Fatalf("json.Unmarshal(lifecycle output) error = %v", err)
			}
			if decoded.Status != tt.status {
				t.Fatalf("decoded.Status = %q, want %q", decoded.Status, tt.status)
			}
		})
	}
}

func TestChannelRoutesRenderPeerThreadAndGroupSeparately(t *testing.T) {
	t.Parallel()

	deps := newTestDeps(t, stubClient{
		channelRoutesFn: func(_ context.Context, id string) ([]ChannelRouteRecord, error) {
			if id != "chan-1" {
				t.Fatalf("ChannelRoutes() id = %q, want chan-1", id)
			}
			return []ChannelRouteRecord{{
				RoutingKeyHash:    "hash-1",
				Scope:             channelspkg.ScopeWorkspace,
				WorkspaceID:       "ws-alpha",
				ChannelInstanceID: "chan-1",
				PeerID:            "peer-1",
				ThreadID:          "thread-1",
				GroupID:           "group-1",
				SessionID:         "sess-1",
				AgentName:         "coder",
				LastActivityAt:    fixedTestNow,
				CreatedAt:         fixedTestNow,
				UpdatedAt:         fixedTestNow,
			}}, nil
		},
	})

	stdout, _, err := executeRootCommand(t, deps, "channel", "routes", "chan-1", "-o", "human")
	if err != nil {
		t.Fatalf("channel routes human error = %v", err)
	}

	for _, token := range []string{"Channel Routes", "Peer", "Thread", "Group", "peer-1", "thread-1", "group-1", "sess-1"} {
		if !strings.Contains(stdout, token) {
			t.Fatalf("channel routes human output missing %q: %s", token, stdout)
		}
	}
}

func TestChannelTestDeliveryUsesTypedTargetPayload(t *testing.T) {
	t.Parallel()

	var (
		capturedID      string
		capturedRequest ChannelTestDeliveryRequest
	)
	deps := newTestDeps(t, stubClient{
		testChannelDeliveryFn: func(_ context.Context, id string, request ChannelTestDeliveryRequest) (ChannelTestDeliveryRecord, error) {
			capturedID = id
			capturedRequest = request
			return ChannelTestDeliveryRecord{
				Status:  "resolved",
				Message: request.Message,
				DeliveryTarget: DeliveryTargetRecord{
					ChannelInstanceID: id,
					PeerID:            request.Target.PeerID,
					ThreadID:          request.Target.ThreadID,
					GroupID:           request.Target.GroupID,
					Mode:              request.Target.Mode,
				},
			}, nil
		},
	})

	stdout, _, err := executeRootCommand(
		t,
		deps,
		"channel", "test-delivery", "chan-1",
		"--message", "hello",
		"--peer-id", "peer-1",
		"--thread-id", "thread-1",
		"--group-id", "group-1",
		"--mode", "reply",
		"-o", "json",
	)
	if err != nil {
		t.Fatalf("channel test-delivery error = %v", err)
	}

	if capturedID != "chan-1" {
		t.Fatalf("capturedID = %q, want chan-1", capturedID)
	}
	if capturedRequest.Message != "hello" || capturedRequest.Target.PeerID != "peer-1" || capturedRequest.Target.ThreadID != "thread-1" || capturedRequest.Target.GroupID != "group-1" || capturedRequest.Target.Mode != channelspkg.DeliveryModeReply {
		t.Fatalf("capturedRequest = %#v", capturedRequest)
	}

	var decoded ChannelTestDeliveryRecord
	if err := json.Unmarshal([]byte(stdout), &decoded); err != nil {
		t.Fatalf("json.Unmarshal(channel test-delivery) error = %v", err)
	}
	if decoded.DeliveryTarget.ThreadID != "thread-1" || decoded.DeliveryTarget.Mode != channelspkg.DeliveryModeReply {
		t.Fatalf("decoded = %#v, want typed delivery target", decoded)
	}
}

func TestChannelBundleAndHelpers(t *testing.T) {
	t.Parallel()

	record := testChannelRecord(t)
	bundle := channelBundle(record)

	human, err := bundle.human()
	if err != nil {
		t.Fatalf("channelBundle().human() error = %v", err)
	}
	if !strings.Contains(human, "Delivery Defaults") || !strings.Contains(human, `{"mode":"reply","peer_id":"peer-default"}`) {
		t.Fatalf("channelBundle().human() = %q, want delivery defaults", human)
	}

	toon, err := bundle.toon()
	if err != nil {
		t.Fatalf("channelBundle().toon() error = %v", err)
	}
	if !strings.Contains(toon, "channel{id,display_name,platform,extension_name,scope,workspace_id,enabled,status,routing,include_peer,include_thread,include_group,delivery_defaults,created_at,updated_at}:") {
		t.Fatalf("channelBundle().toon() = %q, want channel TOON object", toon)
	}

	if got := channelRoutingPolicyLabel(channelspkg.RoutingPolicy{}); got != "" {
		t.Fatalf("channelRoutingPolicyLabel(empty) = %q, want empty string", got)
	}
	if _, err := parseRequiredChannelJSON("{not-json"); err == nil {
		t.Fatal("parseRequiredChannelJSON(invalid) error = nil, want non-nil")
	}
	if _, err := parseChannelScope("bogus"); err == nil {
		t.Fatal("parseChannelScope(bogus) error = nil, want non-nil")
	}
}

func testChannelRecord(t *testing.T) ChannelRecord {
	t.Helper()

	return ChannelRecord{
		ID:            "chan-1",
		Scope:         channelspkg.ScopeWorkspace,
		WorkspaceID:   "ws-alpha",
		Platform:      "telegram",
		ExtensionName: "ext-telegram",
		DisplayName:   "Support",
		Enabled:       true,
		Status:        channelspkg.ChannelStatusReady,
		RoutingPolicy: channelspkg.RoutingPolicy{
			IncludePeer:   true,
			IncludeThread: true,
		},
		DeliveryDefaults: mustJSON(t, map[string]string{
			"mode":    "reply",
			"peer_id": "peer-default",
		}),
		CreatedAt: fixedTestNow.Add(-time.Hour),
		UpdatedAt: fixedTestNow,
	}
}
