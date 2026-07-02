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

const nativeNetworkChannelUpdateRequiredFields = "purpose, fanout_policy, or coordinator_peer_id"

type networkChannelCreateInput struct {
	WorkspaceID       string `json:"workspace_id"`
	Channel           string `json:"channel"`
	Purpose           string `json:"purpose"`
	FanoutPolicy      string `json:"fanout_policy,omitempty"`
	CoordinatorPeerID string `json:"coordinator_peer_id,omitempty"`
}

type networkChannelUpdateInput struct {
	WorkspaceID       string  `json:"workspace_id"`
	Channel           string  `json:"channel"`
	Purpose           *string `json:"purpose,omitempty"`
	FanoutPolicy      *string `json:"fanout_policy,omitempty"`
	CoordinatorPeerID *string `json:"coordinator_peer_id,omitempty"`
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
	fanoutPolicy := store.NormalizeNetworkFanoutPolicy(input.FanoutPolicy)
	coordinatorPeerID := strings.TrimSpace(input.CoordinatorPeerID)
	if err := store.ValidateNetworkChannelFanoutConfiguration(fanoutPolicy, coordinatorPeerID); err != nil {
		return toolspkg.ToolResult{}, nativeNetworkInputError(req.ToolID, err)
	}
	workspaceID, err := n.nativeNetworkWorkspaceID(ctx, req.ToolID, input.WorkspaceID, scope)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	entry := store.NetworkChannelEntry{
		Channel:           channel,
		WorkspaceID:       workspaceID,
		Purpose:           purpose,
		FanoutPolicy:      fanoutPolicy,
		CoordinatorPeerID: coordinatorPeerID,
		CreatedBy:         strings.TrimSpace(scope.AgentName),
	}
	if err := n.deps.NetworkStore.WriteNetworkChannel(ctx, entry); err != nil {
		return toolspkg.ToolResult{}, nativeNetworkInputError(req.ToolID, err)
	}
	return structuredNetworkResult(
		nativeNetworkChannelPayload(entry),
		"channel "+channel,
	)
}

func nativeNetworkChannelPayload(entry store.NetworkChannelEntry) map[string]any {
	return map[string]any{
		"channel":             strings.TrimSpace(entry.Channel),
		"workspace_id":        strings.TrimSpace(entry.WorkspaceID),
		"purpose":             strings.TrimSpace(entry.Purpose),
		"fanout_policy":       store.NormalizeNetworkFanoutPolicy(entry.FanoutPolicy),
		"coordinator_peer_id": strings.TrimSpace(entry.CoordinatorPeerID),
	}
}

func (n *daemonNativeTools) networkChannelUpdate(
	ctx context.Context,
	scope toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input networkChannelUpdateInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	channel, err := nativeNetworkChannel(req.ToolID, input.Channel)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	if input.Purpose == nil && input.FanoutPolicy == nil && input.CoordinatorPeerID == nil {
		return toolspkg.ToolResult{}, nativeRequiredInputError(req.ToolID, nativeNetworkChannelUpdateRequiredFields)
	}
	workspaceID, err := n.nativeNetworkWorkspaceID(ctx, req.ToolID, input.WorkspaceID, scope)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	ref := store.NetworkChannelRef{WorkspaceID: workspaceID, Channel: channel}
	patch := store.NetworkChannelPatch{}
	if input.Purpose != nil {
		purpose := strings.TrimSpace(*input.Purpose)
		patch.Purpose = &purpose
	}
	if input.FanoutPolicy != nil {
		policy := strings.TrimSpace(*input.FanoutPolicy)
		fanoutPolicy := store.NormalizeNetworkFanoutPolicy(policy)
		if err := store.ValidateNetworkFanoutPolicy(fanoutPolicy); err != nil {
			return toolspkg.ToolResult{}, nativeNetworkInputError(req.ToolID, err)
		}
		patch.FanoutPolicy = &fanoutPolicy
	}
	if input.CoordinatorPeerID != nil {
		coordinatorPeerID := strings.TrimSpace(*input.CoordinatorPeerID)
		patch.CoordinatorPeerID = &coordinatorPeerID
	}
	if err := n.deps.NetworkStore.PatchNetworkChannel(ctx, ref, patch); err != nil {
		return toolspkg.ToolResult{}, nativeNetworkInputError(req.ToolID, err)
	}
	entry, err := n.deps.NetworkStore.GetNetworkChannel(ctx, ref)
	if err != nil {
		return toolspkg.ToolResult{}, nativeNetworkInputError(req.ToolID, err)
	}
	return structuredNetworkResult(nativeNetworkChannelPayload(entry), "channel "+channel)
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
		aghconfig.NormalizeAgentName(scope.AgentName) == aghconfig.OnboardingAgentName {
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
