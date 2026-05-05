package cli

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/pedronauck/agh/internal/api/contract"
)

func TestUnixSocketClientNetworkMethods(t *testing.T) {
	t.Parallel()

	t.Run("Should call network routes through the unix socket client", func(t *testing.T) {
		t.Parallel()

		directID := "direct_99401d24bee62651d189e5a561785466"
		client := &unixSocketClient{
			socketPath: "/tmp/agh.sock",
			httpClient: &http.Client{
				Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
					switch {
					case req.Method == http.MethodGet && req.URL.Path == "/api/network/status":
						return newHTTPResponse(
							http.StatusOK,
							`{"network":{"enabled":true,"status":"running","listener_host":"127.0.0.1","listener_port":4222,"local_peers":1,"remote_peers":2,"channels":1,"queued_messages":3,"queued_sessions":1,"delivery_workers":1,"messages_sent":4,"messages_received":5,"messages_rejected":1,"messages_delivered":3,"workflow_tagged_events":2,"handoff_tagged_events":1,"kind_metrics":[{"kind":"say","sent":4,"received":5,"rejected":1,"delivered":3}]}}`,
						), nil
					case req.Method == http.MethodGet && req.URL.Path == "/api/network/peers":
						if got := req.URL.Query().Get("channel"); got != "builders" {
							t.Fatalf("network peers channel query = %q, want builders", got)
						}
						return newHTTPResponse(
							http.StatusOK,
							`{"peers":[{"peer_id":"reviewer.sess-a","session_id":"sess-a","channel":"builders","local":true,"peer_card":{"peer_id":"reviewer.sess-a","display_name":"Reviewer","profiles_supported":["v0"],"capabilities":[{"id":"send","summary":"Send direct messages"}],"artifacts_supported":["text"],"trust_modes_supported":["untrusted"]}}]}`,
						), nil
					case req.Method == http.MethodGet && req.URL.Path == "/api/network/channels":
						return newHTTPResponse(
							http.StatusOK,
							`{"channels":[{"channel":"builders","peer_count":2}]}`,
						), nil
					case req.Method == http.MethodGet && req.URL.Path == "/api/network/channels/builders/threads":
						if got := req.URL.Query().Get("limit"); got != "2" {
							t.Fatalf("network threads limit query = %q, want 2", got)
						}
						if got := req.URL.Query().Get("after"); got != "thread_0" {
							t.Fatalf("network threads after query = %q, want thread_0", got)
						}
						return newHTTPResponse(
							http.StatusOK,
							`{"threads":[{"channel":"builders","thread_id":"thread_launch","root_message_id":"msg-root","opened_by_peer_id":"coder.sess-a","message_count":2,"participant_count":2,"open_work_count":1,"last_message_preview":"ready"}]}`,
						), nil
					case req.Method == http.MethodGet && req.URL.Path == "/api/network/channels/builders/threads/thread_launch":
						return newHTTPResponse(
							http.StatusOK,
							`{"thread":{"channel":"builders","thread_id":"thread_launch","root_message_id":"msg-root","opened_by_peer_id":"coder.sess-a","message_count":2,"participant_count":2,"open_work_count":1,"last_message_preview":"ready"}}`,
						), nil
					case req.Method == http.MethodGet &&
						req.URL.Path == "/api/network/channels/builders/threads/thread_launch/messages":
						if got := req.URL.Query().Get("limit"); got != "2" {
							t.Fatalf("network thread messages limit query = %q, want 2", got)
						}
						if got := req.URL.Query().Get("before"); got != "msg-3" {
							t.Fatalf("network thread messages before query = %q, want msg-3", got)
						}
						if got := req.URL.Query().Get("after"); got != "msg-1" {
							t.Fatalf("network thread messages after query = %q, want msg-1", got)
						}
						if got := req.URL.Query().Get("kind"); got != "say" {
							t.Fatalf("network thread messages kind query = %q, want say", got)
						}
						if got := req.URL.Query().Get("work_id"); got != "work_1" {
							t.Fatalf("network thread messages work_id query = %q, want work_1", got)
						}
						return newHTTPResponse(
							http.StatusOK,
							`{"messages":[{"message_id":"msg-thread-1","channel":"builders","surface":"thread","thread_id":"thread_launch","kind":"say","direction":"outbound","peer_from":"coder.sess-a","work_id":"work_1","preview_text":"ready","body":{"text":"ready"},"timestamp":"2026-04-03T12:00:00Z"}]}`,
						), nil
					case req.Method == http.MethodGet && req.URL.Path == "/api/network/channels/builders/directs":
						if got := req.URL.Query().Get("peer_id"); got != "reviewer.sess-b" {
							t.Fatalf("network directs peer_id query = %q, want reviewer.sess-b", got)
						}
						return newHTTPResponse(
							http.StatusOK,
							`{"directs":[{"channel":"builders","direct_id":"direct_99401d24bee62651d189e5a561785466","peer_a":"coder.sess-a","peer_b":"reviewer.sess-b","message_count":1,"open_work_count":1,"last_message_preview":"please review"}]}`,
						), nil
					case req.Method == http.MethodPost &&
						req.URL.Path == "/api/network/channels/builders/directs/resolve":
						body, err := io.ReadAll(req.Body)
						if err != nil {
							t.Fatalf("io.ReadAll(network direct resolve body) error = %v", err)
						}
						if !strings.Contains(string(body), `"session_id":"sess-a"`) ||
							!strings.Contains(string(body), `"peer_id":"reviewer.sess-b"`) {
							t.Fatalf("network direct resolve body = %s, want session and peer", body)
						}
						return newHTTPResponse(
							http.StatusOK,
							`{"direct":{"channel":"builders","direct_id":"direct_99401d24bee62651d189e5a561785466","peer_a":"coder.sess-a","peer_b":"reviewer.sess-b","message_count":1,"open_work_count":1,"last_message_preview":"please review"}}`,
						), nil
					case req.Method == http.MethodGet &&
						req.URL.Path == "/api/network/channels/builders/directs/direct_99401d24bee62651d189e5a561785466":
						return newHTTPResponse(
							http.StatusOK,
							`{"direct":{"channel":"builders","direct_id":"direct_99401d24bee62651d189e5a561785466","peer_a":"coder.sess-a","peer_b":"reviewer.sess-b","message_count":1,"open_work_count":1,"last_message_preview":"please review"}}`,
						), nil
					case req.Method == http.MethodGet &&
						req.URL.Path == "/api/network/channels/builders/directs/direct_99401d24bee62651d189e5a561785466/messages":
						if got := req.URL.Query().Get("limit"); got != "2" {
							t.Fatalf("network direct messages limit query = %q, want 2", got)
						}
						if got := req.URL.Query().Get("work_id"); got != "work_1" {
							t.Fatalf("network direct messages work_id query = %q, want work_1", got)
						}
						return newHTTPResponse(
							http.StatusOK,
							`{"messages":[{"message_id":"msg-direct-1","channel":"builders","surface":"direct","direct_id":"direct_99401d24bee62651d189e5a561785466","kind":"say","direction":"outbound","peer_from":"coder.sess-a","peer_to":"reviewer.sess-b","work_id":"work_1","preview_text":"please review","body":{"text":"please review"},"timestamp":"2026-04-03T12:00:00Z"}]}`,
						), nil
					case req.Method == http.MethodGet && req.URL.Path == "/api/network/work/work_1":
						return newHTTPResponse(
							http.StatusOK,
							`{"work":{"work_id":"work_1","channel":"builders","surface":"direct","direct_id":"direct_99401d24bee62651d189e5a561785466","opened_by_peer_id":"coder.sess-a","opened_session_id":"sess-a","target_peer_id":"reviewer.sess-b","state":"open"}}`,
						), nil
					case req.Method == http.MethodPost && req.URL.Path == "/api/network/send":
						body, err := io.ReadAll(req.Body)
						if err != nil {
							t.Fatalf("io.ReadAll(network send body) error = %v", err)
						}
						if !strings.Contains(string(body), `"channel":"builders"`) ||
							!strings.Contains(string(body), `"surface":"thread"`) ||
							!strings.Contains(string(body), `"thread_id":"thread_launch"`) ||
							!strings.Contains(string(body), `"work_id":"work_1"`) ||
							!strings.Contains(string(body), `"agh.workflow_id":"wf-1"`) ||
							!strings.Contains(string(body), `"agh.handoff_version":3`) {
							t.Fatalf("network send body = %s, want ext metadata", body)
						}
						return newHTTPResponse(
							http.StatusOK,
							`{"message":{"id":"msg-1","session_id":"sess-a","channel":"builders","surface":"thread","thread_id":"thread_launch","kind":"say","work_id":"work_1","ext":{"agh.workflow_id":"wf-1","agh.handoff_version":3}}}`,
						), nil
					case req.Method == http.MethodGet && req.URL.Path == "/api/network/inbox":
						if got := req.URL.Query().Get("session_id"); got != "sess-a" {
							t.Fatalf("network inbox session_id query = %q, want sess-a", got)
						}
						return newHTTPResponse(
							http.StatusOK,
							`{"messages":[{"protocol":"agh-network/v0","id":"msg-inbox","kind":"say","channel":"builders","surface":"direct","direct_id":"direct_99401d24bee62651d189e5a561785466","from":"reviewer.sess-a","work_id":"work_1","ts":1775823000,"body":{"text":"review this","intent":"review"},"ext":{"agh.workflow_id":"wf-1","agh.handoff_version":3}}]}`,
						), nil
					default:
						return newHTTPResponse(http.StatusNotFound, `{"error":"missing"}`), nil
					}
				}),
			},
		}

		ctx := context.Background()

		status, err := client.NetworkStatus(ctx)
		if err != nil || status.MessagesDelivered != 3 || len(status.KindMetrics) != 1 {
			t.Fatalf("NetworkStatus() = %#v, %v", status, err)
		}

		peers, err := client.NetworkPeers(ctx, NetworkPeersQuery{Channel: "builders"})
		if err != nil || len(peers) != 1 || peers[0].PeerID != "reviewer.sess-a" {
			t.Fatalf("NetworkPeers() = %#v, %v", peers, err)
		}

		channels, err := client.NetworkChannels(ctx)
		if err != nil || len(channels) != 1 || channels[0].PeerCount != 2 {
			t.Fatalf("NetworkChannels() = %#v, %v", channels, err)
		}

		threads, err := client.NetworkThreads(ctx, NetworkThreadsQuery{
			Channel: "builders",
			Limit:   2,
			After:   "thread_0",
		})
		if err != nil || len(threads) != 1 || threads[0].ThreadID != "thread_launch" {
			t.Fatalf("NetworkThreads() = %#v, %v", threads, err)
		}

		thread, err := client.NetworkThread(ctx, "builders", "thread_launch")
		if err != nil || thread.ThreadID != "thread_launch" {
			t.Fatalf("NetworkThread() = %#v, %v", thread, err)
		}

		threadMessages, err := client.NetworkThreadMessages(ctx, NetworkConversationMessagesQuery{
			Channel:  "builders",
			ThreadID: "thread_launch",
			Limit:    2,
			Before:   "msg-3",
			After:    "msg-1",
			Kind:     "say",
			WorkID:   "work_1",
		})
		if err != nil || len(threadMessages) != 1 || threadMessages[0].MessageID != "msg-thread-1" {
			t.Fatalf("NetworkThreadMessages() = %#v, %v", threadMessages, err)
		}

		directs, err := client.NetworkDirects(ctx, NetworkDirectsQuery{
			Channel: "builders",
			PeerID:  "reviewer.sess-b",
		})
		if err != nil || len(directs) != 1 || directs[0].DirectID != directID {
			t.Fatalf("NetworkDirects() = %#v, %v", directs, err)
		}

		resolvedDirect, err := client.NetworkDirectResolve(ctx, "builders", NetworkDirectResolveRequest{
			SessionID: "sess-a",
			PeerID:    "reviewer.sess-b",
		})
		if err != nil || resolvedDirect.DirectID != directID {
			t.Fatalf("NetworkDirectResolve() = %#v, %v", resolvedDirect, err)
		}

		direct, err := client.NetworkDirect(ctx, "builders", directID)
		if err != nil || direct.DirectID != directID {
			t.Fatalf("NetworkDirect() = %#v, %v", direct, err)
		}

		directMessages, err := client.NetworkDirectMessages(ctx, NetworkConversationMessagesQuery{
			Channel:  "builders",
			DirectID: directID,
			Limit:    2,
			WorkID:   "work_1",
		})
		if err != nil || len(directMessages) != 1 || directMessages[0].MessageID != "msg-direct-1" {
			t.Fatalf("NetworkDirectMessages() = %#v, %v", directMessages, err)
		}

		work, err := client.NetworkWork(ctx, "work_1")
		if err != nil || work.WorkID != "work_1" || work.DirectID != directID {
			t.Fatalf("NetworkWork() = %#v, %v", work, err)
		}

		sent, err := client.NetworkSend(ctx, NetworkSendRequest{
			SessionID: "sess-a",
			Channel:   "builders",
			Surface:   "thread",
			ThreadID:  "thread_launch",
			Kind:      "say",
			Body:      json.RawMessage(`{"text":"hello"}`),
			WorkID:    "work_1",
			Ext: map[string]json.RawMessage{
				"agh.workflow_id":     json.RawMessage(`"wf-1"`),
				"agh.handoff_version": json.RawMessage(`3`),
			},
		})
		if err != nil || sent.ID != "msg-1" || sent.ThreadID != "thread_launch" || sent.WorkID != "work_1" {
			t.Fatalf("NetworkSend() = %#v, %v", sent, err)
		}

		inbox, err := client.NetworkInbox(ctx, "sess-a")
		if err != nil || len(inbox) != 1 || string(inbox[0].Ext["agh.workflow_id"]) != `"wf-1"` {
			t.Fatalf("NetworkInbox() = %#v, %v", inbox, err)
		}
	})
}

