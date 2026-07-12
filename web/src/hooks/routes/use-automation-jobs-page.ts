import { startTransition, useEffect, useRef, useState } from "react";
import { toast } from "sonner";

import {
  automationJobToDraft,
  createAutomationJobDraft,
  createAutomationDialogHandle,
  createLoopTargetJobDraft,
  normalizeAutomationRetry,
  useAutomationJob,
  useAutomationJobRuns,
  useAutomationJobs,
  useCreateAutomationJob,
  useDeleteAutomationJob,
  useTriggerAutomationJob,
  useUpdateAutomationJob,
} from "@/systems/automation";
import type {
  AutomationRun,
  AutomationScopeFilter,
  CreateAutomationJobRequest,
} from "@/systems/automation";

import {
  automationUnavailableMessage,
  buildEmptyState,
  normalizeAutomationSchedule,
  resolveSelectedId,
  useAutomationPageBase,
  type AutomationCreateSeed,
  type AutomationRouteSearch,
  type JobEditorState,
} from "./use-automation-page-base";

export function useAutomationJobsPage(
  seed: AutomationCreateSeed = {},
  search: AutomationRouteSearch = {}
) {
  const page = useAutomationPageBase("jobs", search);
  const [editor, setEditor] = useState<JobEditorState | null>(null);
  const [queuedRun, setQueuedRun] = useState<{ jobId: string; run: AutomationRun } | null>(null);
  const jobSubmitInFlightRef = useRef(false);
  const seededRef = useRef(false);
  const [editorHandle] = useState(createAutomationDialogHandle);

  const jobsQuery = useAutomationJobs(page.listFilters);
  const jobs = jobsQuery.jobs;
  const runtimeUnavailableMessage = automationUnavailableMessage(
    "jobs",
    page.automationRuntime,
    jobsQuery.error
  );
  const effectiveSelectedJobId = resolveSelectedId(page.selectedId, jobs);

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

  const selectedJob = jobDetailQuery.data ?? jobs.find(job => job.id === effectiveSelectedJobId);

  const persistedRuns = jobRunsQuery.data ?? [];
  const displayedRuns =
    queuedRun &&
    queuedRun.jobId === effectiveSelectedJobId &&
    !persistedRuns.some(run => run.id === queuedRun.run.id)
      ? [queuedRun.run, ...persistedRuns]
      : persistedRuns;

  const handleScopeChange = (nextScope: AutomationScopeFilter) => {
    startTransition(() => {
      page.setScopeFilter(nextScope);
      page.setSelectedId(null);
      setEditor(null);
      setQueuedRun(null);
    });
  };

  const handleCreate = () => {
    setEditor({ draft: createAutomationJobDraft(page.activeWorkspaceId), mode: "create" });
  };

  // Open the create sheet pre-targeted at a Loop when arriving from the detail CTA.
  useEffect(() => {
    if (seededRef.current || !seed.loop || !page.activeWorkspaceId) return;
    seededRef.current = true;
    setEditor({
      draft: createLoopTargetJobDraft(page.activeWorkspaceId, seed.loop),
      mode: "create",
    });
  }, [seed.loop, page.activeWorkspaceId]);

  const handleEdit = () => {
    if (!selectedJob) {
      return;
    }

    setEditor({ draft: automationJobToDraft(selectedJob), id: selectedJob.id, mode: "edit" });
  };

  const handleSubmit = async () => {
    if (!editor || jobSubmitInFlightRef.current) {
      return;
    }

    jobSubmitInFlightRef.current = true;
    try {
      const payload = {
        ...editor.draft,
        retry: normalizeAutomationRetry(editor.draft.retry ?? undefined),
        schedule: normalizeAutomationSchedule(editor.draft.schedule),
      };
      const job =
        editor.mode === "create"
          ? await createJobMutation.mutateAsync(payload)
          : await updateJobMutation.mutateAsync({ data: payload, id: editor.id });

      page.setSelectedId(job.id);
      setEditor(null);
      toast.success(
        editor.mode === "create" ? `Created job ${job.name}.` : `Updated job ${job.name}.`
      );
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to save automation job");
    } finally {
      jobSubmitInFlightRef.current = false;
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
      await updateJobMutation.mutateAsync({ data: { enabled }, id: selectedJob.id });
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
    jobs.length === 0
      ? buildEmptyState({
          hasQuery: page.searchQuery.trim() !== "",
          kind: "jobs",
          onCreate: handleCreate,
        })
      : null;

  const listPanelProps = {
    activeWorkspaceName: page.activeWorkspace?.name,
    errorMessage: jobsQuery.error?.message ?? null,
    isLoading: jobsQuery.isLoading,
    hasNextPage: jobsQuery.hasNextPage,
    isFetchingNextPage: jobsQuery.isFetchingNextPage,
    jobs,
    kind: "jobs" as const,
    onSearchChange: page.setSearchQuery,
    onLoadMore: () => void jobsQuery.fetchNextPage(),
    onSelect: (id: string) =>
      startTransition(() => {
        page.setSelectedId(id);
        setQueuedRun(null);
      }),
    scopeFilter: page.scopeFilter,
    searchQuery: page.searchQuery,
    selectedId: effectiveSelectedJobId,
    totalCount: jobsQuery.total,
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
    workspaces: page.workspaces,
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
    currentTotalCount: jobsQuery.total,
    detailPanelProps,
    editorDialogProps,
    handleCreate,
    handleScopeChange,
    initialError:
      runtimeUnavailableMessage && jobs.length === 0
        ? new Error(runtimeUnavailableMessage)
        : jobsQuery.error && jobs.length === 0
          ? jobsQuery.error
          : null,
    isInitialLoading: jobsQuery.isLoading && jobs.length === 0,
    listPanelProps,
    runtimeUnavailableMessage,
    scopeFilter: page.scopeFilter,
  };
}
