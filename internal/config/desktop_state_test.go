package config

// Invariant: desktop-state limits default, overlay, and reject out-of-range values by canonical path.
// Owning layer: config loading and validation. Canonical suite: desktop_state_test.go.

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestDesktopStateConfigShouldLoadDefaultsAndOverlays(t *testing.T) {
	t.Parallel()

	t.Run("Should load desktop-state defaults (UT-020)", func(t *testing.T) {
		t.Parallel()
		homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
		if err != nil {
			t.Fatalf("ResolveHomePathsFrom() error = %v", err)
		}
		cfg := DefaultWithHome(homePaths)
		if cfg.DesktopState.MaxValueKiB != 256 || cfg.DesktopState.MaxKeysPerWorkspace != 512 {
			t.Fatalf("DesktopState = %#v, want 256/512", cfg.DesktopState)
		}
	})

	t.Run("Should apply a global desktop-state overlay (UT-020)", func(t *testing.T) {
		t.Parallel()
		homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
		if err != nil {
			t.Fatalf("ResolveHomePathsFrom() error = %v", err)
		}
		writeFile(t, homePaths.ConfigFile, `
[desktop_state]
max_value_kib = 128
max_keys_per_workspace = 256
`)
		cfg, err := LoadForHome(homePaths)
		if err != nil {
			t.Fatalf("LoadForHome() error = %v", err)
		}
		if cfg.DesktopState.MaxValueKiB != 128 || cfg.DesktopState.MaxKeysPerWorkspace != 256 {
			t.Fatalf("DesktopState = %#v, want 128/256", cfg.DesktopState)
		}
	})

	t.Run("Should reject workspace desktop-state overlays as global-only (UT-020)", func(t *testing.T) {
		t.Parallel()
		homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
		if err != nil {
			t.Fatalf("ResolveHomePathsFrom() error = %v", err)
		}
		workspaceRoot := t.TempDir()
		writeFile(t, filepath.Join(workspaceRoot, DirName, ConfigName), `
[desktop_state]
max_value_kib = 64
`)
		_, err = LoadForHome(homePaths, WithWorkspaceRoot(workspaceRoot))
		if err == nil || !strings.Contains(err.Error(), "desktop-state limits are global-only") {
			t.Fatalf("LoadForHome() error = %v, want global-only rejection", err)
		}
	})

	t.Run("Should load the canonical config example (UT-020)", func(t *testing.T) {
		t.Parallel()
		examplePath := filepath.Join("..", "..", "config.toml")
		cfg := DefaultWithHome(HomePaths{})
		if err := ApplyConfigOverlayFile(examplePath, &cfg); err != nil {
			t.Fatalf("ApplyConfigOverlayFile(config.toml) error = %v", err)
		}
		if err := cfg.DesktopState.Validate(); err != nil {
			t.Fatalf("DesktopState.Validate() error = %v", err)
		}
		if cfg.DesktopState.MaxValueKiB != 256 || cfg.DesktopState.MaxKeysPerWorkspace != 512 {
			t.Fatalf("config.toml DesktopState = %#v, want 256/512", cfg.DesktopState)
		}
	})
}

func TestDesktopStateConfigShouldRejectOutOfRangeValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		config   DesktopStateConfig
		wantPath string
	}{
		{
			name:     "Should reject zero max value KiB (UT-020)",
			config:   DesktopStateConfig{MaxValueKiB: 0, MaxKeysPerWorkspace: 512},
			wantPath: desktopStateMaxValueKiBPath,
		},
		{
			name:     "Should reject max value KiB above range (UT-020)",
			config:   DesktopStateConfig{MaxValueKiB: 5000, MaxKeysPerWorkspace: 512},
			wantPath: desktopStateMaxValueKiBPath,
		},
		{
			name:     "Should reject max keys below range (UT-020)",
			config:   DesktopStateConfig{MaxValueKiB: 256, MaxKeysPerWorkspace: 15},
			wantPath: desktopStateMaxKeysPerWorkspacePath,
		},
		{
			name:     "Should reject max keys above range (UT-020)",
			config:   DesktopStateConfig{MaxValueKiB: 256, MaxKeysPerWorkspace: 8193},
			wantPath: desktopStateMaxKeysPerWorkspacePath,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			err := test.config.Validate()
			if err == nil {
				t.Fatal("Validate() error = nil, want validation error")
			}
			var validationError ValidationError
			if !errors.As(err, &validationError) {
				t.Fatalf("Validate() error = %T %v, want ValidationError", err, err)
			}
			if validationError.Path != test.wantPath || !strings.Contains(err.Error(), test.wantPath) {
				t.Fatalf("Validate() error = %v path=%q, want %q", err, validationError.Path, test.wantPath)
			}
		})
	}
}
