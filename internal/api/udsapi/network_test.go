package udsapi

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/api/testutil"
	"github.com/pedronauck/agh/internal/network"
	"github.com/pedronauck/agh/internal/store"
)

func TestNetworkHandlersValidateRequestsAndMapErrors(t *testing.T) {
	t.Parallel()

	homePaths := newTestHomePaths(t)
	handlers := newTestHandlers(t, stubSessionManager{}, stubObserver{}, homePaths)
	handlers.Config.Network.Enabled = true
	sendCalls := 0
	handlers.Network = stubNetworkService{
		SendFn: func(context.Context, network.SendRequest) (string, error) {
			sendCalls++
			return "", nil
		},
	}
	engine := newTestRouter(t, handlers)

	inboxResp := performRequest(t, engine, http.MethodGet, "/api/network/inbox", nil)
	if inboxResp.Code != http.StatusBadRequest {
		t.Fatalf("inbox status = %d, want %d", inboxResp.Code, http.StatusBadRequest)
	}
	if !strings.Contains(inboxResp.Body.String(), "session_id query is required") {
		t.Fatalf("inbox body = %q, want session_id validation", inboxResp.Body.String())
	}

	sendResp := performRequest(t, engine, http.MethodPost, "/api/network/send", []byte(`{}`))
	if sendResp.Code != http.StatusBadRequest {
		t.Fatalf("send status = %d, want %d; body=%s", sendResp.Code, http.StatusBadRequest, sendResp.Body.String())
	}
	if !strings.Contains(sendResp.Body.String(), "session_id is required") {
		t.Fatalf("send body = %q, want session_id validation", sendResp.Body.String())
	}

	t.Run("Should reject raw claim tokens before sending network messages", func(t *testing.T) {
		rawTokenResp := performRequest(
			t,
			engine,
			http.MethodPost,
			"/api/network/send",
			[]byte(
				`{"session_id":"sess-a","channel":"builders","kind":"say","body":{"claim_token":"agh_claim_uds"}}`,
			),
		)
		if rawTokenResp.Code != http.StatusBadRequest {
			t.Fatalf(
				"raw token send status = %d, want %d; body=%s",
				rawTokenResp.Code,
				http.StatusBadRequest,
				rawTokenResp.Body.String(),
			)
		}
		if !strings.Contains(rawTokenResp.Body.String(), "network_raw_token_rejected") {
			t.Fatalf("raw token send body = %q, want network_raw_token_rejected", rawTokenResp.Body.String())
		}
		if sendCalls != 0 {
			t.Fatalf("Network.Send calls = %d, want 0 for invalid send requests", sendCalls)
		}
	})
}

func TestNetworkHandlersPreserveWorkflowMetadata(t *testing.T) {
	t.Parallel()

	t.Run("Should preserve workflow metadata", func(t *testing.T) {
		homePaths := newTestHomePaths(t)
		handlers := newTestHandlers(t, stubSessionManager{}, stubObserver{}, homePaths)
		handlers.Config.Network.Enabled = true

		var seenRequest network.SendRequest
		handlers.Network = stubNetworkService{
			SendFn: func(_ context.Context, req network.SendRequest) (string, error) {
				seenRequest = req
				return "msg-1", nil
			},
			InboxFn: func(_ context.Context, _ string) ([]network.Envelope, error) {
				return []network.Envelope{{
					Protocol: network.ProtocolV0,
					ID:       "msg-inbox",
					Kind:     network.KindSay,
					Channel:  "builders",
					From:     "reviewer.sess-a",
					TS:       1775823000,
					Body:     json.RawMessage(`{"text":"review this","intent":"review"}`),
					Ext: network.ExtensionMap{
						"agh.workflow_id":     json.RawMessage(`"wf-1"`),
						"agh.handoff_version": json.RawMessage(`3`),
					},
				}}, nil
			},
		}
		engine := newTestRouter(t, handlers)

		sendResp := performRequest(
			t,
			engine,
			http.MethodPost,
			"/api/network/send",
			[]byte(
				`{"session_id":"sess-a","channel":"builders","surface":"thread","thread_id":"thread_launch_db","kind":"say","body":{"text":"hello"},"ext":{"agh.workflow_id":"wf-1","agh.handoff_version":3}}`,
			),
		)
		if sendResp.Code != http.StatusOK {
			t.Fatalf("send status = %d, want %d; body=%s", sendResp.Code, http.StatusOK, sendResp.Body.String())
		}
		if string(seenRequest.Ext["agh.workflow_id"]) != `"wf-1"` ||
			string(seenRequest.Ext["agh.handoff_version"]) != `3` {
			t.Fatalf("seenRequest.Ext = %#v, want preserved workflow metadata", seenRequest.Ext)
		}
		if seenRequest.Channel != "builders" {
			t.Fatalf("seenRequest.Channel = %q, want builders", seenRequest.Channel)
		}
		if seenRequest.Surface == nil || *seenRequest.Surface != network.SurfaceThread {
			t.Fatalf("seenRequest.Surface = %#v, want thread", seenRequest.Surface)
		}
		if seenRequest.ThreadID == nil || *seenRequest.ThreadID != "thread_launch_db" {
			t.Fatalf("seenRequest.ThreadID = %#v, want thread_launch_db", seenRequest.ThreadID)
		}

		inboxResp := performRequest(t, engine, http.MethodGet, "/api/network/inbox?session_id=sess-a", nil)
		if inboxResp.Code != http.StatusOK {
			t.Fatalf("inbox status = %d, want %d", inboxResp.Code, http.StatusOK)
		}
		if !strings.Contains(inboxResp.Body.String(), `"channel":"builders"`) ||
			!strings.Contains(inboxResp.Body.String(), `"agh.workflow_id":"wf-1"`) ||
			!strings.Contains(inboxResp.Body.String(), `"agh.handoff_version":3`) {
			t.Fatalf("inbox body = %s, want workflow metadata", inboxResp.Body.String())
		}
	})
}

