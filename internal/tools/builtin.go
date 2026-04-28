package tools

import "encoding/json"

const (
	// BuiltinSourceOwner is the source owner for daemon-compiled AGH tools.
	BuiltinSourceOwner = "daemon"
)

const (
	// ToolIDToolList lists tools in the caller's effective registry projection.
	ToolIDToolList ToolID = "agh__tool_list"
	// ToolIDToolSearch searches tools in the caller's effective registry projection.
	ToolIDToolSearch ToolID = "agh__tool_search"
	// ToolIDToolInfo reads one tool descriptor and diagnostics view.
	ToolIDToolInfo ToolID = "agh__tool_info"
	// ToolIDSkillList lists skills through the existing skill registry.
	ToolIDSkillList ToolID = "agh__skill_list"
	// ToolIDSkillSearch searches skills through the existing skill registry.
	ToolIDSkillSearch ToolID = "agh__skill_search"
	// ToolIDSkillView reads one skill and its verified body.
	ToolIDSkillView ToolID = "agh__skill_view"
	// ToolIDNetworkPeers lists visible network peers.
	ToolIDNetworkPeers ToolID = "agh__network_peers"
	// ToolIDNetworkSend sends one network message through the existing network manager.
	ToolIDNetworkSend ToolID = "agh__network_send"
	// ToolIDTaskList lists task summaries through the task service.
	ToolIDTaskList ToolID = "agh__task_list"
	// ToolIDTaskRead reads one task view through the task service.
	ToolIDTaskRead ToolID = "agh__task_read"
	// ToolIDTaskCreate creates one root task through the task service.
	ToolIDTaskCreate ToolID = "agh__task_create"
	// ToolIDTaskChildCreate creates one child task through the task service.
	ToolIDTaskChildCreate ToolID = "agh__task_child_create"
	// ToolIDTaskUpdate updates one task through the task service.
	ToolIDTaskUpdate ToolID = "agh__task_update"
	// ToolIDTaskCancel cancels one task through the task service.
	ToolIDTaskCancel ToolID = "agh__task_cancel"
	// ToolIDTaskRunList lists task runs through the task service.
	ToolIDTaskRunList ToolID = "agh__task_run_list"
)

const (
	// ToolsetIDBootstrap groups registry self-inspection tools.
	ToolsetIDBootstrap ToolsetID = "agh__bootstrap"
	// ToolsetIDCatalog groups registry and skill catalog tools.
	ToolsetIDCatalog ToolsetID = "agh__catalog"
	// ToolsetIDCoordination groups network coordination tools.
	ToolsetIDCoordination ToolsetID = "agh__coordination"
	// ToolsetIDTasks groups bounded task tools.
	ToolsetIDTasks ToolsetID = "agh__tasks"
)

