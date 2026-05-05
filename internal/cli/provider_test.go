package cli

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	aghconfig "github.com/pedronauck/agh/internal/config"
)

func TestProviderAuthStatusCommand(t *testing.T) {
	t.Parallel()

	t.Run("Should report native CLI auth without requiring credentials", func(t *testing.T) {
		t.Parallel()

		deps := newTestDeps(t, nil)
		stdout, _, err := executeRootCommand(t, deps, "provider", "auth", "status", "claude", "-o", "json")
		if err != nil {
			t.Fatalf("provider auth status error = %v", err)
		}

		var record providerAuthStatusRecord
		if err := json.Unmarshal([]byte(stdout), &record); err != nil {
			t.Fatalf("json.Unmarshal(provider auth status) error = %v", err)
		}
		if got, want := record.Provider, "claude"; got != want {
			t.Fatalf("Provider = %q, want %q", got, want)
		}
		if got, want := record.AuthMode, "native_cli"; got != want {
			t.Fatalf("AuthMode = %q, want %q", got, want)
		}
		if len(record.Credentials) != 0 {
			t.Fatalf("Credentials = %#v, want none for native CLI provider", record.Credentials)
		}
	})

	t.Run("Should report missing required bound secret credentials", func(t *testing.T) {
		t.Parallel()

		deps := newTestDeps(t, nil)
		deps.loadConfig = func() (aghconfig.Config, error) {
			cfg := aghconfig.DefaultWithHome(mustTestHomePaths(t))
			cfg.Providers["custom"] = aghconfig.ProviderConfig{
				Command:  "custom-agent --acp",
				AuthMode: aghconfig.ProviderAuthModeBoundSecret,
				CredentialSlots: []aghconfig.ProviderCredentialSlot{
					{
						Name:      "api_key",
						TargetEnv: "CUSTOM_API_KEY",
						SecretRef: "env:CUSTOM_API_KEY",
						Kind:      "api_key",
						Required:  true,
					},
				},
			}
			return cfg, nil
		}

		stdout, _, err := executeRootCommand(t, deps, "provider", "auth", "status", "custom", "-o", "json")
		if err != nil {
			t.Fatalf("provider auth status error = %v", err)
		}

		var record providerAuthStatusRecord
		if err := json.Unmarshal([]byte(stdout), &record); err != nil {
			t.Fatalf("json.Unmarshal(provider auth status) error = %v", err)
		}
		if got, want := record.State, "missing_required"; got != want {
			t.Fatalf("State = %q, want %q", got, want)
		}
		if len(record.Credentials) != 1 || record.Credentials[0].Present {
			t.Fatalf("Credentials = %#v, want one missing credential", record.Credentials)
		}
	})

	t.Run("Should run configured native status command", func(t *testing.T) {
		t.Parallel()

		deps := newTestDeps(t, nil)
		deps.loadConfig = func() (aghconfig.Config, error) {
			cfg := aghconfig.DefaultWithHome(mustTestHomePaths(t))
			cfg.Providers["claude"] = aghconfig.ProviderConfig{
				AuthStatusCmd: "claude auth status",
			}
			return cfg, nil
		}
		deps.runProviderAuthCommand = func(
			_ context.Context,
			spec providerAuthCommandSpec,
		) (providerAuthCommandResult, error) {
			if !strings.Contains(spec.Command, "claude auth status") {
				t.Fatalf("Command = %q, want claude status command", spec.Command)
			}
			if providerTestEnvValue(spec.Env, "AGH_PROVIDER_AUTH_MODE") != "native_cli" {
				t.Fatalf("AGH_PROVIDER_AUTH_MODE missing from provider auth command env: %#v", spec.Env)
			}
			return providerAuthCommandResult{ExitCode: 0, Stdout: "logged in"}, nil
		}

		stdout, _, err := executeRootCommand(t, deps, "provider", "auth", "status", "claude", "-o", "json")
		if err != nil {
			t.Fatalf("provider auth status error = %v", err)
		}

		var record providerAuthStatusRecord
		if err := json.Unmarshal([]byte(stdout), &record); err != nil {
			t.Fatalf("json.Unmarshal(provider auth status) error = %v", err)
		}
		if got, want := record.State, "authenticated"; got != want {
			t.Fatalf("State = %q, want %q", got, want)
		}
		if record.Probe == nil || record.Probe.Stdout != "logged in" {
			t.Fatalf("Probe = %#v, want status command result", record.Probe)
		}
	})
}

