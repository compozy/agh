package contract

import (
	"fmt"

	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	"github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/resources"
	"github.com/pedronauck/agh/internal/subprocess"
	"github.com/pedronauck/agh/internal/tools"
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
	{Name: "ResourceRecord", Value: ResourceRecord{}},
	{Name: "ResourceGetParams", Value: ResourceGetParams{}},
	{Name: "ResourcesListParams", Value: ResourcesListParams{}},
	{Name: "ResourceSnapshotRecord", Value: ResourceSnapshotRecord{}},
	{Name: "ResourcesSnapshotParams", Value: ResourcesSnapshotParams{}},
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
	{Name: "EnvironmentProfilePayload", Value: hooks.EnvironmentProfilePayload{}},
	{Name: "EnvironmentPreparePayload", Value: hooks.EnvironmentPreparePayload{}},
	{Name: "EnvironmentReadyPayload", Value: hooks.EnvironmentReadyPayload{}},
	{Name: "EnvironmentSyncBeforePayload", Value: hooks.EnvironmentSyncBeforePayload{}},
	{Name: "EnvironmentSyncAfterPayload", Value: hooks.EnvironmentSyncAfterPayload{}},
	{Name: "EnvironmentStopPayload", Value: hooks.EnvironmentStopPayload{}},
	{Name: "EnvironmentPreparePatch", Value: hooks.EnvironmentPreparePatch{}},
	{Name: "EnvironmentSyncBeforePatch", Value: hooks.EnvironmentSyncBeforePatch{}},
	{Name: "EnvironmentObservationPatch", Value: hooks.EnvironmentObservationPatch{}},
	{Name: "EnvironmentStopPatch", Value: hooks.EnvironmentStopPatch{}},
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
	{Name: "AutonomyObservationPatch", Value: hooks.AutonomyObservationPatch{}},
	{Name: "CoordinatorContext", Value: hooks.CoordinatorContext{}},
	{Name: "CoordinatorPreSpawnPayload", Value: hooks.CoordinatorPreSpawnPayload{}},
	{Name: "CoordinatorLifecyclePayload", Value: hooks.CoordinatorLifecyclePayload{}},
	{Name: "CoordinatorSpawnPatch", Value: hooks.CoordinatorSpawnPatch{}},
	{Name: "TaskRunClaimCriteria", Value: hooks.TaskRunClaimCriteria{}},
	{Name: "TaskRunContext", Value: hooks.TaskRunContext{}},
	{Name: "TaskRunEnqueuedPayload", Value: hooks.TaskRunEnqueuedPayload{}},
	{Name: "TaskRunPreClaimPayload", Value: hooks.TaskRunPreClaimPayload{}},
	{Name: "TaskRunPostClaimPayload", Value: hooks.TaskRunPostClaimPayload{}},
	{Name: "TaskRunLeasePayload", Value: hooks.TaskRunLeasePayload{}},
	{Name: "TaskRunPreClaimPatch", Value: hooks.TaskRunPreClaimPatch{}},
	{Name: "PermissionSet", Value: hooks.PermissionSet{}},
	{Name: "SpawnContext", Value: hooks.SpawnContext{}},
	{Name: "SpawnPreCreatePayload", Value: hooks.SpawnPreCreatePayload{}},
	{Name: "SpawnLifecyclePayload", Value: hooks.SpawnLifecyclePayload{}},
	{Name: "SpawnCreatePatch", Value: hooks.SpawnCreatePatch{}},
	{Name: "AutonomyMatcher", Value: hooks.AutonomyMatcher{}},
	{Name: "HookMatcher", Value: hooks.HookMatcher{}},
	{Name: "HookDecl", Value: hooks.HookDecl{}},
}

