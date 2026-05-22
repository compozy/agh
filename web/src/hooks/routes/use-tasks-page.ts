import { useCallback, useDeferredValue, useMemo, useState } from "react";
import { toast } from "sonner";

import {
  countTasksByStatus,
  matchesTaskQuery,
  taskIsDraft,
  useApproveTask,
  useArchiveTask,
  useCreateTask,
  useDeleteTask,
  useDismissTask,
  useEnqueueTaskRun,
  useMarkTaskRead,
  usePublishTask,
  useRejectTask,
  useTaskDashboard,
  useTaskInbox,
  useTasks,
} from "@/systems/tasks";
import {
  useDrainScheduler,
  usePauseScheduler,
  useResumeScheduler,
  useSchedulerBacklog,
  useSchedulerStatus,
} from "@/systems/scheduler";
import { DEFAULT_TASK_TEMPLATE_ID, getTaskTemplate } from "@/systems/tasks/lib/task-templates";
import type { TaskTemplateId } from "@/systems/tasks/lib/task-templates";
import {
  applyTemplateDefaultsToTaskEditorDraft,
  buildCreateTaskRequest,
  createTaskEditorDraft,
  EMPTY_TASK_EDITOR_DRAFT,
  type TaskEditorDraft,
} from "@/systems/tasks/lib/task-editor";
import { getKanbanColumns, groupTasksForKanban } from "@/systems/tasks/lib/task-grouping";
import type { KanbanColumnGroup } from "@/systems/tasks/lib/task-grouping";
import type { InboxLaneFilterId } from "@/systems/tasks/lib/inbox-grouping";
import type {
  TaskDashboardFilter,
  TaskInboxFilter,
  TaskListFilter,
  TaskListItem,
  TaskOwnerKind,
  TaskPriority,
  TaskScope,
  TaskStatus,
  TaskViewMode,
} from "@/systems/tasks";
import { useActiveWorkspace } from "@/systems/workspace";
import { workspaceFilterForActiveScope } from "./workspace-scope-filter";

type TaskScopeFilter = "all" | TaskScope;
type InboxLaneFilter = InboxLaneFilterId;

export type TaskListSortKey = "recent" | "priority";

const TASK_PRIORITY_RANK: Record<TaskPriority, number> = {
  urgent: 0,
  high: 1,
  medium: 2,
  low: 3,
};

function taskActivityTimestamp(task: TaskListItem): number {
  const value = task.last_activity_at ?? task.updated_at;
  if (!value) {
    return 0;
  }
  const parsed = Date.parse(value);
  return Number.isNaN(parsed) ? 0 : parsed;
}

interface UseTasksPageOptions {
  initialMode?: TaskViewMode;
  forceListData?: boolean;
}

export type CreateTaskDraftInput = TaskEditorDraft;

