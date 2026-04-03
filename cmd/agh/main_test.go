package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/pedronauck/agh/internal/version"
)

type stubDaemonRunner struct {
	runErr error
	ran    bool
}

func (s *stubDaemonRunner) Run(context.Context) error {
	s.ran = true
	return s.runErr
}

func TestRunPrintsVersion(t *testing.T) {
	original := version.Version
	version.Version = "test-version"
	t.Cleanup(func() {
		version.Version = original
	})

	var stdout bytes.Buffer
	exitCode := run(context.Background(), []string{"version"}, &stdout, &bytes.Buffer{})
	if exitCode != 0 {
		t.Fatalf("run(version) exit code = %d, want 0", exitCode)
	}
	if got := strings.TrimSpace(stdout.String()); got != "agh test-version" {
		t.Fatalf("run(version) output = %q, want %q", got, "agh test-version")
	}
}

func TestRunStartDelegatesToDaemon(t *testing.T) {
	original := newDaemon
	runner := &stubDaemonRunner{}
	newDaemon = func() (daemonRunner, error) {
		return runner, nil
	}
	t.Cleanup(func() {
		newDaemon = original
	})

	exitCode := run(context.Background(), []string{"start"}, &bytes.Buffer{}, &bytes.Buffer{})
	if exitCode != 0 {
		t.Fatalf("run(start) exit code = %d, want 0", exitCode)
	}
	if !runner.ran {
		t.Fatal("run(start) did not invoke daemon.Run")
	}
}

func TestRunStartReturnsErrorStatus(t *testing.T) {
	original := newDaemon
	newDaemon = func() (daemonRunner, error) {
		return nil, errors.New("boom")
	}
	t.Cleanup(func() {
		newDaemon = original
	})

	var stderr bytes.Buffer
	exitCode := run(context.Background(), []string{"start"}, &bytes.Buffer{}, &stderr)
	if exitCode != 1 {
		t.Fatalf("run(start) exit code = %d, want 1", exitCode)
	}
	if !strings.Contains(stderr.String(), "boom") {
		t.Fatalf("run(start) stderr = %q, want error message", stderr.String())
	}
}

func TestRunUnknownCommandPrintsUsage(t *testing.T) {
	var stderr bytes.Buffer
	exitCode := run(context.Background(), []string{"mystery"}, &bytes.Buffer{}, &stderr)
	if exitCode != 2 {
		t.Fatalf("run(unknown) exit code = %d, want 2", exitCode)
	}
	if !strings.Contains(stderr.String(), "usage: agh [start|version]") {
		t.Fatalf("run(unknown) stderr = %q, want usage", stderr.String())
	}
}