func TestNetworkClientHelpersAndAliases(t *testing.T) {
	t.Parallel()

	t.Run("Should build network query helpers and preserve contract aliases", func(t *testing.T) {
		t.Parallel()

		if got := networkPeersValues(NetworkPeersQuery{Channel: "builders"}); got.Get("channel") != "builders" {
			t.Fatalf("networkPeersValues() = %v, want channel filter", got)
		}
		if got := networkInboxValues("sess-a"); got.Get("session_id") != "sess-a" {
			t.Fatalf("networkInboxValues() = %v, want session_id filter", got)
		}
		if got := networkThreadsValues(NetworkThreadsQuery{Limit: 2, After: "thread_0"}); got.Get("limit") != "2" ||
			got.Get("after") != "thread_0" {
			t.Fatalf("networkThreadsValues() = %v, want list cursor filters", got)
		}
		if got := networkDirectsValues(NetworkDirectsQuery{
			PeerID: "reviewer.sess-b",
			Limit:  2,
			After:  "direct_0",
		}); got.Get("peer_id") != "reviewer.sess-b" || got.Get("limit") != "2" || got.Get("after") != "direct_0" {
			t.Fatalf("networkDirectsValues() = %v, want peer/list filters", got)
		}
		if got := networkConversationMessagesValues(NetworkConversationMessagesQuery{
			Limit:  2,
			Before: "msg-3",
			After:  "msg-1",
			Kind:   "say",
			WorkID: "work_1",
		}); got.Get("limit") != "2" || got.Get("before") != "msg-3" ||
			got.Get("after") != "msg-1" || got.Get("kind") != "say" || got.Get("work_id") != "work_1" {
			t.Fatalf("networkConversationMessagesValues() = %v, want message filters", got)
		}
		if got, err := networkThreadPath(" builders ", " thread_1 "); err != nil ||
			got != "/api/network/channels/builders/threads/thread_1" {
			t.Fatalf("networkThreadPath() = %q, %v", got, err)
		}
		if got, err := networkDirectMessagesPath("builders", "direct_1"); err != nil ||
			got != "/api/network/channels/builders/directs/direct_1/messages" {
			t.Fatalf("networkDirectMessagesPath() = %q, %v", got, err)
		}

		tests := []struct {
			name    string
			cliType any
			want    any
		}{
			{name: "NetworkStatusRecord", cliType: NetworkStatusRecord{}, want: contract.NetworkStatusPayload{}},
			{
				name:    "NetworkKindMetricRecord",
				cliType: NetworkKindMetricRecord{},
				want:    contract.NetworkKindMetricPayload{},
			},
			{name: "NetworkSendRequest", cliType: NetworkSendRequest{}, want: contract.NetworkSendRequest{}},
			{name: "NetworkSendRecord", cliType: NetworkSendRecord{}, want: contract.NetworkSendPayload{}},
			{name: "NetworkPeerRecord", cliType: NetworkPeerRecord{}, want: contract.NetworkPeerPayload{}},
			{name: "NetworkChannelRecord", cliType: NetworkChannelRecord{}, want: contract.NetworkChannelPayload{}},
			{
				name:    "NetworkThreadRecord",
				cliType: NetworkThreadRecord{},
				want:    contract.NetworkThreadSummaryPayload{},
			},
			{
				name:    "NetworkDirectRoomRecord",
				cliType: NetworkDirectRoomRecord{},
				want:    contract.NetworkDirectRoomPayload{},
			},
			{
				name:    "NetworkConversationMessageRecord",
				cliType: NetworkConversationMessageRecord{},
				want:    contract.NetworkConversationMessagePayload{},
			},
			{name: "NetworkWorkRecord", cliType: NetworkWorkRecord{}, want: contract.NetworkWorkPayload{}},
			{
				name:    "NetworkDirectResolveRequest",
				cliType: NetworkDirectResolveRequest{},
				want:    contract.NetworkDirectResolveRequest{},
			},
			{name: "NetworkEnvelopeRecord", cliType: NetworkEnvelopeRecord{}, want: contract.NetworkEnvelopePayload{}},
		}
		for _, tt := range tests {
			if gotType, wantType := reflect.TypeOf(tt.cliType), reflect.TypeOf(tt.want); gotType != wantType {
				t.Fatalf("%s type = %v, want %v", tt.name, gotType, wantType)
			}
		}
	})
}
