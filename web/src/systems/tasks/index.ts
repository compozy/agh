// Types
export type {
  AddTaskDependencyRequest,
  AttachTaskRunSessionRequest,
  CancelTaskRequest,
  CancelTaskRunRequest,
  ClaimTaskRunRequest,
  CompleteTaskRunRequest,
  CreateChildTaskRequest,
  CreateTaskRequest,
  EnqueueTaskRunRequest,
  FailTaskRunRequest,
  StartTaskRunRequest,
  TaskApprovalPolicy,
  TaskApprovalState,
  TaskChildSummary,
  TaskDashboardFilter,
  TaskDashboardView,
  TaskDetailView,
  TaskInboxFilter,
  TaskInboxGroup,
  TaskInboxItem,
  TaskInboxLane,
  TaskInboxView,
  TaskListFilter,
  TaskListItem,
  TaskOwnerKind,
  TaskPriority,
  TaskRecord,
  TaskRun,
  TaskRunDetailView,
  TaskRunStatus,
  TaskRunsFilter,
  TaskScope,
  TaskStatus,
  TaskStreamFilter,
  TaskStreamPayload,
  TaskSummary,
  TaskTimelineFilter,
  TaskTimelineItem,
  TaskTreeNode,
  TaskTreeView,
  TaskTriageState,
  TaskViewMode,
  UpdateTaskRequest,
} from "./types";

// Adapters
export {
  TasksApiError,
  addTaskDependency,
  approveTask,
  archiveTask,
  attachTaskRunSession,
  cancelTask,
  cancelTaskRun,
  claimTaskRun,
  completeTaskRun,
  createChildTask,
  createTask,
  dismissTask,
  enqueueTaskRun,
  failTaskRun,
  getTask,
  getTaskDashboard,
  getTaskInbox,
  getTaskRun,
  getTaskTimeline,
  getTaskTree,
  listTaskRuns,
  listTasks,
  markTaskRead,
  publishTask,
  rejectTask,
  removeTaskDependency,
  startTaskRun,
  updateTask,
} from "./adapters/tasks-api";

// Query infrastructure
export { tasksKeys } from "./lib/query-keys";
export {
  taskDashboardOptions,
  taskDetailOptions,
  taskInboxOptions,
  taskRunDetailOptions,
  taskRunsOptions,
  taskTimelineOptions,
  taskTreeOptions,
  tasksListOptions,
} from "./lib/query-options";

// Formatters and helpers
export type { TaskSemanticTone } from "./lib/task-formatters";
export {
  countTasksByStatus,
  formatAttemptLabel,
  formatRelativeTime,
  matchesTaskQuery,
  taskApprovalStateLabel,
  taskHasApprovalPending,
  taskInboxLaneLabel,
  taskIsBlocked,
  taskIsDraft,
  taskLaneTone,
  taskOwnerKindLabel,
  taskOwnerLabel,
  taskPriorityLabel,
  taskPriorityTone,
  taskRunStatusTone,
  taskStatusLabel,
  taskStatusTone,
} from "./lib/task-formatters";

// Templates and grouping
export {
  DEFAULT_TASK_TEMPLATE_ID,
  TASK_TEMPLATES,
  applyTemplateToCreatePayload,
  getTaskTemplate,
} from "./lib/task-templates";
export type {
  TaskTemplate,
  TaskTemplateBadge,
  TaskTemplateDefaults,
  TaskTemplateId,
  TaskTemplatePreview,
} from "./lib/task-templates";

export { getKanbanColumns, groupTasksForKanban, resolveKanbanColumnId } from "./lib/task-grouping";
export type { KanbanColumnGroup, TaskKanbanColumn, TaskKanbanColumnId } from "./lib/task-grouping";

// Read hooks
export { useTask, useTaskRuns, useTasks } from "./hooks/use-tasks";
export { useTaskRunDetail, useTaskTimeline, useTaskTree } from "./hooks/use-task-live";
export { useTaskDashboard } from "./hooks/use-task-dashboard";
export { useTaskInbox } from "./hooks/use-task-inbox";

// Mutation hooks
export {
  useAddTaskDependency,
  useApproveTask,
  useArchiveTask,
  useAttachTaskRunSession,
  useCancelTask,
  useCancelTaskRun,
  useClaimTaskRun,
  useCompleteTaskRun,
  useCreateChildTask,
  useCreateTask,
  useDismissTask,
  useEnqueueTaskRun,
  useFailTaskRun,
  useMarkTaskRead,
  usePublishTask,
  useRejectTask,
  useRemoveTaskDependency,
  useStartTaskRun,
  useUpdateTask,
} from "./hooks/use-task-actions";

// Components
export { TASKS_SHELL_TITLE, TasksPageShell } from "./components/tasks-page-shell";
export { TaskCard } from "./components/task-card";
export type { TaskCardProps } from "./components/task-card";
export { TasksListPanel } from "./components/tasks-list-panel";
export type { TasksListPanelProps } from "./components/tasks-list-panel";
export { TasksDetailPreviewPanel } from "./components/tasks-detail-preview-panel";
export type { TasksDetailPreviewPanelProps } from "./components/tasks-detail-preview-panel";
export { TasksKanbanBoard } from "./components/tasks-kanban-board";
export type { TasksKanbanBoardProps } from "./components/tasks-kanban-board";
export { TasksEmptyState } from "./components/tasks-empty-state";
export type { TasksEmptyStateProps } from "./components/tasks-empty-state";
export { TasksCreateModal } from "./components/tasks-create-modal";
export type { TasksCreateModalProps } from "./components/tasks-create-modal";
