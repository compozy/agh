package contract

import (
	"fmt"

	apicontract "github.com/pedronauck/agh/internal/api/contract"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	"github.com/pedronauck/agh/internal/hooks"
	memcontract "github.com/pedronauck/agh/internal/memory/contract"
	"github.com/pedronauck/agh/internal/resources"
	"github.com/pedronauck/agh/internal/subprocess"
	"github.com/pedronauck/agh/internal/tools"
)

const (
	sdkAgentHeartbeatPolicyResolvedPayloadValue = "AgentHeartbeatPolicyResolvedPayload"
	sdkAgentHeartbeatWakeAfterPayloadValue      = "AgentHeartbeatWakeAfterPayload"
	sdkAgentHeartbeatWakeBeforePayloadValue     = "AgentHeartbeatWakeBeforePayload"
	sdkAgentLifecyclePatchValue                 = "AgentLifecyclePatch"
	sdkAgentLifecyclePayloadValue               = "AgentLifecyclePayload"
	sdkAgentPreStartPayloadValue                = "AgentPreStartPayload"
	sdkAgentSoulMutationAfterPayloadValue       = "AgentSoulMutationAfterPayload"
	sdkAgentSoulMutationResponseValue           = "AgentSoulMutationResponse"
	sdkAgentSoulPayloadValue                    = "AgentSoulPayload"
	sdkAgentSoulSnapshotResolvedPayloadValue    = "AgentSoulSnapshotResolvedPayload"
	sdkAgentStartPatchValue                     = "AgentStartPatch"
	sdkAuthoredContextObservationPatchValue     = "AuthoredContextObservationPatch"
	sdkAuthoredContextProvenanceValue           = "AuthoredContextProvenance"
	sdkAuthoredMutationProvenanceValue          = "AuthoredMutationProvenance"
	sdkAutomationFirePatchValue                 = "AutomationFirePatch"
	sdkAutomationJobPostFirePayloadValue        = "AutomationJobPostFirePayload"
	sdkAutomationJobPreFirePayloadValue         = "AutomationJobPreFirePayload"
	sdkAutomationObservationPatchValue          = "AutomationObservationPatch"
	sdkAutomationRunCompletedPayloadValue       = "AutomationRunCompletedPayload"
	sdkAutomationRunFailedPayloadValue          = "AutomationRunFailedPayload"
	sdkAutomationSchedulePayloadValue           = "AutomationSchedulePayload"
	sdkAutomationTriggerPostFirePayloadValue    = "AutomationTriggerPostFirePayload"
	sdkAutomationTriggerPreFirePayloadValue     = "AutomationTriggerPreFirePayload"
	sdkAutonomyMatcherValue                     = "AutonomyMatcher"
	sdkAutonomyObservationPatchValue            = "AutonomyObservationPatch"
	sdkBridgeInstanceValue                      = "BridgeInstance"
	sdkContextBlockValue                        = "ContextBlock"
	sdkContextCompactPayloadValue               = "ContextCompactPayload"
	sdkContextCompactionPatchValue              = "ContextCompactionPatch"
	sdkControlPatchValue                        = "ControlPatch"
	sdkCoordinatorContextValue                  = "CoordinatorContext"
	sdkCoordinatorLifecyclePayloadValue         = "CoordinatorLifecyclePayload"
	sdkCoordinatorPreSpawnPayloadValue          = "CoordinatorPreSpawnPayload"
	sdkCoordinatorSpawnPatchValue               = "CoordinatorSpawnPatch"
	sdkEventRecordPatchValue                    = "EventRecordPatch"
	sdkEventRecordPayloadValue                  = "EventRecordPayload"
	sdkHeartbeatMutationResponseValue           = "HeartbeatMutationResponse"
	sdkHeartbeatPolicyPayloadValue              = "HeartbeatPolicyPayload"
	sdkInputPreSubmitPatchValue                 = "InputPreSubmitPatch"
	sdkInputPreSubmitPayloadValue               = "InputPreSubmitPayload"
	sdkMessagePatchValue                        = "MessagePatch"
	sdkMessagePayloadValue                      = "MessagePayload"
	sdkNetworkMessagePersistedPayloadValue      = "NetworkMessagePersistedPayload"
	sdkNetworkObservationPatchValue             = "NetworkObservationPatch"
	sdkNetworkPayloadValue                      = "NetworkPayload"
	sdkNetworkWorkClosedPayloadValue            = "NetworkWorkClosedPayload"
	sdkPayloadBaseValue                         = "PayloadBase"
	sdkPermissionOptionValue                    = "PermissionOption"
	sdkPermissionRequestPatchValue              = "PermissionRequestPatch"
	sdkPermissionRequestPayloadValue            = "PermissionRequestPayload"
	sdkPermissionResolutionPayloadValue         = "PermissionResolutionPayload"
	sdkPermissionSetValue                       = "PermissionSet"
	sdkPermissionToolCallValue                  = "PermissionToolCall"
	sdkPromptPatchValue                         = "PromptPatch"
	sdkPromptPayloadValue                       = "PromptPayload"
	sdkResourceRecordValue                      = "ResourceRecord"
	sdkSandboxObservationPatchValue             = "SandboxObservationPatch"
	sdkSandboxPreparePatchValue                 = "SandboxPreparePatch"
	sdkSandboxPreparePayloadValue               = "SandboxPreparePayload"
	sdkSandboxProfilePayloadValue               = "SandboxProfilePayload"
	sdkSandboxReadyPayloadValue                 = "SandboxReadyPayload"
	sdkSandboxStopPatchValue                    = "SandboxStopPatch"
	sdkSandboxStopPayloadValue                  = "SandboxStopPayload"
	sdkSandboxSyncAfterPayloadValue             = "SandboxSyncAfterPayload"
	sdkSandboxSyncBeforePatchValue              = "SandboxSyncBeforePatch"
	sdkSandboxSyncBeforePayloadValue            = "SandboxSyncBeforePayload"
	sdkSessionContextValue                      = "SessionContext"
	sdkSessionCreatePatchValue                  = "SessionCreatePatch"
	sdkSessionHealthUpdateAfterPayloadValue     = "SessionHealthUpdateAfterPayload"
	sdkSessionLifecyclePayloadValue             = "SessionLifecyclePayload"
	sdkSessionMessagePersistedPayloadValue      = "SessionMessagePersistedPayload"
	sdkSpawnContextValue                        = "SpawnContext"
	sdkSpawnCreatePatchValue                    = "SpawnCreatePatch"
	sdkSpawnLifecyclePayloadValue               = "SpawnLifecyclePayload"
	sdkSpawnPreCreatePayloadValue               = "SpawnPreCreatePayload"
	sdkTaskRunClaimCriteriaValue                = "TaskRunClaimCriteria"
	sdkTaskRunContextValue                      = "TaskRunContext"
	sdkTaskRunEnqueuedPayloadValue              = "TaskRunEnqueuedPayload"
	sdkTaskRunLeasePayloadValue                 = "TaskRunLeasePayload"
	sdkTaskRunPostClaimPayloadValue             = "TaskRunPostClaimPayload"
	sdkTaskRunPreClaimPatchValue                = "TaskRunPreClaimPatch"
	sdkTaskRunPreClaimPayloadValue              = "TaskRunPreClaimPayload"
	sdkToolCallPatchValue                       = "ToolCallPatch"
	sdkToolCallRefValue                         = "ToolCallRef"
	sdkToolLocationValue                        = "ToolLocation"
	sdkToolPostCallPayloadValue                 = "ToolPostCallPayload"
	sdkToolPostErrorPayloadValue                = "ToolPostErrorPayload"
	sdkToolPreCallPayloadValue                  = "ToolPreCallPayload"
	sdkToolResultPatchValue                     = "ToolResultPatch"
	sdkTurnContextValue                         = "TurnContext"
	sdkTurnPatchValue                           = "TurnPatch"
	sdkTurnPayloadValue                         = "TurnPayload"
)