func TestNetworkDirectResolveCreatesRoom(t *testing.T) {
	t.Parallel()

	t.Run("Should create deterministic direct room", func(t *testing.T) {
		homePaths := newTestHomePaths(t)
		handlers := newTestHandlers(t, stubSessionManager{}, stubObserver{}, homePaths)
		handlers.Config.Network.Enabled = true

		localSessionID := "sess-local"
		handlers.Network = stubNetworkService{
			ListPeersFn: func(_ context.Context, channel string) ([]network.PeerInfo, error) {
				if channel != "builders" {
					t.Fatalf("ListPeers() channel = %q, want builders", channel)
				}
				return []network.PeerInfo{
					{
						PeerID:    "coder.sess-abc",
						Channel:   "builders",
						Local:     true,
						SessionID: &localSessionID,
					},
					{
						PeerID:  "reviewer.sess-xyz",
						Channel: "builders",
					},
				}, nil
			},
		}

		wantDirectID, wantPeerA, wantPeerB, err := network.DirectRoomIdentity(
			"builders",
			"coder.sess-abc",
			"reviewer.sess-xyz",
		)
		if err != nil {
			t.Fatalf("DirectRoomIdentity() error = %v", err)
		}
		handlers.NetworkStore = testutil.StubNetworkStore{
			ResolveDirectRoomFn: func(
				_ context.Context,
				entry store.NetworkDirectRoomEntry,
			) (store.NetworkDirectRoomSummary, error) {
				if entry.Channel != "builders" ||
					entry.DirectID != wantDirectID ||
					entry.PeerA != wantPeerA ||
					entry.PeerB != wantPeerB {
					t.Fatalf("ResolveDirectRoom() entry = %#v, want deterministic direct-room identity", entry)
				}
				return store.NetworkDirectRoomSummary{
					Channel:        entry.Channel,
					DirectID:       entry.DirectID,
					PeerA:          entry.PeerA,
					PeerB:          entry.PeerB,
					OpenedAt:       time.Date(2026, 4, 3, 12, 0, 1, 0, time.UTC),
					LastActivityAt: time.Date(2026, 4, 3, 12, 0, 1, 0, time.UTC),
				}, nil
			},
		}

		engine := newTestRouter(t, handlers)
		resp := performRequest(
			t,
			engine,
			http.MethodPost,
			"/api/network/channels/builders/directs/resolve",
			[]byte(`{"session_id":"sess-local","peer_id":"reviewer.sess-xyz"}`),
		)
		if resp.Code != http.StatusOK {
			t.Fatalf("direct resolve status = %d, want %d; body=%s", resp.Code, http.StatusOK, resp.Body.String())
		}

		var payload contract.NetworkDirectRoomResponse
		decodeJSONResponse(t, resp, &payload)
		if payload.Direct.DirectID != wantDirectID ||
			payload.Direct.PeerA != wantPeerA ||
			payload.Direct.PeerB != wantPeerB {
			t.Fatalf("direct resolve payload = %#v, want deterministic room", payload.Direct)
		}
	})
}

func TestNetworkHandlersExposeTypedCapabilityPayloads(t *testing.T) {
	t.Parallel()

	t.Run("Should expose typed capability payloads", func(t *testing.T) {
		homePaths := newTestHomePaths(t)
		handlers := newTestHandlers(t, stubSessionManager{}, stubObserver{}, homePaths)
		handlers.Config.Network.Enabled = true
		handlers.Network = stubNetworkService{
			ListPeersFn: func(context.Context, string) ([]network.PeerInfo, error) {
				return []network.PeerInfo{{
					PeerID:  "reviewer.sess-a",
					Channel: "builders",
					Local:   true,
					PeerCard: network.PeerCard{
						PeerID:              "reviewer.sess-a",
						Capabilities:        []string{"review-pr"},
						ProfilesSupported:   []string{network.ProtocolV0},
						ArtifactsSupported:  []string{"capability"},
						TrustModesSupported: []string{"untrusted"},
						Ext: network.ExtensionMap{
							"agh.capabilities_brief": json.RawMessage(
								`[{"id":"review-pr","summary":"Review pull requests"}]`,
							),
							"agh.workflow_id": json.RawMessage(`"wf-1"`),
						},
					},
				}}, nil
			},
		}
		engine := newTestRouter(t, handlers)

		resp := performRequest(t, engine, http.MethodGet, "/api/network/peers", nil)
		if resp.Code != http.StatusOK {
			t.Fatalf("peers status = %d, want %d; body=%s", resp.Code, http.StatusOK, resp.Body.String())
		}

		var payload contract.NetworkPeersResponse
		if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
			t.Fatalf("json.Unmarshal(peers) error = %v", err)
		}
		if got, want := payload.Peers[0].PeerCard.Capabilities, []contract.NetworkCapabilityBriefPayload{{
			ID:      "review-pr",
			Summary: "Review pull requests",
		}}; !reflect.DeepEqual(got, want) {
			t.Fatalf("peer capabilities = %#v, want %#v", got, want)
		}
		if _, ok := payload.Peers[0].PeerCard.Ext["agh.capabilities_brief"]; ok {
			t.Fatalf("capability brief ext should be stripped: %#v", payload.Peers[0].PeerCard.Ext)
		}
		if got, want := string(payload.Peers[0].PeerCard.Ext["agh.workflow_id"]), `"wf-1"`; got != want {
			t.Fatalf("workflow ext = %q, want %q", got, want)
		}
	})
}
