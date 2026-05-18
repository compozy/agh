package network

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestTransportShutdownCanceledContextContract(t *testing.T) {
	t.Parallel()

	t.Run("Should stop embedded server when shutdown context is already canceled", func(t *testing.T) {
		t.Parallel()

		transport, err := NewTransport(
			context.Background(),
			testNetworkConfig(),
			WithTransportReadyTimeout(2*time.Second),
		)
		if err != nil {
			t.Fatalf("NewTransport() error = %v", err)
		}
		t.Cleanup(func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if err := transport.Shutdown(shutdownCtx); err != nil {
				t.Errorf("cleanup Shutdown() error = %v", err)
			}
		})

		shutdownCtx, cancel := context.WithCancel(context.Background())
		cancel()
		err = transport.Shutdown(shutdownCtx)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Shutdown(canceled context) error = %v, want context.Canceled", err)
		}
		if transport.server == nil {
			t.Fatal("embedded server = nil, want server instance")
		}
		if transport.server.Running() {
			t.Fatal("embedded server running after Shutdown(canceled context), want stopped")
		}
		if err := transport.Shutdown(context.Background()); err != nil {
			t.Fatalf("Shutdown(second call) error = %v", err)
		}
	})
}
