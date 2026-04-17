import { Outlet, createFileRoute, useChildMatches } from "@tanstack/react-router";

export const Route = createFileRoute("/_app/tasks/$id")({
  component: TaskDetailRoute,
});

function TaskDetailRoute() {
  const { id } = Route.useParams();
  const childMatches = useChildMatches();
  const hasChildMatch = childMatches.length > 0;

  if (hasChildMatch) {
    return <Outlet />;
  }

  return (
    <div
      className="flex min-h-0 flex-1 items-center justify-center px-6 py-12 text-sm text-[color:var(--color-text-tertiary)]"
      data-testid="tasks-detail-placeholder"
    >
      Detail for task <span className="ml-1 font-mono">{id}</span> will render here.
    </div>
  );
}
