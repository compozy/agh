package globaldb

import (
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/testutil"
)

func TestOpenGlobalDBCreatesNetworkChannelsSchema(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)

	assertTablesPresent(t, globalDB.db, "network_channels")
	assertTableColumns(t, globalDB.db, "network_channels", []string{
		"channel",
		"workspace_id",
		"purpose",
		"created_by",
		"created_at",
		"updated_at",
	})
	hasWorkspaceFK, err := tableHasForeignKey(testutil.Context(t), globalDB.db, "network_channels", "workspaces")
	if err != nil {
		t.Fatalf("tableHasForeignKey(network_channels, workspaces) error = %v", err)
	}
	if !hasWorkspaceFK {
		t.Fatal("network_channels is missing a workspaces foreign key")
	}
	assertIndexesPresent(t, globalDB.db, "network_channels", "idx_network_channels_workspace_updated_at")
}

func TestGlobalDBWriteAndListNetworkChannels(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	workspaceID := registerWorkspaceForGlobalTests(
		t,
		globalDB,
		"ws-alpha",
		filepath.Join(t.TempDir(), "ws-alpha"),
	)
	recordedAt := time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC)
	globalDB.now = func() time.Time { return recordedAt }

	first := store.NetworkChannelEntry{
		Channel:     " coord.core ",
		WorkspaceID: workspaceID,
		Purpose:     "Cross-agent coordination",
		CreatedBy:   "codex",
	}
	if err := globalDB.WriteNetworkChannel(testutil.Context(t), first); err != nil {
		t.Fatalf("WriteNetworkChannel(first) error = %v", err)
	}
	if err := globalDB.WriteNetworkChannel(testutil.Context(t), store.NetworkChannelEntry{
		Channel:     "coord.core",
		WorkspaceID: workspaceID,
		Purpose:     "Updated purpose",
		CreatedBy:   "claude",
		UpdatedAt:   recordedAt.Add(time.Minute),
	}); err != nil {
		t.Fatalf("WriteNetworkChannel(update) error = %v", err)
	}
	if err := globalDB.WriteNetworkChannel(testutil.Context(t), store.NetworkChannelEntry{
		Channel:     "ops.alerts",
		WorkspaceID: workspaceID,
		Purpose:     "Operational alerts",
		CreatedBy:   "gemini",
		CreatedAt:   recordedAt.Add(2 * time.Minute),
		UpdatedAt:   recordedAt.Add(2 * time.Minute),
	}); err != nil {
		t.Fatalf("WriteNetworkChannel(second) error = %v", err)
	}

	entry, err := globalDB.GetNetworkChannel(testutil.Context(t), "coord.core")
	if err != nil {
		t.Fatalf("GetNetworkChannel() error = %v", err)
	}
	if got, want := entry.Channel, "coord.core"; got != want {
		t.Fatalf("entry.Channel = %q, want %q", got, want)
	}
	if got, want := entry.Purpose, "Updated purpose"; got != want {
		t.Fatalf("entry.Purpose = %q, want %q", got, want)
	}
	if got, want := entry.CreatedBy, "codex"; got != want {
		t.Fatalf("entry.CreatedBy = %q, want %q", got, want)
	}
	if got, want := entry.CreatedAt, recordedAt; !got.Equal(want) {
		t.Fatalf("entry.CreatedAt = %s, want %s", got, want)
	}

	entries, err := globalDB.ListNetworkChannels(testutil.Context(t), store.NetworkChannelQuery{
		WorkspaceID: workspaceID,
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("ListNetworkChannels() error = %v", err)
	}
	if got, want := len(entries), 2; got != want {
		t.Fatalf("len(entries) = %d, want %d", got, want)
	}
	if got, want := entries[0].Channel, "ops.alerts"; got != want {
		t.Fatalf("entries[0].Channel = %q, want %q", got, want)
	}
}

func TestGlobalDBGetNetworkChannelNotFound(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	_, err := globalDB.GetNetworkChannel(testutil.Context(t), "missing")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("GetNetworkChannel(missing) error = %v, want sql.ErrNoRows", err)
	}
}

func TestGlobalDBDeleteNetworkChannel(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	workspaceID := registerWorkspaceForGlobalTests(
		t,
		globalDB,
		"ws-alpha",
		filepath.Join(t.TempDir(), "ws-alpha"),
	)
	if err := globalDB.WriteNetworkChannel(testutil.Context(t), store.NetworkChannelEntry{
		Channel:     " coord.core ",
		WorkspaceID: workspaceID,
		Purpose:     "Cross-agent coordination",
		CreatedBy:   "codex",
	}); err != nil {
		t.Fatalf("WriteNetworkChannel() error = %v", err)
	}
	if err := globalDB.DeleteNetworkChannel(testutil.Context(t), "coord.core"); err != nil {
		t.Fatalf("DeleteNetworkChannel() error = %v", err)
	}
	if _, err := globalDB.GetNetworkChannel(testutil.Context(t), "coord.core"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("GetNetworkChannel(after delete) error = %v, want sql.ErrNoRows", err)
	}
}

