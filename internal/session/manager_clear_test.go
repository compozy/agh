package session

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/compozy/agh/internal/acp"
	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/testutil"
)

func TestClearConversationRestartsSameSessionWithFreshContext(t *testing.T) {
	t.Parallel()

	t.Run("Should restart the same session with fresh provider and transcript state", func(t *testing.T) {
		h := newHarness(t)
		session := createSession(t, h)

		firstEvents, err := h.manager.Prompt(testutil.Context(t), session.ID, "before clear")
		if err != nil {
			t.Fatalf("Prompt(before clear) error = %v", err)
		}
		collectEvents(t, firstEvents)

		originalACP := session.Info().ACPSessionID

		cleared, err := h.manager.ClearConversation(testutil.Context(t), session.ID)
		if err != nil {
			t.Fatalf("ClearConversation() error = %v", err)
		}
		t.Cleanup(func() {
			if err := h.manager.Stop(testutil.Context(t), cleared.ID); err != nil {
				t.Fatalf("cleanup Stop() error = %v", err)
			}
		})

		if got, want := cleared.ID, session.ID; got != want {
			t.Fatalf("cleared.ID = %q, want %q", got, want)
		}
		if got := cleared.Info().State; got != StateActive {
			t.Fatalf("cleared state = %q, want %q", got, StateActive)
		}
		if got := cleared.Info().ACPSessionID; got == "" || got == originalACP {
			t.Fatalf("cleared ACP session id = %q, want fresh non-empty id distinct from %q", got, originalACP)
		}
		if got := len(h.driver.startCalls); got != 2 {
			t.Fatalf("len(startCalls) = %d, want 2", got)
		}
		if got := h.driver.startCalls[1].ResumeSessionID; got != "" {
			t.Fatalf("clear restart ResumeSessionID = %q, want empty for fresh provider context", got)
		}

		messages, err := h.manager.Transcript(testutil.Context(t), cleared.ID)
		if err != nil {
			t.Fatalf("Transcript(after clear) error = %v", err)
		}
		if got := len(messages); got != 0 {
			t.Fatalf("Transcript(after clear) len = %d, want 0", got)
		}

		stored := readStoredEvents(t, cleared)
		if got := len(stored); got != 0 {
			t.Fatalf("stored events after clear = %d, want 0", got)
		}

		secondEvents, err := h.manager.Prompt(testutil.Context(t), cleared.ID, "after clear")
		if err != nil {
			t.Fatalf("Prompt(after clear) error = %v", err)
		}
		collectEvents(t, secondEvents)

		stored = readStoredEvents(t, cleared)
		if got := len(stored); got == 0 {
			t.Fatal("stored events after second prompt = 0, want persisted prompt data")
		}
		for _, event := range stored {
			if strings.Contains(event.Content, "before clear") {
				t.Fatalf("stored event content still contains cleared prompt: %s", event.Content)
			}
		}
	})
}

func TestClearConversationRejectsPromptInProgress(t *testing.T) {
	t.Parallel()

	t.Run("Should reject clearing while a prompt is in progress", func(t *testing.T) {
		h := newHarness(t)
		session := createSession(t, h)
		releasePrompt := make(chan struct{})
		h.driver.promptHook = func(_ *fakeProcess, req acp.PromptRequest) (<-chan acp.AgentEvent, error) {
			events := make(chan acp.AgentEvent)
			go func() {
				defer close(events)
				<-releasePrompt
				events <- acp.AgentEvent{
					Type:      acp.EventTypeDone,
					SessionID: session.Info().ACPSessionID,
					TurnID:    req.TurnID,
				}
			}()
			return events, nil
		}

		eventsCh, err := h.manager.Prompt(testutil.Context(t), session.ID, "hello")
		if err != nil {
			t.Fatalf("Prompt() error = %v", err)
		}
		waitForCondition(t, "prompt setup", func() bool {
			return session.IsPrompting()
		})

		_, err = h.manager.ClearConversation(testutil.Context(t), session.ID)
		if !errors.Is(err, ErrPromptInProgress) {
			t.Fatalf("ClearConversation() error = %v, want %v", err, ErrPromptInProgress)
		}

		close(releasePrompt)
		collectEvents(t, eventsCh)
		if stopErr := h.manager.Stop(testutil.Context(t), session.ID); stopErr != nil {
			t.Fatalf("cleanup Stop() error = %v", stopErr)
		}
	})
}

