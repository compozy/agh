package cli

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestNetworkCommandsAndFormatting(t *testing.T) {
	t.Parallel()

	expiresAt := time.Date(2026, 4, 11, 18, 0, 0, 0, time.UTC).Unix()
	var seenPeersQuery NetworkPeersQuery
	var seenSendRequest NetworkSendRequest

	client := stubClient{
		networkStatusFn: func(context.Context) (NetworkStatusRecord, error) {
			return NetworkStatusRecord{
				Enabled:              true,
				Status:               "running",
				ListenerHost:         "127.0.0.1",
				ListenerPort:         4222,
				LocalPeers:           1,
				RemotePeers:          2,
				Spaces:               1,
				QueuedMessages:       3,
				QueuedSessions:       1,
				DeliveryWorkers:      1,
				MessagesSent:         4,
				MessagesReceived:     5,
				MessagesRejected:     1,
				MessagesDelivered:    3,
				WorkflowTaggedEvents: 2,
				HandoffTaggedEvents:  1,
				LastDisconnect:       "transport lost",
				KindMetrics: []NetworkKindMetricRecord{{
					Kind:      "say",
					Sent:      4,
					Received:  5,
					Rejected:  1,
					Delivered: 3,
				}},
			}, nil
		},
		networkPeersFn: func(_ context.Context, query NetworkPeersQuery) ([]NetworkPeerRecord, error) {
			seenPeersQuery = query
			displayName := "Reviewer"
			sessionID := "sess-a"
			lastSeen := fixedTestNow
			expires := fixedTestNow.Add(time.Minute)
			return []NetworkPeerRecord{{
				PeerID:    "reviewer.sess-a",
				SessionID: &sessionID,
				Space:     "builders",
				Local:     true,
				PeerCard: NetworkPeerCardRecord{
					PeerID:              "reviewer.sess-a",
					DisplayName:         &displayName,
					ProfilesSupported:   []string{"v0"},
					Capabilities:        []string{"send"},
					ArtifactsSupported:  []string{"text"},
					TrustModesSupported: []string{"untrusted"},
				},
				LastSeen:  &lastSeen,
				ExpiresAt: &expires,
			}}, nil
		},
		networkSpacesFn: func(context.Context) ([]NetworkSpaceRecord, error) {
			return []NetworkSpaceRecord{{Space: "builders", PeerCount: 2}}, nil
		},
		networkSendFn: func(_ context.Context, request NetworkSendRequest) (NetworkSendRecord, error) {
			seenSendRequest = request
			return NetworkSendRecord{
				ID:            "msg-1",
				SessionID:     request.SessionID,
				Space:         request.Space,
				Kind:          request.Kind,
				TraceID:       request.TraceID,
				CausationID:   request.CausationID,
				InteractionID: request.InteractionID,
				ReplyTo:       request.ReplyTo,
				ExpiresAt:     request.ExpiresAt,
				Ext:           request.Ext,
			}, nil
		},
		networkInboxFn: func(_ context.Context, sessionID string) ([]NetworkEnvelopeRecord, error) {
			replyTo := "msg-root"
			traceID := "trace-1"
			causationID := "cause-1"
			return []NetworkEnvelopeRecord{{
				Protocol:    "agh-network/v0",
				ID:          "msg-inbox",
				Kind:        "direct",
				Space:       "builders",
				From:        "reviewer.sess-a",
				ReplyTo:     &replyTo,
				TraceID:     &traceID,
				CausationID: &causationID,
				TS:          fixedTestNow.Unix(),
				Body:        mustJSON(t, map[string]any{"text": "review this", "intent": "review"}),
				Ext: map[string]json.RawMessage{
					"agh.workflow_id":     mustJSON(t, "wf-1"),
					"agh.handoff_version": mustJSON(t, 3),
				},
			}}, nil
		},
	}
	deps := newTestDeps(t, client)

	statusOut, _, err := executeRootCommand(t, deps, "network", "status", "-o", "human")
	if err != nil {
		t.Fatalf("network status error = %v", err)
	}
	if !strings.Contains(statusOut, "Network") || !strings.Contains(statusOut, "Queued Messages") || !strings.Contains(statusOut, "Kind Metrics") {
		t.Fatalf("network status output = %q, want summary and metrics", statusOut)
	}

	peersOut, _, err := executeRootCommand(t, deps, "network", "peers", "builders", "-o", "json")
	if err != nil {
		t.Fatalf("network peers error = %v", err)
	}
	if seenPeersQuery.Space != "builders" {
		t.Fatalf("seenPeersQuery.Space = %q, want builders", seenPeersQuery.Space)
	}
	var peers []NetworkPeerRecord
	if err := json.Unmarshal([]byte(peersOut), &peers); err != nil {
		t.Fatalf("json.Unmarshal(network peers) error = %v", err)
	}
	if len(peers) != 1 || peers[0].PeerID != "reviewer.sess-a" {
		t.Fatalf("peers = %#v, want one reviewer peer", peers)
	}

	spacesOut, _, err := executeRootCommand(t, deps, "network", "spaces", "-o", "toon")
	if err != nil {
		t.Fatalf("network spaces error = %v", err)
	}
	if !strings.Contains(spacesOut, "network_spaces[1]{space,peer_count}:") {
		t.Fatalf("network spaces toon = %q, want TOON list", spacesOut)
	}

	sendOut, _, err := executeRootCommand(t, deps,
		"network", "send",
		"--session", "sess-a",
		"--space", "builders",
		"--kind", "say",
		"--body", `{"text":"hello"}`,
		"--interaction-id", "int-1",
		"--reply-to", "msg-root",
		"--trace-id", "trace-1",
		"--causation-id", "cause-1",
		"--expires-at", "2026-04-11T18:00:00Z",
		"--ext", `{"agh.workflow_id":"wf-1","agh.handoff_version":3}`,
		"-o", "json",
	)
	if err != nil {
		t.Fatalf("network send error = %v", err)
	}
	if string(seenSendRequest.Body) != `{"text":"hello"}` {
		t.Fatalf("seenSendRequest.Body = %s, want body JSON", string(seenSendRequest.Body))
	}
	if seenSendRequest.ExpiresAt == nil || *seenSendRequest.ExpiresAt != expiresAt {
		t.Fatalf("seenSendRequest.ExpiresAt = %#v, want %d", seenSendRequest.ExpiresAt, expiresAt)
	}
	if string(seenSendRequest.Ext["agh.workflow_id"]) != `"wf-1"` || string(seenSendRequest.Ext["agh.handoff_version"]) != `3` {
		t.Fatalf("seenSendRequest.Ext = %#v, want workflow metadata", seenSendRequest.Ext)
	}
	var sent NetworkSendRecord
	if err := json.Unmarshal([]byte(sendOut), &sent); err != nil {
		t.Fatalf("json.Unmarshal(network send) error = %v", err)
	}
	if sent.ID != "msg-1" || sent.TraceID != "trace-1" {
		t.Fatalf("sent = %#v, want sent payload", sent)
	}

	inboxOut, _, err := executeRootCommand(t, deps, "network", "inbox", "--session", "sess-a", "-o", "human")
	if err != nil {
		t.Fatalf("network inbox error = %v", err)
	}
	if !strings.Contains(inboxOut, "wf-1") || !strings.Contains(inboxOut, "3") {
		t.Fatalf("network inbox output = %q, want workflow and handoff metadata", inboxOut)
	}
}