func TestProviderAuthLoginCommand(t *testing.T) {
	t.Parallel()

	t.Run("Should run configured login command", func(t *testing.T) {
		t.Parallel()

		deps := newTestDeps(t, nil)
		deps.loadConfig = func() (aghconfig.Config, error) {
			cfg := aghconfig.DefaultWithHome(mustTestHomePaths(t))
			cfg.Providers["codex"] = aghconfig.ProviderConfig{
				AuthLoginCmd: "codex login",
			}
			return cfg, nil
		}
		deps.runProviderAuthCommand = func(
			_ context.Context,
			spec providerAuthCommandSpec,
		) (providerAuthCommandResult, error) {
			if spec.Command != "codex login" {
				t.Fatalf("Command = %q, want codex login", spec.Command)
			}
			return providerAuthCommandResult{ExitCode: 0, Stdout: "ok"}, nil
		}

		stdout, _, err := executeRootCommand(t, deps, "provider", "auth", "login", "codex", "-o", "json")
		if err != nil {
			t.Fatalf("provider auth login error = %v", err)
		}

		var record providerAuthStatusRecord
		if err := json.Unmarshal([]byte(stdout), &record); err != nil {
			t.Fatalf("json.Unmarshal(provider auth login) error = %v", err)
		}
		if got, want := record.State, "login_completed"; got != want {
			t.Fatalf("State = %q, want %q", got, want)
		}
	})

	t.Run("Should run builtin Pi login against the isolated Pi auth directory", func(t *testing.T) {
		t.Parallel()

		homePaths := mustTestHomePaths(t)
		deps := newTestDeps(t, nil)
		deps.resolveHome = func() (aghconfig.HomePaths, error) {
			return homePaths, nil
		}
		deps.loadConfig = func() (aghconfig.Config, error) {
			cfg := aghconfig.DefaultWithHome(homePaths)
			cfg.Providers["pi"] = aghconfig.ProviderConfig{
				EnvPolicy:  aghconfig.ProviderEnvPolicyIsolated,
				HomePolicy: aghconfig.ProviderHomePolicyIsolated,
			}
			return cfg, nil
		}
		deps.runProviderAuthCommand = func(
			_ context.Context,
			spec providerAuthCommandSpec,
		) (providerAuthCommandResult, error) {
			if spec.Command != "npx -y pi-acp@latest --terminal-login" {
				t.Fatalf("Command = %q, want Pi terminal login command", spec.Command)
			}
			wantHome := filepath.Join(homePaths.HomeDir, "providers", "pi")
			if got := providerTestEnvValue(spec.Env, "HOME"); got != wantHome {
				t.Fatalf("HOME = %q, want %q", got, wantHome)
			}
			wantAgentDir := filepath.Join(wantHome, ".pi", "agent")
			if got := providerTestEnvValue(spec.Env, "PI_CODING_AGENT_DIR"); got != wantAgentDir {
				t.Fatalf("PI_CODING_AGENT_DIR = %q, want %q", got, wantAgentDir)
			}
			return providerAuthCommandResult{ExitCode: 0, Stdout: "ok"}, nil
		}

		stdout, _, err := executeRootCommand(t, deps, "provider", "auth", "login", "pi", "-o", "json")
		if err != nil {
			t.Fatalf("provider auth login error = %v", err)
		}

		var record providerAuthStatusRecord
		if err := json.Unmarshal([]byte(stdout), &record); err != nil {
			t.Fatalf("json.Unmarshal(provider auth login) error = %v", err)
		}
		if got, want := record.AuthMode, "native_cli"; got != want {
			t.Fatalf("AuthMode = %q, want %q", got, want)
		}
		if got, want := record.LoginCommand, "npx -y pi-acp@latest --terminal-login"; got != want {
			t.Fatalf("LoginCommand = %q, want %q", got, want)
		}
	})

	t.Run("Should reject builtin wrapped provider login without running Pi terminal auth", func(t *testing.T) {
		t.Parallel()

		deps := newTestDeps(t, nil)
		deps.runProviderAuthCommand = func(
			_ context.Context,
			spec providerAuthCommandSpec,
		) (providerAuthCommandResult, error) {
			t.Fatalf("runProviderAuthCommand(%q) called, want no native login for wrapped provider", spec.Command)
			return providerAuthCommandResult{}, nil
		}

		_, _, err := executeRootCommand(t, deps, "provider", "auth", "login", "openrouter", "-o", "json")
		if err == nil {
			t.Fatal("provider auth login openrouter error = nil, want missing auth_login_command error")
		}
		if !strings.Contains(err.Error(), `provider "openrouter" does not define auth_login_command`) {
			t.Fatalf("provider auth login openrouter error = %v, want missing auth_login_command", err)
		}
	})
}

func mustTestHomePaths(t *testing.T) aghconfig.HomePaths {
	t.Helper()

	homePaths, err := aghconfig.ResolveHomePathsFrom(t.TempDir())
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	return homePaths
}

func providerTestEnvValue(env []string, key string) string {
	prefix := key + "="
	for _, entry := range env {
		if value, ok := strings.CutPrefix(entry, prefix); ok {
			return value
		}
	}
	return ""
}