func TestBackupSessionDB(t *testing.T) {
	t.Parallel()

	t.Run("Should roll back partial rename failures", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "session.db")
		walPath := dbPath + "-wal"

		if err := os.WriteFile(dbPath, []byte("db"), 0o600); err != nil {
			t.Fatalf("WriteFile(session.db) error = %v", err)
		}
		if err := os.WriteFile(walPath, []byte("wal"), 0o600); err != nil {
			t.Fatalf("WriteFile(session.db-wal) error = %v", err)
		}

		blockedBackupDir := walPath + ".clear-backup"
		if err := os.Mkdir(blockedBackupDir, 0o755); err != nil {
			t.Fatalf("Mkdir(blocked backup) error = %v", err)
		}
		if err := os.WriteFile(filepath.Join(blockedBackupDir, "sentinel"), []byte("x"), 0o600); err != nil {
			t.Fatalf("WriteFile(blocked backup sentinel) error = %v", err)
		}

		_, err := backupSessionDB(dbPath)
		if err == nil {
			t.Fatal("backupSessionDB() error = nil, want rollback failure path")
		}
		if !strings.Contains(err.Error(), "remove stale clear backup") {
			t.Fatalf("backupSessionDB() error = %v, want stale backup failure", err)
		}

		if got, readErr := os.ReadFile(dbPath); readErr != nil || string(got) != "db" {
			t.Fatalf("ReadFile(restored session.db) = %q, %v, want db", got, readErr)
		}
		if got, readErr := os.ReadFile(walPath); readErr != nil || string(got) != "wal" {
			t.Fatalf("ReadFile(original session.db-wal) = %q, %v, want wal", got, readErr)
		}
		if _, statErr := os.Stat(dbPath + ".clear-backup"); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("Stat(session.db.clear-backup) error = %v, want os.ErrNotExist", statErr)
		}
		if _, statErr := os.Stat(blockedBackupDir); statErr != nil {
			t.Fatalf("Stat(blocked backup dir) error = %v", statErr)
		}
	})

	t.Run("Should restore interrupted clear backups before stored event queries", func(t *testing.T) {
		h := newHarness(t)
		session := createSession(t, h)

		eventsCh, err := h.manager.Prompt(testutil.Context(t), session.ID, "before interrupted clear")
		if err != nil {
			t.Fatalf("Prompt() error = %v", err)
		}
		collectEvents(t, eventsCh)
		if err := h.manager.Stop(testutil.Context(t), session.ID); err != nil {
			t.Fatalf("Stop() error = %v", err)
		}

		dbPath := session.DBPath()
		backups, err := backupSessionDB(dbPath)
		if err != nil {
			t.Fatalf("backupSessionDB() error = %v", err)
		}
		if got := len(backups); got == 0 {
			t.Fatal("backupSessionDB() backups = 0, want at least session database backup")
		}
		if _, statErr := os.Stat(dbPath); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("Stat(session database after backup) error = %v, want os.ErrNotExist", statErr)
		}
		if _, statErr := os.Stat(dbPath + ".clear-backup"); statErr != nil {
			t.Fatalf("Stat(session database backup) error = %v", statErr)
		}

		freshManager := newManagerWithHarness(t, h)
		events, err := freshManager.Events(testutil.Context(t), session.ID, store.EventQuery{})
		if err != nil {
			t.Fatalf("Events(after interrupted clear backup) error = %v", err)
		}
		if got := len(events); got == 0 {
			t.Fatal("Events(after interrupted clear backup) = 0, want restored transcript events")
		}
		if _, statErr := os.Stat(dbPath); statErr != nil {
			t.Fatalf("Stat(restored session database) error = %v", statErr)
		}
		if _, statErr := os.Stat(dbPath + ".clear-backup"); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("Stat(discarded interrupted backup) error = %v, want os.ErrNotExist", statErr)
		}
	})

	t.Run("Should discard committed clear backups without restoring old events", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "session.db")
		backupPath := dbPath + ".clear-backup"

		if err := os.WriteFile(dbPath, []byte("fresh"), 0o600); err != nil {
			t.Fatalf("WriteFile(fresh session.db) error = %v", err)
		}
		if err := os.WriteFile(backupPath, []byte("old"), 0o600); err != nil {
			t.Fatalf("WriteFile(session.db clear backup) error = %v", err)
		}
		if err := commitSessionDBClear(dbPath); err != nil {
			t.Fatalf("commitSessionDBClear() error = %v", err)
		}

		if err := recoverSessionDBClear(dbPath); err != nil {
			t.Fatalf("recoverSessionDBClear(committed) error = %v", err)
		}
		if got, readErr := os.ReadFile(dbPath); readErr != nil || string(got) != "fresh" {
			t.Fatalf("ReadFile(session.db) = %q, %v, want fresh", got, readErr)
		}
		if _, statErr := os.Stat(backupPath); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("Stat(committed backup) error = %v, want os.ErrNotExist", statErr)
		}
		if _, statErr := os.Stat(sessionDBClearCommitPath(dbPath)); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("Stat(clear commit marker) error = %v, want os.ErrNotExist", statErr)
		}
	})
}
