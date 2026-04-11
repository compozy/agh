package core

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	aghconfig "github.com/pedronauck/agh/internal/config"
)

func TestResolveUserHomeDir(t *testing.T) {
	t.Parallel()

	type result struct {
		got  string
		want string
		err  error
	}

	tests := []struct {
		name               string
		run                func(t *testing.T) result
		wantErrContains    string
		wantErrNotContains string
	}{
		{
			name: "ShouldPreferResolvedLookupValue",
			run: func(t *testing.T) result {
				t.Helper()

				want := filepath.Join(t.TempDir(), "user-home")
				homePaths, err := aghconfig.ResolveHomePathsFrom(filepath.Join(t.TempDir(), aghconfig.DirName))
				if err != nil {
					t.Fatalf("ResolveHomePathsFrom() error = %v", err)
				}

				got, resolveErr := resolveUserHomeDir(homePaths, func() (string, error) {
					return want, nil
				})
				return result{got: got, want: want, err: resolveErr}
			},
		},
		{
			name: "ShouldFallbackToCanonicalAGHHomeParentWhenLookupFails",
			run: func(t *testing.T) result {
				t.Helper()

				aghHome := filepath.Join(t.TempDir(), aghconfig.DirName)
				homePaths, err := aghconfig.ResolveHomePathsFrom(aghHome)
				if err != nil {
					t.Fatalf("ResolveHomePathsFrom() error = %v", err)
				}

				got, resolveErr := resolveUserHomeDir(homePaths, func() (string, error) {
					return "", errors.New("boom")
				})
				return result{got: got, want: filepath.Dir(homePaths.HomeDir), err: resolveErr}
			},
		},
		{
			name: "ShouldReturnRedactedErrorWhenResolvePathFailsWithoutFallback",
			run: func(t *testing.T) result {
				t.Helper()

				homePaths, err := aghconfig.ResolveHomePathsFrom(filepath.Join(t.TempDir(), "agh-home"))
				if err != nil {
					t.Fatalf("ResolveHomePathsFrom() error = %v", err)
				}

				got, resolveErr := resolveUserHomeDirWithResolver(
					homePaths,
					func() (string, error) {
						return "secret-user-home", nil
					},
					func(string) (string, error) {
						return "", errors.New("boom")
					},
				)
				return result{got: got, want: "", err: resolveErr}
			},
			wantErrContains:    "resolve user home directory",
			wantErrNotContains: "secret-user",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.run(t)
			if result.got != result.want {
				t.Fatalf("resolveUserHomeDir() = %q, want %q", result.got, result.want)
			}

			if tt.wantErrContains == "" {
				if result.err != nil {
					t.Fatalf("resolveUserHomeDir() error = %v, want nil", result.err)
				}
				return
			}

			if result.err == nil {
				t.Fatal("resolveUserHomeDir() error = nil, want non-nil")
			}
			if !strings.Contains(result.err.Error(), tt.wantErrContains) {
				t.Fatalf("resolveUserHomeDir() error = %q, want substring %q", result.err.Error(), tt.wantErrContains)
			}
			if tt.wantErrNotContains != "" && strings.Contains(result.err.Error(), tt.wantErrNotContains) {
				t.Fatalf("resolveUserHomeDir() error = %q, should not include %q", result.err.Error(), tt.wantErrNotContains)
			}
		})
	}
}
