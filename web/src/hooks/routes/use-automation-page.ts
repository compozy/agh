import { startTransition, useDeferredValue, useMemo, useState } from "react";
import { toast } from "sonner";

import {
  automationJobToDraft,
  automationTriggerToDraft,
  createAutomationDialogHandle,
  createAutomationJobDraft,
  createAutomationTriggerDraft,
  filterAutomationJobs,
  filterAutomationTriggers,
  normalizeAutomationRetry,
  sortAutomationJobs,
  sortAutomationTriggers,
  useAutomationJob,
  useAutomationJobs,
  useAutomationJobRuns,
  useAutomationTrigger,
  useAutomationTriggers,
  useAutomationTriggerRuns,
  useCreateAutomationJob,
  useCreateAutomationTrigger,
  useDeleteAutomationJob,
  useDeleteAutomationTrigger,
  useTriggerAutomationJob,
  useUpdateAutomationJob,
  useUpdateAutomationTrigger,
} from "@/systems/automation";
import type {
  AutomationRun,
  AutomationScopeFilter,
  CreateAutomationJobRequest,
  CreateAutomationTriggerRequest,
} from "@/systems/automation";
import { useActiveWorkspace } from "@/systems/workspace";

type JobEditorState =
  | {
      draft: CreateAutomationJobRequest;
      mode: "create";
    }
  | {
      draft: CreateAutomationJobRequest;
      id: string;
      mode: "edit";
    };

type TriggerEditorState =
  | {
      draft: CreateAutomationTriggerRequest;
      mode: "create";
    }
  | {
      draft: CreateAutomationTriggerRequest;
      id: string;
      mode: "edit";
    };

function buildEmptyState({
  hasQuery,
  kind,
  onCreate,
}: {
  hasQuery: boolean;
  kind: "jobs" | "triggers";
  onCreate: () => void;
}) {
  if (hasQuery) {
    return {
      description: "Try a different search term or adjust the current scope filter.",
      icon: "search" as const,
      title: kind === "jobs" ? "No jobs found" : "No triggers found",
    };
  }

  if (kind === "jobs") {
    return {
      actionLabel: "Create Job",
      description:
        "Scheduled jobs dispatch prompts to agents on a time-based cadence. Create your first job to start automating.",
      icon: "jobs" as const,
      onAction: onCreate,
      title: "No jobs configured",
    };
  }

  return {
    actionLabel: "Create Trigger",
    description:
      "Event-driven triggers react to daemon events, webhooks, and extension signals. Create your first trigger to enable reactive automation.",
    icon: "triggers" as const,
    onAction: onCreate,
    title: "No triggers configured",
  };
}

function resolveSelectedId<T extends { id: string }>(selectedId: string | null, items: T[]) {
  if (selectedId && items.some(item => item.id === selectedId)) {
    return selectedId;
  }

  return items[0]?.id ?? null;
}

function useAutomationPageBase() {
  const { activeWorkspace, activeWorkspaceId } = useActiveWorkspace();
  const [scopeFilter, setScopeFilter] = useState<AutomationScopeFilter>("all");
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState("");
  const deferredSearchQuery = useDeferredValue(searchQuery);

  const scopedWorkspaceId =
    scopeFilter === "workspace" ? (activeWorkspaceId ?? undefined) : undefined;

  const listFilters = useMemo(
    () => ({
      limit: 50,
      scope: scopeFilter === "all" ? undefined : scopeFilter,
      workspace_id: scopedWorkspaceId,
    }),
    [scopeFilter, scopedWorkspaceId]
  );

  return {
    activeWorkspace,
    activeWorkspaceId,
    deferredSearchQuery,
    listFilters,
    scopeFilter,
    searchQuery,
    selectedId,
    setScopeFilter,
    setSearchQuery,
    setSelectedId,
  };
}

