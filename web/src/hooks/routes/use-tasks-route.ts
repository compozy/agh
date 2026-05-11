import { useChildMatches, useNavigate } from "@tanstack/react-router";

import { useTasksPage } from "@/hooks/routes/use-tasks-page";

type SurfaceMode = "list" | "kanban" | "dashboard" | "inbox";

export interface TasksRouteView {
  page: ReturnType<typeof useTasksPage>;
  hasChildMatch: boolean;
  routedTaskId: string | null;
  isCreateRoute: boolean;
  surfaceMode: SurfaceMode;
  shellCount: number;
  handleModeSelect: (next: SurfaceMode) => void;
  openCreateRoute: () => void;
}

export function useTasksRoute(): TasksRouteView {
  const navigate = useNavigate({ from: "/tasks" });
  const childMatches = useChildMatches();
  const hasChildMatch = childMatches.length > 0;
  const page = useTasksPage({ forceListData: hasChildMatch });
  const currentChildRouteId = String(childMatches.at(-1)?.id ?? "");
  const routedTaskId = extractRoutedTaskId(childMatches);
  const isCreateRoute = currentChildRouteId.includes("/tasks/new");

  const surfaceMode: SurfaceMode = hasChildMatch ? "list" : page.mode;

  const shellCount =
    surfaceMode === "inbox"
      ? (page.inbox?.total ?? 0)
      : surfaceMode === "dashboard"
        ? (page.dashboard?.totals.tasks_total ?? page.tasksCount)
        : page.tasksCount;

  const handleModeSelect = (next: SurfaceMode) => {
    page.handleModeChange(next);
    if (hasChildMatch) {
      void navigate({ to: "/tasks" });
    }
  };

  const openCreateRoute = () => {
    void navigate({ search: () => ({ template: undefined }), to: "/tasks/new" });
  };

  return {
    page,
    hasChildMatch,
    routedTaskId,
    isCreateRoute,
    surfaceMode,
    shellCount,
    handleModeSelect,
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
