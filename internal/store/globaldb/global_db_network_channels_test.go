package globaldb

import (
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/testutil"
)

const networkStoreTestWorkspaceID = "ws-network-store"

func TestNetworkChannels(t *testing.T) {
	tests := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "Should create the network channels schema on open",
			run:  assertOpenGlobalDBCreatesNetworkChannelsSchema,
		},
		{
			name: "Should write and list network channels",
			run:  assertGlobalDBWriteAndListNetworkChannels,
		},
		{
			name: "Should patch network channels without replacing unspecified fields",
			run:  assertGlobalDBPatchNetworkChannelsPreservesUnspecifiedFields,
		},
		{
			name: "Should reject patches that break channel fanout coupling",
			run:  assertGlobalDBPatchNetworkChannelRejectsInvalidFanoutCoupling,
		},
		{
			name: "Should return sql.ErrNoRows for missing network channels",
			run:  assertGlobalDBGetNetworkChannelNotFound,
		},
		{
			name: "Should delete a network channel",
			run:  assertGlobalDBDeleteNetworkChannel,
		},
		{
			name: "Should cascade network channels when a workspace is deleted",
			run:  assertGlobalDBDeleteWorkspaceCascadesNetworkChannels,
		},
		{
			name: "Should wrap timestamp parse failures when listing network channels",
			run:  assertGlobalDBListNetworkChannelsWrapsTimestampParseFailures,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t)
		})
	}
}