// SDKRootTypes returns the canonical generated SDK contract roots.
func SDKRootTypes() []NamedType {
	return append([]NamedType(nil), sdkRootTypes...)
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

var namedHookTypes = map[string]NamedType{
	"PayloadBase":               {Name: "PayloadBase", Value: hooks.PayloadBase{}},
	"SessionContext":            {Name: "SessionContext", Value: hooks.SessionContext{}},
	"TurnContext":               {Name: "TurnContext", Value: hooks.TurnContext{}},
	"ContextBlock":              {Name: "ContextBlock", Value: hooks.ContextBlock{}},
	"ToolCallRef":               {Name: "ToolCallRef", Value: hooks.ToolCallRef{}},
	"ToolLocation":              {Name: "ToolLocation", Value: hooks.ToolLocation{}},
	"PermissionOption":          {Name: "PermissionOption", Value: hooks.PermissionOption{}},
	"PermissionToolCall":        {Name: "PermissionToolCall", Value: hooks.PermissionToolCall{}},
	"ControlPatch":              {Name: "ControlPatch", Value: hooks.ControlPatch{}},
	"SessionLifecyclePayload":   {Name: "SessionLifecyclePayload", Value: hooks.SessionLifecyclePayload{}},
	"SessionPreCreatePayload":   {Name: "SessionPreCreatePayload", Value: hooks.SessionPreCreatePayload{}},
	"SessionPostCreatePayload":  {Name: "SessionPostCreatePayload", Value: hooks.SessionPostCreatePayload{}},
	"SessionPreResumePayload":   {Name: "SessionPreResumePayload", Value: hooks.SessionPreResumePayload{}},
	"SessionPostResumePayload":  {Name: "SessionPostResumePayload", Value: hooks.SessionPostResumePayload{}},
	"SessionPreStopPayload":     {Name: "SessionPreStopPayload", Value: hooks.SessionPreStopPayload{}},
	"SessionPostStopPayload":    {Name: "SessionPostStopPayload", Value: hooks.SessionPostStopPayload{}},
	"SessionCreatePatch":        {Name: "SessionCreatePatch", Value: hooks.SessionCreatePatch{}},
	"SessionPostCreatePatch":    {Name: "SessionPostCreatePatch", Value: hooks.SessionPostCreatePatch{}},
	"SessionPreResumePatch":     {Name: "SessionPreResumePatch", Value: hooks.SessionPreResumePatch{}},
	"SessionPostResumePatch":    {Name: "SessionPostResumePatch", Value: hooks.SessionPostResumePatch{}},
	"SessionPreStopPatch":       {Name: "SessionPreStopPatch", Value: hooks.SessionPreStopPatch{}},
	"SessionPostStopPatch":      {Name: "SessionPostStopPatch", Value: hooks.SessionPostStopPatch{}},
	"EnvironmentProfilePayload": {Name: "EnvironmentProfilePayload", Value: hooks.EnvironmentProfilePayload{}},
	"EnvironmentPreparePayload": {Name: "EnvironmentPreparePayload", Value: hooks.EnvironmentPreparePayload{}},
	"EnvironmentReadyPayload":   {Name: "EnvironmentReadyPayload", Value: hooks.EnvironmentReadyPayload{}},
	"EnvironmentSyncBeforePayload": {
		Name:  "EnvironmentSyncBeforePayload",
		Value: hooks.EnvironmentSyncBeforePayload{},
	},
	"EnvironmentSyncAfterPayload": {
		Name:  "EnvironmentSyncAfterPayload",
		Value: hooks.EnvironmentSyncAfterPayload{},
	},
	"EnvironmentStopPayload":  {Name: "EnvironmentStopPayload", Value: hooks.EnvironmentStopPayload{}},
	"EnvironmentPreparePatch": {Name: "EnvironmentPreparePatch", Value: hooks.EnvironmentPreparePatch{}},
	"EnvironmentSyncBeforePatch": {
		Name:  "EnvironmentSyncBeforePatch",
		Value: hooks.EnvironmentSyncBeforePatch{},
	},
	"EnvironmentObservationPatch": {
		Name:  "EnvironmentObservationPatch",
		Value: hooks.EnvironmentObservationPatch{},
	},
	"EnvironmentReadyPatch":     {Name: "EnvironmentReadyPatch", Value: hooks.EnvironmentReadyPatch{}},
	"EnvironmentSyncAfterPatch": {Name: "EnvironmentSyncAfterPatch", Value: hooks.EnvironmentSyncAfterPatch{}},
	"EnvironmentStopPatch":      {Name: "EnvironmentStopPatch", Value: hooks.EnvironmentStopPatch{}},
	"InputPreSubmitPayload":     {Name: "InputPreSubmitPayload", Value: hooks.InputPreSubmitPayload{}},
	"InputPreSubmitPatch":       {Name: "InputPreSubmitPatch", Value: hooks.InputPreSubmitPatch{}},
	"PromptPayload":             {Name: "PromptPayload", Value: hooks.PromptPayload{}},
	"PromptPatch":               {Name: "PromptPatch", Value: hooks.PromptPatch{}},
	"EventRecordPayload":        {Name: "EventRecordPayload", Value: hooks.EventRecordPayload{}},
	"EventPreRecordPayload":     {Name: "EventPreRecordPayload", Value: hooks.EventPreRecordPayload{}},
	"EventPostRecordPayload":    {Name: "EventPostRecordPayload", Value: hooks.EventPostRecordPayload{}},
	"EventRecordPatch":          {Name: "EventRecordPatch", Value: hooks.EventRecordPatch{}},
	"EventPreRecordPatch":       {Name: "EventPreRecordPatch", Value: hooks.EventPreRecordPatch{}},
	"EventPostRecordPatch":      {Name: "EventPostRecordPatch", Value: hooks.EventPostRecordPatch{}},
	"AutomationSchedulePayload": {Name: "AutomationSchedulePayload", Value: hooks.AutomationSchedulePayload{}},
	"AutomationJobPreFirePayload": {
		Name:  "AutomationJobPreFirePayload",
		Value: hooks.AutomationJobPreFirePayload{},
	},
	"AutomationJobPostFirePayload": {
		Name:  "AutomationJobPostFirePayload",
		Value: hooks.AutomationJobPostFirePayload{},
	},
	"AutomationTriggerPreFirePayload": {
		Name:  "AutomationTriggerPreFirePayload",
		Value: hooks.AutomationTriggerPreFirePayload{},
	},
	"AutomationTriggerPostFirePayload": {
		Name:  "AutomationTriggerPostFirePayload",
		Value: hooks.AutomationTriggerPostFirePayload{},
	},
	"AutomationRunCompletedPayload": {
		Name:  "AutomationRunCompletedPayload",
		Value: hooks.AutomationRunCompletedPayload{},
	},
	"AutomationRunFailedPayload": {
		Name:  "AutomationRunFailedPayload",
		Value: hooks.AutomationRunFailedPayload{},
	},
	"AutomationFirePatch": {
		Name:  "AutomationFirePatch",
		Value: hooks.AutomationFirePatch{},
	},
	"AutomationObservationPatch": {
		Name:  "AutomationObservationPatch",
		Value: hooks.AutomationObservationPatch{},
	},
	"AgentPreStartPayload": {
		Name:  "AgentPreStartPayload",
		Value: hooks.AgentPreStartPayload{},
	},
	"AgentLifecyclePayload": {
		Name:  "AgentLifecyclePayload",
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
	"AgentStartPatch":             {Name: "AgentStartPatch", Value: hooks.AgentStartPatch{}},
	"AgentLifecyclePatch":         {Name: "AgentLifecyclePatch", Value: hooks.AgentLifecyclePatch{}},
	"AgentSpawnedPatch":           {Name: "AgentSpawnedPatch", Value: hooks.AgentSpawnedPatch{}},
	"AgentCrashedPatch":           {Name: "AgentCrashedPatch", Value: hooks.AgentCrashedPatch{}},
	"AgentStoppedPatch":           {Name: "AgentStoppedPatch", Value: hooks.AgentStoppedPatch{}},
	"TurnPayload":                 {Name: "TurnPayload", Value: hooks.TurnPayload{}},
	"TurnStartPayload":            {Name: "TurnStartPayload", Value: hooks.TurnStartPayload{}},
	"TurnEndPayload":              {Name: "TurnEndPayload", Value: hooks.TurnEndPayload{}},
	"TurnPatch":                   {Name: "TurnPatch", Value: hooks.TurnPatch{}},
	"TurnStartPatch":              {Name: "TurnStartPatch", Value: hooks.TurnStartPatch{}},
	"TurnEndPatch":                {Name: "TurnEndPatch", Value: hooks.TurnEndPatch{}},
	"MessagePayload":              {Name: "MessagePayload", Value: hooks.MessagePayload{}},
	"MessageStartPayload":         {Name: "MessageStartPayload", Value: hooks.MessageStartPayload{}},
	"MessageDeltaPayload":         {Name: "MessageDeltaPayload", Value: hooks.MessageDeltaPayload{}},
	"MessageEndPayload":           {Name: "MessageEndPayload", Value: hooks.MessageEndPayload{}},
	"MessagePatch":                {Name: "MessagePatch", Value: hooks.MessagePatch{}},
	"MessageStartPatch":           {Name: "MessageStartPatch", Value: hooks.MessageStartPatch{}},
	"MessageDeltaPatch":           {Name: "MessageDeltaPatch", Value: hooks.MessageDeltaPatch{}},
	"MessageEndPatch":             {Name: "MessageEndPatch", Value: hooks.MessageEndPatch{}},
	"ToolPreCallPayload":          {Name: "ToolPreCallPayload", Value: hooks.ToolPreCallPayload{}},
	"ToolPostCallPayload":         {Name: "ToolPostCallPayload", Value: hooks.ToolPostCallPayload{}},
	"ToolPostErrorPayload":        {Name: "ToolPostErrorPayload", Value: hooks.ToolPostErrorPayload{}},
	"ToolCallPatch":               {Name: "ToolCallPatch", Value: hooks.ToolCallPatch{}},
	"ToolResultPatch":             {Name: "ToolResultPatch", Value: hooks.ToolResultPatch{}},
	"ToolPostErrorPatch":          {Name: "ToolPostErrorPatch", Value: hooks.ToolPostErrorPatch{}},
	"PermissionRequestPayload":    {Name: "PermissionRequestPayload", Value: hooks.PermissionRequestPayload{}},
	"PermissionResolutionPayload": {Name: "PermissionResolutionPayload", Value: hooks.PermissionResolutionPayload{}},
	"PermissionResolvedPayload":   {Name: "PermissionResolvedPayload", Value: hooks.PermissionResolvedPayload{}},
	"PermissionDeniedPayload":     {Name: "PermissionDeniedPayload", Value: hooks.PermissionDeniedPayload{}},
	"PermissionRequestPatch":      {Name: "PermissionRequestPatch", Value: hooks.PermissionRequestPatch{}},
	"PermissionResolvedPatch":     {Name: "PermissionResolvedPatch", Value: hooks.PermissionResolvedPatch{}},
	"PermissionDeniedPatch":       {Name: "PermissionDeniedPatch", Value: hooks.PermissionDeniedPatch{}},
	"ContextCompactPayload":       {Name: "ContextCompactPayload", Value: hooks.ContextCompactPayload{}},
	"ContextPreCompactPayload":    {Name: "ContextPreCompactPayload", Value: hooks.ContextPreCompactPayload{}},
	"ContextPostCompactPayload":   {Name: "ContextPostCompactPayload", Value: hooks.ContextPostCompactPayload{}},
	"ContextCompactionPatch":      {Name: "ContextCompactionPatch", Value: hooks.ContextCompactionPatch{}},
	"ContextPreCompactPatch":      {Name: "ContextPreCompactPatch", Value: hooks.ContextPreCompactPatch{}},
	"ContextPostCompactPatch":     {Name: "ContextPostCompactPatch", Value: hooks.ContextPostCompactPatch{}},
	"AutonomyObservationPatch":    {Name: "AutonomyObservationPatch", Value: hooks.AutonomyObservationPatch{}},
	"CoordinatorContext":          {Name: "CoordinatorContext", Value: hooks.CoordinatorContext{}},
	"CoordinatorPreSpawnPayload": {
		Name:  "CoordinatorPreSpawnPayload",
		Value: hooks.CoordinatorPreSpawnPayload{},
	},
	"CoordinatorLifecyclePayload": {
		Name:  "CoordinatorLifecyclePayload",
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
	"CoordinatorSpawnPatch":       {Name: "CoordinatorSpawnPatch", Value: hooks.CoordinatorSpawnPatch{}},
	"CoordinatorObservationPatch": {Name: "CoordinatorObservationPatch", Value: hooks.CoordinatorObservationPatch{}},
	"TaskRunClaimCriteria":        {Name: "TaskRunClaimCriteria", Value: hooks.TaskRunClaimCriteria{}},
	"TaskRunContext":              {Name: "TaskRunContext", Value: hooks.TaskRunContext{}},
	"TaskRunEnqueuedPayload": {
		Name:  "TaskRunEnqueuedPayload",
		Value: hooks.TaskRunEnqueuedPayload{},
	},
	"TaskRunPreClaimPayload": {
		Name:  "TaskRunPreClaimPayload",
		Value: hooks.TaskRunPreClaimPayload{},
	},
	"TaskRunPostClaimPayload": {
		Name:  "TaskRunPostClaimPayload",
		Value: hooks.TaskRunPostClaimPayload{},
	},
	"TaskRunLeasePayload": {
		Name:  "TaskRunLeasePayload",
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
	"TaskRunPreClaimPatch":    {Name: "TaskRunPreClaimPatch", Value: hooks.TaskRunPreClaimPatch{}},
	"TaskRunObservationPatch": {Name: "TaskRunObservationPatch", Value: hooks.TaskRunObservationPatch{}},
	"PermissionSet":           {Name: "PermissionSet", Value: hooks.PermissionSet{}},
	"SpawnContext":            {Name: "SpawnContext", Value: hooks.SpawnContext{}},
	"SpawnPreCreatePayload": {
		Name:  "SpawnPreCreatePayload",
		Value: hooks.SpawnPreCreatePayload{},
	},
	"SpawnLifecyclePayload": {
		Name:  "SpawnLifecyclePayload",
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
	"SpawnCreatePatch":      {Name: "SpawnCreatePatch", Value: hooks.SpawnCreatePatch{}},
	"SpawnObservationPatch": {Name: "SpawnObservationPatch", Value: hooks.SpawnObservationPatch{}},
	"AutonomyMatcher":       {Name: "AutonomyMatcher", Value: hooks.AutonomyMatcher{}},
}

func namedHookType(name string) (NamedType, error) {
	namedType, ok := namedHookTypes[name]
	if !ok {
		return NamedType{}, fmt.Errorf("unknown hook contract type %q", name)
	}
	return namedType, nil
}
