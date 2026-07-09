package daemon

import (
	"context"
	"errors"
	"fmt"
	"strings"

	aghconfig "github.com/compozy/agh/internal/config"
	looppkg "github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/session"
	workspacepkg "github.com/compozy/agh/internal/workspace"
)

type loopSessionAgentResolver interface {
	ResolveAgent(name string, resolved *workspacepkg.ResolvedWorkspace) (aghconfig.AgentDef, error)
}

type loopSessionPolicyGate struct {
	workspaceResolver workspacepkg.RuntimeResolver
	agentResolver     loopSessionAgentResolver
}

func (g *loopSessionPolicyGate) apply(
	ctx context.Context,
	opts *session.CreateOpts,
	agentName string,
	allowedTools []string,
) error {
	if opts == nil {
		return errors.New("daemon: loop session options are required")
	}
	resolved, err := g.resolveWorkspace(ctx, *opts)
	if err != nil {
		return err
	}
	policy, err := g.resolvePolicy(agentName, &resolved)
	if err != nil {
		return err
	}
	applySessionSandboxPolicy(opts, policy)
	applySessionPermissionPolicy(opts, policy)
	if err := applyAllowedToolsNarrowing(opts, allowedTools); err != nil {
		return err
	}
	return nil
}

func (g *loopSessionPolicyGate) resolveWorkspace(
	ctx context.Context,
	opts session.CreateOpts,
) (workspacepkg.ResolvedWorkspace, error) {
	if g == nil || g.workspaceResolver == nil {
		return workspacepkg.ResolvedWorkspace{}, workspacepkg.ErrWorkspaceResolverUnavailable
	}
	if workspaceID := strings.TrimSpace(opts.Workspace); workspaceID != "" {
		resolved, err := g.workspaceResolver.Resolve(ctx, workspaceID)
		if err != nil {
			return workspacepkg.ResolvedWorkspace{}, fmt.Errorf(
				"daemon: resolve loop session workspace %q: %w",
				workspaceID,
				err,
			)
		}
		return resolved, nil
	}
	workspacePath := strings.TrimSpace(opts.WorkspacePath)
	if workspacePath == "" {
		return workspacepkg.ResolvedWorkspace{}, errors.New("daemon: loop session workspace is required")
	}
	resolved, err := g.workspaceResolver.ResolveOrRegister(ctx, workspacePath)
	if err != nil {
		return workspacepkg.ResolvedWorkspace{}, fmt.Errorf(
			"daemon: resolve loop session workspace path %q: %w",
			workspacePath,
			err,
		)
	}
	return resolved, nil
}

func (g *loopSessionPolicyGate) resolvePolicy(
	agentName string,
	resolved *workspacepkg.ResolvedWorkspace,
) (SessionPolicy, error) {
	if resolved == nil {
		return SessionPolicy{}, errors.New("daemon: resolved loop session workspace is required")
	}
	agentDef, err := g.resolveAgent(agentName, resolved)
	if err != nil {
		return SessionPolicy{}, fmt.Errorf(
			"%w: resolve loop session agent policy for %q: %w",
			looppkg.ErrValidation,
			strings.TrimSpace(agentName),
			err,
		)
	}
	resolvedAgent, err := resolved.Config.ResolveAgent(agentDef)
	if err != nil {
		return SessionPolicy{}, fmt.Errorf(
			"%w: resolve loop session policy for %q: %w",
			looppkg.ErrValidation,
			strings.TrimSpace(agentName),
			err,
		)
	}
	return sessionPolicyFromResolvedAgentWorkspace(resolvedAgent, resolved), nil
}

func (g *loopSessionPolicyGate) resolveAgent(
	agentName string,
	resolved *workspacepkg.ResolvedWorkspace,
) (aghconfig.AgentDef, error) {
	target := strings.TrimSpace(agentName)
	if target == "" {
		return aghconfig.AgentDef{}, errors.New("daemon: loop session agent name is required")
	}
	if g != nil && g.agentResolver != nil {
		return g.agentResolver.ResolveAgent(target, resolved)
	}
	for _, agent := range resolved.Agents {
		if strings.TrimSpace(agent.Name) == target {
			return agent, nil
		}
	}
	return aghconfig.AgentDef{}, fmt.Errorf("%w: %s", workspacepkg.ErrAgentNotAvailable, target)
}

func sessionPolicyFromResolvedAgentWorkspace(
	agent aghconfig.ResolvedAgent,
	resolved *workspacepkg.ResolvedWorkspace,
) SessionPolicy {
	policy := SessionPolicy{
		Runtime: SessionRuntimePolicy{
			Permissions: aghconfig.PermissionMode(strings.TrimSpace(agent.Permissions)),
		},
	}
	if sandboxRef := resolvedWorkspaceSandboxRef(resolved); sandboxRef != "" {
		policy.Sandbox = SessionSandboxPolicy{
			Mode:       SessionSandboxModeRef,
			SandboxRef: sandboxRef,
		}
	}
	return policy
}

func resolvedWorkspaceSandboxRef(resolved *workspacepkg.ResolvedWorkspace) string {
	if resolved == nil {
		return ""
	}
	if sandboxRef := strings.TrimSpace(resolved.SandboxRef); sandboxRef != "" {
		return sandboxRef
	}
	return strings.TrimSpace(resolved.Config.Defaults.Sandbox)
}
