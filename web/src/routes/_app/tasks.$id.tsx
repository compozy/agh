import { AlertCircle, Loader2 } from "lucide-react";
import { Outlet, createFileRoute, useChildMatches } from "@tanstack/react-router";

import { useTaskDetailRoute } from "@/hooks/routes/use-task-detail-route";
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
} from "@/systems/tasks";
import type { TasksDetailTabItem } from "@/systems/tasks/components/tasks-detail-tabs";

export const Route = createFileRoute("/_app/tasks/$id")({
  component: TaskDetailRoute,
});

function TaskDetailRoute() {
  const { id } = Route.useParams();
  const childMatches = useChildMatches();
  const hasChildMatch = childMatches.length > 0;

  const { page, orchestration, deleteMutation, handleDeleteTask, latestEventSeq } =
    useTaskDetailRoute(id);

  if (hasChildMatch) {
    return <Outlet />;
  }

  if (page.detailLoading) {
    return (
      <div className="flex flex-1 items-center justify-center" data-testid="tasks-detail-loading">
        <Loader2 className="size-5 animate-spin text-[color:var(--color-text-tertiary)]" />
      </div>
    );
  }

  if (page.notFound || (!page.detail && page.fatalError)) {
    return (
      <div
        className="flex flex-1 flex-col items-center justify-center gap-2 px-6 text-center"
        data-testid="tasks-detail-not-found"
      >
        <AlertCircle className="size-6 text-[color:var(--color-danger)]" />
        <p className="text-sm text-[color:var(--color-text-secondary)]">
          {page.fatalError?.message ?? `Task ${id} not found.`}
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
        <Loader2 className="size-5 animate-spin text-[color:var(--color-text-tertiary)]" />
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

  const hasActiveRun = Boolean(detail.summary?.active_run);

  return (
    <div className="flex min-h-0 flex-1 flex-col" data-testid="tasks-detail-content">
      <TasksDetailHeader
        detail={detail}
        isCancelPending={page.isCancelPending}
        isDeletePending={deleteMutation.isPending}
        isEnqueuePending={page.isEnqueuePending}
        isPublishPending={page.isPublishPending}
        onCancel={page.handleCancelTask}
        onDelete={handleDeleteTask}
        onEnqueueRun={page.handleEnqueueRun}
        onPublish={page.handlePublishTask}
      />

      <TasksDetailTabs active={page.panel} items={tabItems} onChange={page.handlePanelChange} />

      <div className="flex min-h-0 flex-1 overflow-y-auto">
        {page.panel === "overview" ? <TasksDetailOverviewPanel detail={detail} /> : null}
        {page.panel === "runs" ? (
          <TasksDetailRunsPanel
            errorMessage={page.runsError?.message ?? null}
            isLoading={page.runsLoading}
            runs={page.runs}
            taskId={id}
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
              taskId: id,
              profile: orchestration.profile,
              isLoading: orchestration.profileLoading,
              errorMessage: orchestration.profileError?.message ?? null,
              hasActiveRun,
              isSetPending: orchestration.isSetProfilePending,
              isDeletePending: orchestration.isDeleteProfilePending,
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
