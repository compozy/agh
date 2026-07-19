package spec

import (
	"fmt"

	"github.com/compozy/agh/internal/api/contract"
	"github.com/getkin/kin-openapi/openapi3"
)

var desktopStateComponentSchemas = []struct {
	name  string
	value any
}{
	{name: "DesktopStateEntry", value: contract.DesktopStateEntry{}},
	{name: "DesktopStateListResponse", value: contract.DesktopStateListResponse{}},
	{name: "DesktopStatePutRequest", value: contract.DesktopStatePutRequest{}},
	{name: "DesktopStateApplyOp", value: contract.DesktopStateApplyOp{}},
	{name: "DesktopStateApplyRequest", value: contract.DesktopStateApplyRequest{}},
	{name: "DesktopStateApplyResponse", value: contract.DesktopStateApplyResponse{}},
	{name: "DesktopStateErrorPayload", value: contract.DesktopStateErrorPayload{}},
	{name: "DesktopStateSubscribeFrame", value: contract.DesktopStateSubscribeFrame{}},
	{name: "DesktopStateApplyFrame", value: contract.DesktopStateApplyFrame{}},
	{name: "DesktopStatePingFrame", value: contract.DesktopStatePingFrame{}},
	{name: "DesktopStateSnapshotFrame", value: contract.DesktopStateSnapshotFrame{}},
	{name: "DesktopStateEventFrame", value: contract.DesktopStateEventFrame{}},
	{name: "DesktopStateAckResult", value: contract.DesktopStateAckResult{}},
	{name: "DesktopStateAckFrame", value: contract.DesktopStateAckFrame{}},
	{name: "DesktopStateErrorFrame", value: contract.DesktopStateErrorFrame{}},
	{name: "DesktopStatePongFrame", value: contract.DesktopStatePongFrame{}},
	{name: "DesktopStateWebSocketContract", value: contract.DesktopStateWebSocketContract{}},
}

func registerDesktopStateComponentSchemas(schemas openapi3.Schemas) error {
	for _, component := range desktopStateComponentSchemas {
		schemaRef, err := schemaRefForValue(component.value, schemas)
		if err != nil {
			return fmt.Errorf("build %s: %w", component.name, err)
		}
		schemas[component.name] = schemaRef
	}
	return nil
}

func customizeDesktopStateEntrySchema(schema *openapi3.Schema) {
	if schema == nil {
		return
	}
	value := schema.Properties["value"]
	if value != nil && value.Value != nil {
		value.Value.Nullable = true
	}
}

func customizeDesktopStateFrameSchema(op string) func(*openapi3.Schema) {
	return func(schema *openapi3.Schema) {
		if schema == nil {
			return
		}
		opSchema := schema.Properties["op"]
		if opSchema != nil && opSchema.Value != nil {
			setStringEnum(opSchema.Value, []string{op})
		}
	}
}
