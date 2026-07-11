import { useCallback, useDeferredValue, useMemo, useState } from "react";

import {
  useTaskInbox,
  useTasks,
  type TaskDashboardFilter,
  type TaskInboxFilter,
  type TaskListFilter,
  type TaskListSortKey,
  type TaskPriority,
  type TaskStatus,
  type TaskViewMode,
} from "@/systems/tasks";
import { groupTasksForKanban } from "@/systems/tasks/lib/task-grouping";
import type { KanbanColumnGroup } from "@/systems/tasks/lib/task-grouping";
import type { InboxLaneFilterId } from "@/systems/tasks/lib/inbox-grouping";
import { taskStatusCountsFromFacets } from "@/systems/tasks/lib/task-list-query";
import type { TaskFilterOwnerOption } from "@/systems/tasks/lib/tasks-list-filters";
import { useDaemonStatus } from "@/systems/status";
import { useActiveWorkspace } from "@/systems/workspace";

import { DEFAULT_TASK_LIST_LIMIT, defaultTaskCatalogFilter } from "./task-catalog-route-filter";
import { useTasksDashboardPage } from "./use-tasks-dashboard-page";
import { useTasksPageActions } from "./use-tasks-page-actions";
import { taskScopeForActiveWorkspace } from "./workspace-scope-filter";

type InboxLaneFilter = InboxLaneFilterId;

interface UseTasksPageOptions {
  initialMode?: TaskViewMode;
  forceListData?: boolean;
}

