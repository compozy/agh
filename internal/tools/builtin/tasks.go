package builtin

import toolspkg "github.com/compozy/agh/internal/tools"

const (
	tasksNotificationsKey = "notifications"
)

const (
	tasksBridgesKey                = "bridges"
	tasksExecutionProfileKey       = "execution_profile"
	tasksReviewsKey                = "reviews"
	tasksRunsKey                   = "runs"
	tasksTaskExecutionProfileValue = "task execution profile"
	tasksTasksKey                  = "tasks"
)

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
		[]string{tasksTasksKey, "coordination"},
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
		[]string{tasksTasksKey, "coordination"},
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
		[]string{tasksTasksKey, "create"},
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
		[]string{tasksTasksKey, "create", "children"},
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
		[]string{tasksTasksKey, "update"},
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
		[]string{tasksTasksKey, "cancel"},
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
		[]string{tasksTasksKey, tasksRunsKey},
		[]string{"task runs", "run history"},
	),
	nativeDescriptor(
		toolspkg.ToolIDTaskRunReviewRequest,
		"task_run_review_request",
		"Task Run Review Request",
		"Request a persisted review for one terminal task run through the task service.",
		taskRunReviewRequestInputSchema,
		toolspkg.RiskMutating,
		false,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDTasks},
		[]string{tasksTasksKey, tasksReviewsKey},
		[]string{"request task review", "run review"},
	),
	nativeDescriptor(
		toolspkg.ToolIDTaskRunReviewList,
		"task_run_review_list",
		"Task Run Review List",
		"List persisted task-run reviews through the task service.",
		taskRunReviewListInputSchema,
		toolspkg.RiskRead,
		true,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDTasks},
		[]string{tasksTasksKey, tasksReviewsKey},
		[]string{"review history", "task reviews"},
	),
	nativeDescriptor(
		toolspkg.ToolIDTaskRunReviewShow,
		"task_run_review_show",
		"Task Run Review Show",
		"Read one persisted task-run review through the task service.",
		taskRunReviewShowInputSchema,
		toolspkg.RiskRead,
		true,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDTasks},
		[]string{tasksTasksKey, tasksReviewsKey},
		[]string{"review detail", "show task review"},
	),
	nativeDescriptor(
		toolspkg.ToolIDTaskExecutionProfileGet,
		"task_execution_profile_get",
		"Task Execution Profile Get",
		"Read one task execution profile through the task service.",
		taskExecutionProfileGetInputSchema,
		toolspkg.RiskRead,
		true,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDTasks},
		[]string{tasksTasksKey, tasksExecutionProfileKey},
		[]string{tasksTaskExecutionProfileValue, "profile get"},
	),
	nativeDescriptor(
		toolspkg.ToolIDTaskExecutionProfileSet,
		"task_execution_profile_set",
		"Task Execution Profile Set",
		"Update one task execution profile through the task service.",
		taskExecutionProfileSetInputSchema,
		toolspkg.RiskMutating,
		false,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDTasks},
		[]string{tasksTasksKey, tasksExecutionProfileKey},
		[]string{tasksTaskExecutionProfileValue, "profile set"},
	),
	nativeDescriptor(
		toolspkg.ToolIDTaskExecutionProfileDelete,
		"task_execution_profile_delete",
		"Task Execution Profile Delete",
		"Delete one task execution profile through the task service.",
		taskExecutionProfileDeleteInputSchema,
		toolspkg.RiskDestructive,
		false,
		true,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDTasks},
		[]string{tasksTasksKey, tasksExecutionProfileKey},
		[]string{tasksTaskExecutionProfileValue, "profile delete"},
	),
	nativeDescriptor(
		toolspkg.ToolIDTaskNotificationSubscribe,
		"task_notification_subscribe",
		"Task Notification Subscribe",
		"Create one bridge notification subscription for terminal task events.",
		taskNotificationSubscribeInputSchema,
		toolspkg.RiskMutating,
		false,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDTasks},
		[]string{tasksTasksKey, tasksNotificationsKey, tasksBridgesKey},
		[]string{"task notification subscribe", "bridge task subscription"},
	),
	nativeDescriptor(
		toolspkg.ToolIDTaskNotificationList,
		"task_notification_list",
		"Task Notification List",
		"List bridge notification subscriptions for one task.",
		taskNotificationListInputSchema,
		toolspkg.RiskRead,
		true,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDTasks},
		[]string{tasksTasksKey, tasksNotificationsKey, tasksBridgesKey},
		[]string{"task notification list", "bridge task subscriptions"},
	),
	nativeDescriptor(
		toolspkg.ToolIDTaskNotificationShow,
		"task_notification_show",
		"Task Notification Show",
		"Read one bridge notification subscription for one task.",
		taskNotificationShowInputSchema,
		toolspkg.RiskRead,
		true,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDTasks},
		[]string{tasksTasksKey, tasksNotificationsKey, tasksBridgesKey},
		[]string{"task notification show", "bridge task subscription detail"},
	),
	nativeDescriptor(
		toolspkg.ToolIDTaskNotificationDelete,
		"task_notification_delete",
		"Task Notification Delete",
		"Delete one bridge notification subscription for one task.",
		taskNotificationDeleteInputSchema,
		toolspkg.RiskDestructive,
		false,
		true,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDTasks},
		[]string{tasksTasksKey, tasksNotificationsKey, tasksBridgesKey},
		[]string{"task notification delete", "unsubscribe bridge task notification"},
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

