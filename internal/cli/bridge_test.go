package cli

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	bridgepkg "github.com/pedronauck/agh/internal/bridges"
)

func TestBridgeListRendersScopePlatformAndStatusInHumanOutput(t *testing.T) {
	t.Parallel()

	deps := newTestDeps(t, &stubClient{
		listBridgesFn: func(context.Context) ([]BridgeRecord, error) {
			return []BridgeRecord{testBridgeRecord(t)}, nil
		},
	})

	stdout, _, err := executeRootCommand(t, deps, "bridge", "list", "-o", "human")
	if err != nil {
		t.Fatalf("bridge list human error = %v", err)
	}

	for _, token := range []string{"Bridges", "Platform", "Scope", "Status", "telegram", "workspace", "ready", "peer, thread"} {
		if !strings.Contains(stdout, token) {
			t.Fatalf("bridge list human output missing %q: %s", token, stdout)
		}
	}
}

func TestBridgeGetReturnsStructuredJSONOutput(t *testing.T) {
	t.Parallel()

	expected := testBridgeRecord(t)
	deps := newTestDeps(t, &stubClient{
		getBridgeFn: func(_ context.Context, id string) (BridgeRecord, error) {
			if id != expected.ID {
				t.Fatalf("GetBridge() id = %q, want %q", id, expected.ID)
			}
			return expected, nil
		},
	})

	stdout, _, err := executeRootCommand(t, deps, "bridge", "get", expected.ID, "-o", "json")
	if err != nil {
		t.Fatalf("bridge get json error = %v", err)
	}

	var decoded BridgeRecord
	if err := json.Unmarshal([]byte(stdout), &decoded); err != nil {
		t.Fatalf("json.Unmarshal(bridge get) error = %v", err)
	}
	if decoded.ID != expected.ID || decoded.Scope != expected.Scope || decoded.Status != expected.Status ||
		decoded.WorkspaceID != expected.WorkspaceID {
		t.Fatalf("decoded = %#v, want %#v", decoded, expected)
	}
}

func TestBridgeCreateBuildsSharedRequestAndDerivesDisabledStatus(t *testing.T) {
	t.Parallel()

	var captured CreateBridgeRequest
	deps := newTestDeps(t, &stubClient{
		createBridgeFn: func(_ context.Context, request CreateBridgeRequest) (BridgeRecord, error) {
			captured = request
			record := testBridgeRecord(t)
			record.Enabled = request.Enabled
			record.Status = bridgepkg.BridgeStatusStarting
			if !request.Enabled {
				record.Status = bridgepkg.BridgeStatusDisabled
			}
			record.Scope = request.Scope
			record.WorkspaceID = request.WorkspaceID
			record.Platform = request.Platform
			record.ExtensionName = request.ExtensionName
			record.DisplayName = request.DisplayName
			record.RoutingPolicy = request.RoutingPolicy
			record.DeliveryDefaults = json.RawMessage(request.DeliveryDefaults)
			return record, nil
		},
	})

	stdout, _, err := executeRootCommand(
		t,
		deps,
		"bridge", "create",
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
		t.Fatalf("bridge create error = %v", err)
	}

	if captured.Scope != bridgepkg.ScopeWorkspace || captured.WorkspaceID != "ws-alpha" {
		t.Fatalf("captured scope payload = %#v", captured)
	}
	if captured.Enabled {
		t.Fatalf("captured lifecycle enabled = %t, want false", captured.Enabled)
	}
	if !captured.RoutingPolicy.IncludePeer || !captured.RoutingPolicy.IncludeGroup ||
		captured.RoutingPolicy.IncludeThread {
		t.Fatalf("captured routing policy = %#v", captured.RoutingPolicy)
	}
	if string(captured.DeliveryDefaults) != `{"mode":"reply","group_id":"group-1"}` {
		t.Fatalf("captured delivery defaults = %s", string(captured.DeliveryDefaults))
	}

	var decoded BridgeRecord
	if err := json.Unmarshal([]byte(stdout), &decoded); err != nil {
		t.Fatalf("json.Unmarshal(bridge create) error = %v", err)
	}
	if decoded.Status != bridgepkg.BridgeStatusDisabled {
		t.Fatalf("decoded.Status = %q, want disabled", decoded.Status)
	}
}

