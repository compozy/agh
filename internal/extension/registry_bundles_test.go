package extension

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	bridgepkg "github.com/pedronauck/agh/internal/bridges"
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

func TestLoadBundleSpecsRejectsCaseInsensitiveDuplicateBundleNames(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	bundlesDir := filepath.Join(rootDir, "bundles")
	if err := os.MkdirAll(bundlesDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(bundlesDir, "alpha.json"), []byte(`{
		"name": "Marketing",
		"profiles": [{
			"name": "default",
			"channels": {
				"primary": "ops",
				"items": [{"name": "ops"}]
			}
		}]
	}`), 0o644); err != nil {
		t.Fatalf("os.WriteFile(alpha.json) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(bundlesDir, "beta.json"), []byte(`{
		"name": "marketing",
		"profiles": [{
			"name": "default",
			"channels": {
				"primary": "ops",
				"items": [{"name": "ops"}]
			}
		}]
	}`), 0o644); err != nil {
		t.Fatalf("os.WriteFile(beta.json) error = %v", err)
	}

	_, err := LoadBundleSpecs(rootDir, &Manifest{
		Name: "bundle-guard",
		Resources: ResourcesConfig{
			Bundles: []string{"bundles"},
		},
	})
	if !errors.Is(err, ErrBundleInvalid) {
		t.Fatalf("LoadBundleSpecs() error = %v, want ErrBundleInvalid", err)
	}
}

func TestBundleSpecValidateRejectsCaseInsensitiveDuplicateProfilesAndInvalidDeliveryDefaults(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		spec BundleSpec
	}{
		{
			name: "Should reject case-insensitive duplicate profile names",
			spec: BundleSpec{
				Name: "marketing",
				Profiles: []BundleProfile{
					{
						Name: "Default",
						Channels: BundleChannelsConfig{
							Primary: "ops",
							Items:   []BundleChannel{{Name: "ops"}},
						},
					},
					{
						Name: "default",
						Channels: BundleChannelsConfig{
							Primary: "ops-2",
							Items:   []BundleChannel{{Name: "ops-2"}},
						},
					},
				},
			},
		},
		{
			name: "Should reject invalid bridge delivery default JSON",
			spec: BundleSpec{
				Name: "marketing",
				Profiles: []BundleProfile{{
					Name: "default",
					Channels: BundleChannelsConfig{
						Primary: "ops",
						Items:   []BundleChannel{{Name: "ops"}},
					},
					Bridges: []BundleBridgePreset{{
						Name:             "telegram-main",
						Platform:         "telegram",
						DisplayName:      "Marketing Bridge",
						RoutingPolicy:    bridgepkg.RoutingPolicy{IncludePeer: true},
						DeliveryDefaults: []byte(`{invalid`),
					}},
				}},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.spec.Validate(&Manifest{
				Name: "bundle-guard",
				Bridge: BridgeConfig{
					Platform:    "telegram",
					DisplayName: "Telegram",
				},
				Capabilities: CapabilitiesConfig{
					Provides: []string{"bridge.adapter"},
				},
			})
			if !errors.Is(err, ErrBundleInvalid) {
				t.Fatalf("BundleSpec.Validate() error = %v, want ErrBundleInvalid", err)
			}
		})
	}
}
