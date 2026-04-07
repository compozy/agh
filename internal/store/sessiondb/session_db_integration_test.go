//go:build integration

package sessiondb

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/testutil"
)

func TestSessionDBLifecyclePersistsAcrossReopen(t *testing.T) {
	sessionDir := t.TempDir()
	path := filepath.Join(sessionDir, SessionDatabaseName)
	ctx := testutil.Context(t)

	sessionDB, err := OpenSessionDB(ctx, "sess-integration", path)
	if err != nil {
		t.Fatalf("OpenSessionDB() error = %v", err)
	}
	sessionDB.now = func() time.Time {
		return time.Date(2026, 4, 3, 19, 0, 0, 0, time.UTC)
	}

	if err := sessionDB.Record(ctx, SessionEvent{
		TurnID:    "turn-1",
		Type:      "agent_message",
		AgentName: "coder",
		Content:   `{"text":"hello"}`,
	}); err != nil {
		t.Fatalf("Record() error = %v", err)
	}
	if err := sessionDB.RecordTokenUsage(ctx, TokenUsage{
		TurnID:       "turn-1",
		OutputTokens: int64Pointer(42),
	}); err != nil {
		t.Fatalf("RecordTokenUsage() error = %v", err)
	}
	if err := sessionDB.Close(ctx); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	reopened, err := OpenSessionDB(ctx, "sess-integration", path)
	if err != nil {
		t.Fatalf("OpenSessionDB(reopen) error = %v", err)
	}
	defer func() {
		if closeErr := reopened.Close(ctx); closeErr != nil {
			t.Fatalf("Close() error = %v", closeErr)
		}
	}()

	events, err := reopened.Query(ctx, EventQuery{})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if got, want := len(events), 1; got != want {
		t.Fatalf("len(events) = %d, want %d", got, want)
	}
	if events[0].Sequence != 1 || events[0].TurnID != "turn-1" {
		t.Fatalf("events[0] = %#v, want sequence=1 turn-1", events[0])
	}
}

func TestSessionDBSupportsConcurrentReadersWithSingleWriter(t *testing.T) {
	sessionDB := openTestSessionDB(t, "sess-concurrency")
	ctx := testutil.Context(t)

	const (
		readerCount = 6
		eventCount  = 150
	)

	errCh := make(chan error, readerCount+1)
	var writerWG sync.WaitGroup
	writerWG.Add(1)
	go func() {
		defer writerWG.Done()
		for i := 0; i < eventCount; i++ {
			if err := sessionDB.Record(ctx, SessionEvent{
				TurnID:    fmt.Sprintf("turn-%03d", i),
				Type:      "agent_message",
				AgentName: "coder",
				Content:   fmt.Sprintf(`{"index":%d}`, i),
			}); err != nil {
				errCh <- fmt.Errorf("writer: %w", err)
				return
			}
		}
	}()

	var readersWG sync.WaitGroup
	for i := 0; i < readerCount; i++ {
		readersWG.Add(1)
		go func() {
			defer readersWG.Done()
			for j := 0; j < eventCount; j++ {
				events, err := sessionDB.Query(ctx, EventQuery{Limit: 10})
				if err != nil {
					errCh <- fmt.Errorf("reader: %w", err)
					return
				}
				if len(events) > 10 {
					errCh <- fmt.Errorf("reader: len(events) = %d, want <= 10", len(events))
					return
				}
			}
		}()
	}

	writerWG.Wait()
	readersWG.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Fatal(err)
		}
	}

	events, err := sessionDB.Query(ctx, EventQuery{})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if got, want := len(events), eventCount; got != want {
		t.Fatalf("len(events) = %d, want %d", got, want)
	}
}

func int64Pointer(value int64) *int64 {
	return &value
}
