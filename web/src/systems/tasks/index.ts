// Types
export type {
  AddTaskDependencyRequest,
  AgentContextView,
  AgentTaskContextSection,
  AttachTaskRunSessionRequest,
  CancelTaskRequest,
  CancelTaskRunRequest,
  CompleteTaskRunRequest,
  CreateChildTaskRequest,
  CreateTaskRequest,
  EnqueueTaskRunRequest,
  FailTaskRunRequest,
  FanOutTaskRunsRequest,
  FanOutTaskRunsResponse,
  ForceFailTaskRunRequest,
  ForceReleaseTaskRunRequest,
  PauseTaskRequest,
  RecoverTaskRequest,
  RecoverTaskRunRequest,
  RecoverTaskRunResult,
  RetryTaskRunRequest,
  RetryTaskRunResult,
  ResumeTaskRequest,
  StartTaskRunRequest,
  TaskApprovalPolicy,
  TaskBlockedReason,
  TaskBlockedReasonSource,
  TaskBlockKind,
  TaskApprovalState,
  TaskBridgeNotificationCursor,
  TaskBridgeNotificationDeliveryMode,
  TaskBridgeNotificationSubscription,
  TaskBridgeNotificationSubscriptionCreateRequest,
  TaskBridgeNotificationSubscriptionScope,
  TaskBridgeNotificationSubscriptionsFilter,
  TaskChildSummary,
  TaskContextBundle,
  TaskContextCurrentRun,
  TaskContextPriorAttempt,
  TaskContextRecentEvent,
  TaskContextReviewContinuation,
  TaskContextReviewHistoryEntry,
  TaskDashboardFilter,
  TaskDashboardView,
  TaskDetailView,
  TaskExecutionProfile,
  TaskExecutionProfileCoordinator,
  TaskExecutionProfileCoordinatorMode,
  TaskExecutionProfileParticipants,
  TaskExecutionProfileReviewSelectors,
  TaskExecutionProfileSandbox,
  TaskExecutionProfileSandboxMode,
  TaskExecutionProfileSetRequest,
  TaskExecutionProfileWorker,
  TaskExecutionProfileWorkerMode,
  TaskInboxFilter,
  TaskInboxGroup,
  TaskInboxItem,
  TaskInboxLane,
  TaskInboxView,
  TaskInspectView,
  TaskListFilter,
  TaskListFacets,
  TaskListItem,
  TaskListPage,
  TaskListStableFilter,
  TaskListSortKey,
  TaskOwnerKind,
  TaskPriority,
  TaskRecord,
  TaskReviewsFilter,
  TaskRun,
  TaskRunDetailView,
  TaskRunInspectView,
  TaskRunReview,
  TaskRunReviewContinuationRun,
  TaskRunReviewOutcome,
  TaskRunReviewPolicy,
  TaskRunReviewRequest,
  TaskRunReviewRequestResult,
  TaskRunReviewStatus,
  TaskRunReviewVerdict,
  TaskRunReviewVerdictRequest,
  TaskRunReviewVerdictResult,
  TaskRunReviewsFilter,
  TaskRunStatus,
  TaskRunsFilter,
  TaskScope,
  TaskStatus,
  TaskStreamFilter,
  TaskStreamPayload,
  TaskStreamTimelineEvent,
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
  buildTaskStreamUrl,
  cancelTask,
  cancelTaskRun,
  clearTaskBlock,
  completeTaskRun,
  createChildTask,
  createTask,
  createTaskBridgeNotificationSubscription,
  deleteTask,
  deleteTaskBridgeNotificationSubscription,
  deleteTaskExecutionProfile,
  dismissTask,
  enqueueTaskRun,
  fanOutTaskRuns,
  failTaskRun,
  forceFailTaskRun,
  forceReleaseTaskRun,
  getAgentContext,
  getTask,
  getTaskBridgeNotificationSubscription,
  getTaskContextBundle,
  getTaskDashboard,
  getTaskExecutionProfile,
  getTaskInbox,
  getTaskRun,
  getTaskRunReview,
  getTaskTimeline,
  getTaskTree,
  inspectRun,
  inspectTask,
  listTaskBridgeNotificationSubscriptions,
  listTaskReviews,
  listTaskRunReviews,
  listTaskRuns,
  listTasks,
  markTaskRead,
  pauseTask,
  publishTask,
  recoverTask,
  recoverTaskRun,
  rejectTask,
  removeTaskDependency,
  requestTaskRunReview,
  retryTaskRun,
  resumeTask,
  setTaskExecutionProfile,
  startTaskRun,
  submitTaskRunReviewVerdict,
  updateTask,
} from "./adapters/tasks-api";

