package globaldb

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	bundlemodel "github.com/pedronauck/agh/internal/bundles/model"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/testutil"
)

func TestOpenGlobalDBCreatesBundleActivationTableWithExpectedColumns(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)

	assertTableColumns(t, globalDB.db, "bundle_activations", []string{
		"id",
		"extension_name",
		"bundle_name",
		"profile_name",
		"scope",
		"workspace_id",
		"spec_content_hash",
		"bind_primary_channel_default",
		"created_at",
		"updated_at",
	})
}

func TestOpenGlobalDBMigratesLegacyBundleActivationSpecHashColumn(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), GlobalDatabaseName)
	db, err := sql.Open(sqliteDriverName, sqliteDSN(path))
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	if _, err := db.ExecContext(testutil.Context(t), `CREATE TABLE bundle_activations (
		id                           TEXT PRIMARY KEY,
		extension_name               TEXT NOT NULL,
		bundle_name                  TEXT NOT NULL,
		profile_name                 TEXT NOT NULL,
		scope                        TEXT NOT NULL,
		workspace_id                 TEXT,
		bind_primary_channel_default BOOLEAN NOT NULL DEFAULT 0,
		created_at                   TEXT NOT NULL,
		updated_at                   TEXT NOT NULL
	)`); err != nil {
		t.Fatalf("ExecContext(create legacy bundle_activations) error = %v", err)
	}
	legacyCreatedAt := time.Date(2026, 4, 14, 20, 0, 0, 0, time.UTC)
	if _, err := db.ExecContext(
		testutil.Context(t),
		`INSERT INTO bundle_activations (
			id, extension_name, bundle_name, profile_name, scope, workspace_id, bind_primary_channel_default, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"act-legacy",
		"marketing-team",
		"marketing",
		"default",
		string(bundlemodel.ScopeGlobal),
		nil,
		true,
		store.FormatTimestamp(legacyCreatedAt),
		store.FormatTimestamp(legacyCreatedAt),
	); err != nil {
		t.Fatalf("ExecContext(insert legacy bundle activation) error = %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("db.Close() error = %v", err)
	}

	globalDB, err := OpenGlobalDB(testutil.Context(t), path)
	if err != nil {
		t.Fatalf("OpenGlobalDB() error = %v", err)
	}
	t.Cleanup(func() {
		if err := globalDB.Close(testutil.Context(t)); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})

	assertTableColumns(t, globalDB.db, "bundle_activations", []string{
		"id",
		"extension_name",
		"bundle_name",
		"profile_name",
		"scope",
		"workspace_id",
		"bind_primary_channel_default",
		"created_at",
		"updated_at",
		"spec_content_hash",
	})

	loaded, err := globalDB.GetBundleActivation(testutil.Context(t), "act-legacy")
	if err != nil {
		t.Fatalf("GetBundleActivation() error = %v", err)
	}
	if got, want := loaded.ExtensionName, "marketing-team"; got != want {
		t.Fatalf("loaded.ExtensionName = %q, want %q", got, want)
	}
	if got, want := loaded.BundleName, "marketing"; got != want {
		t.Fatalf("loaded.BundleName = %q, want %q", got, want)
	}
	if got, want := loaded.ProfileName, "default"; got != want {
		t.Fatalf("loaded.ProfileName = %q, want %q", got, want)
	}
	if got, want := loaded.Scope, bundlemodel.ScopeGlobal; got != want {
		t.Fatalf("loaded.Scope = %q, want %q", got, want)
	}
	if got := loaded.SpecContentHash; got != "" {
		t.Fatalf("loaded.SpecContentHash = %q, want empty for migrated legacy row", got)
	}
}

func TestGlobalDBBundleActivationRoundTripWithSpecHashAndInventory(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	now := time.Date(2026, 4, 14, 23, 0, 0, 0, time.UTC)
	if _, err := globalDB.db.ExecContext(
		testutil.Context(t),
		`INSERT INTO extensions (name, version, source, enabled, manifest_path, installed_at, capabilities, actions, checksum)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"marketing-team",
		"1.0.0",
		"managed",
		true,
		"/tmp/marketing-team/extension.toml",
		store.FormatTimestamp(now),
		"{}",
		"{}",
		"checksum",
	); err != nil {
		t.Fatalf("insert extension row error = %v", err)
	}

	workspaceID := registerWorkspaceForGlobalTests(t, globalDB, "bundle-workspace", filepath.Join(t.TempDir(), "bundle-workspace"))
	activation := bundlemodel.Activation{
		ID:                          "act_marketing",
		ExtensionName:               "marketing-team",
		BundleName:                  "marketing",
		ProfileName:                 "default",
		Scope:                       bundlemodel.ScopeWorkspace,
		WorkspaceID:                 workspaceID,
		SpecContentHash:             "abc123",
		BindPrimaryChannelAsDefault: true,
		CreatedAt:                   now,
		UpdatedAt:                   now,
	}
	if err := globalDB.CreateBundleActivation(testutil.Context(t), activation); err != nil {
		t.Fatalf("CreateBundleActivation() error = %v", err)
	}

	loaded, err := globalDB.GetBundleActivation(testutil.Context(t), activation.ID)
	if err != nil {
		t.Fatalf("GetBundleActivation() error = %v", err)
	}
	if loaded.SpecContentHash != activation.SpecContentHash {
		t.Fatalf("SpecContentHash = %q, want %q", loaded.SpecContentHash, activation.SpecContentHash)
	}
	if loaded.WorkspaceID != workspaceID {
		t.Fatalf("WorkspaceID = %q, want %q", loaded.WorkspaceID, workspaceID)
	}

	activation.BindPrimaryChannelAsDefault = false
	activation.SpecContentHash = "def456"
	activation.UpdatedAt = now.Add(time.Minute)
	if err := globalDB.UpdateBundleActivation(testutil.Context(t), activation); err != nil {
		t.Fatalf("UpdateBundleActivation() error = %v", err)
	}

	listed, err := globalDB.ListBundleActivations(testutil.Context(t))
	if err != nil {
		t.Fatalf("ListBundleActivations() error = %v", err)
	}
	if got, want := len(listed), 1; got != want {
		t.Fatalf("len(ListBundleActivations()) = %d, want %d", got, want)
	}
	if listed[0].SpecContentHash != "def456" {
		t.Fatalf("listed[0].SpecContentHash = %q, want def456", listed[0].SpecContentHash)
	}

	inventory := []bundlemodel.InventoryItem{{
		ActivationID:  activation.ID,
		ResourceKind:  "bridge_instance",
		ResourceID:    "bri_123",
		ResourceName:  "Marketing Telegram",
		RecordedAtUTC: now,
	}}
	if err := globalDB.ReplaceBundleActivationInventory(testutil.Context(t), activation.ID, inventory); err != nil {
		t.Fatalf("ReplaceBundleActivationInventory() error = %v", err)
	}

	loadedInventory, err := globalDB.ListBundleActivationInventory(testutil.Context(t), activation.ID)
	if err != nil {
		t.Fatalf("ListBundleActivationInventory() error = %v", err)
	}
	if got, want := len(loadedInventory), 1; got != want {
		t.Fatalf("len(ListBundleActivationInventory()) = %d, want %d", got, want)
	}
	if loadedInventory[0].ResourceID != inventory[0].ResourceID {
		t.Fatalf("inventory resource id = %q, want %q", loadedInventory[0].ResourceID, inventory[0].ResourceID)
	}

	if err := globalDB.DeleteBundleActivation(testutil.Context(t), activation.ID); err != nil {
		t.Fatalf("DeleteBundleActivation() error = %v", err)
	}
	if _, err := globalDB.GetBundleActivation(testutil.Context(t), activation.ID); err == nil {
		t.Fatal("GetBundleActivation() after delete error = nil, want not found")
	}

	remainingInventory, err := globalDB.ListBundleActivationInventory(testutil.Context(t), activation.ID)
	if err != nil {
		t.Fatalf("ListBundleActivationInventory(after delete) error = %v", err)
	}
	if got := len(remainingInventory); got != 0 {
		t.Fatalf("len(ListBundleActivationInventory(after delete)) = %d, want 0", got)
	}
}

