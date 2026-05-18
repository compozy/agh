package e2e

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestRuntimeHarnessHTTPUntilContract(t *testing.T) {
	t.Parallel()

	t.Run("Should reject nil prompt predicate before issuing HTTP request", func(t *testing.T) {
		t.Parallel()

		var requests atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			requests.Add(1)
			w.Header().Set("Content-Type", "text/event-stream")
			if _, err := fmt.Fprint(w, "event: done\ndata: [DONE]\n\n"); err != nil {
				t.Errorf("write SSE response error = %v", err)
			}
		}))
		defer server.Close()

		harness := &RuntimeHarness{
			WorkspaceID: "ws-1",
			HTTPBaseURL: server.URL,
			HTTPClient:  server.Client(),
		}

		records, err := harness.PromptSessionHTTPUntil(context.Background(), "sess-1", "hello", nil)
		if err == nil || !strings.Contains(err.Error(), "SSE predicate is required") {
			t.Fatalf("PromptSessionHTTPUntil(nil predicate) error = %v, want SSE predicate validation", err)
		}
		if records != nil {
			t.Fatalf("PromptSessionHTTPUntil(nil predicate) records = %#v, want nil", records)
		}
		if got := requests.Load(); got != 0 {
			t.Fatalf("PromptSessionHTTPUntil(nil predicate) issued %d requests, want 0", got)
		}
	})

	t.Run("Should reject nil stream predicate before issuing HTTP request", func(t *testing.T) {
		t.Parallel()

		var requests atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			requests.Add(1)
			w.Header().Set("Content-Type", "text/event-stream")
			if _, err := fmt.Fprint(w, "event: done\ndata: [DONE]\n\n"); err != nil {
				t.Errorf("write SSE response error = %v", err)
			}
		}))
		defer server.Close()

		harness := &RuntimeHarness{
			WorkspaceID: "ws-1",
			HTTPBaseURL: server.URL,
			HTTPClient:  server.Client(),
		}

		records, err := harness.StreamSessionHTTPUntil(context.Background(), "sess-1", nil)
		if err == nil || !strings.Contains(err.Error(), "SSE predicate is required") {
			t.Fatalf("StreamSessionHTTPUntil(nil predicate) error = %v, want SSE predicate validation", err)
		}
		if records != nil {
			t.Fatalf("StreamSessionHTTPUntil(nil predicate) records = %#v, want nil", records)
		}
		if got := requests.Load(); got != 0 {
			t.Fatalf("StreamSessionHTTPUntil(nil predicate) issued %d requests, want 0", got)
		}
	})
}