function useTasksPage(options: UseTasksPageOptions = {}) {
  const { activeWorkspace, activeWorkspaceId } = useActiveWorkspace();

  const [mode, setMode] = useState<TaskViewMode>(options.initialMode ?? "list");
  const [scopeFilter, setScopeFilter] = useState<TaskScopeFilter>("all");
  const [statusFilter, setStatusFilter] = useState<TaskStatus | null>(null);
  const [ownerFilter, setOwnerFilter] = useState<string | null>(null);
  const [priorityFilter, setPriorityFilter] = useState<TaskPriority | null>(null);
  const [sortBy, setSortBy] = useState<TaskListSortKey>("recent");
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedTaskId, setSelectedTaskIdState] = useState<string | null>(null);
  const [isSelectionDismissed, setSelectionDismissed] = useState(false);
  const [isCreateModalOpen, setCreateModalOpen] = useState(false);
  const [includeDrafts, setIncludeDrafts] = useState(true);
  const [createDraft, setCreateDraft] = useState<CreateTaskDraftInput>(EMPTY_TASK_EDITOR_DRAFT);
  const [createTemplateId, setCreateTemplateId] =
    useState<TaskTemplateId>(DEFAULT_TASK_TEMPLATE_ID);
  const [inboxLaneFilter, setInboxLaneFilter] = useState<InboxLaneFilter>("all");
  const [inboxStatusFilter, setInboxStatusFilter] = useState<TaskStatus | null>(null);
  const [inboxPriorityFilter, setInboxPriorityFilter] = useState<TaskPriority | null>(null);
  const [inboxUnreadOnly, setInboxUnreadOnly] = useState(false);
  const [inboxSearchQuery, setInboxSearchQuery] = useState("");

  const deferredSearchQuery = useDeferredValue(searchQuery);
  const deferredInboxQuery = useDeferredValue(inboxSearchQuery);
  const scopedWorkspace = workspaceFilterForActiveScope(scopeFilter, activeWorkspaceId);
  const backendScope = scopeFilter === "all" ? undefined : scopeFilter;

  const listFilters: TaskListFilter = useMemo(
    () => ({
      scope: backendScope,
      workspace: scopedWorkspace,
      status: statusFilter ?? undefined,
      include_drafts: includeDrafts,
      owner_ref: ownerFilter ?? undefined,
      limit: 100,
    }),
    [backendScope, includeDrafts, ownerFilter, scopedWorkspace, statusFilter]
  );

  const dashboardFilters: TaskDashboardFilter = useMemo(
    () => ({
      scope: backendScope,
      workspace: scopedWorkspace,
    }),
    [backendScope, scopedWorkspace]
  );

  const schedulerBacklogFilters = useMemo(
    () => ({
      limit: 5,
      workspace: scopedWorkspace,
      include_paused: true,
    }),
    [scopedWorkspace]
  );

  const inboxFilters: TaskInboxFilter = useMemo(
    () => ({
      scope: backendScope,
      workspace: scopedWorkspace,
      unread: inboxUnreadOnly ? true : undefined,
      query: deferredInboxQuery.trim() ? deferredInboxQuery.trim() : undefined,
      limit: 100,
    }),
    [backendScope, deferredInboxQuery, inboxUnreadOnly, scopedWorkspace]
  );

  const isListTab = mode === "list" || mode === "kanban" || options.forceListData === true;
  const tasksQuery = useTasks(listFilters, { enabled: isListTab });
  const dashboardQuery = useTaskDashboard(dashboardFilters, { enabled: mode === "dashboard" });
  const schedulerStatusQuery = useSchedulerStatus({ enabled: mode === "dashboard" });
  const schedulerBacklogQuery = useSchedulerBacklog(schedulerBacklogFilters, {
    enabled: mode === "dashboard",
  });
  const inboxQuery = useTaskInbox(inboxFilters, { enabled: mode === "inbox" });

  const createMutation = useCreateTask();
  const deleteMutation = useDeleteTask();
  const publishMutation = usePublishTask();
  const enqueueMutation = useEnqueueTaskRun();
  const approveMutation = useApproveTask();
  const rejectMutation = useRejectTask();
  const markReadMutation = useMarkTaskRead();
  const archiveMutation = useArchiveTask();
  const dismissMutation = useDismissTask();
  const pauseSchedulerMutation = usePauseScheduler();
  const resumeSchedulerMutation = useResumeScheduler();
  const drainSchedulerMutation = useDrainScheduler();

  const allTasks = tasksQuery.data ?? [];
  const visibleTasks = useMemo(() => {
    const filtered = allTasks.filter(task => {
      if (!matchesTaskQuery(task, deferredSearchQuery)) {
        return false;
      }
      if (priorityFilter && task.priority !== priorityFilter) {
        return false;
      }
      return true;
    });
    if (sortBy === "priority") {
      return [...filtered].sort((a, b) => {
        const rankA = a.priority ? TASK_PRIORITY_RANK[a.priority] : Number.MAX_SAFE_INTEGER;
        const rankB = b.priority ? TASK_PRIORITY_RANK[b.priority] : Number.MAX_SAFE_INTEGER;
        if (rankA !== rankB) {
          return rankA - rankB;
        }
        return taskActivityTimestamp(b) - taskActivityTimestamp(a);
      });
    }
    return [...filtered].sort((a, b) => taskActivityTimestamp(b) - taskActivityTimestamp(a));
  }, [allTasks, deferredSearchQuery, priorityFilter, sortBy]);

  const draftTasks = useMemo(() => visibleTasks.filter(taskIsDraft), [visibleTasks]);
  const statusCounts = useMemo(() => countTasksByStatus(allTasks), [allTasks]);
  const kanbanColumns: KanbanColumnGroup[] = useMemo(
    () => groupTasksForKanban(visibleTasks),
    [visibleTasks]
  );

  const ownerOptions = useMemo(() => {
    const seen = new Map<string, { kind?: TaskOwnerKind; ref: string }>();
    for (const task of allTasks) {
      const owner = task.owner;
      if (!owner?.ref) {
        continue;
      }

      if (!seen.has(owner.ref)) {
        seen.set(owner.ref, { kind: owner.kind, ref: owner.ref });
      }
    }

    return Array.from(seen.values()).sort((a, b) => a.ref.localeCompare(b.ref));
  }, [allTasks]);

  const setSelectedTaskId = useCallback((taskId: string | null) => {
    setSelectedTaskIdState(taskId);
    if (taskId !== null) {
      setSelectionDismissed(false);
    }
  }, []);

  const dismissSelectedTask = useCallback(() => {
    setSelectedTaskIdState(null);
    setSelectionDismissed(true);
  }, []);

  const effectiveSelectedTaskId = useMemo(() => {
    if (isSelectionDismissed) {
      return null;
    }
    if (selectedTaskId && visibleTasks.some(task => task.id === selectedTaskId)) {
      return selectedTaskId;
    }

    return visibleTasks[0]?.id ?? null;
  }, [isSelectionDismissed, selectedTaskId, visibleTasks]);

  const selectedTask: TaskListItem | null = useMemo(() => {
    if (!effectiveSelectedTaskId) {
      return null;
    }

    return visibleTasks.find(task => task.id === effectiveSelectedTaskId) ?? null;
  }, [effectiveSelectedTaskId, visibleTasks]);

  const handleModeChange = useCallback((next: TaskViewMode) => {
    setMode(next);
    setSearchQuery("");
  }, []);

  const handleScopeChange = useCallback((next: TaskScopeFilter) => {
    setScopeFilter(next);
    setSelectedTaskId(null);
  }, []);

  const handleStatusChange = useCallback((next: TaskStatus | null) => {
    setStatusFilter(next);
  }, []);

  const handleOwnerChange = useCallback((next: string | null) => {
    setOwnerFilter(next);
  }, []);

  const handlePriorityChange = useCallback((next: TaskPriority | null) => {
    setPriorityFilter(next);
  }, []);

  const handleSortChange = useCallback((next: TaskListSortKey) => {
    setSortBy(next);
  }, []);

  const handleInboxLaneChange = useCallback((next: InboxLaneFilter) => {
    setInboxLaneFilter(next);
  }, []);

  const handleInboxStatusChange = useCallback((next: TaskStatus | null) => {
    setInboxStatusFilter(next);
  }, []);

  const handleInboxPriorityChange = useCallback((next: TaskPriority | null) => {
    setInboxPriorityFilter(next);
  }, []);

  const handleInboxUnreadToggle = useCallback((next: boolean) => {
    setInboxUnreadOnly(next);
  }, []);

  const handleOpenCreateModal = useCallback(
    (templateId: TaskTemplateId = DEFAULT_TASK_TEMPLATE_ID) => {
      setCreateTemplateId(templateId);
      setCreateDraft(createTaskEditorDraft(templateId, activeWorkspaceId));
      setCreateModalOpen(true);
    },
    [activeWorkspaceId]
  );

  const handleCloseCreateModal = useCallback(() => setCreateModalOpen(false), []);
  const handleToggleIncludeDrafts = useCallback((next: boolean) => setIncludeDrafts(next), []);

  const handleTemplateChange = useCallback((templateId: TaskTemplateId) => {
    setCreateTemplateId(templateId);
    setCreateDraft(current => applyTemplateDefaultsToTaskEditorDraft(current, templateId));
  }, []);

  const submitCreateTask = useCallback(
    async (draft: CreateTaskDraftInput, asDraft: boolean) => {
      const trimmedTitle = draft.title.trim();
      if (!trimmedTitle) {
        toast.error("Provide a title before creating the task.");
        return null;
      }

      if (draft.scope === "workspace" && !activeWorkspaceId) {
        toast.error("Select an active workspace before creating a workspace task.");
        return null;
      }

      const payload = buildCreateTaskRequest(draft, {
        activeWorkspaceId,
        asDraft,
        templateId: createTemplateId,
      });

      try {
        const created = await createMutation.mutateAsync(payload);
        const wantsImmediateRun =
          !payload.draft && getTaskTemplate(createTemplateId).preview.enqueueOnSubmit;
        if (wantsImmediateRun && created.id) {
          try {
            await enqueueMutation.mutateAsync({ id: created.id });
          } catch (runError) {
            const message =
              runError instanceof Error ? runError.message : "Failed to enqueue first run";
            toast.error(`Task created, but enqueue failed: ${message}`);
          }
        }

        toast.success(
          payload.draft ? `Saved draft "${trimmedTitle}".` : `Created task "${trimmedTitle}".`
        );

        setCreateDraft(EMPTY_TASK_EDITOR_DRAFT);
        setCreateModalOpen(false);
        if (created.id) {
          setSelectedTaskId(created.id);
        }

        return created;
      } catch (error) {
        toast.error(error instanceof Error ? error.message : "Failed to create task");
        return null;
      }
    },
    [activeWorkspaceId, createMutation, createTemplateId, enqueueMutation]
  );

  const handlePublishTask = useCallback(
    async (taskId: string) => {
      try {
        await publishMutation.mutateAsync({ id: taskId });
        toast.success("Task published.");
      } catch (error) {
        toast.error(error instanceof Error ? error.message : "Failed to publish task");
      }
    },
    [publishMutation]
  );

  const handleDeleteTask = useCallback(
    async (taskId: string) => {
      if (effectiveSelectedTaskId === taskId) {
        setSelectedTaskId(null);
      }
      try {
        await deleteMutation.mutateAsync({ id: taskId });
        toast.success("Task deleted.");
      } catch (error) {
        toast.error(error instanceof Error ? error.message : "Failed to delete task");
      }
    },
    [deleteMutation, effectiveSelectedTaskId]
  );

  const handleApproveTask = useCallback(
    async (taskId: string) => {
      try {
        await approveMutation.mutateAsync({ id: taskId });
        toast.success("Task approved.");
      } catch (error) {
        toast.error(error instanceof Error ? error.message : "Failed to approve task");
      }
    },
    [approveMutation]
  );

  const handleRejectTask = useCallback(
    async (taskId: string) => {
      try {
        await rejectMutation.mutateAsync({ id: taskId });
        toast.success("Task rejected.");
      } catch (error) {
        toast.error(error instanceof Error ? error.message : "Failed to reject task");
      }
    },
    [rejectMutation]
  );

  const handleMarkTaskRead = useCallback(
    async (taskId: string) => {
      try {
        await markReadMutation.mutateAsync({ id: taskId });
      } catch (error) {
        toast.error(error instanceof Error ? error.message : "Failed to mark task read");
      }
    },
    [markReadMutation]
  );

  const handleArchiveTask = useCallback(
    async (taskId: string) => {
      try {
        await archiveMutation.mutateAsync({ id: taskId });
        toast.success("Task archived.");
      } catch (error) {
        toast.error(error instanceof Error ? error.message : "Failed to archive task");
      }
    },
    [archiveMutation]
  );

  const handleDismissTask = useCallback(
    async (taskId: string) => {
      try {
        await dismissMutation.mutateAsync({ id: taskId });
      } catch (error) {
        toast.error(error instanceof Error ? error.message : "Failed to dismiss task");
      }
    },
    [dismissMutation]
  );

  const handleRetryTask = useCallback(
    async (taskId: string) => {
      try {
        await enqueueMutation.mutateAsync({ id: taskId });
        toast.success("Retry enqueued.");
      } catch (error) {
        toast.error(error instanceof Error ? error.message : "Failed to enqueue retry");
      }
    },
    [enqueueMutation]
  );

  const handlePauseScheduler = useCallback(
    async (reason: string) => {
      try {
        const normalizedReason = reason.trim();
        await pauseSchedulerMutation.mutateAsync(
          normalizedReason ? { reason: normalizedReason } : {}
        );
        toast.success("Scheduler paused.");
      } catch (error) {
        const message = error instanceof Error ? error.message : "Failed to pause scheduler";
        toast.error(message);
        throw error;
      }
    },
    [pauseSchedulerMutation]
  );

  const handleResumeScheduler = useCallback(
    async (reason?: string) => {
      try {
        const normalizedReason = reason?.trim();
        await resumeSchedulerMutation.mutateAsync(
          normalizedReason ? { reason: normalizedReason } : {}
        );
        toast.success("Scheduler resumed.");
      } catch (error) {
        const message = error instanceof Error ? error.message : "Failed to resume scheduler";
        toast.error(message);
        throw error;
      }
    },
    [resumeSchedulerMutation]
  );

  const handleDrainScheduler = useCallback(
    async ({ reason, timeoutSeconds }: { reason?: string; timeoutSeconds?: number }) => {
      try {
        const normalizedReason = reason?.trim();
        await drainSchedulerMutation.mutateAsync({
          ...(normalizedReason ? { reason: normalizedReason } : {}),
          timeout_seconds: timeoutSeconds ?? 60,
        });
        toast.success("Scheduler drain requested.");
      } catch (error) {
        const message = error instanceof Error ? error.message : "Failed to drain scheduler";
        toast.error(message);
        throw error;
      }
    },
    [drainSchedulerMutation]
  );

  const isEmpty = !tasksQuery.isLoading && allTasks.length === 0;
  const isFilteredEmpty =
    !isEmpty && !tasksQuery.isLoading && visibleTasks.length === 0 && allTasks.length > 0;

  return {
    activeWorkspaceId,
    activeWorkspaceName: activeWorkspace?.name ?? null,
    allTasks,
    canSubmitCreate: createDraft.title.trim().length > 0,
    createDraft,
    createTemplate: getTaskTemplate(createTemplateId),
    createTemplateId,
    dashboard: dashboardQuery.data ?? null,
    dashboardError: dashboardQuery.error ?? null,
    dashboardLoading: dashboardQuery.isLoading && !dashboardQuery.data,
    dashboardFetching: dashboardQuery.isFetching,
    schedulerStatus: schedulerStatusQuery.data ?? null,
    schedulerStatusError: schedulerStatusQuery.error ?? null,
    schedulerStatusLoading: schedulerStatusQuery.isLoading && !schedulerStatusQuery.data,
    schedulerBacklog: schedulerBacklogQuery.data ?? null,
    schedulerBacklogError: schedulerBacklogQuery.error ?? null,
    schedulerBacklogLoading: schedulerBacklogQuery.isLoading && !schedulerBacklogQuery.data,
    dismissSelectedTask,
    draftTasks,
    effectiveSelectedTaskId,
    handleApproveTask,
    handleArchiveTask,
    handleCloseCreateModal,
    handleDeleteTask,
    handleDismissTask,
    handleInboxLaneChange,
    handleInboxPriorityChange,
    handleInboxStatusChange,
    handleInboxUnreadToggle,
    handleMarkTaskRead,
    handleModeChange,
    handleOpenCreateModal,
    handleOwnerChange,
    handlePriorityChange,
    handlePublishTask,
    handleRejectTask,
    handleDrainScheduler,
    handlePauseScheduler,
    handleRetryTask,
    handleResumeScheduler,
    handleScopeChange,
    handleSortChange,
    handleStatusChange,
    handleTemplateChange,
    handleToggleIncludeDrafts,
    inbox: inboxQuery.data ?? null,
    inboxError: inboxQuery.error ?? null,
    inboxLaneFilter,
    inboxLoading: inboxQuery.isLoading && !inboxQuery.data,
    inboxFetching: inboxQuery.isFetching,
    inboxPriorityFilter,
    inboxSearchQuery,
    inboxStatusFilter,
    inboxUnreadOnly,
    inboxUpdatedAt: inboxQuery.dataUpdatedAt,
    includeDrafts,
    isApproveTaskPending: approveMutation.isPending,
    isArchiveTaskPending: archiveMutation.isPending,
    isCreateModalOpen,
    isCreatePending: createMutation.isPending || enqueueMutation.isPending,
    isDeletePending: deleteMutation.isPending,
    isDismissTaskPending: dismissMutation.isPending,
    isEmpty,
    isFilteredEmpty,
    isMarkReadTaskPending: markReadMutation.isPending,
    isPublishPending: publishMutation.isPending,
    isRejectTaskPending: rejectMutation.isPending,
    isRetryTaskPending: enqueueMutation.isPending,
    isSchedulerDrainPending: drainSchedulerMutation.isPending,
    isSchedulerPausePending: pauseSchedulerMutation.isPending,
    isSchedulerResumePending: resumeSchedulerMutation.isPending,
    kanbanColumns,
    kanbanColumnDefinitions: getKanbanColumns(),
    listError: tasksQuery.error ?? null,
    listLoading: tasksQuery.isLoading && allTasks.length === 0,
    listUpdatedAt: tasksQuery.dataUpdatedAt,
    mode,
    ownerFilter,
    ownerOptions,
    priorityFilter,
    scopeFilter,
    searchQuery,
    selectedTask,
    setCreateDraft,
    setInboxSearchQuery,
    setSearchQuery,
    setSelectedTaskId,
    sortBy,
    statusCounts,
    statusFilter,
    submitCreateTask,
    tasksCount: allTasks.length,
    visibleTasks,
  };
}

export { useTasksPage };
export type { InboxLaneFilter, TaskScopeFilter, UseTasksPageOptions };
