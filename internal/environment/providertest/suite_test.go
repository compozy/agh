package providertest

import (
	"context"
	"errors"
	"strings"
	"testing"

	acpsdk "github.com/coder/acp-go-sdk"
	"github.com/pedronauck/agh/internal/environment"
)

func TestRunLifecycleExercisesProviderContract(t *testing.T) {
	t.Parallel()

	provider := &suiteTestProvider{backend: environment.BackendLocal}
	prepared := RunLifecycle(t, LifecycleCase{
		Provider: provider,
		Backend:  environment.BackendLocal,
		PrepareRequest: environment.PrepareRequest{
			EnvironmentID: "env-suite",
			LocalRootDir:  t.TempDir(),
			Environment:   environment.Resolved{Backend: environment.BackendLocal},
		},
		AssertPrepared: func(t *testing.T, prepared environment.Prepared) {
			t.Helper()
			if prepared.State.EnvironmentID != "env-suite" {
				t.Fatalf("prepared environment id = %q, want env-suite", prepared.State.EnvironmentID)
			}
		},
		AssertFinalState: func(t *testing.T, state environment.SessionState) {
			t.Helper()
			if state.Backend != environment.BackendLocal {
				t.Fatalf("final state backend = %q, want %q", state.Backend, environment.BackendLocal)
			}
		},
	})

	if prepared.State.EnvironmentID != "env-suite" {
		t.Fatalf("RunLifecycle() state environment id = %q, want env-suite", prepared.State.EnvironmentID)
	}
	if !provider.syncedToRuntime {
		t.Fatal("RunLifecycle() did not call SyncToRuntime")
	}
	if !provider.syncedFromRuntime {
		t.Fatal("RunLifecycle() did not call SyncFromRuntime")
	}
	if !provider.destroyed {
		t.Fatal("RunLifecycle() did not call Destroy")
	}
}

func TestRunLifecycleDetectsInvalidProviderCases(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("provider failure")
	tests := []struct {
		name     string
		provider environment.Provider
		backend  environment.Backend
		wantText string
	}{
		{name: "nil provider", provider: nil, backend: environment.BackendLocal, wantText: "provider = nil"},
		{
			name:     "backend mismatch",
			provider: &suiteTestProvider{backend: environment.BackendDaytona},
			backend:  environment.BackendLocal,
			wantText: "Provider.Backend()",
		},
		{
			name:     "prepare error",
			provider: &suiteTestProvider{backend: environment.BackendLocal, prepareErr: wantErr},
			backend:  environment.BackendLocal,
			wantText: "Provider.Prepare()",
		},
		{
			name: "missing launcher",
			provider: &suiteTestProvider{
				backend:  environment.BackendLocal,
				launcher: nil,
				toolHost: suiteTestToolHost{},
			},
			backend:  environment.BackendLocal,
			wantText: "Prepared.Launcher",
		},
		{
			name: "missing tool host",
			provider: &suiteTestProvider{
				backend:  environment.BackendLocal,
				launcher: suiteTestLauncher{},
				toolHost: nil,
			},
			backend:  environment.BackendLocal,
			wantText: "Prepared.ToolHost",
		},
		{
			name:     "sync to runtime error",
			provider: &suiteTestProvider{backend: environment.BackendLocal, syncToErr: wantErr},
			backend:  environment.BackendLocal,
			wantText: "Provider.SyncToRuntime()",
		},
		{
			name:     "sync from runtime error",
			provider: &suiteTestProvider{backend: environment.BackendLocal, syncFromErr: wantErr},
			backend:  environment.BackendLocal,
			wantText: "Provider.SyncFromRuntime()",
		},
		{
			name:     "destroy error",
			provider: &suiteTestProvider{backend: environment.BackendLocal, destroyErr: wantErr},
			backend:  environment.BackendLocal,
			wantText: "Provider.Destroy()",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := runLifecycle(context.Background(), LifecycleCase{
				Provider: tt.provider,
				Backend:  tt.backend,
				PrepareRequest: environment.PrepareRequest{
					EnvironmentID: "env-suite",
					LocalRootDir:  t.TempDir(),
				},
			})
			if err == nil {
				t.Fatal("runLifecycle() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.wantText) {
				t.Fatalf("runLifecycle() error = %q, want text %q", err, tt.wantText)
			}
		})
	}
}

