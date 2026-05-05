package httpapi

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/api/testutil"
	"github.com/pedronauck/agh/internal/network"
	"github.com/pedronauck/agh/internal/store"
)

func TestNetworkDirectResolveCreatesRoom(t *testing.T) {
	t.Parallel()

	t.Run("Should create deterministic direct room", func(t *testing.T) {
		homePaths := newTestHomePaths(t)
		handlers := newTestHandlers(t, stubSessionManager{}, stubObserver{}, homePaths)
		handlers.Config.Network.Enabled = true

		localSessionID := "sess-local"
		handlers.Network = testutil.StubNetworkService{
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
