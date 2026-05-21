import {
  apiClient,
  apiRequestFailed,
  defaultApiErrorMessage,
  requireResponseData,
} from "@/lib/api-client";

import type {
  AddTaskDependencyRequest,
  AgentContextView,
  AttachTaskRunSessionRequest,
  CancelTaskRequest,
  CancelTaskRunRequest,
  ClaimTaskRunRequest,
  CompleteTaskRunRequest,
  CreateChildTaskRequest,
  CreateTaskRequest,
  EnqueueTaskRunRequest,
  FailTaskRunRequest,
  ForceFailTaskRunRequest,
  ForceReleaseTaskRunRequest,
  PauseTaskRequest,
  RetryTaskRunRequest,
  RetryTaskRunResult,
  ResumeTaskRequest,
  StartTaskRunRequest,
  TaskBridgeNotificationSubscription,
  TaskBridgeNotificationSubscriptionCreateRequest,
  TaskBridgeNotificationSubscriptionsFilter,
  TaskContextBundle,
  TaskDashboardFilter,
  TaskDashboardView,
  TaskDetailView,
  TaskExecutionProfile,
  TaskExecutionProfileSetRequest,
  TaskInboxFilter,
  TaskInboxView,
  TaskInspectView,
  TaskListFilter,
  TaskListItem,
  TaskRecord,
  TaskReviewsFilter,
  TaskRun,
  TaskRunDetailView,
  TaskRunInspectView,
  TaskRunReview,
  TaskRunReviewRequest,
  TaskRunReviewRequestResult,
  TaskRunReviewVerdictRequest,
  TaskRunReviewVerdictResult,
  TaskRunReviewsFilter,
  TaskRunsFilter,
  TaskStreamFilter,
  TaskTimelineFilter,
  TaskTimelineItem,
  TaskTreeView,
  TaskTriageState,
  UpdateTaskRequest,
} from "../types";

export class TasksApiError extends Error {
  constructor(
    message: string,
    public readonly status: number
  ) {
    super(message);
    this.name = "TasksApiError";
  }
}

export function buildTaskStreamUrl(taskId: string, filters: TaskStreamFilter = {}): string {
  const trimmedId = taskId.trim();
  if (trimmedId === "") {
    throw new TasksApiError("task id is required to build stream url", 400);
  }
  const path = `/api/tasks/${encodeURIComponent(trimmedId)}/stream`;
  if (filters.after_sequence === undefined) {
    return path;
  }
  return `${path}?after_sequence=${encodeURIComponent(String(filters.after_sequence))}`;
}

function normalizeOptionalText(value?: string | null): string | undefined {
  if (typeof value !== "string") {
    return undefined;
  }

  const normalized = value.trim();
  return normalized === "" ? undefined : normalized;
}

function normalizeListFilter(filters: TaskListFilter = {}): TaskListFilter {
  return {
    scope: filters.scope,
    workspace: normalizeOptionalText(filters.workspace),
    status: filters.status,
    priority: filters.priority,
    include_drafts: filters.include_drafts,
    approval_state: filters.approval_state,
    owner_kind: filters.owner_kind,
    owner_ref: normalizeOptionalText(filters.owner_ref),
    parent_task_id: normalizeOptionalText(filters.parent_task_id),
    network_channel: normalizeOptionalText(filters.network_channel),
    query: normalizeOptionalText(filters.query),
    limit: filters.limit,
  };
}

function normalizeDashboardFilter(filters: TaskDashboardFilter = {}): TaskDashboardFilter {
  return {
    scope: filters.scope,
    workspace: normalizeOptionalText(filters.workspace),
    owner_kind: filters.owner_kind,
    owner_ref: normalizeOptionalText(filters.owner_ref),
    network_channel: normalizeOptionalText(filters.network_channel),
    origin_kind: filters.origin_kind,
  };
}

function normalizeInboxFilter(filters: TaskInboxFilter = {}): TaskInboxFilter {
  return {
    scope: filters.scope,
    workspace: normalizeOptionalText(filters.workspace),
    owner_kind: filters.owner_kind,
    owner_ref: normalizeOptionalText(filters.owner_ref),
    lane: filters.lane,
    unread: filters.unread,
    query: normalizeOptionalText(filters.query),
    limit: filters.limit,
  };
}

