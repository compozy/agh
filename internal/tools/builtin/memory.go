package builtin

import toolspkg "github.com/pedronauck/agh/internal/tools"

var memoryTools = []toolspkg.Descriptor{
	nativeDescriptor(
		toolspkg.ToolIDMemoryList,
		"memory_list",
		"Memory List",
		"List memory headers through the current memory store.",
		memoryListInputSchema,
		toolspkg.RiskRead,
		true,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDMemory},
		[]string{"memory", "list"},
		[]string{"memory list", "memory headers"},
	),
	nativeDescriptor(
		toolspkg.ToolIDMemoryRead,
		"memory_read",
		"Memory Read",
		"Read one memory document through the current memory store.",
		memoryReadInputSchema,
		toolspkg.RiskRead,
		true,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDMemory},
		[]string{"memory", "read"},
		[]string{"memory read", "memory document"},
	),
	nativeDescriptor(
		toolspkg.ToolIDMemorySearch,
		"memory_search",
		"Memory Search",
		"Search memory documents through the current memory store.",
		memorySearchInputSchema,
		toolspkg.RiskRead,
		true,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDMemory},
		[]string{"memory", "search"},
		[]string{"memory search", "recall memory"},
	),
	nativeDescriptor(
		toolspkg.ToolIDMemoryHistory,
		"memory_history",
		"Memory History",
		"Read redacted memory operation history.",
		memoryHistoryInputSchema,
		toolspkg.RiskRead,
		true,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDMemory},
		[]string{"memory", "history"},
		[]string{"memory history", "memory operations"},
	),
}

func memoryDescriptors() []toolspkg.Descriptor {
	return memoryTools
}

const memoryListInputSchema = `{
	"type":"object",
	"properties":{
		"scope":{"type":"string"},
		"workspace":{"type":"string"},
		"limit":{"type":"integer"}
	},
	"additionalProperties":false
}`

const memoryReadInputSchema = `{
	"type":"object",
	"required":["filename"],
	"properties":{
		"filename":{"type":"string"},
		"scope":{"type":"string"},
		"workspace":{"type":"string"}
	},
	"additionalProperties":false
}`

const memorySearchInputSchema = `{
	"type":"object",
	"properties":{
		"query":{"type":"string"},
		"q":{"type":"string"},
		"scope":{"type":"string"},
		"workspace":{"type":"string"},
		"limit":{"type":"integer"}
	},
	"additionalProperties":false
}`

const memoryHistoryInputSchema = `{
	"type":"object",
	"properties":{
		"scope":{"type":"string"},
		"workspace":{"type":"string"},
		"operation":{"type":"string"},
		"since":{"type":"string"},
		"limit":{"type":"integer"}
	},
	"additionalProperties":false
}`
