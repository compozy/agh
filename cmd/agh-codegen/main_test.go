package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckJSONFileIgnoresFormattingDifferences(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "openapi.json")
	if err := os.WriteFile(path, []byte("{\n  \"z\": 1,\n  \"nested\": {\"b\": 2, \"a\": [1, 2]}\n}\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	want := []byte("{\"nested\":{\"a\":[1,2],\"b\":2},\"z\":1}")
	if err := checkJSONFile(path, want); err != nil {
		t.Fatalf("checkJSONFile() error = %v, want nil", err)
	}
}

func TestCheckJSONFileRejectsContentDifferences(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "openapi.json")
	if err := os.WriteFile(path, []byte("{\"version\":1}\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	err := checkJSONFile(path, []byte("{\"version\":2}\n"))
	if err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("checkJSONFile() error = %v, want stale", err)
	}
}

func TestFormatTypeScriptMatchesRepositoryFormatter(t *testing.T) {
	t.Parallel()

	formatted, err := formatTypeScript("sdk/typescript/src/generated/contracts.ts", []byte("export type Value =\n  | \"a\"\n  | \"b\";\n"))
	if err != nil {
		t.Fatalf("formatTypeScript() error = %v", err)
	}
	if got, want := string(formatted), "export type Value = \"a\" | \"b\";\n"; got != want {
		t.Fatalf("formatTypeScript() = %q, want %q", got, want)
	}
}
