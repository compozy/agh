package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/notifications"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/testutil"
)

func TestGlobalDBNotificationCursorSchemaMigration(t *testing.T) {
	t.Parallel()

	t.Run("Should create notification cursor schema on fresh DB", func(t *testing.T) {
		t.Parallel()

		globalDB := openTestGlobalDB(t)

		assertNotificationCursorSchema(t, globalDB.db)
	})

	t.Run("Should migrate previous global schema", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		dbPath := filepath.Join(t.TempDir(), GlobalDatabaseName)
		legacyDB := openPreviousNotificationCursorSchemaDB(t, dbPath)
		insertMigrationRecordsThroughVersion(t, legacyDB, 18)
		if err := legacyDB.Close(); err != nil {
			t.Fatalf("legacyDB.Close() error = %v", err)
		}

		globalDB, err := OpenGlobalDB(ctx, dbPath)
		if err != nil {
			t.Fatalf("OpenGlobalDB() error = %v", err)
		}
		t.Cleanup(func() {
			if err := globalDB.Close(ctx); err != nil {
				t.Fatalf("Close() error = %v", err)
			}
		})

		assertNotificationCursorSchema(t, globalDB.db)
	})
}

func TestNotificationCursorSchemaStatements(t *testing.T) {
	t.Parallel()

	t.Run("Should use shared notification cursor DDL in fresh global schema", func(t *testing.T) {
		t.Parallel()

		for _, statement := range notificationCursorSchemaStatements() {
			if !schemaStatementsContain(globalSchemaStatements, statement) {
				t.Fatalf("globalSchemaStatements missing notification cursor statement %q", statement)
			}
		}
	})
}

