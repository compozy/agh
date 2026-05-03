package procutil

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestFilteredDaemonEnvRemovesCredentialShapedVariables(t *testing.T) {
	t.Parallel()

	t.Run("Should keep operational variables and drop secrets", func(t *testing.T) {
		t.Parallel()

		env := FilteredDaemonEnv([]string{
			"PATH=/usr/bin",
			"HOME=/home/agh",
			"AGH_HOME=/tmp/agh",
			"PROVIDER_CODEX_HOME=/tmp/provider",
			"OPENAI_API_KEY=sk-secret",
			"GITHUB_TOKEN=ghp-secret",
			"SESSION_MANAGER=local/session",
			"CLIENT_SECRET=client-secret",
			"MALFORMED",
		})

		for _, leaked := range []string{
			"OPENAI_API_KEY=sk-secret",
			"GITHUB_TOKEN=ghp-secret",
			"SESSION_MANAGER=local/session",
			"CLIENT_SECRET=client-secret",
			"MALFORMED",
		} {
			if containsEnvEntry(env, leaked) {
				t.Fatalf("FilteredDaemonEnv() leaked %q in %#v", leaked, env)
			}
		}
		for _, kept := range []string{
			"PATH=/usr/bin",
			"HOME=/home/agh",
			"AGH_HOME=/tmp/agh",
			"PROVIDER_CODEX_HOME=/tmp/provider",
		} {
			if !containsEnvEntry(env, kept) {
				t.Fatalf("FilteredDaemonEnv() missing %q in %#v", kept, env)
			}
		}
	})
}

func TestLaunchSandboxFiltersFallbackEnvironment(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-launch-secret")
	t.Setenv("AGH_HOME", "/tmp/agh")

	t.Run("Should filter inherited daemon secrets", func(t *testing.T) {
		env := launchSandbox(nil)
		if hasEnvPrefix(env, "OPENAI_API_KEY=") {
			t.Fatalf("launchSandbox(nil) leaked OPENAI_API_KEY in %#v", env)
		}
		if !hasEnvPrefix(env, "AGH_HOME=/tmp/agh") {
			t.Fatalf("launchSandbox(nil) missing AGH_HOME in %#v", env)
		}
	})
}

func TestAttachCommandLogRedactsRecentError(t *testing.T) {
	t.Parallel()

	t.Run("Should redact token shaped stderr before wrapping", func(t *testing.T) {
		t.Parallel()

		logPath := filepath.Join(t.TempDir(), "command.log")
		if err := os.WriteFile(logPath, []byte("info\nerror: token=super-secret\n"), 0o600); err != nil {
			t.Fatalf("os.WriteFile(%q) error = %v", logPath, err)
		}

		err := attachCommandLog(errors.New("launch failed"), logPath, 0)
		if err == nil {
			t.Fatal("attachCommandLog() error = nil, want wrapped error")
		}
		if strings.Contains(err.Error(), "super-secret") {
			t.Fatalf("attachCommandLog() = %v, want redacted secret", err)
		}
		if !strings.Contains(err.Error(), "token=[REDACTED]") {
			t.Fatalf("attachCommandLog() = %v, want redacted token marker", err)
		}
	})
}

func containsEnvEntry(env []string, target string) bool {
	return slices.Contains(env, target)
}

func hasEnvPrefix(env []string, prefix string) bool {
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			return true
		}
	}
	return false
}
