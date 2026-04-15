package core

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	bundlepkg "github.com/pedronauck/agh/internal/bundles"
	extensionpkg "github.com/pedronauck/agh/internal/extension"
	"github.com/pedronauck/agh/internal/network"
	observepkg "github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func TestBundleCatalogPayloadsAndDeclaredChannels(t *testing.T) {
	t.Parallel()

	catalog := BundleCatalogPayloads([]bundlepkg.CatalogEntry{{
		ExtensionName: " ext-bundle ",
		Bundle: extensionpkg.BundleSpec{
			Name:        " ops ",
			Description: " Operations bundle ",
			Profiles: []extensionpkg.BundleProfile{{
				Name:        " default ",
				Description: " Primary profile ",
				Channels: extensionpkg.BundleChannelsConfig{
					Primary: "primary",
					Items: []extensionpkg.BundleChannel{
						{Name: " primary ", Description: " Main channel "},
						{Name: " secondary ", Description: " Backup channel "},
					},
				},
				Jobs:     []extensionpkg.BundleJob{{Name: "job-a"}},
				Triggers: []extensionpkg.BundleTrigger{{Name: "trigger-a"}},
				Bridges:  []extensionpkg.BundleBridgePreset{{Name: "bridge-a"}},
			}},
		},
	}})

	if got, want := len(catalog), 1; got != want {
		t.Fatalf("len(catalog) = %d, want %d", got, want)
	}
	if catalog[0].ExtensionName != "ext-bundle" || catalog[0].BundleName != "ops" || catalog[0].Profiles[0].PrimaryChannel != "primary" {
		t.Fatalf("catalog payload = %#v", catalog[0])
	}
	if got, want := len(catalog[0].Profiles[0].Channels), 2; got != want {
		t.Fatalf("len(profile channels) = %d, want %d", got, want)
	}
	if !catalog[0].Profiles[0].Channels[0].Primary || catalog[0].Profiles[0].Channels[1].Primary {
		t.Fatalf("channel primary flags = %#v", catalog[0].Profiles[0].Channels)
	}

	declared := DeclaredNetworkChannelPayloads([]bundlepkg.DeclaredChannel{{
		ActivationID:  " act-1 ",
		ExtensionName: " ext-bundle ",
		BundleName:    " ops ",
		ProfileName:   " default ",
		WorkspaceID:   " ws-1 ",
		Name:          " builders ",
		Description:   " Build channel ",
		Primary:       true,
	}})
	if got, want := len(declared), 1; got != want {
		t.Fatalf("len(declared) = %d, want %d", got, want)
	}
	if declared[0].ActivationID != "act-1" || declared[0].Name != "builders" || !declared[0].Primary {
		t.Fatalf("declared payload = %#v", declared[0])
	}
}

func TestStatusForBundleErrorAndChannelHelpers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "nil", err: nil, want: http.StatusOK},
		{name: "activation missing", err: bundlepkg.ErrActivationNotFound, want: http.StatusNotFound},
		{name: "bundle missing", err: bundlepkg.ErrBundleNotFound, want: http.StatusNotFound},
		{name: "profile missing", err: bundlepkg.ErrProfileNotFound, want: http.StatusNotFound},
		{name: "extension missing", err: extensionpkg.ErrExtensionNotFound, want: http.StatusNotFound},
		{name: "default channel busy", err: bundlepkg.ErrDefaultChannelBusy, want: http.StatusConflict},
		{name: "extension has active bundles", err: extensionpkg.ErrExtensionHasActiveBundles, want: http.StatusConflict},
		{name: "webhook unsupported", err: bundlepkg.ErrWebhookUnsupported, want: http.StatusBadRequest},
		{name: "workspace missing", err: workspacepkg.ErrWorkspaceNotFound, want: http.StatusNotFound},
		{name: "workspace root missing", err: workspacepkg.ErrWorkspaceRootMissing, want: http.StatusGone},
		{name: "default", err: errors.New("boom"), want: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := StatusForBundleError(tt.err); got != tt.want {
				t.Fatalf("StatusForBundleError(%v) = %d, want %d", tt.err, got, tt.want)
			}
		})
	}

	sessionChannel := "builders"
	sessions := []*session.SessionInfo{
		{ID: "sess-visible", Channel: " builders ", State: session.StateActive},
		{ID: "sess-stopped", Channel: "builders", State: session.StateStopped},
	}
	peers := []network.PeerInfo{{PeerID: "peer-1", Channel: "operators"}}
	if !networkChannelExists(sessions, peers, sessionChannel) {
		t.Fatal("networkChannelExists() = false, want true for visible session channel")
	}
	if !networkChannelExists(nil, []network.PeerInfo{{PeerID: "peer-2", Channel: "match"}}, "match") {
		t.Fatal("networkChannelExists() = false, want true for peer channel")
	}
	if networkChannelExists(nil, peers, "missing") {
		t.Fatal("networkChannelExists() = true, want false for missing channel")
	}
	if !isNetworkChannelNotFound(errNetworkChannelNotFound) {
		t.Fatal("isNetworkChannelNotFound() = false, want true")
	}
}

