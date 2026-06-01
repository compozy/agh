package e2elane

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPreCommitHook(t *testing.T) {
	t.Parallel()

	t.Run("Should fail before lint-staged when react-doctor reports regressions", func(t *testing.T) {
		t.Parallel()

		repoRoot := repoRoot(t)
		workdir := t.TempDir()
		hookPath := filepath.Join(workdir, "pre-commit")
		hookSource := filepath.Join(repoRoot, ".husky", "pre-commit")

		hookContents, err := os.ReadFile(hookSource)
		if err != nil {
			t.Fatalf("os.ReadFile(%q) error = %v", hookSource, err)
		}
		if err := os.WriteFile(hookPath, hookContents, 0o755); err != nil {
			t.Fatalf("os.WriteFile(%q) error = %v", hookPath, err)
		}

		binDir := filepath.Join(workdir, "bin")
		if err := os.MkdirAll(binDir, 0o755); err != nil {
			t.Fatalf("os.MkdirAll(%q) error = %v", binDir, err)
		}

		markerPath := filepath.Join(workdir, "lint-staged-invoked")
		writeExecutable(
			t,
			filepath.Join(binDir, "react-doctor"),
			"#!/bin/sh\nprintf '%s\n' 'react doctor regression' >&2\nexit 1\n",
		)
		writeExecutable(
			t,
			filepath.Join(binDir, "bunx"),
			"#!/bin/sh\nprintf '%s\n' 'invoked' > \"$BUNX_MARKER\"\nexit 0\n",
		)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		t.Cleanup(cancel)

		cmd := exec.CommandContext(ctx, hookPath)
		cmd.Dir = workdir
		cmd.Env = append(
			os.Environ(),
			"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"),
			"TMPDIR="+workdir,
			"BUNX_MARKER="+markerPath,
		)

		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatal("expected hook to fail when react-doctor reports regressions")
		}
		if ctx.Err() != nil {
			t.Fatalf("hook execution timed out: %v", ctx.Err())
		}

		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("hook error = %T %v, want *exec.ExitError", err, err)
		}
		if got, want := exitErr.ExitCode(), 1; got != want {
			t.Fatalf("hook exit code = %d, want %d", got, want)
		}

		outputText := string(output)
		if !strings.Contains(outputText, "react doctor regression") {
			t.Fatalf("hook output = %q, want captured react-doctor stderr", outputText)
		}
		if !strings.Contains(outputText, "React Doctor found staged regressions.") {
			t.Fatalf("hook output = %q, want failure guidance", outputText)
		}

		_, statErr := os.Stat(markerPath)
		if statErr == nil {
			t.Fatalf("expected lint-staged marker %q to be absent", markerPath)
		}
		if !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("os.Stat(%q) error = %v, want not-exist", markerPath, statErr)
		}
	})
}

func writeExecutable(t *testing.T, path string, contents string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(contents), 0o755); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", path, err)
	}
}
