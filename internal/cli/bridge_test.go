package cli

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/compozy/agh/internal/api/contract"
	bridgepkg "github.com/compozy/agh/internal/bridges"
)

func TestBridgeListRendersScopePlatformAndStatusInHumanOutput(t *testing.T) {
	t.Parallel()

	deps := newTestDeps(t, &stubClient{
		listBridgesFn: func(context.Context, BridgeListQuery) (BridgeListRecord, error) {
			return testBridgeListRecord(t), nil
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

func TestBridgeListForwardsCatalogQueryAndPreservesPageMetadata(t *testing.T) {
	t.Parallel()

	t.Run("Should forward every catalog filter and return the counted response", func(t *testing.T) {
		t.Parallel()

		var captured BridgeListQuery
		result := testBridgeListRecord(t)
		result.Page = contract.CountedCursorPagePayload{
			NextCursor: "bridge-cursor",
			HasMore:    true,
			Total:      75,
			Limit:      25,
		}
		deps := newTestDeps(t, &stubClient{
			listBridgesFn: func(_ context.Context, query BridgeListQuery) (BridgeListRecord, error) {
				captured = query
				return result, nil
			},
		})

		stdout, _, err := executeRootCommand(
			t,
			deps,
			"bridge", "list",
			"--scope", "all",
			"--workspace-id", "ws-alpha",
			"--q", "needle",
			"--platform", "telegram",
			"--status", "ready",
			"--sort", "name",
			"--cursor", "bridge-cursor-in",
			"--limit", "25",
			"-o", "json",
		)
		if err != nil {
			t.Fatalf("bridge list json error = %v", err)
		}
		want := BridgeListQuery{
			Scope:       "all",
			WorkspaceID: "ws-alpha",
			Search:      "needle",
			Platform:    "telegram",
			Status:      "ready",
			Sort:        "name",
			Cursor:      "bridge-cursor-in",
			Limit:       25,
		}
		if captured != want {
			t.Fatalf("ListBridges() query = %#v, want %#v", captured, want)
		}
		var decoded BridgeListRecord
		if err := json.Unmarshal([]byte(stdout), &decoded); err != nil {
			t.Fatalf("json.Unmarshal(bridge list) error = %v", err)
		}
		if decoded.Page != result.Page || len(decoded.Bridges) != 1 {
			t.Fatalf("decoded bridge list = %#v, want counted response %#v", decoded, result)
		}
	})

	t.Run("Should reject ambiguous workspace filters before calling the daemon", func(t *testing.T) {
		t.Parallel()

		deps := newTestDeps(t, &stubClient{
			listBridgesFn: func(context.Context, BridgeListQuery) (BridgeListRecord, error) {
				t.Fatal("ListBridges() should not be called for ambiguous workspace filters")
				return BridgeListRecord{}, nil
			},
		})
		_, _, err := executeRootCommand(
			t,
			deps,
			"bridge", "list",
			"--workspace-id", "ws-alpha",
			"--workspace", "alpha",
		)
		if err == nil || !strings.Contains(err.Error(), "cannot be combined") {
			t.Fatalf("bridge list ambiguous workspace error = %v, want validation", err)
		}
	})

	for _, output := range []string{"human", "toon", "jsonl"} {
		t.Run("Should preserve the continuation cursor in "+output+" output", func(t *testing.T) {
			t.Parallel()

			result := testBridgeListRecord(t)
			result.Page = contract.CountedCursorPagePayload{
				NextCursor: "bridge-cursor",
				HasMore:    true,
				Total:      75,
				Limit:      25,
			}
			deps := newTestDeps(t, &stubClient{
				listBridgesFn: func(context.Context, BridgeListQuery) (BridgeListRecord, error) {
					return result, nil
				},
			})
			stdout, _, err := executeRootCommand(t, deps, "bridge", "list", "-o", output)
			if err != nil {
				t.Fatalf("bridge list %s error = %v", output, err)
			}
			if !strings.Contains(stdout, "bridge-cursor") {
				t.Fatalf("bridge list %s output lost next cursor: %s", output, stdout)
			}
			if output == "jsonl" && !strings.Contains(stdout, `"type":"page"`) {
				t.Fatalf("bridge list jsonl output missing page record: %s", stdout)
			}
		})
	}

	t.Run("Should preserve persisted and effective health status in typed JSONL items", func(t *testing.T) {
		t.Parallel()

		result := testBridgeListRecord(t)
		bridgeID := result.Bridges[0].ID
		result.Bridges[0].Status = bridgepkg.BridgeStatusReady
		result.BridgeHealth[bridgeID] = contract.BridgeHealthPayload{
			BridgeInstanceID: bridgeID,
			Status:           bridgepkg.BridgeStatusError,
			LastError:        "adapter crashed",
		}
		deps := newTestDeps(t, &stubClient{
			listBridgesFn: func(context.Context, BridgeListQuery) (BridgeListRecord, error) {
				return result, nil
			},
		})

		stdout, _, err := executeRootCommand(t, deps, "bridge", "list", "-o", "jsonl")
		if err != nil {
			t.Fatalf("bridge list jsonl error = %v", err)
		}
		lines := strings.Split(strings.TrimSpace(stdout), "\n")
		if got, want := len(lines), 2; got != want {
			t.Fatalf("bridge list jsonl lines = %d, want %d: %s", got, want, stdout)
		}
		var item struct {
			Type   string                       `json:"type"`
			Bridge contract.BridgePayload       `json:"bridge"`
			Health contract.BridgeHealthPayload `json:"health"`
		}
		if err := json.Unmarshal([]byte(lines[0]), &item); err != nil {
			t.Fatalf("json.Unmarshal(jsonl item) error = %v", err)
		}
		if item.Type != "bridge" || item.Bridge.Status != bridgepkg.BridgeStatusReady ||
			item.Health.Status != bridgepkg.BridgeStatusError || item.Health.LastError != "adapter crashed" {
			t.Fatalf("jsonl item = %#v, want typed bridge plus effective error health", item)
		}
		var terminal struct {
			Type   string                              `json:"type"`
			Page   contract.CountedCursorPagePayload   `json:"page"`
			Facets contract.BridgeCatalogFacetsPayload `json:"facets"`
		}
		if err := json.Unmarshal([]byte(lines[1]), &terminal); err != nil {
			t.Fatalf("json.Unmarshal(jsonl terminal) error = %v", err)
		}
		if terminal.Type != "page" || terminal.Page != result.Page || terminal.Facets.Statuses.Ready != 1 {
			t.Fatalf("jsonl terminal = %#v, want page and facets", terminal)
		}
	})
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
	if decoded.ID != expected.ID || decoded.Scope != expected.Scope ||
		decoded.Status != expected.Status ||
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
			record.ProviderConfig = json.RawMessage(request.ProviderConfig)
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
		"--provider-config", `{"api_base_url":"https://slack.test/api"}`,
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
	if string(captured.ProviderConfig) != `{"api_base_url":"https://slack.test/api"}` {
		t.Fatalf("captured provider config = %s", string(captured.ProviderConfig))
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
	if err == nil ||
		!strings.Contains(err.Error(), "--workspace-id is required when --scope=workspace") {
		t.Fatalf("bridge create error = %v, want missing workspace-id validation", err)
	}
}

func TestBridgeCreateRejectsOperationalStatusFlag(t *testing.T) {
	t.Parallel()

	t.Run("Should reject operational status flag", func(t *testing.T) {
		t.Parallel()

		deps := newTestDeps(t, &stubClient{
			createBridgeFn: func(context.Context, CreateBridgeRequest) (BridgeRecord, error) {
				t.Fatal(
					"CreateBridge() should not be called when operational status flag is provided",
				)
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
			updated.ProviderConfig = json.RawMessage(*request.ProviderConfig)
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
		"--provider-config", `{"api_base_url":"https://slack.test/api"}`,
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
	if captured.RoutingPolicy == nil || !captured.RoutingPolicy.IncludePeer ||
		!captured.RoutingPolicy.IncludeThread ||
		!captured.RoutingPolicy.IncludeGroup {
		t.Fatalf("captured routing policy = %#v", captured.RoutingPolicy)
	}
	if captured.DeliveryDefaults == nil || string(*captured.DeliveryDefaults) != "null" {
		t.Fatalf("captured delivery defaults = %#v", captured.DeliveryDefaults)
	}
	if captured.ProviderConfig == nil ||
		string(*captured.ProviderConfig) != `{"api_base_url":"https://slack.test/api"}` {
		t.Fatalf("captured provider config = %#v", captured.ProviderConfig)
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

func TestBridgeTargetsUseDaemonClientAndRenderDirectory(t *testing.T) {
	t.Parallel()

	deps := newTestDeps(t, &stubClient{
		bridgeTargetsFn: func(_ context.Context, id string, query string, limit int) (BridgeTargetsRecord, error) {
			if id != "brg-1" {
				t.Fatalf("BridgeTargets() id = %q, want brg-1", id)
			}
			if query != "support" || limit != 25 {
				t.Fatalf("BridgeTargets() query/limit = %q/%d, want support/25", query, limit)
			}
			return BridgeTargetsRecord{
				BridgeID: "brg-1",
				Targets: []BridgeTargetRecord{{
					BridgeID:       "brg-1",
					CanonicalRoute: "telegram:channel:support",
					DisplayName:    "Support room",
					Normalized:     "support room",
					TargetType:     bridgepkg.BridgeTargetTypeChannel,
					Qualifier:      "telegram",
					Capabilities:   []string{"direct-send", "reply"},
					UpdatedAt:      fixedTestNow,
					LastSeenAt:     fixedTestNow,
				}},
				Total:       1,
				CacheStale:  false,
				GeneratedAt: fixedTestNow,
			}, nil
		},
	})

	stdout, _, err := executeRootCommand(
		t,
		deps,
		"bridge", "targets", "brg-1", "--query", "support", "--limit", "25", "-o", "human",
	)
	if err != nil {
		t.Fatalf("bridge targets human error = %v", err)
	}

	for _, token := range []string{"Bridge Targets", "telegram:channel:support", "Support room", "telegram", "direct-send,reply"} {
		if !strings.Contains(stdout, token) {
			t.Fatalf("bridge targets human output missing %q: %s", token, stdout)
		}
	}
}

func TestBridgeResolveUsesDaemonClientAndReportsAmbiguity(t *testing.T) {
	t.Parallel()

	deps := newTestDeps(t, &stubClient{
		resolveBridgeTargetFn: func(_ context.Context, id string, name string) (BridgeResolveTargetRecord, error) {
			if id != "brg-1" || name != "support" {
				t.Fatalf("ResolveBridgeTarget() id/name = %q/%q, want brg-1/support", id, name)
			}
			return BridgeResolveTargetRecord{
				Result: bridgepkg.ResolveBridgeTargetResult{
					Step:      4,
					Ambiguous: true,
					Candidates: []bridgepkg.BridgeTarget{{
						BridgeID:       "brg-1",
						CanonicalRoute: "telegram:channel:support",
						DisplayName:    "Support room",
						Normalized:     "support room",
						TargetType:     bridgepkg.BridgeTargetTypeChannel,
						Qualifier:      "telegram",
						Capabilities:   []string{"reply"},
						UpdatedAt:      fixedTestNow,
						LastSeenAt:     fixedTestNow,
					}},
				},
			}, nil
		},
	})

	stdout, _, err := executeRootCommand(
		t,
		deps,
		"bridge",
		"resolve",
		"brg-1",
		"support",
		"-o",
		"human",
	)
	if err != nil {
		t.Fatalf("bridge resolve human error = %v", err)
	}

	for _, token := range []string{"Bridge Target", "unresolved", "Step", "4", "Ambiguous", "true", "Candidates", "1"} {
		if !strings.Contains(stdout, token) {
			t.Fatalf("bridge resolve human output missing %q: %s", token, stdout)
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
	if decoded.DeliveryTarget.ThreadID != "thread-1" ||
		decoded.DeliveryTarget.Mode != bridgepkg.DeliveryModeReply {
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
		"bridge{id,display_name,platform,extension_name,scope,workspace_id,enabled,status,routing,include_peer,include_thread,include_group,notification_suppress,delivery_defaults,created_at,updated_at}:",
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
		t.Fatalf(
			"parseRequiredBridgeJSON(object) = %s, want preserved object",
			string(*validObject),
		)
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
			t.Fatalf(
				"parseRequiredBridgeJSON(%s) error = %v, want object-or-null validation",
				raw,
				err,
			)
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

func testBridgeListRecord(t *testing.T) BridgeListRecord {
	t.Helper()

	item := testBridgeRecord(t)
	return BridgeListRecord{
		Bridges: []contract.BridgePayload{{
			ID:                   item.ID,
			Scope:                item.Scope,
			WorkspaceID:          item.WorkspaceID,
			Platform:             item.Platform,
			ExtensionName:        item.ExtensionName,
			DisplayName:          item.DisplayName,
			Enabled:              item.Enabled,
			Status:               item.Status,
			RoutingPolicy:        item.RoutingPolicy,
			DeliveryDefaults:     contract.BridgeDeliveryDefaultsPayload(item.DeliveryDefaults),
			NotificationSuppress: item.NotificationSuppress,
			CreatedAt:            item.CreatedAt,
			UpdatedAt:            item.UpdatedAt,
		}},
		BridgeHealth: map[string]contract.BridgeHealthPayload{
			item.ID: {BridgeInstanceID: item.ID, Status: bridgepkg.BridgeStatusReady},
		},
		Facets: contract.BridgeCatalogFacetsPayload{
			Platforms: map[string]int{"telegram": 1},
			Statuses:  contract.BridgeStatusCountsPayload{Ready: 1},
		},
		Page: contract.CountedCursorPagePayload{Total: 1, Limit: bridgepkg.DefaultBridgeCatalogLimit},
	}
}
