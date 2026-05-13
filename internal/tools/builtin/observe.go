package builtin

import toolspkg "github.com/pedronauck/agh/internal/tools"

var observeTools = []toolspkg.Descriptor{
	nativeDescriptor(
		toolspkg.ToolIDObserveEvents,
		"observe_events",
		"Observe Events",
		"Read redacted observability events through the current observe query surface.",
		observeEventsInputSchema,
		toolspkg.RiskRead,
		true,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDObserve},
		[]string{"observe", "events"},
		[]string{"observe events", "runtime events"},
	),
	nativeDescriptor(
		toolspkg.ToolIDObserveMetrics,
		"observe_metrics",
		"Observe Metrics",
		"Read daemon observability health and metrics through the current observe surface.",
		emptyInputSchema,
		toolspkg.RiskRead,
		true,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDObserve},
		[]string{"observe", "metrics", "health"},
		[]string{"observe health", "runtime metrics"},
	),
	nativeDescriptor(
		toolspkg.ToolIDObserveSearch,
		"observe_search",
		"Observe Search",
		"Search redacted observability events using current observe filters.",
		observeSearchInputSchema,
		toolspkg.RiskRead,
		true,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDObserve},
		[]string{"observe", "search"},
		[]string{"observe search", "search runtime events"},
	),
}

func observeDescriptors() []toolspkg.Descriptor {
	return observeTools
}

const observeEventsInputSchema = `{
	"type":"object",
	"required":["workspace_id"],
	"properties":{
		"workspace_id":{"type":"string"},
		"session_id":{"type":"string"},
		"agent_name":{"type":"string"},
		"type":{"type":"string"},
		"since":{"type":"string"},
		"limit":{"type":"integer"}
	},
	"additionalProperties":false
}`

const observeSearchInputSchema = `{
	"type":"object",
	"required":["workspace_id","query"],
	"properties":{
		"workspace_id":{"type":"string"},
		"query":{"type":"string"},
		"session_id":{"type":"string"},
		"agent_name":{"type":"string"},
		"type":{"type":"string"},
		"since":{"type":"string"},
		"limit":{"type":"integer"}
	},
	"additionalProperties":false
}`
