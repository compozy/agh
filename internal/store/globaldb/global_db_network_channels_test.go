package globaldb

import (
	"database/sql"
	"errors"
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
	assertTableHasNoForeignKeys(t, globalDB.db, "network_channels")
}

func TestGlobalDBWriteAndListNetworkChannels(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	recordedAt := time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC)
	globalDB.now = func() time.Time { return recordedAt }

	first := store.NetworkChannelEntry{
		Channel:     "coord.core",
		WorkspaceID: "ws-alpha",
		Purpose:     "Cross-agent coordination",
		CreatedBy:   "codex",
	}
	if err := globalDB.WriteNetworkChannel(testutil.Context(t), first); err != nil {
		t.Fatalf("WriteNetworkChannel(first) error = %v", err)
	}
	if err := globalDB.WriteNetworkChannel(testutil.Context(t), store.NetworkChannelEntry{
		Channel:     "coord.core",
		WorkspaceID: "ws-alpha",
		Purpose:     "Updated purpose",
		CreatedBy:   "claude",
		UpdatedAt:   recordedAt.Add(time.Minute),
	}); err != nil {
		t.Fatalf("WriteNetworkChannel(update) error = %v", err)
	}
	if err := globalDB.WriteNetworkChannel(testutil.Context(t), store.NetworkChannelEntry{
		Channel:     "ops.alerts",
		WorkspaceID: "ws-alpha",
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
		WorkspaceID: "ws-alpha",
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
	if err := globalDB.WriteNetworkChannel(testutil.Context(t), store.NetworkChannelEntry{
		Channel:     "coord.core",
		WorkspaceID: "ws-alpha",
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

func TestGlobalDBListNetworkChannelsWrapsTimestampParseFailures(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
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
		"ws-alpha",
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
