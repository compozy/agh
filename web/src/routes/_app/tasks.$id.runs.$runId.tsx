import { AlertCircle, Play } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";

import { Spinner } from "@agh/ui";
import type { TopbarRouteContext } from "@/types/topbar";
import { useTaskRunPage } from "@/hooks/routes/use-task-run-page";
import {
  TaskRunDetailHeader,
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
  const timelineQuery = useTaskTimeline(id, {}, { enabled: Boolean(id) });

  if (page.runLoading) {
    return (
      <div
        className="flex flex-1 items-center justify-center"
        data-testid="tasks-run-detail-loading"
      >
        <Spinner className="size-5 text-(--subtle)" />
      </div>
    );
  }

  if (page.notFound || (!page.run && page.fatalError)) {
    return (
      <div
        className="flex flex-1 flex-col items-center justify-center gap-2 px-6 text-center"
        data-testid="tasks-run-detail-not-found"
      >
        <AlertCircle className="size-6 text-(--danger)" />
        <p className="text-sm text-(--muted)">
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
        <Spinner className="size-5 text-(--subtle)" />
      </div>
    );
  }

  const timelineItems = timelineQuery.data ?? [];

  return (
    <div className="flex min-h-0 flex-1 flex-col" data-testid="tasks-run-detail-content">
      <TaskRunDetailHeader
        isCancelPending={page.isCancelPending}
        onCancelRun={page.handleCancelRun}
        run={run}
      />

      <div
        className="flex min-h-0 flex-1 flex-col gap-6 overflow-y-auto px-6 py-5"
        data-testid="tasks-run-detail-main"
      >
        <TaskRunTimelinePanel
          isLive={page.isLive}
          isLoading={timelineQuery.isLoading && timelineItems.length === 0}
          items={timelineItems}
          run={run}
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
