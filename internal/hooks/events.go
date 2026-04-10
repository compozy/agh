package hooks

import "fmt"

// HookEventFamily groups hook events into the documented taxonomy families.
type HookEventFamily string

const (
	HookEventFamilySession    HookEventFamily = "session"
	HookEventFamilyInput      HookEventFamily = "input"
	HookEventFamilyPrompt     HookEventFamily = "prompt"
	HookEventFamilyEvent      HookEventFamily = "event"
	HookEventFamilyAgent      HookEventFamily = "agent"
	HookEventFamilyTurn       HookEventFamily = "turn"
	HookEventFamilyMessage    HookEventFamily = "message"
	HookEventFamilyTool       HookEventFamily = "tool"
	HookEventFamilyPermission HookEventFamily = "permission"
	HookEventFamilyContext    HookEventFamily = "context"
)

// Validate ensures the event family is part of the supported taxonomy.
func (f HookEventFamily) Validate() error {
	switch f {
	case HookEventFamilySession,
		HookEventFamilyInput,
		HookEventFamilyPrompt,
		HookEventFamilyEvent,
		HookEventFamilyAgent,
		HookEventFamilyTurn,
		HookEventFamilyMessage,
		HookEventFamilyTool,
		HookEventFamilyPermission,
		HookEventFamilyContext:
		return nil
	default:
		return fmt.Errorf("hooks: invalid hook event family %q", f)
	}
}

// HookEvent identifies when a hook fires.
type HookEvent string

const (
	HookSessionPreCreate  HookEvent = "session.pre_create"
	HookSessionPostCreate HookEvent = "session.post_create"
	HookSessionPreResume  HookEvent = "session.pre_resume"
	HookSessionPostResume HookEvent = "session.post_resume"
	HookSessionPreStop    HookEvent = "session.pre_stop"
	HookSessionPostStop   HookEvent = "session.post_stop"

	HookInputPreSubmit HookEvent = "input.pre_submit"

	HookPromptPostAssemble HookEvent = "prompt.post_assemble"

	HookEventPreRecord  HookEvent = "event.pre_record"
	HookEventPostRecord HookEvent = "event.post_record"

	HookAgentPreStart HookEvent = "agent.pre_start"
	HookAgentSpawned  HookEvent = "agent.spawned"
	HookAgentCrashed  HookEvent = "agent.crashed"
	HookAgentStopped  HookEvent = "agent.stopped"

	HookTurnStart HookEvent = "turn.start"
	HookTurnEnd   HookEvent = "turn.end"

	HookMessageStart HookEvent = "message.start"
	HookMessageDelta HookEvent = "message.delta"
	HookMessageEnd   HookEvent = "message.end"

	HookToolPreCall   HookEvent = "tool.pre_call"
	HookToolPostCall  HookEvent = "tool.post_call"
	HookToolPostError HookEvent = "tool.post_error"

	HookPermissionRequest  HookEvent = "permission.request"
	HookPermissionResolved HookEvent = "permission.resolved"
	HookPermissionDenied   HookEvent = "permission.denied"

	HookContextPreCompact  HookEvent = "context.pre_compact"
	HookContextPostCompact HookEvent = "context.post_compact"
)

type hookEventSpec struct {
	family       HookEventFamily
	syncEligible bool
}

