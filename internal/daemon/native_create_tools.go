package daemon

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/api/contract"
	core "github.com/compozy/agh/internal/api/core"
	aghconfig "github.com/compozy/agh/internal/config"
	"github.com/compozy/agh/internal/store"
	toolspkg "github.com/compozy/agh/internal/tools"
)

type networkChannelCreateInput struct {
	WorkspaceID string `json:"workspace_id"`
	Channel     string `json:"channel"`
	Purpose     string `json:"purpose"`
}

type agentCreateInput struct {
	Scope          string   `json:"scope"`
	Workspace      string   `json:"workspace,omitempty"`
	Name           string   `json:"name"`
	Provider       string   `json:"provider"`
	Model          string   `json:"model,omitempty"`
	Command        string   `json:"command,omitempty"`
	Prompt         string   `json:"prompt"`
	Permissions    string   `json:"permissions,omitempty"`
	Tools          []string `json:"tools,omitempty"`
	Toolsets       []string `json:"toolsets,omitempty"`
	DenyTools      []string `json:"deny_tools,omitempty"`
	CategoryPath   []string `json:"category_path,omitempty"`
	DisabledSkills []string `json:"disabled_skills,omitempty"`
}

func (n *daemonNativeTools) networkChannelCreate(
	ctx context.Context,
	scope toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input networkChannelCreateInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	channel, err := nativeNetworkChannel(req.ToolID, input.Channel)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	purpose := strings.TrimSpace(input.Purpose)
	if purpose == "" {
		return toolspkg.ToolResult{}, nativeRequiredInputError(req.ToolID, "purpose")
	}
	workspaceID, err := n.nativeNetworkWorkspaceID(ctx, req.ToolID, input.WorkspaceID, scope)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	entry := store.NetworkChannelEntry{
		Channel:     channel,
		WorkspaceID: workspaceID,
		Purpose:     purpose,
		CreatedBy:   strings.TrimSpace(scope.AgentName),
	}
	if err := n.deps.NetworkStore.WriteNetworkChannel(ctx, entry); err != nil {
		return toolspkg.ToolResult{}, nativeNetworkInputError(req.ToolID, err)
	}
	return structuredNetworkResult(
		map[string]any{"channel": channel, "workspace_id": workspaceID, "purpose": purpose},
		"channel "+channel,
	)
}

func (n *daemonNativeTools) agentCreate(
	ctx context.Context,
	scope toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input agentCreateInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	createReq, err := n.agentCreateRequest(req.ToolID, scope, input)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	agent, err := core.CreateAgentFromRequest(
		ctx,
		createReq,
		n.deps.HomePaths,
		n.deps.Workspaces,
		string(req.ToolID),
	)
	if err != nil {
		return toolspkg.ToolResult{}, nativeAgentCreateToolError(req.ToolID, err)
	}
	payload := core.AgentPayloadFromDef(agent)
	return structuredResult(map[string]any{"agent": payload}, "agent "+payload.Name)
}

func (n *daemonNativeTools) agentCreateRequest(
	id toolspkg.ToolID,
	scope toolspkg.Scope,
	input agentCreateInput,
) (contract.CreateAgentRequest, error) {
	createReq := contract.CreateAgentRequest{
		Scope:     contract.AgentCreateScope(strings.TrimSpace(input.Scope)),
		Workspace: strings.TrimSpace(input.Workspace),
		Agent: contract.CreateAgentPayload{
			Name:         strings.TrimSpace(input.Name),
			Provider:     strings.TrimSpace(input.Provider),
			Command:      strings.TrimSpace(input.Command),
			Model:        strings.TrimSpace(input.Model),
			Prompt:       input.Prompt,
			Permissions:  contract.SettingsPermissionMode(strings.TrimSpace(input.Permissions)),
			Tools:        trimNativeStrings(input.Tools),
			Toolsets:     trimNativeStrings(input.Toolsets),
			DenyTools:    trimNativeStrings(input.DenyTools),
			CategoryPath: trimNativeStrings(input.CategoryPath),
		},
	}
	if len(input.DisabledSkills) > 0 {
		createReq.Agent.Skills = &contract.CreateAgentSkillsConfig{
			Disabled: trimNativeStrings(input.DisabledSkills),
		}
	}
	// The bundled onboarding agent runs with approve-all over its toolsets, so a prompt-injection
	// attempt could try to author a global-scope agent. Pin it to workspace-scoped authoring.
	if createReq.Scope == contract.AgentCreateScopeGlobal &&
		strings.TrimSpace(scope.AgentName) == aghconfig.OnboardingAgentName {
		return contract.CreateAgentRequest{}, toolspkg.NewToolError(
			toolspkg.ErrorCodeDenied,
			id,
			"the onboarding agent may only author workspace-scoped agents",
			toolspkg.ErrToolDenied,
			toolspkg.ReasonScopeMismatch,
		)
	}
	if createReq.Scope == contract.AgentCreateScopeWorkspace {
		workspaceRef, err := nativeCallerWorkspaceInput(id, "workspace", createReq.Workspace, scope)
		if err != nil {
			return contract.CreateAgentRequest{}, err
		}
		if strings.TrimSpace(workspaceRef) == "" {
			return contract.CreateAgentRequest{}, nativeRequiredInputError(id, "workspace")
		}
		createReq.Workspace = workspaceRef
	}
	return createReq, nil
}

func nativeAgentCreateToolError(id toolspkg.ToolID, err error) error {
	switch {
	case errors.Is(err, aghconfig.ErrAgentDefinitionExists):
		return toolspkg.NewToolError(
			toolspkg.ErrorCodeConflict,
			id,
			err.Error(),
			fmt.Errorf("%w: %w", toolspkg.ErrToolConflict, err),
			toolspkg.ReasonConflictedID,
		)
	case errors.Is(err, aghconfig.ErrInvalidAgentDefinition):
		return toolspkg.NewToolError(
			toolspkg.ErrorCodeInvalidInput,
			id,
			err.Error(),
			fmt.Errorf("%w: %w", toolspkg.ErrToolInvalidInput, err),
			toolspkg.ReasonSchemaInvalid,
		)
	default:
		return nativeNetworkInputError(id, err)
	}
}