func TestNetworkPayloadHelpersCloneAndNormalize(t *testing.T) {
	t.Parallel()

	joinedAt := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	lastSeen := joinedAt.Add(5 * time.Minute)
	expiresAt := joinedAt.Add(10 * time.Minute)
	displayName := " Support Bot "
	sessionID := "sess-1"
	ext := map[string]json.RawMessage{"role": json.RawMessage(`"support"`)}
	peerPayloads := NetworkPeerPayloadsFromInfos([]network.PeerInfo{{
		SessionID: &sessionID,
		PeerID:    "peer-1",
		Channel:   "builders",
		Local:     true,
		PeerCard: network.PeerCard{
			PeerID:              "peer-1",
			DisplayName:         &displayName,
			ProfilesSupported:   []string{"default"},
			Capabilities:        []string{"chat"},
			ArtifactsSupported:  []string{"text"},
			TrustModesSupported: []string{"strict"},
			Ext:                 ext,
		},
		JoinedAt:  &joinedAt,
		LastSeen:  &lastSeen,
		ExpiresAt: &expiresAt,
	}})

	if got, want := len(peerPayloads), 1; got != want {
		t.Fatalf("len(peerPayloads) = %d, want %d", got, want)
	}
	if peerPayloads[0].DisplayName != "Support Bot" {
		t.Fatalf("DisplayName = %q, want %q", peerPayloads[0].DisplayName, "Support Bot")
	}
	if peerPayloads[0].PeerCard.Ext["role"] == nil {
		t.Fatalf("PeerCard.Ext = %#v, want copied metadata", peerPayloads[0].PeerCard.Ext)
	}

	displayName = "mutated"
	ext["role"][0] = '['
	if peerPayloads[0].DisplayName != "Support Bot" || string(peerPayloads[0].PeerCard.Ext["role"]) != `"support"` {
		t.Fatalf("peer payload mutated with source data = %#v", peerPayloads[0])
	}

	channelPayloads := NetworkChannelPayloadsFromInfos([]network.ChannelInfo{{Channel: "builders", PeerCount: 2}})
	if got, want := len(channelPayloads), 1; got != want {
		t.Fatalf("len(channelPayloads) = %d, want %d", got, want)
	}
	if channelPayloads[0].Channel != "builders" || channelPayloads[0].PeerCount != 2 {
		t.Fatalf("channel payload = %#v", channelPayloads[0])
	}
}

