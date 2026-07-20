import { AlertCircle } from "lucide-react";
import { toast } from "sonner";

import { cn, PAGE_CONTENT_GUTTER, Spinner } from "@agh/ui";

import {
  TasksDetailChildrenPanel,
  TasksDetailDependenciesPanel,
  TasksDetailHeader,
  TasksDetailOrchestrationPanel,
  TasksDetailOverviewPanel,
  TasksDetailRunsPanel,
  TasksDetailTabs,
  TasksMultiAgentPanel,
  TasksTimelinePanel,
  useDeleteTask,
  useTaskDetailOrchestrationTab,
  useTaskDetailPage,
  type TasksDetailTabItem,
} from "@/systems/tasks";

import { useOsShell } from "../../hooks/use-os-shell";

export function TaskDetailLocation({ taskId }: { taskId: string }) {
  const { coordinator } = useOsShell();
  const page = useTaskDetailPage(taskId);
  const deleteMutation = useDeleteTask();
  const latestEventSeq = page.detail?.task?.latest_event_seq ?? null;
  const orchestration = useTaskDetailOrchestrationTab(taskId, {
    enabled: page.panel === "orchestration",
    latestEventSeq,
  });

  const handleDeleteTask = async (id: string) => {
    coordinator.userOpen({ app: "tasks", location: { pathname: "/tasks", search: {} } });
    try {
      await deleteMutation.mutateAsync({ id });
      toast.success("Task deleted.");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to delete task");
    }
  };

  if (page.detailLoading) {
    return (
      <div className="flex flex-1 items-center justify-center" data-testid="tasks-detail-loading">
        <Spinner className="size-5 text-subtle" />
      </div>
    );
  }

  if (page.notFound || (!page.detail && page.fatalError)) {
    return (
      <div
        className={cn(
          PAGE_CONTENT_GUTTER,
          "flex flex-1 flex-col items-center justify-center gap-2 py-10 text-center"
        )}
        data-testid="tasks-detail-not-found"
      >
        <AlertCircle className="size-6 text-danger" />
        <p className="text-sm text-muted">
          {page.fatalError?.message ?? `Task ${taskId} not found.`}
        </p>
      </div>
    );
  }

  const detail = page.detail;
  if (!detail) {
    return (
      <div
        className="flex flex-1 items-center justify-center"
        data-testid="tasks-detail-placeholder"
      >
        <Spinner className="size-5 text-subtle" />
      </div>
    );
  }

  const children = detail.children ?? [];
  const dependencies = detail.dependency_references ?? [];
  const tabItems: TasksDetailTabItem[] = [
    { id: "overview", label: "Overview" },
    { id: "runs", label: "Runs", count: page.runs.length },
    { id: "timeline", label: "Events", live: page.isLive },
    {
      id: "agents",
      label: "Agents",
      count: page.multiAgent.descendantCount,
      live: page.multiAgent.liveCount > 0,
    },
    { id: "children", label: "Children", count: children.length },
    { id: "dependencies", label: "Dependencies", count: dependencies.length },
    { id: "orchestration", label: "Orchestration" },
  ];

  return (
    <div
      className={cn(PAGE_CONTENT_GUTTER, "flex min-h-0 flex-1 flex-col")}
      data-testid="tasks-detail-content"
    >
      <TasksDetailHeader
        detail={detail}
        pending={{
          cancel: page.isCancelPending,
          delete: deleteMutation.isPending,
          enqueue: page.isEnqueuePending,
          pause: page.isPausePending,
          publish: page.isPublishPending,
          recover: page.isRecoverPending,
          resume: page.isResumePending,
        }}
        onCancel={page.handleCancelTask}
        onDelete={handleDeleteTask}
        onEnqueueRun={page.handleEnqueueRun}
        onPause={page.handlePauseTask}
        onPublish={page.handlePublishTask}
        onRecover={page.handleRecoverTask}
        onResume={page.handleResumeTask}
      />

      <TasksDetailTabs active={page.panel} items={tabItems} onChange={page.handlePanelChange} />

      <div className="flex min-h-0 flex-1 overflow-y-auto">
        {page.panel === "overview" ? (
          <TasksDetailOverviewPanel
            detail={detail}
            inspect={page.inspect}
            inspectErrorMessage={page.inspectError?.message ?? null}
            inspectLoading={page.inspectLoading}
          />
        ) : null}
        {page.panel === "runs" ? (
          <TasksDetailRunsPanel
            errorMessage={page.runsError?.message ?? null}
            isLoading={page.runsLoading}
            runs={page.runs}
            taskId={taskId}
          />
        ) : null}
        {page.panel === "timeline" ? (
          <TasksTimelinePanel
            canLoadMore={page.isTimelineSaturated}
            errorMessage={page.timelineError?.message ?? null}
            isLive={page.isLive}
            isLoading={page.timelineLoading}
            items={page.timeline}
            onLoadMore={page.handleTimelineLoadMore}
          />
        ) : null}
        {page.panel === "agents" ? (
          <TasksMultiAgentPanel
            activeDescendants={page.multiAgent.activeDescendants}
            agents={page.multiAgent.nodes}
            descendantCount={page.multiAgent.descendantCount}
            errorMessage={page.treeError?.message ?? null}
            liveCount={page.multiAgent.liveCount}
            state={page.multiAgent.state}
            timeline={page.timeline}
          />
        ) : null}
        {page.panel === "children" ? (
          <TasksDetailChildrenPanel
            errorMessage={page.detailError?.message ?? null}
            items={children}
          />
        ) : null}
        {page.panel === "dependencies" ? (
          <TasksDetailDependenciesPanel
            dependencies={dependencies}
            errorMessage={page.detailError?.message ?? null}
          />
        ) : null}
        {page.panel === "orchestration" ? (
          <TasksDetailOrchestrationPanel
            fanOut={{
              isPending: orchestration.isFanOutPending,
              onFanOut: orchestration.handleFanOutRuns,
            }}
            notifications={{
              subscriptions: orchestration.subscriptions,
              isLoading: orchestration.subscriptionsLoading,
              errorMessage: orchestration.subscriptionsError?.message ?? null,
              isCreatePending: orchestration.isCreateSubscriptionPending,
              isDeletePending: orchestration.isDeleteSubscriptionPending,
              onCreate: orchestration.handleCreateSubscription,
              onDelete: orchestration.handleDeleteSubscription,
            }}
            profile={{
              taskId,
              profile: orchestration.profile,
              errorMessage: orchestration.profileError?.message ?? null,
              state: {
                hasActiveRun: Boolean(detail.summary?.active_run),
                isDeletePending: orchestration.isDeleteProfilePending,
                isLoading: orchestration.profileLoading,
                isSetPending: orchestration.isSetProfilePending,
              },
              onSetProfile: orchestration.handleSetProfile,
              onDeleteProfile: orchestration.handleDeleteProfile,
            }}
            reviews={{
              reviews: orchestration.reviews,
              isLoading: orchestration.reviewsLoading,
              errorMessage: orchestration.reviewsError?.message ?? null,
            }}
            stream={{
              latestEventSeq,
              hasLatestEventSeq: orchestration.hasLatestEventSeq,
              streamSeedSequence: orchestration.streamSeedSequence,
              streamState: orchestration.streamState,
              streamErrorMessage: orchestration.streamErrorMessage,
            }}
          />
        ) : null}
      </div>
    </div>
  );
}
