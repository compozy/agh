package memory

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/session"
)

func TestNewRecallAugmenterReturnsOriginalMessageWhenSessionOrQueryIsEmpty(t *testing.T) {
	t.Parallel()

	augmenter := NewRecallAugmenter(NewStore(filepath.Join(t.TempDir(), "global")))

	got, err := augmenter(context.Background(), nil, "hello")
	if err != nil {
		t.Fatalf("Augment(nil session) error = %v", err)
	}
	if got != "hello" {
		t.Fatalf("Augment(nil session) = %q, want original message", got)
	}

	got, err = augmenter(context.Background(), &session.Session{Type: session.SessionTypeUser}, "   ")
	if err != nil {
		t.Fatalf("Augment(blank query) error = %v", err)
	}
	if got != "   " {
		t.Fatalf("Augment(blank query) = %q, want original message", got)
	}
}

func TestNewRecallAugmenterPrependsRecallAndPreservesUserMessage(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	workspaceRoot := filepath.Join(baseDir, "workspace")
	store := NewStore(filepath.Join(baseDir, "global")).ForWorkspace(workspaceRoot)
	if err := store.EnsureDirs(); err != nil {
		t.Fatalf("Store.EnsureDirs() error = %v", err)
	}
	if err := store.Write(ScopeWorkspace, "auth.md", mustMemoryContent(t, testMemoryMeta{
		Name:        "Auth",
		Description: "Auth migration notes",
		Type:        MemoryTypeProject,
	}, "Remember auth sessions and migration details.\n")); err != nil {
		t.Fatalf("Store.Write() error = %v", err)
	}

	augmenter := NewRecallAugmenter(store)
	got, err := augmenter(
		context.Background(),
		&session.Session{Type: session.SessionTypeUser, Workspace: workspaceRoot},
		"auth migration",
	)
	if err != nil {
		t.Fatalf("Augment() error = %v", err)
	}
	if !strings.Contains(got, "Relevant durable memory for this turn:") {
		t.Fatalf("Augment() = %q, want recall header", got)
	}
	if !strings.Contains(got, "Auth") {
		t.Fatalf("Augment() = %q, want memory metadata", got)
	}
	if !strings.Contains(got, "User message:\nauth migration") {
		t.Fatalf("Augment() = %q, want preserved user message", got)
	}
}

func TestBuildRecallBlockSkipsZeroScoreEntriesAndCapsResults(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	block := buildRecallBlock([]SearchResult{
		{
			Name:    "Ignore",
			Scope:   ScopeWorkspace,
			Score:   0,
			Snippet: "should not appear",
		},
		{
			Name:    "One",
			Scope:   ScopeWorkspace,
			Score:   0.9,
			Snippet: "first result",
			ModTime: now,
		},
		{
			Name:    "Two",
			Scope:   ScopeGlobal,
			Score:   0.8,
			Snippet: "second result",
			ModTime: now.Add(-48 * time.Hour),
		},
		{
			Name:    "Three",
			Scope:   ScopeGlobal,
			Score:   0.7,
			Snippet: "third result",
			ModTime: now,
		},
		{
			Name:    "Four",
			Scope:   ScopeGlobal,
			Score:   0.6,
			Snippet: "fourth result",
			ModTime: now,
		},
	}, now)

	if strings.Contains(block, "Ignore") {
		t.Fatalf("buildRecallBlock() = %q, want zero-score result omitted", block)
	}
	if !strings.Contains(block, "One") || !strings.Contains(block, "Two") || !strings.Contains(block, "Three") {
		t.Fatalf("buildRecallBlock() = %q, want first three positive-scored results", block)
	}
	if strings.Contains(block, "Four") {
		t.Fatalf("buildRecallBlock() = %q, want max result count enforced", block)
	}
	if !strings.Contains(block, "Freshness:") {
		t.Fatalf("buildRecallBlock() = %q, want freshness warning for stale memory", block)
	}
}
