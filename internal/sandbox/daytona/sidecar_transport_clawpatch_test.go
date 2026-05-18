package daytona

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pedronauck/agh/internal/testutil"
)

func TestSidecarSessionCleanupClawpatch(t *testing.T) {
	t.Parallel()

	t.Run("Should close endpoint exactly once after Stop", func(t *testing.T) {
		t.Parallel()

		session, closeCount := newClawpatchSidecarSession(t)
		if err := session.Stop(testutil.Context(t)); err != nil {
			t.Fatalf("Stop() error = %v", err)
		}
		if err := session.Stop(testutil.Context(t)); err != nil {
			t.Fatalf("Stop(second) error = %v", err)
		}
		if got := closeCount.Load(); got != 1 {
			t.Fatalf("endpoint close count = %d, want 1", got)
		}
	})

	t.Run("Should close endpoint exactly once after Wait observes server exit", func(t *testing.T) {
		t.Parallel()

		session, closeCount := newClawpatchSidecarSession(t)
		if err := session.Wait(); err != nil {
			t.Fatalf("Wait() error = %v", err)
		}
		if err := session.Wait(); err != nil {
			t.Fatalf("Wait(second) error = %v", err)
		}
		if got := closeCount.Load(); got != 1 {
			t.Fatalf("endpoint close count = %d, want 1", got)
		}
	})
}

func newClawpatchSidecarSession(t *testing.T) (*sidecarSession, *atomic.Int32) {
	t.Helper()

	var closeCount atomic.Int32
	server := newClawpatchSidecarServer(t)
	endpoint := newClawpatchSidecarEndpoint(t, server, &closeCount)
	conn, response, err := websocket.DefaultDialer.Dial(
		endpoint.wsURL(sidecarSessionStreamBasePath, "session-1", "stream"),
		nil,
	)
	t.Cleanup(func() {
		if response != nil && response.Body != nil {
			if err := response.Body.Close(); err != nil {
				t.Errorf("websocket response body Close() error = %v", err)
			}
		}
	})
	if err != nil {
		t.Fatalf("websocket.Dial() error = %v", err)
	}
	return newSidecarSession(conn, endpoint, "session-1", server.Client(), time.Second), &closeCount
}

func TestSidecarTransportDialCleanupClawpatch(t *testing.T) {
	t.Parallel()

	t.Run("Should close endpoint when launch fails", func(t *testing.T) {
		t.Parallel()

		var closeCount atomic.Int32
		server := newClawpatchDialServer(t, func(writer http.ResponseWriter, request *http.Request) {
			if request.Method == http.MethodPost && request.URL.Path == "/v1/launch" {
				http.Error(writer, "launch failed", http.StatusInternalServerError)
				return
			}
			http.NotFound(writer, request)
		})
		endpoint := newClawpatchSidecarEndpoint(t, server, &closeCount)
		transport := &sidecarTransport{httpClient: server.Client(), closeTimeout: time.Second}

		_, err := transport.dialEndpoint(testutil.Context(t), endpoint, "echo ok")
		if err == nil {
			t.Fatal("dialEndpoint(launch failure) error = nil, want non-nil")
		}
		if got := closeCount.Load(); got != 1 {
			t.Fatalf("endpoint close count = %d, want 1", got)
		}
	})

	t.Run("Should close endpoint when websocket connect fails", func(t *testing.T) {
		t.Parallel()

		var closeCount atomic.Int32
		server := newClawpatchDialServer(t, func(writer http.ResponseWriter, request *http.Request) {
			switch {
			case request.Method == http.MethodPost && request.URL.Path == "/v1/launch":
				writer.Header().Set("Content-Type", "application/json")
				writer.WriteHeader(http.StatusCreated)
				writeClawpatchSidecarResponse(t, writer, "{\"id\":\"session-1\"}")
			case request.Method == http.MethodGet && request.URL.Path == "/v1/sessions/session-1/stream":
				http.Error(writer, "websocket failed", http.StatusInternalServerError)
			default:
				http.NotFound(writer, request)
			}
		})
		endpoint := newClawpatchSidecarEndpoint(t, server, &closeCount)
		transport := &sidecarTransport{httpClient: server.Client(), closeTimeout: time.Second}

		_, err := transport.dialEndpoint(testutil.Context(t), endpoint, "echo ok")
		if err == nil {
			t.Fatal("dialEndpoint(connect failure) error = nil, want non-nil")
		}
		if got := closeCount.Load(); got != 1 {
			t.Fatalf("endpoint close count = %d, want 1", got)
		}
	})
}

func newClawpatchSidecarEndpoint(
	t *testing.T,
	server *httptest.Server,
	closeCount *atomic.Int32,
) sidecarEndpoint {
	t.Helper()

	baseURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("url.Parse(server.URL) error = %v", err)
	}
	return sidecarEndpoint{
		base:       baseURL,
		httpClient: server.Client(),
		wsDialer:   websocket.DefaultDialer,
		closeFn: func() error {
			closeCount.Add(1)
			return nil
		},
	}
}

func newClawpatchSidecarServer(t *testing.T) *httptest.Server {
	t.Helper()

	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch {
		case request.Method == http.MethodDelete && request.URL.Path == "/v1/sessions/session-1":
			writer.WriteHeader(http.StatusNoContent)
		case request.Method == http.MethodGet && request.URL.Path == "/v1/sessions/session-1/stream":
			conn, err := upgrader.Upgrade(writer, request, nil)
			if err != nil {
				t.Errorf("websocket Upgrade() error = %v", err)
				return
			}
			payload, err := json.Marshal(sidecarExitPayload{ExitCode: 0})
			if err != nil {
				t.Errorf("json.Marshal(exit) error = %v", err)
				return
			}
			frame := append([]byte{sidecarFrameServerExit}, payload...)
			if err := conn.WriteMessage(websocket.BinaryMessage, frame); err != nil {
				t.Errorf("conn.WriteMessage(exit) error = %v", err)
			}
			if err := conn.Close(); err != nil {
				t.Errorf("conn.Close() error = %v", err)
			}
		default:
			http.Error(writer, fmt.Sprintf("unexpected %s %s", request.Method, request.URL.Path), http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func newClawpatchDialServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

func writeClawpatchSidecarResponse(t *testing.T, writer http.ResponseWriter, body string) {
	t.Helper()

	if _, err := writer.Write([]byte(body)); err != nil {
		t.Errorf("writer.Write() error = %v", err)
	}
}
