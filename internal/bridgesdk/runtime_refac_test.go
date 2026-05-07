package bridgesdk

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"
	"time"

	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	"github.com/pedronauck/agh/internal/subprocess"
)

func TestRuntimeRefacs(t *testing.T) {
	t.Parallel()

	t.Run("Should not hold runtime lock while initialize callback runs", func(t *testing.T) {
		t.Parallel()

		var runtime *Runtime
		var err error
		runtime, err = NewRuntime(RuntimeConfig{
			ExtensionInfo: subprocess.InitializeExtensionInfo{
				Name:    "telegram-adapter",
				Version: "1.0.0",
			},
			Initialize: func(context.Context, *Session) error {
				if got := runtime.Session(); got != nil {
					return errors.New("runtime session visible before initialize commit")
				}
				return nil
			},
			Deliver: func(_ context.Context, session *Session, request bridgepkg.DeliveryRequest) (bridgepkg.DeliveryAck, error) {
				return session.AckDelivery(request, "remote-1", "")
			},
		})
		if err != nil {
			t.Fatalf("NewRuntime() error = %v", err)
		}
		runtime.peer = NewPeer(io.Reader(nil), io.Discard)

		raw, err := json.Marshal(testInitializeRequest())
		if err != nil {
			t.Fatalf("json.Marshal(initialize request) error = %v", err)
		}

		done := make(chan error, 1)
		go func() {
			_, handleErr := runtime.handleInitialize(t.Context(), raw)
			done <- handleErr
		}()

		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("handleInitialize() error = %v", err)
			}
		case <-time.After(250 * time.Millisecond):
			t.Fatal("handleInitialize() timed out, likely held runtime lock across callback")
		}
		if runtime.Session() == nil {
			t.Fatal("runtime.Session() after initialize = nil, want committed session")
		}
	})

	t.Run("Should retry shutdown handler after a failed first attempt", func(t *testing.T) {
		t.Parallel()

		shutdownCalls := 0
		runtime, err := NewRuntime(RuntimeConfig{
			ExtensionInfo: subprocess.InitializeExtensionInfo{
				Name:    "telegram-adapter",
				Version: "1.0.0",
			},
			Deliver: func(_ context.Context, session *Session, request bridgepkg.DeliveryRequest) (bridgepkg.DeliveryAck, error) {
				return session.AckDelivery(request, "remote-1", "")
			},
			Shutdown: func(context.Context, *Session, subprocess.ShutdownRequest) error {
				shutdownCalls++
				if shutdownCalls == 1 {
					return errors.New("provider shutdown failed")
				}
				return nil
			},
		})
		if err != nil {
			t.Fatalf("NewRuntime() error = %v", err)
		}
		runtime.session = &Session{}

		if _, err := runtime.handleShutdown(t.Context(), json.RawMessage(`{"reason":"test"}`)); err == nil {
			t.Fatal("first handleShutdown() error = nil, want provider failure")
		}

		response, err := runtime.handleShutdown(t.Context(), json.RawMessage(`{"reason":"retry"}`))
		if err != nil {
			t.Fatalf("second handleShutdown() error = %v", err)
		}
		if got, want := shutdownCalls, 2; got != want {
			t.Fatalf("shutdownCalls = %d, want %d", got, want)
		}
		shutdown, ok := response.(subprocess.ShutdownResponse)
		if !ok {
			t.Fatalf("response = %T, want subprocess.ShutdownResponse", response)
		}
		if !shutdown.Acknowledged {
			t.Fatal("shutdown.Acknowledged = false, want true")
		}
	})

	t.Run("Should decode whitespace null params as empty object", func(t *testing.T) {
		t.Parallel()

		var target map[string]any
		if err := decodeParams(json.RawMessage(" \n null \t "), &target); err != nil {
			t.Fatalf("decodeParams(whitespace null) error = %v", err)
		}
		if target == nil {
			t.Fatal("target = nil, want decoded empty object")
		}
	})
}
