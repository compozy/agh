import {
  parseTasksSurfaceMode,
  validateTaskCreateSearch,
  validateTaskDetailSearch,
  type ResolvedTaskDetailSearch,
  type TaskViewMode,
} from "@/systems/tasks";
import type { OsWindowRoute } from "../../lib/os-types";

export type TaskWindowLocation =
  | { kind: "catalog"; mode: TaskViewMode }
  | { kind: "create"; search: ReturnType<typeof validateTaskCreateSearch> }
  | { kind: "detail"; taskId: string; search: ResolvedTaskDetailSearch }
  | { kind: "edit"; taskId: string }
  | { kind: "run"; taskId: string; runId: string };

function decodePathSegment(value: string): string {
  try {
    return decodeURIComponent(value);
  } catch {
    return value;
  }
}

export function parseTaskWindowLocation(location: OsWindowRoute): TaskWindowLocation {
  if (location.pathname === "/tasks/new") {
    return { kind: "create", search: validateTaskCreateSearch(location.search) };
  }

  const runMatch = /^\/tasks\/([^/]+)\/runs\/([^/]+)$/.exec(location.pathname);
  if (runMatch) {
    return {
      kind: "run",
      taskId: decodePathSegment(runMatch[1]),
      runId: decodePathSegment(runMatch[2]),
    };
  }

  const editMatch = /^\/tasks\/([^/]+)\/edit$/.exec(location.pathname);
  if (editMatch) return { kind: "edit", taskId: decodePathSegment(editMatch[1]) };

  const detailMatch = /^\/tasks\/([^/]+)$/.exec(location.pathname);
  if (detailMatch) {
    return {
      kind: "detail",
      taskId: decodePathSegment(detailMatch[1]),
      search: validateTaskDetailSearch(location.search),
    };
  }

  return { kind: "catalog", mode: parseTasksSurfaceMode(location.search) };
}
