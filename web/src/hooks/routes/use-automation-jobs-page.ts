import { useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import { toast } from "sonner";

import {
  useAutomationJobEditor,
  useAutomationJobs,
  useTriggerAutomationJob,
} from "@/systems/automation";

import {
  automationListError,
  automationUnavailableMessage,
  useAutomationCreateSeed,
  useAutomationPageBase,
  type AutomationCreateSeed,
  type AutomationRouteSearch,
} from "./use-automation-page-base";

export function useAutomationJobsPage(
  seed: AutomationCreateSeed = {},
  search: AutomationRouteSearch = {}
) {
  const page = useAutomationPageBase("jobs", search);
  const navigate = useNavigate();
  const [runPendingId, setRunPendingId] = useState<string | null>(null);

  const jobsQuery = useAutomationJobs(page.listFilters);
  const jobs = jobsQuery.jobs;
  const runtimeUnavailableMessage = automationUnavailableMessage(
    "jobs",
    page.automationRuntime,
    jobsQuery.error
  );

  const editor = useAutomationJobEditor({
    activeWorkspaceId: page.activeWorkspaceId,
    workspaces: page.workspaces,
    onSaved: job => void navigate({ to: "/jobs/$jobId", params: { jobId: job.id } }),
  });

  useAutomationCreateSeed("jobs", seed, page.activeWorkspaceId, editor.openLoopCreate);

  const triggerJobMutation = useTriggerAutomationJob();
  const onRunJob = async (id: string) => {
    setRunPendingId(id);
    try {
      const run = await triggerJobMutation.mutateAsync({ id });
      toast.success(`Queued run ${run.id}.`);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to trigger automation job");
    }
    setRunPendingId(null);
  };

  return {
    clearFilters: page.clearFilters,
    editorDialogProps: editor.editorDialogProps,
    enabledFilter: page.enabledFilter,
    error: automationListError(runtimeUnavailableMessage, jobsQuery.error, jobs.length),
    handleCreate: editor.openCreate,
    hasActiveFilters: page.hasActiveFilters,
    hasChildMatch: page.hasChildMatch,
    hasNextPage: jobsQuery.hasNextPage,
    isFetchingNextPage: jobsQuery.isFetchingNextPage,
    isLoading: jobsQuery.isLoading && jobs.length === 0,
    jobs,
    loadMore: () => void jobsQuery.fetchNextPage(),
    onRunJob: (id: string) => void onRunJob(id),
    runPendingId,
    runtimeUnavailableMessage,
    scopeFilter: page.scopeFilter,
    searchQuery: page.searchQuery,
    setEnabledFilter: page.setEnabledFilter,
    setScopeFilter: page.setScopeFilter,
    setSearchQuery: page.setSearchQuery,
    setSourceFilter: page.setSourceFilter,
    setView: page.setView,
    sourceFilter: page.sourceFilter,
    total: jobsQuery.total,
    view: page.view,
  };
}
