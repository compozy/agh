package extension

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/testutil"
)

func TestRegistryBlocksDisableAndUninstallWithActiveBundles(t *testing.T) {
	t.Parallel()

	env := newRegistryTestEnvWithBundleActivations(t)
	dir, manifest, checksum := createRegistryTestExtension(t, "bundle-guard", registryManifestOptions{})
	if err := env.registry.Install(manifest, dir, checksum); err != nil {
		t.Fatalf("Install() error = %v", err)
	}

	if _, err := env.db.Exec(
		`INSERT INTO bundle_activations (
			id, extension_name, bundle_name, profile_name, scope, workspace_id, spec_content_hash, bind_primary_channel_default, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"act_guard",
		manifest.Name,
		"bundle",
		"default",
		"global",
		nil,
		"hash",
		false,
		store.FormatTimestamp(env.installedAt),
		store.FormatTimestamp(env.installedAt),
	); err != nil {
		t.Fatalf("insert bundle activation error = %v", err)
	}

	if err := env.registry.Disable(manifest.Name); !errors.Is(err, ErrExtensionHasActiveBundles) {
		t.Fatalf("Disable() error = %v, want ErrExtensionHasActiveBundles", err)
	}
	if err := env.registry.Uninstall(manifest.Name); !errors.Is(err, ErrExtensionHasActiveBundles) {
		t.Fatalf("Uninstall() error = %v, want ErrExtensionHasActiveBundles", err)
	}
}

func newRegistryTestEnvWithBundleActivations(t *testing.T) registryTestEnv {
	t.Helper()

	dbPath := t.TempDir() + "/agh-registry.db"
	db, err := store.OpenSQLiteDatabase(testutil.Context(t), dbPath, func(ctx context.Context, db *sql.DB) error {
		return store.EnsureSchema(ctx, db, []string{
			registryTestExtensionsTableSchema,
			`CREATE TABLE IF NOT EXISTS bundle_activations (
				id                           TEXT PRIMARY KEY,
				extension_name               TEXT NOT NULL,
				bundle_name                  TEXT NOT NULL,
				profile_name                 TEXT NOT NULL,
				scope                        TEXT NOT NULL,
				workspace_id                 TEXT,
				spec_content_hash            TEXT,
				bind_primary_channel_default BOOLEAN NOT NULL DEFAULT 0,
				created_at                   TEXT NOT NULL,
				updated_at                   TEXT NOT NULL
			);`,
		})
	})
	if err != nil {
		t.Fatalf("OpenSQLiteDatabase() error = %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("db.Close() error = %v", err)
		}
	})

	installedAt := time.Date(2026, 4, 10, 15, 30, 0, 0, time.UTC)
	registry := NewRegistry(db)
	registry.now = func() time.Time { return installedAt }
	return registryTestEnv{
		db:          db,
		registry:    registry,
		installedAt: installedAt,
	}
}
