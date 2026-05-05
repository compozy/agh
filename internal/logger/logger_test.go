package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewWritesStructuredLogsToFile(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "logs", "agh.log")

	log, closeFn, err := New(WithLevel("debug"), WithFile(logFile), WithMirrorToStderr(false))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	log.Info("hello", "component", "test")

	if err := closeFn(); err != nil {
		t.Fatalf("closeFn() error = %v", err)
	}

	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(data), `"msg":"hello"`) {
		t.Fatalf("log file = %q, want hello message", string(data))
	}
}

func TestParseLevelRejectsUnsupportedValue(t *testing.T) {
	t.Parallel()

	if _, err := ParseLevel("trace"); err == nil {
		t.Fatal("ParseLevel() error = nil, want non-nil")
	}
}

func TestParseLevelAcceptsConfiguredValues(t *testing.T) {
	t.Parallel()

	tests := []string{"", "debug", "info", "warn", "error"}
	for _, tt := range tests {
		if _, err := ParseLevel(tt); err != nil {
			t.Fatalf("ParseLevel(%q) error = %v", tt, err)
		}
	}
}

func TestNewWithoutFileStillBuildsLogger(t *testing.T) {
	t.Parallel()

	log, closeFn, err := New(WithMirrorToStderr(false))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if log == nil {
		t.Fatal("New() logger = nil")
	}
	if err := closeFn(); err != nil {
		t.Fatalf("closeFn() error = %v", err)
	}
}

func TestMirrorToStderrHelpers(t *testing.T) {
	t.Parallel()

	t.Run("ShouldDisableMirrorWhenEnvExplicitlyFalse", func(t *testing.T) {
		t.Parallel()

		getenv := func(string) string {
			return "false"
		}
		if MirrorToStderrEnabled(getenv) {
			t.Fatal("MirrorToStderrEnabled(false) = true, want false")
		}
	})

	t.Run("ShouldDefaultMirrorWhenEnvUnset", func(t *testing.T) {
		t.Parallel()

		getenv := func(string) string {
			return ""
		}
		if !MirrorToStderrEnabled(getenv) {
			t.Fatal("MirrorToStderrEnabled(unset) = false, want true")
		}
	})

	t.Run("ShouldInjectDetachedEnvOverride", func(t *testing.T) {
		t.Parallel()

		sandbox := WithMirrorToStderrEnv([]string{"PATH=/usr/bin"}, false)
		joined := strings.Join(sandbox, "\n")
		if !strings.Contains(joined, "AGH_INTERNAL_LOG_MIRROR_STDERR=0") {
			t.Fatalf("WithMirrorToStderrEnv(false) = %q, want AGH_INTERNAL_LOG_MIRROR_STDERR=0", joined)
		}
	})
}
