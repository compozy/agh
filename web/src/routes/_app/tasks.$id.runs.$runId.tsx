import { AlertCircle, Loader2, Play } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";

import type { TopbarRouteContext } from "@/types/topbar";
import { useTaskRunPage } from "@/hooks/routes/use-task-run-page";
import {
  TaskRunActivityPanel,
  TaskRunDetailHeader,
  TaskRunIdentityPanel,
  TaskRunProgressPanel,
  TasksReviewsCard,
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

  if (page.runLoading) {
    return (
      <div
        className="flex flex-1 items-center justify-center"
        data-testid="tasks-run-detail-loading"
      >
        <Loader2 className="size-5 animate-spin text-(--subtle)" />
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
        <Loader2 className="size-5 animate-spin text-(--subtle)" />
      </div>
    );
  }

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
        <TaskRunIdentityPanel run={run} />
        <TaskRunProgressPanel run={run} />
        <TaskRunActivityPanel run={run} />
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
