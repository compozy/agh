package network

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	aghconfig "github.com/compozy/agh/internal/config"
	"github.com/nats-io/nats.go"
)

func TestNewTransportRejectsMissingRuntimeInputs(t *testing.T) {
	t.Parallel()

	cfg := testNetworkConfig()

	if _, err := NewTransport(nilTransportContext(), cfg); err == nil {
		t.Fatal("NewTransport(nil) error = nil, want non-nil")
	}

	invalid := cfg
	invalid.MaxPayload = 0
	if _, err := NewTransport(context.Background(), invalid); err == nil {
		t.Fatal("NewTransport(invalid config) error = nil, want non-nil")
	}
}

func TestTransportSubjectHelpers(t *testing.T) {
	t.Parallel()

	broadcast, err := BroadcastSubject(testWorkspaceID, "builders")
	if err != nil {
		t.Fatalf("BroadcastSubject() error = %v", err)
	}
	if got, want := broadcast, "agh.network.v0.wks_test.builders.broadcast"; got != want {
		t.Fatalf("BroadcastSubject() = %q, want %q", got, want)
	}

	direct, err := DirectSubject(testWorkspaceID, "builders", "reviewer.sess-xyz")
	if err != nil {
		t.Fatalf("DirectSubject() error = %v", err)
	}
	if !strings.HasPrefix(direct, "agh.network.v0.wks_test.builders.peer.") {
		t.Fatalf("DirectSubject() = %q, want peer subject", direct)
	}

	if _, err := BroadcastSubject(testWorkspaceID, "Bad Channel"); err == nil {
		t.Fatal("BroadcastSubject(invalid channel) error = nil, want non-nil")
	}
	if _, err := BroadcastSubject("", "builders"); err == nil {
		t.Fatal("BroadcastSubject(empty workspace) error = nil, want non-nil")
	}
	if _, err := BroadcastSubject("   ", "builders"); err == nil {
		t.Fatal("BroadcastSubject(blank workspace) error = nil, want non-nil")
	}
	if _, err := DirectSubject(testWorkspaceID, "builders", "BadPeer"); err == nil {
		t.Fatal("DirectSubject(invalid peer) error = nil, want non-nil")
	}
	if _, err := DirectSubject("", "builders", "reviewer.sess-xyz"); err == nil {
		t.Fatal("DirectSubject(empty workspace) error = nil, want non-nil")
	}

	otherBroadcast, err := BroadcastSubject("wks_other", "builders")
	if err != nil {
		t.Fatalf("BroadcastSubject(other workspace) error = %v", err)
	}
	if otherBroadcast == broadcast {
		t.Fatalf("BroadcastSubject(other workspace) = %q, want workspace-qualified subject", otherBroadcast)
	}

	otherDirect, err := DirectSubject("wks_other", "builders", "reviewer.sess-xyz")
	if err != nil {
		t.Fatalf("DirectSubject(other workspace) error = %v", err)
	}
	if otherDirect == direct {
		t.Fatalf("DirectSubject(other workspace) = %q, want workspace-qualified subject", otherDirect)
	}
}

func TestTransportLifecycleAndMethodGuards(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	var reconnectCalled bool
	var disconnectCalled bool

	transport, err := NewTransport(
		ctx,
		testNetworkConfig(),
		WithTransportLogger(logger),
		WithTransportReadyTimeout(2*time.Second),
		WithTransportPublishTimeout(2*time.Second),
		WithTransportReconnectHandler(func() { reconnectCalled = true }),
		WithTransportDisconnectHandler(func(error) { disconnectCalled = true }),
	)
	if err != nil {
		t.Fatalf("NewTransport() error = %v", err)
	}

	if got := transport.Port(); got <= 0 {
		t.Fatalf("Port() = %d, want positive port", got)
	}
	if got := transport.ClientURL(); got == "" {
		t.Fatal("ClientURL() = empty, want non-empty")
	}
	if reconnectCalled {
		t.Fatal("reconnect handler called unexpectedly")
	}
	if disconnectCalled {
		t.Fatal("disconnect handler called unexpectedly")
	}
	if got := resolvedTransportPort(nil); got != 0 {
		t.Fatalf("resolvedTransportPort(nil) = %d, want 0", got)
	}

	subject, err := BroadcastSubject(testWorkspaceID, "builders")
	if err != nil {
		t.Fatalf("BroadcastSubject() error = %v", err)
	}

	if err := transport.Publish(nilTransportContext(), subject, []byte("x")); err == nil {
		t.Fatal("Publish(nil ctx) error = nil, want non-nil")
	}
	if _, err := transport.Subscribe("", nil); err == nil {
		t.Fatal("Subscribe(invalid args) error = nil, want non-nil")
	}

	received := make(chan string, 1)
	subscription, err := transport.Subscribe(subject, func(msg *nats.Msg) {
		received <- string(msg.Data)
	})
	if err != nil {
		t.Fatalf("Subscribe(valid) error = %v", err)
	}
	t.Cleanup(func() {
		_ = subscription.Unsubscribe()
	})

	if err := transport.Publish(ctx, subject, []byte("unit-message")); err != nil {
		t.Fatalf("Publish(valid) error = %v", err)
	}

	select {
	case got := <-received:
		if got != "unit-message" {
			t.Fatalf("received payload = %q, want %q", got, "unit-message")
		}
	case <-ctx.Done():
		t.Fatalf("timed out waiting for unit test message: %v", ctx.Err())
	}

	if err := transport.Drain(ctx); err != nil {
		t.Fatalf("Drain() error = %v", err)
	}
	if err := transport.Drain(ctx); err != nil {
		t.Fatalf("Drain(second call) error = %v", err)
	}
	if err := transport.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if err := transport.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown(second call) error = %v", err)
	}
}

func testNetworkConfig() aghconfig.NetworkConfig {
	return aghconfig.NetworkConfig{
		Enabled:        true,
		DefaultChannel: "default",
		Port:           -1,
		MaxPayload:     1 << 20,
		GreetInterval:  30,
		MaxReplayAge:   300,
		MaxQueueDepth:  100,
	}
}

func nilTransportContext() context.Context {
	return nil
}
