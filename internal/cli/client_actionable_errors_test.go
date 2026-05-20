package cli

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestUnixSocketClientActionableDaemonErrors(t *testing.T) {
	t.Parallel()

	t.Run("Should include daemon start guidance when socket is missing", func(t *testing.T) {
		t.Parallel()

		root, err := os.MkdirTemp("", "agh-missing-socket-")
		if err != nil {
			t.Fatalf("os.MkdirTemp() error = %v", err)
		}
		t.Cleanup(func() {
			if removeErr := os.RemoveAll(root); removeErr != nil {
				t.Errorf("os.RemoveAll(%q) error = %v", root, removeErr)
			}
		})

		socketPath := filepath.Join(root, "agh.sock")
		client, err := NewClient(socketPath)
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}

		_, err = client.DaemonStatus(context.Background())
		if err == nil {
			t.Fatal("DaemonStatus() error = nil, want missing daemon socket failure")
		}
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("DaemonStatus() error = %v, want os.ErrNotExist in chain", err)
		}
		assertDaemonUnavailableGuidance(t, err, socketPath)
	})

	t.Run("Should include daemon start guidance when socket refuses connection", func(t *testing.T) {
		t.Parallel()

		socketPath := "/tmp/agh-stale.sock"
		client := &unixSocketClient{
			socketPath: socketPath,
			httpClient: &http.Client{
				Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
					return nil, &url.Error{
						Op:  "Get",
						URL: baseURL + "/api/daemon/status",
						Err: syscall.ECONNREFUSED,
					}
				}),
			},
		}

		_, err := client.DaemonStatus(context.Background())
		if err == nil {
			t.Fatal("DaemonStatus() error = nil, want stale daemon socket failure")
		}
		if !errors.Is(err, syscall.ECONNREFUSED) {
			t.Fatalf("DaemonStatus() error = %v, want syscall.ECONNREFUSED in chain", err)
		}
		assertDaemonUnavailableGuidance(t, err, socketPath)
	})
}

func assertDaemonUnavailableGuidance(t *testing.T, err error, socketPath string) {
	t.Helper()

	for _, want := range []string{
		"daemon unavailable",
		socketPath,
		"agh daemon start",
		"agh daemon status",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("DaemonStatus() error = %q, want %q", err.Error(), want)
		}
	}
}
