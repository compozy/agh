import { useChildMatches, useNavigate, useSearch } from "@tanstack/react-router";

import { useTasksPage } from "@/hooks/routes/use-tasks-page";
import type { TasksSurfaceMode } from "@/routes/_app/tasks";

type SurfaceMode = TasksSurfaceMode;

export interface TasksRouteView {
  page: ReturnType<typeof useTasksPage>;
  hasChildMatch: boolean;
  routedTaskId: string | null;
  surfaceMode: SurfaceMode;
  openCreateRoute: () => void;
}

export function useTasksRoute(): TasksRouteView {
  const navigate = useNavigate({ from: "/tasks" });
  const search = useSearch({ from: "/_app/tasks" });
  const childMatches = useChildMatches();
  const hasChildMatch = childMatches.length > 0;
  const routedMode: SurfaceMode = search.mode ?? "list";
  const page = useTasksPage({ forceListData: hasChildMatch, mode: routedMode });
  const routedTaskId = extractRoutedTaskId(childMatches);

  const surfaceMode: SurfaceMode = hasChildMatch ? "list" : routedMode;

  const openCreateRoute = () => {
    void navigate({ search: () => ({ template: undefined }), to: "/tasks/new" });
  };

  return {
    page,
    hasChildMatch,
    routedTaskId,
    surfaceMode,
    openCreateRoute,
  };
}

function extractRoutedTaskId(matches: Array<unknown>): string | null {
  for (let index = matches.length - 1; index >= 0; index -= 1) {
    const match = matches[index];
    if (!match || typeof match !== "object" || !("params" in match)) {
      continue;
    }
    const params = (match as { params?: Record<string, unknown> }).params;
    if (!params || typeof params.id !== "string") {
      continue;
    }
    return params.id;
  }
  return null;
}