const taskRunReviewRequestInputSchema = `{
	"type":"object",
	"required":["task_id","run_id"],
	"properties":{
		"task_id":{"type":"string"},
		"run_id":{"type":"string"},
		"policy":{"type":"string","enum":["","on_success","on_failure","always"]},
		"review_round":{"type":"integer"},
		"attempt":{"type":"integer"},
		"parent_review_id":{"type":"string"},
		"reason":{"type":"string"}
	},
	"additionalProperties":false
}`

const taskRunReviewListInputSchema = `{
	"type":"object",
	"properties":{
		"task_id":{"type":"string"},
		"run_id":{"type":"string"},
		"status":{"type":"string","enum":["","requested","routed","in_review","recorded","circuit_opened","canceled"]},
		"reviewer_session_id":{"type":"string"},
		"limit":{"type":"integer"}
	},
	"additionalProperties":false
}`

const taskRunReviewShowInputSchema = `{
	"type":"object",
	"required":["review_id"],
	"properties":{
		"review_id":{"type":"string"}
	},
	"additionalProperties":false
}`

const taskExecutionProfileGetInputSchema = `{
	"type":"object",
	"required":["task_id"],
	"properties":{
		"task_id":{"type":"string"}
	},
	"additionalProperties":false
}`

const taskExecutionProfileDeleteInputSchema = `{
	"type":"object",
	"required":["task_id"],
	"properties":{
		"task_id":{"type":"string"}
	},
	"additionalProperties":false
}`

const taskExecutionProfileSetInputSchema = `{
	"type":"object",
	"required":["task_id","profile"],
	"properties":{
		"task_id":{"type":"string"},
		"profile":` + taskExecutionProfileSchema + `
	},
	"additionalProperties":false
}`

const taskNotificationSubscribeInputSchema = `{
	"type":"object",
	"required":["task_id","bridge_instance_id"],
	"properties":{
		"task_id":{"type":"string"},
		"subscription_id":{"type":"string"},
		"bridge_instance_id":{"type":"string"},
		"scope":{"type":"string","enum":["","global","workspace"]},
		"workspace_id":{"type":"string"},
		"peer_id":{"type":"string"},
		"thread_id":{"type":"string"},
		"group_id":{"type":"string"},
		"delivery_mode":{"type":"string","enum":["","direct-send","reply"]}
	},
	"additionalProperties":false
}`