var hookEventSpecs = map[HookEvent]hookEventSpec{
	HookSessionPreCreate:  {family: HookEventFamilySession, syncEligible: true},
	HookSessionPostCreate: {family: HookEventFamilySession, syncEligible: true},
	HookSessionPreResume:  {family: HookEventFamilySession, syncEligible: true},
	HookSessionPostResume: {family: HookEventFamilySession, syncEligible: true},
	HookSessionPreStop:    {family: HookEventFamilySession, syncEligible: true},
	HookSessionPostStop:   {family: HookEventFamilySession, syncEligible: true},
	HookInputPreSubmit:    {family: HookEventFamilyInput, syncEligible: true},
	HookPromptPostAssemble: {
		family:       HookEventFamilyPrompt,
		syncEligible: true,
	},
	HookEventPreRecord:  {family: HookEventFamilyEvent, syncEligible: false},
	HookEventPostRecord: {family: HookEventFamilyEvent, syncEligible: false},
	HookAgentPreStart:   {family: HookEventFamilyAgent, syncEligible: true},
	HookAgentSpawned:    {family: HookEventFamilyAgent, syncEligible: true},
	HookAgentCrashed:    {family: HookEventFamilyAgent, syncEligible: true},
	HookAgentStopped:    {family: HookEventFamilyAgent, syncEligible: true},
	HookTurnStart:       {family: HookEventFamilyTurn, syncEligible: true},
	HookTurnEnd:         {family: HookEventFamilyTurn, syncEligible: true},
	HookMessageStart:    {family: HookEventFamilyMessage, syncEligible: true},
	HookMessageDelta:    {family: HookEventFamilyMessage, syncEligible: false},
	HookMessageEnd:      {family: HookEventFamilyMessage, syncEligible: true},
	HookToolPreCall:     {family: HookEventFamilyTool, syncEligible: true},
	HookToolPostCall:    {family: HookEventFamilyTool, syncEligible: true},
	HookToolPostError:   {family: HookEventFamilyTool, syncEligible: true},
	HookPermissionRequest: {
		family:       HookEventFamilyPermission,
		syncEligible: true,
	},
	HookPermissionResolved: {
		family:       HookEventFamilyPermission,
		syncEligible: false,
	},
	HookPermissionDenied: {
		family:       HookEventFamilyPermission,
		syncEligible: false,
	},
	HookContextPreCompact:  {family: HookEventFamilyContext, syncEligible: true},
	HookContextPostCompact: {family: HookEventFamilyContext, syncEligible: true},
}

var allHookEvents = []HookEvent{
	HookSessionPreCreate,
	HookSessionPostCreate,
	HookSessionPreResume,
	HookSessionPostResume,
	HookSessionPreStop,
	HookSessionPostStop,
	HookInputPreSubmit,
	HookPromptPostAssemble,
	HookEventPreRecord,
	HookEventPostRecord,
	HookAgentPreStart,
	HookAgentSpawned,
	HookAgentCrashed,
	HookAgentStopped,
	HookTurnStart,
	HookTurnEnd,
	HookMessageStart,
	HookMessageDelta,
	HookMessageEnd,
	HookToolPreCall,
	HookToolPostCall,
	HookToolPostError,
	HookPermissionRequest,
	HookPermissionResolved,
	HookPermissionDenied,
	HookContextPreCompact,
	HookContextPostCompact,
}

func init() {
	if err := validateHookEventSpecsConsistency(); err != nil {
		panic(err)
	}
}

// AllHookEvents returns the full taxonomy in deterministic order.
func AllHookEvents() []HookEvent {
	events := make([]HookEvent, len(allHookEvents))
	copy(events, allHookEvents)
	return events
}

// String returns the literal hook event value.
func (e HookEvent) String() string {
	return string(e)
}

// Family reports the taxonomy family for the event.
func (e HookEvent) Family() HookEventFamily {
	spec, ok := hookEventSpecs[e]
	if !ok {
		return ""
	}
	return spec.family
}

// SyncEligible reports whether the event accepts sync hooks.
func (e HookEvent) SyncEligible() bool {
	spec, ok := hookEventSpecs[e]
	return ok && spec.syncEligible
}

// Validate ensures the event is part of the supported taxonomy.
func (e HookEvent) Validate() error {
	if _, ok := hookEventSpecs[e]; !ok {
		return fmt.Errorf("hooks: invalid hook event %q", e)
	}
	return nil
}

func validateHookEventSpecsConsistency() error {
	eventsFromList := make(map[HookEvent]struct{}, len(allHookEvents))
	for _, event := range allHookEvents {
		eventsFromList[event] = struct{}{}
		if _, ok := hookEventSpecs[event]; !ok {
			return fmt.Errorf("hooks: event %q exists in allHookEvents but is missing from hookEventSpecs", event)
		}
	}
	for event := range hookEventSpecs {
		if _, ok := eventsFromList[event]; !ok {
			return fmt.Errorf("hooks: event %q exists in hookEventSpecs but is missing from allHookEvents", event)
		}
	}
	return nil
}