func TestBridgeCreateRejectsWorkspaceScopeWithoutWorkspaceID(t *testing.T) {
	t.Parallel()

	deps := newTestDeps(t, &stubClient{
		createBridgeFn: func(context.Context, CreateBridgeRequest) (BridgeRecord, error) {
			t.Fatal("CreateBridge() should not be called when workspace scope is invalid")
			return BridgeRecord{}, nil
		},
	})

	_, _, err := executeRootCommand(
		t,
		deps,
		"bridge", "create",
		"--scope", "workspace",
		"--platform", "telegram",
		"--extension", "ext-telegram",
		"--display-name", "Support",
	)
	if err == nil || !strings.Contains(err.Error(), "--workspace-id is required when --scope=workspace") {
		t.Fatalf("bridge create error = %v, want missing workspace-id validation", err)
	}
}

func TestBridgeCreateRejectsOperationalStatusFlag(t *testing.T) {
	t.Parallel()

	t.Run("Should reject operational status flag", func(t *testing.T) {
		t.Parallel()

		deps := newTestDeps(t, &stubClient{
			createBridgeFn: func(context.Context, CreateBridgeRequest) (BridgeRecord, error) {
				t.Fatal("CreateBridge() should not be called when operational status flag is provided")
				return BridgeRecord{}, nil
			},
		})

		_, _, err := executeRootCommand(
			t,
			deps,
			"bridge", "create",
			"--scope", "global",
			"--platform", "telegram",
			"--extension", "ext-telegram",
			"--display-name", "Support",
			"--enabled=false",
			"--status", "ready",
		)
		if err == nil || !strings.Contains(err.Error(), "unknown flag: --status") {
			t.Fatalf("bridge create error = %v, want unknown status flag", err)
		}
	})
}

func TestBridgeUpdateMergesRoutingPolicyAndAllowsNullDeliveryDefaults(t *testing.T) {
	t.Parallel()

	current := testBridgeRecord(t)
	current.RoutingPolicy = bridgepkg.RoutingPolicy{
		IncludePeer:   true,
		IncludeThread: false,
		IncludeGroup:  true,
	}

	var (
		getCalls int
		captured UpdateBridgeRequest
		updateID string
	)
	deps := newTestDeps(t, &stubClient{
		getBridgeFn: func(_ context.Context, id string) (BridgeRecord, error) {
			getCalls++
			if id != current.ID {
				t.Fatalf("GetBridge() id = %q, want %q", id, current.ID)
			}
			return current, nil
		},
		updateBridgeFn: func(_ context.Context, id string, request UpdateBridgeRequest) (BridgeRecord, error) {
			updateID = id
			captured = request
			updated := current
			updated.DisplayName = *request.DisplayName
			updated.RoutingPolicy = *request.RoutingPolicy
			updated.DeliveryDefaults = json.RawMessage(*request.DeliveryDefaults)
			return updated, nil
		},
	})

	stdout, _, err := executeRootCommand(
		t,
		deps,
		"bridge", "update", current.ID,
		"--display-name", "Support Ops",
		"--include-thread",
		"--delivery-defaults", "null",
		"-o", "json",
	)
	if err != nil {
		t.Fatalf("bridge update error = %v", err)
	}

	if getCalls != 1 || updateID != current.ID {
		t.Fatalf("getCalls/updateID = %d/%q, want 1/%q", getCalls, updateID, current.ID)
	}
	if captured.DisplayName == nil || *captured.DisplayName != "Support Ops" {
		t.Fatalf("captured display name = %#v", captured.DisplayName)
	}
	if captured.RoutingPolicy == nil || !captured.RoutingPolicy.IncludePeer || !captured.RoutingPolicy.IncludeThread ||
		!captured.RoutingPolicy.IncludeGroup {
		t.Fatalf("captured routing policy = %#v", captured.RoutingPolicy)
	}
	if captured.DeliveryDefaults == nil || string(*captured.DeliveryDefaults) != "null" {
		t.Fatalf("captured delivery defaults = %#v", captured.DeliveryDefaults)
	}

	var decoded BridgeRecord
	if err := json.Unmarshal([]byte(stdout), &decoded); err != nil {
		t.Fatalf("json.Unmarshal(bridge update) error = %v", err)
	}
	if decoded.DisplayName != "Support Ops" || !decoded.RoutingPolicy.IncludeThread {
		t.Fatalf("decoded = %#v, want updated display name and thread routing", decoded)
	}
}

