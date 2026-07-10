package core

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/compozy/agh/internal/api/contract"
	aghconfig "github.com/compozy/agh/internal/config"
	"github.com/compozy/agh/internal/session"
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

func createAgentDraftFromRequest(req contract.CreateAgentRequest) (aghconfig.AgentDefinitionDraft, error) {
	agent := req.Agent
	agentName := aghconfig.NormalizeAgentName(agent.Name)
	if agentName == "" {
		return aghconfig.AgentDefinitionDraft{}, errors.Join(
			errCreateAgentRequestInvalid,
			errors.New("agent.name is required"),
		)
	}
	if err := aghconfig.ValidatePublicAgentName(agentName); err != nil {
		return aghconfig.AgentDefinitionDraft{}, errors.Join(
			errCreateAgentRequestInvalid,
			aghconfig.ErrInvalidAgentDefinition,
			err,
		)
	}
	if strings.TrimSpace(agent.Provider) == "" {
		return aghconfig.AgentDefinitionDraft{}, errors.Join(
			errCreateAgentRequestInvalid,
			errors.New("agent.provider is required"),
		)
	}
	if strings.TrimSpace(agent.Prompt) == "" {
		return aghconfig.AgentDefinitionDraft{}, errors.Join(
			errCreateAgentRequestInvalid,
			errors.New("agent.prompt is required"),
		)
	}
	if err := session.ValidateReasoningEffort(string(agent.ReasoningEffort)); err != nil {
		return aghconfig.AgentDefinitionDraft{}, errors.Join(
			errCreateAgentRequestInvalid,
			aghconfig.ErrInvalidAgentDefinition,
			err,
		)
	}
	disabledSkills := []string(nil)
	if agent.Skills != nil {
		disabledSkills = append([]string(nil), agent.Skills.Disabled...)
	}
	return aghconfig.AgentDefinitionDraft{
		Name:            agentName,
		Provider:        agent.Provider,
		Command:         agent.Command,
		Model:           agent.Model,
		ReasoningEffort: string(agent.ReasoningEffort),
		Tools:           append([]string(nil), agent.Tools...),
		Toolsets:        append([]string(nil), agent.Toolsets...),
		DenyTools:       append([]string(nil), agent.DenyTools...),
		Permissions:     string(agent.Permissions),
		Skills:          aghconfig.AgentSkillsConfig{Disabled: disabledSkills},
		CategoryPath:    append([]string(nil), agent.CategoryPath...),
		Prompt:          agent.Prompt,
	}, nil
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
