package bridgesdk

import (
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"
)

func BenchmarkInboundBatchKey(b *testing.B) {
	envelope := testInboundEnvelope("idem-1", "msg-1", "hello world")

	b.ReportAllocs()

	for b.Loop() {
		if InboundBatchKey(envelope) == "" {
			b.Fatal("InboundBatchKey() = empty string")
		}
	}
}

func BenchmarkInstanceCacheSnapshot(b *testing.B) {
	cache := NewInstanceCache(testManagedRuntime(
		"brg-1", "brg-2", "brg-3", "brg-4",
		"brg-5", "brg-6", "brg-7", "brg-8",
	))

	b.ReportAllocs()

	for b.Loop() {
		runtime := cache.Snapshot()
		if runtime == nil {
			b.Fatal("Snapshot() = nil")
		}
	}
}

func BenchmarkFixedWindowRateLimiterAllow(b *testing.B) {
	limiter := NewFixedWindowRateLimiter(1<<30, time.Hour)
	if limiter == nil {
		b.Fatal("NewFixedWindowRateLimiter() = nil")
	}

	b.ReportAllocs()

	for b.Loop() {
		if !limiter.Allow("same-client") {
			b.Fatal("Allow() = false")
		}
	}
}

func BenchmarkPeerCallRoundTrip(b *testing.B) {
	type echoParams struct {
		Message string `json:"message"`
	}
	type echoResult struct {
		Message string `json:"message"`
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	leftConn, rightConn := net.Pipe()
	defer func() {
		_ = leftConn.Close()
	}()
	defer func() {
		_ = rightConn.Close()
	}()

	left := NewPeer(leftConn, leftConn)
	right := NewPeer(rightConn, rightConn)

	if err := right.Handle("echo", func(_ context.Context, raw json.RawMessage) (any, error) {
		var params echoParams
		if err := json.Unmarshal(raw, &params); err != nil {
			return nil, err
		}
		return echoResult(params), nil
	}); err != nil {
		b.Fatalf("right.Handle() error = %v", err)
	}

	done := make(chan struct{}, 2)
	go func() {
		_ = left.Serve(ctx)
		done <- struct{}{}
	}()
	go func() {
		_ = right.Serve(ctx)
		done <- struct{}{}
	}()

	params := echoParams{Message: "hello"}
	var result echoResult

	b.ReportAllocs()

	for b.Loop() {
		result = echoResult{}
		if err := left.Call(ctx, "echo", params, &result); err != nil {
			b.Fatalf("left.Call() error = %v", err)
		}
		if result.Message != params.Message {
			b.Fatalf("result.Message = %q, want %q", result.Message, params.Message)
		}
	}
	b.StopTimer()

	cancel()
	_ = leftConn.Close()
	_ = rightConn.Close()
	<-done
	<-done
}
