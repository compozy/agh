import { Outlet, createFileRoute, useChildMatches } from "@tanstack/react-router";

import { TasksPageShell } from "@/systems/tasks";

export const Route = createFileRoute("/_app/tasks")({
  component: TasksRoute,
});

function TasksRoute() {
  const childMatches = useChildMatches();
  const hasChildMatch = childMatches.length > 0;

  return (
    <TasksPageShell>
      {hasChildMatch ? (
        <Outlet />
      ) : (
        <div
          className="flex min-h-0 flex-1 items-center justify-center px-6 py-12 text-sm text-[color:var(--color-text-tertiary)]"
          data-testid="tasks-shell-placeholder"
        >
          Tasks workspace is ready. Screen content will render here.
        </div>
      )}
    </TasksPageShell>
  );
}
