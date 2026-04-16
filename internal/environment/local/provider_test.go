package local

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"reflect"
	"testing"
	"time"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/environment"
	"github.com/pedronauck/agh/internal/environment/providertest"
)

func TestLocalProviderBackend(t *testing.T) {
	t.Parallel()

	provider := NewProvider()
	if got := provider.Backend(); got != environment.BackendLocal {
		t.Fatalf("Backend() = %q, want %q", got, environment.BackendLocal)
	}
}

func TestLocalProviderPrepareReturnsLocalRuntime(t *testing.T) {
	t.Parallel()

	req := newTestPrepareRequest(t)
	provider := NewProvider(WithPermissionMode(aghconfig.PermissionModeDenyAll))

	prepared, err := provider.Prepare(context.Background(), req)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	closePreparedToolHost(t, prepared)

	assertPreparedMatchesRequest(t, prepared, req)
	if prepared.State.PreparedAt.IsZero() {
		t.Fatal("Prepared.State.PreparedAt is zero, want preparation timestamp")
	}
	if prepared.Launcher == nil {
		t.Fatal("Prepared.Launcher = nil, want local launcher")
	}
	if prepared.ToolHost == nil {
		t.Fatal("Prepared.ToolHost = nil, want local tool host")
	}

	if err := prepared.ToolHost.WriteTextFile(context.Background(), "nested/file.txt", "local content"); err != nil {
		t.Fatalf("Prepared.ToolHost.WriteTextFile() error = %v", err)
	}
	content, err := prepared.ToolHost.ReadTextFile(context.Background(), "nested/file.txt")
	if err != nil {
		t.Fatalf("Prepared.ToolHost.ReadTextFile() error = %v", err)
	}
	if content != "local content" {
		t.Fatalf("Prepared.ToolHost.ReadTextFile() = %q, want %q", content, "local content")
	}
}

func TestLocalProviderPrepareClonesMutableInputs(t *testing.T) {
	t.Parallel()

	req := newTestPrepareRequest(t)
	provider := NewProvider(WithPermissionMode(aghconfig.PermissionModeApproveAll))

	prepared, err := provider.Prepare(context.Background(), req)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	closePreparedToolHost(t, prepared)

	req.LocalAdditionalDirs[0] = "/mutated"
	req.AgentEnv[0] = "MUTATED=true"
	req.ProviderState[0] = '['

	if got := prepared.RuntimeAdditionalDirs[0]; got == "/mutated" {
		t.Fatal("Prepared.RuntimeAdditionalDirs aliased request LocalAdditionalDirs")
	}
	if got := prepared.Launch.AdditionalDirs[0]; got == "/mutated" {
		t.Fatal("Prepared.Launch.AdditionalDirs aliased request LocalAdditionalDirs")
	}
	if got := prepared.Launch.Env[0]; got == "MUTATED=true" {
		t.Fatal("Prepared.Launch.Env aliased request AgentEnv")
	}
	if got := string(prepared.State.ProviderState); got != `{"sandbox":"local"}` {
		t.Fatalf("Prepared.State.ProviderState = %s, want original provider state", got)
	}
}

func TestLocalProviderPreparePreservesNilMutableInputs(t *testing.T) {
	t.Parallel()

	req := newTestPrepareRequest(t)
	req.LocalAdditionalDirs = nil
	req.AgentEnv = nil
	req.ProviderState = nil
	provider := NewProvider(WithPermissionMode(aghconfig.PermissionModeApproveAll))

	prepared, err := provider.Prepare(context.Background(), req)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	closePreparedToolHost(t, prepared)

	if prepared.RuntimeAdditionalDirs != nil {
		t.Fatalf("Prepared.RuntimeAdditionalDirs = %#v, want nil", prepared.RuntimeAdditionalDirs)
	}
	if prepared.State.RuntimeAdditionalDirs != nil {
		t.Fatalf("Prepared.State.RuntimeAdditionalDirs = %#v, want nil", prepared.State.RuntimeAdditionalDirs)
	}
	if prepared.Launch.AdditionalDirs != nil {
		t.Fatalf("Prepared.Launch.AdditionalDirs = %#v, want nil", prepared.Launch.AdditionalDirs)
	}
	if prepared.Launch.Env != nil {
		t.Fatalf("Prepared.Launch.Env = %#v, want nil", prepared.Launch.Env)
	}
	if prepared.State.ProviderState != nil {
		t.Fatalf("Prepared.State.ProviderState = %s, want nil", prepared.State.ProviderState)
	}
}

