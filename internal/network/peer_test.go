package network

import (
	"encoding/json"
	"slices"
	"testing"
	"time"

	sessionpkg "github.com/pedronauck/agh/internal/session"
)

func TestPeerRegistryIsolatesChannelsExpiresRemotesAndLeavesLocal(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	registry, err := NewPeerRegistry(10*time.Second, WithPeerRegistryClock(func() time.Time { return now }))
	if err != nil {
		t.Fatalf("NewPeerRegistry() error = %v", err)
	}

	localCard := mustPeerCard(t, "coder.sess-local")
	remoteBuilders := mustPeerCard(t, "reviewer.sess-builders")
	remoteOps := mustPeerCard(t, "reviewer.sess-ops")

	if _, err := registry.RegisterLocal("sess-local", "builders", localCard, now); err != nil {
		t.Fatalf("RegisterLocal(local) error = %v", err)
	}
	if _, stored, err := registry.RefreshRemote("builders", remoteBuilders, now); err != nil {
		t.Fatalf("RefreshRemote(builders) error = %v", err)
	} else if !stored {
		t.Fatal("RefreshRemote(builders) stored = false, want true")
	}
	if _, stored, err := registry.RefreshRemote("ops", remoteOps, now); err != nil {
		t.Fatalf("RefreshRemote(ops) error = %v", err)
	} else if !stored {
		t.Fatal("RefreshRemote(ops) stored = false, want true")
	}

	if _, ok := registry.LookupPresence("builders", remoteBuilders.PeerID, now); !ok {
		t.Fatalf("LookupPresence(builders, %q) = missing, want present", remoteBuilders.PeerID)
	}
	if _, ok := registry.LookupPresence("builders", remoteOps.PeerID, now); ok {
		t.Fatalf("LookupPresence(builders, %q) = present, want isolated by channel", remoteOps.PeerID)
	}

	peers := registry.ListPeers("builders", now)
	if got, want := len(peers), 2; got != want {
		t.Fatalf("len(ListPeers(builders)) = %d, want %d", got, want)
	}
	if !peers[0].Local || peers[1].Local {
		t.Fatalf("ListPeers(builders) local ordering mismatch = %#v", peers)
	}

	expiredAt := now.Add(21 * time.Second)
	if _, ok := registry.LookupPresence("builders", remoteBuilders.PeerID, expiredAt); ok {
		t.Fatalf("LookupPresence(builders, %q) after expiry = present, want expired", remoteBuilders.PeerID)
	}
	if _, ok := registry.LookupPresence("builders", localCard.PeerID, expiredAt); !ok {
		t.Fatalf("LookupPresence(builders, %q) local = missing after remote expiry", localCard.PeerID)
	}

	if _, ok := registry.LeaveLocal("sess-local"); !ok {
		t.Fatal("LeaveLocal(sess-local) ok = false, want true")
	}
	if _, ok := registry.LookupPresence("builders", localCard.PeerID, expiredAt); ok {
		t.Fatalf("LookupPresence(builders, %q) after leave = present, want removed", localCard.PeerID)
	}
}

