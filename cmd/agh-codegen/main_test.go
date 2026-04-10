package main

import (
	"errors"
	"os"
	"path/filepath"
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

func TestFormatTypeScript(t *testing.T) {
	t.Run("ShouldMatchRepositoryFormatter", func(t *testing.T) {
		t.Parallel()

		formatted, err := formatTypeScript("sdk/typescript/src/generated/contracts.ts", []byte("export type Value =\n  | \"a\"\n  | \"b\";\n"))
		if err != nil {
			t.Fatalf("formatTypeScript() error = %v", err)
		}
		if got, want := string(formatted), "export type Value = \"a\" | \"b\";\n"; got != want {
			t.Fatalf("formatTypeScript() = %q, want %q", got, want)
		}
	})
}
