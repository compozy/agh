package udsapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/pedronauck/agh/internal/network"
)

func TestNetworkHandlersValidateRequestsAndMapErrors(t *testing.T) {
	t.Parallel()

	homePaths := newTestHomePaths(t)
	handlers := newTestHandlers(t, stubSessionManager{}, stubObserver{}, homePaths)
	handlers.Config.Network.Enabled = true
	handlers.Network = stubNetworkService{
		SendFn: func(context.Context, network.SendRequest) (string, error) {
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
}

func TestNetworkHandlersPreserveWorkflowMetadata(t *testing.T) {
	t.Parallel()

	homePaths := newTestHomePaths(t)
	handlers := newTestHandlers(t, stubSessionManager{}, stubObserver{}, homePaths)
	handlers.Config.Network.Enabled = true

	var seenRequest network.SendRequest
	handlers.Network = stubNetworkService{
		SendFn: func(_ context.Context, req network.SendRequest) (string, error) {
			seenRequest = req
			return "msg-1", nil
		},
		InboxFn: func(_ context.Context, sessionID string) ([]network.Envelope, error) {
			return []network.Envelope{{
				Protocol: network.ProtocolV0,
				ID:       "msg-inbox",
				Kind:     network.KindDirect,
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

	sendResp := performRequest(t, engine, http.MethodPost, "/api/network/send", []byte(`{"session_id":"sess-a","channel":"builders","kind":"say","body":{"text":"hello"},"ext":{"agh.workflow_id":"wf-1","agh.handoff_version":3}}`))
	if sendResp.Code != http.StatusOK {
		t.Fatalf("send status = %d, want %d; body=%s", sendResp.Code, http.StatusOK, sendResp.Body.String())
	}
	if string(seenRequest.Ext["agh.workflow_id"]) != `"wf-1"` || string(seenRequest.Ext["agh.handoff_version"]) != `3` {
		t.Fatalf("seenRequest.Ext = %#v, want preserved workflow metadata", seenRequest.Ext)
	}

	inboxResp := performRequest(t, engine, http.MethodGet, "/api/network/inbox?session_id=sess-a", nil)
	if inboxResp.Code != http.StatusOK {
		t.Fatalf("inbox status = %d, want %d", inboxResp.Code, http.StatusOK)
	}
	if !strings.Contains(inboxResp.Body.String(), `"agh.workflow_id":"wf-1"`) || !strings.Contains(inboxResp.Body.String(), `"agh.handoff_version":3`) {
		t.Fatalf("inbox body = %s, want workflow metadata", inboxResp.Body.String())
	}
}
