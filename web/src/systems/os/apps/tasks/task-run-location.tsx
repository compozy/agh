import { AlertCircle } from "lucide-react";

import { cn, PAGE_CONTENT_GUTTER, Spinner } from "@agh/ui";

import {
  formatTaskRunBounds,
  TaskRunConversationPanel,
  TaskRunCoordinationInvitationHost,
  useTaskRunConversation,
} from "@/systems/network";
import {
  TaskInspectDiagnosticsCard,
  TaskRunDetailHeader,
  TaskRunTimelinePanel,
  TasksReviewsCard,
  useTaskRunPage,
  useTaskTimeline,
} from "@/systems/tasks";

export function TaskRunLocation({ taskId, runId }: { taskId: string; runId: string }) {
  const page = useTaskRunPage(taskId, runId);
  const authoritativeTaskId = page.run?.task?.id ?? "";
  const timelineQuery = useTaskTimeline(
    authoritativeTaskId,
    {},
    {
      enabled: Boolean(authoritativeTaskId),
    }
  );
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
        className={cn(
          PAGE_CONTENT_GUTTER,
          "flex flex-1 flex-col items-center justify-center gap-2 py-10 text-center"
        )}
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
    <div
      className={cn(PAGE_CONTENT_GUTTER, "flex min-h-0 flex-1 flex-col")}
      data-testid="tasks-run-detail-content"
    >
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
        className="flex min-h-0 flex-1 flex-col gap-6 overflow-y-auto py-5"
        data-testid="tasks-run-detail-main"
      >
        <div
          className="rounded-md border border-border px-3 py-2 text-sm"
          data-testid="tasks-run-participation-chip"
        >
          Participation: {participation?.mode ?? "local"}
          {participation?.mode === "live" ? ` · ${participation.channel_id}` : ""}
        </div>
        {authoritativeTaskId && coordinationWorkspaceId ? (
          <TaskRunCoordinationInvitationHost
            runId={record.id}
            taskId={authoritativeTaskId}
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
