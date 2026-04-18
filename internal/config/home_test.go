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

	for _, dir := range []string{
		paths.HomeDir,
		paths.AgentsDir,
		paths.SkillsDir,
		paths.MemoryDir,
		paths.SessionsDir,
		paths.RestartsDir,
		paths.LogsDir,
	} {
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
		t.Fatalf(
			"ResolveHomePathsFrom() HomeDir = %q, want %q",
			paths.HomeDir,
			filepath.Join(userHome, "agh-test-home"),
		)
	}
	if got, want := paths.MemoryDir, filepath.Join(userHome, "agh-test-home", MemoryDirName); got != want {
		t.Fatalf("ResolveHomePathsFrom() MemoryDir = %q, want %q", got, want)
	}
	if got, want := paths.SkillsDir, filepath.Join(userHome, "agh-test-home", SkillsDirName); got != want {
		t.Fatalf("ResolveHomePathsFrom() SkillsDir = %q, want %q", got, want)
	}
	if got, want := paths.RestartsDir, filepath.Join(userHome, "agh-test-home", RestartsDirName); got != want {
		t.Fatalf("ResolveHomePathsFrom() RestartsDir = %q, want %q", got, want)
	}
	if got, want := paths.NetworkAuditFile, filepath.Join(
		userHome,
		"agh-test-home",
		LogsDirName,
		NetworkAuditFileName,
	); got != want {
		t.Fatalf("ResolveHomePathsFrom() NetworkAuditFile = %q, want %q", got, want)
	}
}

func TestResolvePathVariants(t *testing.T) {
	if got, err := ResolvePath(""); err != nil || got != "" {
		t.Fatalf("ResolvePath(blank) = %q, %v, want empty nil", got, err)
	}

	got, err := ResolvePath("daemon.sock")
	if err != nil {
		t.Fatalf("ResolvePath(relative) error = %v", err)
	}
	if !filepath.IsAbs(got) {
		t.Fatalf("ResolvePath(relative) = %q, want absolute path", got)
	}
}

func TestResolveUserAgentsSkillsDirUsesHOMEOverride(t *testing.T) {
	home := filepath.Join(t.TempDir(), "custom-home")

	got, err := ResolveUserAgentsSkillsDir(func(key string) string {
		if key == "HOME" {
			return home
		}
		return ""
	})
	if err != nil {
		t.Fatalf("ResolveUserAgentsSkillsDir(HOME) error = %v", err)
	}

	if want := filepath.Join(home, ".agents", "skills"); got != want {
		t.Fatalf("ResolveUserAgentsSkillsDir(HOME) = %q, want %q", got, want)
	}
}

func TestResolveUserAgentsSkillsDirFallsBackToUserHome(t *testing.T) {
	got, err := ResolveUserAgentsSkillsDir(func(string) string { return "" })
	if err != nil {
		t.Fatalf("ResolveUserAgentsSkillsDir(fallback) error = %v", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("os.UserHomeDir() error = %v", err)
	}
	absHome, err := filepath.Abs(home)
	if err != nil {
		t.Fatalf("filepath.Abs(%q) error = %v", home, err)
	}
	if want := filepath.Join(absHome, ".agents", "skills"); got != want {
		t.Fatalf("ResolveUserAgentsSkillsDir(fallback) = %q, want %q", got, want)
	}
}

func TestEnsureHomeLayoutRejectsEmptyPaths(t *testing.T) {
	if err := EnsureHomeLayout(HomePaths{}); err == nil {
		t.Fatal("EnsureHomeLayout() error = nil, want non-nil")
	}
}
