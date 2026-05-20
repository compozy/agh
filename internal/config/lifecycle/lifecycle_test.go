package lifecycle

import "testing"

func TestClassifyPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		path          string
		wantLifecycle Lifecycle
		wantDiffClass DiffClass
		wantErr       bool
	}{
		{
			name:          "Should classify disabled skills as live",
			path:          "skills.disabled_skills",
			wantLifecycle: Live,
			wantDiffClass: DiffClassLive,
		},
		{
			name:          "Should classify provider descendants as restart required",
			path:          "providers.codex.command",
			wantLifecycle: RestartRequired,
			wantDiffClass: DiffClassRestartRequired,
		},
		{
			name:          "Should classify sandbox descendants as session rebind",
			path:          "sandboxes.daytona-dev.backend",
			wantLifecycle: SessionRebind,
			wantDiffClass: DiffClassSessionRebind,
		},
		{
			name:          "Should classify reload timeout as live",
			path:          "daemon.reload_timeouts.providers",
			wantLifecycle: Live,
			wantDiffClass: DiffClassLive,
		},
		{
			name:    "Should reject unknown paths",
			path:    "unknown.path",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ClassifyPath(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Fatal("ClassifyPath() error = nil, want non-nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("ClassifyPath() error = %v", err)
			}
			if got.Lifecycle != tt.wantLifecycle {
				t.Fatalf("ClassifyPath().Lifecycle = %q, want %q", got.Lifecycle, tt.wantLifecycle)
			}
			if got.DiffClass != tt.wantDiffClass {
				t.Fatalf("ClassifyPath().DiffClass = %q, want %q", got.DiffClass, tt.wantDiffClass)
			}
		})
	}
}

func TestClassifyPaths(t *testing.T) {
	t.Parallel()

	t.Run("Should default empty mutations to live", func(t *testing.T) {
		t.Parallel()

		gotLifecycle, gotDiffClass, err := ClassifyPaths(nil)
		if err != nil {
			t.Fatalf("ClassifyPaths() error = %v", err)
		}
		if gotLifecycle != Live || gotDiffClass != DiffClassLive {
			t.Fatalf(
				"ClassifyPaths() = (%q, %q), want (%q, %q)",
				gotLifecycle,
				gotDiffClass,
				Live,
				DiffClassLive,
			)
		}
	})

	t.Run("Should let restart required dominate mixed changes", func(t *testing.T) {
		t.Parallel()

		gotLifecycle, gotDiffClass, err := ClassifyPaths([]string{
			"daemon.reload_timeouts.providers",
			"providers.codex.command",
		})
		if err != nil {
			t.Fatalf("ClassifyPaths() error = %v", err)
		}
		if gotLifecycle != RestartRequired || gotDiffClass != DiffClassRestartRequired {
			t.Fatalf(
				"ClassifyPaths() = (%q, %q), want (%q, %q)",
				gotLifecycle,
				gotDiffClass,
				RestartRequired,
				DiffClassRestartRequired,
			)
		}
	})
}

func TestNextActionForLifecycle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		lifecycle Lifecycle
		status    Status
		want      NextAction
	}{
		{
			name:      "Should retry failed applies",
			lifecycle: Live,
			status:    StatusFailed,
			want:      NextActionRetry,
		},
		{
			name:      "Should request daemon restart for blocked restart required applies",
			lifecycle: RestartRequired,
			status:    StatusBlocked,
			want:      NextActionRestartDaemon,
		},
		{
			name:      "Should request new sessions for applied session rebind changes",
			lifecycle: SessionRebind,
			status:    StatusApplied,
			want:      NextActionNewSession,
		},
		{
			name:      "Should return none for live applied changes",
			lifecycle: Live,
			status:    StatusApplied,
			want:      NextActionNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := NextActionForLifecycle(tt.lifecycle, tt.status); got != tt.want {
				t.Fatalf("NextActionForLifecycle() = %q, want %q", got, tt.want)
			}
		})
	}
}
