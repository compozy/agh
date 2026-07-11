package spec

import (
	"reflect"

	"github.com/compozy/agh/internal/api/contract"
	"github.com/getkin/kin-openapi/openapi3"
)

var schemaCustomizers = map[reflect.Type]func(*openapi3.Schema){
	reflect.TypeFor[binaryResponse](): func(schema *openapi3.Schema) {
		*schema = *openapi3.NewStringSchema()
		schema.Format = "binary"
	},
	reflect.TypeFor[contract.LoopGraph](): customizeLoopGraphSchema,
	rawMessageType: func(schema *openapi3.Schema) {
		*schema = *openapi3.NewSchema()
	},
	reflect.TypeFor[contract.BridgeProviderConfigPayload](): func(schema *openapi3.Schema) {
		*schema = *bridgeProviderConfigSchema()
	},
	reflect.TypeFor[contract.BridgeDeliveryDefaultsPayload](): func(schema *openapi3.Schema) {
		*schema = *bridgeDeliveryDefaultsSchema()
	},
	reflect.TypeFor[contract.NetworkSendRequest](): customizeNetworkSendRequestSchema,
	reflect.TypeFor[contract.TaskPayload]():        describeTaskBlockedReasonsProperty,
	reflect.TypeFor[contract.TaskSummaryPayload](): describeTaskBlockedReasonsProperty,
}
