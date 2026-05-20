package doctor

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/diagnostics"
)

func TestRegistry(t *testing.T) {
	t.Parallel()

	t.Run("Should reject duplicate probe identifiers", func(t *testing.T) {
		t.Parallel()

		registry := NewRegistry()
		probe := &ProbeFunc{ProbeID: "doctor.a", ProbeCategory: contract.CategoryDaemon}
		if err := registry.Register(probe); err != nil {
			t.Fatalf("Register(first) error = %v", err)
		}
		if err := registry.Register(probe); err == nil {
			t.Fatal("Register(duplicate) error = nil, want duplicate failure")
		}
	})

	t.Run("Should return probes sorted by identifier", func(t *testing.T) {
		t.Parallel()

		registry := NewRegistry()
		for _, probe := range []*ProbeFunc{
			{ProbeID: "doctor.z", ProbeCategory: contract.CategoryDaemon},
			{ProbeID: "doctor.a", ProbeCategory: contract.CategoryDaemon},
		} {
			if err := registry.Register(probe); err != nil {
				t.Fatalf("Register(%s) error = %v", probe.ProbeID, err)
			}
		}
		probes := registry.Probes()
		if got, want := probes[0].ID(), "doctor.a"; got != want {
			t.Fatalf("first probe ID = %q, want %q", got, want)
		}
	})
}

func TestRunner(t *testing.T) {
	t.Parallel()

	t.Run("Should sanitize returned probe diagnostics", func(t *testing.T) {
		t.Parallel()

		registry := NewRegistry()
		if err := registry.Register(&ProbeFunc{
			ProbeID:       "doctor.provider.auth",
			ProbeCategory: contract.CategoryProvider,
			RunFunc: func(context.Context, *ProbeEnv) ([]contract.DiagnosticItem, error) {
				return []contract.DiagnosticItem{
					diagnostics.NewItem(
						"doctor.provider.auth",
						contract.CodeProviderNotAuthenticated,
						contract.CategoryProvider,
						"Provider auth",
						"stderr token=probe-secret",
						contract.SeverityWarn,
						contract.FreshnessLive,
						diagnostics.WithEvidence(map[string]any{"access_token": "secret-value"}),
					),
				}, nil
			},
		}); err != nil {
			t.Fatalf("Register() error = %v", err)
		}
		runner, err := NewRunner(registry)
		if err != nil {
			t.Fatalf("NewRunner() error = %v", err)
		}

		items, err := runner.Run(context.Background(), RunOptions{})
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if len(items) != 1 {
			t.Fatalf("Run() item count = %d, want 1", len(items))
		}
		if strings.Contains(items[0].Message, "probe-secret") {
			t.Fatalf("Diagnostic message = %q leaked secret", items[0].Message)
		}
		if items[0].Evidence["access_token"] != "[REDACTED]" {
			t.Fatalf("Evidence[access_token] = %#v, want redacted marker", items[0].Evidence["access_token"])
		}
	})

	t.Run("Should return structured diagnostic when probe fails", func(t *testing.T) {
		t.Parallel()

		registry := NewRegistry()
		if err := registry.Register(&ProbeFunc{
			ProbeID:       "doctor.provider.cli",
			ProbeCategory: contract.CategoryProvider,
			RunFunc: func(context.Context, *ProbeEnv) ([]contract.DiagnosticItem, error) {
				return nil, errors.New("provider stderr token=raw-secret")
			},
		}); err != nil {
			t.Fatalf("Register() error = %v", err)
		}
		runner, err := NewRunner(registry)
		if err != nil {
			t.Fatalf("NewRunner() error = %v", err)
		}

		items, err := runner.Run(context.Background(), RunOptions{})
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if len(items) != 1 {
			t.Fatalf("Run() item count = %d, want 1", len(items))
		}
		if items[0].Code != contract.CodeProbeFailed {
			t.Fatalf("failure Code = %q, want %q", items[0].Code, contract.CodeProbeFailed)
		}
		if strings.Contains(items[0].Message, "raw-secret") {
			t.Fatalf("failure Message = %q leaked raw error", items[0].Message)
		}
	})

	t.Run("Should classify probe timeouts as timeout diagnostics", func(t *testing.T) {
		t.Parallel()

		registry := NewRegistry()
		if err := registry.Register(&ProbeFunc{
			ProbeID:       "doctor.daemon.timeout",
			ProbeCategory: contract.CategoryDaemon,
			RunFunc: func(ctx context.Context, _ *ProbeEnv) ([]contract.DiagnosticItem, error) {
				<-ctx.Done()
				return nil, ctx.Err()
			},
		}); err != nil {
			t.Fatalf("Register() error = %v", err)
		}
		runner, err := NewRunner(registry)
		if err != nil {
			t.Fatalf("NewRunner() error = %v", err)
		}

		items, err := runner.Run(context.Background(), RunOptions{ProbeTimeout: time.Millisecond})
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if len(items) != 1 {
			t.Fatalf("Run() item count = %d, want 1", len(items))
		}
		if items[0].Code != contract.CodeProbeTimeout {
			t.Fatalf("timeout Code = %q, want %q", items[0].Code, contract.CodeProbeTimeout)
		}
	})
}
