//go:build mage

package main

import (
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
