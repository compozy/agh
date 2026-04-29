package builtin

import toolspkg "github.com/pedronauck/agh/internal/tools"

var catalogTools = []toolspkg.Descriptor{
	nativeDescriptor(
		toolspkg.ToolIDToolList,
		"tool_list",
		"Tool List",
		"List tools in the caller's effective registry projection.",
		toolListInputSchema,
		toolspkg.RiskRead,
		true,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDBootstrap, toolspkg.ToolsetIDCatalog},
		[]string{"tools", "registry", "catalog"},
		[]string{"available tools", "tool registry"},
	),
	nativeDescriptor(
		toolspkg.ToolIDToolSearch,
		"tool_search",
		"Tool Search",
		"Search tools in the caller's effective registry projection.",
		toolSearchInputSchema,
		toolspkg.RiskRead,
		true,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDBootstrap, toolspkg.ToolsetIDCatalog},
		[]string{"tools", "registry", "catalog"},
		[]string{"find tools", "tool registry search"},
	),
	nativeDescriptor(
		toolspkg.ToolIDToolInfo,
		"tool_info",
		"Tool Info",
		"Read one tool descriptor and effective diagnostics view.",
		toolInfoInputSchema,
		toolspkg.RiskRead,
		true,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDBootstrap, toolspkg.ToolsetIDCatalog},
		[]string{"tools", "registry", "diagnostics"},
		[]string{"tool descriptor", "tool policy diagnostics"},
	),
}

func catalogDescriptors() []toolspkg.Descriptor {
	return catalogTools
}

const toolListInputSchema = `{
	"type":"object",
	"properties":{
		"limit":{"type":"integer"}
	},
	"additionalProperties":false
}`

const toolSearchInputSchema = `{
	"type":"object",
	"required":["query"],
	"properties":{
		"query":{"type":"string"},
		"limit":{"type":"integer"}
	},
	"additionalProperties":false
}`

const toolInfoInputSchema = `{
	"type":"object",
	"required":["tool_id"],
	"properties":{
		"tool_id":{"type":"string"}
	},
	"additionalProperties":false
}`
