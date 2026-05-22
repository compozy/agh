package daemon

import (
	"context"
	"errors"
	"fmt"
	"strings"

	aghconfig "github.com/compozy/agh/internal/config"
	workspacepkg "github.com/compozy/agh/internal/workspace"
)

// CoordinatorConfigResolver resolves coordinator policy without starting coordinator behavior.
type CoordinatorConfigResolver interface {
	ResolveCoordinatorConfig(ctx context.Context, workspaceID string) (aghconfig.CoordinatorConfig, error)
}

type coordinatorAgentResolver interface {
	ResolveAgent(name string, resolved *workspacepkg.ResolvedWorkspace) (aghconfig.AgentDef, error)
}

type defaultCoordinatorConfigResolver struct {
	config            *aghconfig.Config
	workspaceResolver workspacepkg.RuntimeResolver
	agents            coordinatorAgentResolver
}

var _ CoordinatorConfigResolver = (*defaultCoordinatorConfigResolver)(nil)

func newCoordinatorConfigResolver(
	cfg *aghconfig.Config,
	workspaceResolver workspacepkg.RuntimeResolver,
	agents coordinatorAgentResolver,
) CoordinatorConfigResolver {
	return &defaultCoordinatorConfigResolver{
		config:            cfg,
		workspaceResolver: workspaceResolver,
		agents:            agents,
	}
}

func (r *defaultCoordinatorConfigResolver) ResolveCoordinatorConfig(
	ctx context.Context,
	workspaceID string,
) (aghconfig.CoordinatorConfig, error) {
	if ctx == nil {
		return aghconfig.CoordinatorConfig{}, errors.New("daemon: coordinator config context is required")
	}
	if r.config == nil {
		return aghconfig.CoordinatorConfig{}, errors.New("daemon: coordinator config is required")
	}

	cfg := r.config
	var resolvedWorkspace *workspacepkg.ResolvedWorkspace
	if target := strings.TrimSpace(workspaceID); target != "" {
		if r.workspaceResolver == nil {
			return aghconfig.CoordinatorConfig{}, errors.New(
				"daemon: workspace resolver is required for workspace coordinator config",
			)
		}
		resolved, err := r.workspaceResolver.Resolve(ctx, target)
		if err != nil {
			return aghconfig.CoordinatorConfig{}, fmt.Errorf(
				"daemon: resolve coordinator workspace %q: %w",
				target,
				err,
			)
		}
		cfg = &resolved.Config
		resolvedWorkspace = &resolved
	}

	fallback := aghconfig.DefaultCoordinatorAgentDef()
	agentName := strings.TrimSpace(cfg.Autonomy.Coordinator.AgentName)
	if agentName == "" {
		agentName = fallback.Name
	}
	if agentName != "" && r.agents != nil {
		agent, err := r.agents.ResolveAgent(agentName, resolvedWorkspace)
		if err == nil {
			fallback = agent
		} else if !errors.Is(err, workspacepkg.ErrAgentNotAvailable) {
			return aghconfig.CoordinatorConfig{}, fmt.Errorf("daemon: resolve coordinator agent %q: %w", agentName, err)
		}
	}

	resolved, err := cfg.ResolveCoordinatorConfig(fallback)
	if err != nil {
		return aghconfig.CoordinatorConfig{}, fmt.Errorf("daemon: resolve coordinator config: %w", err)
	}
	return resolved, nil
}
