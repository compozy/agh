import { startTransition, useDeferredValue, useMemo, useState } from "react";
import { toast } from "sonner";

import {
  automationJobToDraft,
  automationTriggerToDraft,
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
  AutomationJob,
  AutomationRun,
  AutomationScopeFilter,
  AutomationTrigger,
  CreateAutomationJobRequest,
  CreateAutomationTriggerRequest,
} from "@/systems/automation";
import { useActiveWorkspace } from "@/systems/workspace";

type AutomationTab = "jobs" | "triggers";

type AutomationEditorState =
  | {
      draft: CreateAutomationJobRequest;
      kind: "jobs";
      mode: "create";
    }
  | {
      draft: CreateAutomationTriggerRequest;
      kind: "triggers";
      mode: "create";
    }
  | {
      draft: CreateAutomationJobRequest;
      id: string;
      kind: "jobs";
      mode: "edit";
    }
  | {
      draft: CreateAutomationTriggerRequest;
      id: string;
      kind: "triggers";
      mode: "edit";
    };

function buildEmptyState({
  activeTab,
  hasQuery,
  onCreate,
}: {
  activeTab: AutomationTab;
  hasQuery: boolean;
  onCreate: () => void;
}) {
  if (hasQuery) {
    return {
      description: "Try a different search term or adjust the current scope filter.",
      icon: "search" as const,
      title: activeTab === "jobs" ? "No jobs found" : "No triggers found",
    };
  }

  if (activeTab === "jobs") {
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

function useAutomationPage() {
  const { activeWorkspace, activeWorkspaceId } = useActiveWorkspace();

  const [activeTab, setActiveTab] = useState<AutomationTab>("jobs");
  const [scopeFilter, setScopeFilter] = useState<AutomationScopeFilter>("all");
  const [selectedJobId, setSelectedJobId] = useState<string | null>(null);
  const [selectedTriggerId, setSelectedTriggerId] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState("");
  const [editor, setEditor] = useState<AutomationEditorState | null>(null);
  const [queuedRun, setQueuedRun] = useState<{ jobId: string; run: AutomationRun } | null>(null);

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

  const jobsQuery = useAutomationJobs(listFilters);
  const triggersQuery = useAutomationTriggers(listFilters);

  const jobs = jobsQuery.data ?? [];
  const triggers = triggersQuery.data ?? [];

  const visibleJobs = useMemo(
    () => sortAutomationJobs(filterAutomationJobs(jobs, deferredSearchQuery)),
    [deferredSearchQuery, jobs]
  );
  const visibleTriggers = useMemo(
    () => sortAutomationTriggers(filterAutomationTriggers(triggers, deferredSearchQuery)),
    [deferredSearchQuery, triggers]
  );

  const currentList = activeTab === "jobs" ? visibleJobs : visibleTriggers;
  const currentTotalCount = activeTab === "jobs" ? jobs.length : triggers.length;
  const currentListLoading = activeTab === "jobs" ? jobsQuery.isLoading : triggersQuery.isLoading;
  const currentListError = activeTab === "jobs" ? jobsQuery.error : triggersQuery.error;

  const effectiveSelectedJobId = useMemo(() => {
    if (selectedJobId && visibleJobs.some(job => job.id === selectedJobId)) {
      return selectedJobId;
    }

    return visibleJobs[0]?.id ?? null;
  }, [selectedJobId, visibleJobs]);

  const effectiveSelectedTriggerId = useMemo(() => {
    if (selectedTriggerId && visibleTriggers.some(trigger => trigger.id === selectedTriggerId)) {
      return selectedTriggerId;
    }

    return visibleTriggers[0]?.id ?? null;
  }, [selectedTriggerId, visibleTriggers]);

  const jobDetailQuery = useAutomationJob(effectiveSelectedJobId ?? "", {
    enabled: activeTab === "jobs" && Boolean(effectiveSelectedJobId),
  });
  const triggerDetailQuery = useAutomationTrigger(effectiveSelectedTriggerId ?? "", {
    enabled: activeTab === "triggers" && Boolean(effectiveSelectedTriggerId),
  });

  const jobRunsQuery = useAutomationJobRuns(
    effectiveSelectedJobId ?? "",
    { limit: 10 },
    { enabled: activeTab === "jobs" && Boolean(effectiveSelectedJobId) }
  );
  const triggerRunsQuery = useAutomationTriggerRuns(
    effectiveSelectedTriggerId ?? "",
    { limit: 10 },
    { enabled: activeTab === "triggers" && Boolean(effectiveSelectedTriggerId) }
  );

  const createJobMutation = useCreateAutomationJob();
  const updateJobMutation = useUpdateAutomationJob();
  const deleteJobMutation = useDeleteAutomationJob();
  const triggerJobMutation = useTriggerAutomationJob();
  const createTriggerMutation = useCreateAutomationTrigger();
  const updateTriggerMutation = useUpdateAutomationTrigger();
  const deleteTriggerMutation = useDeleteAutomationTrigger();

  const selectedItem =
    activeTab === "jobs"
      ? (jobDetailQuery.data ??
        visibleJobs.find(job => job.id === effectiveSelectedJobId) ??
        jobs.find(job => job.id === effectiveSelectedJobId))
      : (triggerDetailQuery.data ??
        visibleTriggers.find(trigger => trigger.id === effectiveSelectedTriggerId) ??
        triggers.find(trigger => trigger.id === effectiveSelectedTriggerId));

  const selectedJob =
    activeTab === "jobs" ? (selectedItem as AutomationJob | undefined) : undefined;
  const selectedTrigger =
    activeTab === "triggers" ? (selectedItem as AutomationTrigger | undefined) : undefined;

  const displayedRuns = useMemo(() => {
    if (activeTab === "jobs") {
      const baseRuns = jobRunsQuery.data ?? [];
      if (
        queuedRun &&
        queuedRun.jobId === effectiveSelectedJobId &&
        !baseRuns.some(run => run.id === queuedRun.run.id)
      ) {
        return [queuedRun.run, ...baseRuns];
      }

      return baseRuns;
    }

    return triggerRunsQuery.data ?? [];
  }, [activeTab, effectiveSelectedJobId, jobRunsQuery.data, queuedRun, triggerRunsQuery.data]);

  const runsLoading = activeTab === "jobs" ? jobRunsQuery.isLoading : triggerRunsQuery.isLoading;
  const runsError = activeTab === "jobs" ? jobRunsQuery.error : triggerRunsQuery.error;

  const handleTabChange = (nextTab: AutomationTab) => {
    startTransition(() => {
      setActiveTab(nextTab);
      setEditor(null);
      setSearchQuery("");
      setQueuedRun(null);
    });
  };

  const handleScopeChange = (nextScope: AutomationScopeFilter) => {
    startTransition(() => {
      setScopeFilter(nextScope);
      setEditor(null);
      setSelectedJobId(null);
      setSelectedTriggerId(null);
      setQueuedRun(null);
    });
  };

  const handleCreate = () => {
    setEditor(
      activeTab === "jobs"
        ? {
            draft: createAutomationJobDraft(activeWorkspaceId),
            kind: "jobs",
            mode: "create",
          }
        : {
            draft: createAutomationTriggerDraft(activeWorkspaceId),
            kind: "triggers",
            mode: "create",
          }
    );
  };

  const handleEdit = () => {
    if (!selectedItem) {
      return;
    }

    setEditor(
      activeTab === "jobs" && selectedJob
        ? {
            draft: automationJobToDraft(selectedJob),
            id: selectedJob.id,
            kind: "jobs",
            mode: "edit",
          }
        : selectedTrigger
          ? {
              draft: automationTriggerToDraft(selectedTrigger),
              id: selectedTrigger.id,
              kind: "triggers",
              mode: "edit",
            }
          : null
    );
  };

  const handleSubmitJob = async () => {
    if (!editor || editor.kind !== "jobs") {
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

      setSelectedJobId(job.id);
      setEditor(null);
      toast.success(
        editor.mode === "create" ? `Created job ${job.name}.` : `Updated job ${job.name}.`
      );
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to save automation job");
    }
  };

  const handleSubmitTrigger = async () => {
    if (!editor || editor.kind !== "triggers") {
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

      setSelectedTriggerId(trigger.id);
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
    if (!selectedItem) {
      return;
    }

    try {
      if (activeTab === "jobs") {
        await deleteJobMutation.mutateAsync({ id: selectedItem.id });
        setSelectedJobId(null);
        setQueuedRun(null);
      } else {
        await deleteTriggerMutation.mutateAsync({ id: selectedItem.id });
        setSelectedTriggerId(null);
      }

      toast.success(`Deleted ${selectedItem.name}.`);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to delete automation");
    }
  };

  const handleToggleEnabled = async (enabled: boolean) => {
    if (!selectedItem) {
      return;
    }

    try {
      if (activeTab === "jobs") {
        await updateJobMutation.mutateAsync({
          data: { enabled },
          id: selectedItem.id,
        });
      } else {
        await updateTriggerMutation.mutateAsync({
          data: { enabled },
          id: selectedItem.id,
        });
      }

      toast.success(`${enabled ? "Enabled" : "Disabled"} ${selectedItem.name}.`);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to update automation state");
    }
  };

  const handleTriggerNow = async () => {
    if (activeTab !== "jobs" || !selectedItem) {
      return;
    }

    try {
      const run = await triggerJobMutation.mutateAsync({ id: selectedItem.id });
      setQueuedRun({ jobId: selectedItem.id, run });
      toast.success(`Queued run ${run.id}.`);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to trigger automation job");
    }
  };

  const hasVisibleSearchQuery = deferredSearchQuery.trim() !== "";
  const emptyState =
    currentList.length === 0
      ? buildEmptyState({
          activeTab,
          hasQuery: hasVisibleSearchQuery,
          onCreate: handleCreate,
        })
      : null;

  const listPanelProps = {
    activeWorkspaceName: activeWorkspace?.name,
    errorMessage: currentListError?.message ?? null,
    isLoading: currentListLoading,
    jobs: visibleJobs,
    kind: activeTab,
    onSearchChange: setSearchQuery,
    onSelect: (id: string) =>
      startTransition(() => {
        if (activeTab === "jobs") {
          setSelectedJobId(id);
          setQueuedRun(null);
        } else {
          setSelectedTriggerId(id);
        }
      }),
    scopeFilter,
    searchQuery,
    selectedId: activeTab === "jobs" ? effectiveSelectedJobId : effectiveSelectedTriggerId,
    totalCount: currentTotalCount,
    triggers: visibleTriggers,
  };

  const detailPanelProps = {
    emptyState,
    error: activeTab === "jobs" ? jobDetailQuery.error : triggerDetailQuery.error,
    isDeleting: deleteJobMutation.isPending || deleteTriggerMutation.isPending,
    isLoading: activeTab === "jobs" ? jobDetailQuery.isLoading : triggerDetailQuery.isLoading,
    isTogglePending: updateJobMutation.isPending || updateTriggerMutation.isPending,
    isTriggerPending: triggerJobMutation.isPending,
    item: selectedItem,
    kind: activeTab,
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
    runsError,
    runsLoading,
  };

  const editorDialogProps = {
    activeWorkspaceId,
    editor: editor
      ? editor.kind === "jobs"
        ? {
            ...editor,
            isPending: createJobMutation.isPending || updateJobMutation.isPending,
            onCancel: () => setEditor(null),
            onChange: (draft: CreateAutomationJobRequest) =>
              setEditor(current => (current?.kind === "jobs" ? { ...current, draft } : current)),
            onSubmit: () => {
              void handleSubmitJob();
            },
          }
        : {
            ...editor,
            isPending: createTriggerMutation.isPending || updateTriggerMutation.isPending,
            onCancel: () => setEditor(null),
            onChange: (draft: CreateAutomationTriggerRequest) =>
              setEditor(current =>
                current?.kind === "triggers" ? { ...current, draft } : current
              ),
            onSubmit: () => {
              void handleSubmitTrigger();
            },
          }
      : null,
  };

  return {
    activeTab,
    currentTotalCount,
    detailPanelProps,
    editorDialogProps,
    handleCreate,
    handleScopeChange,
    handleTabChange,
    initialError: currentListError && currentTotalCount === 0 ? currentListError : null,
    isInitialLoading: currentListLoading && currentTotalCount === 0,
    listPanelProps,
    scopeFilter,
  };
}

export { useAutomationPage };
