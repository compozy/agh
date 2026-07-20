import { createFileRoute } from "@tanstack/react-router";

import { createOsRouteSync } from "@/systems/os";
import type { TopbarRouteContext } from "@/types/topbar";
import { preloadTasksRoute } from "./-tasks-preload";

export type TasksSurfaceMode = "list" | "kanban" | "dashboard" | "inbox";

export interface TasksRouteSearch {
  /** Surface mode; `list` is the default and stays out of the URL. */
  mode?: Exclude<TasksSurfaceMode, "list">;
}

function validateTasksSearch(search: Record<string, unknown>): TasksRouteSearch {
  return {
    mode:
      search.mode === "kanban" || search.mode === "dashboard" || search.mode === "inbox"
        ? search.mode
        : undefined,
  };
}

export const Route = createFileRoute("/_app/tasks")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { crumb: { label: "Tasks", to: "/tasks" } },
  }),
  validateSearch: validateTasksSearch,
  loaderDeps: ({ search }) => ({ mode: search.mode }),
  loader: ({ context, deps }) => preloadTasksRoute(context.queryClient, deps.mode),
  component: createOsRouteSync("tasks"),
});