// HookContractSpec binds one hook event to its payload and patch contracts.
type HookContractSpec struct {
	Event   hooks.HookEvent
	Payload NamedType
	Patch   NamedType
}

var sdkRootTypes = []NamedType{
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
	{Name: "ResourceKind", Value: resources.ResourceKind("")},
	{Name: "ResourceScopeKind", Value: resources.ResourceScopeKind("")},
	{Name: "ResourceScope", Value: resources.ResourceScope{}},
	{Name: "ResourceSourceKind", Value: resources.ResourceSourceKind("")},
	{Name: "ResourceSource", Value: resources.ResourceSource{}},
	{Name: "ResourceOwnerKind", Value: resources.ResourceOwnerKind("")},
	{Name: "ResourceOwner", Value: resources.ResourceOwner{}},
	{Name: sdkResourceRecordValue, Value: ResourceRecord{}},
	{Name: "ResourceGetParams", Value: ResourceGetParams{}},
	{Name: "ResourcesListParams", Value: ResourcesListParams{}},
	{Name: "ResourceSnapshotRecord", Value: ResourceSnapshotRecord{}},
	{Name: "ResourcesSnapshotParams", Value: ResourcesSnapshotParams{}},
	{Name: "AuthoredValidationStatus", Value: apicontract.AuthoredValidationStatus("")},
	{Name: "AuthoredDiagnosticSeverity", Value: apicontract.AuthoredDiagnosticSeverity("")},
	{Name: "AuthoredContextDiagnosticPayload", Value: apicontract.AuthoredContextDiagnosticPayload{}},
	{Name: "AgentSoulSectionPayload", Value: apicontract.AgentSoulSectionPayload{}},
	{Name: sdkAgentSoulPayloadValue, Value: apicontract.AgentSoulPayload{}},
	{Name: "AgentSoulValidateRequest", Value: apicontract.AgentSoulValidateRequest{}},
	{Name: "AgentSoulPutRequest", Value: apicontract.AgentSoulPutRequest{}},
	{Name: "AgentSoulDeleteRequest", Value: apicontract.AgentSoulDeleteRequest{}},
	{Name: "AgentSoulRollbackRequest", Value: apicontract.AgentSoulRollbackRequest{}},
	{Name: "AgentSoulHistoryRequest", Value: apicontract.AgentSoulHistoryRequest{}},
	{Name: "AgentSoulHistoryResponse", Value: apicontract.AgentSoulHistoryResponse{}},
	{Name: sdkAgentSoulMutationResponseValue, Value: apicontract.AgentSoulMutationResponse{}},
	{Name: "SessionSoulRefreshRequest", Value: apicontract.SessionSoulRefreshRequest{}},
	{Name: sdkHeartbeatPolicyPayloadValue, Value: apicontract.HeartbeatPolicyPayload{}},
	{Name: "HeartbeatValidateRequest", Value: apicontract.HeartbeatValidateRequest{}},
	{Name: "HeartbeatPutRequest", Value: apicontract.HeartbeatPutRequest{}},
	{Name: "HeartbeatDeleteRequest", Value: apicontract.HeartbeatDeleteRequest{}},
	{Name: "HeartbeatRollbackRequest", Value: apicontract.HeartbeatRollbackRequest{}},
	{Name: "HeartbeatHistoryRequest", Value: apicontract.HeartbeatHistoryRequest{}},
	{Name: "HeartbeatHistoryResponse", Value: apicontract.HeartbeatHistoryResponse{}},
	{Name: sdkHeartbeatMutationResponseValue, Value: apicontract.HeartbeatMutationResponse{}},
	{Name: "HeartbeatStatusRequest", Value: apicontract.HeartbeatStatusRequest{}},
	{Name: "HeartbeatStatusResponse", Value: apicontract.HeartbeatStatusResponse{}},
	{Name: "HeartbeatWakeRequest", Value: apicontract.HeartbeatWakeRequest{}},
	{Name: "HeartbeatWakeResponse", Value: apicontract.HeartbeatWakeResponse{}},
	{Name: "SessionHealthPayload", Value: apicontract.SessionHealthPayload{}},
	{Name: "SessionHealthResponse", Value: apicontract.SessionHealthResponse{}},
	{Name: "SessionStatusResponse", Value: apicontract.SessionStatusResponse{}},
	{Name: "SessionInspectResponse", Value: apicontract.SessionInspectResponse{}},
	{Name: "HeartbeatWakeStatePayload", Value: apicontract.HeartbeatWakeStatePayload{}},
	{Name: "HeartbeatWakeEventPayload", Value: apicontract.HeartbeatWakeEventPayload{}},
	{Name: "HeartbeatWakeDecisionPayload", Value: apicontract.HeartbeatWakeDecisionPayload{}},
	{Name: "ShutdownRequest", Value: subprocess.ShutdownRequest{}},
	{Name: "ShutdownResponse", Value: subprocess.ShutdownResponse{}},
	{Name: sdkBridgeInstanceValue, Value: bridgepkg.BridgeInstance{}},
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
	{Name: "BridgeTargetSnapshotRequest", Value: bridgepkg.BridgeTargetSnapshotRequest{}},
	{Name: "BridgeTargetSnapshotResponse", Value: bridgepkg.BridgeTargetSnapshotResponse{}},
	{Name: "BridgeTargetSnapshot", Value: bridgepkg.BridgeTargetSnapshot{}},
	{Name: "BridgeTargetType", Value: bridgepkg.BridgeTargetType("")},
	{Name: "DeliveryMode", Value: bridgepkg.DeliveryMode("")},
	{Name: "DeliveryOperation", Value: bridgepkg.DeliveryOperation("")},
	{Name: "DeliveryMessageReference", Value: bridgepkg.DeliveryMessageReference{}},
	{Name: "DeliveryErrorDetail", Value: bridgepkg.DeliveryErrorDetail{}},
	{Name: "DeliveryResumeState", Value: bridgepkg.DeliveryResumeState{}},
	{Name: "MessageSender", Value: bridgepkg.MessageSender{}},
	{Name: "MessageContent", Value: bridgepkg.MessageContent{}},
	{Name: "MessageAttachment", Value: bridgepkg.MessageAttachment{}},
	{Name: "Tool", Value: tools.Tool{}},
	{Name: "ToolID", Value: tools.ToolID("")},
	{Name: "RiskClass", Value: tools.RiskClass("")},
	{Name: "ToolContent", Value: tools.ToolContent{}},
	{Name: "ArtifactRef", Value: tools.ArtifactRef{}},
	{Name: "Redaction", Value: tools.Redaction{}},
	{Name: "ToolResult", Value: tools.ToolResult{}},
	{Name: "ExtensionToolRuntimeDescriptor", Value: tools.ExtensionToolRuntimeDescriptor{}},
	{Name: "ExtensionProvideToolsResponse", Value: tools.ExtensionProvideToolsResponse{}},
	{Name: "ExtensionToolCallRequest", Value: tools.ExtensionToolCallRequest{}},
	{Name: "ExtensionToolCallResponse", Value: tools.ExtensionToolCallResponse{}},
	{Name: "ModelSourceListParams", Value: ModelSourceListParams{}},
	{Name: "ModelSourceListResponse", Value: ModelSourceListResponse{}},
	{Name: "ModelSourceRow", Value: ModelSourceRow{}},
	{Name: "MemoryScope", Value: memcontract.Scope("")},
	{Name: "HookEventFamily", Value: hooks.HookEventFamily("")},
	{Name: "HookRunOutcome", Value: hooks.HookRunOutcome("")},
	{Name: "HookSkillSource", Value: hooks.HookSkillSource("")},
	{Name: sdkPayloadBaseValue, Value: hooks.PayloadBase{}},
	{Name: sdkSessionContextValue, Value: hooks.SessionContext{}},
	{Name: sdkTurnContextValue, Value: hooks.TurnContext{}},
	{Name: sdkContextBlockValue, Value: hooks.ContextBlock{}},
	{Name: sdkToolCallRefValue, Value: hooks.ToolCallRef{}},
	{Name: sdkToolLocationValue, Value: hooks.ToolLocation{}},
	{Name: sdkPermissionOptionValue, Value: hooks.PermissionOption{}},
	{Name: sdkPermissionToolCallValue, Value: hooks.PermissionToolCall{}},
	{Name: sdkControlPatchValue, Value: hooks.ControlPatch{}},
	{Name: sdkSessionLifecyclePayloadValue, Value: hooks.SessionLifecyclePayload{}},
	{Name: sdkSessionCreatePatchValue, Value: hooks.SessionCreatePatch{}},
	{Name: sdkSandboxProfilePayloadValue, Value: hooks.SandboxProfilePayload{}},
	{Name: sdkSandboxPreparePayloadValue, Value: hooks.SandboxPreparePayload{}},
	{Name: sdkSandboxReadyPayloadValue, Value: hooks.SandboxReadyPayload{}},
	{Name: sdkSandboxSyncBeforePayloadValue, Value: hooks.SandboxSyncBeforePayload{}},
	{Name: sdkSandboxSyncAfterPayloadValue, Value: hooks.SandboxSyncAfterPayload{}},
	{Name: sdkSandboxStopPayloadValue, Value: hooks.SandboxStopPayload{}},
	{Name: sdkSandboxPreparePatchValue, Value: hooks.SandboxPreparePatch{}},
	{Name: sdkSandboxSyncBeforePatchValue, Value: hooks.SandboxSyncBeforePatch{}},
	{Name: sdkSandboxObservationPatchValue, Value: hooks.SandboxObservationPatch{}},
	{Name: sdkSandboxStopPatchValue, Value: hooks.SandboxStopPatch{}},
	{Name: sdkInputPreSubmitPayloadValue, Value: hooks.InputPreSubmitPayload{}},
	{Name: sdkInputPreSubmitPatchValue, Value: hooks.InputPreSubmitPatch{}},
	{Name: sdkPromptPayloadValue, Value: hooks.PromptPayload{}},
	{Name: sdkPromptPatchValue, Value: hooks.PromptPatch{}},
	{Name: sdkEventRecordPayloadValue, Value: hooks.EventRecordPayload{}},
	{Name: sdkEventRecordPatchValue, Value: hooks.EventRecordPatch{}},
	{Name: sdkAutomationSchedulePayloadValue, Value: hooks.AutomationSchedulePayload{}},
	{Name: sdkAutomationJobPreFirePayloadValue, Value: hooks.AutomationJobPreFirePayload{}},
	{Name: sdkAutomationJobPostFirePayloadValue, Value: hooks.AutomationJobPostFirePayload{}},
	{Name: sdkAutomationTriggerPreFirePayloadValue, Value: hooks.AutomationTriggerPreFirePayload{}},
	{Name: sdkAutomationTriggerPostFirePayloadValue, Value: hooks.AutomationTriggerPostFirePayload{}},
	{Name: sdkAutomationRunCompletedPayloadValue, Value: hooks.AutomationRunCompletedPayload{}},
	{Name: sdkAutomationRunFailedPayloadValue, Value: hooks.AutomationRunFailedPayload{}},
	{Name: sdkAutomationFirePatchValue, Value: hooks.AutomationFirePatch{}},
	{Name: sdkAutomationObservationPatchValue, Value: hooks.AutomationObservationPatch{}},
	{Name: sdkAgentPreStartPayloadValue, Value: hooks.AgentPreStartPayload{}},
	{Name: sdkAgentLifecyclePayloadValue, Value: hooks.AgentLifecyclePayload{}},
	{Name: sdkAgentStartPatchValue, Value: hooks.AgentStartPatch{}},
	{Name: sdkAgentLifecyclePatchValue, Value: hooks.AgentLifecyclePatch{}},
	{Name: sdkAuthoredContextProvenanceValue, Value: hooks.AuthoredContextProvenance{}},
	{Name: sdkAuthoredMutationProvenanceValue, Value: hooks.AuthoredMutationProvenance{}},
	{Name: sdkAgentSoulSnapshotResolvedPayloadValue, Value: hooks.AgentSoulSnapshotResolvedPayload{}},
	{Name: sdkAgentSoulMutationAfterPayloadValue, Value: hooks.AgentSoulMutationAfterPayload{}},
	{Name: sdkAgentHeartbeatPolicyResolvedPayloadValue, Value: hooks.AgentHeartbeatPolicyResolvedPayload{}},
	{Name: sdkAgentHeartbeatWakeBeforePayloadValue, Value: hooks.AgentHeartbeatWakeBeforePayload{}},
	{Name: sdkAgentHeartbeatWakeAfterPayloadValue, Value: hooks.AgentHeartbeatWakeAfterPayload{}},
	{Name: sdkSessionHealthUpdateAfterPayloadValue, Value: hooks.SessionHealthUpdateAfterPayload{}},
	{Name: sdkAuthoredContextObservationPatchValue, Value: hooks.AuthoredContextObservationPatch{}},
	{Name: sdkNetworkPayloadValue, Value: hooks.NetworkPayload{}},
	{Name: sdkNetworkObservationPatchValue, Value: hooks.NetworkObservationPatch{}},
	{Name: sdkTurnPayloadValue, Value: hooks.TurnPayload{}},
	{Name: sdkTurnPatchValue, Value: hooks.TurnPatch{}},
	{Name: sdkMessagePayloadValue, Value: hooks.MessagePayload{}},
	{Name: sdkSessionMessagePersistedPayloadValue, Value: hooks.SessionMessagePersistedPayload{}},
	{Name: sdkMessagePatchValue, Value: hooks.MessagePatch{}},
	{Name: sdkToolPreCallPayloadValue, Value: hooks.ToolPreCallPayload{}},
	{Name: sdkToolPostCallPayloadValue, Value: hooks.ToolPostCallPayload{}},
	{Name: sdkToolPostErrorPayloadValue, Value: hooks.ToolPostErrorPayload{}},
	{Name: sdkToolCallPatchValue, Value: hooks.ToolCallPatch{}},
	{Name: sdkToolResultPatchValue, Value: hooks.ToolResultPatch{}},
	{Name: sdkPermissionRequestPayloadValue, Value: hooks.PermissionRequestPayload{}},
	{Name: sdkPermissionResolutionPayloadValue, Value: hooks.PermissionResolutionPayload{}},
	{Name: sdkPermissionRequestPatchValue, Value: hooks.PermissionRequestPatch{}},
	{Name: sdkContextCompactPayloadValue, Value: hooks.ContextCompactPayload{}},
	{Name: sdkContextCompactionPatchValue, Value: hooks.ContextCompactionPatch{}},
	{Name: sdkAutonomyObservationPatchValue, Value: hooks.AutonomyObservationPatch{}},
	{Name: sdkCoordinatorContextValue, Value: hooks.CoordinatorContext{}},
	{Name: sdkCoordinatorPreSpawnPayloadValue, Value: hooks.CoordinatorPreSpawnPayload{}},
	{Name: sdkCoordinatorLifecyclePayloadValue, Value: hooks.CoordinatorLifecyclePayload{}},
	{Name: sdkCoordinatorSpawnPatchValue, Value: hooks.CoordinatorSpawnPatch{}},
	{Name: sdkTaskRunClaimCriteriaValue, Value: hooks.TaskRunClaimCriteria{}},
	{Name: sdkTaskRunContextValue, Value: hooks.TaskRunContext{}},
	{Name: sdkTaskRunEnqueuedPayloadValue, Value: hooks.TaskRunEnqueuedPayload{}},
	{Name: sdkTaskRunPreClaimPayloadValue, Value: hooks.TaskRunPreClaimPayload{}},
	{Name: sdkTaskRunPostClaimPayloadValue, Value: hooks.TaskRunPostClaimPayload{}},
	{Name: sdkTaskRunLeasePayloadValue, Value: hooks.TaskRunLeasePayload{}},
	{Name: sdkTaskRunPreClaimPatchValue, Value: hooks.TaskRunPreClaimPatch{}},
	{Name: sdkPermissionSetValue, Value: hooks.PermissionSet{}},
	{Name: sdkSpawnContextValue, Value: hooks.SpawnContext{}},
	{Name: sdkSpawnPreCreatePayloadValue, Value: hooks.SpawnPreCreatePayload{}},
	{Name: sdkSpawnLifecyclePayloadValue, Value: hooks.SpawnLifecyclePayload{}},
	{Name: sdkSpawnCreatePatchValue, Value: hooks.SpawnCreatePatch{}},
	{Name: sdkAutonomyMatcherValue, Value: hooks.AutonomyMatcher{}},
	{Name: "NetworkMatcher", Value: hooks.NetworkMatcher{}},
	{Name: "CompactionMatcher", Value: hooks.CompactionMatcher{}},
	{Name: "HookMatcher", Value: hooks.HookMatcher{}},
	{Name: "HookDecl", Value: hooks.HookDecl{}},
}