func TestPeerRegistryAccessorsAndChannelSummaries(t *testing.T) {
	t.Parallel()

	if _, err := NewPeerRegistry(0); err == nil {
		t.Fatal("NewPeerRegistry(0) error = nil, want non-nil")
	}

	now := time.Date(2026, 4, 10, 12, 30, 0, 0, time.UTC)
	registry, err := NewPeerRegistry(15 * time.Second)
	if err != nil {
		t.Fatalf("NewPeerRegistry() error = %v", err)
	}
	if got, want := registry.GreetInterval(), 15*time.Second; got != want {
		t.Fatalf("GreetInterval() = %s, want %s", got, want)
	}

	displayName := "Review Bot"
	local := PeerCard{
		PeerID:              "reviewer.sess-b",
		DisplayName:         &displayName,
		ProfilesSupported:   []string{ProtocolV0},
		Capabilities:        []string{"chat.review"},
		ArtifactsSupported:  []string{"recipe"},
		TrustModesSupported: []string{"unverified"},
	}
	if _, err := registry.RegisterLocal("sess-b", "builders", local, now); err != nil {
		t.Fatalf("RegisterLocal(local) error = %v", err)
	}
	remote := mustPeerCard(t, "coder.sess-a")
	if _, stored, err := registry.RefreshRemote("builders", remote, now); err != nil {
		t.Fatalf("RefreshRemote(remote) error = %v", err)
	} else if !stored {
		t.Fatal("RefreshRemote(remote) stored = false, want true")
	}

	if matches := registry.MatchLocalPeers("builders", "Review Bot"); len(matches) != 1 {
		t.Fatalf("MatchLocalPeers(display name) len = %d, want 1", len(matches))
	}
	if matches := registry.MatchLocalPeers("builders", "chat.review"); len(matches) != 1 {
		t.Fatalf("MatchLocalPeers(capability) len = %d, want 1", len(matches))
	}
	if entry, ok := registry.RemoteByPeer("builders", remote.PeerID, now); !ok {
		t.Fatalf("RemoteByPeer(%q) = missing, want present", remote.PeerID)
	} else if got, want := entry.PeerID, remote.PeerID; got != want {
		t.Fatalf("RemoteByPeer().PeerID = %q, want %q", got, want)
	}

	channels := registry.ListChannels(now)
	if got, want := len(channels), 1; got != want {
		t.Fatalf("len(ListChannels()) = %d, want %d", got, want)
	}
	if got, want := channels[0].PeerCount, 2; got != want {
		t.Fatalf("ListChannels()[0].PeerCount = %d, want %d", got, want)
	}

	if _, err := DefaultPeerCard("Bad Peer"); err == nil {
		t.Fatal("DefaultPeerCard(invalid) error = nil, want non-nil")
	}
}

func TestPeerRegistryMoveLocalPeerAndIgnoreMatchingRemoteAdvertisement(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 10, 12, 40, 0, 0, time.UTC)
	registry, err := NewPeerRegistry(10 * time.Second)
	if err != nil {
		t.Fatalf("NewPeerRegistry() error = %v", err)
	}
	local := mustPeerCard(t, "reviewer.sess-b")
	if _, err := registry.RegisterLocal("sess-b", "builders", local, now); err != nil {
		t.Fatalf("RegisterLocal(builders) error = %v", err)
	}

	if _, stored, err := registry.RefreshRemote("builders", local, now); err != nil {
		t.Fatalf("RefreshRemote(local peer) error = %v", err)
	} else if stored {
		t.Fatal("RefreshRemote(local peer) stored = true, want ignored")
	}
	if _, ok := registry.RemoteByPeer("builders", local.PeerID, now); ok {
		t.Fatalf("RemoteByPeer(%q) = present, want local advertisement ignored", local.PeerID)
	}

	if _, err := registry.RegisterLocal("sess-b", "ops", local, now); err != nil {
		t.Fatalf("RegisterLocal(ops) error = %v", err)
	}
	if _, ok := registry.LocalByPeer("builders", local.PeerID); ok {
		t.Fatalf("LocalByPeer(builders, %q) = present after move, want removed", local.PeerID)
	}
	if moved, ok := registry.LocalByPeer("ops", local.PeerID); !ok {
		t.Fatalf("LocalByPeer(ops, %q) = missing after move", local.PeerID)
	} else if got, want := moved.Channel, "ops"; got != want {
		t.Fatalf("moved.Channel = %q, want %q", got, want)
	}
}

