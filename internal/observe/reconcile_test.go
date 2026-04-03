package observe

import (
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/store"
)

func TestReconciliationIndexesSessionDirNotInDB(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	sessionDir := filepath.Join(h.home.SessionsDir, "sess-new")
	metaPath := store.SessionMetaFile(sessionDir)
	now := h.now.Add(30 * time.Minute)

	if err := store.WriteSessionMeta(metaPath, store.SessionMeta{
		ID:        "sess-new",
		Name:      "New",
		AgentName: "coder",
		Workspace: h.workspace,
		State:     "active",
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("WriteSessionMeta() error = %v", err)
	}

	result, err := h.observer.Reconcile(testContext(t))
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	sort.Strings(result.Indexed)
	if got, want := result.Indexed, []string{"sess-new"}; !equalStrings(got, want) {
		t.Fatalf("Indexed = %#v, want %#v", got, want)
	}

	sessions, err := h.observer.registry.ListSessions(testContext(t), store.SessionListQuery{})
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if got, want := len(sessions), 1; got != want {
		t.Fatalf("len(sessions) = %d, want %d", got, want)
	}
	if sessions[0].State != "stopped" {
		t.Fatalf("sessions[0].State = %q, want stopped", sessions[0].State)
	}

	meta, err := store.ReadSessionMeta(metaPath)
	if err != nil {
		t.Fatalf("ReadSessionMeta() error = %v", err)
	}
	if meta.State != "stopped" {
		t.Fatalf("meta.State = %q, want stopped", meta.State)
	}
}

func TestReconciliationMarksMissingDirectoryAsOrphaned(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	now := h.now
	if err := h.observer.registry.RegisterSession(testContext(t), store.SessionInfo{
		ID:        "sess-orphan",
		Name:      "Orphan",
		AgentName: "coder",
		Workspace: h.workspace,
		State:     "active",
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("RegisterSession() error = %v", err)
	}

	result, err := h.observer.Reconcile(testContext(t))
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	sort.Strings(result.Orphaned)
	if got, want := result.Orphaned, []string{"sess-orphan"}; !equalStrings(got, want) {
		t.Fatalf("Orphaned = %#v, want %#v", got, want)
	}

	sessions, err := h.observer.registry.ListSessions(testContext(t), store.SessionListQuery{})
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if got, want := len(sessions), 1; got != want {
		t.Fatalf("len(sessions) = %d, want %d", got, want)
	}
	if sessions[0].State != "orphaned" {
		t.Fatalf("sessions[0].State = %q, want orphaned", sessions[0].State)
	}
}

func equalStrings(got []string, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