var nativeBuiltinTools = []Descriptor{
	builtinDescriptor(
		ToolIDToolList,
		"tool_list",
		"Tool List",
		"List tools in the caller's effective registry projection.",
		toolListInputSchema,
		RiskRead,
		true,
		false,
		false,
		[]ToolsetID{ToolsetIDBootstrap, ToolsetIDCatalog},
		[]string{"tools", "registry", "catalog"},
		[]string{"available tools", "tool registry"},
	),
	builtinDescriptor(
		ToolIDToolSearch,
		"tool_search",
		"Tool Search",
		"Search tools in the caller's effective registry projection.",
		toolSearchInputSchema,
		RiskRead,
		true,
		false,
		false,
		[]ToolsetID{ToolsetIDBootstrap, ToolsetIDCatalog},
		[]string{"tools", "registry", "catalog"},
		[]string{"find tools", "tool registry search"},
	),
	builtinDescriptor(
		ToolIDToolInfo,
		"tool_info",
		"Tool Info",
		"Read one tool descriptor and effective diagnostics view.",
		toolInfoInputSchema,
		RiskRead,
		true,
		false,
		false,
		[]ToolsetID{ToolsetIDBootstrap, ToolsetIDCatalog},
		[]string{"tools", "registry", "diagnostics"},
		[]string{"tool descriptor", "tool policy diagnostics"},
	),
	builtinDescriptor(
		ToolIDSkillList,
		"skill_list",
		"Skill List",
		"List skills through the existing skill registry.",
		skillListInputSchema,
		RiskRead,
		true,
		false,
		false,
		[]ToolsetID{ToolsetIDCatalog},
		[]string{"skills", "catalog"},
		[]string{"available skills", "skill registry"},
	),
	builtinDescriptor(
		ToolIDSkillSearch,
		"skill_search",
		"Skill Search",
		"Search skills through the existing skill registry.",
		skillSearchInputSchema,
		RiskRead,
		true,
		false,
		false,
		[]ToolsetID{ToolsetIDCatalog},
		[]string{"skills", "catalog"},
		[]string{"find skills", "skill registry search"},
	),
	builtinDescriptor(
		ToolIDSkillView,
		"skill_view",
		"Skill View",
		"Read one skill and its verified body through the existing skill registry.",
		skillViewInputSchema,
		RiskRead,
		true,
		false,
		false,
		[]ToolsetID{ToolsetIDCatalog},
		[]string{"skills", "catalog", "content"},
		[]string{"skill body", "skill instructions"},
	),
	builtinDescriptor(
		ToolIDNetworkPeers,
		"network_peers",
		"Network Peers",
		"List visible AGH network peers through the existing network manager.",
		networkPeersInputSchema,
		RiskRead,
		true,
		false,
		false,
		[]ToolsetID{ToolsetIDCoordination},
		[]string{"network", "peers"},
		[]string{"network peers", "channel presence"},
	),
	builtinDescriptor(
		ToolIDNetworkSend,
		"network_send",
		"Network Send",
		"Send one AGH network message through the existing network manager.",
		networkSendInputSchema,
		RiskOpenWorld,
		false,
		false,
		true,
		[]ToolsetID{ToolsetIDCoordination},
		[]string{"network", "send"},
		[]string{"network message", "send to peer"},
	),
	builtinDescriptor(
		ToolIDTaskList,
		"task_list",
		"Task List",
		"List task summaries through the existing task service.",
		taskListInputSchema,
		RiskRead,
		true,
		false,
		false,
		[]ToolsetID{ToolsetIDTasks},
		[]string{"tasks", "coordination"},
		[]string{"task summaries", "task inbox"},
	),
	builtinDescriptor(
		ToolIDTaskRead,
		"task_read",
		"Task Read",
		"Read one task view through the existing task service.",
		taskReadInputSchema,
		RiskRead,
		true,
		false,
		false,
		[]ToolsetID{ToolsetIDTasks},
		[]string{"tasks", "coordination"},
		[]string{"task details", "task view"},
	),
	builtinDescriptor(
		ToolIDTaskCreate,
		"task_create",
		"Task Create",
		"Create one root task through the existing task service.",
		taskCreateInputSchema,
		RiskMutating,
		false,
		false,
		false,
		[]ToolsetID{ToolsetIDTasks},
		[]string{"tasks", "create"},
		[]string{"create task", "new task"},
	),
	builtinDescriptor(
		ToolIDTaskChildCreate,
		"task_child_create",
		"Task Child Create",
		"Create one child task through the existing task service.",
		taskChildCreateInputSchema,
		RiskMutating,
		false,
		false,
		false,
		[]ToolsetID{ToolsetIDTasks},
		[]string{"tasks", "create", "children"},
		[]string{"create child task", "task lineage"},
	),
	builtinDescriptor(
		ToolIDTaskUpdate,
		"task_update",
		"Task Update",
		"Update one task through the existing task service.",
		taskUpdateInputSchema,
		RiskMutating,
		false,
		false,
		false,
		[]ToolsetID{ToolsetIDTasks},
		[]string{"tasks", "update"},
		[]string{"edit task", "patch task"},
	),
	builtinDescriptor(
		ToolIDTaskCancel,
		"task_cancel",
		"Task Cancel",
		"Cancel one task through the existing task service.",
		taskCancelInputSchema,
		RiskDestructive,
		false,
		true,
		false,
		[]ToolsetID{ToolsetIDTasks},
		[]string{"tasks", "cancel"},
		[]string{"cancel task", "stop task tree"},
	),
	builtinDescriptor(
		ToolIDTaskRunList,
		"task_run_list",
		"Task Run List",
		"List task runs through the existing task service.",
		taskRunListInputSchema,
		RiskRead,
		true,
		false,
		false,
		[]ToolsetID{ToolsetIDTasks},
		[]string{"tasks", "runs"},
		[]string{"task runs", "run history"},
	),
}

