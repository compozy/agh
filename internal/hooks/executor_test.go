package hooks

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestNativeExecutorExecuteCallsCallback(t *testing.T) {
	t.Parallel()

	var called bool
	executor := NewNativeExecutor(func(ctx context.Context, hook RegisteredHook, payload []byte) ([]byte, error) {
		called = true
		if hook.Name != "native-hook" {
			t.Fatalf("hook.Name = %q, want %q", hook.Name, "native-hook")
		}
		if got := string(payload); got != `{"value":"demo"}` {
			t.Fatalf("payload = %q, want %q", got, `{"value":"demo"}`)
		}
		if ctx == nil {
			t.Fatal("ctx = nil, want non-nil")
		}
		return []byte(`{"ok":true}`), nil
	})

	output, err := executor.Execute(t.Context(), RegisteredHook{Name: "native-hook"}, []byte(`{"value":"demo"}`))
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if !called {
		t.Fatal("native callback was not called")
	}
	if got := string(output); got != `{"ok":true}` {
		t.Fatalf("output = %q, want %q", got, `{"ok":true}`)
	}
}

func TestNativeExecutorExecuteRecoversPanic(t *testing.T) {
	t.Parallel()

	executor := NewNativeExecutor(func(context.Context, RegisteredHook, []byte) ([]byte, error) {
		panic("boom")
	})

	output, err := executor.Execute(t.Context(), RegisteredHook{Name: "panic-hook"}, nil)
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "panic-hook") || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("Execute() error = %q, want panic detail", err)
	}
	if output != nil {
		t.Fatalf("output = %q, want nil", string(output))
	}
}

func TestSubprocessExecutorExecuteCapturesStdout(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("subprocess shell test requires POSIX shell")
	}

	executor := NewSubprocessExecutor("/bin/sh", []string{"-c", "printf 'hello-from-hook'"})

	output, err := executor.Execute(t.Context(), RegisteredHook{Name: "stdout-hook"}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if got := string(output); got != "hello-from-hook" {
		t.Fatalf("output = %q, want %q", got, "hello-from-hook")
	}
}

func TestSubprocessExecutorExecutePassesPayloadViaStdin(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("subprocess shell test requires POSIX shell")
	}

	executor := NewSubprocessExecutor("/bin/sh", []string{"-c", "payload=$(cat); printf '%s' \"$payload\""})
	payload := []byte(`{"event":"session.post_create","session_id":"session-123"}`)

	output, err := executor.Execute(t.Context(), RegisteredHook{Name: "stdin-hook"}, payload)
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if string(output) != string(payload) {
		t.Fatalf("output = %q, want %q", string(output), string(payload))
	}
}

func TestSubprocessExecutorExecuteTimesOut(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("subprocess shell test requires POSIX shell")
	}

	executor := NewSubprocessExecutor("/bin/sh", []string{"-c", "while :; do :; done"})

	started := time.Now()
	_, err := executor.Execute(t.Context(), RegisteredHook{
		Name:    "timeout-hook",
		Timeout: 120 * time.Millisecond,
	}, nil)
	elapsed := time.Since(started)
	if err == nil {
		t.Fatal("Execute() error = nil, want timeout error")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("Execute() error = %q, want timeout detail", err)
	}
	if elapsed > 2*time.Second {
		t.Fatalf("Execute() elapsed = %s, want prompt timeout handling", elapsed)
	}
}

func TestSubprocessExecutorExecuteFiltersEnvironment(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("subprocess shell test requires POSIX shell")
	}

	t.Setenv("HOOK_TEST_AMBIENT_SECRET", "ambient-secret")
	executor := NewSubprocessExecutor(
		"/bin/sh",
		[]string{"-c", `printf '%s|%s|%s' "${HOOK_TEST_AMBIENT_SECRET:-}" "${PATH:+present}" "${HOOK_CUSTOM_ENV:-}"`},
		WithSubprocessEnv(map[string]string{"HOOK_CUSTOM_ENV": "custom-value"}),
	)

	output, err := executor.Execute(t.Context(), RegisteredHook{Name: "env-hook"}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if got := string(output); got != "|present|custom-value" {
		t.Fatalf("output = %q, want %q", got, "|present|custom-value")
	}
}

func TestSubprocessExecutorExecuteCapturesStderrOnFailure(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("subprocess shell test requires POSIX shell")
	}

	executor := NewSubprocessExecutor("/bin/sh", []string{"-c", "printf 'partial-stdout'; printf 'problem' >&2; exit 7"})

	output, err := executor.Execute(t.Context(), RegisteredHook{Name: "stderr-hook"}, nil)
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if got := string(output); got != "partial-stdout" {
		t.Fatalf("output = %q, want %q", got, "partial-stdout")
	}
	if !strings.Contains(err.Error(), "hook command failed") || !strings.Contains(err.Error(), "redacted output") {
		t.Fatalf("Execute() error = %q, want stderr summary detail", err)
	}
}

func TestSubprocessExecutorExecuteCapsCapturedOutput(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("subprocess shell test requires POSIX shell")
	}

	executor := NewSubprocessExecutor(
		"/bin/sh",
		[]string{"-c", "yes x | tr -d '\\n' | head -c 9000; yes y | tr -d '\\n' | head -c 9000 >&2; exit 7"},
	)

	output, err := executor.Execute(t.Context(), RegisteredHook{Name: "truncate-hook"}, nil)
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(string(output), subprocessCaptureTruncate) {
		t.Fatalf("output = %q, want truncation marker", string(output))
	}
	if !strings.Contains(err.Error(), "truncated") {
		t.Fatalf("Execute() error = %q, want truncated stderr summary", err)
	}
}

func TestWasmExecutorExecuteReturnsErrNotImplemented(t *testing.T) {
	t.Parallel()

	executor := &WasmExecutor{}

	output, err := executor.Execute(t.Context(), RegisteredHook{Name: "wasm-hook"}, nil)
	if !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("Execute() error = %v, want ErrNotImplemented", err)
	}
	if output != nil {
		t.Fatalf("output = %q, want nil", string(output))
	}
}