func TestBridgeLifecycleCommandsUseDaemonClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		args      []string
		status    bridgepkg.BridgeStatus
		enableFn  func(context.Context, string) (BridgeRecord, error)
		disableFn func(context.Context, string) (BridgeRecord, error)
		restartFn func(context.Context, string) (BridgeRecord, error)
	}{
		{
			name:   "enable",
			args:   []string{"bridge", "enable", "brg-1", "-o", "json"},
			status: bridgepkg.BridgeStatusStarting,
			enableFn: func(_ context.Context, id string) (BridgeRecord, error) {
				record := testBridgeRecord(t)
				record.ID = id
				record.Enabled = true
				record.Status = bridgepkg.BridgeStatusStarting
				return record, nil
			},
		},
		{
			name:   "disable",
			args:   []string{"bridge", "disable", "brg-1", "-o", "json"},
			status: bridgepkg.BridgeStatusDisabled,
			disableFn: func(_ context.Context, id string) (BridgeRecord, error) {
				record := testBridgeRecord(t)
				record.ID = id
				record.Enabled = false
				record.Status = bridgepkg.BridgeStatusDisabled
				return record, nil
			},
		},
		{
			name:   "restart",
			args:   []string{"bridge", "restart", "brg-1", "-o", "json"},
			status: bridgepkg.BridgeStatusStarting,
			restartFn: func(_ context.Context, id string) (BridgeRecord, error) {
				record := testBridgeRecord(t)
				record.ID = id
				record.Enabled = true
				record.Status = bridgepkg.BridgeStatusStarting
				return record, nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			deps := newTestDeps(t, &stubClient{
				enableBridgeFn:  tt.enableFn,
				disableBridgeFn: tt.disableFn,
				restartBridgeFn: tt.restartFn,
			})

			stdout, _, err := executeRootCommand(t, deps, tt.args...)
			if err != nil {
				t.Fatalf("executeRootCommand(%v) error = %v", tt.args, err)
			}

			var decoded BridgeRecord
			if err := json.Unmarshal([]byte(stdout), &decoded); err != nil {
				t.Fatalf("json.Unmarshal(lifecycle output) error = %v", err)
			}
			if decoded.Status != tt.status {
				t.Fatalf("decoded.Status = %q, want %q", decoded.Status, tt.status)
			}
		})
	}
}

func TestBridgeRoutesRenderPeerThreadAndGroupSeparately(t *testing.T) {
	t.Parallel()

	deps := newTestDeps(t, &stubClient{
		bridgeRoutesFn: func(_ context.Context, id string) ([]BridgeRouteRecord, error) {
			if id != "brg-1" {
				t.Fatalf("BridgeRoutes() id = %q, want brg-1", id)
			}
			return []BridgeRouteRecord{{
				RoutingKeyHash:   "hash-1",
				Scope:            bridgepkg.ScopeWorkspace,
				WorkspaceID:      "ws-alpha",
				BridgeInstanceID: "brg-1",
				PeerID:           "peer-1",
				ThreadID:         "thread-1",
				GroupID:          "group-1",
				SessionID:        "sess-1",
				AgentName:        "coder",
				LastActivityAt:   fixedTestNow,
				CreatedAt:        fixedTestNow,
				UpdatedAt:        fixedTestNow,
			}}, nil
		},
	})

	stdout, _, err := executeRootCommand(t, deps, "bridge", "routes", "brg-1", "-o", "human")
	if err != nil {
		t.Fatalf("bridge routes human error = %v", err)
	}

	for _, token := range []string{"Bridge Routes", "Peer", "Thread", "Group", "peer-1", "thread-1", "group-1", "sess-1"} {
		if !strings.Contains(stdout, token) {
			t.Fatalf("bridge routes human output missing %q: %s", token, stdout)
		}
	}
}

