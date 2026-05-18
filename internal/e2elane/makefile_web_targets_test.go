package e2elane

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestMakefileWebValidationTargetsUseTurbo(t *testing.T) {
	t.Parallel()

	repoRoot := repoRoot(t)
	recipes := makeTargetRecipes(t, filepath.Join(repoRoot, "Makefile"))

	tests := []struct {
		name          string
		target        string
		wantSnippet   string
		forbidSnippet string
	}{
		{
			name:          "Should delegate web typecheck to the Turbo web filter",
			target:        "web-typecheck",
			wantSnippet:   "bunx turbo run typecheck --filter=./web",
			forbidSnippet: "cd web && bun run typecheck",
		},
		{
			name:          "Should delegate web test to the Turbo web filter",
			target:        "web-test",
			wantSnippet:   "bunx turbo run test --filter=./web",
			forbidSnippet: "cd web && bun run test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			recipe := recipes[tt.target]
			if !strings.Contains(recipe, tt.wantSnippet) {
				t.Fatalf("Makefile target %s recipe = %q, want snippet %q", tt.target, recipe, tt.wantSnippet)
			}
			if strings.Contains(recipe, tt.forbidSnippet) {
				t.Fatalf(
					"Makefile target %s recipe unexpectedly referenced %q: %q",
					tt.target,
					tt.forbidSnippet,
					recipe,
				)
			}
		})
	}
}

func TestMakefileWebValidationTargetsDryRunTurbo(t *testing.T) {
	t.Parallel()

	repoRoot := repoRoot(t)
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "Should dry run web typecheck through Turbo",
			args: []string{"make", "-n", "web-typecheck"},
			want: "bunx turbo run typecheck --filter=./web",
		},
		{
			name: "Should dry run web test through Turbo",
			args: []string{"make", "-n", "web-test"},
			want: "bunx turbo run test --filter=./web",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			output := runRepoCommand(t, repoRoot, tt.args...)
			if !strings.Contains(output, tt.want) {
				t.Fatalf("%s output = %q, want snippet %q", strings.Join(tt.args, " "), output, tt.want)
			}
		})
	}
}
