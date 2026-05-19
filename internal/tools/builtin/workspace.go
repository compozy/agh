package builtin

import toolspkg "github.com/pedronauck/agh/internal/tools"

const (
	workspaceWorkspaceKey = "workspace"
)

const (
	workspaceListKey = "list"
)

var workspaceTools = []toolspkg.Descriptor{
	nativeDescriptor(
		toolspkg.ToolIDWorkspaceList,
		"workspace_list",
		"Workspace List",
		"List registered workspaces through the existing workspace discovery surface.",
		emptyInputSchema,
		toolspkg.RiskRead,
		true,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDWorkspace},
		[]string{workspaceWorkspaceKey, workspaceListKey},
		[]string{"workspace list", "registered workspaces"},
	),
	nativeDescriptor(
		toolspkg.ToolIDWorkspaceInfo,
		"workspace_info",
		"Workspace Info",
		"Read one registered workspace record.",
		workspaceRefInputSchema,
		toolspkg.RiskRead,
		true,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDWorkspace},
		[]string{workspaceWorkspaceKey, "info"},
		[]string{"workspace info", "workspace record"},
	),
	nativeDescriptor(
		toolspkg.ToolIDWorkspaceDescribe,
		"workspace_describe",
		"Workspace Describe",
		"Read one resolved workspace detail projection with sessions, agents, skills, and providers.",
		workspaceRefInputSchema,
		toolspkg.RiskRead,
		true,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDWorkspace},
		[]string{workspaceWorkspaceKey, "describe"},
		[]string{"workspace describe", "workspace detail"},
	),
}

func workspaceDescriptors() []toolspkg.Descriptor {
	return workspaceTools
}

const workspaceRefInputSchema = `{
	"type":"object",
	"required":["workspace"],
	"properties":{
		"workspace":{"type":"string"}
	},
	"additionalProperties":false
}`
