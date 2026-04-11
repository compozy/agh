package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"syscall"
	"testing"
)

func TestCheckJSONFile(t *testing.T) {
	tests := []struct {
		name        string
		fileContent []byte
		want        []byte
		wantErr     error
	}{
		{
			name:        "ShouldIgnoreFormattingDifferences",
			fileContent: []byte("{\n  \"z\": 1,\n  \"nested\": {\"b\": 2, \"a\": [1, 2]}\n}\n"),
			want:        []byte("{\"nested\":{\"a\":[1,2],\"b\":2},\"z\":1}"),
		},
		{
			name:        "ShouldRejectContentDifferences",
			fileContent: []byte("{\"version\":1}\n"),
			want:        []byte("{\"version\":2}\n"),
			wantErr:     ErrStaleGeneratedFile,
		},
		{
			name:        "ShouldIgnoreEquivalentNumberRepresentations",
			fileContent: []byte("{\"version\":1.0}\n"),
			want:        []byte("{\"version\":1}\n"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "openapi.json")
			if err := os.WriteFile(path, tt.fileContent, 0o644); err != nil {
				t.Fatalf("os.WriteFile() error = %v", err)
			}

			err := checkJSONFile(path, tt.want)
			switch {
			case tt.wantErr == nil && err != nil:
				t.Fatalf("checkJSONFile() error = %v, want nil", err)
			case tt.wantErr != nil && !errors.Is(err, tt.wantErr):
				t.Fatalf("checkJSONFile() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestShutdownSignals(t *testing.T) {
	t.Parallel()

	signals := shutdownSignals()
	if !slices.ContainsFunc(signals, func(signal os.Signal) bool {
		return signal == os.Interrupt
	}) {
		t.Fatalf("shutdownSignals() = %#v, want os.Interrupt", signals)
	}
	if !slices.ContainsFunc(signals, func(signal os.Signal) bool {
		return signal == syscall.SIGTERM
	}) {
		t.Fatalf("shutdownSignals() = %#v, want syscall.SIGTERM", signals)
	}
}

func TestFormatTypeScript(t *testing.T) {
	t.Run("ShouldMatchRepositoryFormatter", func(t *testing.T) {
		t.Parallel()

		formatted, err := formatTypeScript(context.Background(), "sdk/typescript/src/generated/contracts.ts", []byte("export type Value =\n  | \"a\"\n  | \"b\";\n"))
		if err != nil {
			t.Fatalf("formatTypeScript() error = %v", err)
		}
		if got, want := string(formatted), "export type Value = \"a\" | \"b\";\n"; got != want {
			t.Fatalf("formatTypeScript() = %q, want %q", got, want)
		}
	})

	t.Run("ShouldReturnContextErrorWhenCanceled", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := formatTypeScript(ctx, "sdk/typescript/src/generated/contracts.ts", []byte("export type Value = \"a\";\n"))
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("formatTypeScript() error = %v, want context.Canceled", err)
		}
	})
}

func TestRemoveTemporaryFile(t *testing.T) {
	t.Run("ShouldIgnoreMissingFile", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "missing-openapi.json")
		if err := removeTemporaryFile(path); err != nil {
			t.Fatalf("removeTemporaryFile() error = %v, want nil", err)
		}
	})

	t.Run("ShouldReturnErrorWhenDirectoryDoesNotPermitDeletion", func(t *testing.T) {
		t.Parallel()

		if runtime.GOOS == "windows" {
			t.Skip("directory permission semantics differ on windows")
		}

		dir := filepath.Join(t.TempDir(), "locked")
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatalf("os.Mkdir() error = %v", err)
		}

		path := filepath.Join(dir, "openapi.json")
		if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
			t.Fatalf("os.WriteFile() error = %v", err)
		}

		if err := os.Chmod(dir, 0o500); err != nil {
			t.Fatalf("os.Chmod(%q) error = %v", dir, err)
		}
		t.Cleanup(func() {
			if err := os.Chmod(dir, 0o700); err != nil {
				t.Fatalf("restore directory permissions: %v", err)
			}
		})

		err := removeTemporaryFile(path)
		if err == nil {
			t.Fatal("removeTemporaryFile() error = nil, want non-nil")
		}
		if !strings.Contains(err.Error(), path) {
			t.Fatalf("removeTemporaryFile() error = %q, want path %q in message", err.Error(), path)
		}
	})
}
