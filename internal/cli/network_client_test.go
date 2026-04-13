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

	client := &unixSocketClient{
		socketPath: "/tmp/agh.sock",
		httpClient: &http.Client{
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				switch {
				case req.Method == http.MethodGet && req.URL.Path == "/api/network/status":
					return newHTTPResponse(http.StatusOK, `{"network":{"enabled":true,"status":"running","listener_host":"127.0.0.1","listener_port":4222,"local_peers":1,"remote_peers":2,"channels":1,"queued_messages":3,"queued_sessions":1,"delivery_workers":1,"messages_sent":4,"messages_received":5,"messages_rejected":1,"messages_delivered":3,"workflow_tagged_events":2,"handoff_tagged_events":1,"kind_metrics":[{"kind":"say","sent":4,"received":5,"rejected":1,"delivered":3}]}}`), nil
				case req.Method == http.MethodGet && req.URL.Path == "/api/network/peers":
					if got := req.URL.Query().Get("channel"); got != "builders" {
						t.Fatalf("network peers channel query = %q, want builders", got)
					}
					return newHTTPResponse(http.StatusOK, `{"peers":[{"peer_id":"reviewer.sess-a","session_id":"sess-a","channel":"builders","local":true,"peer_card":{"peer_id":"reviewer.sess-a","display_name":"Reviewer","profiles_supported":["v0"],"capabilities":["send"],"artifacts_supported":["text"],"trust_modes_supported":["untrusted"]}}]}`), nil
				case req.Method == http.MethodGet && req.URL.Path == "/api/network/channels":
					return newHTTPResponse(http.StatusOK, `{"channels":[{"channel":"builders","peer_count":2}]}`), nil
				case req.Method == http.MethodPost && req.URL.Path == "/api/network/send":
					body, err := io.ReadAll(req.Body)
					if err != nil {
						t.Fatalf("io.ReadAll(network send body) error = %v", err)
					}
					if !strings.Contains(string(body), `"channel":"builders"`) ||
						!strings.Contains(string(body), `"agh.workflow_id":"wf-1"`) ||
						!strings.Contains(string(body), `"agh.handoff_version":3`) {
						t.Fatalf("network send body = %s, want ext metadata", body)
					}
					return newHTTPResponse(http.StatusOK, `{"message":{"id":"msg-1","session_id":"sess-a","channel":"builders","kind":"say","ext":{"agh.workflow_id":"wf-1","agh.handoff_version":3}}}`), nil
				case req.Method == http.MethodGet && req.URL.Path == "/api/network/inbox":
					if got := req.URL.Query().Get("session_id"); got != "sess-a" {
						t.Fatalf("network inbox session_id query = %q, want sess-a", got)
					}
					return newHTTPResponse(http.StatusOK, `{"messages":[{"protocol":"agh-network/v0","id":"msg-inbox","kind":"direct","channel":"builders","from":"reviewer.sess-a","ts":1775823000,"body":{"text":"review this","intent":"review"},"ext":{"agh.workflow_id":"wf-1","agh.handoff_version":3}}]}`), nil
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

	sent, err := client.NetworkSend(ctx, NetworkSendRequest{
		SessionID: "sess-a",
		Channel:   "builders",
		Kind:      "say",
		Body:      json.RawMessage(`{"text":"hello"}`),
		Ext: map[string]json.RawMessage{
			"agh.workflow_id":     json.RawMessage(`"wf-1"`),
			"agh.handoff_version": json.RawMessage(`3`),
		},
	})
	if err != nil || sent.ID != "msg-1" {
		t.Fatalf("NetworkSend() = %#v, %v", sent, err)
	}

	inbox, err := client.NetworkInbox(ctx, "sess-a")
	if err != nil || len(inbox) != 1 || string(inbox[0].Ext["agh.workflow_id"]) != `"wf-1"` {
		t.Fatalf("NetworkInbox() = %#v, %v", inbox, err)
	}
}

func TestNetworkClientHelpersAndAliases(t *testing.T) {
	t.Parallel()

	if got := networkPeersValues(NetworkPeersQuery{Channel: "builders"}); got.Get("channel") != "builders" {
		t.Fatalf("networkPeersValues() = %v, want channel filter", got)
	}
	if got := networkInboxValues("sess-a"); got.Get("session_id") != "sess-a" {
		t.Fatalf("networkInboxValues() = %v, want session_id filter", got)
	}

	tests := []struct {
		name    string
		cliType any
		want    any
	}{
		{name: "NetworkStatusRecord", cliType: NetworkStatusRecord{}, want: contract.NetworkStatusPayload{}},
		{name: "NetworkKindMetricRecord", cliType: NetworkKindMetricRecord{}, want: contract.NetworkKindMetricPayload{}},
		{name: "NetworkSendRequest", cliType: NetworkSendRequest{}, want: contract.NetworkSendRequest{}},
		{name: "NetworkSendRecord", cliType: NetworkSendRecord{}, want: contract.NetworkSendPayload{}},
		{name: "NetworkPeerRecord", cliType: NetworkPeerRecord{}, want: contract.NetworkPeerPayload{}},
		{name: "NetworkChannelRecord", cliType: NetworkChannelRecord{}, want: contract.NetworkChannelPayload{}},
		{name: "NetworkEnvelopeRecord", cliType: NetworkEnvelopeRecord{}, want: contract.NetworkEnvelopePayload{}},
	}
	for _, tt := range tests {
		if gotType, wantType := reflect.TypeOf(tt.cliType), reflect.TypeOf(tt.want); gotType != wantType {
			t.Fatalf("%s type = %v, want %v", tt.name, gotType, wantType)
		}
	}
}
