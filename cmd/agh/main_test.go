package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/compozy/agh/internal/version"
)

func TestRunPrintsVersion(t *testing.T) {
	t.Cleanup(version.OverrideVersionForTesting("test-version"))

	var stdout bytes.Buffer
	exitCode := run(context.Background(), []string{"version"}, &stdout, &bytes.Buffer{})
	if exitCode != 0 {
		t.Fatalf("run(version) exit code = %d, want 0", exitCode)
	}
	if got := strings.TrimSpace(stdout.String()); got != "agh test-version" {
		t.Fatalf("run(version) output = %q, want %q", got, "agh test-version")
	}
}

func TestRunHelpShowsRootUsage(t *testing.T) {
	var stdout bytes.Buffer
	exitCode := run(context.Background(), nil, &stdout, &bytes.Buffer{})
	if exitCode != 0 {
		t.Fatalf("run(help) exit code = %d, want 0", exitCode)
	}
	if got := stdout.String(); !strings.Contains(got, "Usage:") || !strings.Contains(got, "agh") {
		t.Fatalf("run(help) output = %q, want root usage", got)
	}
}

func TestRunUnknownCommandReturnsError(t *testing.T) {
	var stderr bytes.Buffer
	exitCode := run(context.Background(), []string{"mystery"}, &bytes.Buffer{}, &stderr)
	if exitCode != 1 {
		t.Fatalf("run(unknown) exit code = %d, want 1", exitCode)
	}
	if got := stderr.String(); !strings.Contains(got, "unknown command") {
		t.Fatalf("run(unknown) stderr = %q, want unknown command error", got)
	}
}
