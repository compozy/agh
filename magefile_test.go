//go:build mage

package main

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/compozy/agh/internal/e2elane"
)

func TestShouldEnsureWebBundle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		plan e2elane.Plan
		want bool
	}{
		{
			name: "Should require the bundle for runtime Go suites",
			plan: e2elane.Plan{
				GoSuites: []e2elane.GoSuite{{Packages: []string{"./internal/daemon"}}},
			},
			want: true,
		},
		{
			name: "Should require the bundle for daemon-served browser suites",
			plan: e2elane.Plan{
				ScriptSuites:                []e2elane.ScriptSuite{{Dir: "web", Script: "test:e2e:daemon-served"}},
				RequiresDaemonServedBrowser: true,
			},
			want: true,
		},
		{
			name: "Should skip the bundle for non-browser script suites alone",
			plan: e2elane.Plan{
				ScriptSuites: []e2elane.ScriptSuite{{Dir: "scripts", Script: "echo"}},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := shouldEnsureWebBundle(tt.plan); got != tt.want {
				t.Fatalf("shouldEnsureWebBundle() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWithRaceEnabledEnv(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		overrides map[string]string
		want      map[string]string
		wantInput map[string]string
	}{
		{
			name: "Should set cgo for race commands without mutating the input",
			overrides: map[string]string{
				"CI":          "true",
				"CGO_ENABLED": "0",
			},
			want: map[string]string{
				"CI":          "true",
				"CGO_ENABLED": "1",
			},
			wantInput: map[string]string{
				"CI":          "true",
				"CGO_ENABLED": "0",
			},
		},
		{
			name: "Should work with nil input",
			want: map[string]string{
				"CGO_ENABLED": "1",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := withRaceEnabledEnv(tt.overrides)
			for key, want := range tt.want {
				if got[key] != want {
					t.Fatalf("withRaceEnabledEnv() %s = %q, want %q", key, got[key], want)
				}
			}
			if tt.wantInput != nil {
				for key, want := range tt.wantInput {
					if tt.overrides[key] != want {
						t.Fatalf("withRaceEnabledEnv() mutated input %s to %q, want %q", key, tt.overrides[key], want)
					}
				}

				got["EXTRA"] = "value"
				if _, ok := tt.overrides["EXTRA"]; ok {
					t.Fatal("withRaceEnabledEnv() reused the input map")
				}
			}
		})
	}
}

func TestRunRaceEnabledGoCommand(t *testing.T) {
	t.Parallel()

	t.Run("Should wrap subprocess failures with command context", func(t *testing.T) {
		t.Parallel()

		err := runRaceEnabledGoCommand(context.Background(), nil, "definitely-not-a-go-subcommand")
		if err == nil {
			t.Fatal("runRaceEnabledGoCommand() error = nil, want non-nil")
		}
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("runRaceEnabledGoCommand() error = %v, want exec.ExitError in chain", err)
		}
		if !strings.Contains(err.Error(), "definitely-not-a-go-subcommand") {
			t.Fatalf("runRaceEnabledGoCommand() error = %v, want command arguments in message", err)
		}
	})

	t.Run("Should respect canceled contexts", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := runRaceEnabledGoCommand(ctx, nil, "version")
		if err == nil {
			t.Fatal("runRaceEnabledGoCommand() error = nil, want context cancellation")
		}
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("runRaceEnabledGoCommand() error = %v, want context.Canceled in chain", err)
		}
	})
}

func TestInstallerCheck(t *testing.T) {
	t.Parallel()

	t.Run("Should validate the installer script in dry-run mode", func(t *testing.T) {
		t.Parallel()

		if err := InstallerCheck(); err != nil {
			t.Fatalf("InstallerCheck() error = %v", err)
		}
	})
}
