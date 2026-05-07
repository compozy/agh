package bridgesdk

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestWebhookRefacs(t *testing.T) {
	t.Parallel()

	t.Run("Should report request body close errors", func(t *testing.T) {
		t.Parallel()

		request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/webhook", nil)
		request.Body = closeErrorReadCloser{Reader: strings.NewReader("{}")}
		_, err := readBodyWithLimit(httptest.NewRecorder(), request, defaultWebhookBodyLimit)
		if err == nil {
			t.Fatal("readBodyWithLimit() error = nil, want close error")
		}
		if !strings.Contains(err.Error(), "close failed") {
			t.Fatalf("readBodyWithLimit() error = %v, want close failure", err)
		}
	})

	t.Run("Should round positive subsecond retry after to one second", func(t *testing.T) {
		t.Parallel()

		handler, err := NewWebhookHandler(WebhookGuardConfig{
			AllowedMethods:      []string{http.MethodPost},
			AllowedContentTypes: []string{"application/json"},
		}, func(http.ResponseWriter, *http.Request, WebhookRequest) error {
			return &HTTPError{
				StatusCode: http.StatusTooManyRequests,
				Message:    "slow down",
				RetryAfter: 100 * time.Millisecond,
			}
		})
		if err != nil {
			t.Fatalf("NewWebhookHandler() error = %v", err)
		}

		recorder := httptest.NewRecorder()
		request := httptest.NewRequestWithContext(
			context.Background(),
			http.MethodPost,
			"/webhook",
			strings.NewReader("{}"),
		)
		request.Header.Set("Content-Type", "application/json")
		handler.ServeHTTP(recorder, request)

		if got, want := recorder.Header().Get("Retry-After"), "1"; got != want {
			t.Fatalf("Retry-After = %q, want %q", got, want)
		}
	})
}

type closeErrorReadCloser struct {
	io.Reader
}

func (closeErrorReadCloser) Close() error {
	return errors.New("close failed")
}
