package daemon

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	aghconfig "github.com/pedronauck/agh/internal/config"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func TestCoordinatorConfigResolverReturnsBundledDefaultIdentity(t *testing.T) {
	t.Parallel()

	cfg := defaultCoordinatorResolverConfig(t)
	resolver := newCoordinatorConfigResolver(&cfg, nil, nil)

	resolved, err := resolver.ResolveCoordinatorConfig(context.Background(), "")
	if err != nil {
		t.Fatalf("ResolveCoordinatorConfig() error = %v", err)
	}
	if got, want := resolved.AgentName, aghconfig.DefaultCoordinatorAgentName; got != want {
		t.Fatalf("ResolveCoordinatorConfig() AgentName = %q, want %q", got, want)
	}
	if resolved.Enabled {
		t.Fatal("ResolveCoordinatorConfig() Enabled = true, want bundled default false")
	}
	if got, want := resolved.DefaultTTL, aghconfig.DefaultCoordinatorTTL; got != want {
		t.Fatalf("ResolveCoordinatorConfig() DefaultTTL = %s, want %s", got, want)
	}
}

func TestCoordinatorConfigResolverPrefersGlobalConfigOverBundledDefault(t *testing.T) {
	t.Parallel()

	cfg := defaultCoordinatorResolverConfig(t)
	cfg.Autonomy.Coordinator.Enabled = true
	cfg.Autonomy.Coordinator.AgentName = "global-coordinator"
	cfg.Autonomy.Coordinator.Provider = "codex"
	cfg.Autonomy.Coordinator.Model = "global-model"
	cfg.Autonomy.Coordinator.DefaultTTL = 4 * time.Hour
	cfg.Autonomy.Coordinator.MaxChildren = 4
	resolver := newCoordinatorConfigResolver(&cfg, nil, nil)

	resolved, err := resolver.ResolveCoordinatorConfig(context.Background(), "")
	if err != nil {
		t.Fatalf("ResolveCoordinatorConfig() error = %v", err)
	}
	if !resolved.Enabled {
		t.Fatal("ResolveCoordinatorConfig() Enabled = false, want global true")
	}
	if got, want := resolved.AgentName, "global-coordinator"; got != want {
		t.Fatalf("ResolveCoordinatorConfig() AgentName = %q, want %q", got, want)
	}
	if got, want := resolved.Provider, "codex"; got != want {
		t.Fatalf("ResolveCoordinatorConfig() Provider = %q, want %q", got, want)
	}
	if got, want := resolved.Model, "global-model"; got != want {
		t.Fatalf("ResolveCoordinatorConfig() Model = %q, want %q", got, want)
	}
	if got, want := resolved.DefaultTTL, 4*time.Hour; got != want {
		t.Fatalf("ResolveCoordinatorConfig() DefaultTTL = %s, want %s", got, want)
	}
	if got, want := resolved.MaxChildren, 4; got != want {
		t.Fatalf("ResolveCoordinatorConfig() MaxChildren = %d, want %d", got, want)
	}
}

