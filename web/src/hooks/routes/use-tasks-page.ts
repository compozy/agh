import { useCallback, useDeferredValue, useMemo, useState } from "react";
import { toast } from "sonner";

import {
  countTasksByStatus,
  matchesTaskQuery,
  taskIsDraft,
  useCreateTask,
  useEnqueueTaskRun,
  usePublishTask,
  useTaskDashboard,
  useTaskInbox,
  useTasks,
} from "@/systems/tasks";
import {
  DEFAULT_TASK_TEMPLATE_ID,
  applyTemplateToCreatePayload,
  getTaskTemplate,
} from "@/systems/tasks/lib/task-templates";
import type { TaskTemplateId } from "@/systems/tasks/lib/task-templates";
import { getKanbanColumns, groupTasksForKanban } from "@/systems/tasks/lib/task-grouping";
import type { KanbanColumnGroup } from "@/systems/tasks/lib/task-grouping";
import type {
  CreateTaskRequest,
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

type TaskScopeFilter = "all" | TaskScope;

interface UseTasksPageOptions {
  initialMode?: TaskViewMode;
}

export interface CreateTaskDraftInput {
  title: string;
  description: string;
  scope: TaskScope;
  priority: TaskPriority;
  ownerKind: TaskOwnerKind | "";
  ownerRef: string;
  parentTaskId: string;
  maxAttempts: number | null;
  approvalPolicy: "none" | "manual";
  networkChannel: string;
  identifier: string;
}

const EMPTY_DRAFT: CreateTaskDraftInput = {
  title: "",
  description: "",
  scope: "workspace",
  priority: "medium",
  ownerKind: "",
  ownerRef: "",
  parentTaskId: "",
  maxAttempts: null,
  approvalPolicy: "none",
  networkChannel: "",
  identifier: "",
};

function useTasksPage(options: UseTasksPageOptions = {}) {
  const { activeWorkspace, activeWorkspaceId } = useActiveWorkspace();

  const [mode, setMode] = useState<TaskViewMode>(options.initialMode ?? "list");
  const [scopeFilter, setScopeFilter] = useState<TaskScopeFilter>("all");
  const [statusFilter, setStatusFilter] = useState<TaskStatus | null>(null);
  const [ownerFilter, setOwnerFilter] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedTaskId, setSelectedTaskId] = useState<string | null>(null);
  const [isCreateModalOpen, setCreateModalOpen] = useState(false);
  const [includeDrafts, setIncludeDrafts] = useState(true);
  const [createDraft, setCreateDraft] = useState<CreateTaskDraftInput>(EMPTY_DRAFT);
  const [createTemplateId, setCreateTemplateId] =
    useState<TaskTemplateId>(DEFAULT_TASK_TEMPLATE_ID);

  const deferredSearchQuery = useDeferredValue(searchQuery);
  const scopedWorkspace =
    scopeFilter === "workspace" ? (activeWorkspaceId ?? undefined) : undefined;

  const listFilters: TaskListFilter = useMemo(
    () => ({
      scope: scopeFilter === "all" ? undefined : scopeFilter,
      workspace: scopedWorkspace,
      status: statusFilter ?? undefined,
      include_drafts: includeDrafts,
      owner_ref: ownerFilter ?? undefined,
      limit: 100,
    }),
    [includeDrafts, ownerFilter, scopeFilter, scopedWorkspace, statusFilter]
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

  const createMutation = useCreateTask();
  const publishMutation = usePublishTask();
  const enqueueMutation = useEnqueueTaskRun();

  const allTasks = tasksQuery.data ?? [];
  const visibleTasks = useMemo(() => {
    return allTasks.filter(task => matchesTaskQuery(task, deferredSearchQuery));
  }, [allTasks, deferredSearchQuery]);

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

  const handleOwnerChange = useCallback((next: string | null) => {
    setOwnerFilter(next);
  }, []);

  const handleOpenCreateModal = useCallback(
    (templateId: TaskTemplateId = DEFAULT_TASK_TEMPLATE_ID) => {
      const template = getTaskTemplate(templateId);
      setCreateTemplateId(templateId);
      setCreateDraft({
        ...EMPTY_DRAFT,
        scope: activeWorkspaceId ? "workspace" : "global",
        priority: template.defaults.priority ?? "medium",
        maxAttempts:
          typeof template.defaults.max_attempts === "number"
            ? template.defaults.max_attempts
            : null,
        approvalPolicy: template.defaults.approval_policy ?? "none",
        networkChannel: template.defaults.network_channel ?? "",
      });
      setCreateModalOpen(true);
    },
    [activeWorkspaceId]
  );

  const handleCloseCreateModal = useCallback(() => setCreateModalOpen(false), []);
  const handleToggleIncludeDrafts = useCallback((next: boolean) => setIncludeDrafts(next), []);

  const handleTemplateChange = useCallback((templateId: TaskTemplateId) => {
    const template = getTaskTemplate(templateId);
    setCreateTemplateId(templateId);
    setCreateDraft(current => ({
      ...current,
      priority: template.defaults.priority ?? current.priority,
      maxAttempts:
        typeof template.defaults.max_attempts === "number"
          ? template.defaults.max_attempts
          : current.maxAttempts,
      approvalPolicy: template.defaults.approval_policy ?? current.approvalPolicy,
      networkChannel: template.defaults.network_channel ?? current.networkChannel,
    }));
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

      const ownerKind = draft.ownerKind || undefined;
      const ownerRef = draft.ownerRef.trim();
      const owner =
        ownerKind && ownerRef
          ? { kind: ownerKind, ref: ownerRef }
          : ownerKind === undefined && ownerRef === ""
            ? undefined
            : null;

      const basePayload: CreateTaskRequest = {
        title: trimmedTitle,
        description: draft.description.trim() || undefined,
        scope: draft.scope,
        workspace: draft.scope === "workspace" ? (activeWorkspaceId ?? undefined) : undefined,
        priority: draft.priority,
        max_attempts: draft.maxAttempts ?? undefined,
        draft: asDraft || createTemplateId === "recurring",
        owner,
        approval_policy: draft.approvalPolicy === "manual" ? "manual" : undefined,
        network_channel: draft.networkChannel.trim() || undefined,
        identifier: draft.identifier.trim() || undefined,
      };

      const payload = applyTemplateToCreatePayload(basePayload, createTemplateId);

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

        setCreateDraft(EMPTY_DRAFT);
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
    dashboard: dashboardQuery.data,
    dashboardError: dashboardQuery.error ?? null,
    dashboardLoading: dashboardQuery.isLoading && !dashboardQuery.data,
    draftTasks,
    effectiveSelectedTaskId,
    handleCloseCreateModal,
    handleModeChange,
    handleOpenCreateModal,
    handleOwnerChange,
    handlePublishTask,
    handleScopeChange,
    handleStatusChange,
    handleTemplateChange,
    handleToggleIncludeDrafts,
    inbox: inboxQuery.data,
    inboxError: inboxQuery.error ?? null,
    inboxLoading: inboxQuery.isLoading && !inboxQuery.data,
    includeDrafts,
    isCreateModalOpen,
    isCreatePending: createMutation.isPending || enqueueMutation.isPending,
    isEmpty,
    isFilteredEmpty,
    isPublishPending: publishMutation.isPending,
    kanbanColumns,
    kanbanColumnDefinitions: getKanbanColumns(),
    listError: tasksQuery.error ?? null,
    listLoading: tasksQuery.isLoading && allTasks.length === 0,
    mode,
    ownerFilter,
    ownerOptions,
    scopeFilter,
    searchQuery,
    selectedTask,
    setCreateDraft,
    setSearchQuery,
    setSelectedTaskId,
    statusCounts,
    statusFilter,
    submitCreateTask,
    tasksCount: allTasks.length,
    visibleTasks,
  };
}

export { useTasksPage };
export type { TaskScopeFilter, UseTasksPageOptions };