const taskNotificationListInputSchema = `{
	"type":"object",
	"required":["task_id"],
	"properties":{
		"task_id":{"type":"string"},
		"bridge_instance_id":{"type":"string"},
		"scope":{"type":"string","enum":["","global","workspace"]},
		"workspace_id":{"type":"string"},
		"limit":{"type":"integer"}
	},
	"additionalProperties":false
}`

const taskNotificationShowInputSchema = `{
	"type":"object",
	"required":["task_id","subscription_id"],
	"properties":{
		"task_id":{"type":"string"},
		"subscription_id":{"type":"string"}
	},
	"additionalProperties":false
}`

const taskNotificationDeleteInputSchema = `{
	"type":"object",
	"required":["task_id","subscription_id"],
	"properties":{
		"task_id":{"type":"string"},
		"subscription_id":{"type":"string"}
	},
	"additionalProperties":false
}`

const taskExecutionProfileSchema = `{
	"type":"object",
	"properties":{
		"task_id":{"type":"string"},
		"coordinator":` + coordinatorProfileSchema + `,
		"worker":` + workerProfileSchema + `,
		"review":` + reviewProfileSchema + `,
		"participants":` + participantPolicySchema + `,
		"sandbox":` + sandboxPolicySchema + `
	},
	"additionalProperties":false
}`

const coordinatorProfileSchema = `{
	"type":"object",
	"properties":{
		"mode":{"type":"string","enum":["","inherit","guided"]},
		"agent_name":{"type":"string"},
		"provider":{"type":"string"},
		"model":{"type":"string"},
		"guidance":{"type":"string"}
	},
	"additionalProperties":false
}`

const workerProfileSchema = `{
	"type":"object",
	"properties":{
		"mode":{"type":"string","enum":["","inherit","select"]},
		"agent_name":{"type":"string"},
		"provider":{"type":"string"},
		"model":{"type":"string"},
		"allowed_agent_names":{"type":"array","items":{"type":"string"}},
		"preferred_agent_names":{"type":"array","items":{"type":"string"}},
		"required_capabilities":{"type":"array","items":{"type":"string"}},
		"preferred_capabilities":{"type":"array","items":{"type":"string"}}
	},
	"additionalProperties":false
}`

const reviewProfileSchema = `{
	"type":"object",
	"properties":{
		"agent_name":{"type":"string"},
		"provider":{"type":"string"},
		"model":{"type":"string"},
		"allowed_agent_names":{"type":"array","items":{"type":"string"}},
		"preferred_agent_names":{"type":"array","items":{"type":"string"}},
		"allowed_channel_ids":{"type":"array","items":{"type":"string"}},
		"preferred_channel_ids":{"type":"array","items":{"type":"string"}},
		"allowed_peer_ids":{"type":"array","items":{"type":"string"}},
		"preferred_peer_ids":{"type":"array","items":{"type":"string"}},
		"required_capabilities":{"type":"array","items":{"type":"string"}},
		"preferred_capabilities":{"type":"array","items":{"type":"string"}}
	},
	"additionalProperties":false
}`

const participantPolicySchema = `{
	"type":"object",
	"properties":{
		"allowed_channel_ids":{"type":"array","items":{"type":"string"}},
		"preferred_channel_ids":{"type":"array","items":{"type":"string"}},
		"allowed_peer_ids":{"type":"array","items":{"type":"string"}},
		"preferred_peer_ids":{"type":"array","items":{"type":"string"}},
		"allowed_agent_names":{"type":"array","items":{"type":"string"}},
		"preferred_agent_names":{"type":"array","items":{"type":"string"}},
		"required_capabilities":{"type":"array","items":{"type":"string"}},
		"preferred_capabilities":{"type":"array","items":{"type":"string"}}
	},
	"additionalProperties":false
}`

const sandboxPolicySchema = `{
	"type":"object",
	"properties":{
		"mode":{"type":"string","enum":["","inherit","none","ref"]},
		"sandbox_ref":{"type":"string"}
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