func TestGlobalDBNotificationCursorStore(t *testing.T) {
	t.Parallel()

	t.Run("Should advance and read a cursor", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		globalDB := openTestGlobalDB(t)
		service := notifications.NewService(globalDB)
		key := notificationCursorTestKey()
		now := notificationCursorTestTime()

		cursor, err := service.Advance(ctx, notifications.AdvanceCursor{
			Key:             key,
			LastSequence:    7,
			DeliveryID:      "delivery-7",
			LastDeliveredAt: now.Add(-time.Minute),
			Now:             now,
		})
		if err != nil {
			t.Fatalf("Advance() error = %v", err)
		}
		if cursor.LastSequence != 7 || cursor.LastDeliveryID != "delivery-7" {
			t.Fatalf("cursor = %#v, want sequence 7 delivery-7", cursor)
		}

		stored, err := service.Get(ctx, key)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if stored.LastSequence != cursor.LastSequence || stored.LastDeliveryID != cursor.LastDeliveryID {
			t.Fatalf("stored cursor = %#v, want %#v", stored, cursor)
		}
	})

	t.Run("Should refresh idempotent replay diagnostics and updated timestamp", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		globalDB := openTestGlobalDB(t)
		service := notifications.NewService(globalDB)
		key := notificationCursorTestKey()
		firstTime := notificationCursorTestTime()

		first, err := service.Advance(ctx, notifications.AdvanceCursor{
			Key:          key,
			LastSequence: 11,
			DeliveryID:   "delivery-11",
			Now:          firstTime,
		})
		if err != nil {
			t.Fatalf("Advance(first) error = %v", err)
		}
		if _, err := service.RecordError(ctx, notifications.CursorError{
			Key:       key,
			LastError: "bridge delivery failed",
			Now:       firstTime.Add(30 * time.Minute),
		}); err != nil {
			t.Fatalf("RecordError() error = %v", err)
		}
		second, err := service.Advance(ctx, notifications.AdvanceCursor{
			Key:          key,
			LastSequence: 11,
			DeliveryID:   "delivery-11",
			Now:          firstTime.Add(time.Hour),
		})
		if err != nil {
			t.Fatalf("Advance(replay) error = %v", err)
		}

		if got, want := second.LastError, ""; got != want {
			t.Fatalf("idempotent replay LastError = %q, want empty", got)
		}
		if !second.UpdatedAt.Equal(firstTime.Add(time.Hour).UTC()) {
			t.Fatalf("idempotent replay UpdatedAt = %s, want %s", second.UpdatedAt, firstTime.Add(time.Hour).UTC())
		}
		if !second.LastDeliveredAt.Equal(first.LastDeliveredAt) {
			t.Fatalf("idempotent replay LastDeliveredAt = %s, want %s", second.LastDeliveredAt, first.LastDeliveredAt)
		}
	})

	t.Run("Should reject non monotonic advances", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		globalDB := openTestGlobalDB(t)
		service := notifications.NewService(globalDB)
		key := notificationCursorTestKey()
		now := notificationCursorTestTime()

		if _, err := service.Advance(ctx, notifications.AdvanceCursor{
			Key:          key,
			LastSequence: 15,
			DeliveryID:   "delivery-15",
			Now:          now,
		}); err != nil {
			t.Fatalf("Advance(seed) error = %v", err)
		}
		if _, err := service.Advance(ctx, notifications.AdvanceCursor{
			Key:          key,
			LastSequence: 15,
			DeliveryID:   "delivery-15b",
			Now:          now.Add(time.Minute),
		}); !errors.Is(err, notifications.ErrNonMonotonicCursor) {
			t.Fatalf("Advance(forked replay) error = %v, want ErrNonMonotonicCursor", err)
		}
		if _, err := service.Advance(ctx, notifications.AdvanceCursor{
			Key:          key,
			LastSequence: 14,
			DeliveryID:   "delivery-14",
			Now:          now.Add(2 * time.Minute),
		}); !errors.Is(err, notifications.ErrNonMonotonicCursor) {
			t.Fatalf("Advance(backward) error = %v, want ErrNonMonotonicCursor", err)
		}
	})

	t.Run("Should reset a cursor only with an explicit reason", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		globalDB := openTestGlobalDB(t)
		service := notifications.NewService(globalDB)
		key := notificationCursorTestKey()
		now := notificationCursorTestTime()

		if _, err := service.Advance(ctx, notifications.AdvanceCursor{
			Key:          key,
			LastSequence: 21,
			DeliveryID:   "delivery-21",
			Now:          now,
		}); err != nil {
			t.Fatalf("Advance(seed) error = %v", err)
		}
		if _, err := service.Reset(ctx, notifications.ResetCursor{
			Key:          key,
			LastSequence: 2,
			Reason:       "operator replay after bridge repair",
			Now:          now.Add(time.Minute),
		}); err != nil {
			t.Fatalf("Reset() error = %v", err)
		}

		cursor, err := service.Get(ctx, key)
		if err != nil {
			t.Fatalf("Get(after reset) error = %v", err)
		}
		if cursor.LastSequence != 2 || cursor.LastDeliveryID != "" || !cursor.LastDeliveredAt.IsZero() {
			t.Fatalf("cursor after reset = %#v, want sequence 2 with cleared delivery metadata", cursor)
		}
		if _, err := service.Reset(ctx, notifications.ResetCursor{
			Key:          key,
			LastSequence: 0,
			Now:          now.Add(2 * time.Minute),
		}); !errors.Is(err, notifications.ErrResetReasonRequired) {
			t.Fatalf("Reset(without reason) error = %v, want ErrResetReasonRequired", err)
		}
	})

	t.Run("Should record errors without advancing delivery progress", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		globalDB := openTestGlobalDB(t)
		service := notifications.NewService(globalDB)
		key := notificationCursorTestKey()
		now := notificationCursorTestTime()

		if _, err := service.Advance(ctx, notifications.AdvanceCursor{
			Key:          key,
			LastSequence: 31,
			DeliveryID:   "delivery-31",
			Now:          now,
		}); err != nil {
			t.Fatalf("Advance(seed) error = %v", err)
		}
		cursor, err := service.RecordError(ctx, notifications.CursorError{
			Key:       key,
			LastError: "bridge delivery failed",
			Now:       now.Add(time.Minute),
		})
		if err != nil {
			t.Fatalf("RecordError() error = %v", err)
		}
		if cursor.LastSequence != 31 || cursor.LastDeliveryID != "delivery-31" {
			t.Fatalf("cursor after RecordError = %#v, want original delivery progress", cursor)
		}
		if cursor.LastError != "bridge delivery failed" {
			t.Fatalf("cursor.LastError = %q, want diagnostic", cursor.LastError)
		}
	})

	t.Run("Should create a cursor when recording the first error", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		globalDB := openTestGlobalDB(t)
		service := notifications.NewService(globalDB)
		key := notificationCursorTestKey()
		now := notificationCursorTestTime()

		cursor, err := service.RecordError(ctx, notifications.CursorError{
			Key:       key,
			LastError: "bridge delivery failed",
			Now:       now,
		})
		if err != nil {
			t.Fatalf("RecordError() error = %v", err)
		}
		if cursor.LastSequence != 0 || cursor.LastDeliveryID != "" {
			t.Fatalf("cursor after first RecordError = %#v, want zero delivery progress", cursor)
		}
		if cursor.LastError != "bridge delivery failed" {
			t.Fatalf("cursor.LastError = %q, want diagnostic", cursor.LastError)
		}
		if !cursor.UpdatedAt.Equal(now.UTC()) {
			t.Fatalf("cursor.UpdatedAt = %s, want %s", cursor.UpdatedAt, now.UTC())
		}
	})

	t.Run("Should list cursors with stable filters", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		globalDB := openTestGlobalDB(t)
		service := notifications.NewService(globalDB)
		now := notificationCursorTestTime()
		inputs := []notifications.AdvanceCursor{
			{
				Key: notifications.CursorKey{
					ConsumerID: "consumer-a",
					StreamName: "task_events",
					SubjectID:  "task-a",
				},
				LastSequence: 1,
				DeliveryID:   "delivery-a",
				Now:          now,
			},
			{
				Key: notifications.CursorKey{
					ConsumerID: "consumer-b",
					StreamName: "task_events",
					SubjectID:  "task-b",
				},
				LastSequence: 2,
				DeliveryID:   "delivery-b",
				Now:          now,
			},
		}
		for _, input := range inputs {
			if _, err := service.Advance(ctx, input); err != nil {
				t.Fatalf("Advance(%q) error = %v", input.Key.ConsumerID, err)
			}
		}

		cursors, err := service.List(ctx, notifications.CursorQuery{StreamName: "task_events", Limit: 1})
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(cursors) != 1 || cursors[0].Key.ConsumerID != "consumer-a" {
			t.Fatalf("List() = %#v, want first stable task_events cursor", cursors)
		}
	})
}