// BuiltinSource returns the provenance shared by daemon-compiled AGH tools.
func BuiltinSource() SourceRef {
	return SourceRef{
		Kind:  SourceBuiltin,
		Owner: BuiltinSourceOwner,
		Scope: "daemon",
	}
}

// BuiltinNativeDescriptors returns the MVP native_go built-in descriptors.
func BuiltinNativeDescriptors() []Descriptor {
	descriptors := make([]Descriptor, 0, len(nativeBuiltinTools))
	for _, descriptor := range nativeBuiltinTools {
		descriptors = append(descriptors, cloneDescriptor(descriptor))
	}
	return descriptors
}

// BuiltinToolsetCatalog returns the built-in toolset definitions.
func BuiltinToolsetCatalog() (ToolsetCatalog, error) {
	return NewToolsetCatalog(
		Toolset{
			ID: ToolsetIDBootstrap,
			Tools: []string{
				ToolIDToolList.String(),
				ToolIDToolSearch.String(),
				ToolIDToolInfo.String(),
			},
		},
		Toolset{
			ID:       ToolsetIDCatalog,
			Tools:    []string{"agh__skill_*"},
			Toolsets: []ToolsetID{ToolsetIDBootstrap},
		},
		Toolset{
			ID: ToolsetIDCoordination,
			Tools: []string{
				ToolIDNetworkPeers.String(),
				ToolIDNetworkSend.String(),
			},
		},
		Toolset{
			ID:    ToolsetIDTasks,
			Tools: []string{"agh__task_*"},
		},
	)
}

func builtinDescriptor(
	id ToolID,
	nativeName string,
	title string,
	description string,
	inputSchema string,
	risk RiskClass,
	readOnly bool,
	destructive bool,
	openWorld bool,
	toolsets []ToolsetID,
	tags []string,
	searchHints []string,
) Descriptor {
	return Descriptor{
		ID:              id,
		Backend:         BackendRef{Kind: BackendNativeGo, NativeName: nativeName},
		DisplayTitle:    title,
		Description:     description,
		InputSchema:     json.RawMessage(inputSchema),
		OutputSchema:    json.RawMessage(`{"type":"object"}`),
		Source:          BuiltinSource(),
		Visibility:      VisibilityModel,
		Risk:            risk,
		ReadOnly:        readOnly,
		Destructive:     destructive,
		OpenWorld:       openWorld,
		ConcurrencySafe: true,
		Toolsets:        cloneToolsets(toolsets),
		Tags:            cloneStrings(tags),
		SearchHints:     cloneStrings(searchHints),
	}
}

const toolListInputSchema = `{
	"type":"object",
	"properties":{
		"limit":{"type":"integer"}
	},
	"additionalProperties":false
}`

const toolSearchInputSchema = `{
	"type":"object",
	"required":["query"],
	"properties":{
		"query":{"type":"string"},
		"limit":{"type":"integer"}
	},
	"additionalProperties":false
}`

const toolInfoInputSchema = `{
	"type":"object",
	"required":["tool_id"],
	"properties":{
		"tool_id":{"type":"string"}
	},
	"additionalProperties":false
}`

const skillListInputSchema = `{
	"type":"object",
	"properties":{
		"workspace_id":{"type":"string"},
		"limit":{"type":"integer"}
	},
	"additionalProperties":false
}`

const skillSearchInputSchema = `{
	"type":"object",
	"required":["query"],
	"properties":{
		"query":{"type":"string"},
		"workspace_id":{"type":"string"},
		"limit":{"type":"integer"}
	},
	"additionalProperties":false
}`

const skillViewInputSchema = `{
	"type":"object",
	"required":["name"],
	"properties":{
		"name":{"type":"string"},
		"workspace_id":{"type":"string"}
	},
	"additionalProperties":false
}`

