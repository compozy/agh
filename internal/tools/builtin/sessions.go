package builtin

import toolspkg "github.com/pedronauck/agh/internal/tools"

const (
	sessionsSessionsKey = "sessions"
)

const (
	sessionsEventsKey  = "events"
	sessionsHistoryKey = "history"
	sessionsListKey    = "list"
	sessionsStatusKey  = "status"
)

var sessionTools = []toolspkg.Descriptor{
	nativeDescriptor(
		toolspkg.ToolIDSessionList,
		"session_list",
		"Session List",
		"List runtime sessions through the existing session query surface.",
		sessionListInputSchema,
		toolspkg.RiskRead,
		true,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDSessions},
		[]string{sessionsSessionsKey, sessionsListKey},
		[]string{"session list", "runtime sessions"},
	),
	nativeDescriptor(
		toolspkg.ToolIDSessionStatus,
		"session_status",
		"Session Status",
		"Read one runtime session snapshot through the existing session status surface.",
		sessionIDInputSchema,
		toolspkg.RiskRead,
		true,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDSessions},
		[]string{sessionsSessionsKey, sessionsStatusKey},
		[]string{"session status", "session snapshot"},
	),
	nativeDescriptor(
		toolspkg.ToolIDSessionHistory,
		"session_history",
		"Session History",
		"Read grouped turn history for one runtime session.",
		sessionEventQueryInputSchema,
		toolspkg.RiskRead,
		true,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDSessions},
		[]string{sessionsSessionsKey, sessionsHistoryKey},
		[]string{"session history", "turn history"},
	),
	nativeDescriptor(
		toolspkg.ToolIDSessionEvents,
		"session_events",
		"Session Events",
		"Read persisted events for one runtime session.",
		sessionEventQueryInputSchema,
		toolspkg.RiskRead,
		true,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDSessions},
		[]string{sessionsSessionsKey, sessionsEventsKey},
		[]string{"session events", "event log"},
	),
	nativeDescriptor(
		toolspkg.ToolIDSessionDescribe,
		"session_describe",
		"Session Describe",
		"Read a composite read-only session description with status, events, and history.",
		sessionEventQueryInputSchema,
		toolspkg.RiskRead,
		true,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDSessions},
		[]string{sessionsSessionsKey, "describe"},
		[]string{"session describe", "session detail"},
	),
}

func sessionDescriptors() []toolspkg.Descriptor {
	return sessionTools
}

const sessionListInputSchema = `{
	"type":"object",
	"properties":{
		"workspace":{"type":"string"},
		"limit":{"type":"integer"}
	},
	"additionalProperties":false
}`

const sessionIDInputSchema = `{
	"type":"object",
	"required":["workspace_id","session_id"],
	"properties":{
		"workspace_id":{"type":"string"},
		"session_id":{"type":"string"}
	},
	"additionalProperties":false
}`

const sessionEventQueryInputSchema = `{
	"type":"object",
	"required":["workspace_id","session_id"],
	"properties":{
		"workspace_id":{"type":"string"},
		"session_id":{"type":"string"},
		"type":{"type":"string"},
		"agent_name":{"type":"string"},
		"turn_id":{"type":"string"},
		"after_sequence":{"type":"integer"},
		"limit":{"type":"integer"},
		"since":{"type":"string"}
	},
	"additionalProperties":false
}`
