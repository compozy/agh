import { AlertCircle } from "lucide-react";

import {
  cn,
  Empty,
  JsonViewer,
  LiveBadge,
  Markdown,
  PAGE_CONTENT_GUTTER,
  Section,
  Skeleton,
} from "@agh/ui";

import {
  formatTaskRunBounds,
  TaskRunConversationPanel,
  TaskRunCoordinationInvitationHost,
  useTaskRunConversation,
} from "@/systems/network";
import {
  TaskActivityItem,
  TaskRunForceFailDialog,
  TaskRunInspectDrawer,
  TaskRunOutcome,
  TaskRunRail,
  TaskRunReviewCard,
  TaskRunSubhead,
} from "@/systems/tasks";

import { TaskRunTopbar } from "./task-run-topbar";
import { useTaskRunLocation, type TaskRunLocationController } from "./use-task-run-location";

function TaskRunResultSection({ controller }: { controller: TaskRunLocationController }) {
  const record = controller.record;
  if (!record) return null;
  const result = record.result;

  return (
    <Section data-testid="tasks-run-result" label="Result">
      {result == null ? (
        <p className="rounded-lg border border-line bg-canvas-soft px-4 py-3.5 text-small-body text-muted">
          {record.status === "completed"
            ? "No result recorded."
            : record.ended_at
              ? "No result recorded. The run ended before reaching a checkpoint."
              : "No result yet. It lands here the moment the run completes."}
        </p>
      ) : typeof result === "string" ? (
        <div className="rounded-lg border border-line bg-canvas-soft px-4 py-3.5">
          <Markdown>{result}</Markdown>
        </div>
      ) : (
        <JsonViewer data-testid="tasks-run-result-json" value={result} />
      )}
    </Section>
  );
}

export function TaskRunLocation({ taskId, runId }: { taskId: string; runId: string }) {
  const controller = useTaskRunLocation(taskId, runId);
  const { page, record } = controller;
  const conversation = useTaskRunConversation(runId, page.run?.network);

  if (page.runLoading) {
    return (
      <div
        className={cn(PAGE_CONTENT_GUTTER, "flex min-h-0 flex-1 flex-col gap-4 py-5")}
        data-testid="tasks-run-detail-loading"
      >
        <Skeleton className="h-6 w-72" />
        <div className="grid gap-8 lg:grid-cols-[minmax(0,1fr)_320px]">
          <div className="flex flex-col gap-3">
            <Skeleton className="h-24 rounded-lg" />
            <Skeleton className="h-48 rounded-lg" />
          </div>
          <Skeleton className="hidden h-72 rounded-lg lg:block" />
        </div>
      </div>
    );
  }

  if (page.notFound || !page.run || !record) {
    return (
      <div
        className={cn(PAGE_CONTENT_GUTTER, "flex flex-1 items-center justify-center py-8")}
        data-testid="tasks-run-detail-not-found"
      >
        <Empty
          description={page.fatalError?.message ?? `Run ${runId} was not found.`}
          icon={AlertCircle}
          title="Run not found"
        />
      </div>
    );
  }

  const timelineItems = controller.timelineItems.filter(item => item.run?.id === record.id);
  const participation = record.resolved_network_participation;
  const coordinationWorkspaceId =
    (participation?.mode === "live" ? participation.workspace_id : undefined) ??
    page.task?.task.workspace_id ??
    "";
  const nextAttempt =
    controller.taskRuns.find(candidate => candidate.previous_run_id === record.id) ?? null;

  return (
    <div
      className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden"
      data-testid="tasks-run-detail-content"
    >
      <TaskRunTopbar controller={controller} />
      <TaskRunForceFailDialog
        dialog={controller.forceFailDialog}
        isPending={page.isForceFailPending}
      />
      <TaskRunInspectDrawer
        inspect={page.inspect}
        onOpenChange={controller.setInspectOpen}
        open={controller.inspectOpen}
        run={page.run}
      />

      <div className="min-h-0 flex-1 overflow-y-auto">
        <div className={cn(PAGE_CONTENT_GUTTER, "@container pt-5 pb-16")}>
          <TaskRunSubhead run={page.run} />

          <div className="grid items-start gap-8 @min-[64rem]:grid-cols-[minmax(0,1fr)_320px]">
            <main className="flex min-w-0 flex-col gap-6" data-testid="tasks-run-detail-main">
              <TaskRunOutcome nextAttempt={nextAttempt} run={page.run} />
              <TaskRunResultSection controller={controller} />

              {page.reviews.length > 0 ? (
                <Section
                  count={page.reviews.length}
                  data-testid="tasks-run-reviews"
                  label="Reviews"
                >
                  <div className="flex flex-col gap-2.5">
                    {page.reviews.map(review => (
                      <TaskRunReviewCard key={review.review_id} review={review} />
                    ))}
                  </div>
                </Section>
              ) : null}

              {controller.authoritativeTaskId && coordinationWorkspaceId ? (
                <TaskRunCoordinationInvitationHost
                  runId={record.id}
                  taskId={controller.authoritativeTaskId}
                  workspaceId={coordinationWorkspaceId}
                />
              ) : null}
              {conversation && page.run.network ? (
                <TaskRunConversationPanel
                  boundsLabel={formatTaskRunBounds(record, page.run.network)}
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

              <Section
                data-testid="tasks-run-activity"
                label="Run activity"
                right={
                  page.isLive ? <LiveBadge data-testid="tasks-run-activity-live" /> : undefined
                }
              >
                {timelineItems.length === 0 ? (
                  <p className="rounded-lg border border-line bg-canvas-soft px-4 py-3.5 text-small-body text-muted">
                    No events recorded for this attempt yet.
                  </p>
                ) : (
                  <div className="overflow-hidden rounded-lg border border-line bg-canvas-soft">
                    {timelineItems.map(item => (
                      <TaskActivityItem isLive={page.isLive} item={item} key={item.event_id} />
                    ))}
                  </div>
                )}
              </Section>
            </main>

            <aside className="min-w-0 @min-[64rem]:sticky @min-[64rem]:top-0">
              <TaskRunRail
                onInspect={() => controller.setInspectOpen(true)}
                run={page.run}
                taskId={controller.authoritativeTaskId || taskId}
                taskRuns={controller.taskRuns}
              />
            </aside>
          </div>
        </div>
      </div>
    </div>
  );
}