func TestNetworkSendParsersRejectInvalidFlags(t *testing.T) {
	t.Parallel()

	deps := newTestDeps(t, stubClient{
		networkSendFn: func(context.Context, NetworkSendRequest) (NetworkSendRecord, error) {
			return NetworkSendRecord{}, nil
		},
	})

	if _, _, err := executeRootCommand(t, deps,
		"network", "send",
		"--session", "sess-a",
		"--space", "builders",
		"--kind", "say",
		"--body", `not-json`,
	); err == nil || !strings.Contains(err.Error(), "--body must be valid JSON") {
		t.Fatalf("invalid --body error = %v, want JSON validation", err)
	}

	if _, _, err := executeRootCommand(t, deps,
		"network", "send",
		"--session", "sess-a",
		"--space", "builders",
		"--kind", "say",
		"--body", `{"text":"ok"}`,
		"--ext", `[]`,
	); err == nil || !strings.Contains(err.Error(), "--ext must be a JSON object") {
		t.Fatalf("invalid --ext error = %v, want object validation", err)
	}

	if _, _, err := executeRootCommand(t, deps,
		"network", "send",
		"--session", "sess-a",
		"--space", "builders",
		"--kind", "say",
		"--body", `{"text":"ok"}`,
		"--expires-at", `tomorrow`,
	); err == nil || !strings.Contains(err.Error(), "--expires-at must be unix seconds or RFC3339") {
		t.Fatalf("invalid --expires-at error = %v, want timestamp validation", err)
	}
}
