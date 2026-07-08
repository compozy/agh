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

type loopActionAgentResolver interface {
	ResolveAgent(name string, resolved *workspacepkg.ResolvedWorkspace) (aghconfig.AgentDef, error)
}

func (b *loopActionSessionBinder) applyPolicyGate(
	ctx context.Context,
	opts *session.CreateOpts,
	agentName string,
	allowedTools []string,
) error {
	if opts == nil {
		return errors.New("daemon: loop action session options are required")
	}
	resolved, err := b.resolveActionSessionWorkspace(ctx, *opts)
	if err != nil {
		return err
	}
	policy, err := b.resolveActionSessionPolicy(agentName, &resolved)
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

func (b *loopActionSessionBinder) resolveActionSessionWorkspace(
	ctx context.Context,
	opts session.CreateOpts,
) (workspacepkg.ResolvedWorkspace, error) {
	if b == nil || b.workspaceResolver == nil {
		return workspacepkg.ResolvedWorkspace{}, workspacepkg.ErrWorkspaceResolverUnavailable
	}
	if workspaceID := strings.TrimSpace(opts.Workspace); workspaceID != "" {
		resolved, err := b.workspaceResolver.Resolve(ctx, workspaceID)
		if err != nil {
			return workspacepkg.ResolvedWorkspace{}, fmt.Errorf(
				"daemon: resolve loop action workspace %q: %w",
				workspaceID,
				err,
			)
		}
		return resolved, nil
	}
	workspacePath := strings.TrimSpace(opts.WorkspacePath)
	if workspacePath == "" {
		return workspacepkg.ResolvedWorkspace{}, errors.New("daemon: loop action workspace is required")
	}
	resolved, err := b.workspaceResolver.ResolveOrRegister(ctx, workspacePath)
	if err != nil {
		return workspacepkg.ResolvedWorkspace{}, fmt.Errorf(
			"daemon: resolve loop action workspace path %q: %w",
			workspacePath,
			err,
		)
	}
	return resolved, nil
}

func (b *loopActionSessionBinder) resolveActionSessionPolicy(
	agentName string,
	resolved *workspacepkg.ResolvedWorkspace,
) (SessionPolicy, error) {
	if resolved == nil {
		return SessionPolicy{}, errors.New("daemon: resolved loop action workspace is required")
	}
	agentDef, err := b.resolveActionSessionAgent(agentName, resolved)
	if err != nil {
		return SessionPolicy{}, fmt.Errorf(
			"%w: resolve loop action agent policy for %q: %w",
			looppkg.ErrValidation,
			strings.TrimSpace(agentName),
			err,
		)
	}
	resolvedAgent, err := resolved.Config.ResolveAgent(agentDef)
	if err != nil {
		return SessionPolicy{}, fmt.Errorf(
			"%w: resolve loop action session policy for %q: %w",
			looppkg.ErrValidation,
			strings.TrimSpace(agentName),
			err,
		)
	}
	return sessionPolicyFromResolvedAgentWorkspace(resolvedAgent, resolved), nil
}

func (b *loopActionSessionBinder) resolveActionSessionAgent(
	agentName string,
	resolved *workspacepkg.ResolvedWorkspace,
) (aghconfig.AgentDef, error) {
	target := strings.TrimSpace(agentName)
	if target == "" {
		return aghconfig.AgentDef{}, errors.New("daemon: loop action agent name is required")
	}
	if b != nil && b.agentResolver != nil {
		return b.agentResolver.ResolveAgent(target, resolved)
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
