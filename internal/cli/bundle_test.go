package cli

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestBundleCommands(t *testing.T) {
	t.Parallel()

	t.Run("Should activate bundle presets through the daemon client", func(t *testing.T) {
		t.Parallel()

		var captured ActivateBundleRequest
		deps := newTestDeps(t, &stubClient{
			activateBundleFn: func(_ context.Context, request ActivateBundleRequest) (BundleActivationRecord, error) {
				captured = request
				return sampleBundleActivationRecord(), nil
			},
		})
		stdout, _, err := executeRootCommand(
			t,
			deps,
			"bundle",
			"activate",
			"--extension",
			"marketing-team",
			"--bundle",
			"marketing",
			"--profile",
			"default",
			"--scope",
			"workspace",
			"--workspace",
			"ws-marketing",
			"--bind-primary-channel-as-default",
			"--json",
		)
		if err != nil {
			t.Fatalf("bundle activate error = %v", err)
		}
		if captured.ExtensionName != "marketing-team" ||
			captured.BundleName != "marketing" ||
			captured.ProfileName != "default" ||
			captured.Scope != "workspace" ||
			captured.Workspace != "ws-marketing" ||
			!captured.BindPrimaryChannelAsDefault {
			t.Fatalf("captured request = %#v", captured)
		}

		var payload BundleActivationRecord
		if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
			t.Fatalf("json.Unmarshal(stdout) error = %v\nstdout=%s", err, stdout)
		}
		if got, want := len(payload.Agents), 1; got != want {
			t.Fatalf("len(payload.Agents) = %d, want %d", got, want)
		}
		if !payload.Agents[0].HasSoul || !payload.Agents[0].HasHeartbeat {
			t.Fatalf("payload.Agents[0] sidecar flags = %#v", payload.Agents[0])
		}
		if !strings.Contains(stdout, `"resource_kind": "agent.soul"`) {
			t.Fatalf("stdout = %s, want agent.soul inventory", stdout)
		}
	})

	t.Run("Should preview bundle presets without activation", func(t *testing.T) {
		t.Parallel()

		var captured ActivateBundleRequest
		deps := newTestDeps(t, &stubClient{
			previewBundleActivationFn: func(
				_ context.Context,
				request ActivateBundleRequest,
			) (BundleActivationRecord, error) {
				captured = request
				item := sampleBundleActivationRecord()
				item.ID = "preview_marketing"
				return item, nil
			},
		})
		stdout, _, err := executeRootCommand(
			t,
			deps,
			"bundle",
			"preview",
			"--extension",
			"marketing-team",
			"--bundle",
			"marketing",
			"--profile",
			"default",
			"--scope",
			"global",
			"--json",
		)
		if err != nil {
			t.Fatalf("bundle preview error = %v", err)
		}
		if captured.ExtensionName != "marketing-team" ||
			captured.BundleName != "marketing" ||
			captured.ProfileName != "default" ||
			captured.Scope != "global" ||
			captured.BindPrimaryChannelAsDefault {
			t.Fatalf("captured preview request = %#v", captured)
		}

		var payload BundleActivationRecord
		if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
			t.Fatalf("json.Unmarshal(stdout) error = %v\nstdout=%s", err, stdout)
		}
		if got, want := payload.ID, "preview_marketing"; got != want {
			t.Fatalf("payload.ID = %q, want %q", got, want)
		}
		if got, want := len(payload.Inventory), 3; got != want {
			t.Fatalf("len(payload.Inventory) = %d, want %d", got, want)
		}
	})

	t.Run("Should list catalog profiles with agent counts", func(t *testing.T) {
		t.Parallel()

		deps := newTestDeps(t, &stubClient{
			listBundleCatalogFn: func(context.Context) ([]BundleCatalogRecord, error) {
				return []BundleCatalogRecord{{
					ExtensionName: "marketing-team",
					BundleName:    "marketing",
					Profiles: []BundleProfileCatalogRecord{{
						Name:       "default",
						AgentCount: 1,
						JobCount:   1,
					}},
				}}, nil
			},
		})
		stdout, _, err := executeRootCommand(t, deps, "bundle", "catalog", "--json")
		if err != nil {
			t.Fatalf("bundle catalog error = %v", err)
		}
		if !strings.Contains(stdout, `"agent_count": 1`) {
			t.Fatalf("stdout = %s, want agent_count", stdout)
		}
	})

	t.Run("Should list active bundle activations", func(t *testing.T) {
		t.Parallel()

		deps := newTestDeps(t, &stubClient{
			listBundleActivationsFn: func(context.Context) ([]BundleActivationRecord, error) {
				return []BundleActivationRecord{sampleBundleActivationRecord()}, nil
			},
		})
		stdout, _, err := executeRootCommand(t, deps, "bundle", "list", "--json")
		if err != nil {
			t.Fatalf("bundle list error = %v", err)
		}

		var payload struct {
			Activations []BundleActivationRecord `json:"activations"`
		}
		if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
			t.Fatalf("json.Unmarshal(stdout) error = %v\nstdout=%s", err, stdout)
		}
		if got, want := len(payload.Activations), 1; got != want {
			t.Fatalf("len(payload.Activations) = %d, want %d", got, want)
		}
		if got, want := len(payload.Activations[0].Agents), 1; got != want {
			t.Fatalf("len(payload.Activations[0].Agents) = %d, want %d", got, want)
		}
	})

	t.Run("Should get one bundle activation by id", func(t *testing.T) {
		t.Parallel()

		var capturedID string
		deps := newTestDeps(t, &stubClient{
			getBundleActivationFn: func(_ context.Context, id string) (BundleActivationRecord, error) {
				capturedID = id
				return sampleBundleActivationRecord(), nil
			},
		})
		stdout, _, err := executeRootCommand(t, deps, "bundle", "get", "act_marketing", "--json")
		if err != nil {
			t.Fatalf("bundle get error = %v", err)
		}
		if capturedID != "act_marketing" {
			t.Fatalf("captured activation id = %q, want act_marketing", capturedID)
		}

		var payload BundleActivationRecord
		if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
			t.Fatalf("json.Unmarshal(stdout) error = %v\nstdout=%s", err, stdout)
		}
		if got, want := payload.ID, "act_marketing"; got != want {
			t.Fatalf("payload.ID = %q, want %q", got, want)
		}
	})

	t.Run("Should update and clear default-channel binding explicitly", func(t *testing.T) {
		t.Parallel()

		var captured UpdateBundleActivationRequest
		deps := newTestDeps(t, &stubClient{
			updateBundleActivationFn: func(
				_ context.Context,
				id string,
				request UpdateBundleActivationRequest,
			) (BundleActivationRecord, error) {
				if id != "act_marketing" {
					t.Fatalf("activation id = %q, want act_marketing", id)
				}
				captured = request
				item := sampleBundleActivationRecord()
				item.BindPrimaryChannelAsDefault = request.BindPrimaryChannelAsDefault
				return item, nil
			},
		})
		if _, _, err := executeRootCommand(
			t,
			deps,
			"bundle",
			"update",
			"act_marketing",
			"--clear-primary-channel-default",
			"--json",
		); err != nil {
			t.Fatalf("bundle update clear error = %v", err)
		}
		if captured.BindPrimaryChannelAsDefault {
			t.Fatalf("captured.BindPrimaryChannelAsDefault = true, want false")
		}
	})

	t.Run("Should reject ambiguous update flags before calling the client", func(t *testing.T) {
		t.Parallel()

		deps := newTestDeps(t, &stubClient{})
		_, _, err := executeRootCommand(t, deps, "bundle", "update", "act_marketing")
		if err == nil {
			t.Fatal("bundle update without flags error = nil, want validation error")
		}
		if !strings.Contains(err.Error(), "--bind-primary-channel-as-default") {
			t.Fatalf("bundle update error = %v, want flag guidance", err)
		}
	})

	t.Run("Should deactivate one bundle activation by id", func(t *testing.T) {
		t.Parallel()

		var capturedID string
		deps := newTestDeps(t, &stubClient{
			deactivateBundleFn: func(_ context.Context, id string) error {
				capturedID = id
				return nil
			},
		})
		stdout, _, err := executeRootCommand(t, deps, "bundle", "deactivate", "act_marketing", "--json")
		if err != nil {
			t.Fatalf("bundle deactivate error = %v", err)
		}
		if capturedID != "act_marketing" {
			t.Fatalf("captured activation id = %q, want act_marketing", capturedID)
		}

		var payload map[string]string
		if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
			t.Fatalf("json.Unmarshal(stdout) error = %v\nstdout=%s", err, stdout)
		}
		if got, want := payload["deactivated"], "act_marketing"; got != want {
			t.Fatalf("payload[deactivated] = %q, want %q", got, want)
		}
	})

	t.Run("Should expose bundle network settings", func(t *testing.T) {
		t.Parallel()

		deps := newTestDeps(t, &stubClient{
			bundleNetworkSettingsFn: func(context.Context) (BundleNetworkSettingsRecord, error) {
				return BundleNetworkSettingsRecord{
					ConfiguredDefaultChannel: "default",
					EffectiveDefaultChannel:  "marketing",
					EffectiveDefaultSource:   "bundle:act_marketing",
					DeclaredChannels: []DeclaredNetworkChannelRecord{{
						ActivationID: "act_marketing",
						Name:         "marketing",
						Primary:      true,
					}},
				}, nil
			},
		})
		stdout, _, err := executeRootCommand(t, deps, "bundle", "network-settings", "-o", "toon")
		if err != nil {
			t.Fatalf("bundle network-settings error = %v", err)
		}
		if !strings.Contains(stdout, "bundle_network") || !strings.Contains(stdout, "marketing") {
			t.Fatalf("stdout = %q, want bundle network toon output", stdout)
		}
	})
}