func TestCoordinatorConfigResolverPrefersWorkspaceConfig(t *testing.T) {
	t.Parallel()

	global := defaultCoordinatorResolverConfig(t)
	global.Autonomy.Coordinator.Enabled = true
	global.Autonomy.Coordinator.Provider = "claude"
	global.Autonomy.Coordinator.Model = "global-model"
	global.Autonomy.Coordinator.DefaultTTL = 2 * time.Hour
	global.Autonomy.Coordinator.MaxChildren = 5

	workspaceCfg := global
	workspaceCfg.Autonomy.Coordinator.Enabled = false
	workspaceCfg.Autonomy.Coordinator.AgentName = "workspace-coordinator"
	workspaceCfg.Autonomy.Coordinator.Provider = "codex"
	workspaceCfg.Autonomy.Coordinator.Model = "workspace-model"
	workspaceCfg.Autonomy.Coordinator.DefaultTTL = 3 * time.Hour
	workspaceCfg.Autonomy.Coordinator.MaxChildren = 2

	resolver := newCoordinatorConfigResolver(
		&global,
		&coordinatorWorkspaceResolverStub{
			resolved: workspacepkg.ResolvedWorkspace{
				Workspace: workspacepkg.Workspace{ID: "ws-1"},
				Config:    workspaceCfg,
			},
		},
		nil,
	)

	resolved, err := resolver.ResolveCoordinatorConfig(context.Background(), "ws-1")
	if err != nil {
		t.Fatalf("ResolveCoordinatorConfig(workspace) error = %v", err)
	}
	if resolved.Enabled {
		t.Fatal("ResolveCoordinatorConfig() Enabled = true, want workspace false")
	}
	if got, want := resolved.AgentName, "workspace-coordinator"; got != want {
		t.Fatalf("ResolveCoordinatorConfig() AgentName = %q, want %q", got, want)
	}
	if got, want := resolved.Provider, "codex"; got != want {
		t.Fatalf("ResolveCoordinatorConfig() Provider = %q, want %q", got, want)
	}
	if got, want := resolved.Model, "workspace-model"; got != want {
		t.Fatalf("ResolveCoordinatorConfig() Model = %q, want %q", got, want)
	}
	if got, want := resolved.DefaultTTL, 3*time.Hour; got != want {
		t.Fatalf("ResolveCoordinatorConfig() DefaultTTL = %s, want %s", got, want)
	}
	if got, want := resolved.MaxChildren, 2; got != want {
		t.Fatalf("ResolveCoordinatorConfig() MaxChildren = %d, want %d", got, want)
	}
}

func TestCoordinatorConfigResolverUsesAgentFallbackForProviderModel(t *testing.T) {
	t.Parallel()

	cfg := defaultCoordinatorResolverConfig(t)
	cfg.Defaults.Provider = "codex"
	resolver := newCoordinatorConfigResolver(
		&cfg,
		nil,
		coordinatorAgentResolverStub{
			agent: aghconfig.AgentDef{
				Name:     aghconfig.DefaultCoordinatorAgentName,
				Provider: "claude",
				Model:    "agent-model",
				Prompt:   "agent fallback",
			},
		},
	)

	resolved, err := resolver.ResolveCoordinatorConfig(context.Background(), "")
	if err != nil {
		t.Fatalf("ResolveCoordinatorConfig() error = %v", err)
	}
	if got, want := resolved.Provider, "claude"; got != want {
		t.Fatalf("ResolveCoordinatorConfig() Provider = %q, want agent fallback %q", got, want)
	}
	if got, want := resolved.Model, "agent-model"; got != want {
		t.Fatalf("ResolveCoordinatorConfig() Model = %q, want agent fallback %q", got, want)
	}
}

type coordinatorWorkspaceResolverStub struct {
	resolved workspacepkg.ResolvedWorkspace
	err      error
}

func (s *coordinatorWorkspaceResolverStub) Resolve(
	context.Context,
	string,
) (workspacepkg.ResolvedWorkspace, error) {
	if s.err != nil {
		return workspacepkg.ResolvedWorkspace{}, s.err
	}
	return s.resolved, nil
}

func (s *coordinatorWorkspaceResolverStub) ResolveOrRegister(
	context.Context,
	string,
) (workspacepkg.ResolvedWorkspace, error) {
	if s.err != nil {
		return workspacepkg.ResolvedWorkspace{}, s.err
	}
	return s.resolved, nil
}

type coordinatorAgentResolverStub struct {
	agent aghconfig.AgentDef
	err   error
}

func (s coordinatorAgentResolverStub) ResolveAgent(
	string,
	*workspacepkg.ResolvedWorkspace,
) (aghconfig.AgentDef, error) {
	if s.err != nil {
		return aghconfig.AgentDef{}, s.err
	}
	return s.agent, nil
}

func defaultCoordinatorResolverConfig(t *testing.T) aghconfig.Config {
	t.Helper()

	homePaths, err := aghconfig.ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	return aghconfig.DefaultWithHome(homePaths)
}
