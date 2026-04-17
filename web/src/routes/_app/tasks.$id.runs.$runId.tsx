import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_app/tasks/$id/runs/$runId")({
  component: TaskRunDetailRoute,
});

function TaskRunDetailRoute() {
  const { id, runId } = Route.useParams();

  return (
    <div
      className="flex min-h-0 flex-1 items-center justify-center px-6 py-12 text-sm text-[color:var(--color-text-tertiary)]"
      data-testid="tasks-run-detail-placeholder"
    >
      Run <span className="mx-1 font-mono">{runId}</span> for task
      <span className="ml-1 font-mono">{id}</span> will render here.
    </div>
  );
}