func TestLocalProviderOptions(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	provider := NewProvider(
		WithLogger(logger),
		WithStopTimeout(25*time.Millisecond),
		WithPermissionMode(aghconfig.PermissionModeApproveAll),
	)
	concrete, ok := provider.(*localProvider)
	if !ok {
		t.Fatalf("NewProvider() type = %T, want *localProvider", provider)
	}
	if concrete.logger != logger {
		t.Fatal("WithLogger() did not set provider logger")
	}
	if concrete.stopTimeout != 25*time.Millisecond {
		t.Fatalf("WithStopTimeout() = %v, want %v", concrete.stopTimeout, 25*time.Millisecond)
	}
	if concrete.permissionMode != aghconfig.PermissionModeApproveAll {
		t.Fatalf(
			"WithPermissionMode() = %q, want %q",
			concrete.permissionMode,
			aghconfig.PermissionModeApproveAll,
		)
	}

	provider = NewProvider(WithLogger(nil))
	concrete, ok = provider.(*localProvider)
	if !ok {
		t.Fatalf("NewProvider(WithLogger(nil)) type = %T, want *localProvider", provider)
	}
	if concrete.logger == nil {
		t.Fatal("NewProvider(WithLogger(nil)) logger = nil, want default logger")
	}
}

func TestLocalProviderPrepareReturnsToolHostErrors(t *testing.T) {
	t.Parallel()

	req := newTestPrepareRequest(t)
	req.Permissions = "invalid"
	provider := NewProvider()

	prepared, err := provider.Prepare(context.Background(), req)
	if err == nil {
		closePreparedToolHost(t, prepared)
		t.Fatal("Prepare() error = nil, want invalid permission mode error")
	}
}

func TestLocalProviderNoopLifecycleMethods(t *testing.T) {
	t.Parallel()

	provider := NewProvider()
	state := environment.SessionState{
		EnvironmentID:         "env-local",
		Backend:               environment.BackendLocal,
		RuntimeRootDir:        t.TempDir(),
		RuntimeAdditionalDirs: []string{t.TempDir()},
	}

	if result, err := provider.SyncToRuntime(context.Background(), state, environment.SyncOptions{
		Reason: environment.SyncReasonStart,
	}); err != nil {
		t.Fatalf("SyncToRuntime() error = %v", err)
	} else if result.FilesSynced != 0 || result.BytesTransferred != 0 || len(result.Errors) != 0 {
		t.Fatalf("SyncToRuntime() result = %#v, want empty result", result)
	}
	if result, err := provider.SyncFromRuntime(context.Background(), state, environment.SyncOptions{
		Reason: environment.SyncReasonStop,
	}); err != nil {
		t.Fatalf("SyncFromRuntime() error = %v", err)
	} else if result.FilesSynced != 0 || result.BytesTransferred != 0 || len(result.Errors) != 0 {
		t.Fatalf("SyncFromRuntime() result = %#v, want empty result", result)
	}
	if err := provider.Destroy(context.Background(), state); err != nil {
		t.Fatalf("Destroy() error = %v", err)
	}
}

func TestLocalProviderRegistryResolvesLocalDefault(t *testing.T) {
	t.Parallel()

	registry, err := NewRegistry(WithPermissionMode(aghconfig.PermissionModeApproveAll))
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	provider, err := registry.Provider(environment.BackendLocal)
	if err != nil {
		t.Fatalf("Provider(%q) error = %v", environment.BackendLocal, err)
	}
	if got := provider.Backend(); got != environment.BackendLocal {
		t.Fatalf("Provider(%q).Backend() = %q, want %q", environment.BackendLocal, got, environment.BackendLocal)
	}

	defaultProvider, err := registry.DefaultProvider()
	if err != nil {
		t.Fatalf("DefaultProvider() error = %v", err)
	}
	if got := defaultProvider.Backend(); got != environment.BackendLocal {
		t.Fatalf("DefaultProvider().Backend() = %q, want %q", got, environment.BackendLocal)
	}
}

func TestLocalProviderLifecycleCompliance(t *testing.T) {
	t.Parallel()

	req := newTestPrepareRequest(t)
	provider := NewProvider(WithPermissionMode(aghconfig.PermissionModeApproveAll))

	prepared := providertest.RunLifecycle(t, providertest.LifecycleCase{
		Provider:       provider,
		Backend:        environment.BackendLocal,
		PrepareRequest: req,
		AssertPrepared: func(t *testing.T, prepared environment.Prepared) {
			t.Helper()
			assertPreparedMatchesRequest(t, prepared, req)
		},
		AssertFinalState: func(t *testing.T, state environment.SessionState) {
			t.Helper()
			if state.Backend != environment.BackendLocal {
				t.Fatalf("final state backend = %q, want %q", state.Backend, environment.BackendLocal)
			}
		},
	})
	closePreparedToolHost(t, prepared)
}