// Query infrastructure
export { tasksKeys } from "./lib/query-keys";
export {
  agentContextOptions,
  taskBridgeNotificationSubscriptionOptions,
  taskBridgeNotificationSubscriptionsOptions,
  taskContextBundleOptions,
  taskDashboardOptions,
  taskDetailOptions,
  taskExecutionProfileOptions,
  taskInboxBadgeOptions,
  taskInboxOptions,
  taskReviewsOptions,
  taskRunDetailOptions,
  taskRunReviewDetailOptions,
  taskRunReviewsOptions,
  taskRunsOptions,
  taskTimelineOptions,
  taskTreeOptions,
  tasksListOptions,
} from "./lib/query-options";

// Formatters and helpers
export type { BlockedReasonChip, TaskStatusSignal } from "./lib/task-formatters";
export {
  computeElapsed,
  countTasksByStatus,
  formatAttemptLabel,
  formatDurationMs,
  formatPercent,
  formatRelativeTime,
  matchesTaskQuery,
  ownerAvatarKindFor,
  projectBlockedReasonChips,
  taskApprovalStateLabel,
  taskBlockedSourceLabel,
  taskBlockKindLabel,
  taskCanRecover,
  taskHasApprovalPending,
  taskInboxLaneLabel,
  taskIsBlocked,
  taskIsDraft,
  taskWakeIndicatorApplies,
  taskLaneTone,
  taskOwnerKindLabel,
  taskOwnerLabel,
  taskPriorityLabel,
  taskPriorityTone,
  taskRunStatusLabel,
  taskRunStatusTone,
  taskShortId,
  taskStatusLabel,
  taskStatusSignal,
  taskStatusTone,
  toRunCardStatus,
} from "./lib/task-formatters";
export { taskRunCanRecover } from "./lib/task-run-recovery";
export {
  parseTasksSurfaceMode,
  validateTaskCreateSearch,
  validateTasksSearch,
} from "./lib/task-location-search";
export type { TaskCreateSearch, TasksRouteSearch } from "./lib/task-location-search";
export {
  resolveTaskDetailSearch,
  TASK_DETAIL_TABS,
  validateTaskDetailSearch,
} from "./lib/task-detail-search";
export type {
  ResolvedTaskDetailSearch,
  TaskDetailSearch,
  TaskDetailTab,
} from "./lib/task-detail-search";
export {
  humanizeTaskEvent,
  matchesActivityFilter,
  TASK_ACTIVITY_FILTERS,
} from "./lib/task-activity-copy";
export type {
  TaskActivityCategory,
  TaskActivityFilter,
  TaskActivityView,
} from "./lib/task-activity-copy";
export { resolveTaskCommandState, taskAttemptsUsed } from "./lib/task-command-state";
export type { TaskCommandState, TaskPrimaryCommand } from "./lib/task-command-state";
export { projectTaskExceptionPills } from "./lib/task-detail-pills";
export type { TaskExceptionPill } from "./lib/task-detail-pills";
export { DEFAULT_TASK_LIST_LIMIT, defaultTaskCatalogFilter } from "./lib/task-catalog-filter";
export { taskScopeForActiveWorkspace } from "./lib/workspace-scope";

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
  TaskTemplateBadgeTone,
  TaskTemplateDefaults,
  TaskTemplateId,
  TaskTemplatePreview,
} from "./lib/task-templates";

export {
  getKanbanColumns,
  getTaskListGroups,
  groupTasksForKanban,
  groupTasksForList,
  resolveKanbanColumnId,
  resolveTaskListGroupId,
} from "./lib/task-grouping";
export type {
  KanbanColumnGroup,
  TaskKanbanColumn,
  TaskKanbanColumnId,
  TaskListGroupBucket,
  TaskListGroupDefinition,
  TaskListGroupId,
} from "./lib/task-grouping";

export {
  INBOX_GROUPS,
  INBOX_UI_LANES,
  backendLaneToUiLane,
  inboxGroupDotProps,
  resolveInboxGroupId,
} from "./lib/inbox-grouping";
export type {
  InboxGroupDefinition,
  InboxGroupId,
  InboxLaneDefinition,
  InboxLaneFilterId,
  InboxUiLane,
} from "./lib/inbox-grouping";

