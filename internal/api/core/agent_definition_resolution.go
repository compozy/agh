package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/compozy/agh/internal/api/contract"
	aghconfig "github.com/compozy/agh/internal/config"
	workspacepkg "github.com/compozy/agh/internal/workspace"
)

type resolvedAgentDefinition struct {
	Entry              AgentCatalogEntry
	OperationWorkspace string
	WorkspaceRoot      string
}

func (h *BaseHandlers) resolveAgentDefinition(
	ctx context.Context,
	workspaceRef string,
	name string,
) (resolvedAgentDefinition, error) {
	target := aghconfig.NormalizeAgentName(name)
	if target == "" {
		return resolvedAgentDefinition{}, errors.Join(
			errAgentDefinitionInvalid,
			errors.New("agent name is required"),
		)
	}

	workspaceRef = strings.TrimSpace(workspaceRef)
	if workspaceRef != "" {
		if h.Workspaces == nil {
			return resolvedAgentDefinition{}, fmt.Errorf(
				"api: %w",
				workspacepkg.ErrWorkspaceResolverUnavailable,
			)
		}
		resolved, err := h.Workspaces.Resolve(ctx, workspaceRef)
		if err != nil {
			return resolvedAgentDefinition{}, err
		}
		for _, agent := range resolved.Agents {
			if aghconfig.NormalizeAgentName(agent.Name) != target {
				continue
			}
			entry := h.agentCatalogEntryFromDef(agent, resolved.ID)
			return resolvedAgentDefinition{
				Entry:              entry,
				OperationWorkspace: strings.TrimSpace(resolved.ID),
				WorkspaceRoot:      strings.TrimSpace(resolved.RootDir),
			}, nil
		}
		return resolvedAgentDefinition{}, fmt.Errorf(
			"api: agent %q is not available in workspace %q: %w",
			target,
			workspaceRef,
			workspacepkg.ErrAgentNotAvailable,
		)
	}

	agent, err := h.AgentLoader(target, h.HomePaths)
	if err != nil {
		return resolvedAgentDefinition{}, err
	}
	return resolvedAgentDefinition{
		Entry: h.agentCatalogEntryFromDef(agent, ""),
	}, nil
}

func (h *BaseHandlers) duplicateAgentTarget(
	ctx context.Context,
	req contract.DuplicateAgentRequest,
	source resolvedAgentDefinition,
) (string, contract.AgentOrigin, string, error) {
	scope := req.Scope
	if scope == "" {
		scope = contract.AgentCreateScope(source.Entry.Origin)
	}
	targetName := aghconfig.NormalizeAgentName(req.Name)
	switch scope {
	case contract.AgentCreateScopeGlobal:
		return filepath.Join(h.HomePaths.AgentsDir, targetName), contract.AgentOriginGlobal, "", nil
	case contract.AgentCreateScopeWorkspace:
		workspaceRef := strings.TrimSpace(req.Workspace)
		if workspaceRef == "" && source.Entry.Origin == contract.AgentOriginWorkspace {
			if source.WorkspaceRoot == "" {
				return "", "", "", errors.Join(
					errAgentDefinitionInvalid,
					errors.New("source workspace root is unavailable"),
				)
			}
			return filepath.Join(
				source.WorkspaceRoot,
				aghconfig.DirName,
				aghconfig.AgentsDirName,
				targetName,
			), contract.AgentOriginWorkspace, source.OperationWorkspace, nil
		}
		if workspaceRef == "" {
			return "", "", "", errors.Join(
				errAgentDefinitionInvalid,
				errors.New("workspace is required for workspace-scoped duplicate"),
			)
		}
		if h.Workspaces == nil {
			return "", "", "", fmt.Errorf(
				"api: %w",
				workspacepkg.ErrWorkspaceResolverUnavailable,
			)
		}
		resolved, err := h.Workspaces.Resolve(ctx, workspaceRef)
		if err != nil {
			return "", "", "", err
		}
		return filepath.Join(
			resolved.RootDir,
			aghconfig.DirName,
			aghconfig.AgentsDirName,
			targetName,
		), contract.AgentOriginWorkspace, strings.TrimSpace(resolved.ID), nil
	default:
		return "", "", "", errors.Join(
			errAgentDefinitionInvalid,
			fmt.Errorf("scope must be %q or %q", contract.AgentCreateScopeGlobal, contract.AgentCreateScopeWorkspace),
		)
	}
}

func (h *BaseHandlers) globalAgentTwinExists(name string, effectiveSourcePath string) bool {
	path := filepath.Join(
		h.HomePaths.AgentsDir,
		aghconfig.NormalizeAgentName(name),
		aghconfig.AgentDefinitionFileName,
	)
	if filepath.Clean(path) == filepath.Clean(effectiveSourcePath) {
		return false
	}
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	if err != nil {
		h.Logger.Debug("api: global agent twin is unavailable for disclosure", "path", path, "error", err)
		return false
	}
	if !info.Mode().IsRegular() {
		return false
	}
	agent, err := aghconfig.LoadAgentDefFile(path)
	if err != nil {
		h.Logger.Debug("api: global agent twin is invalid for disclosure", "path", path, "error", err)
		return false
	}
	return aghconfig.NormalizeAgentName(agent.Name) == aghconfig.NormalizeAgentName(name)
}
