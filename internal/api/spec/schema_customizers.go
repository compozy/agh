package spec

import (
	"reflect"

	"github.com/compozy/agh/internal/api/contract"
	"github.com/compozy/agh/internal/network/participation"
	"github.com/getkin/kin-openapi/openapi3"
)

var putNetworkCoordinationInvitationRequestType = reflect.TypeFor[contract.PutNetworkCoordinationInvitationRequest]()

var schemaCustomizers = map[reflect.Type]func(*openapi3.Schema){
	reflect.TypeFor[binaryResponse](): func(schema *openapi3.Schema) {
		*schema = *openapi3.NewStringSchema()
		schema.Format = "binary"
	},
	reflect.TypeFor[contract.LoopGraph]():                  customizeLoopGraphSchema,
	reflect.TypeFor[contract.DesktopStateEntry]():          customizeDesktopStateEntrySchema,
	reflect.TypeFor[contract.DesktopStateSubscribeFrame](): customizeDesktopStateFrameSchema("sub"),
	reflect.TypeFor[contract.DesktopStateApplyFrame]():     customizeDesktopStateFrameSchema("apply"),
	reflect.TypeFor[contract.DesktopStatePingFrame]():      customizeDesktopStateFrameSchema("ping"),
	reflect.TypeFor[contract.DesktopStateSnapshotFrame]():  customizeDesktopStateFrameSchema("snapshot"),
	reflect.TypeFor[contract.DesktopStateEventFrame]():     customizeDesktopStateFrameSchema("event"),
	reflect.TypeFor[contract.DesktopStateAckFrame]():       customizeDesktopStateFrameSchema("ack"),
	reflect.TypeFor[contract.DesktopStateErrorFrame]():     customizeDesktopStateFrameSchema("error"),
	reflect.TypeFor[contract.DesktopStatePongFrame]():      customizeDesktopStateFrameSchema("pong"),
	reflect.TypeFor[contract.DesktopStateSafeNumber](): func(schema *openapi3.Schema) {
		*schema = *openapi3.NewIntegerSchema().
			WithMin(0).
			WithMax(float64(contract.DesktopStateMaxSafeNumber))
	},
	reflect.TypeFor[contract.SettingsMCPSecretInputPayload]():  customizeSettingsMCPSecretInputSchema,
	reflect.TypeFor[contract.SettingsMCPAuthExchangeRequest](): customizeSettingsMCPAuthExchangeRequestSchema,
	rawMessageType: func(schema *openapi3.Schema) {
		*schema = *openapi3.NewSchema()
	},
	reflect.TypeFor[contract.BridgeProviderConfigPayload](): func(schema *openapi3.Schema) {
		*schema = *bridgeProviderConfigSchema()
	},
	reflect.TypeFor[contract.BridgeDeliveryDefaultsPayload](): func(schema *openapi3.Schema) {
		*schema = *bridgeDeliveryDefaultsSchema()
	},
	reflect.TypeFor[contract.NetworkSendRequest]():              customizeNetworkSendRequestSchema,
	reflect.TypeFor[contract.NetworkSubscriptionRequest]():      customizeClosedObjectSchema,
	reflect.TypeFor[contract.PromoteNetworkThreadTaskRequest](): customizeClosedObjectSchema,
	reflect.TypeFor[contract.PutNetworkCoordinationRequest]():   customizePutNetworkCoordinationRequestSchema,
	putNetworkCoordinationInvitationRequestType:                 customizePutNetworkCoordinationInvitationRequestSchema,
	reflect.TypeFor[contract.TaskPayload]():                     describeTaskBlockedReasonsProperty,
	reflect.TypeFor[contract.TaskSummaryPayload]():              describeTaskBlockedReasonsProperty,
	reflect.TypeFor[participation.Request]():                    customizeParticipationRequestSchema,
	reflect.TypeFor[participation.Spec]():                       customizeParticipationSpecSchema,
}

func customizeClosedObjectSchema(schema *openapi3.Schema) {
	if schema != nil {
		schema.WithoutAdditionalProperties()
	}
}