func TestGlobalDBDeleteWorkspaceCascadesNetworkChannels(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	workspaceID := registerWorkspaceForGlobalTests(
		t,
		globalDB,
		"ws-alpha",
		filepath.Join(t.TempDir(), "ws-alpha"),
	)
	if err := globalDB.WriteNetworkChannel(testutil.Context(t), store.NetworkChannelEntry{
		Channel:     "coord.core",
		WorkspaceID: workspaceID,
		Purpose:     "Cross-agent coordination",
		CreatedBy:   "codex",
	}); err != nil {
		t.Fatalf("WriteNetworkChannel() error = %v", err)
	}

	if err := globalDB.DeleteWorkspace(testutil.Context(t), workspaceID); err != nil {
		t.Fatalf("DeleteWorkspace() error = %v", err)
	}
	if _, err := globalDB.GetNetworkChannel(testutil.Context(t), "coord.core"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("GetNetworkChannel(after workspace delete) error = %v, want sql.ErrNoRows", err)
	}
}

func TestGlobalDBListNetworkChannelsWrapsTimestampParseFailures(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	workspaceID := registerWorkspaceForGlobalTests(
		t,
		globalDB,
		"ws-alpha",
		filepath.Join(t.TempDir(), "ws-alpha"),
	)
	if _, err := globalDB.db.ExecContext(
		testutil.Context(t),
		`INSERT INTO network_channels (
			channel,
			workspace_id,
			purpose,
			created_by,
			created_at,
			updated_at
		) VALUES (?, ?, ?, ?, ?, ?)`,
		"coord.core",
		workspaceID,
		"Cross-agent coordination",
		"codex",
		"not-a-timestamp",
		store.FormatTimestamp(time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC)),
	); err != nil {
		t.Fatalf("ExecContext(insert invalid network channel) error = %v", err)
	}

	_, err := globalDB.ListNetworkChannels(testutil.Context(t), store.NetworkChannelQuery{})
	if err == nil {
		t.Fatal("ListNetworkChannels(invalid timestamp) error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "parse network channel created_at") {
		t.Fatalf("ListNetworkChannels(invalid timestamp) error = %v, want wrapped timestamp parse context", err)
	}
}

func TestMigrateGlobalSchemaRebuildsNetworkChannelsWithWorkspaceForeignKey(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t)
	db, err := store.OpenSQLiteDatabase(ctx, filepath.Join(t.TempDir(), "network-channels-migrate.db"), nil)
	if err != nil {
		t.Fatalf("OpenSQLiteDatabase() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	statements := []string{
		`CREATE TABLE workspaces (
			id            TEXT PRIMARY KEY,
			root_dir      TEXT NOT NULL UNIQUE,
			add_dirs      TEXT NOT NULL DEFAULT '[]',
			name          TEXT NOT NULL UNIQUE,
			default_agent TEXT DEFAULT '',
			created_at    TEXT NOT NULL,
			updated_at    TEXT NOT NULL
		);`,
		`CREATE TABLE network_channels (
			channel      TEXT PRIMARY KEY,
			workspace_id TEXT NOT NULL,
			purpose      TEXT NOT NULL,
			created_by   TEXT NOT NULL DEFAULT '',
			created_at   TEXT NOT NULL,
			updated_at   TEXT NOT NULL
		);`,
		`INSERT INTO workspaces (id, root_dir, add_dirs, name, default_agent, created_at, updated_at)
		 VALUES ('ws-alpha', '/tmp/ws-alpha', '[]', 'ws-alpha', '', '2026-04-11T12:00:00Z', '2026-04-11T12:00:00Z')`,
		`INSERT INTO network_channels (channel, workspace_id, purpose, created_by, created_at, updated_at)
		 VALUES ('coord.core', 'ws-alpha', 'Coordination', 'codex', '2026-04-11T12:00:00Z', '2026-04-11T12:00:00Z')`,
		`INSERT INTO network_channels (channel, workspace_id, purpose, created_by, created_at, updated_at)
		 VALUES ('orphaned', 'ws-missing', 'Stale row', 'codex', '2026-04-11T12:00:00Z', '2026-04-11T12:00:00Z')`,
	}
	for _, stmt := range statements {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("ExecContext(%q) error = %v", stmt, err)
		}
	}

	if err := migrateGlobalSchema(ctx, db); err != nil {
		t.Fatalf("migrateGlobalSchema() error = %v", err)
	}

	hasWorkspaceFK, err := tableHasForeignKey(ctx, db, "network_channels", "workspaces")
	if err != nil {
		t.Fatalf("tableHasForeignKey(network_channels, workspaces) error = %v", err)
	}
	if !hasWorkspaceFK {
		t.Fatal("network_channels is missing a workspaces foreign key after migration")
	}

	rows, err := db.QueryContext(ctx, `SELECT channel FROM network_channels ORDER BY channel ASC`)
	if err != nil {
		t.Fatalf("QueryContext(list migrated network channels) error = %v", err)
	}
	defer func() { _ = rows.Close() }()

	channels := make([]string, 0, 1)
	for rows.Next() {
		var channel string
		if err := rows.Scan(&channel); err != nil {
			t.Fatalf("Scan(channel) error = %v", err)
		}
		channels = append(channels, channel)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows.Err(list migrated network channels) error = %v", err)
	}
	if got, want := channels, []string{"coord.core"}; !testutil.EqualStringSlices(got, want) {
		t.Fatalf("migrated channels = %#v, want %#v", got, want)
	}

	if _, err := db.ExecContext(ctx, `DELETE FROM workspaces WHERE id = ?`, "ws-alpha"); err != nil {
		t.Fatalf("ExecContext(delete workspace) error = %v", err)
	}

	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM network_channels`).Scan(&count); err != nil {
		t.Fatalf("QueryRowContext(count migrated channels) error = %v", err)
	}
	if count != 0 {
		t.Fatalf("network_channels count = %d, want 0 after workspace delete", count)
	}
}