func TestCoreConversionHelpers(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	later := now.Add(5 * time.Minute)

	usage := TokenUsagePayloadFromUsage(&acp.TokenUsage{
		TurnID:           "turn-1",
		InputTokens:      int64Ptr(10),
		OutputTokens:     int64Ptr(20),
		TotalTokens:      int64Ptr(30),
		ThoughtTokens:    int64Ptr(3),
		CacheReadTokens:  int64Ptr(4),
		CacheWriteTokens: int64Ptr(5),
		ContextUsed:      int64Ptr(6),
		ContextSize:      int64Ptr(7),
		CostAmount:       float64Ptr(1.23),
		CostCurrency:     stringPtr("USD"),
		Timestamp:        now,
	})
	if usage == nil || usage.TotalTokens == nil || *usage.TotalTokens != 30 || usage.CostCurrency == nil || *usage.CostCurrency != "USD" {
		t.Fatalf("TokenUsagePayloadFromUsage() = %#v", usage)
	}
	if TokenUsagePayloadFromUsage(nil) != nil {
		t.Fatal("TokenUsagePayloadFromUsage(nil) != nil")
	}

	health := BridgeHealthPayloadFromObserve(observepkg.BridgeInstanceHealth{
		BridgeInstanceID:        "brg-1",
		Status:                  bridgepkg.BridgeStatusDegraded,
		RouteCount:              2,
		DeliveryBacklog:         1,
		DeliveryDroppedTotal:    3,
		DeliveryDroppedByReason: map[string]int{"rate_limit": 2},
		DeliveryFailuresTotal:   4,
		AuthFailuresTotal:       5,
		LastSuccessAt:           now,
		LastError:               "timeout",
		LastErrorAt:             later,
	})
	if health.LastSuccessAt == nil || health.LastErrorAt == nil || health.DeliveryDroppedByReason["rate_limit"] != 2 {
		t.Fatalf("BridgeHealthPayloadFromObserve() = %#v", health)
	}

	if got := string(PayloadJSON("  ")); got != "null" {
		t.Fatalf("PayloadJSON(blank) = %s, want null", got)
	}
	if got := string(PayloadJSON(`{"ok":true}`)); got != `{"ok":true}` {
		t.Fatalf("PayloadJSON(valid json) = %s", got)
	}
	if got := string(PayloadJSON("not-json")); got != `"not-json"` {
		t.Fatalf("PayloadJSON(string) = %s, want quoted string", got)
	}

	if workspaceID, workspace := sessionWorkspaceFromInfo(&session.SessionInfo{WorkspaceID: " ws-1 ", Workspace: " /tmp/ws "}); workspaceID != "ws-1" || workspace != "/tmp/ws" {
		t.Fatalf("sessionWorkspaceFromInfo() = %q/%q", workspaceID, workspace)
	}
	if workspaceID, workspace := sessionWorkspaceFromInfo(nil); workspaceID != "" || workspace != "" {
		t.Fatalf("sessionWorkspaceFromInfo(nil) = %q/%q", workspaceID, workspace)
	}

	if got := laterTimePtr(nil, now); got == nil || !got.Equal(now) {
		t.Fatalf("laterTimePtr(nil, now) = %#v", got)
	}
	if got := laterTimePtr(&later, now); got == nil || !got.Equal(later) {
		t.Fatalf("laterTimePtr(later, earlier) = %#v", got)
	}
	if got := laterTimePtr(&later, time.Time{}); got == nil || !got.Equal(later) {
		t.Fatalf("laterTimePtr(later, zero) = %#v", got)
	}

	role := json.RawMessage(`"support"`)
	proof := network.Proof{"role": role}
	clonedProof := cloneProofPtr(&proof)
	if clonedProof == nil || string(clonedProof["role"]) != `"support"` {
		t.Fatalf("cloneProofPtr() = %#v", clonedProof)
	}
	proof["role"][0] = '['
	if string(clonedProof["role"]) != `"support"` {
		t.Fatalf("cloneProofPtr() mutated with source proof = %#v", clonedProof)
	}
	if cloneProofPtr(nil) != nil {
		t.Fatal("cloneProofPtr(nil) != nil")
	}

	peerSessionID := "sess-1"
	peerName := "Peer Display"
	if got := networkPeerDisplayName(network.PeerInfo{
		SessionID: &peerSessionID,
		PeerID:    "peer-1",
		PeerCard:  network.PeerCard{DisplayName: &peerName},
	}, nil); got != "Peer Display" {
		t.Fatalf("networkPeerDisplayName(peer card) = %q", got)
	}

	if got := networkPeerDisplayName(network.PeerInfo{
		SessionID: &peerSessionID,
		PeerID:    "peer-1",
	}, map[string]*session.SessionInfo{
		"sess-1": {Name: "Session Name", AgentName: "coder"},
	}); got != "Session Name" {
		t.Fatalf("networkPeerDisplayName(session name) = %q", got)
	}

	if got := networkPeerDisplayName(network.PeerInfo{
		SessionID: &peerSessionID,
		PeerID:    "peer-1",
	}, map[string]*session.SessionInfo{
		"sess-1": {AgentName: "coder"},
	}); got != "coder" {
		t.Fatalf("networkPeerDisplayName(agent fallback) = %q", got)
	}

	if got := networkPeerDisplayName(network.PeerInfo{PeerID: " peer-1 "}, nil); got != "peer-1" {
		t.Fatalf("networkPeerDisplayName(peer id fallback) = %q", got)
	}
}

