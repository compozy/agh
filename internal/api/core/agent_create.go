package core

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/compozy/agh/internal/api/contract"
	aghconfig "github.com/compozy/agh/internal/config"
	workspacepkg "github.com/compozy/agh/internal/workspace"
)

// CreateAgentFromRequest validates and persists one AGENT.md definition from a
// shared create-agent request. It is the single authoring path reused by the
// HTTP handler and the agh__agent_create native tool.
func CreateAgentFromRequest(
	ctx context.Context,
	req contract.CreateAgentRequest,
	homePaths aghconfig.HomePaths,
	workspaces WorkspaceService,
	transportName string,
) (aghconfig.AgentDef, error) {
	draft, err := createAgentDraftFromRequest(req)
	if err != nil {
		return aghconfig.AgentDef{}, err
	}
	path, err := createAgentDefinitionPathFor(ctx, req, homePaths, workspaces, transportName)
	if err != nil {
		return aghconfig.AgentDef{}, err
	}
	return aghconfig.CreateAgentDefFile(path, draft, false)
}

func createAgentDefinitionPathFor(
	ctx context.Context,
	req contract.CreateAgentRequest,
	homePaths aghconfig.HomePaths,
	workspaces WorkspaceService,
	transportName string,
) (string, error) {
	name := aghconfig.NormalizeAgentName(req.Agent.Name)
	switch req.Scope {
	case contract.AgentCreateScopeGlobal:
		return filepath.Join(homePaths.AgentsDir, name, aghconfig.AgentDefinitionFileName), nil
	case contract.AgentCreateScopeWorkspace:
		workspaceRef := strings.TrimSpace(req.Workspace)
		if workspaceRef == "" {
			return "", errors.Join(
				errCreateAgentRequestInvalid,
				errors.New("workspace is required for workspace-scoped agents"),
			)
		}
		if workspaces == nil {
			return "", fmt.Errorf("%s: %w", transportName, workspacepkg.ErrWorkspaceResolverUnavailable)
		}
		resolved, err := workspaces.Resolve(ctx, workspaceRef)
		if err != nil {
			return "", err
		}
		rootDir := strings.TrimSpace(resolved.RootDir)
		if rootDir == "" {
			return "", fmt.Errorf("%s: %w", transportName, workspacepkg.ErrWorkspaceRootMissing)
		}
		return filepath.Join(
			rootDir,
			aghconfig.DirName,
			aghconfig.AgentsDirName,
			name,
			aghconfig.AgentDefinitionFileName,
		), nil
	default:
		return "", errors.Join(
			errCreateAgentRequestInvalid,
			fmt.Errorf(
				"scope must be %q or %q",
				contract.AgentCreateScopeWorkspace,
				contract.AgentCreateScopeGlobal,
			),
		)
	}
}
