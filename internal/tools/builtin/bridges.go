package builtin

import toolspkg "github.com/pedronauck/agh/internal/tools"

const (
	bridgesBridgesKey = "bridges"
	bridgesHealthKey  = "health"
)

var bridgeTools = []toolspkg.Descriptor{
	nativeDescriptor(
		toolspkg.ToolIDBridgesList,
		"bridges_list",
		"Bridges List",
		"List bridge instances without bridge credentials or secret bindings.",
		emptyInputSchema,
		toolspkg.RiskRead,
		true,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDBridges},
		[]string{bridgesBridgesKey, "list"},
		[]string{"bridge list", "bridge instances"},
	),
	nativeDescriptor(
		toolspkg.ToolIDBridgesStatus,
		"bridges_status",
		"Bridges Status",
		"Read bridge status and health without bridge credentials or secret bindings.",
		bridgesStatusInputSchema,
		toolspkg.RiskRead,
		true,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDBridges},
		[]string{bridgesBridgesKey, "status", bridgesHealthKey},
		[]string{"bridge status", "bridge health"},
	),
}

func bridgeDescriptors() []toolspkg.Descriptor {
	return bridgeTools
}

const bridgesStatusInputSchema = `{
	"type":"object",
	"properties":{
		"bridge_id":{"type":"string"}
	},
	"additionalProperties":false
}`