function useAutomationJobsPage() {
  const page = useAutomationPageBase();
  const [editor, setEditor] = useState<JobEditorState | null>(null);
  const [queuedRun, setQueuedRun] = useState<{ jobId: string; run: AutomationRun } | null>(null);
  const editorHandle = useMemo(() => createAutomationDialogHandle(), []);

  const jobsQuery = useAutomationJobs(page.listFilters);
  const jobs = jobsQuery.data ?? [];
  const visibleJobs = useMemo(
    () => sortAutomationJobs(filterAutomationJobs(jobs, page.deferredSearchQuery)),
    [jobs, page.deferredSearchQuery]
  );
  const effectiveSelectedJobId = useMemo(
    () => resolveSelectedId(page.selectedId, visibleJobs),
    [page.selectedId, visibleJobs]
  );

  const jobDetailQuery = useAutomationJob(effectiveSelectedJobId ?? "", {
    enabled: Boolean(effectiveSelectedJobId),
  });
  const jobRunsQuery = useAutomationJobRuns(
    effectiveSelectedJobId ?? "",
    { limit: 10 },
    { enabled: Boolean(effectiveSelectedJobId) }
  );

  const createJobMutation = useCreateAutomationJob();
  const updateJobMutation = useUpdateAutomationJob();
  const deleteJobMutation = useDeleteAutomationJob();
  const triggerJobMutation = useTriggerAutomationJob();

  const selectedJob =
    jobDetailQuery.data ??
    visibleJobs.find(job => job.id === effectiveSelectedJobId) ??
    jobs.find(job => job.id === effectiveSelectedJobId);

  const displayedRuns = useMemo(() => {
    const runs = jobRunsQuery.data ?? [];
    if (
      queuedRun &&
      queuedRun.jobId === effectiveSelectedJobId &&
      !runs.some(run => run.id === queuedRun.run.id)
    ) {
      return [queuedRun.run, ...runs];
    }

    return runs;
  }, [effectiveSelectedJobId, jobRunsQuery.data, queuedRun]);

  const handleScopeChange = (nextScope: AutomationScopeFilter) => {
    startTransition(() => {
      page.setScopeFilter(nextScope);
      page.setSelectedId(null);
      setEditor(null);
      setQueuedRun(null);
    });
  };

  const handleCreate = () => {
    setEditor({
      draft: createAutomationJobDraft(page.activeWorkspaceId),
      mode: "create",
    });
  };

  const handleEdit = () => {
    if (!selectedJob) {
      return;
    }

    setEditor({
      draft: automationJobToDraft(selectedJob),
      id: selectedJob.id,
      mode: "edit",
    });
  };

  const handleSubmit = async () => {
    if (!editor) {
      return;
    }

    try {
      const payload = {
        ...editor.draft,
        retry: normalizeAutomationRetry(editor.draft.retry ?? undefined),
      };
      const job =
        editor.mode === "create"
          ? await createJobMutation.mutateAsync(payload)
          : await updateJobMutation.mutateAsync({
              data: payload,
              id: editor.id,
            });

      page.setSelectedId(job.id);
      setEditor(null);
      toast.success(
        editor.mode === "create" ? `Created job ${job.name}.` : `Updated job ${job.name}.`
      );
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to save automation job");
    }
  };

  const handleDelete = async () => {
    if (!selectedJob) {
      return;
    }

    try {
      await deleteJobMutation.mutateAsync({ id: selectedJob.id });
      page.setSelectedId(null);
      setQueuedRun(null);
      toast.success(`Deleted ${selectedJob.name}.`);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to delete automation job");
    }
  };

  const handleToggleEnabled = async (enabled: boolean) => {
    if (!selectedJob) {
      return;
    }

    try {
      await updateJobMutation.mutateAsync({
        data: { enabled },
        id: selectedJob.id,
      });
      toast.success(`${enabled ? "Enabled" : "Disabled"} ${selectedJob.name}.`);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to update automation state");
    }
  };

  const handleTriggerNow = async () => {
    if (!selectedJob) {
      return;
    }

    try {
      const run = await triggerJobMutation.mutateAsync({ id: selectedJob.id });
      setQueuedRun({ jobId: selectedJob.id, run });
      toast.success(`Queued run ${run.id}.`);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to trigger automation job");
    }
  };

  const emptyState =
    visibleJobs.length === 0
      ? buildEmptyState({
          hasQuery: page.deferredSearchQuery.trim() !== "",
          kind: "jobs",
          onCreate: handleCreate,
        })
      : null;

  const listPanelProps = {
    activeWorkspaceName: page.activeWorkspace?.name,
    errorMessage: jobsQuery.error?.message ?? null,
    isLoading: jobsQuery.isLoading,
    jobs: visibleJobs,
    kind: "jobs" as const,
    onSearchChange: page.setSearchQuery,
    onSelect: (id: string) =>
      startTransition(() => {
        page.setSelectedId(id);
        setQueuedRun(null);
      }),
    scopeFilter: page.scopeFilter,
    searchQuery: page.searchQuery,
    selectedId: effectiveSelectedJobId,
    totalCount: jobs.length,
    triggers: [],
  };

  const detailPanelProps = {
    emptyState,
    error: jobDetailQuery.error,
    state: {
      isDeleting: deleteJobMutation.isPending,
      isLoading: jobDetailQuery.isLoading,
      isTogglePending: updateJobMutation.isPending,
      isTriggerPending: triggerJobMutation.isPending,
    },
    item: selectedJob,
    kind: "jobs" as const,
    onDelete: () => {
      void handleDelete();
    },
    onEdit: handleEdit,
    onToggleEnabled: (enabled: boolean) => {
      void handleToggleEnabled(enabled);
    },
    onTriggerNow: () => {
      void handleTriggerNow();
    },
    runs: displayedRuns,
    runsError: jobRunsQuery.error,
    runsLoading: jobRunsQuery.isLoading,
  };

  const editorDialogProps = {
    activeWorkspaceId: page.activeWorkspaceId,
    handle: editorHandle,
    editor: editor
      ? {
          ...editor,
          kind: "jobs" as const,
          isPending: createJobMutation.isPending || updateJobMutation.isPending,
          onCancel: () => setEditor(null),
          onChange: (draft: CreateAutomationJobRequest) =>
            setEditor(current => (current ? { ...current, draft } : current)),
          onSubmit: () => {
            void handleSubmit();
          },
        }
      : null,
  };

  return {
    currentTotalCount: jobs.length,
    detailPanelProps,
    editorDialogProps,
    handleCreate,
    handleScopeChange,
    initialError: jobsQuery.error && jobs.length === 0 ? jobsQuery.error : null,
    isInitialLoading: jobsQuery.isLoading && jobs.length === 0,
    listPanelProps,
    scopeFilter: page.scopeFilter,
  };
}