// SDKRootTypes returns the canonical generated SDK contract roots.
func SDKRootTypes() []NamedType {
	return append([]NamedType(nil), sdkRootTypes...)
}

// BuildHookContracts returns the canonical hook payload/patch registry in event order.
func BuildHookContracts() ([]HookContractSpec, error) {
	descriptors := hooks.AllEventDescriptors()
	specs := make([]HookContractSpec, 0, len(descriptors))
	for _, descriptor := range descriptors {
		payload, err := namedHookType(descriptor.PayloadSchema)
		if err != nil {
			return nil, fmt.Errorf("hook contract %q payload schema: %w", descriptor.Event, err)
		}
		patch, err := namedHookType(descriptor.PatchSchema)
		if err != nil {
			return nil, fmt.Errorf("hook contract %q patch schema: %w", descriptor.Event, err)
		}
		specs = append(specs, HookContractSpec{
			Event:   descriptor.Event,
			Payload: payload,
			Patch:   patch,
		})
	}
	return specs, nil
}

// HookContracts returns the canonical hook payload/patch registry in event order.
func HookContracts() []HookContractSpec {
	specs, err := BuildHookContracts()
	if err != nil {
		panic(err)
	}
	return specs
}

var namedHookTypes = map[string]NamedType{
	sdkPayloadBaseValue:             {Name: sdkPayloadBaseValue, Value: hooks.PayloadBase{}},
	sdkSessionContextValue:          {Name: sdkSessionContextValue, Value: hooks.SessionContext{}},
	sdkTurnContextValue:             {Name: sdkTurnContextValue, Value: hooks.TurnContext{}},
	sdkContextBlockValue:            {Name: sdkContextBlockValue, Value: hooks.ContextBlock{}},
	sdkToolCallRefValue:             {Name: sdkToolCallRefValue, Value: hooks.ToolCallRef{}},
	sdkToolLocationValue:            {Name: sdkToolLocationValue, Value: hooks.ToolLocation{}},
	sdkPermissionOptionValue:        {Name: sdkPermissionOptionValue, Value: hooks.PermissionOption{}},
	sdkPermissionToolCallValue:      {Name: sdkPermissionToolCallValue, Value: hooks.PermissionToolCall{}},
	sdkControlPatchValue:            {Name: sdkControlPatchValue, Value: hooks.ControlPatch{}},
	sdkSessionLifecyclePayloadValue: {Name: sdkSessionLifecyclePayloadValue, Value: hooks.SessionLifecyclePayload{}},
	"SessionPreCreatePayload":       {Name: "SessionPreCreatePayload", Value: hooks.SessionPreCreatePayload{}},
	"SessionPostCreatePayload":      {Name: "SessionPostCreatePayload", Value: hooks.SessionPostCreatePayload{}},
	"SessionPreResumePayload":       {Name: "SessionPreResumePayload", Value: hooks.SessionPreResumePayload{}},
	"SessionPostResumePayload":      {Name: "SessionPostResumePayload", Value: hooks.SessionPostResumePayload{}},
	"SessionPreStopPayload":         {Name: "SessionPreStopPayload", Value: hooks.SessionPreStopPayload{}},
	"SessionPostStopPayload":        {Name: "SessionPostStopPayload", Value: hooks.SessionPostStopPayload{}},
	sdkSessionCreatePatchValue:      {Name: sdkSessionCreatePatchValue, Value: hooks.SessionCreatePatch{}},
	"SessionPostCreatePatch":        {Name: "SessionPostCreatePatch", Value: hooks.SessionPostCreatePatch{}},
	"SessionPreResumePatch":         {Name: "SessionPreResumePatch", Value: hooks.SessionPreResumePatch{}},
	"SessionPostResumePatch":        {Name: "SessionPostResumePatch", Value: hooks.SessionPostResumePatch{}},
	"SessionPreStopPatch":           {Name: "SessionPreStopPatch", Value: hooks.SessionPreStopPatch{}},
	"SessionPostStopPatch":          {Name: "SessionPostStopPatch", Value: hooks.SessionPostStopPatch{}},
	sdkSandboxProfilePayloadValue:   {Name: sdkSandboxProfilePayloadValue, Value: hooks.SandboxProfilePayload{}},
	sdkSandboxPreparePayloadValue:   {Name: sdkSandboxPreparePayloadValue, Value: hooks.SandboxPreparePayload{}},
	sdkSandboxReadyPayloadValue:     {Name: sdkSandboxReadyPayloadValue, Value: hooks.SandboxReadyPayload{}},
	sdkSandboxSyncBeforePayloadValue: {
		Name:  sdkSandboxSyncBeforePayloadValue,
		Value: hooks.SandboxSyncBeforePayload{},
	},
	sdkSandboxSyncAfterPayloadValue: {
		Name:  sdkSandboxSyncAfterPayloadValue,
		Value: hooks.SandboxSyncAfterPayload{},
	},
	sdkSandboxStopPayloadValue:  {Name: sdkSandboxStopPayloadValue, Value: hooks.SandboxStopPayload{}},
	sdkSandboxPreparePatchValue: {Name: sdkSandboxPreparePatchValue, Value: hooks.SandboxPreparePatch{}},
	sdkSandboxSyncBeforePatchValue: {
		Name:  sdkSandboxSyncBeforePatchValue,
		Value: hooks.SandboxSyncBeforePatch{},
	},
	sdkSandboxObservationPatchValue: {
		Name:  sdkSandboxObservationPatchValue,
		Value: hooks.SandboxObservationPatch{},
	},
	"SandboxReadyPatch":           {Name: "SandboxReadyPatch", Value: hooks.SandboxReadyPatch{}},
	"SandboxSyncAfterPatch":       {Name: "SandboxSyncAfterPatch", Value: hooks.SandboxSyncAfterPatch{}},
	sdkSandboxStopPatchValue:      {Name: sdkSandboxStopPatchValue, Value: hooks.SandboxStopPatch{}},
	sdkInputPreSubmitPayloadValue: {Name: sdkInputPreSubmitPayloadValue, Value: hooks.InputPreSubmitPayload{}},
	sdkInputPreSubmitPatchValue:   {Name: sdkInputPreSubmitPatchValue, Value: hooks.InputPreSubmitPatch{}},
	sdkPromptPayloadValue:         {Name: sdkPromptPayloadValue, Value: hooks.PromptPayload{}},
	sdkPromptPatchValue:           {Name: sdkPromptPatchValue, Value: hooks.PromptPatch{}},
	sdkEventRecordPayloadValue:    {Name: sdkEventRecordPayloadValue, Value: hooks.EventRecordPayload{}},
	"EventPreRecordPayload":       {Name: "EventPreRecordPayload", Value: hooks.EventPreRecordPayload{}},
	"EventPostRecordPayload":      {Name: "EventPostRecordPayload", Value: hooks.EventPostRecordPayload{}},
	sdkEventRecordPatchValue:      {Name: sdkEventRecordPatchValue, Value: hooks.EventRecordPatch{}},
	"EventPreRecordPatch":         {Name: "EventPreRecordPatch", Value: hooks.EventPreRecordPatch{}},
	"EventPostRecordPatch":        {Name: "EventPostRecordPatch", Value: hooks.EventPostRecordPatch{}},
	sdkAutomationSchedulePayloadValue: {
		Name:  sdkAutomationSchedulePayloadValue,
		Value: hooks.AutomationSchedulePayload{},
	},
	sdkAutomationJobPreFirePayloadValue: {
		Name:  sdkAutomationJobPreFirePayloadValue,
		Value: hooks.AutomationJobPreFirePayload{},
	},
	sdkAutomationJobPostFirePayloadValue: {
		Name:  sdkAutomationJobPostFirePayloadValue,
		Value: hooks.AutomationJobPostFirePayload{},
	},
	sdkAutomationTriggerPreFirePayloadValue: {
		Name:  sdkAutomationTriggerPreFirePayloadValue,
		Value: hooks.AutomationTriggerPreFirePayload{},
	},
	sdkAutomationTriggerPostFirePayloadValue: {
		Name:  sdkAutomationTriggerPostFirePayloadValue,
		Value: hooks.AutomationTriggerPostFirePayload{},
	},
	sdkAutomationRunCompletedPayloadValue: {
		Name:  sdkAutomationRunCompletedPayloadValue,
		Value: hooks.AutomationRunCompletedPayload{},
	},
	sdkAutomationRunFailedPayloadValue: {
		Name:  sdkAutomationRunFailedPayloadValue,
		Value: hooks.AutomationRunFailedPayload{},
	},
	sdkAutomationFirePatchValue: {
		Name:  sdkAutomationFirePatchValue,
		Value: hooks.AutomationFirePatch{},
	},
	sdkAutomationObservationPatchValue: {
		Name:  sdkAutomationObservationPatchValue,
		Value: hooks.AutomationObservationPatch{},
	},
	sdkAgentPreStartPayloadValue: {
		Name:  sdkAgentPreStartPayloadValue,
		Value: hooks.AgentPreStartPayload{},
	},
	sdkAgentLifecyclePayloadValue: {
		Name:  sdkAgentLifecyclePayloadValue,
		Value: hooks.AgentLifecyclePayload{},
	},
	"AgentSpawnedPayload": {
		Name:  "AgentSpawnedPayload",
		Value: hooks.AgentSpawnedPayload{},
	},
	"AgentCrashedPayload": {
		Name:  "AgentCrashedPayload",
		Value: hooks.AgentCrashedPayload{},
	},
	"AgentStoppedPayload": {
		Name:  "AgentStoppedPayload",
		Value: hooks.AgentStoppedPayload{},
	},
	sdkAgentStartPatchValue:     {Name: sdkAgentStartPatchValue, Value: hooks.AgentStartPatch{}},
	sdkAgentLifecyclePatchValue: {Name: sdkAgentLifecyclePatchValue, Value: hooks.AgentLifecyclePatch{}},
	"AgentSpawnedPatch":         {Name: "AgentSpawnedPatch", Value: hooks.AgentSpawnedPatch{}},
	"AgentCrashedPatch":         {Name: "AgentCrashedPatch", Value: hooks.AgentCrashedPatch{}},
	"AgentStoppedPatch":         {Name: "AgentStoppedPatch", Value: hooks.AgentStoppedPatch{}},
	sdkAuthoredContextProvenanceValue: {
		Name:  sdkAuthoredContextProvenanceValue,
		Value: hooks.AuthoredContextProvenance{},
	},
	sdkAuthoredMutationProvenanceValue: {
		Name:  sdkAuthoredMutationProvenanceValue,
		Value: hooks.AuthoredMutationProvenance{},
	},
	sdkAgentSoulSnapshotResolvedPayloadValue: {
		Name:  sdkAgentSoulSnapshotResolvedPayloadValue,
		Value: hooks.AgentSoulSnapshotResolvedPayload{},
	},
	sdkAgentSoulMutationAfterPayloadValue: {
		Name:  sdkAgentSoulMutationAfterPayloadValue,
		Value: hooks.AgentSoulMutationAfterPayload{},
	},
	sdkAgentHeartbeatPolicyResolvedPayloadValue: {
		Name:  sdkAgentHeartbeatPolicyResolvedPayloadValue,
		Value: hooks.AgentHeartbeatPolicyResolvedPayload{},
	},
	sdkAgentHeartbeatWakeBeforePayloadValue: {
		Name:  sdkAgentHeartbeatWakeBeforePayloadValue,
		Value: hooks.AgentHeartbeatWakeBeforePayload{},
	},
	sdkAgentHeartbeatWakeAfterPayloadValue: {
		Name:  sdkAgentHeartbeatWakeAfterPayloadValue,
		Value: hooks.AgentHeartbeatWakeAfterPayload{},
	},
	sdkSessionHealthUpdateAfterPayloadValue: {
		Name:  sdkSessionHealthUpdateAfterPayloadValue,
		Value: hooks.SessionHealthUpdateAfterPayload{},
	},
	sdkAuthoredContextObservationPatchValue: {
		Name:  sdkAuthoredContextObservationPatchValue,
		Value: hooks.AuthoredContextObservationPatch{},
	},
	sdkNetworkPayloadValue:       {Name: sdkNetworkPayloadValue, Value: hooks.NetworkPayload{}},
	"NetworkThreadOpenedPayload": {Name: "NetworkThreadOpenedPayload", Value: hooks.NetworkThreadOpenedPayload{}},
	"NetworkDirectRoomOpenedPayload": {
		Name:  "NetworkDirectRoomOpenedPayload",
		Value: hooks.NetworkDirectRoomOpenedPayload{},
	},
	sdkNetworkMessagePersistedPayloadValue: {
		Name:  sdkNetworkMessagePersistedPayloadValue,
		Value: hooks.NetworkMessagePersistedPayload{},
	},
	"NetworkWorkOpenedPayload": {Name: "NetworkWorkOpenedPayload", Value: hooks.NetworkWorkOpenedPayload{}},
	"NetworkWorkTransitionedPayload": {
		Name:  "NetworkWorkTransitionedPayload",
		Value: hooks.NetworkWorkTransitionedPayload{},
	},
	sdkNetworkWorkClosedPayloadValue: {Name: sdkNetworkWorkClosedPayloadValue, Value: hooks.NetworkWorkClosedPayload{}},
	sdkNetworkObservationPatchValue:  {Name: sdkNetworkObservationPatchValue, Value: hooks.NetworkObservationPatch{}},
	sdkTurnPayloadValue:              {Name: sdkTurnPayloadValue, Value: hooks.TurnPayload{}},
	"TurnStartPayload":               {Name: "TurnStartPayload", Value: hooks.TurnStartPayload{}},
	"TurnEndPayload":                 {Name: "TurnEndPayload", Value: hooks.TurnEndPayload{}},
	sdkTurnPatchValue:                {Name: sdkTurnPatchValue, Value: hooks.TurnPatch{}},
	"TurnStartPatch":                 {Name: "TurnStartPatch", Value: hooks.TurnStartPatch{}},
	"TurnEndPatch":                   {Name: "TurnEndPatch", Value: hooks.TurnEndPatch{}},
	sdkMessagePayloadValue:           {Name: sdkMessagePayloadValue, Value: hooks.MessagePayload{}},
	"MessageStartPayload":            {Name: "MessageStartPayload", Value: hooks.MessageStartPayload{}},
	"MessageDeltaPayload":            {Name: "MessageDeltaPayload", Value: hooks.MessageDeltaPayload{}},
	"MessageEndPayload":              {Name: "MessageEndPayload", Value: hooks.MessageEndPayload{}},
	sdkSessionMessagePersistedPayloadValue: {
		Name:  sdkSessionMessagePersistedPayloadValue,
		Value: hooks.SessionMessagePersistedPayload{},
	},
	sdkMessagePatchValue:         {Name: sdkMessagePatchValue, Value: hooks.MessagePatch{}},
	"MessageStartPatch":          {Name: "MessageStartPatch", Value: hooks.MessageStartPatch{}},
	"MessageDeltaPatch":          {Name: "MessageDeltaPatch", Value: hooks.MessageDeltaPatch{}},
	"MessageEndPatch":            {Name: "MessageEndPatch", Value: hooks.MessageEndPatch{}},
	sdkToolPreCallPayloadValue:   {Name: sdkToolPreCallPayloadValue, Value: hooks.ToolPreCallPayload{}},
	sdkToolPostCallPayloadValue:  {Name: sdkToolPostCallPayloadValue, Value: hooks.ToolPostCallPayload{}},
	sdkToolPostErrorPayloadValue: {Name: sdkToolPostErrorPayloadValue, Value: hooks.ToolPostErrorPayload{}},
	sdkToolCallPatchValue:        {Name: sdkToolCallPatchValue, Value: hooks.ToolCallPatch{}},
	sdkToolResultPatchValue:      {Name: sdkToolResultPatchValue, Value: hooks.ToolResultPatch{}},
	"ToolPostErrorPatch":         {Name: "ToolPostErrorPatch", Value: hooks.ToolPostErrorPatch{}},
	sdkPermissionRequestPayloadValue: {
		Name:  sdkPermissionRequestPayloadValue,
		Value: hooks.PermissionRequestPayload{},
	},
	sdkPermissionResolutionPayloadValue: {
		Name:  sdkPermissionResolutionPayloadValue,
		Value: hooks.PermissionResolutionPayload{},
	},
	"PermissionResolvedPayload":    {Name: "PermissionResolvedPayload", Value: hooks.PermissionResolvedPayload{}},
	"PermissionDeniedPayload":      {Name: "PermissionDeniedPayload", Value: hooks.PermissionDeniedPayload{}},
	sdkPermissionRequestPatchValue: {Name: sdkPermissionRequestPatchValue, Value: hooks.PermissionRequestPatch{}},
	"PermissionResolvedPatch":      {Name: "PermissionResolvedPatch", Value: hooks.PermissionResolvedPatch{}},
	"PermissionDeniedPatch":        {Name: "PermissionDeniedPatch", Value: hooks.PermissionDeniedPatch{}},
	sdkContextCompactPayloadValue:  {Name: sdkContextCompactPayloadValue, Value: hooks.ContextCompactPayload{}},
	"ContextPreCompactPayload":     {Name: "ContextPreCompactPayload", Value: hooks.ContextPreCompactPayload{}},
	"ContextPostCompactPayload":    {Name: "ContextPostCompactPayload", Value: hooks.ContextPostCompactPayload{}},
	sdkContextCompactionPatchValue: {Name: sdkContextCompactionPatchValue, Value: hooks.ContextCompactionPatch{}},
	"ContextPreCompactPatch":       {Name: "ContextPreCompactPatch", Value: hooks.ContextPreCompactPatch{}},
	"ContextPostCompactPatch":      {Name: "ContextPostCompactPatch", Value: hooks.ContextPostCompactPatch{}},
	sdkAutonomyObservationPatchValue: {
		Name:  sdkAutonomyObservationPatchValue,
		Value: hooks.AutonomyObservationPatch{},
	},
	sdkCoordinatorContextValue: {Name: sdkCoordinatorContextValue, Value: hooks.CoordinatorContext{}},
	sdkCoordinatorPreSpawnPayloadValue: {
		Name:  sdkCoordinatorPreSpawnPayloadValue,
		Value: hooks.CoordinatorPreSpawnPayload{},
	},
	sdkCoordinatorLifecyclePayloadValue: {
		Name:  sdkCoordinatorLifecyclePayloadValue,
		Value: hooks.CoordinatorLifecyclePayload{},
	},
	"CoordinatorSpawnedPayload": {
		Name:  "CoordinatorSpawnedPayload",
		Value: hooks.CoordinatorSpawnedPayload{},
	},
	"CoordinatorDecisionPayload": {
		Name:  "CoordinatorDecisionPayload",
		Value: hooks.CoordinatorDecisionPayload{},
	},
	"CoordinatorStoppedPayload": {
		Name:  "CoordinatorStoppedPayload",
		Value: hooks.CoordinatorStoppedPayload{},
	},
	"CoordinatorFailedPayload": {
		Name:  "CoordinatorFailedPayload",
		Value: hooks.CoordinatorFailedPayload{},
	},
	sdkCoordinatorSpawnPatchValue: {Name: sdkCoordinatorSpawnPatchValue, Value: hooks.CoordinatorSpawnPatch{}},
	"CoordinatorObservationPatch": {Name: "CoordinatorObservationPatch", Value: hooks.CoordinatorObservationPatch{}},
	sdkTaskRunClaimCriteriaValue:  {Name: sdkTaskRunClaimCriteriaValue, Value: hooks.TaskRunClaimCriteria{}},
	sdkTaskRunContextValue:        {Name: sdkTaskRunContextValue, Value: hooks.TaskRunContext{}},
	sdkTaskRunEnqueuedPayloadValue: {
		Name:  sdkTaskRunEnqueuedPayloadValue,
		Value: hooks.TaskRunEnqueuedPayload{},
	},
	sdkTaskRunPreClaimPayloadValue: {
		Name:  sdkTaskRunPreClaimPayloadValue,
		Value: hooks.TaskRunPreClaimPayload{},
	},
	sdkTaskRunPostClaimPayloadValue: {
		Name:  sdkTaskRunPostClaimPayloadValue,
		Value: hooks.TaskRunPostClaimPayload{},
	},
	sdkTaskRunLeasePayloadValue: {
		Name:  sdkTaskRunLeasePayloadValue,
		Value: hooks.TaskRunLeasePayload{},
	},
	"TaskRunLeaseExtendedPayload": {
		Name:  "TaskRunLeaseExtendedPayload",
		Value: hooks.TaskRunLeaseExtendedPayload{},
	},
	"TaskRunLeaseExpiredPayload": {
		Name:  "TaskRunLeaseExpiredPayload",
		Value: hooks.TaskRunLeaseExpiredPayload{},
	},
	"TaskRunLeaseRecoveredPayload": {
		Name:  "TaskRunLeaseRecoveredPayload",
		Value: hooks.TaskRunLeaseRecoveredPayload{},
	},
	"TaskRunReleasedPayload": {
		Name:  "TaskRunReleasedPayload",
		Value: hooks.TaskRunReleasedPayload{},
	},
	"TaskRunCompletedPayload": {
		Name:  "TaskRunCompletedPayload",
		Value: hooks.TaskRunCompletedPayload{},
	},
	"TaskRunFailedPayload": {
		Name:  "TaskRunFailedPayload",
		Value: hooks.TaskRunFailedPayload{},
	},
	sdkTaskRunPreClaimPatchValue: {Name: sdkTaskRunPreClaimPatchValue, Value: hooks.TaskRunPreClaimPatch{}},
	"TaskRunObservationPatch":    {Name: "TaskRunObservationPatch", Value: hooks.TaskRunObservationPatch{}},
	sdkPermissionSetValue:        {Name: sdkPermissionSetValue, Value: hooks.PermissionSet{}},
	sdkSpawnContextValue:         {Name: sdkSpawnContextValue, Value: hooks.SpawnContext{}},
	sdkSpawnPreCreatePayloadValue: {
		Name:  sdkSpawnPreCreatePayloadValue,
		Value: hooks.SpawnPreCreatePayload{},
	},
	sdkSpawnLifecyclePayloadValue: {
		Name:  sdkSpawnLifecyclePayloadValue,
		Value: hooks.SpawnLifecyclePayload{},
	},
	"SpawnCreatedPayload": {
		Name:  "SpawnCreatedPayload",
		Value: hooks.SpawnCreatedPayload{},
	},
	"SpawnParentStoppedPayload": {
		Name:  "SpawnParentStoppedPayload",
		Value: hooks.SpawnParentStoppedPayload{},
	},
	"SpawnTTLExpiredPayload": {
		Name:  "SpawnTTLExpiredPayload",
		Value: hooks.SpawnTTLExpiredPayload{},
	},
	"SpawnReapedPayload": {
		Name:  "SpawnReapedPayload",
		Value: hooks.SpawnReapedPayload{},
	},
	sdkSpawnCreatePatchValue: {Name: sdkSpawnCreatePatchValue, Value: hooks.SpawnCreatePatch{}},
	"SpawnObservationPatch":  {Name: "SpawnObservationPatch", Value: hooks.SpawnObservationPatch{}},
	sdkAutonomyMatcherValue:  {Name: sdkAutonomyMatcherValue, Value: hooks.AutonomyMatcher{}},
}

func namedHookType(name string) (NamedType, error) {
	namedType, ok := namedHookTypes[name]
	if !ok {
		return NamedType{}, fmt.Errorf("unknown hook contract type %q", name)
	}
	return namedType, nil
}
