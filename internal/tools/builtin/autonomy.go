package builtin

import toolspkg "github.com/pedronauck/agh/internal/tools"

var autonomyTools = []toolspkg.Descriptor{
	nativeDescriptor(
		toolspkg.ToolIDTaskRunClaimNext,
		"task_run_claim_next",
		"Task Run Claim Next",
		"Claim the next eligible task run for the caller session.",
		autonomyClaimNextInputSchema,
		toolspkg.RiskMutating,
		false,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDAutonomy},
		[]string{"autonomy", "tasks", "runs"},
		[]string{"claim task run", "next work"},
	),
	nativeDescriptor(
		toolspkg.ToolIDTaskRunHeartbeat,
		"task_run_heartbeat",
		"Task Run Heartbeat",
		"Extend the caller session's active task-run lease.",
		autonomyHeartbeatInputSchema,
		toolspkg.RiskMutating,
		false,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDAutonomy},
		[]string{"autonomy", "tasks", "leases"},
		[]string{"heartbeat task run", "extend lease"},
	),
	nativeDescriptor(
		toolspkg.ToolIDTaskRunComplete,
		"task_run_complete",
		"Task Run Complete",
		"Complete the caller session's active task-run lease.",
		autonomyCompleteInputSchema,
		toolspkg.RiskMutating,
		false,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDAutonomy},
		[]string{"autonomy", "tasks", "leases"},
		[]string{"complete task run", "finish work"},
	),
	nativeDescriptor(
		toolspkg.ToolIDTaskRunFail,
		"task_run_fail",
		"Task Run Fail",
		"Fail the caller session's active task-run lease.",
		autonomyFailInputSchema,
		toolspkg.RiskMutating,
		false,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDAutonomy},
		[]string{"autonomy", "tasks", "leases"},
		[]string{"fail task run", "report failure"},
	),
	nativeDescriptor(
		toolspkg.ToolIDTaskRunRelease,
		"task_run_release",
		"Task Run Release",
		"Release the caller session's active task-run lease back to the queue.",
		autonomyReleaseInputSchema,
		toolspkg.RiskMutating,
		false,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDAutonomy},
		[]string{"autonomy", "tasks", "leases"},
		[]string{"release task run", "handoff work"},
	),
}

func autonomyDescriptors() []toolspkg.Descriptor {
	return autonomyTools
}

const autonomyClaimNextInputSchema = `{
	"type":"object",
	"properties":{
		"workspace_id":{"type":"string"},
		"required_capabilities":{"type":"array","items":{"type":"string"}},
		"priority_min":{"type":"integer"},
		"lease_seconds":{"type":"integer"}
	},
	"additionalProperties":false
}`

const autonomyHeartbeatInputSchema = `{
	"type":"object",
	"required":["run_id"],
	"properties":{
		"run_id":{"type":"string"},
		"lease_seconds":{"type":"integer"}
	},
	"additionalProperties":false
}`

const autonomyCompleteInputSchema = `{
	"type":"object",
	"required":["run_id"],
	"properties":{
		"run_id":{"type":"string"},
		"result":{}
	},
	"additionalProperties":false
}`

const autonomyFailInputSchema = `{
	"type":"object",
	"required":["run_id","error"],
	"properties":{
		"run_id":{"type":"string"},
		"error":{"type":"string"},
		"metadata":{}
	},
	"additionalProperties":false
}`

const autonomyReleaseInputSchema = `{
	"type":"object",
	"required":["run_id"],
	"properties":{
		"run_id":{"type":"string"},
		"reason":{"type":"string"}
	},
	"additionalProperties":false
}`
