package core

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
)

type benchmarkFlushWriter struct {
	bytes.Buffer
}

func (w *benchmarkFlushWriter) Flush() {}

func BenchmarkWriteSSE(b *testing.B) {
	b.ReportAllocs()

	writer := &benchmarkFlushWriter{}
	msg := SSEMessage{
		ID:   "evt-123",
		Name: "agent_message",
		Data: map[string]any{
			"id":    "m-001",
			"delta": "benchmark payload for SSE write path",
			"seq":   42,
		},
	}

	for b.Loop() {
		writer.Reset()
		if err := WriteSSE(writer, msg); err != nil {
			b.Fatalf("WriteSSE() error: %v", err)
		}
	}
}

func BenchmarkEmitObserveEvents(b *testing.B) {
	b.ReportAllocs()

	events := make([]store.EventSummary, 0, 64)
	baseTime := time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)
	for i := range 64 {
		events = append(events, store.EventSummary{
			ID:        fmt.Sprintf("evt-%03d", i),
			SessionID: "sess-1",
			Sequence:  int64(i + 1),
			Type:      "agent_message",
			AgentName: "codex",
			Summary:   "summary",
			Timestamp: baseTime.Add(time.Duration(i) * time.Millisecond),
		})
	}

	writer := &benchmarkFlushWriter{}
	for b.Loop() {
		writer.Reset()
		_ = EmitObserveEvents(writer, events, ObserveCursor{})
	}
}

func BenchmarkSessionPayloadsFromInfos(b *testing.B) {
	b.ReportAllocs()

	infos := make([]*session.Info, 0, 256)
	baseTime := time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)
	for i := range 256 {
		infos = append(infos, &session.Info{
			ID:          fmt.Sprintf("sess-%03d", i),
			Name:        fmt.Sprintf("Session %d", i),
			AgentName:   "codex",
			WorkspaceID: "ws-alpha",
			Workspace:   "/tmp/workspace",
			Channel:     "general",
			State:       session.StateActive,
			CreatedAt:   baseTime,
			UpdatedAt:   baseTime.Add(time.Duration(i) * time.Second),
		})
	}

	for b.Loop() {
		_ = SessionPayloadsFromInfos(infos)
	}
}
