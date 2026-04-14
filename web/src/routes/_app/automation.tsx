import { AlertCircle, Loader2, Plus, Zap } from "lucide-react";
import { startTransition, useDeferredValue, useMemo, useState } from "react";
import { createFileRoute } from "@tanstack/react-router";
import { toast } from "sonner";

import { PillButton } from "@/components/design-system";
import { Button } from "@/components/ui/button";
import {
  AutomationDetailPanel,
  AutomationEditorDialog,
  AutomationListPanel,
  automationJobToDraft,
  automationTriggerToDraft,
  createAutomationJobDraft,
  createAutomationTriggerDraft,
  filterAutomationJobs,
  filterAutomationTriggers,
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

export const Route = createFileRoute("/_app/automation")({
  component: AutomationPage,
});

type AutomationTab = "jobs" | "triggers";

type AutomationEditorState =
  | {
      draft: CreateAutomationJobRequest;
      kind: "jobs";
      mode: "create" | "edit";
    }
  | {
      draft: CreateAutomationTriggerRequest;
      kind: "triggers";
      mode: "create" | "edit";
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

function AutomationPage() {
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
    enabled: activeTab === "jobs" && !!effectiveSelectedJobId,
  });
  const triggerDetailQuery = useAutomationTrigger(effectiveSelectedTriggerId ?? "", {
    enabled: activeTab === "triggers" && !!effectiveSelectedTriggerId,
  });

  const jobRunsQuery = useAutomationJobRuns(
    effectiveSelectedJobId ?? "",
    { limit: 10 },
    { enabled: activeTab === "jobs" && !!effectiveSelectedJobId }
  );
  const triggerRunsQuery = useAutomationTriggerRuns(
    effectiveSelectedTriggerId ?? "",
    { limit: 10 },
    { enabled: activeTab === "triggers" && !!effectiveSelectedTriggerId }
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
            kind: "jobs",
            mode: "edit",
          }
        : selectedTrigger
          ? {
              draft: automationTriggerToDraft(selectedTrigger),
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
      const job =
        editor.mode === "create"
          ? await createJobMutation.mutateAsync(editor.draft)
          : await updateJobMutation.mutateAsync({
              data: editor.draft,
              id: effectiveSelectedJobId ?? "",
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
      const trigger =
        editor.mode === "create"
          ? await createTriggerMutation.mutateAsync(editor.draft)
          : await updateTriggerMutation.mutateAsync({
              data: editor.draft,
              id: effectiveSelectedTriggerId ?? "",
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

  if (currentListLoading && currentTotalCount === 0) {
    return (
      <div className="flex flex-1 items-center justify-center" data-testid="automation-loading">
        <Loader2 className="size-5 animate-spin text-[color:var(--color-text-tertiary)]" />
      </div>
    );
  }

  if (currentListError && currentTotalCount === 0) {
    return (
      <div className="flex flex-1 items-center justify-center" data-testid="automation-error">
        <div className="flex flex-col items-center gap-2 text-center">
          <AlertCircle className="size-6 text-[color:var(--color-danger)]" />
          <p className="text-sm text-[color:var(--color-text-tertiary)]">
            {currentListError.message ?? "Failed to load automation"}
          </p>
        </div>
      </div>
    );
  }

  const hasVisibleSearchQuery = deferredSearchQuery.trim() !== "";
  const emptyState =
    currentList.length === 0
      ? buildEmptyState({
          activeTab,
          hasQuery: hasVisibleSearchQuery,
          onCreate: handleCreate,
        })
      : null;

  return (
    <div className="flex flex-1 flex-col overflow-hidden">
      <header className="flex flex-wrap items-center gap-3 border-b border-[color:var(--color-divider)] px-4 py-3">
        <div className="flex items-center gap-2">
          <Zap className="size-4 text-[color:var(--color-text-primary)]" />
          <h1 className="text-xl font-semibold tracking-[-0.02em] text-[color:var(--color-text-primary)]">
            Automation
          </h1>
          <span className="inline-flex h-5 items-center rounded-md bg-[color:var(--color-surface-panel)] px-1.5 font-mono text-[0.64rem] text-[color:var(--color-text-secondary)]">
            {currentTotalCount}
          </span>
        </div>

        <div className="flex flex-wrap items-center gap-3">
          <div className="flex items-center gap-1.5" data-testid="automation-kind-tabs">
            <PillButton
              active={activeTab === "jobs"}
              data-testid="automation-kind-jobs"
              onClick={() => handleTabChange("jobs")}
            >
              JOBS
            </PillButton>
            <PillButton
              active={activeTab === "triggers"}
              data-testid="automation-kind-triggers"
              onClick={() => handleTabChange("triggers")}
            >
              TRIGGERS
            </PillButton>
          </div>

          <div className="flex items-center gap-1.5" data-testid="automation-scope-tabs">
            {(["all", "global", "workspace"] as const).map(scope => (
              <PillButton
                active={scopeFilter === scope}
                data-testid={`automation-scope-${scope}`}
                key={scope}
                onClick={() => handleScopeChange(scope)}
              >
                {scope.toUpperCase()}
              </PillButton>
            ))}
          </div>
        </div>

        <div className="ml-auto flex items-center gap-2">
          <Button
            className="border-[color:var(--color-divider)] bg-transparent text-[color:var(--color-text-primary)] hover:bg-[color:var(--color-hover)]"
            data-testid="create-automation-btn"
            onClick={handleCreate}
            size="lg"
            type="button"
            variant="outline"
          >
            <Plus className="size-4" />
            {activeTab === "jobs" ? "Job" : "Trigger"}
          </Button>
        </div>
      </header>

      <div className="flex min-h-0 flex-1 overflow-hidden">
        <AutomationListPanel
          activeWorkspaceName={activeWorkspace?.name}
          jobs={visibleJobs}
          kind={activeTab}
          onSearchChange={setSearchQuery}
          onSelect={id =>
            startTransition(() => {
              if (activeTab === "jobs") {
                setSelectedJobId(id);
                setQueuedRun(null);
              } else {
                setSelectedTriggerId(id);
              }
            })
          }
          scopeFilter={scopeFilter}
          searchQuery={searchQuery}
          selectedId={activeTab === "jobs" ? effectiveSelectedJobId : effectiveSelectedTriggerId}
          totalCount={currentTotalCount}
          triggers={visibleTriggers}
        />
        <AutomationDetailPanel
          emptyState={emptyState}
          error={activeTab === "jobs" ? jobDetailQuery.error : triggerDetailQuery.error}
          isDeleting={deleteJobMutation.isPending || deleteTriggerMutation.isPending}
          isLoading={activeTab === "jobs" ? jobDetailQuery.isLoading : triggerDetailQuery.isLoading}
          isTogglePending={updateJobMutation.isPending || updateTriggerMutation.isPending}
          isTriggerPending={triggerJobMutation.isPending}
          item={selectedItem}
          kind={activeTab}
          onDelete={() => {
            void handleDelete();
          }}
          onEdit={handleEdit}
          onToggleEnabled={enabled => {
            void handleToggleEnabled(enabled);
          }}
          onTriggerNow={() => {
            void handleTriggerNow();
          }}
          runs={displayedRuns}
          runsError={runsError}
          runsLoading={runsLoading}
        />
      </div>

      <AutomationEditorDialog
        activeWorkspaceId={activeWorkspaceId}
        editor={
          editor
            ? editor.kind === "jobs"
              ? {
                  ...editor,
                  isPending: createJobMutation.isPending || updateJobMutation.isPending,
                  onCancel: () => setEditor(null),
                  onChange: (draft: CreateAutomationJobRequest) =>
                    setEditor(current =>
                      current?.kind === "jobs" ? { ...current, draft } : current
                    ),
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
            : null
        }
      />
    </div>
  );
}
