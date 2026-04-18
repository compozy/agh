//go:build mage

package main

import (
	"errors"
	"os/exec"
	"testing"

	"github.com/pedronauck/agh/internal/e2elane"
)

func TestShouldEnsureWebBundle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		plan e2elane.Plan
		want bool
	}{
		{
			name: "runtime go suites require the bundle",
			plan: e2elane.Plan{
				GoSuites: []e2elane.GoSuite{{Packages: []string{"./internal/daemon"}}},
			},
			want: true,
		},
		{
			name: "daemon served browser suites require the bundle",
			plan: e2elane.Plan{
				ScriptSuites:                []e2elane.ScriptSuite{{Dir: "web", Script: "test:e2e:daemon-served"}},
				RequiresDaemonServedBrowser: true,
			},
			want: true,
		},
		{
			name: "non browser script suites alone do not require the bundle",
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

	t.Run("Should set cgo for race commands without mutating the input", func(t *testing.T) {
		t.Parallel()

		overrides := map[string]string{
			"CI":          "true",
			"CGO_ENABLED": "0",
		}

		got := withRaceEnabledEnv(overrides)

		if got["CGO_ENABLED"] != "1" {
			t.Fatalf("withRaceEnabledEnv() CGO_ENABLED = %q, want %q", got["CGO_ENABLED"], "1")
		}
		if got["CI"] != "true" {
			t.Fatalf("withRaceEnabledEnv() CI = %q, want %q", got["CI"], "true")
		}
		if overrides["CGO_ENABLED"] != "0" {
			t.Fatalf("withRaceEnabledEnv() mutated input CGO_ENABLED to %q", overrides["CGO_ENABLED"])
		}

		got["EXTRA"] = "value"
		if _, ok := overrides["EXTRA"]; ok {
			t.Fatal("withRaceEnabledEnv() reused the input map")
		}
	})

	t.Run("Should work with nil input", func(t *testing.T) {
		t.Parallel()

		got := withRaceEnabledEnv(nil)
		if got["CGO_ENABLED"] != "1" {
			t.Fatalf("withRaceEnabledEnv(nil) CGO_ENABLED = %q, want %q", got["CGO_ENABLED"], "1")
		}
	})
}

func TestRunRaceEnabledGoCommand(t *testing.T) {
	t.Parallel()

	t.Run("Should wrap subprocess failures with typed context", func(t *testing.T) {
		t.Parallel()

		err := runRaceEnabledGoCommand(nil, "definitely-not-a-go-subcommand")
		if err == nil {
			t.Fatal("runRaceEnabledGoCommand() error = nil, want non-nil")
		}
		if !errors.Is(err, errRaceEnabledGoCommand) {
			t.Fatalf("runRaceEnabledGoCommand() error = %v, want errRaceEnabledGoCommand in chain", err)
		}
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("runRaceEnabledGoCommand() error = %v, want exec.ExitError in chain", err)
		}
	})
}
