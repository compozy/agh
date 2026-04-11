import { AlertCircle, Bot, Loader2 } from "lucide-react";
import { startTransition, useMemo, useState } from "react";
import { createFileRoute } from "@tanstack/react-router";

import { PillButton } from "@/components/design-system";
import {
  AutomationDetailPanel,
  AutomationListPanel,
  automationJobToDraft,
  automationTriggerToDraft,
  createAutomationJobDraft,
  createAutomationTriggerDraft,
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
  AutomationTrigger,
  AutomationRun,
  AutomationScopeFilter,
  CreateAutomationJobRequest,
  CreateAutomationTriggerRequest,
} from "@/systems/automation";
import { useActiveWorkspace } from "@/systems/workspace";
import { WorkspacePageShell } from "@/systems/workspace/components/workspace-page-shell";

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

function AutomationPage() {
  const { activeWorkspace, activeWorkspaceId } = useActiveWorkspace();

  const [activeTab, setActiveTab] = useState<AutomationTab>("jobs");
  const [scopeFilter, setScopeFilter] = useState<AutomationScopeFilter>("all");
  const [selectedJobId, setSelectedJobId] = useState<string | null>(null);
  const [selectedTriggerId, setSelectedTriggerId] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState("");
  const [editor, setEditor] = useState<AutomationEditorState | null>(null);
  const [actionMessage, setActionMessage] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const [queuedRun, setQueuedRun] = useState<{ jobId: string; run: AutomationRun } | null>(null);

  const scopedWorkspaceId =
    scopeFilter === "workspace" ? (activeWorkspaceId ?? undefined) : undefined;
  const listFilters = useMemo(
    () => ({
      scope: scopeFilter === "all" ? undefined : scopeFilter,
      workspace_id: scopedWorkspaceId,
      limit: 50,
    }),
    [scopeFilter, scopedWorkspaceId]
  );

  const jobsQuery = useAutomationJobs(listFilters);
  const triggersQuery = useAutomationTriggers(listFilters);

  const jobs = jobsQuery.data ?? [];
  const triggers = triggersQuery.data ?? [];
  const currentList = activeTab === "jobs" ? jobs : triggers;
  const currentListLoading = activeTab === "jobs" ? jobsQuery.isLoading : triggersQuery.isLoading;
  const currentListError = activeTab === "jobs" ? jobsQuery.error : triggersQuery.error;

  const effectiveSelectedJobId = useMemo(() => {
    if (selectedJobId && jobs.some(job => job.id === selectedJobId)) {
      return selectedJobId;
    }
    return jobs[0]?.id ?? null;
  }, [jobs, selectedJobId]);

  const effectiveSelectedTriggerId = useMemo(() => {
    if (selectedTriggerId && triggers.some(trigger => trigger.id === selectedTriggerId)) {
      return selectedTriggerId;
    }
    return triggers[0]?.id ?? null;
  }, [selectedTriggerId, triggers]);

  const jobDetailQuery = useAutomationJob(effectiveSelectedJobId ?? "", {
    enabled: activeTab === "jobs" && editor === null && !!effectiveSelectedJobId,
  });
  const triggerDetailQuery = useAutomationTrigger(effectiveSelectedTriggerId ?? "", {
    enabled: activeTab === "triggers" && editor === null && !!effectiveSelectedTriggerId,
  });

  const jobRunsQuery = useAutomationJobRuns(
    effectiveSelectedJobId ?? "",
    { limit: 10 },
    { enabled: activeTab === "jobs" && editor === null && !!effectiveSelectedJobId }
  );
  const triggerRunsQuery = useAutomationTriggerRuns(
    effectiveSelectedTriggerId ?? "",
    { limit: 10 },
    { enabled: activeTab === "triggers" && editor === null && !!effectiveSelectedTriggerId }
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
      ? (jobDetailQuery.data ?? jobs.find(job => job.id === effectiveSelectedJobId))
      : (triggerDetailQuery.data ??
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
      setActionMessage(null);
      setActionError(null);
    });
  };

  const handleScopeChange = (nextScope: AutomationScopeFilter) => {
    startTransition(() => {
      setScopeFilter(nextScope);
      setEditor(null);
      setSelectedJobId(null);
      setSelectedTriggerId(null);
      setActionMessage(null);
      setActionError(null);
    });
  };

  const clearFeedback = () => {
    setActionMessage(null);
    setActionError(null);
  };

  const handleCreate = () => {
    clearFeedback();
    setEditor(
      activeTab === "jobs"
        ? {
            kind: "jobs",
            mode: "create",
            draft: createAutomationJobDraft(activeWorkspaceId),
          }
        : {
            kind: "triggers",
            mode: "create",
            draft: createAutomationTriggerDraft(activeWorkspaceId),
          }
    );
  };

  const handleEdit = () => {
    if (!selectedItem) {
      return;
    }

    clearFeedback();
    setEditor(
      activeTab === "jobs" && selectedJob
        ? {
            kind: "jobs",
            mode: "edit",
            draft: automationJobToDraft(selectedJob),
          }
        : selectedTrigger
          ? {
              kind: "triggers",
              mode: "edit",
              draft: automationTriggerToDraft(selectedTrigger),
            }
          : null
    );
  };

  const handleSubmitJob = async () => {
    if (!editor || editor.kind !== "jobs") {
      return;
    }

    clearFeedback();

    try {
      const job =
        editor.mode === "create"
          ? await createJobMutation.mutateAsync(editor.draft)
          : await updateJobMutation.mutateAsync({
              id: effectiveSelectedJobId ?? "",
              data: editor.draft,
            });

      setSelectedJobId(job.id);
      setEditor(null);
      setActionMessage(
        editor.mode === "create" ? `Created job ${job.name}.` : `Updated job ${job.name}.`
      );
    } catch (error) {
      setActionError(error instanceof Error ? error.message : "Failed to save automation job");
    }
  };

  const handleSubmitTrigger = async () => {
    if (!editor || editor.kind !== "triggers") {
      return;
    }

    clearFeedback();

    try {
      const trigger =
        editor.mode === "create"
          ? await createTriggerMutation.mutateAsync(editor.draft)
          : await updateTriggerMutation.mutateAsync({
              id: effectiveSelectedTriggerId ?? "",
              data: editor.draft,
            });

      setSelectedTriggerId(trigger.id);
      setEditor(null);
      setActionMessage(
        editor.mode === "create"
          ? `Created trigger ${trigger.name}.`
          : `Updated trigger ${trigger.name}.`
      );
    } catch (error) {
      setActionError(error instanceof Error ? error.message : "Failed to save automation trigger");
    }
  };

  const handleDelete = async () => {
    if (!selectedItem) {
      return;
    }

    clearFeedback();

    try {
      if (activeTab === "jobs") {
        await deleteJobMutation.mutateAsync({ id: selectedItem.id });
        setSelectedJobId(null);
        setQueuedRun(null);
      } else {
        await deleteTriggerMutation.mutateAsync({ id: selectedItem.id });
        setSelectedTriggerId(null);
      }

      setActionMessage(`Deleted ${selectedItem.name}.`);
    } catch (error) {
      setActionError(error instanceof Error ? error.message : "Failed to delete automation");
    }
  };

  const handleToggleEnabled = async (enabled: boolean) => {
    if (!selectedItem) {
      return;
    }

    clearFeedback();

    try {
      if (activeTab === "jobs") {
        await updateJobMutation.mutateAsync({
          id: selectedItem.id,
          data: { enabled },
        });
      } else {
        await updateTriggerMutation.mutateAsync({
          id: selectedItem.id,
          data: { enabled },
        });
      }

      setActionMessage(`${enabled ? "Enabled" : "Disabled"} ${selectedItem.name}.`);
    } catch (error) {
      setActionError(error instanceof Error ? error.message : "Failed to update automation state");
    }
  };

  const handleTriggerNow = async () => {
    if (activeTab !== "jobs" || !selectedItem) {
      return;
    }

    clearFeedback();

    try {
      const run = await triggerJobMutation.mutateAsync({ id: selectedItem.id });
      setQueuedRun({ jobId: selectedItem.id, run });
      setActionMessage(`Queued run ${run.id}.`);
    } catch (error) {
      setActionError(error instanceof Error ? error.message : "Failed to trigger automation job");
    }
  };

  if (currentListLoading && currentList.length === 0) {
    return (
      <div className="flex flex-1 items-center justify-center" data-testid="automation-loading">
        <Loader2 className="size-5 animate-spin text-[color:var(--color-text-tertiary)]" />
      </div>
    );
  }

  if (currentListError && currentList.length === 0) {
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

  return (
    <WorkspacePageShell
      count={currentList.length}
      controls={
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
                key={scope}
                active={scopeFilter === scope}
                data-testid={`automation-scope-${scope}`}
                onClick={() => handleScopeChange(scope)}
              >
                {scope.toUpperCase()}
              </PillButton>
            ))}
          </div>
        </div>
      }
      icon={<Bot className="size-4" />}
      meta={
        <div className="flex flex-col items-end gap-1" data-testid="automation-meta">
          <span className="text-xs text-[color:var(--color-text-tertiary)]">
            {scopeFilter === "workspace" && activeWorkspace
              ? `Workspace ${activeWorkspace.name}`
              : "Unified jobs and triggers"}
          </span>
          {actionMessage ? (
            <span className="text-xs text-[color:var(--color-success)]">{actionMessage}</span>
          ) : null}
          {actionError ? (
            <span className="text-xs text-[color:var(--color-danger)]">{actionError}</span>
          ) : null}
        </div>
      }
      title="Automation"
    >
      <AutomationListPanel
        jobs={jobs}
        kind={activeTab}
        onCreate={handleCreate}
        onSearchChange={setSearchQuery}
        onSelect={id =>
          startTransition(() => {
            if (activeTab === "jobs") {
              setSelectedJobId(id);
              setQueuedRun(null);
            } else {
              setSelectedTriggerId(id);
            }
            clearFeedback();
          })
        }
        scopeFilter={scopeFilter}
        searchQuery={searchQuery}
        selectedId={activeTab === "jobs" ? effectiveSelectedJobId : effectiveSelectedTriggerId}
        triggers={triggers}
      />
      <AutomationDetailPanel
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
    </WorkspacePageShell>
  );
}
