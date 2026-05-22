package daemon

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	toolspkg "github.com/compozy/agh/internal/tools"
)

const (
	nativeMCPAuthToolsAPISettingsMCPServersPath = "/api/settings/mcp-servers"
	nativeMCPCallableDiscoveryNote              = "Auth-blocked MCP tools are omitted from callable discovery; " +
		"use agh__mcp_status or agh__mcp_auth_status for repair detail."
)

type mcpAuthStatusInput struct {
	ServerName string `json:"server_name"`
}

type mcpAuthStatusPayload struct {
	Status      toolspkg.MCPAuthStatus `json:"status"`
	RepairPaths mcpAuthRepairPaths     `json:"repair_paths"`
}

type mcpStatusPayload struct {
	ServerName            string                 `json:"server_name"`
	State                 string                 `json:"state"`
	Auth                  toolspkg.MCPAuthStatus `json:"auth"`
	RepairPaths           mcpAuthRepairPaths     `json:"repair_paths"`
	CallableDiscoveryNote string                 `json:"callable_discovery_note"`
}

type mcpAuthRepairPaths struct {
	StatusCLI    string `json:"status_cli"`
	LoginCLI     string `json:"login_cli"`
	LogoutCLI    string `json:"logout_cli"`
	SettingsHTTP string `json:"settings_http"`
	SettingsUDS  string `json:"settings_uds"`
	Note         string `json:"note"`
}

func (n *daemonNativeTools) mcpAuthToolBindings(
	availability toolspkg.NativeAvailabilityFunc,
) map[toolspkg.ToolID]nativeToolBinding {
	return map[toolspkg.ToolID]nativeToolBinding{
		toolspkg.ToolIDMCPStatus: {
			call:         n.mcpStatus,
			availability: availability,
		},
		toolspkg.ToolIDMCPAuthStatus: {
			call:         n.mcpAuthStatus,
			availability: availability,
		},
	}
}

func (n *daemonNativeTools) mcpStatus(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input mcpAuthStatusInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	serverName, err := requiredNativeString(req.ToolID, "server_name", input.ServerName)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	provider := n.mcpAuthProvider()
	if provider == nil {
		return toolspkg.ToolResult{}, toolspkg.NewToolError(
			toolspkg.ErrorCodeUnavailable,
			req.ToolID,
			"mcp status provider is unavailable",
			toolspkg.ErrToolUnavailable,
			toolspkg.ReasonDependencyMissing,
		)
	}
	status, err := provider.Status(ctx, toolspkg.SourceRef{
		Kind:          toolspkg.SourceMCP,
		Owner:         serverName,
		RawServerName: serverName,
	})
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	if strings.TrimSpace(status.ServerName) == "" {
		status.ServerName = serverName
	}
	state := mcpProbeState(status)
	payload := mcpStatusPayload{
		ServerName:            status.ServerName,
		State:                 state,
		Auth:                  status,
		RepairPaths:           mcpAuthRepairPathsFor(status.ServerName),
		CallableDiscoveryNote: nativeMCPCallableDiscoveryNote,
	}
	return structuredResult(payload, fmt.Sprintf("%s %s", status.ServerName, state))
}

func (n *daemonNativeTools) mcpAuthStatus(
	ctx context.Context,
	_ toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	var input mcpAuthStatusInput
	if err := decodeNativeInput(req, &input); err != nil {
		return toolspkg.ToolResult{}, err
	}
	serverName, err := requiredNativeString(req.ToolID, "server_name", input.ServerName)
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	provider := n.mcpAuthProvider()
	if provider == nil {
		return toolspkg.ToolResult{}, toolspkg.NewToolError(
			toolspkg.ErrorCodeUnavailable,
			req.ToolID,
			"mcp auth status provider is unavailable",
			toolspkg.ErrToolUnavailable,
			toolspkg.ReasonDependencyMissing,
		)
	}
	status, err := provider.Status(ctx, toolspkg.SourceRef{
		Kind:          toolspkg.SourceMCP,
		Owner:         serverName,
		RawServerName: serverName,
	})
	if err != nil {
		return toolspkg.ToolResult{}, err
	}
	if strings.TrimSpace(status.ServerName) == "" {
		status.ServerName = serverName
	}
	payload := mcpAuthStatusPayload{
		Status:      status,
		RepairPaths: mcpAuthRepairPathsFor(status.ServerName),
	}
	return structuredResult(payload, fmt.Sprintf("%s %s", status.ServerName, status.Status))
}

func mcpAuthRepairPathsFor(serverName string) mcpAuthRepairPaths {
	arg := strconv.Quote(strings.TrimSpace(serverName))
	return mcpAuthRepairPaths{
		StatusCLI:    "agh mcp auth status " + arg,
		LoginCLI:     "agh mcp auth login " + arg,
		LogoutCLI:    "agh mcp auth logout " + arg,
		SettingsHTTP: nativeMCPAuthToolsAPISettingsMCPServersPath,
		SettingsUDS:  nativeMCPAuthToolsAPISettingsMCPServersPath,
		Note:         "Login and logout remain management-only and are not exposed as tool calls.",
	}
}

func mcpProbeState(status toolspkg.MCPAuthStatus) string {
	if reason, ok := toolspkg.MCPAuthStatusReason(status); ok {
		switch reason {
		case toolspkg.ReasonMCPAuthRequired,
			toolspkg.ReasonMCPAuthExpired,
			toolspkg.ReasonMCPAuthInvalid,
			toolspkg.ReasonMCPAuthRefreshFailed:
			return "auth-blocked"
		case toolspkg.ReasonMCPAuthUnconfigured:
			return "unavailable"
		}
	}
	return "healthy"
}
