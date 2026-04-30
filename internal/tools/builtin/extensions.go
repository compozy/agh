package builtin

import toolspkg "github.com/pedronauck/agh/internal/tools"

var extensionTools = []toolspkg.Descriptor{
	nativeExtensionDescriptor(
		toolspkg.ToolIDExtensionsSearch,
		"extensions_search",
		"Extensions Search",
		"Search configured extension marketplace sources.",
		extensionSearchInputSchema,
		toolspkg.RiskRead,
		true,
		false,
		[]string{"extensions", "marketplace", "catalog"},
		[]string{"extension marketplace search", "find extensions"},
	),
	nativeExtensionDescriptor(
		toolspkg.ToolIDExtensionsList,
		"extensions_list",
		"Extensions List",
		"List installed extensions through the daemon extension registry.",
		emptyInputSchema,
		toolspkg.RiskRead,
		true,
		false,
		[]string{"extensions", "installed", "catalog"},
		[]string{"installed extensions", "extension status"},
	),
	nativeExtensionDescriptor(
		toolspkg.ToolIDExtensionsInfo,
		"extensions_info",
		"Extensions Info",
		"Read one installed extension status and runtime projection.",
		extensionNameInputSchema,
		toolspkg.RiskRead,
		true,
		false,
		[]string{"extensions", "status"},
		[]string{"extension info", "extension status"},
	),
	nativeExtensionDescriptor(
		toolspkg.ToolIDExtensionsInstall,
		"extensions_install",
		"Extensions Install",
		"Install one extension through the managed local or marketplace lifecycle.",
		extensionInstallInputSchema,
		toolspkg.RiskMutating,
		false,
		false,
		[]string{"extensions", "install", "marketplace", "mutation"},
		[]string{"install extension", "add extension"},
	),
	nativeExtensionDescriptor(
		toolspkg.ToolIDExtensionsUpdate,
		"extensions_update",
		"Extensions Update",
		"Update marketplace-installed extensions through the managed lifecycle.",
		extensionUpdateInputSchema,
		toolspkg.RiskMutating,
		false,
		false,
		[]string{"extensions", "update", "marketplace", "mutation"},
		[]string{"update extension", "upgrade extension"},
	),
	nativeExtensionDescriptor(
		toolspkg.ToolIDExtensionsRemove,
		"extensions_remove",
		"Extensions Remove",
		"Remove one managed installed extension with rollback on reload failure.",
		extensionNameInputSchema,
		toolspkg.RiskDestructive,
		false,
		true,
		[]string{"extensions", "remove", "mutation"},
		[]string{"remove extension", "uninstall extension"},
	),
	nativeExtensionDescriptor(
		toolspkg.ToolIDExtensionsEnable,
		"extensions_enable",
		"Extensions Enable",
		"Enable one installed extension through the runtime extension lifecycle.",
		extensionNameInputSchema,
		toolspkg.RiskMutating,
		false,
		false,
		[]string{"extensions", "enable", "mutation"},
		[]string{"enable extension", "activate extension"},
	),
	nativeExtensionDescriptor(
		toolspkg.ToolIDExtensionsDisable,
		"extensions_disable",
		"Extensions Disable",
		"Disable one installed extension through the runtime extension lifecycle.",
		extensionNameInputSchema,
		toolspkg.RiskMutating,
		false,
		false,
		[]string{"extensions", "disable", "mutation"},
		[]string{"disable extension", "deactivate extension"},
	),
}

func extensionDescriptors() []toolspkg.Descriptor {
	return extensionTools
}

func nativeExtensionDescriptor(
	id toolspkg.ToolID,
	nativeName string,
	title string,
	description string,
	inputSchema string,
	risk toolspkg.RiskClass,
	readOnly bool,
	destructive bool,
	tags []string,
	searchHints []string,
) toolspkg.Descriptor {
	return nativeDescriptor(
		id,
		nativeName,
		title,
		description,
		inputSchema,
		risk,
		readOnly,
		destructive,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDExtensions},
		tags,
		searchHints,
	)
}

const extensionSearchInputSchema = `{
	"type":"object",
	"required":["query"],
	"properties":{
		"query":{"type":"string"},
		"source":{"type":"string"},
		"limit":{"type":"integer"}
	},
	"additionalProperties":false
}`

const extensionNameInputSchema = `{
	"type":"object",
	"required":["name"],
	"properties":{
		"name":{"type":"string"}
	},
	"additionalProperties":false
}`

const extensionInstallInputSchema = `{
	"type":"object",
	"properties":{
		"source":{"type":"string","enum":["local","marketplace"]},
		"path":{"type":"string"},
		"checksum":{"type":"string"},
		"slug":{"type":"string"},
		"registry":{"type":"string"},
		"version":{"type":"string"},
		"asset":{"type":"string"}
	},
	"additionalProperties":false
}`

const extensionUpdateInputSchema = `{
	"type":"object",
	"properties":{
		"name":{"type":"string"},
		"all":{"type":"boolean"},
		"check_only":{"type":"boolean"}
	},
	"additionalProperties":false
}`
