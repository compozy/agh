package network

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func BenchmarkFormatNetworkMessageDirect(b *testing.B) {
	envelope := benchmarkDirectEnvelope(b)

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

func BenchmarkFormatNetworkMessageGuidanceModes(b *testing.B) {
	envelope := benchmarkDirectEnvelope(b)
	cases := []struct {
		name string
		mode networkMessageGuidanceMode
	}{
		{name: "verbose", mode: networkMessageGuidanceVerbose},
		{name: "compact", mode: networkMessageGuidanceCompact},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()

			var rendered string
			for b.Loop() {
				var err error
				rendered, err = formatNetworkMessageWithGuidance(envelope, tc.mode)
				if err != nil {
					b.Fatalf("formatNetworkMessageWithGuidance(%s) error = %v", tc.name, err)
				}
				if rendered == "" {
					b.Fatal("formatNetworkMessageWithGuidance() = empty, want content")
				}
			}
			b.ReportMetric(float64(len(rendered)), "bytes/message")
			b.ReportMetric(float64((len(rendered)+3)/4), "est_tokens/message")
		})
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
			if _, err := registry.RegisterLocal(sessionID, testWorkspaceID, channel, card, now); err != nil {
				b.Fatalf("RegisterLocal(%q, %q) error = %v", sessionID, channel, err)
			}
		}

		for remoteIdx := range remotePerChannel {
			peerID := fmt.Sprintf("remote.%02d.%02d", channelIdx, remoteIdx)
			card := benchmarkPeerCard(b, peerID, "bench-remote")
			if _, stored, err := registry.RefreshRemote(testWorkspaceID, channel, card, now); err != nil {
				b.Fatalf("RefreshRemote(%q, %q) error = %v", channel, peerID, err)
			} else if !stored {
				b.Fatalf("RefreshRemote(%q, %q) stored = false, want true", channel, peerID)
			}
		}
	}

	b.ReportAllocs()

	for b.Loop() {
		peers := registry.ListPeers(testWorkspaceID, targetChannel, now)
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
		To:      new("reviewer.sess-bench"),
		ReplyTo: new("msg-root-bench"),
		TraceID: new("trace-bench"),
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

func benchmarkDirectEnvelope(b *testing.B) Envelope {
	b.Helper()

	return Envelope{
		Protocol:    ProtocolV0,
		WorkspaceID: testWorkspaceID,
		ID:          "msg-bench-direct",
		Kind:        KindSay,
		Channel:     "builders",
		Surface:     new(SurfaceDirect),
		DirectID:    new(testDirectRef().DirectID),
		From:        "coder.sess-bench",
		To:          new("reviewer.sess-bench"),
		WorkID:      new("work_bench-direct"),
		ReplyTo:     new("msg-root-bench"),
		TraceID:     new("trace-bench-direct"),
		CausationID: new("msg-cause-bench"),
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
