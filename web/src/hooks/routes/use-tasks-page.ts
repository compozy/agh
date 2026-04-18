import { useCallback, useDeferredValue, useMemo, useState } from "react";

import {
  countTasksByStatus,
  matchesTaskQuery,
  taskIsDraft,
  useTaskDashboard,
  useTaskInbox,
  useTasks,
} from "@/systems/tasks";
import type {
  TaskDashboardFilter,
  TaskInboxFilter,
  TaskListFilter,
  TaskListItem,
  TaskScope,
  TaskStatus,
  TaskViewMode,
} from "@/systems/tasks";
import { useActiveWorkspace } from "@/systems/workspace";

type TaskScopeFilter = "all" | TaskScope;

interface UseTasksPageOptions {
  initialMode?: TaskViewMode;
}

function useTasksPage(options: UseTasksPageOptions = {}) {
  const { activeWorkspace, activeWorkspaceId } = useActiveWorkspace();

  const [mode, setMode] = useState<TaskViewMode>(options.initialMode ?? "list");
  const [scopeFilter, setScopeFilter] = useState<TaskScopeFilter>("all");
  const [statusFilter, setStatusFilter] = useState<TaskStatus | null>(null);
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedTaskId, setSelectedTaskId] = useState<string | null>(null);
  const [isCreateModalOpen, setCreateModalOpen] = useState(false);
  const [includeDrafts, setIncludeDrafts] = useState(true);

  const deferredSearchQuery = useDeferredValue(searchQuery);
  const scopedWorkspace =
    scopeFilter === "workspace" ? (activeWorkspaceId ?? undefined) : undefined;

  const listFilters: TaskListFilter = useMemo(
    () => ({
      scope: scopeFilter === "all" ? undefined : scopeFilter,
      workspace: scopedWorkspace,
      status: statusFilter ?? undefined,
      include_drafts: includeDrafts,
      limit: 100,
    }),
    [includeDrafts, scopeFilter, scopedWorkspace, statusFilter]
  );

  const dashboardFilters: TaskDashboardFilter = useMemo(
    () => ({
      scope: scopeFilter === "all" ? undefined : scopeFilter,
      workspace: scopedWorkspace,
    }),
    [scopeFilter, scopedWorkspace]
  );

  const inboxFilters: TaskInboxFilter = useMemo(
    () => ({
      scope: scopeFilter === "all" ? undefined : scopeFilter,
      workspace: scopedWorkspace,
    }),
    [scopeFilter, scopedWorkspace]
  );

  const isListTab = mode === "list" || mode === "kanban";
  const tasksQuery = useTasks(listFilters, { enabled: isListTab });
  const dashboardQuery = useTaskDashboard(dashboardFilters, { enabled: mode === "dashboard" });
  const inboxQuery = useTaskInbox(inboxFilters, { enabled: mode === "inbox" });

  const allTasks = tasksQuery.data ?? [];
  const visibleTasks = useMemo(() => {
    return allTasks.filter(task => matchesTaskQuery(task, deferredSearchQuery));
  }, [allTasks, deferredSearchQuery]);

  const draftTasks = useMemo(() => visibleTasks.filter(taskIsDraft), [visibleTasks]);
  const statusCounts = useMemo(() => countTasksByStatus(allTasks), [allTasks]);

  const effectiveSelectedTaskId = useMemo(() => {
    if (selectedTaskId && visibleTasks.some(task => task.id === selectedTaskId)) {
      return selectedTaskId;
    }

    return visibleTasks[0]?.id ?? null;
  }, [selectedTaskId, visibleTasks]);

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

  const handleOpenCreateModal = useCallback(() => setCreateModalOpen(true), []);
  const handleCloseCreateModal = useCallback(() => setCreateModalOpen(false), []);
  const handleToggleIncludeDrafts = useCallback((next: boolean) => setIncludeDrafts(next), []);

  const isEmpty = !tasksQuery.isLoading && allTasks.length === 0;

  return {
    activeWorkspaceId,
    activeWorkspaceName: activeWorkspace?.name ?? null,
    allTasks,
    dashboard: dashboardQuery.data,
    dashboardError: dashboardQuery.error ?? null,
    dashboardLoading: dashboardQuery.isLoading && !dashboardQuery.data,
    draftTasks,
    effectiveSelectedTaskId,
    handleCloseCreateModal,
    handleModeChange,
    handleOpenCreateModal,
    handleScopeChange,
    handleStatusChange,
    handleToggleIncludeDrafts,
    inbox: inboxQuery.data,
    inboxError: inboxQuery.error ?? null,
    inboxLoading: inboxQuery.isLoading && !inboxQuery.data,
    includeDrafts,
    isCreateModalOpen,
    isEmpty,
    listError: tasksQuery.error ?? null,
    listLoading: tasksQuery.isLoading && allTasks.length === 0,
    mode,
    scopeFilter,
    searchQuery,
    selectedTask,
    setSearchQuery,
    setSelectedTaskId,
    statusCounts,
    statusFilter,
    tasksCount: allTasks.length,
    visibleTasks,
  };
}

export { useTasksPage };
export type { TaskScopeFilter, UseTasksPageOptions };
