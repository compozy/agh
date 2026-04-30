package daemon

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	toolspkg "github.com/pedronauck/agh/internal/tools"
)

type mcpAuthStatusInput struct {
	ServerName string `json:"server_name"`
}

type mcpAuthStatusPayload struct {
	Status      toolspkg.MCPAuthStatus `json:"status"`
	RepairPaths mcpAuthRepairPaths     `json:"repair_paths"`
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
		toolspkg.ToolIDMCPAuthStatus: {
			call:         n.mcpAuthStatus,
			availability: availability,
		},
	}
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
		SettingsHTTP: "/api/settings/mcp-servers",
		SettingsUDS:  "/api/settings/mcp-servers",
		Note:         "Login and logout remain management-only and are not exposed as tool calls.",
	}
}
