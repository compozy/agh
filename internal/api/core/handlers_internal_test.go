package core

import (
	"errors"
	"path/filepath"
	"testing"

	aghconfig "github.com/pedronauck/agh/internal/config"
)

func TestResolveUserHomeDir(t *testing.T) {
	t.Run("ShouldPreferResolvedLookupValue", func(t *testing.T) {
		t.Parallel()

		want := filepath.Join(t.TempDir(), "user-home")
		homePaths, err := aghconfig.ResolveHomePathsFrom(filepath.Join(t.TempDir(), aghconfig.DirName))
		if err != nil {
			t.Fatalf("ResolveHomePathsFrom() error = %v", err)
		}

		got, err := resolveUserHomeDir(homePaths, func() (string, error) {
			return want, nil
		})
		if err != nil {
			t.Fatalf("resolveUserHomeDir() error = %v", err)
		}
		if got != want {
			t.Fatalf("resolveUserHomeDir() = %q, want %q", got, want)
		}
	})

	t.Run("ShouldFallbackToCanonicalAGHHomeParentWhenLookupFails", func(t *testing.T) {
		t.Parallel()

		aghHome := filepath.Join(t.TempDir(), aghconfig.DirName)
		homePaths, err := aghconfig.ResolveHomePathsFrom(aghHome)
		if err != nil {
			t.Fatalf("ResolveHomePathsFrom() error = %v", err)
		}

		got, err := resolveUserHomeDir(homePaths, func() (string, error) {
			return "", errors.New("boom")
		})
		if err != nil {
			t.Fatalf("resolveUserHomeDir() error = %v, want nil", err)
		}

		want := filepath.Dir(homePaths.HomeDir)
		if got != want {
			t.Fatalf("resolveUserHomeDir() = %q, want %q", got, want)
		}
	})

	t.Run("ShouldReturnErrorWhenLookupFailsAndFallbackIsUnavailable", func(t *testing.T) {
		t.Parallel()

		homePaths, err := aghconfig.ResolveHomePathsFrom(filepath.Join(t.TempDir(), "agh-home"))
		if err != nil {
			t.Fatalf("ResolveHomePathsFrom() error = %v", err)
		}

		got, err := resolveUserHomeDir(homePaths, func() (string, error) {
			return "", errors.New("boom")
		})
		if err == nil {
			t.Fatal("resolveUserHomeDir() error = nil, want non-nil")
		}
		if got != "" {
			t.Fatalf("resolveUserHomeDir() = %q, want empty string", got)
		}
	})
}