func TestBridgeTestDeliveryUsesTypedTargetPayload(t *testing.T) {
	t.Parallel()

	var (
		capturedID      string
		capturedRequest BridgeTestDeliveryRequest
	)
	deps := newTestDeps(t, &stubClient{
		testBridgeDeliveryFn: func(_ context.Context, id string, request BridgeTestDeliveryRequest) (BridgeTestDeliveryRecord, error) {
			capturedID = id
			capturedRequest = request
			return BridgeTestDeliveryRecord{
				Status:  "resolved",
				Message: request.Message,
				DeliveryTarget: DeliveryTargetRecord{
					BridgeInstanceID: id,
					PeerID:           request.Target.PeerID,
					ThreadID:         request.Target.ThreadID,
					GroupID:          request.Target.GroupID,
					Mode:             request.Target.Mode,
				},
			}, nil
		},
	})

	stdout, _, err := executeRootCommand(
		t,
		deps,
		"bridge", "test-delivery", "brg-1",
		"--message", "hello",
		"--peer-id", "peer-1",
		"--thread-id", "thread-1",
		"--group-id", "group-1",
		"--mode", "reply",
		"-o", "json",
	)
	if err != nil {
		t.Fatalf("bridge test-delivery error = %v", err)
	}

	if capturedID != "brg-1" {
		t.Fatalf("capturedID = %q, want brg-1", capturedID)
	}
	if capturedRequest.Message != "hello" || capturedRequest.Target.PeerID != "peer-1" ||
		capturedRequest.Target.ThreadID != "thread-1" ||
		capturedRequest.Target.GroupID != "group-1" ||
		capturedRequest.Target.Mode != bridgepkg.DeliveryModeReply {
		t.Fatalf("capturedRequest = %#v", capturedRequest)
	}

	var decoded BridgeTestDeliveryRecord
	if err := json.Unmarshal([]byte(stdout), &decoded); err != nil {
		t.Fatalf("json.Unmarshal(bridge test-delivery) error = %v", err)
	}
	if decoded.DeliveryTarget.ThreadID != "thread-1" || decoded.DeliveryTarget.Mode != bridgepkg.DeliveryModeReply {
		t.Fatalf("decoded = %#v, want typed delivery target", decoded)
	}
}

func TestBridgeBundleAndHelpers(t *testing.T) {
	t.Parallel()

	record := testBridgeRecord(t)
	bundle := bridgeBundle(record)

	human, err := bundle.human()
	if err != nil {
		t.Fatalf("bridgeBundle().human() error = %v", err)
	}
	if !strings.Contains(human, "Delivery Defaults") ||
		!strings.Contains(human, `{"mode":"reply","peer_id":"peer-default"}`) {
		t.Fatalf("bridgeBundle().human() = %q, want delivery defaults", human)
	}

	toon, err := bundle.toon()
	if err != nil {
		t.Fatalf("bridgeBundle().toon() error = %v", err)
	}
	if !strings.Contains(
		toon,
		"bridge{id,display_name,platform,extension_name,scope,workspace_id,enabled,status,routing,include_peer,include_thread,include_group,delivery_defaults,created_at,updated_at}:",
	) {
		t.Fatalf("bridgeBundle().toon() = %q, want bridge TOON object", toon)
	}

	if got := bridgeRoutingPolicyLabel(bridgepkg.RoutingPolicy{}); got != "" {
		t.Fatalf("bridgeRoutingPolicyLabel(empty) = %q, want empty string", got)
	}
	if _, err := parseRequiredBridgeJSON("{not-json"); err == nil {
		t.Fatal("parseRequiredBridgeJSON(invalid) error = nil, want non-nil")
	}
	if _, err := parseBridgeScope("bogus"); err == nil {
		t.Fatal("parseBridgeScope(bogus) error = nil, want non-nil")
	}
}

func TestParseRequiredBridgeJSONEnforcesObjectOrNull(t *testing.T) {
	t.Parallel()

	validObject, err := parseRequiredBridgeJSON(`{"mode":"reply"}`)
	if err != nil {
		t.Fatalf("parseRequiredBridgeJSON(object) error = %v", err)
	}
	if string(*validObject) != `{"mode":"reply"}` {
		t.Fatalf("parseRequiredBridgeJSON(object) = %s, want preserved object", string(*validObject))
	}

	validNull, err := parseRequiredBridgeJSON(`null`)
	if err != nil {
		t.Fatalf("parseRequiredBridgeJSON(null) error = %v", err)
	}
	if string(*validNull) != "null" {
		t.Fatalf("parseRequiredBridgeJSON(null) = %s, want null", string(*validNull))
	}

	for _, raw := range []string{`[]`, `"text"`, `123`} {
		if _, err := parseRequiredBridgeJSON(
			raw,
		); err == nil ||
			!strings.Contains(err.Error(), "must be a JSON object or null") {
			t.Fatalf("parseRequiredBridgeJSON(%s) error = %v, want object-or-null validation", raw, err)
		}
	}
}

func testBridgeRecord(t *testing.T) BridgeRecord {
	t.Helper()

	return BridgeRecord{
		ID:            "brg-1",
		Scope:         bridgepkg.ScopeWorkspace,
		WorkspaceID:   "ws-alpha",
		Platform:      "telegram",
		ExtensionName: "ext-telegram",
		DisplayName:   "Support",
		Enabled:       true,
		Status:        bridgepkg.BridgeStatusReady,
		RoutingPolicy: bridgepkg.RoutingPolicy{
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