func sampleBundleActivationRecord() BundleActivationRecord {
	return BundleActivationRecord{
		ID:                          "act_marketing",
		ExtensionName:               "marketing-team",
		BundleName:                  "marketing",
		BundleDescription:           "Marketing team bundle",
		ProfileName:                 "default",
		ProfileDescription:          "Default profile",
		Scope:                       "workspace",
		WorkspaceID:                 "ws-marketing",
		BindPrimaryChannelAsDefault: true,
		Channels: []BundleChannelRecord{{
			Name:        "marketing",
			Description: "Marketing coordination",
			Primary:     true,
		}},
		Agents: []BundleAgentRecord{{
			ID:           "agt_marketer",
			Name:         "marketer",
			Provider:     "claude",
			Model:        "sonnet",
			HasSoul:      true,
			HasHeartbeat: true,
		}},
		Jobs: []BundleJobRecord{{
			ID:        "job_daily",
			Name:      "daily-sync",
			AgentName: "marketer",
			Enabled:   true,
		}},
		Triggers: []BundleTriggerRecord{{
			ID:        "trg_session",
			Name:      "session-opened",
			AgentName: "marketer",
			Event:     "session.created",
			Enabled:   true,
		}},
		Bridges: []BundleBridgeRecord{{
			ID:            "bri_linear",
			Name:          "linear-main",
			ExtensionName: "linear",
			Platform:      "linear",
			DisplayName:   "Linear",
		}},
		Inventory: []BundleInventoryRecord{
			{ResourceKind: "agent", ResourceID: "agt_marketer", ResourceName: "marketer"},
			{ResourceKind: "agent.soul", ResourceID: "sol_marketer", ResourceName: "marketer"},
			{ResourceKind: "agent.heartbeat", ResourceID: "hbt_marketer", ResourceName: "marketer"},
		},
		CreatedAt: time.Date(2026, 5, 3, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 5, 3, 10, 0, 0, 0, time.UTC),
	}
}