const networkPeersInputSchema = `{
	"type":"object",
	"properties":{
		"channel":{"type":"string"}
	},
	"additionalProperties":false
}`

const networkSendInputSchema = `{
	"type":"object",
	"required":["channel","kind","body"],
	"properties":{
		"session_id":{"type":"string"},
		"channel":{"type":"string"},
		"kind":{"type":"string"},
		"to":{"type":"string"},
		"body":{"type":"object"},
		"interaction_id":{"type":"string"},
		"reply_to":{"type":"string"},
		"trace_id":{"type":"string"},
		"causation_id":{"type":"string"},
		"expires_at":{"type":"integer"},
		"id":{"type":"string"},
		"ext":{"type":"object"}
	},
	"additionalProperties":false
}`

const taskListInputSchema = `{
	"type":"object",
	"properties":{
		"scope":{"type":"string"},
		"workspace_id":{"type":"string"},
		"status":{"type":"string"},
		"priority":{"type":"string"},
		"approval_state":{"type":"string"},
		"owner_kind":{"type":"string"},
		"owner_ref":{"type":"string"},
		"parent_task_id":{"type":"string"},
		"network_channel":{"type":"string"},
		"search":{"type":"string"},
		"limit":{"type":"integer"}
	},
	"additionalProperties":false
}`

const taskReadInputSchema = `{
	"type":"object",
	"required":["task_id"],
	"properties":{
		"task_id":{"type":"string"}
	},
	"additionalProperties":false
}`

const taskCreateInputSchema = `{
	"type":"object",
	"required":["scope","title"],
	"properties":{
		"id":{"type":"string"},
		"identifier":{"type":"string"},
		"scope":{"type":"string"},
		"workspace_id":{"type":"string"},
		"network_channel":{"type":"string"},
		"title":{"type":"string"},
		"description":{"type":"string"},
		"priority":{"type":"string"},
		"max_attempts":{"type":"integer"},
		"draft":{"type":"boolean"},
		"approval_policy":{"type":"string"},
		"owner":` + ownerSchema + `,
		"metadata":{}
	},
	"additionalProperties":false
}`

const taskChildCreateInputSchema = `{
	"type":"object",
	"required":["parent_task_id","scope","title"],
	"properties":{
		"parent_task_id":{"type":"string"},
		"id":{"type":"string"},
		"identifier":{"type":"string"},
		"scope":{"type":"string"},
		"workspace_id":{"type":"string"},
		"network_channel":{"type":"string"},
		"title":{"type":"string"},
		"description":{"type":"string"},
		"priority":{"type":"string"},
		"max_attempts":{"type":"integer"},
		"draft":{"type":"boolean"},
		"approval_policy":{"type":"string"},
		"owner":` + ownerSchema + `,
		"metadata":{}
	},
	"additionalProperties":false
}`

const taskUpdateInputSchema = `{
	"type":"object",
	"required":["task_id"],
	"properties":{
		"task_id":{"type":"string"},
		"title":{"type":"string"},
		"description":{"type":"string"},
		"priority":{"type":"string"},
		"max_attempts":{"type":"integer"},
		"approval_policy":{"type":"string"},
		"metadata":{},
		"network_channel":{"type":"string"},
		"owner":` + ownerSchema + `,
		"clear_owner":{"type":"boolean"}
	},
	"additionalProperties":false
}`

const taskCancelInputSchema = `{
	"type":"object",
	"required":["task_id"],
	"properties":{
		"task_id":{"type":"string"},
		"reason":{"type":"string"},
		"metadata":{}
	},
	"additionalProperties":false
}`

const taskRunListInputSchema = `{
	"type":"object",
	"required":["task_id"],
	"properties":{
		"task_id":{"type":"string"},
		"status":{"type":"string"},
		"session_id":{"type":"string"},
		"coordination_channel_id":{"type":"string"},
		"limit":{"type":"integer"}
	},
	"additionalProperties":false
}`

const ownerSchema = `{
	"type":"object",
	"required":["kind","ref"],
	"properties":{
		"kind":{"type":"string"},
		"ref":{"type":"string"}
	},
	"additionalProperties":false
}`