// Read hooks
export { useTask, useTaskRuns, useTasks } from "./hooks/use-tasks";
export {
  useTaskInspect,
  useTaskRunDetail,
  useTaskRunInspect,
  useTaskTimeline,
  useTaskTree,
} from "./hooks/use-task-live";
export { useTaskDashboard } from "./hooks/use-task-dashboard";
export { useTaskInbox, useTaskInboxBadge } from "./hooks/use-task-inbox";
export { useTaskExecutionProfile } from "./hooks/use-task-profile";
export { useTaskReviews, useTaskRunReview, useTaskRunReviews } from "./hooks/use-task-reviews";
export { useAgentContext, useTaskContextBundle } from "./hooks/use-task-context-bundle";
export {
  useTaskBridgeNotificationSubscription,
  useTaskBridgeNotificationSubscriptions,
} from "./hooks/use-task-notifications";
export { useTaskStream } from "./hooks/use-task-stream";
export type {
  TaskStreamEventSource,
  TaskStreamEventSourceFactory,
  UseTaskStreamOptions,
} from "./hooks/use-task-stream";
export { useTasksPage } from "./hooks/use-tasks-page";
export type { InboxLaneFilter, UseTasksPageOptions } from "./hooks/use-tasks-page";
export { useTaskCreateState } from "./hooks/use-task-create-state";
export { useTaskEditState } from "./hooks/use-task-edit-state";
export { useTaskDetailPage } from "./hooks/use-task-detail-page";
export type { UseTaskDetailPageOptions } from "./hooks/use-task-detail-page";
export { useTaskOperatorLayer } from "./hooks/use-task-operator-layer";
export type { UseTaskOperatorLayerOptions } from "./hooks/use-task-operator-layer";
export { useTaskRunPage } from "./hooks/use-task-run-page";
export type { UseTaskRunPageOptions } from "./hooks/use-task-run-page";
export { useLiveElapsed } from "./hooks/use-live-elapsed";
export { useTaskPauseDialog } from "./hooks/use-task-pause-dialog";
export { useForceFailDialog } from "./hooks/use-force-fail-dialog";
export { useTaskFanOutDialog } from "./hooks/use-task-fan-out-dialog";

// Mutation hooks
export {
  useAddTaskDependency,
  useApproveTask,
  useArchiveTask,
  useAttachTaskRunSession,
  useCancelTask,
  useCancelTaskRun,
  useClearTaskBlock,
  useCompleteTaskRun,
  useCreateChildTask,
  useCreateTask,
  useDeleteTask,
  useDismissTask,
  useEnqueueTaskRun,
  useFanOutTaskRuns,
  useFailTaskRun,
  useForceFailTaskRun,
  useForceReleaseTaskRun,
  useMarkTaskRead,
  usePauseTask,
  usePublishTask,
  useRecoverTask,
  useRejectTask,
  useRemoveTaskDependency,
  useRetryTaskRun,
  useResumeTask,
  useStartTaskRun,
  useUpdateTask,
} from "./hooks/use-task-actions";
export { useRecoverTaskRun } from "./hooks/use-task-run-recovery";
export {
  useDeleteTaskExecutionProfile,
  useSetTaskExecutionProfile,
} from "./hooks/use-task-profile";
export { useRequestTaskRunReview, useSubmitTaskRunReviewVerdict } from "./hooks/use-task-reviews";
export {
  useCreateTaskBridgeNotificationSubscription,
  useDeleteTaskBridgeNotificationSubscription,
} from "./hooks/use-task-notifications";

// Components
export { TaskCard } from "./components/task-card";
export type { TaskCardProps } from "./components/task-card";
export { TasksListRow } from "./components/tasks-list-row";
export type { TasksListRowProps } from "./components/tasks-list-row";
export { TasksListSurface } from "./components/tasks-list-surface";
export type { TasksListSurfaceProps } from "./components/tasks-list-surface";
export { TasksListToolbar } from "./components/tasks-list-toolbar";
export type { TasksListToolbarProps } from "./components/tasks-list-toolbar";
export { TasksListFilters } from "./components/tasks-list-filters";
export type { TasksListFiltersProps } from "./components/tasks-list-filters";
export { TasksListSort } from "./components/tasks-list-sort";
export type { TasksListSortProps } from "./components/tasks-list-sort";
export {
  applyTaskFilterChips,
  buildTaskFilterFields,
  taskOwnerFilterValue,
  taskFiltersToChips,
} from "./lib/tasks-list-filters";
export type {
  TaskFilterFieldKey,
  TaskFilterHandlers,
  TaskFilterOwnerOption,
  TaskFilterState,
} from "./lib/tasks-list-filters";
export {
  applyInboxFilterChips,
  buildInboxFilterFields,
  inboxFiltersToChips,
} from "./lib/inbox-filters";
export type {
  InboxFilterFieldKey,
  InboxFilterHandlers,
  InboxFilterState,
  InboxLaneCount,
} from "./lib/inbox-filters";
export { TaskGroup } from "./components/task-group";
export type { TaskGroupProps } from "./components/task-group";
export { TasksKanbanBoard } from "./components/tasks-kanban-board";
export type { TasksKanbanBoardProps } from "./components/tasks-kanban-board";
export { TasksEmptyState } from "./components/tasks-empty-state";
export type { TasksEmptyStateProps } from "./components/tasks-empty-state";
export { TaskEditorModal } from "./components/task-editor-modal";
export type { TaskEditorModalMode, TaskEditorModalProps } from "./components/task-editor-modal";
export { TaskEditorSurface } from "./components/task-editor-surface";
export type {
  TaskEditorSurfaceMode,
  TaskEditorSurfaceProps,
} from "./components/task-editor-surface";