func assertOpenGlobalDBCreatesNetworkChannelsSchema(t *testing.T) {
	t.Helper()

	globalDB := openTestGlobalDB(t)

	assertTablesPresent(t, globalDB.db, "network_channels")
	assertTableColumns(t, globalDB.db, "network_channels", []string{
		"workspace_id",
		"channel",
		"purpose",
		"created_by",
		"created_at",
		"updated_at",
		"fanout_policy",
		"coordinator_peer_id",
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

func assertGlobalDBWriteAndListNetworkChannels(t *testing.T) {
	t.Helper()

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

	entry, err := globalDB.GetNetworkChannel(testutil.Context(t), store.NetworkChannelRef{
		WorkspaceID: workspaceID,
		Channel:     "coord.core",
	})
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

func assertGlobalDBPatchNetworkChannelsPreservesUnspecifiedFields(t *testing.T) {
	t.Helper()

	globalDB := openTestGlobalDB(t)
	workspaceID := registerWorkspaceForGlobalTests(
		t,
		globalDB,
		"ws-alpha",
		filepath.Join(t.TempDir(), "ws-alpha"),
	)
	recordedAt := time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC)
	if err := globalDB.WriteNetworkChannel(testutil.Context(t), store.NetworkChannelEntry{
		Channel:           "coord.core",
		WorkspaceID:       workspaceID,
		Purpose:           "Original purpose",
		FanoutPolicy:      store.NetworkFanoutPolicyCoordinator,
		CoordinatorPeerID: "reviewer.sess-a",
		CreatedBy:         "codex",
		CreatedAt:         recordedAt,
		UpdatedAt:         recordedAt,
	}); err != nil {
		t.Fatalf("WriteNetworkChannel() error = %v", err)
	}

	purpose := "Pair reviews"
	if err := globalDB.PatchNetworkChannel(
		testutil.Context(t),
		store.NetworkChannelRef{WorkspaceID: workspaceID, Channel: "coord.core"},
		store.NetworkChannelPatch{Purpose: &purpose, UpdatedAt: recordedAt.Add(time.Minute)},
	); err != nil {
		t.Fatalf("PatchNetworkChannel(purpose) error = %v", err)
	}
	entry, err := globalDB.GetNetworkChannel(testutil.Context(t), store.NetworkChannelRef{
		WorkspaceID: workspaceID,
		Channel:     "coord.core",
	})
	if err != nil {
		t.Fatalf("GetNetworkChannel(after purpose patch) error = %v", err)
	}
	if entry.Purpose != "Pair reviews" ||
		entry.FanoutPolicy != store.NetworkFanoutPolicyCoordinator ||
		entry.CoordinatorPeerID != "reviewer.sess-a" {
		t.Fatalf("entry after purpose patch = %#v", entry)
	}

	fanoutPolicy := store.NetworkFanoutPolicyCapabilityMatch
	coordinatorPeerID := ""
	if err := globalDB.PatchNetworkChannel(
		testutil.Context(t),
		store.NetworkChannelRef{WorkspaceID: workspaceID, Channel: "coord.core"},
		store.NetworkChannelPatch{
			FanoutPolicy:      &fanoutPolicy,
			CoordinatorPeerID: &coordinatorPeerID,
			UpdatedAt:         recordedAt.Add(2 * time.Minute),
		},
	); err != nil {
		t.Fatalf("PatchNetworkChannel(policy) error = %v", err)
	}
	entry, err = globalDB.GetNetworkChannel(testutil.Context(t), store.NetworkChannelRef{
		WorkspaceID: workspaceID,
		Channel:     "coord.core",
	})
	if err != nil {
		t.Fatalf("GetNetworkChannel(after policy patch) error = %v", err)
	}
	if entry.Purpose != "Pair reviews" ||
		entry.FanoutPolicy != store.NetworkFanoutPolicyCapabilityMatch ||
		entry.CoordinatorPeerID != "" {
		t.Fatalf("entry after policy patch = %#v", entry)
	}
}

func assertGlobalDBPatchNetworkChannelRejectsInvalidFanoutCoupling(t *testing.T) {
	t.Helper()

	globalDB := openTestGlobalDB(t)
	workspaceID := registerWorkspaceForGlobalTests(
		t,
		globalDB,
		"ws-alpha",
		filepath.Join(t.TempDir(), "ws-alpha"),
	)
	recordedAt := time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC)
	if err := globalDB.WriteNetworkChannel(testutil.Context(t), store.NetworkChannelEntry{
		Channel:           "coord.core",
		WorkspaceID:       workspaceID,
		Purpose:           "Original purpose",
		FanoutPolicy:      store.NetworkFanoutPolicyCoordinator,
		CoordinatorPeerID: "reviewer.sess-a",
		CreatedBy:         "codex",
		CreatedAt:         recordedAt,
		UpdatedAt:         recordedAt,
	}); err != nil {
		t.Fatalf("WriteNetworkChannel() error = %v", err)
	}

	coordinatorPeerID := ""
	err := globalDB.PatchNetworkChannel(
		testutil.Context(t),
		store.NetworkChannelRef{WorkspaceID: workspaceID, Channel: "coord.core"},
		store.NetworkChannelPatch{
			CoordinatorPeerID: &coordinatorPeerID,
			UpdatedAt:         recordedAt.Add(time.Minute),
		},
	)
	if err == nil || !strings.Contains(err.Error(), "coordinator_peer_id is required") {
		t.Fatalf("PatchNetworkChannel(clear coordinator) error = %v, want coordinator peer validation", err)
	}
	entry, err := globalDB.GetNetworkChannel(testutil.Context(t), store.NetworkChannelRef{
		WorkspaceID: workspaceID,
		Channel:     "coord.core",
	})
	if err != nil {
		t.Fatalf("GetNetworkChannel(after invalid patch) error = %v", err)
	}
	if entry.FanoutPolicy != store.NetworkFanoutPolicyCoordinator ||
		entry.CoordinatorPeerID != "reviewer.sess-a" {
		t.Fatalf("entry after invalid patch = %#v, want original fanout coupling", entry)
	}
}

func assertGlobalDBGetNetworkChannelNotFound(t *testing.T) {
	t.Helper()

	globalDB := openTestGlobalDB(t)
	_, err := globalDB.GetNetworkChannel(testutil.Context(t), store.NetworkChannelRef{
		WorkspaceID: networkStoreTestWorkspaceID,
		Channel:     "missing",
	})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("GetNetworkChannel(missing) error = %v, want sql.ErrNoRows", err)
	}
}

func assertGlobalDBDeleteNetworkChannel(t *testing.T) {
	t.Helper()

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
	if err := globalDB.DeleteNetworkChannel(testutil.Context(t), store.NetworkChannelRef{
		WorkspaceID: workspaceID,
		Channel:     "coord.core",
	}); err != nil {
		t.Fatalf("DeleteNetworkChannel() error = %v", err)
	}
	if _, err := globalDB.GetNetworkChannel(testutil.Context(t), store.NetworkChannelRef{
		WorkspaceID: workspaceID,
		Channel:     "coord.core",
	}); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("GetNetworkChannel(after delete) error = %v, want sql.ErrNoRows", err)
	}
}

func assertGlobalDBDeleteWorkspaceCascadesNetworkChannels(t *testing.T) {
	t.Helper()

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
	if _, err := globalDB.GetNetworkChannel(testutil.Context(t), store.NetworkChannelRef{
		WorkspaceID: workspaceID,
		Channel:     "coord.core",
	}); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("GetNetworkChannel(after workspace delete) error = %v, want sql.ErrNoRows", err)
	}
}

func assertGlobalDBListNetworkChannelsWrapsTimestampParseFailures(t *testing.T) {
	t.Helper()

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

	_, err := globalDB.ListNetworkChannels(testutil.Context(t), store.NetworkChannelQuery{
		WorkspaceID: workspaceID,
	})
	if err == nil {
		t.Fatal("ListNetworkChannels(invalid timestamp) error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "parse network channel created_at") {
		t.Fatalf("ListNetworkChannels(invalid timestamp) error = %v, want wrapped timestamp parse context", err)
	}
}
