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
export type { TaskSemanticTone, TaskStatusSignal } from "./lib/task-formatters";
export {
  countTasksByStatus,
  formatAttemptLabel,
  formatDurationMs,
  formatPercent,
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
  taskShortId,
  taskStatusLabel,
  taskStatusSignal,
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
export { TasksListRow } from "./components/tasks-list-row";
export type { TasksListRowProps } from "./components/tasks-list-row";
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

// Task detail + run detail components
export { TasksDetailHeader } from "./components/tasks-detail-header";
export type { TasksDetailHeaderProps } from "./components/tasks-detail-header";
export { TasksDetailTabs } from "./components/tasks-detail-tabs";
export type { TasksDetailTabItem, TasksDetailTabsProps } from "./components/tasks-detail-tabs";
export { TasksDetailOverviewPanel } from "./components/tasks-detail-overview-panel";
export type { TasksDetailOverviewPanelProps } from "./components/tasks-detail-overview-panel";
export { TasksTimelinePanel } from "./components/tasks-timeline-panel";
export type { TasksTimelinePanelProps } from "./components/tasks-timeline-panel";
export { TasksDetailRunsPanel } from "./components/tasks-detail-runs-panel";
export type { TasksDetailRunsPanelProps } from "./components/tasks-detail-runs-panel";
export { TasksDetailChildrenPanel } from "./components/tasks-detail-children-panel";
export type { TasksDetailChildrenPanelProps } from "./components/tasks-detail-children-panel";
export { TasksDetailDependenciesPanel } from "./components/tasks-detail-dependencies-panel";
export type { TasksDetailDependenciesPanelProps } from "./components/tasks-detail-dependencies-panel";

export { TaskRunDetailHeader } from "./components/task-run-detail-header";
export type { TaskRunDetailHeaderProps } from "./components/task-run-detail-header";
export {
  TaskRunActivityPanel,
  TaskRunIdentityPanel,
  TaskRunProgressPanel,
} from "./components/task-run-detail-panels";
export type {
  TaskRunActivityPanelProps,
  TaskRunIdentityPanelProps,
  TaskRunProgressPanelProps,
} from "./components/task-run-detail-panels";
export { TaskRunDetailSessionLink } from "./components/task-run-detail-session-link";
export type { TaskRunDetailSessionLinkProps } from "./components/task-run-detail-session-link";

export { TasksMultiAgentPanel } from "./components/tasks-multi-agent-panel";
export type { TasksMultiAgentPanelProps } from "./components/tasks-multi-agent-panel";

// Dashboard + Inbox aggregate components
export { TasksDashboardCards } from "./components/tasks-dashboard-cards";
export type { TasksDashboardCardsProps } from "./components/tasks-dashboard-cards";
export { TasksDashboardStatusBreakdown } from "./components/tasks-dashboard-status-breakdown";
export type { TasksDashboardStatusBreakdownProps } from "./components/tasks-dashboard-status-breakdown";
export { TasksDashboardQueueHealth } from "./components/tasks-dashboard-queue-health";
export type { TasksDashboardQueueHealthProps } from "./components/tasks-dashboard-queue-health";
export { TasksDashboardActiveRuns } from "./components/tasks-dashboard-active-runs";
export type { TasksDashboardActiveRunsProps } from "./components/tasks-dashboard-active-runs";
export { TasksDashboardView } from "./components/tasks-dashboard-view";
export type { TasksDashboardViewProps } from "./components/tasks-dashboard-view";

export { TasksInboxLaneTabs } from "./components/tasks-inbox-lane-tabs";
export type { TasksInboxLaneTabsProps } from "./components/tasks-inbox-lane-tabs";
export { TasksInboxItem } from "./components/tasks-inbox-item";
export type { TasksInboxItemProps } from "./components/tasks-inbox-item";
export { TasksInboxView } from "./components/tasks-inbox-view";
export type { TasksInboxViewProps } from "./components/tasks-inbox-view";
