package doctor

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/compozy/agh/internal/api/contract"
	bridgepkg "github.com/compozy/agh/internal/bridges"
	"github.com/compozy/agh/internal/diagnostics"
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

func TestBridgeProbeCategoryFilter(t *testing.T) {
	t.Run("Should skip bridge checks without an explicit bridge filter", func(t *testing.T) {
		t.Parallel()

		source := &bridgeProbeSourceStub{
			instances: []bridgepkg.BridgeInstance{{
				ID:            "brg-slack",
				Platform:      "slack",
				ExtensionName: "slack",
			}},
		}
		registry := NewRegistry()
		if err := registry.Register(&BridgeProbe{Source: source}); err != nil {
			t.Fatalf("Register(bridge) error = %v", err)
		}
		runner, err := NewRunner(registry)
		if err != nil {
			t.Fatalf("NewRunner() error = %v", err)
		}

		items, err := runner.Run(context.Background(), RunOptions{})
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if source.listCalls != 0 || source.checkCalls != 0 {
			t.Fatalf("bridge source calls = list:%d check:%d, want 0/0", source.listCalls, source.checkCalls)
		}
		if len(items) != 0 {
			t.Fatalf("bridge diagnostic count = %d, want 0", len(items))
		}
	})

	t.Run("Should run only bridge checks and preserve failing secret remediation", func(t *testing.T) {
		t.Parallel()

		source := &bridgeProbeSourceStub{
			instances: []bridgepkg.BridgeInstance{{
				ID:            "brg-slack",
				Platform:      "slack",
				ExtensionName: "slack",
			}},
			response: bridgepkg.BridgeCheckResponse{Checks: []bridgepkg.BridgeCheckRecord{{
				Check:       "provider.identity",
				Status:      bridgepkg.BridgeCheckStatusFail,
				Remediation: "Bind the required bot_token bridge secret and retry.",
			}}},
		}
		otherProbeCalls := 0
		registry := NewRegistry()
		if err := registry.Register(&ProbeFunc{
			ProbeID:       "runtime.providers",
			ProbeCategory: contract.CategoryProvider,
			RunFunc: func(context.Context, *ProbeEnv) ([]contract.DiagnosticItem, error) {
				otherProbeCalls++
				return nil, nil
			},
		}); err != nil {
			t.Fatalf("Register(provider) error = %v", err)
		}
		if err := registry.Register(&BridgeProbe{Source: source}); err != nil {
			t.Fatalf("Register(bridge) error = %v", err)
		}
		runner, err := NewRunner(registry)
		if err != nil {
			t.Fatalf("NewRunner() error = %v", err)
		}

		items, err := runner.Run(context.Background(), RunOptions{Only: []string{contract.CategoryBridge}})
		if err != nil {
			t.Fatalf("Run(bridge only) error = %v", err)
		}
		if otherProbeCalls != 0 {
			t.Fatalf("provider probe calls = %d, want 0", otherProbeCalls)
		}
		if source.listCalls != 1 || source.checkCalls != 1 {
			t.Fatalf("bridge source calls = list:%d check:%d, want 1/1", source.listCalls, source.checkCalls)
		}
		if len(items) != 1 {
			t.Fatalf("bridge diagnostic count = %d, want 1", len(items))
		}
		item := items[0]
		if item.Category != contract.CategoryBridge || item.Severity != contract.SeverityError ||
			!strings.Contains(item.Message, "bot_token") {
			t.Fatalf("bridge diagnostic = %#v, want failed bot_token remediation", item)
		}
		if item.SuggestedCommand != "agh bridge verify brg-slack" {
			t.Fatalf("SuggestedCommand = %q, want bridge verify follow-up", item.SuggestedCommand)
		}
	})

	t.Run("Should never describe a failed check without remediation as passed", func(t *testing.T) {
		t.Parallel()

		item := bridgeCheckDiagnosticItem(
			bridgepkg.BridgeInstance{ID: "brg-slack", Platform: "slack", ExtensionName: "slack"},
			bridgepkg.BridgeCheckRecord{Check: "provider.identity", Status: bridgepkg.BridgeCheckStatusFail},
		)
		if item.Severity != contract.SeverityError || strings.Contains(strings.ToLower(item.Message), "passed") ||
			!strings.Contains(strings.ToLower(item.Message), "no remediation") {
			t.Fatalf("bridge failure diagnostic = %#v, want explicit missing-remediation error", item)
		}
	})
}

type bridgeProbeSourceStub struct {
	instances  []bridgepkg.BridgeInstance
	response   bridgepkg.BridgeCheckResponse
	listCalls  int
	checkCalls int
}

func (s *bridgeProbeSourceStub) ListInstances(context.Context) ([]bridgepkg.BridgeInstance, error) {
	s.listCalls++
	return append([]bridgepkg.BridgeInstance(nil), s.instances...), nil
}

func (s *bridgeProbeSourceStub) CheckBridge(
	_ context.Context,
	extensionName string,
	request bridgepkg.BridgeCheckRequest,
) (bridgepkg.BridgeCheckResponse, error) {
	s.checkCalls++
	if extensionName != "slack" || request.BridgeInstanceID != "brg-slack" {
		return bridgepkg.BridgeCheckResponse{}, errors.New("unexpected bridge check request")
	}
	return s.response, nil
}