function useTasksPage(options: UseTasksPageOptions = {}) {
  const workspace = useActiveWorkspace();
  const daemonStatus = useDaemonStatus();
  const { activeWorkspace } = workspace;
  const userHomeDir = daemonStatus.data?.user_home_dir;
  const [mode, setMode] = useState<TaskViewMode>(options.initialMode ?? "list");
  const [statusFilter, setStatusFilter] = useState<TaskStatus | null>(null);
  const [ownerFilter, setOwnerFilter] = useState<TaskFilterOwnerOption | null>(null);
  const [priorityFilter, setPriorityFilter] = useState<TaskPriority | null>(null);
  const [sortBy, setSortBy] = useState<TaskListSortKey>("recent");
  const [searchQuery, setSearchQuery] = useState("");
  const [inboxLaneFilter, setInboxLaneFilter] = useState<InboxLaneFilter>("all");
  const [inboxStatusFilter, setInboxStatusFilter] = useState<TaskStatus | null>(null);
  const [inboxPriorityFilter, setInboxPriorityFilter] = useState<TaskPriority | null>(null);
  const [inboxUnreadOnly, setInboxUnreadOnly] = useState(false);
  const [inboxSearchQuery, setInboxSearchQuery] = useState("");
  const deferredSearchQuery = useDeferredValue(searchQuery);
  const deferredInboxQuery = useDeferredValue(inboxSearchQuery);

  const activeTaskScope = useMemo(
    () => taskScopeForActiveWorkspace(activeWorkspace, userHomeDir),
    [activeWorkspace, userHomeDir]
  );
  const hasActiveTaskScope = activeTaskScope !== null;
  const scopeSourceError =
    (!userHomeDir ? daemonStatus.error : null) ??
    (!activeWorkspace ? workspace.error : null) ??
    null;
  const scopeLoading =
    !hasActiveTaskScope &&
    !scopeSourceError &&
    (!workspace.hasHydrated || workspace.isPending || daemonStatus.isPending);
  const scopeError = useMemo(() => {
    if (hasActiveTaskScope || scopeLoading) {
      return null;
    }
    if (scopeSourceError) {
      return scopeSourceError;
    }
    if (!userHomeDir) {
      return new Error("Daemon status did not provide a user home directory.");
    }
    return new Error("No active workspace is available for task scope.");
  }, [activeWorkspace, hasActiveTaskScope, scopeLoading, scopeSourceError, userHomeDir]);
  const listFilters: TaskListFilter = useMemo(
    () => ({
      ...(activeTaskScope ? defaultTaskCatalogFilter(activeTaskScope) : {}),
      status: statusFilter ?? undefined,
      priority: priorityFilter ?? undefined,
      include_drafts: true,
      owner_kind: ownerFilter?.kind,
      owner_ref: ownerFilter?.ref,
      query: deferredSearchQuery.trim() || undefined,
      sort: sortBy,
      limit: DEFAULT_TASK_LIST_LIMIT,
    }),
    [activeTaskScope, deferredSearchQuery, ownerFilter, priorityFilter, sortBy, statusFilter]
  );
  const dashboardFilters: TaskDashboardFilter = useMemo(
    () => ({
      scope: activeTaskScope?.scope,
      workspace: activeTaskScope?.workspace,
    }),
    [activeTaskScope]
  );
  const inboxFilters: TaskInboxFilter = useMemo(
    () => ({
      scope: activeTaskScope?.scope,
      workspace: activeTaskScope?.workspace,
      lane: inboxLaneFilter === "all" ? undefined : inboxLaneFilter,
      status: inboxStatusFilter ?? undefined,
      priority: inboxPriorityFilter ?? undefined,
      unread: inboxUnreadOnly ? true : undefined,
      query: deferredInboxQuery.trim() || undefined,
      limit: DEFAULT_TASK_LIST_LIMIT,
    }),
    [
      activeTaskScope,
      deferredInboxQuery,
      inboxLaneFilter,
      inboxPriorityFilter,
      inboxStatusFilter,
      inboxUnreadOnly,
    ]
  );
  const inboxBadgeFilters: TaskInboxFilter = useMemo(
    () => ({
      scope: activeTaskScope?.scope,
      workspace: activeTaskScope?.workspace,
      limit: 1,
    }),
    [activeTaskScope]
  );

  const isListTab =
    hasActiveTaskScope && (mode === "list" || mode === "kanban" || options.forceListData === true);
  const tasksQuery = useTasks(listFilters, { enabled: isListTab });
  const inboxQuery = useTaskInbox(inboxFilters, {
    enabled: hasActiveTaskScope && mode === "inbox",
  });
  const inboxBadgeQuery = useTaskInbox(inboxBadgeFilters, { enabled: hasActiveTaskScope });
  const dashboard = useTasksDashboardPage(
    dashboardFilters,
    hasActiveTaskScope && mode === "dashboard"
  );

  const allTasks = tasksQuery.data ?? [];
  const visibleTasks = allTasks;
  const statusCounts = useMemo(
    () => taskStatusCountsFromFacets(tasksQuery.facets),
    [tasksQuery.facets]
  );
  const kanbanColumns: KanbanColumnGroup[] = useMemo(
    () => groupTasksForKanban(visibleTasks),
    [visibleTasks]
  );
  const ownerOptions = useMemo(
    () =>
      tasksQuery.facets.owners.map(facet => ({
        kind: facet.owner.kind,
        ref: facet.owner.ref,
      })),
    [tasksQuery.facets.owners]
  );

  const effectiveSelectedTaskId = visibleTasks[0]?.id ?? null;
  const actions = useTasksPageActions();
  const loadMoreTasks = useCallback(() => {
    void tasksQuery.fetchNextPage();
  }, [tasksQuery]);
  const loadMoreInbox = useCallback(() => {
    void inboxQuery.fetchNextPage();
  }, [inboxQuery]);
  const retryTasks = useCallback(() => {
    if (tasksQuery.isFetchNextPageError) {
      void tasksQuery.fetchNextPage();
      return;
    }
    void tasksQuery.refetch();
  }, [tasksQuery]);
  const retryInbox = useCallback(() => {
    if (inboxQuery.isFetchNextPageError) {
      void inboxQuery.fetchNextPage();
      return;
    }
    void inboxQuery.refetch();
  }, [inboxQuery]);

  const handleModeChange = useCallback((next: TaskViewMode) => {
    setMode(next);
    setSearchQuery("");
  }, []);
  const hasListFilters = Boolean(
    statusFilter || ownerFilter || priorityFilter || deferredSearchQuery.trim()
  );
  const isEmpty =
    hasActiveTaskScope &&
    !scopeLoading &&
    !scopeError &&
    !tasksQuery.isLoading &&
    !tasksQuery.error &&
    tasksQuery.total === 0 &&
    !hasListFilters;

  return {
    ...actions,
    ...dashboard,
    activeWorkspaceName: activeWorkspace?.name ?? null,
    dashboardError: scopeError ?? dashboard.dashboardError,
    dashboardLoading: scopeLoading || dashboard.dashboardLoading,
    effectiveSelectedTaskId,
    handleInboxLaneChange: setInboxLaneFilter,
    handleInboxPriorityChange: setInboxPriorityFilter,
    handleInboxStatusChange: setInboxStatusFilter,
    handleInboxUnreadToggle: setInboxUnreadOnly,
    handleModeChange,
    handleOwnerChange: setOwnerFilter,
    handlePriorityChange: setPriorityFilter,
    handleSortChange: setSortBy,
    handleStatusChange: setStatusFilter,
    hasMoreInbox: inboxQuery.hasNextPage,
    hasMoreTasks: tasksQuery.hasNextPage,
    inbox: inboxQuery.data ?? null,
    inboxError: scopeError ?? inboxQuery.error ?? null,
    inboxLaneFilter,
    inboxLoading: scopeLoading || (inboxQuery.isLoading && !inboxQuery.data),
    inboxPriorityFilter,
    inboxSearchQuery,
    inboxStatusFilter,
    inboxUnreadOnly,
    inboxUnreadCount: inboxBadgeQuery.data?.unread_total,
    inboxUpdatedAt: inboxQuery.dataUpdatedAt,
    isEmpty,
    isLoadingMoreInbox: inboxQuery.isFetchingNextPage,
    isLoadingMoreTasks: tasksQuery.isFetchingNextPage,
    kanbanColumns,
    listError: scopeError ?? tasksQuery.error ?? null,
    listLoading: scopeLoading || (tasksQuery.isLoading && allTasks.length === 0),
    listUpdatedAt: tasksQuery.dataUpdatedAt,
    loadMoreInbox,
    loadMoreTasks,
    mode,
    ownerFilter,
    ownerOptions,
    priorityFilter,
    retryInbox,
    retryTasks,
    searchQuery,
    setInboxSearchQuery,
    setSearchQuery,
    sortBy,
    statusCounts,
    statusFilter,
    hasActiveTaskScope,
    scopeError,
    scopeLoading,
    tasksCount: tasksQuery.dataUpdatedAt > 0 ? tasksQuery.total : undefined,
    visibleTasks,
  };
}

export { useTasksPage };
export type { InboxLaneFilter, UseTasksPageOptions };