export async function listTasks(
  filters: TaskListFilter = {},
  signal?: AbortSignal
): Promise<TaskListItem[]> {
  const { data, error, response } = await apiClient.GET("/api/tasks", {
    params: { query: normalizeListFilter(filters) },
    signal,
  });

  if (apiRequestFailed(response, error)) {
    throw new TasksApiError(
      defaultApiErrorMessage("Failed to fetch tasks", response, error),
      response.status
    );
  }

  return requireResponseData(data, response, "Failed to fetch tasks").tasks;
}

export async function getTask(id: string, signal?: AbortSignal): Promise<TaskDetailView> {
  const { data, error, response } = await apiClient.GET("/api/tasks/{id}", {
    params: { path: { id } },
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new TasksApiError(`Task not found: ${id}`, 404);
    }

    throw new TasksApiError(
      defaultApiErrorMessage(`Failed to fetch task "${id}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to fetch task "${id}"`).task;
}

export async function inspectTask(id: string, signal?: AbortSignal): Promise<TaskInspectView> {
  const { data, error, response } = await apiClient.GET("/api/tasks/{id}/inspect", {
    params: { path: { id } },
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new TasksApiError(`Task not found: ${id}`, 404);
    }

    throw new TasksApiError(
      defaultApiErrorMessage(`Failed to inspect task "${id}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to inspect task "${id}"`).inspect;
}

export async function deleteTask(id: string, signal?: AbortSignal): Promise<void> {
  const { error, response } = await apiClient.DELETE("/api/tasks/{id}", {
    params: { path: { id } },
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new TasksApiError(`Task not found: ${id}`, 404);
    }

    throw new TasksApiError(
      defaultApiErrorMessage(`Failed to delete task "${id}"`, response, error),
      response.status
    );
  }
}

export async function createTask(
  body: CreateTaskRequest,
  signal?: AbortSignal
): Promise<TaskRecord> {
  const { data, error, response } = await apiClient.POST("/api/tasks", { body, signal });

  if (apiRequestFailed(response, error)) {
    throw new TasksApiError(
      defaultApiErrorMessage("Failed to create task", response, error),
      response.status
    );
  }

  return requireResponseData(data, response, "Failed to create task").task;
}

export async function updateTask(
  id: string,
  body: UpdateTaskRequest,
  signal?: AbortSignal
): Promise<TaskRecord> {
  const { data, error, response } = await apiClient.PATCH("/api/tasks/{id}", {
    params: { path: { id } },
    body,
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new TasksApiError(`Task not found: ${id}`, 404);
    }

    throw new TasksApiError(
      defaultApiErrorMessage(`Failed to update task "${id}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to update task "${id}"`).task;
}

export async function publishTask(id: string, signal?: AbortSignal): Promise<TaskRecord> {
  const { data, error, response } = await apiClient.POST("/api/tasks/{id}/publish", {
    params: { path: { id } },
    body: {},
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new TasksApiError(`Task not found: ${id}`, 404);
    }

    throw new TasksApiError(
      defaultApiErrorMessage(`Failed to publish task "${id}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to publish task "${id}"`).task;
}

export async function cancelTask(
  id: string,
  body: CancelTaskRequest = {},
  signal?: AbortSignal
): Promise<TaskRecord> {
  const { data, error, response } = await apiClient.POST("/api/tasks/{id}/cancel", {
    params: { path: { id } },
    body,
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new TasksApiError(`Task not found: ${id}`, 404);
    }

    throw new TasksApiError(
      defaultApiErrorMessage(`Failed to cancel task "${id}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to cancel task "${id}"`).task;
}

export async function pauseTask(
  id: string,
  body: PauseTaskRequest,
  signal?: AbortSignal
): Promise<TaskRecord> {
  const { data, error, response } = await apiClient.POST("/api/tasks/{id}/pause", {
    params: { path: { id } },
    body,
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new TasksApiError(`Task not found: ${id}`, 404);
    }

    throw new TasksApiError(
      defaultApiErrorMessage(`Failed to pause task "${id}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to pause task "${id}"`).task;
}

export async function resumeTask(
  id: string,
  body: ResumeTaskRequest = {},
  signal?: AbortSignal
): Promise<TaskRecord> {
  const { data, error, response } = await apiClient.POST("/api/tasks/{id}/resume", {
    params: { path: { id } },
    body,
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new TasksApiError(`Task not found: ${id}`, 404);
    }

    throw new TasksApiError(
      defaultApiErrorMessage(`Failed to resume task "${id}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to resume task "${id}"`).task;
}

export async function approveTask(id: string, signal?: AbortSignal): Promise<TaskRecord> {
  const { data, error, response } = await apiClient.POST("/api/tasks/{id}/approve", {
    params: { path: { id } },
    body: {},
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new TasksApiError(`Task not found: ${id}`, 404);
    }

    throw new TasksApiError(
      defaultApiErrorMessage(`Failed to approve task "${id}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to approve task "${id}"`).task;
}

export async function rejectTask(id: string, signal?: AbortSignal): Promise<TaskRecord> {
  const { data, error, response } = await apiClient.POST("/api/tasks/{id}/reject", {
    params: { path: { id } },
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new TasksApiError(`Task not found: ${id}`, 404);
    }

    throw new TasksApiError(
      defaultApiErrorMessage(`Failed to reject task "${id}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to reject task "${id}"`).task;
}

export async function createChildTask(
  parentId: string,
  body: CreateChildTaskRequest,
  signal?: AbortSignal
): Promise<TaskRecord> {
  const { data, error, response } = await apiClient.POST("/api/tasks/{id}/children", {
    params: { path: { id: parentId } },
    body,
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new TasksApiError(`Task not found: ${parentId}`, 404);
    }

    throw new TasksApiError(
      defaultApiErrorMessage(`Failed to create child task for "${parentId}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to create child task for "${parentId}"`).task;
}

export async function addTaskDependency(
  id: string,
  body: AddTaskDependencyRequest,
  signal?: AbortSignal
): Promise<TaskDetailView> {
  const { data, error, response } = await apiClient.POST("/api/tasks/{id}/dependencies", {
    params: { path: { id } },
    body,
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new TasksApiError(`Task not found: ${id}`, 404);
    }

    throw new TasksApiError(
      defaultApiErrorMessage(`Failed to add dependency to task "${id}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to add dependency to task "${id}"`).task;
}

export async function removeTaskDependency(
  id: string,
  dependsOnId: string,
  signal?: AbortSignal
): Promise<TaskDetailView> {
  const { data, error, response } = await apiClient.DELETE(
    "/api/tasks/{id}/dependencies/{depends_on_id}",
    {
      params: { path: { id, depends_on_id: dependsOnId } },
      signal,
    }
  );

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new TasksApiError(`Task not found: ${id}`, 404);
    }

    throw new TasksApiError(
      defaultApiErrorMessage(`Failed to remove dependency from task "${id}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to remove dependency from task "${id}"`).task;
}

export async function listTaskRuns(
  id: string,
  filters: TaskRunsFilter = {},
  signal?: AbortSignal
): Promise<TaskRun[]> {
  const { data, error, response } = await apiClient.GET("/api/tasks/{id}/runs", {
    params: {
      path: { id },
      query: {
        status: filters.status,
        session_id: normalizeOptionalText(filters.session_id),
        limit: filters.limit,
      },
    },
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new TasksApiError(`Task not found: ${id}`, 404);
    }

    throw new TasksApiError(
      defaultApiErrorMessage(`Failed to fetch runs for task "${id}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to fetch runs for task "${id}"`).runs;
}

export async function enqueueTaskRun(
  id: string,
  body: EnqueueTaskRunRequest = {},
  signal?: AbortSignal
): Promise<TaskRun> {
  const { data, error, response } = await apiClient.POST("/api/tasks/{id}/runs", {
    params: { path: { id } },
    body,
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new TasksApiError(`Task not found: ${id}`, 404);
    }

    throw new TasksApiError(
      defaultApiErrorMessage(`Failed to enqueue run for task "${id}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to enqueue run for task "${id}"`).run;
}

export async function getTaskTimeline(
  id: string,
  filters: TaskTimelineFilter = {},
  signal?: AbortSignal
): Promise<TaskTimelineItem[]> {
  const { data, error, response } = await apiClient.GET("/api/tasks/{id}/timeline", {
    params: {
      path: { id },
      query: {
        after_sequence: filters.after_sequence,
        limit: filters.limit,
      },
    },
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new TasksApiError(`Task not found: ${id}`, 404);
    }

    throw new TasksApiError(
      defaultApiErrorMessage(`Failed to fetch timeline for task "${id}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to fetch timeline for task "${id}"`).timeline;
}

export async function getTaskTree(id: string, signal?: AbortSignal): Promise<TaskTreeView> {
  const { data, error, response } = await apiClient.GET("/api/tasks/{id}/tree", {
    params: { path: { id } },
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new TasksApiError(`Task not found: ${id}`, 404);
    }

    throw new TasksApiError(
      defaultApiErrorMessage(`Failed to fetch tree for task "${id}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to fetch tree for task "${id}"`).tree;
}

export async function markTaskRead(id: string, signal?: AbortSignal): Promise<TaskTriageState> {
  const { data, error, response } = await apiClient.POST("/api/tasks/{id}/triage/read", {
    params: { path: { id } },
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new TasksApiError(`Task not found: ${id}`, 404);
    }

    throw new TasksApiError(
      defaultApiErrorMessage(`Failed to mark task "${id}" as read`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to mark task "${id}" as read`).triage;
}

export async function archiveTask(id: string, signal?: AbortSignal): Promise<TaskTriageState> {
  const { data, error, response } = await apiClient.POST("/api/tasks/{id}/triage/archive", {
    params: { path: { id } },
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new TasksApiError(`Task not found: ${id}`, 404);
    }

    throw new TasksApiError(
      defaultApiErrorMessage(`Failed to archive task "${id}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to archive task "${id}"`).triage;
}

export async function dismissTask(id: string, signal?: AbortSignal): Promise<TaskTriageState> {
  const { data, error, response } = await apiClient.POST("/api/tasks/{id}/triage/dismiss", {
    params: { path: { id } },
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new TasksApiError(`Task not found: ${id}`, 404);
    }

    throw new TasksApiError(
      defaultApiErrorMessage(`Failed to dismiss task "${id}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to dismiss task "${id}"`).triage;
}

export async function getTaskRun(id: string, signal?: AbortSignal): Promise<TaskRunDetailView> {
  const { data, error, response } = await apiClient.GET("/api/task-runs/{id}", {
    params: { path: { id } },
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new TasksApiError(`Task run not found: ${id}`, 404);
    }

    throw new TasksApiError(
      defaultApiErrorMessage(`Failed to fetch task run "${id}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to fetch task run "${id}"`).run;
}

export async function inspectRun(id: string, signal?: AbortSignal): Promise<TaskRunInspectView> {
  const { data, error, response } = await apiClient.GET("/api/runs/{id}/inspect", {
    params: { path: { id } },
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new TasksApiError(`Task run not found: ${id}`, 404);
    }

    throw new TasksApiError(
      defaultApiErrorMessage(`Failed to inspect task run "${id}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to inspect task run "${id}"`).inspect;
}

export async function attachTaskRunSession(
  id: string,
  body: AttachTaskRunSessionRequest,
  signal?: AbortSignal
): Promise<TaskRun> {
  const { data, error, response } = await apiClient.POST("/api/task-runs/{id}/attach-session", {
    params: { path: { id } },
    body,
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new TasksApiError(`Task run not found: ${id}`, 404);
    }

    throw new TasksApiError(
      defaultApiErrorMessage(`Failed to attach session to task run "${id}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to attach session to task run "${id}"`).run;
}

export async function cancelTaskRun(
  id: string,
  body: CancelTaskRunRequest = {},
  signal?: AbortSignal
): Promise<TaskRun> {
  const { data, error, response } = await apiClient.POST("/api/task-runs/{id}/cancel", {
    params: { path: { id } },
    body,
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new TasksApiError(`Task run not found: ${id}`, 404);
    }

    throw new TasksApiError(
      defaultApiErrorMessage(`Failed to cancel task run "${id}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to cancel task run "${id}"`).run;
}

export async function claimTaskRun(
  id: string,
  body: ClaimTaskRunRequest = {},
  signal?: AbortSignal
): Promise<TaskRun> {
  const { data, error, response } = await apiClient.POST("/api/task-runs/{id}/claim", {
    params: { path: { id } },
    body,
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new TasksApiError(`Task run not found: ${id}`, 404);
    }

    throw new TasksApiError(
      defaultApiErrorMessage(`Failed to claim task run "${id}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to claim task run "${id}"`).run;
}

export async function startTaskRun(
  id: string,
  body: StartTaskRunRequest = {},
  signal?: AbortSignal
): Promise<TaskRun> {
  const { data, error, response } = await apiClient.POST("/api/task-runs/{id}/start", {
    params: { path: { id } },
    body,
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new TasksApiError(`Task run not found: ${id}`, 404);
    }

    throw new TasksApiError(
      defaultApiErrorMessage(`Failed to start task run "${id}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to start task run "${id}"`).run;
}

export async function completeTaskRun(
  id: string,
  body: CompleteTaskRunRequest = {},
  signal?: AbortSignal
): Promise<TaskRun> {
  const { data, error, response } = await apiClient.POST("/api/task-runs/{id}/complete", {
    params: { path: { id } },
    body,
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new TasksApiError(`Task run not found: ${id}`, 404);
    }

    throw new TasksApiError(
      defaultApiErrorMessage(`Failed to complete task run "${id}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to complete task run "${id}"`).run;
}

export async function failTaskRun(
  id: string,
  body: FailTaskRunRequest,
  signal?: AbortSignal
): Promise<TaskRun> {
  const { data, error, response } = await apiClient.POST("/api/task-runs/{id}/fail", {
    params: { path: { id } },
    body,
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new TasksApiError(`Task run not found: ${id}`, 404);
    }

    throw new TasksApiError(
      defaultApiErrorMessage(`Failed to fail task run "${id}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to fail task run "${id}"`).run;
}

export async function forceReleaseTaskRun(
  id: string,
  body: ForceReleaseTaskRunRequest = {},
  signal?: AbortSignal
): Promise<TaskRun> {
  const { data, error, response } = await apiClient.POST("/api/runs/{id}/release", {
    params: { path: { id } },
    body,
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new TasksApiError(`Task run not found: ${id}`, 404);
    }

    throw new TasksApiError(
      defaultApiErrorMessage(`Failed to release task run "${id}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to release task run "${id}"`).run;
}

export async function forceFailTaskRun(
  id: string,
  body: ForceFailTaskRunRequest,
  signal?: AbortSignal
): Promise<TaskRun> {
  const { data, error, response } = await apiClient.POST("/api/runs/{id}/fail", {
    params: { path: { id } },
    body,
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new TasksApiError(`Task run not found: ${id}`, 404);
    }

    throw new TasksApiError(
      defaultApiErrorMessage(`Failed to fail task run "${id}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to fail task run "${id}"`).run;
}

export async function retryTaskRun(
  id: string,
  body: RetryTaskRunRequest = {},
  signal?: AbortSignal
): Promise<RetryTaskRunResult> {
  const { data, error, response } = await apiClient.POST("/api/runs/{id}/retry", {
    params: { path: { id } },
    body,
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new TasksApiError(`Task run not found: ${id}`, 404);
    }

    throw new TasksApiError(
      defaultApiErrorMessage(`Failed to retry task run "${id}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to retry task run "${id}"`);
}

export async function getTaskDashboard(
  filters: TaskDashboardFilter = {},
  signal?: AbortSignal
): Promise<TaskDashboardView> {
  const { data, error, response } = await apiClient.GET("/api/observe/tasks/dashboard", {
    params: { query: normalizeDashboardFilter(filters) },
    signal,
  });

  if (apiRequestFailed(response, error)) {
    throw new TasksApiError(
      defaultApiErrorMessage("Failed to fetch task dashboard", response, error),
      response.status
    );
  }

  return requireResponseData(data, response, "Failed to fetch task dashboard").dashboard;
}

export async function getTaskInbox(
  filters: TaskInboxFilter = {},
  signal?: AbortSignal
): Promise<TaskInboxView> {
  const { data, error, response } = await apiClient.GET("/api/observe/tasks/inbox", {
    params: { query: normalizeInboxFilter(filters) },
    signal,
  });

  if (apiRequestFailed(response, error)) {
    throw new TasksApiError(
      defaultApiErrorMessage("Failed to fetch task inbox", response, error),
      response.status
    );
  }

  return requireResponseData(data, response, "Failed to fetch task inbox").inbox;
}

function normalizeBridgeNotificationFilter(
  filters: TaskBridgeNotificationSubscriptionsFilter = {}
): TaskBridgeNotificationSubscriptionsFilter {
  return {
    bridge_instance_id: normalizeOptionalText(filters.bridge_instance_id),
    scope: filters.scope,
    workspace_id: normalizeOptionalText(filters.workspace_id),
    limit: filters.limit,
  };
}

export async function getTaskExecutionProfile(
  id: string,
  signal?: AbortSignal
): Promise<TaskExecutionProfile> {
  const { data, error, response } = await apiClient.GET("/api/tasks/{id}/execution-profile", {
    params: { path: { id } },
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new TasksApiError(`Task not found: ${id}`, 404);
    }

    throw new TasksApiError(
      defaultApiErrorMessage(`Failed to fetch execution profile for task "${id}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to fetch execution profile for task "${id}"`)
    .profile;
}

export async function setTaskExecutionProfile(
  id: string,
  body: TaskExecutionProfileSetRequest,
  signal?: AbortSignal
): Promise<TaskExecutionProfile> {
  const { data, error, response } = await apiClient.PUT("/api/tasks/{id}/execution-profile", {
    params: { path: { id } },
    body,
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new TasksApiError(`Task not found: ${id}`, 404);
    }
    if (response.status === 409) {
      throw new TasksApiError(
        defaultApiErrorMessage(`Execution profile conflict for task "${id}"`, response, error),
        409
      );
    }

    throw new TasksApiError(
      defaultApiErrorMessage(`Failed to set execution profile for task "${id}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to set execution profile for task "${id}"`)
    .profile;
}

export async function deleteTaskExecutionProfile(id: string, signal?: AbortSignal): Promise<void> {
  const { error, response } = await apiClient.DELETE("/api/tasks/{id}/execution-profile", {
    params: { path: { id } },
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new TasksApiError(`Task not found: ${id}`, 404);
    }

    throw new TasksApiError(
      defaultApiErrorMessage(
        `Failed to delete execution profile for task "${id}"`,
        response,
        error
      ),
      response.status
    );
  }
}

export async function listTaskRunReviews(
  runId: string,
  filters: TaskRunReviewsFilter = {},
  signal?: AbortSignal
): Promise<TaskRunReview[]> {
  const { data, error, response } = await apiClient.GET("/api/task-runs/{id}/reviews", {
    params: {
      path: { id: runId },
      query: {
        status: filters.status,
        reviewer_session_id: normalizeOptionalText(filters.reviewer_session_id),
        limit: filters.limit,
      },
    },
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new TasksApiError(`Task run not found: ${runId}`, 404);
    }

    throw new TasksApiError(
      defaultApiErrorMessage(`Failed to fetch reviews for task run "${runId}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to fetch reviews for task run "${runId}"`)
    .reviews;
}

export async function listTaskReviews(
  taskId: string,
  filters: TaskReviewsFilter = {},
  signal?: AbortSignal
): Promise<TaskRunReview[]> {
  const { data, error, response } = await apiClient.GET("/api/tasks/{id}/reviews", {
    params: {
      path: { id: taskId },
      query: {
        status: filters.status,
        reviewer_session_id: normalizeOptionalText(filters.reviewer_session_id),
        limit: filters.limit,
      },
    },
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new TasksApiError(`Task not found: ${taskId}`, 404);
    }

    throw new TasksApiError(
      defaultApiErrorMessage(`Failed to fetch reviews for task "${taskId}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to fetch reviews for task "${taskId}"`)
    .reviews;
}

export async function requestTaskRunReview(
  runId: string,
  body: TaskRunReviewRequest,
  signal?: AbortSignal
): Promise<TaskRunReviewRequestResult> {
  const { data, error, response } = await apiClient.POST("/api/task-runs/{id}/reviews", {
    params: { path: { id: runId } },
    body,
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new TasksApiError(`Task run not found: ${runId}`, 404);
    }
    if (response.status === 409) {
      throw new TasksApiError(
        defaultApiErrorMessage(`Review request conflict for task run "${runId}"`, response, error),
        409
      );
    }

    throw new TasksApiError(
      defaultApiErrorMessage(`Failed to request review for task run "${runId}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to request review for task run "${runId}"`);
}

export async function getTaskRunReview(
  reviewId: string,
  signal?: AbortSignal
): Promise<TaskRunReview> {
  const { data, error, response } = await apiClient.GET("/api/task-reviews/{id}", {
    params: { path: { id: reviewId } },
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new TasksApiError(`Task review not found: ${reviewId}`, 404);
    }

    throw new TasksApiError(
      defaultApiErrorMessage(`Failed to fetch task review "${reviewId}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to fetch task review "${reviewId}"`).review;
}

export async function submitTaskRunReviewVerdict(
  reviewId: string,
  body: TaskRunReviewVerdictRequest,
  signal?: AbortSignal
): Promise<TaskRunReviewVerdictResult> {
  const { data, error, response } = await apiClient.POST("/api/task-reviews/{id}/verdict", {
    params: { path: { id: reviewId } },
    body,
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new TasksApiError(`Task review not found: ${reviewId}`, 404);
    }
    if (response.status === 409) {
      throw new TasksApiError(
        defaultApiErrorMessage(`Review verdict conflict for review "${reviewId}"`, response, error),
        409
      );
    }

    throw new TasksApiError(
      defaultApiErrorMessage(`Failed to submit verdict for review "${reviewId}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to submit verdict for review "${reviewId}"`);
}

export async function getAgentContext(signal?: AbortSignal): Promise<AgentContextView> {
  const { data, error, response } = await apiClient.GET("/api/agent/context", {
    signal,
  });

  if (apiRequestFailed(response, error)) {
    throw new TasksApiError(
      defaultApiErrorMessage("Failed to fetch agent context", response, error),
      response.status
    );
  }

  return requireResponseData(data, response, "Failed to fetch agent context").context;
}

export async function getTaskContextBundle(
  signal?: AbortSignal
): Promise<TaskContextBundle | null> {
  const context = await getAgentContext(signal);
  return context.task.bundle ?? null;
}

export async function listTaskBridgeNotificationSubscriptions(
  taskId: string,
  filters: TaskBridgeNotificationSubscriptionsFilter = {},
  signal?: AbortSignal
): Promise<TaskBridgeNotificationSubscription[]> {
  const { data, error, response } = await apiClient.GET("/api/tasks/{id}/notifications/bridges", {
    params: {
      path: { id: taskId },
      query: normalizeBridgeNotificationFilter(filters),
    },
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new TasksApiError(`Task not found: ${taskId}`, 404);
    }

    throw new TasksApiError(
      defaultApiErrorMessage(
        `Failed to fetch bridge notification subscriptions for task "${taskId}"`,
        response,
        error
      ),
      response.status
    );
  }

  return requireResponseData(
    data,
    response,
    `Failed to fetch bridge notification subscriptions for task "${taskId}"`
  ).subscriptions;
}

export async function createTaskBridgeNotificationSubscription(
  taskId: string,
  body: TaskBridgeNotificationSubscriptionCreateRequest,
  signal?: AbortSignal
): Promise<TaskBridgeNotificationSubscription> {
  const { data, error, response } = await apiClient.POST("/api/tasks/{id}/notifications/bridges", {
    params: { path: { id: taskId } },
    body,
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new TasksApiError(`Task or bridge not found for task "${taskId}"`, 404);
    }

    throw new TasksApiError(
      defaultApiErrorMessage(
        `Failed to create bridge notification subscription for task "${taskId}"`,
        response,
        error
      ),
      response.status
    );
  }

  return requireResponseData(
    data,
    response,
    `Failed to create bridge notification subscription for task "${taskId}"`
  ).subscription;
}

export async function getTaskBridgeNotificationSubscription(
  taskId: string,
  subscriptionId: string,
  signal?: AbortSignal
): Promise<TaskBridgeNotificationSubscription> {
  const { data, error, response } = await apiClient.GET(
    "/api/tasks/{id}/notifications/bridges/{subscription_id}",
    {
      params: { path: { id: taskId, subscription_id: subscriptionId } },
      signal,
    }
  );

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new TasksApiError(`Bridge notification subscription not found: ${subscriptionId}`, 404);
    }

    throw new TasksApiError(
      defaultApiErrorMessage(
        `Failed to fetch bridge notification subscription "${subscriptionId}"`,
        response,
        error
      ),
      response.status
    );
  }

  return requireResponseData(
    data,
    response,
    `Failed to fetch bridge notification subscription "${subscriptionId}"`
  ).subscription;
}

export async function deleteTaskBridgeNotificationSubscription(
  taskId: string,
  subscriptionId: string,
  signal?: AbortSignal
): Promise<void> {
  const { error, response } = await apiClient.DELETE(
    "/api/tasks/{id}/notifications/bridges/{subscription_id}",
    {
      params: { path: { id: taskId, subscription_id: subscriptionId } },
      signal,
    }
  );

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new TasksApiError(`Bridge notification subscription not found: ${subscriptionId}`, 404);
    }

    throw new TasksApiError(
      defaultApiErrorMessage(
        `Failed to delete bridge notification subscription "${subscriptionId}"`,
        response,
        error
      ),
      response.status
    );
  }
}