// Task detail page
export { TaskPageActions, TaskPageOverflow, TaskPageStatus } from "./components/task-page-head";
export type {
  TaskPageActionHandlers,
  TaskPageActionsProps,
  TaskPageOverflowProps,
} from "./components/task-page-head";
export { TasksDetailSubhead } from "./components/tasks-detail-subhead";
export { TaskStateBand } from "./components/task-state-band";
export type { TaskStateBandProps } from "./components/task-state-band";
export { TaskNowStrip } from "./components/task-now-strip";
export type { TaskNowStripHandlers, TaskNowStripProps } from "./components/task-now-strip";
export { TaskLinkedRow } from "./components/task-linked-row";
export type { TaskLinkedRowProps, TaskLinkedRowState } from "./components/task-linked-row";
export { TaskSubtasksSection } from "./components/task-subtasks-section";
export { TaskDependenciesSection } from "./components/task-dependencies-section";
export { TaskActivityItem } from "./components/task-activity-item";
export { TaskActivityPanel } from "./components/task-activity-panel";
export type { TaskActivityPanelProps } from "./components/task-activity-panel";
export { TASK_RESULT_ANCHOR_ID, TaskOverviewPanel } from "./components/task-overview-panel";
export type { TaskOverviewPanelProps } from "./components/task-overview-panel";
export { TaskRunsPanel } from "./components/task-runs-panel";
export type { TaskRunsPanelProps } from "./components/task-runs-panel";
export { TaskPropertiesRail } from "./components/task-properties-rail";
export type { TaskPropertiesRailProps } from "./components/task-properties-rail";
export { TaskAutoEnqueueSwitch, TaskPriorityEditor } from "./components/task-rail-editors";
export { TaskInspectDrawer } from "./components/task-inspect-drawer";
export type {
  TaskInspectDrawerProps,
  TaskInspectDrawerTab,
} from "./components/task-inspect-drawer";
export { TaskRawPane } from "./components/task-raw-pane";
export { TaskBridgeSubscriptionsPane } from "./components/task-bridge-subscriptions-pane";
export type { TaskBridgeSubscriptionsPaneProps } from "./components/task-bridge-subscriptions-pane";
export { TaskSetupSheet } from "./components/task-setup-sheet";
export type { TaskSetupSheetProps } from "./components/task-setup-sheet";
export { TaskFanOutDialog } from "./components/task-fan-out-dialog";
export type { TaskFanOutDialogProps } from "./components/task-fan-out-dialog";
export { TaskPauseDialog } from "./components/task-pause-dialog";
export type { TaskPauseDialogProps } from "./components/task-pause-dialog";
export { TaskDeleteAction } from "./components/task-delete-action";

// Run detail page
export {
  TaskRunForceFailDialog,
  TaskRunPageActions,
  TaskRunPageOverflow,
  TaskRunPageStatus,
} from "./components/task-run-page-head";
export type {
  TaskRunForceFailDialogProps,
  TaskRunPageActionsProps,
  TaskRunPageOverflowProps,
} from "./components/task-run-page-head";
export { TaskRunRail } from "./components/task-run-rail";
export type { TaskRunRailProps } from "./components/task-run-rail";
export { TaskRunSubhead } from "./components/task-run-subhead";
export { TaskRunOutcome } from "./components/task-run-outcome";
export { TaskRunReviewCard } from "./components/task-run-review-card";
export { TaskRunInspectDrawer } from "./components/task-run-inspect-drawer";
export type { TaskRunInspectDrawerProps } from "./components/task-run-inspect-drawer";

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

export { TasksInboxItem } from "./components/tasks-inbox-item";
export type { TasksInboxItemProps } from "./components/tasks-inbox-item";
export { TasksInboxView } from "./components/tasks-inbox-view";
export type { TasksInboxViewProps } from "./components/tasks-inbox-view";