func assertNotificationCursorSchema(t *testing.T, db *sql.DB) {
	t.Helper()

	assertTablesPresent(t, db, "notification_cursors")
	assertTableColumns(t, db, "notification_cursors", []string{
		"consumer_id",
		"stream_name",
		"subject_id",
		"last_sequence",
		"last_delivery_id",
		"last_delivered_at",
		"last_error",
		"updated_at",
	})
	assertIndexesPresent(t, db, "notification_cursors", "notification_cursors_stream_sequence_idx")
}

func openPreviousNotificationCursorSchemaDB(t *testing.T, dbPath string) *sql.DB {
	t.Helper()

	ctx := testutil.Context(t)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}

	excluded := make(map[string]struct{})
	for _, statement := range notificationCursorSchemaStatements() {
		excluded[statement] = struct{}{}
	}
	for _, statement := range globalSchemaStatements {
		if _, ok := excluded[statement]; ok {
			continue
		}
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatalf("ExecContext(previous notification schema) error = %v", err)
		}
	}
	if err := store.RunMigrations(ctx, db, nil); err != nil {
		t.Fatalf("RunMigrations(empty) error = %v", err)
	}
	return db
}

func notificationCursorTestKey() notifications.CursorKey {
	return notifications.CursorKey{
		ConsumerID: "bridge_task_subscription:sub-1",
		StreamName: "task_events",
		SubjectID:  "task-1",
	}
}

func notificationCursorTestTime() time.Time {
	return time.Date(2026, 5, 5, 15, 0, 0, 0, time.UTC)
}

func TestNotificationCursorRollbackContext(t *testing.T) {
	t.Parallel()

	parent, cancel := context.WithCancel(context.Background())
	cancel()

	rollbackCtx, rollbackCancel := notificationCursorRollbackContext(parent)
	defer rollbackCancel()

	if rollbackCtx.Err() != nil {
		t.Fatalf("rollbackCtx.Err() = %v, want nil after detaching parent cancellation", rollbackCtx.Err())
	}
	if _, ok := rollbackCtx.Deadline(); !ok {
		t.Fatal("rollbackCtx has no deadline, want bounded rollback timeout")
	}
}