func TestCoreTimeAndSessionHelpers(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 15, 12, 0, 0, 0, time.FixedZone("offset", -3*60*60))
	got := timePointerFromMap(map[string]*time.Time{"sess-1": &now}, "sess-1")
	if got == nil || !got.Equal(now.UTC()) || got.Location() != time.UTC {
		t.Fatalf("timePointerFromMap() = %#v, want UTC copy", got)
	}
	if timePointerFromMap(nil, "sess-1") != nil {
		t.Fatal("timePointerFromMap(nil) != nil")
	}
	if timePointerFromMap(map[string]*time.Time{"sess-1": nil}, "sess-1") != nil {
		t.Fatal("timePointerFromMap(nil entry) != nil")
	}
	if networkChannelSessionVisible(nil) {
		t.Fatal("networkChannelSessionVisible(nil) = true, want false")
	}
	if networkChannelSessionVisible(&session.SessionInfo{State: session.StateStopped, Channel: "builders"}) {
		t.Fatal("networkChannelSessionVisible(stopped) = true, want false")
	}
	if !networkChannelSessionVisible(&session.SessionInfo{State: session.StateActive, Channel: " builders "}) {
		t.Fatal("networkChannelSessionVisible(active) = false, want true")
	}
}

func TestSessionAndNetworkMappingHelpers(t *testing.T) {
	t.Parallel()

	payload := SessionPayloadFromInfo(&session.SessionInfo{
		ID:          "sess-1",
		Name:        "Support session",
		AgentName:   "coder",
		WorkspaceID: " ws-1 ",
		Workspace:   " /tmp/ws ",
		ACPCaps: acp.ACPCaps{
			SupportsLoadSession: true,
			SupportedModes:      []string{"edit"},
		},
	})
	if payload.ID != "sess-1" || payload.WorkspaceID != "ws-1" || payload.WorkspacePath != "/tmp/ws" || payload.ACPCaps == nil {
		t.Fatalf("SessionPayloadFromInfo() = %#v", payload)
	}
	if zero := SessionPayloadFromInfo(nil); zero.ID != "" {
		t.Fatalf("SessionPayloadFromInfo(nil) = %#v", zero)
	}

	ids := networkMessageIDSet([]store.NetworkMessageEntry{
		{MessageID: " msg-1 "},
		{MessageID: ""},
		{MessageID: "msg-2"},
	})
	if _, ok := ids["msg-1"]; !ok {
		t.Fatalf("networkMessageIDSet() missing trimmed msg-1: %#v", ids)
	}

	directions := networkMessageDirectionMap(
		[]store.NetworkAuditEntry{
			{MessageID: " msg-1 ", Direction: network.AuditDirectionSent},
			{MessageID: "msg-1", Direction: network.AuditDirectionReceived},
			{MessageID: "msg-2", Direction: "invalid"},
			{MessageID: "msg-3", Direction: network.AuditDirectionReceived},
		},
		map[string]struct{}{"msg-1": {}, "msg-2": {}},
	)
	if got, want := directions["msg-1"], network.AuditDirectionSent; got != want {
		t.Fatalf("networkMessageDirectionMap(msg-1) = %q, want %q", got, want)
	}
	if _, ok := directions["msg-2"]; ok {
		t.Fatalf("networkMessageDirectionMap(msg-2) unexpectedly set: %#v", directions)
	}

	sessionsByID := sessionInfoMapByID([]*session.SessionInfo{
		{ID: " sess-1 ", Name: "Support"},
		nil,
	})
	if sessionsByID["sess-1"] == nil {
		t.Fatalf("sessionInfoMapByID() missing trimmed session id: %#v", sessionsByID)
	}
}

func int64Ptr(value int64) *int64 {
	return &value
}

func float64Ptr(value float64) *float64 {
	return &value
}

func stringPtr(value string) *string {
	return &value
}
