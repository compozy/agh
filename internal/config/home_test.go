package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveHomeDirUsesAGHHomeOverride(t *testing.T) {
	override := filepath.Join(t.TempDir(), "custom-home")
	t.Setenv("AGH_HOME", override)

	got, err := ResolveHomeDir()
	if err != nil {
		t.Fatalf("ResolveHomeDir() error = %v", err)
	}
	if got != override {
		t.Fatalf("ResolveHomeDir() = %q, want %q", got, override)
	}
}

func TestEnsureHomeLayoutCreatesRequiredDirectories(t *testing.T) {
	paths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}

	if err := EnsureHomeLayout(paths); err != nil {
		t.Fatalf("EnsureHomeLayout() error = %v", err)
	}

	for _, dir := range []string{paths.HomeDir, paths.AgentsDir, paths.MemoryDir, paths.SessionsDir, paths.LogsDir} {
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("Stat(%q) error = %v", dir, err)
		}
		if !info.IsDir() {
			t.Fatalf("Stat(%q) IsDir() = false, want true", dir)
		}
	}
}

func TestResolveHomePathsFromExpandsTildePaths(t *testing.T) {
	userHome, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir() error = %v", err)
	}

	paths, err := ResolveHomePathsFrom("~/agh-test-home")
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	if paths.HomeDir != filepath.Join(userHome, "agh-test-home") {
		t.Fatalf("ResolveHomePathsFrom() HomeDir = %q, want %q", paths.HomeDir, filepath.Join(userHome, "agh-test-home"))
	}
	if got, want := paths.MemoryDir, filepath.Join(userHome, "agh-test-home", MemoryDirName); got != want {
		t.Fatalf("ResolveHomePathsFrom() MemoryDir = %q, want %q", got, want)
	}
}

func TestEnsureHomeLayoutRejectsEmptyPaths(t *testing.T) {
	if err := EnsureHomeLayout(HomePaths{}); err == nil {
		t.Fatal("EnsureHomeLayout() error = nil, want non-nil")
	}
}
