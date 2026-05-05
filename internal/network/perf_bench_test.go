package network

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func BenchmarkFormatNetworkMessageDirect(b *testing.B) {
	envelope := Envelope{
		Protocol:    ProtocolV0,
		ID:          "msg-bench-direct",
		Kind:        KindSay,
		Channel:     "builders",
		From:        "coder.sess-bench",
		To:          stringPtr("reviewer.sess-bench"),
		WorkID:      stringPtr("int-bench-direct"),
		ReplyTo:     stringPtr("msg-root-bench"),
		TraceID:     stringPtr("trace-bench-direct"),
		CausationID: stringPtr("msg-cause-bench"),
		TS:          time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC).Unix(),
		Body: benchmarkRawJSON(b, SayBody{
			Text:   "Review the attached network benchmark payload and summarize the reply strategy.",
			Intent: "review_request",
			Artifacts: []json.RawMessage{
				json.RawMessage(`{"kind":"capability","id":"artifact-1"}`),
				json.RawMessage(`{"kind":"patch","id":"artifact-2"}`),
			},
		}),
	}

	b.ReportAllocs()

	for b.Loop() {
		rendered, err := formatNetworkMessage(envelope)
		if err != nil {
			b.Fatalf("formatNetworkMessage() error = %v", err)
		}
		if rendered == "" {
			b.Fatal("formatNetworkMessage() = empty, want content")
		}
	}
}

func BenchmarkPeerRegistryListPeersFiltered(b *testing.B) {
	now := time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)
	registry, err := NewPeerRegistry(30*time.Second, WithPeerRegistryClock(func() time.Time { return now }))
	if err != nil {
		b.Fatalf("NewPeerRegistry() error = %v", err)
	}

	targetChannel := "chan15"
	const (
		channelCount      = 32
		localPerChannel   = 12
		remotePerChannel  = 12
		expectedPeerCount = localPerChannel + remotePerChannel
	)

	for channelIdx := range channelCount {
		channel := fmt.Sprintf("chan%02d", channelIdx)

		for localIdx := range localPerChannel {
			peerID := fmt.Sprintf("local.%02d.%02d", channelIdx, localIdx)
			card := benchmarkPeerCard(b, peerID, "bench-local")
			sessionID := fmt.Sprintf("sess-%02d-%02d", channelIdx, localIdx)
			if _, err := registry.RegisterLocal(sessionID, channel, card, now); err != nil {
				b.Fatalf("RegisterLocal(%q, %q) error = %v", sessionID, channel, err)
			}
		}

		for remoteIdx := range remotePerChannel {
			peerID := fmt.Sprintf("remote.%02d.%02d", channelIdx, remoteIdx)
			card := benchmarkPeerCard(b, peerID, "bench-remote")
			if _, stored, err := registry.RefreshRemote(channel, card, now); err != nil {
				b.Fatalf("RefreshRemote(%q, %q) error = %v", channel, peerID, err)
			} else if !stored {
				b.Fatalf("RefreshRemote(%q, %q) stored = false, want true", channel, peerID)
			}
		}
	}

	b.ReportAllocs()

	for b.Loop() {
		peers := registry.ListPeers(targetChannel, now)
		if len(peers) != expectedPeerCount {
			b.Fatalf("len(ListPeers(%q)) = %d, want %d", targetChannel, len(peers), expectedPeerCount)
		}
	}
}

func BenchmarkNetworkLogFields(b *testing.B) {
	envelope := Envelope{
		ID:      "msg-bench-log",
		Kind:    KindCapability,
		Channel: "builders",
		From:    "coder.sess-bench",
		To:      stringPtr("reviewer.sess-bench"),
		ReplyTo: stringPtr("msg-root-bench"),
		TraceID: stringPtr("trace-bench"),
		Ext: ExtensionMap{
			"agh.workflow_id":     json.RawMessage(`"wf-bench-001"`),
			"agh.handoff_version": json.RawMessage(`"1"`),
			"agh.handoff_digest":  json.RawMessage(`"sha256:abcdef"`),
			"agh.handoff_source":  json.RawMessage(`{"peer":"reviewer.sess-bench","kind":"handoff"}`),
		},
	}

	b.ReportAllocs()

	for b.Loop() {
		fields := networkLogFields(envelope, "session_id", "sess-bench")
		if len(fields) == 0 {
			b.Fatal("networkLogFields() = empty, want structured fields")
		}
	}
}

func benchmarkPeerCard(b *testing.B, peerID string, capability string) PeerCard {
	b.Helper()

	displayName := "Bench Peer"
	card := PeerCard{
		PeerID:              peerID,
		DisplayName:         &displayName,
		ProfilesSupported:   []string{ProtocolV0},
		Capabilities:        []string{capability, "chat.review"},
		ArtifactsSupported:  []string{"capability"},
		TrustModesSupported: []string{"unverified"},
	}
	normalized, err := normalizePeerCard(card)
	if err != nil {
		b.Fatalf("normalizePeerCard(%q) error = %v", peerID, err)
	}
	return normalized
}

func benchmarkRawJSON(b *testing.B, value any) json.RawMessage {
	b.Helper()

	payload, err := json.Marshal(value)
	if err != nil {
		b.Fatalf("json.Marshal() error = %v", err)
	}
	return payload
}
