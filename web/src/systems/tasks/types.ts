import type { OperationQuery, OperationRequestBody, OperationResponse } from "@/lib/api-contract";

export type TaskListItem = OperationResponse<"listTasks", 200>["tasks"][number];
export type TaskDetailView = OperationResponse<"getTask", 200>["task"];
export type TaskSummary = TaskDetailView["summary"];
export type TaskRecord = TaskDetailView["task"];
export type TaskChildSummary = NonNullable<TaskDetailView["children"]>[number];
export type TaskRun = OperationResponse<"listTaskRuns", 200>["runs"][number];
export type TaskRunDetailView = OperationResponse<"getTaskRun", 200>["run"];
export type TaskTimelineItem = OperationResponse<"getTaskTimeline", 200>["timeline"][number];
export type TaskTreeView = OperationResponse<"getTaskTree", 200>["tree"];
export type TaskTreeNode = TaskTreeView["root"];
export type TaskDashboardView = OperationResponse<"getTaskDashboard", 200>["dashboard"];
export type TaskInboxView = OperationResponse<"getTaskInbox", 200>["inbox"];
export type TaskInboxGroup = NonNullable<TaskInboxView["groups"]>[number];
export type TaskInboxItem = NonNullable<TaskInboxGroup["items"]>[number];
export type TaskTriageState = OperationResponse<"markTaskRead", 200>["triage"];
export type TaskStreamPayload = OperationResponse<"streamTask", 200>;

export type TaskListFilter = OperationQuery<"listTasks">;
export type TaskDashboardFilter = OperationQuery<"getTaskDashboard">;
export type TaskInboxFilter = OperationQuery<"getTaskInbox">;
export type TaskRunsFilter = OperationQuery<"listTaskRuns">;
export type TaskTimelineFilter = OperationQuery<"getTaskTimeline">;
export type TaskStreamFilter = OperationQuery<"streamTask">;

export type CreateTaskRequest = OperationRequestBody<"createTask">;
export type UpdateTaskRequest = OperationRequestBody<"updateTask">;
export type CancelTaskRequest = OperationRequestBody<"cancelTask">;
export type CreateChildTaskRequest = OperationRequestBody<"createChildTask">;
export type AddTaskDependencyRequest = OperationRequestBody<"addTaskDependency">;
export type EnqueueTaskRunRequest = OperationRequestBody<"enqueueTaskRun">;
export type AttachTaskRunSessionRequest = OperationRequestBody<"attachTaskRunSession">;
export type CancelTaskRunRequest = OperationRequestBody<"cancelTaskRun">;
export type ClaimTaskRunRequest = OperationRequestBody<"claimTaskRun">;
export type CompleteTaskRunRequest = OperationRequestBody<"completeTaskRun">;
export type FailTaskRunRequest = OperationRequestBody<"failTaskRun">;
export type StartTaskRunRequest = OperationRequestBody<"startTaskRun">;

export type TaskStatus = TaskRecord["status"];
export type TaskPriority = NonNullable<TaskRecord["priority"]>;
export type TaskScope = TaskRecord["scope"];
export type TaskApprovalPolicy = NonNullable<TaskRecord["approval_policy"]>;
export type TaskApprovalState = NonNullable<TaskRecord["approval_state"]>;
export type TaskOwnerKind = NonNullable<NonNullable<TaskRecord["owner"]>["kind"]>;
export type TaskRunStatus = TaskRun["status"];
export type TaskInboxLane = TaskInboxItem["lane"];

export type TaskViewMode = "list" | "kanban" | "dashboard" | "inbox";
