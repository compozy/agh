package globaldb

import (
	"database/sql"
	"path/filepath"
	"testing"

	hookspkg "github.com/compozy/agh/internal/hooks"
	"github.com/compozy/agh/internal/testutil"
)

func TestTaskEventsTypeSeqIndexFreshDB(t *testing.T) {
	t.Parallel()

	t.Run("Should install the type-sequence index on a fresh database", func(t *testing.T) {
		t.Parallel()

		globalDB := openTestGlobalDB(t)
		assertTaskEventsTypeSeqIndexReady(t, globalDB.db)
	})
}

func TestTaskEventsTypeSeqIndexReopenAfterRestart(t *testing.T) {
	t.Parallel()

	t.Run("Should retain the type-sequence index after reopening", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		path := filepath.Join(t.TempDir(), GlobalDatabaseName)
		first, err := OpenGlobalDB(ctx, path)
		if err != nil {
			t.Fatalf("OpenGlobalDB() error = %v", err)
		}
		assertTaskEventsTypeSeqIndexReady(t, first.db)
		if err := first.Close(ctx); err != nil {
			t.Fatalf("Close(first) error = %v", err)
		}

		second, err := OpenGlobalDB(ctx, path)
		if err != nil {
			t.Fatalf("OpenGlobalDB(reopen) error = %v", err)
		}
		t.Cleanup(func() {
			if err := second.Close(ctx); err != nil {
				t.Errorf("Close(second) error = %v", err)
			}
		})
		assertTaskEventsTypeSeqIndexReady(t, second.db)
	})
}

func assertTaskEventsTypeSeqIndexReady(t *testing.T, db *sql.DB) {
	t.Helper()

	assertIndexSQLContains(t, db, "idx_task_events_type_seq", "task_events(event_type, event_seq)")
	assertQueryPlanUsesIndex(
		t,
		db,
		`SELECT MAX(event_seq) FROM task_events WHERE event_type = ?`,
		"idx_task_events_type_seq",
		string(hookspkg.HookTaskStatusChanged),
	)
}
