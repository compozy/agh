package contract

import (
	"fmt"

	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	"github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/subprocess"
	"github.com/pedronauck/agh/internal/tools"
)

// HookContractSpec binds one hook event to its payload and patch contracts.
type HookContractSpec struct {
	Event   hooks.HookEvent
	Payload NamedType
	Patch   NamedType
}

// SDKRootTypes returns the canonical generated SDK contract roots.
func SDKRootTypes() []NamedType {
	return []NamedType{
		{Name: "InitializeRequest", Value: subprocess.InitializeRequest{}},
		{Name: "InitializeExtension", Value: subprocess.InitializeExtension{}},
		{Name: "InitializeCapabilities", Value: subprocess.InitializeCapabilities{}},
		{Name: "InitializeMethods", Value: subprocess.InitializeMethods{}},
		{Name: "InitializeRuntime", Value: subprocess.InitializeRuntime{}},
		{Name: "InitializeBridgeRuntime", Value: subprocess.InitializeBridgeRuntime{}},
		{Name: "InitializeBridgeBoundSecret", Value: subprocess.InitializeBridgeBoundSecret{}},
		{Name: "InitializeResponse", Value: subprocess.InitializeResponse{}},
		{Name: "InitializeExtensionInfo", Value: subprocess.InitializeExtensionInfo{}},
		{Name: "AcceptedCapabilities", Value: subprocess.AcceptedCapabilities{}},
		{Name: "InitializeSupports", Value: subprocess.InitializeSupports{}},
		{Name: "ShutdownRequest", Value: subprocess.ShutdownRequest{}},
		{Name: "ShutdownResponse", Value: subprocess.ShutdownResponse{}},
		{Name: "BridgeInstance", Value: bridgepkg.BridgeInstance{}},
		{Name: "BridgeStatus", Value: bridgepkg.BridgeStatus("")},
		{Name: "BridgeScope", Value: bridgepkg.Scope("")},
		{Name: "RoutingPolicy", Value: bridgepkg.RoutingPolicy{}},
		{Name: "RoutingKey", Value: bridgepkg.RoutingKey{}},
		{Name: "InboundEventFamily", Value: bridgepkg.InboundEventFamily("")},
		{Name: "InboundMessageEnvelope", Value: bridgepkg.InboundMessageEnvelope{}},
		{Name: "InboundCommand", Value: bridgepkg.InboundCommand{}},
		{Name: "InboundAction", Value: bridgepkg.InboundAction{}},
		{Name: "InboundReaction", Value: bridgepkg.InboundReaction{}},
		{Name: "DeliveryEvent", Value: bridgepkg.DeliveryEvent{}},
		{Name: "DeliveryRequest", Value: bridgepkg.DeliveryRequest{}},
		{Name: "DeliveryAck", Value: bridgepkg.DeliveryAck{}},
		{Name: "DeliverySnapshot", Value: bridgepkg.DeliverySnapshot{}},
		{Name: "DeliveryTarget", Value: bridgepkg.DeliveryTarget{}},
		{Name: "DeliveryMode", Value: bridgepkg.DeliveryMode("")},
		{Name: "DeliveryOperation", Value: bridgepkg.DeliveryOperation("")},
		{Name: "DeliveryMessageReference", Value: bridgepkg.DeliveryMessageReference{}},
		{Name: "DeliveryErrorDetail", Value: bridgepkg.DeliveryErrorDetail{}},
		{Name: "DeliveryResumeState", Value: bridgepkg.DeliveryResumeState{}},
		{Name: "MessageSender", Value: bridgepkg.MessageSender{}},
		{Name: "MessageContent", Value: bridgepkg.MessageContent{}},
		{Name: "MessageAttachment", Value: bridgepkg.MessageAttachment{}},
		{Name: "Tool", Value: tools.Tool{}},
		{Name: "MemoryScope", Value: memory.Scope("")},
		{Name: "HookEventFamily", Value: hooks.HookEventFamily("")},
		{Name: "HookRunOutcome", Value: hooks.HookRunOutcome("")},
		{Name: "HookSkillSource", Value: hooks.HookSkillSource("")},
		{Name: "PayloadBase", Value: hooks.PayloadBase{}},
		{Name: "SessionContext", Value: hooks.SessionContext{}},
		{Name: "TurnContext", Value: hooks.TurnContext{}},
		{Name: "ContextBlock", Value: hooks.ContextBlock{}},
		{Name: "ToolCallRef", Value: hooks.ToolCallRef{}},
		{Name: "ToolLocation", Value: hooks.ToolLocation{}},
		{Name: "PermissionOption", Value: hooks.PermissionOption{}},
		{Name: "PermissionToolCall", Value: hooks.PermissionToolCall{}},
		{Name: "ControlPatch", Value: hooks.ControlPatch{}},
		{Name: "SessionLifecyclePayload", Value: hooks.SessionLifecyclePayload{}},
		{Name: "SessionCreatePatch", Value: hooks.SessionCreatePatch{}},
		{Name: "InputPreSubmitPayload", Value: hooks.InputPreSubmitPayload{}},
		{Name: "InputPreSubmitPatch", Value: hooks.InputPreSubmitPatch{}},
		{Name: "PromptPayload", Value: hooks.PromptPayload{}},
		{Name: "PromptPatch", Value: hooks.PromptPatch{}},
		{Name: "EventRecordPayload", Value: hooks.EventRecordPayload{}},
		{Name: "EventRecordPatch", Value: hooks.EventRecordPatch{}},
		{Name: "AutomationSchedulePayload", Value: hooks.AutomationSchedulePayload{}},
		{Name: "AutomationJobPreFirePayload", Value: hooks.AutomationJobPreFirePayload{}},
		{Name: "AutomationJobPostFirePayload", Value: hooks.AutomationJobPostFirePayload{}},
		{Name: "AutomationTriggerPreFirePayload", Value: hooks.AutomationTriggerPreFirePayload{}},
		{Name: "AutomationTriggerPostFirePayload", Value: hooks.AutomationTriggerPostFirePayload{}},
		{Name: "AutomationRunCompletedPayload", Value: hooks.AutomationRunCompletedPayload{}},
		{Name: "AutomationRunFailedPayload", Value: hooks.AutomationRunFailedPayload{}},
		{Name: "AutomationFirePatch", Value: hooks.AutomationFirePatch{}},
		{Name: "AutomationObservationPatch", Value: hooks.AutomationObservationPatch{}},
		{Name: "AgentPreStartPayload", Value: hooks.AgentPreStartPayload{}},
		{Name: "AgentLifecyclePayload", Value: hooks.AgentLifecyclePayload{}},
		{Name: "AgentStartPatch", Value: hooks.AgentStartPatch{}},
		{Name: "AgentLifecyclePatch", Value: hooks.AgentLifecyclePatch{}},
		{Name: "TurnPayload", Value: hooks.TurnPayload{}},
		{Name: "TurnPatch", Value: hooks.TurnPatch{}},
		{Name: "MessagePayload", Value: hooks.MessagePayload{}},
		{Name: "MessagePatch", Value: hooks.MessagePatch{}},
		{Name: "ToolPreCallPayload", Value: hooks.ToolPreCallPayload{}},
		{Name: "ToolPostCallPayload", Value: hooks.ToolPostCallPayload{}},
		{Name: "ToolPostErrorPayload", Value: hooks.ToolPostErrorPayload{}},
		{Name: "ToolCallPatch", Value: hooks.ToolCallPatch{}},
		{Name: "ToolResultPatch", Value: hooks.ToolResultPatch{}},
		{Name: "PermissionRequestPayload", Value: hooks.PermissionRequestPayload{}},
		{Name: "PermissionResolutionPayload", Value: hooks.PermissionResolutionPayload{}},
		{Name: "PermissionRequestPatch", Value: hooks.PermissionRequestPatch{}},
		{Name: "ContextCompactPayload", Value: hooks.ContextCompactPayload{}},
		{Name: "ContextCompactionPatch", Value: hooks.ContextCompactionPatch{}},
		{Name: "HookMatcher", Value: hooks.HookMatcher{}},
		{Name: "HookDecl", Value: hooks.HookDecl{}},
	}
}