func newTestPrepareRequest(t *testing.T) environment.PrepareRequest {
	t.Helper()

	return environment.PrepareRequest{
		SessionID:           "sess-local",
		WorkspaceID:         "workspace-local",
		EnvironmentID:       "env-local",
		LocalRootDir:        t.TempDir(),
		LocalAdditionalDirs: []string{t.TempDir(), t.TempDir()},
		Environment: environment.Resolved{
			Profile:  "local-dev",
			Backend:  environment.BackendLocal,
			SyncMode: environment.SyncModeNone,
		},
		AgentCommand:  "sh -c 'cat'",
		AgentEnv:      []string{"AGH_SESSION_ID=sess-local", "CUSTOM=value"},
		Permissions:   string(aghconfig.PermissionModeApproveAll),
		ProviderState: json.RawMessage(`{"sandbox":"local"}`),
	}
}

func assertPreparedMatchesRequest(
	t *testing.T,
	prepared environment.Prepared,
	req environment.PrepareRequest,
) {
	t.Helper()

	if prepared.RuntimeRootDir != req.LocalRootDir {
		t.Fatalf("Prepared.RuntimeRootDir = %q, want %q", prepared.RuntimeRootDir, req.LocalRootDir)
	}
	if !reflect.DeepEqual(prepared.RuntimeAdditionalDirs, req.LocalAdditionalDirs) {
		t.Fatalf(
			"Prepared.RuntimeAdditionalDirs = %#v, want %#v",
			prepared.RuntimeAdditionalDirs,
			req.LocalAdditionalDirs,
		)
	}
	if prepared.State.RuntimeRootDir != req.LocalRootDir {
		t.Fatalf("Prepared.State.RuntimeRootDir = %q, want %q", prepared.State.RuntimeRootDir, req.LocalRootDir)
	}
	if !reflect.DeepEqual(prepared.State.RuntimeAdditionalDirs, req.LocalAdditionalDirs) {
		t.Fatalf(
			"Prepared.State.RuntimeAdditionalDirs = %#v, want %#v",
			prepared.State.RuntimeAdditionalDirs,
			req.LocalAdditionalDirs,
		)
	}
	if prepared.State.EnvironmentID != req.EnvironmentID {
		t.Fatalf("Prepared.State.EnvironmentID = %q, want %q", prepared.State.EnvironmentID, req.EnvironmentID)
	}
	if prepared.State.Backend != environment.BackendLocal {
		t.Fatalf("Prepared.State.Backend = %q, want %q", prepared.State.Backend, environment.BackendLocal)
	}
	if prepared.State.Profile != req.Environment.Profile {
		t.Fatalf("Prepared.State.Profile = %q, want %q", prepared.State.Profile, req.Environment.Profile)
	}
	if string(prepared.State.ProviderState) != string(req.ProviderState) {
		t.Fatalf("Prepared.State.ProviderState = %s, want %s", prepared.State.ProviderState, req.ProviderState)
	}
	if prepared.Launch.Command != req.AgentCommand {
		t.Fatalf("Prepared.Launch.Command = %q, want %q", prepared.Launch.Command, req.AgentCommand)
	}
	if prepared.Launch.Cwd != req.LocalRootDir {
		t.Fatalf("Prepared.Launch.Cwd = %q, want %q", prepared.Launch.Cwd, req.LocalRootDir)
	}
	if !reflect.DeepEqual(prepared.Launch.AdditionalDirs, req.LocalAdditionalDirs) {
		t.Fatalf(
			"Prepared.Launch.AdditionalDirs = %#v, want %#v",
			prepared.Launch.AdditionalDirs,
			req.LocalAdditionalDirs,
		)
	}
	if !reflect.DeepEqual(prepared.Launch.Env, req.AgentEnv) {
		t.Fatalf("Prepared.Launch.Env = %#v, want %#v", prepared.Launch.Env, req.AgentEnv)
	}
}

func closePreparedToolHost(t *testing.T, prepared environment.Prepared) {
	t.Helper()

	closer, ok := prepared.ToolHost.(interface{ Close() })
	if !ok {
		return
	}
	t.Cleanup(closer.Close)
}
