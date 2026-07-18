import { AlertCircle, Play } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";

import { Spinner } from "@agh/ui";
import type { TopbarRouteContext } from "@/types/topbar";
import { useTaskRunPage } from "@/hooks/routes/use-task-run-page";
import {
  formatTaskRunBounds,
  TaskRunConversationPanel,
  TaskRunCoordinationInvitationHost,
  useTaskRunConversation,
} from "@/systems/network";
import {
  TaskRunDetailHeader,
  TaskInspectDiagnosticsCard,
  TaskRunTimelinePanel,
  TasksReviewsCard,
  useTaskTimeline,
} from "@/systems/tasks";

export const Route = createFileRoute("/_app/tasks/$id/runs/$runId")({
  beforeLoad: ({ params }): { topbar: TopbarRouteContext } => ({
    topbar: { title: `Run ${params.runId}`, icon: Play },
  }),
  component: TaskRunDetailRoute,
});

function TaskRunDetailRoute() {
  const { id, runId } = Route.useParams();
  const page = useTaskRunPage(id, runId);
  const taskId = page.run?.task?.id ?? "";
  const timelineQuery = useTaskTimeline(taskId, {}, { enabled: Boolean(taskId) });
  const conversation = useTaskRunConversation(runId, page.run?.network);

  if (page.runLoading) {
    return (
      <div
        className="flex flex-1 items-center justify-center"
        data-testid="tasks-run-detail-loading"
      >
        <Spinner className="size-5 text-subtle" />
      </div>
    );
  }

  if (page.notFound || (!page.run && page.fatalError)) {
    return (
      <div
        className="flex flex-1 flex-col items-center justify-center gap-2 px-6 text-center"
        data-testid="tasks-run-detail-not-found"
      >
        <AlertCircle className="size-6 text-danger" />
        <p className="text-sm text-muted">
          {page.fatalError?.message ?? `Run ${runId} not found.`}
        </p>
      </div>
    );
  }

  const run = page.run;
  if (!run) {
    return (
      <div
        className="flex flex-1 items-center justify-center"
        data-testid="tasks-run-detail-placeholder"
      >
        <Spinner className="size-5 text-subtle" />
      </div>
    );
  }

  const timelineItems = timelineQuery.data ?? [];
  const record = run.run;
  const participation = record.resolved_network_participation;
  const coordinationWorkspaceId =
    (participation?.mode === "live" ? participation.workspace_id : undefined) ??
    page.task?.task.workspace_id ??
    "";

  return (
    <div className="flex min-h-0 flex-1 flex-col" data-testid="tasks-run-detail-content">
      <TaskRunDetailHeader
        maxAttempts={page.task?.task.max_attempts}
        pendingActions={
          new Set(
            [
              page.isCancelPending ? "cancel" : null,
              page.isForceReleasePending ? "force-release" : null,
              page.isForceFailPending ? "force-fail" : null,
              page.isRecoverPending ? "recover" : null,
              page.isRetryPending ? "retry" : null,
            ].filter(
              (action): action is "cancel" | "force-release" | "force-fail" | "recover" | "retry" =>
                action !== null
            )
          )
        }
        onCancelRun={page.handleCancelRun}
        onForceFailRun={page.handleForceFailRun}
        onForceReleaseRun={page.handleForceReleaseRun}
        onRecoverRun={page.handleRecoverRun}
        onRetryRun={page.handleRetryRun}
        run={run}
      />

      <div
        className="flex min-h-0 flex-1 flex-col gap-6 overflow-y-auto px-6 py-5"
        data-testid="tasks-run-detail-main"
      >
        <div
          className="rounded-md border border-border px-3 py-2 text-sm"
          data-testid="tasks-run-participation-chip"
        >
          Participation: {participation?.mode ?? "local"}
          {participation?.mode === "live" ? ` · ${participation.channel_id}` : ""}
        </div>
        {taskId && coordinationWorkspaceId ? (
          <TaskRunCoordinationInvitationHost
            runId={record.id}
            taskId={taskId}
            workspaceId={coordinationWorkspaceId}
          />
        ) : null}
        {conversation && run.network ? (
          <TaskRunConversationPanel
            boundsLabel={formatTaskRunBounds(record, run.network)}
            conversationError={conversation.error}
            conversationLoading={conversation.isLoading}
            hasMoreMessages={conversation.hasOlder}
            isFetchingMore={conversation.isLoadingOlder}
            messages={conversation.messages}
            onLoadMore={conversation.loadOlder}
            streamError={conversation.streamError}
            usage={conversation.usage}
          />
        ) : null}
        <TaskRunTimelinePanel
          isLive={page.isLive}
          isLoading={timelineQuery.isLoading && timelineItems.length === 0}
          items={timelineItems}
          run={run}
        />
        <TaskInspectDiagnosticsCard
          errorMessage={page.inspectError?.message ?? null}
          inspect={page.inspect}
          isLoading={page.inspectLoading}
          label="Run inspect diagnostics"
          testId="tasks-run-inspect-diagnostics-card"
        />
        <TasksReviewsCard
          errorMessage={page.reviewsError?.message ?? null}
          isLoading={page.reviewsLoading}
          label="Run reviews"
          reviews={page.reviews}
          testId="tasks-run-reviews-card"
          testIdPrefix="tasks-run-reviews-row"
        />
      </div>
    </div>
  );
}