func TestProjectCapabilityBriefViewMatchesProjectedIDsAndBriefEntries(t *testing.T) {
	t.Parallel()

	t.Run("Should keep capability ids and brief entries aligned in stable order", func(t *testing.T) {
		t.Parallel()

		ids, ext, err := projectCapabilityBriefView([]sessionpkg.NetworkPeerCapability{
			{
				ID:      " review-pr ",
				Summary: " Review pull requests ",
			},
			{
				ID:      "draft-spec",
				Summary: "Draft technical specs",
			},
		})
		if err != nil {
			t.Fatalf("projectCapabilityBriefView() error = %v", err)
		}

		wantBrief := []capabilityBrief{
			{ID: "review-pr", Summary: "Review pull requests"},
			{ID: "draft-spec", Summary: "Draft technical specs"},
		}
		if got, want := ids, []string{"review-pr", "draft-spec"}; !slices.Equal(got, want) {
			t.Fatalf("projected capability ids = %#v, want %#v", got, want)
		}
		if got := decodeCapabilityBriefPayload(t, ext[capabilityBriefExtKey]); !slices.Equal(got, wantBrief) {
			t.Fatalf("capability brief entries = %#v, want %#v", got, wantBrief)
		}
	})

	t.Run("Should keep no-catalog peers discovery-empty without the brief ext key", func(t *testing.T) {
		t.Parallel()

		ids, ext, err := projectCapabilityBriefView(nil)
		if err != nil {
			t.Fatalf("projectCapabilityBriefView(nil) error = %v", err)
		}
		if ids == nil {
			t.Fatal("projected ids = nil, want empty-but-valid slice")
		}
		if got := len(ids); got != 0 {
			t.Fatalf("len(projected ids) = %d, want 0", got)
		}
		if ext != nil && ext[capabilityBriefExtKey] != nil {
			t.Fatalf("projected ext = %#v, want omitted capability brief key", ext)
		}
	})
}

func TestCloneAndNormalizePeerCardPreserveCapabilityBriefExt(t *testing.T) {
	t.Parallel()

	card, err := DefaultPeerCard("reviewer.sess-brief")
	if err != nil {
		t.Fatalf("DefaultPeerCard() error = %v", err)
	}
	if err := applyCapabilityBriefProjection(&card, []sessionpkg.NetworkPeerCapability{{
		ID:      "review-pr",
		Summary: "Review pull requests",
	}}); err != nil {
		t.Fatalf("applyCapabilityBriefProjection() error = %v", err)
	}

	cloned := clonePeerCard(card)
	normalized, err := normalizePeerCard(card)
	if err != nil {
		t.Fatalf("normalizePeerCard() error = %v", err)
	}

	wantCapabilities := append([]string(nil), card.Capabilities...)
	wantBriefRaw := append(json.RawMessage(nil), card.Ext[capabilityBriefExtKey]...)
	card.Capabilities[0] = "mutated"
	card.Ext[capabilityBriefExtKey][0] = '{'

	if got := cloned.Capabilities; !slices.Equal(got, wantCapabilities) {
		t.Fatalf("cloned capabilities = %#v, want %#v", got, wantCapabilities)
	}
	if got := normalized.Capabilities; !slices.Equal(got, wantCapabilities) {
		t.Fatalf("normalized capabilities = %#v, want %#v", got, wantCapabilities)
	}
	if got := string(cloned.Ext[capabilityBriefExtKey]); got != string(wantBriefRaw) {
		t.Fatalf("cloned capability brief raw = %q, want %q", got, string(wantBriefRaw))
	}
	if got := string(normalized.Ext[capabilityBriefExtKey]); got != string(wantBriefRaw) {
		t.Fatalf("normalized capability brief raw = %q, want %q", got, string(wantBriefRaw))
	}
}

func mustPeerCard(t *testing.T, peerID string) PeerCard {
	t.Helper()

	card, err := DefaultPeerCard(peerID)
	if err != nil {
		t.Fatalf("DefaultPeerCard(%q) error = %v", peerID, err)
	}
	return card
}

func decodeCapabilityBriefPayload(t *testing.T, raw json.RawMessage) []capabilityBrief {
	t.Helper()

	if len(raw) == 0 {
		return nil
	}

	var brief []capabilityBrief
	if err := json.Unmarshal(raw, &brief); err != nil {
		t.Fatalf("json.Unmarshal(capability brief) error = %v", err)
	}
	return brief
}
