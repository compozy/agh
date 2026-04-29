package builtin

import toolspkg "github.com/pedronauck/agh/internal/tools"

var taskTools = []toolspkg.Descriptor{
	nativeDescriptor(
		toolspkg.ToolIDTaskList,
		"task_list",
		"Task List",
		"List task summaries through the existing task service.",
		taskListInputSchema,
		toolspkg.RiskRead,
		true,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDTasks},
		[]string{"tasks", "coordination"},
		[]string{"task summaries", "task inbox"},
	),
	nativeDescriptor(
		toolspkg.ToolIDTaskRead,
		"task_read",
		"Task Read",
		"Read one task view through the existing task service.",
		taskReadInputSchema,
		toolspkg.RiskRead,
		true,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDTasks},
		[]string{"tasks", "coordination"},
		[]string{"task details", "task view"},
	),
	nativeDescriptor(
		toolspkg.ToolIDTaskCreate,
		"task_create",
		"Task Create",
		"Create one root task through the existing task service.",
		taskCreateInputSchema,
		toolspkg.RiskMutating,
		false,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDTasks},
		[]string{"tasks", "create"},
		[]string{"create task", "new task"},
	),
	nativeDescriptor(
		toolspkg.ToolIDTaskChildCreate,
		"task_child_create",
		"Task Child Create",
		"Create one child task through the task service.",
		taskChildCreateInputSchema,
		toolspkg.RiskMutating,
		false,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDTasks},
		[]string{"tasks", "create", "children"},
		[]string{"create child task", "task lineage"},
	),
	nativeDescriptor(
		toolspkg.ToolIDTaskUpdate,
		"task_update",
		"Task Update",
		"Update one task through the task service.",
		taskUpdateInputSchema,
		toolspkg.RiskMutating,
		false,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDTasks},
		[]string{"tasks", "update"},
		[]string{"edit task", "patch task"},
	),
	nativeDescriptor(
		toolspkg.ToolIDTaskCancel,
		"task_cancel",
		"Task Cancel",
		"Cancel one task through the task service.",
		taskCancelInputSchema,
		toolspkg.RiskDestructive,
		false,
		true,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDTasks},
		[]string{"tasks", "cancel"},
		[]string{"cancel task", "stop task tree"},
	),
	nativeDescriptor(
		toolspkg.ToolIDTaskRunList,
		"task_run_list",
		"Task Run List",
		"List task runs through the task service.",
		taskRunListInputSchema,
		toolspkg.RiskRead,
		true,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDTasks},
		[]string{"tasks", "runs"},
		[]string{"task runs", "run history"},
	),
}

func taskDescriptors() []toolspkg.Descriptor {
	return taskTools
}

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