type suiteTestProvider struct {
	backend           environment.Backend
	prepareErr        error
	launcher          environment.Launcher
	toolHost          environment.ToolHost
	syncToErr         error
	syncFromErr       error
	destroyErr        error
	syncedToRuntime   bool
	syncedFromRuntime bool
	destroyed         bool
}

func (p *suiteTestProvider) Backend() environment.Backend {
	return p.backend
}

func (p *suiteTestProvider) Prepare(
	_ context.Context,
	req environment.PrepareRequest,
) (environment.Prepared, error) {
	if p.prepareErr != nil {
		return environment.Prepared{}, p.prepareErr
	}
	launcher := p.launcher
	if launcher == nil && p.toolHost == nil {
		launcher = suiteTestLauncher{}
	}
	toolHost := p.toolHost
	if toolHost == nil && p.launcher == nil {
		toolHost = suiteTestToolHost{}
	}
	return environment.Prepared{
		State: environment.SessionState{
			EnvironmentID:  req.EnvironmentID,
			Backend:        environment.BackendLocal,
			RuntimeRootDir: req.LocalRootDir,
		},
		RuntimeRootDir: req.LocalRootDir,
		Launcher:       launcher,
		ToolHost:       toolHost,
	}, nil
}

func (p *suiteTestProvider) SyncToRuntime(
	context.Context,
	environment.SessionState,
	environment.SyncOptions,
) (environment.SyncResult, error) {
	if p.syncToErr != nil {
		return environment.SyncResult{}, p.syncToErr
	}
	p.syncedToRuntime = true
	return environment.SyncResult{}, nil
}

func (p *suiteTestProvider) SyncFromRuntime(
	context.Context,
	environment.SessionState,
	environment.SyncOptions,
) (environment.SyncResult, error) {
	if p.syncFromErr != nil {
		return environment.SyncResult{}, p.syncFromErr
	}
	p.syncedFromRuntime = true
	return environment.SyncResult{}, nil
}

func (p *suiteTestProvider) Destroy(context.Context, environment.SessionState) error {
	if p.destroyErr != nil {
		return p.destroyErr
	}
	p.destroyed = true
	return nil
}

type suiteTestLauncher struct{}

func (suiteTestLauncher) Launch(context.Context, environment.LaunchSpec) (environment.Handle, error) {
	return nil, nil
}

type suiteTestToolHost struct{}

func (suiteTestToolHost) ReadTextFile(context.Context, string) (string, error) {
	return "", nil
}

func (suiteTestToolHost) WriteTextFile(context.Context, string, string) error {
	return nil
}

func (suiteTestToolHost) ResolvePath(path string) (string, error) {
	return path, nil
}

func (suiteTestToolHost) Authorize(environment.PermissionOperation) error {
	return nil
}

func (suiteTestToolHost) PermissionDecision(
	acpsdk.RequestPermissionRequest,
) (environment.PermissionDecision, bool) {
	return environment.PermissionDecisionAllowOnce, false
}

func (suiteTestToolHost) CreateTerminal(
	context.Context,
	acpsdk.CreateTerminalRequest,
) (acpsdk.CreateTerminalResponse, error) {
	return acpsdk.CreateTerminalResponse{}, nil
}

func (suiteTestToolHost) KillTerminal(string) error {
	return nil
}

func (suiteTestToolHost) TerminalOutput(string) (string, error) {
	return "", nil
}

func (suiteTestToolHost) WaitForTerminalExit(context.Context, string) (int, error) {
	return 0, nil
}

func (suiteTestToolHost) ReleaseTerminal(string) error {
	return nil
}