func TestGlobalDBBundleActivationCountByExtension(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	now := time.Date(2026, 4, 14, 23, 30, 0, 0, time.UTC)
	if _, err := globalDB.db.ExecContext(
		testutil.Context(t),
		`INSERT INTO extensions (name, version, source, enabled, manifest_path, installed_at, capabilities, actions, checksum)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"marketing-team",
		"1.0.0",
		"managed",
		true,
		"/tmp/marketing-team/extension.toml",
		store.FormatTimestamp(now),
		"{}",
		"{}",
		"checksum",
	); err != nil {
		t.Fatalf("insert extension row error = %v", err)
	}

	activation := bundlemodel.Activation{
		ID:              "act_count",
		ExtensionName:   "marketing-team",
		BundleName:      "marketing",
		ProfileName:     "default",
		Scope:           bundlemodel.ScopeGlobal,
		SpecContentHash: "hash",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := globalDB.CreateBundleActivation(testutil.Context(t), activation); err != nil {
		t.Fatalf("CreateBundleActivation() error = %v", err)
	}

	count, err := globalDB.CountBundleActivationsForExtension(testutil.Context(t), "marketing-team")
	if err != nil {
		t.Fatalf("CountBundleActivationsForExtension() error = %v", err)
	}
	if got, want := count, 1; got != want {
		t.Fatalf("CountBundleActivationsForExtension() = %d, want %d", got, want)
	}
}