// HookContracts returns the canonical hook payload/patch registry in event order.
func HookContracts() []HookContractSpec {
	descriptors := hooks.AllEventDescriptors()
	specs := make([]HookContractSpec, 0, len(descriptors))
	for _, descriptor := range descriptors {
		payload, err := namedHookType(descriptor.PayloadSchema)
		if err != nil {
			panic(err)
		}
		patch, err := namedHookType(descriptor.PatchSchema)
		if err != nil {
			panic(err)
		}
		specs = append(specs, HookContractSpec{
			Event:   descriptor.Event,
			Payload: payload,
			Patch:   patch,
		})
	}
	return specs
}

func namedHookType(name string) (NamedType, error) {
	switch name {
	case "PayloadBase":
		return NamedType{Name: name, Value: hooks.PayloadBase{}}, nil
	case "SessionContext":
		return NamedType{Name: name, Value: hooks.SessionContext{}}, nil
	case "TurnContext":
		return NamedType{Name: name, Value: hooks.TurnContext{}}, nil
	case "ContextBlock":
		return NamedType{Name: name, Value: hooks.ContextBlock{}}, nil
	case "ToolCallRef":
		return NamedType{Name: name, Value: hooks.ToolCallRef{}}, nil
	case "ToolLocation":
		return NamedType{Name: name, Value: hooks.ToolLocation{}}, nil
	case "PermissionOption":
		return NamedType{Name: name, Value: hooks.PermissionOption{}}, nil
	case "PermissionToolCall":
		return NamedType{Name: name, Value: hooks.PermissionToolCall{}}, nil
	case "ControlPatch":
		return NamedType{Name: name, Value: hooks.ControlPatch{}}, nil
	case "SessionLifecyclePayload":
		return NamedType{Name: name, Value: hooks.SessionLifecyclePayload{}}, nil
	case "SessionPreCreatePayload":
		return NamedType{Name: name, Value: hooks.SessionPreCreatePayload{}}, nil
	case "SessionPostCreatePayload":
		return NamedType{Name: name, Value: hooks.SessionPostCreatePayload{}}, nil
	case "SessionPreResumePayload":
		return NamedType{Name: name, Value: hooks.SessionPreResumePayload{}}, nil
	case "SessionPostResumePayload":
		return NamedType{Name: name, Value: hooks.SessionPostResumePayload{}}, nil
	case "SessionPreStopPayload":
		return NamedType{Name: name, Value: hooks.SessionPreStopPayload{}}, nil
	case "SessionPostStopPayload":
		return NamedType{Name: name, Value: hooks.SessionPostStopPayload{}}, nil
	case "SessionCreatePatch":
		return NamedType{Name: name, Value: hooks.SessionCreatePatch{}}, nil
	case "SessionPostCreatePatch":
		return NamedType{Name: name, Value: hooks.SessionPostCreatePatch{}}, nil
	case "SessionPreResumePatch":
		return NamedType{Name: name, Value: hooks.SessionPreResumePatch{}}, nil
	case "SessionPostResumePatch":
		return NamedType{Name: name, Value: hooks.SessionPostResumePatch{}}, nil
	case "SessionPreStopPatch":
		return NamedType{Name: name, Value: hooks.SessionPreStopPatch{}}, nil
	case "SessionPostStopPatch":
		return NamedType{Name: name, Value: hooks.SessionPostStopPatch{}}, nil
	case "InputPreSubmitPayload":
		return NamedType{Name: name, Value: hooks.InputPreSubmitPayload{}}, nil
	case "InputPreSubmitPatch":
		return NamedType{Name: name, Value: hooks.InputPreSubmitPatch{}}, nil
	case "PromptPayload":
		return NamedType{Name: name, Value: hooks.PromptPayload{}}, nil
	case "PromptPatch":
		return NamedType{Name: name, Value: hooks.PromptPatch{}}, nil
	case "EventRecordPayload":
		return NamedType{Name: name, Value: hooks.EventRecordPayload{}}, nil
	case "EventPreRecordPayload":
		return NamedType{Name: name, Value: hooks.EventPreRecordPayload{}}, nil
	case "EventPostRecordPayload":
		return NamedType{Name: name, Value: hooks.EventPostRecordPayload{}}, nil
	case "EventRecordPatch":
		return NamedType{Name: name, Value: hooks.EventRecordPatch{}}, nil
	case "EventPreRecordPatch":
		return NamedType{Name: name, Value: hooks.EventPreRecordPatch{}}, nil
	case "EventPostRecordPatch":
		return NamedType{Name: name, Value: hooks.EventPostRecordPatch{}}, nil
	case "AutomationSchedulePayload":
		return NamedType{Name: name, Value: hooks.AutomationSchedulePayload{}}, nil
	case "AutomationJobPreFirePayload":
		return NamedType{Name: name, Value: hooks.AutomationJobPreFirePayload{}}, nil
	case "AutomationJobPostFirePayload":
		return NamedType{Name: name, Value: hooks.AutomationJobPostFirePayload{}}, nil
	case "AutomationTriggerPreFirePayload":
		return NamedType{Name: name, Value: hooks.AutomationTriggerPreFirePayload{}}, nil
	case "AutomationTriggerPostFirePayload":
		return NamedType{Name: name, Value: hooks.AutomationTriggerPostFirePayload{}}, nil
	case "AutomationRunCompletedPayload":
		return NamedType{Name: name, Value: hooks.AutomationRunCompletedPayload{}}, nil
	case "AutomationRunFailedPayload":
		return NamedType{Name: name, Value: hooks.AutomationRunFailedPayload{}}, nil
	case "AutomationFirePatch":
		return NamedType{Name: name, Value: hooks.AutomationFirePatch{}}, nil
	case "AutomationObservationPatch":
		return NamedType{Name: name, Value: hooks.AutomationObservationPatch{}}, nil
	case "AgentPreStartPayload":
		return NamedType{Name: name, Value: hooks.AgentPreStartPayload{}}, nil
	case "AgentLifecyclePayload":
		return NamedType{Name: name, Value: hooks.AgentLifecyclePayload{}}, nil
	case "AgentSpawnedPayload":
		return NamedType{Name: name, Value: hooks.AgentSpawnedPayload{}}, nil
	case "AgentCrashedPayload":
		return NamedType{Name: name, Value: hooks.AgentCrashedPayload{}}, nil
	case "AgentStoppedPayload":
		return NamedType{Name: name, Value: hooks.AgentStoppedPayload{}}, nil
	case "AgentStartPatch":
		return NamedType{Name: name, Value: hooks.AgentStartPatch{}}, nil
	case "AgentLifecyclePatch":
		return NamedType{Name: name, Value: hooks.AgentLifecyclePatch{}}, nil
	case "AgentSpawnedPatch":
		return NamedType{Name: name, Value: hooks.AgentSpawnedPatch{}}, nil
	case "AgentCrashedPatch":
		return NamedType{Name: name, Value: hooks.AgentCrashedPatch{}}, nil
	case "AgentStoppedPatch":
		return NamedType{Name: name, Value: hooks.AgentStoppedPatch{}}, nil
	case "TurnPayload":
		return NamedType{Name: name, Value: hooks.TurnPayload{}}, nil
	case "TurnStartPayload":
		return NamedType{Name: name, Value: hooks.TurnStartPayload{}}, nil
	case "TurnEndPayload":
		return NamedType{Name: name, Value: hooks.TurnEndPayload{}}, nil
	case "TurnPatch":
		return NamedType{Name: name, Value: hooks.TurnPatch{}}, nil
	case "TurnStartPatch":
		return NamedType{Name: name, Value: hooks.TurnStartPatch{}}, nil
	case "TurnEndPatch":
		return NamedType{Name: name, Value: hooks.TurnEndPatch{}}, nil
	case "MessagePayload":
		return NamedType{Name: name, Value: hooks.MessagePayload{}}, nil
	case "MessageStartPayload":
		return NamedType{Name: name, Value: hooks.MessageStartPayload{}}, nil
	case "MessageDeltaPayload":
		return NamedType{Name: name, Value: hooks.MessageDeltaPayload{}}, nil
	case "MessageEndPayload":
		return NamedType{Name: name, Value: hooks.MessageEndPayload{}}, nil
	case "MessagePatch":
		return NamedType{Name: name, Value: hooks.MessagePatch{}}, nil
	case "MessageStartPatch":
		return NamedType{Name: name, Value: hooks.MessageStartPatch{}}, nil
	case "MessageDeltaPatch":
		return NamedType{Name: name, Value: hooks.MessageDeltaPatch{}}, nil
	case "MessageEndPatch":
		return NamedType{Name: name, Value: hooks.MessageEndPatch{}}, nil
	case "ToolPreCallPayload":
		return NamedType{Name: name, Value: hooks.ToolPreCallPayload{}}, nil
	case "ToolPostCallPayload":
		return NamedType{Name: name, Value: hooks.ToolPostCallPayload{}}, nil
	case "ToolPostErrorPayload":
		return NamedType{Name: name, Value: hooks.ToolPostErrorPayload{}}, nil
	case "ToolCallPatch":
		return NamedType{Name: name, Value: hooks.ToolCallPatch{}}, nil
	case "ToolResultPatch":
		return NamedType{Name: name, Value: hooks.ToolResultPatch{}}, nil
	case "ToolPostErrorPatch":
		return NamedType{Name: name, Value: hooks.ToolPostErrorPatch{}}, nil
	case "PermissionRequestPayload":
		return NamedType{Name: name, Value: hooks.PermissionRequestPayload{}}, nil
	case "PermissionResolutionPayload":
		return NamedType{Name: name, Value: hooks.PermissionResolutionPayload{}}, nil
	case "PermissionResolvedPayload":
		return NamedType{Name: name, Value: hooks.PermissionResolvedPayload{}}, nil
	case "PermissionDeniedPayload":
		return NamedType{Name: name, Value: hooks.PermissionDeniedPayload{}}, nil
	case "PermissionRequestPatch":
		return NamedType{Name: name, Value: hooks.PermissionRequestPatch{}}, nil
	case "PermissionResolvedPatch":
		return NamedType{Name: name, Value: hooks.PermissionResolvedPatch{}}, nil
	case "PermissionDeniedPatch":
		return NamedType{Name: name, Value: hooks.PermissionDeniedPatch{}}, nil
	case "ContextCompactPayload":
		return NamedType{Name: name, Value: hooks.ContextCompactPayload{}}, nil
	case "ContextPreCompactPayload":
		return NamedType{Name: name, Value: hooks.ContextPreCompactPayload{}}, nil
	case "ContextPostCompactPayload":
		return NamedType{Name: name, Value: hooks.ContextPostCompactPayload{}}, nil
	case "ContextCompactionPatch":
		return NamedType{Name: name, Value: hooks.ContextCompactionPatch{}}, nil
	case "ContextPreCompactPatch":
		return NamedType{Name: name, Value: hooks.ContextPreCompactPatch{}}, nil
	case "ContextPostCompactPatch":
		return NamedType{Name: name, Value: hooks.ContextPostCompactPatch{}}, nil
	default:
		return NamedType{}, fmt.Errorf("unknown hook contract type %q", name)
	}
}