function useAutomationTriggersPage() {
  const page = useAutomationPageBase();
  const [editor, setEditor] = useState<TriggerEditorState | null>(null);
  const editorHandle = useMemo(() => createAutomationDialogHandle(), []);

  const triggersQuery = useAutomationTriggers(page.listFilters);
  const triggers = triggersQuery.data ?? [];
  const visibleTriggers = useMemo(
    () => sortAutomationTriggers(filterAutomationTriggers(triggers, page.deferredSearchQuery)),
    [page.deferredSearchQuery, triggers]
  );
  const effectiveSelectedTriggerId = useMemo(
    () => resolveSelectedId(page.selectedId, visibleTriggers),
    [page.selectedId, visibleTriggers]
  );

  const triggerDetailQuery = useAutomationTrigger(effectiveSelectedTriggerId ?? "", {
    enabled: Boolean(effectiveSelectedTriggerId),
  });
  const triggerRunsQuery = useAutomationTriggerRuns(
    effectiveSelectedTriggerId ?? "",
    { limit: 10 },
    { enabled: Boolean(effectiveSelectedTriggerId) }
  );

  const createTriggerMutation = useCreateAutomationTrigger();
  const updateTriggerMutation = useUpdateAutomationTrigger();
  const deleteTriggerMutation = useDeleteAutomationTrigger();

  const selectedTrigger =
    triggerDetailQuery.data ??
    visibleTriggers.find(trigger => trigger.id === effectiveSelectedTriggerId) ??
    triggers.find(trigger => trigger.id === effectiveSelectedTriggerId);

  const handleScopeChange = (nextScope: AutomationScopeFilter) => {
    startTransition(() => {
      page.setScopeFilter(nextScope);
      page.setSelectedId(null);
      setEditor(null);
    });
  };

  const handleCreate = () => {
    setEditor({
      draft: createAutomationTriggerDraft(page.activeWorkspaceId),
      mode: "create",
    });
  };

  const handleEdit = () => {
    if (!selectedTrigger) {
      return;
    }

    setEditor({
      draft: automationTriggerToDraft(selectedTrigger),
      id: selectedTrigger.id,
      mode: "edit",
    });
  };

  const handleSubmit = async () => {
    if (!editor) {
      return;
    }

    try {
      const payload = {
        ...editor.draft,
        retry: normalizeAutomationRetry(editor.draft.retry ?? undefined),
      };
      const trigger =
        editor.mode === "create"
          ? await createTriggerMutation.mutateAsync(payload)
          : await updateTriggerMutation.mutateAsync({
              data: payload,
              id: editor.id,
            });

      page.setSelectedId(trigger.id);
      setEditor(null);
      toast.success(
        editor.mode === "create"
          ? `Created trigger ${trigger.name}.`
          : `Updated trigger ${trigger.name}.`
      );
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to save automation trigger");
    }
  };

  const handleDelete = async () => {
    if (!selectedTrigger) {
      return;
    }

    try {
      await deleteTriggerMutation.mutateAsync({ id: selectedTrigger.id });
      page.setSelectedId(null);
      toast.success(`Deleted ${selectedTrigger.name}.`);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to delete automation trigger");
    }
  };

  const handleToggleEnabled = async (enabled: boolean) => {
    if (!selectedTrigger) {
      return;
    }

    try {
      await updateTriggerMutation.mutateAsync({
        data: { enabled },
        id: selectedTrigger.id,
      });
      toast.success(`${enabled ? "Enabled" : "Disabled"} ${selectedTrigger.name}.`);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to update automation state");
    }
  };

  const emptyState =
    visibleTriggers.length === 0
      ? buildEmptyState({
          hasQuery: page.deferredSearchQuery.trim() !== "",
          kind: "triggers",
          onCreate: handleCreate,
        })
      : null;

  const listPanelProps = {
    activeWorkspaceName: page.activeWorkspace?.name,
    errorMessage: triggersQuery.error?.message ?? null,
    isLoading: triggersQuery.isLoading,
    jobs: [],
    kind: "triggers" as const,
    onSearchChange: page.setSearchQuery,
    onSelect: (id: string) =>
      startTransition(() => {
        page.setSelectedId(id);
      }),
    scopeFilter: page.scopeFilter,
    searchQuery: page.searchQuery,
    selectedId: effectiveSelectedTriggerId,
    totalCount: triggers.length,
    triggers: visibleTriggers,
  };

  const detailPanelProps = {
    emptyState,
    error: triggerDetailQuery.error,
    state: {
      isDeleting: deleteTriggerMutation.isPending,
      isLoading: triggerDetailQuery.isLoading,
      isTogglePending: updateTriggerMutation.isPending,
      isTriggerPending: false,
    },
    item: selectedTrigger,
    kind: "triggers" as const,
    onDelete: () => {
      void handleDelete();
    },
    onEdit: handleEdit,
    onToggleEnabled: (enabled: boolean) => {
      void handleToggleEnabled(enabled);
    },
    runs: triggerRunsQuery.data ?? [],
    runsError: triggerRunsQuery.error,
    runsLoading: triggerRunsQuery.isLoading,
  };

  const editorDialogProps = {
    activeWorkspaceId: page.activeWorkspaceId,
    handle: editorHandle,
    editor: editor
      ? {
          ...editor,
          kind: "triggers" as const,
          isPending: createTriggerMutation.isPending || updateTriggerMutation.isPending,
          onCancel: () => setEditor(null),
          onChange: (draft: CreateAutomationTriggerRequest) =>
            setEditor(current => (current ? { ...current, draft } : current)),
          onSubmit: () => {
            void handleSubmit();
          },
        }
      : null,
  };

  return {
    currentTotalCount: triggers.length,
    detailPanelProps,
    editorDialogProps,
    handleCreate,
    handleScopeChange,
    initialError: triggersQuery.error && triggers.length === 0 ? triggersQuery.error : null,
    isInitialLoading: triggersQuery.isLoading && triggers.length === 0,
    listPanelProps,
    scopeFilter: page.scopeFilter,
  };
}

export { useAutomationJobsPage, useAutomationTriggersPage };
