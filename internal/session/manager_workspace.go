package session

import (
	"context"
	"errors"
	"fmt"
	"strings"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/store"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func (m *Manager) resolveCreateWorkspace(ctx context.Context, opts CreateOpts) (workspacepkg.ResolvedWorkspace, error) {
	resolver, err := m.requireWorkspaceResolver()
	if err != nil {
		return workspacepkg.ResolvedWorkspace{}, err
	}

	workspaceRef := strings.TrimSpace(opts.Workspace)
	workspacePath := strings.TrimSpace(opts.WorkspacePath)
	switch {
	case workspaceRef == "" && workspacePath == "":
		return workspacepkg.ResolvedWorkspace{}, errors.New("session: workspace or workspace path is required")
	case workspaceRef != "" && workspacePath != "":
		return workspacepkg.ResolvedWorkspace{}, errors.New(
			"session: workspace and workspace path are mutually exclusive",
		)
	case workspacePath != "":
		resolved, err := resolver.ResolveOrRegister(ctx, workspacePath)
		if err != nil {
			return workspacepkg.ResolvedWorkspace{}, fmt.Errorf(
				"session: resolve workspace path %q: %w",
				workspacePath,
				err,
			)
		}
		return resolved, nil
	default:
		resolved, err := resolver.Resolve(ctx, workspaceRef)
		if err != nil {
			return workspacepkg.ResolvedWorkspace{}, fmt.Errorf("session: resolve workspace %q: %w", workspaceRef, err)
		}
		return resolved, nil
	}
}

func (m *Manager) resolveResumeWorkspace(
	ctx context.Context,
	meta store.SessionMeta,
) (workspacepkg.ResolvedWorkspace, error) {
	resolver, err := m.requireWorkspaceResolver()
	if err != nil {
		return workspacepkg.ResolvedWorkspace{}, err
	}

	workspaceID := strings.TrimSpace(meta.WorkspaceID)
	if workspaceID == "" {
		return workspacepkg.ResolvedWorkspace{}, errors.New("session: session workspace id is required")
	}

	resolved, err := resolver.Resolve(ctx, workspaceID)
	if err != nil {
		return workspacepkg.ResolvedWorkspace{}, fmt.Errorf(
			"session: resolve workspace %q for session %q: %w",
			workspaceID,
			meta.ID,
			err,
		)
	}
	return resolved, nil
}

func (m *Manager) requireWorkspaceResolver() (workspacepkg.RuntimeResolver, error) {
	if m.workspace == nil {
		return nil, errors.New("session: workspace resolver is required")
	}
	return m.workspace, nil
}

func resolveWorkspaceAgent(
	agentName string,
	resolvedWorkspace *workspacepkg.ResolvedWorkspace,
) (aghconfig.AgentDef, error) {
	target := strings.TrimSpace(agentName)
	if target == "" {
		return aghconfig.AgentDef{}, errors.New("session: agent name is required")
	}
	if resolvedWorkspace == nil {
		return aghconfig.AgentDef{}, errors.New("session: resolved workspace is required")
	}

	for _, agent := range resolvedWorkspace.Agents {
		if strings.TrimSpace(agent.Name) != target {
			continue
		}
		return agent, nil
	}

	return aghconfig.AgentDef{}, fmt.Errorf("%w: %s", workspacepkg.ErrAgentNotAvailable, target)
}

func (m *Manager) resolveWorkspaceAgent(
	agentName string,
	resolvedWorkspace *workspacepkg.ResolvedWorkspace,
) (aghconfig.AgentDef, error) {
	if m != nil && m.agentResolver != nil {
		return m.agentResolver.ResolveAgent(agentName, resolvedWorkspace)
	}
	return resolveWorkspaceAgent(agentName, resolvedWorkspace)
}

func (m *Manager) resolveWorkspaceSessionAgent(
	agentName string,
	provider string,
	resolvedWorkspace *workspacepkg.ResolvedWorkspace,
) (aghconfig.ResolvedAgent, error) {
	agentDef, err := m.resolveWorkspaceAgent(agentName, resolvedWorkspace)
	if err != nil {
		return aghconfig.ResolvedAgent{}, err
	}

	resolved, err := resolvedWorkspace.Config.ResolveSessionAgent(agentDef, provider)
	if err != nil {
		return aghconfig.ResolvedAgent{}, err
	}
	return resolved, nil
}
