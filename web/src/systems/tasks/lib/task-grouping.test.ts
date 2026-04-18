import { describe, expect, it } from "vitest";

import { getKanbanColumns, groupTasksForKanban, resolveKanbanColumnId } from "./task-grouping";
import type { TaskListItem } from "../types";

function buildTask(id: string, status: TaskListItem["status"]): TaskListItem {
  return {
    id,
    title: `Task ${id}`,
    status,
    scope: "workspace",
    origin: { kind: "web", ref: "op" },
    created_at: "2026-04-11T09:00:00Z",
    updated_at: "2026-04-11T09:00:00Z",
    created_by: { kind: "human", ref: "op" },
  } as TaskListItem;
}

describe("task-grouping", () => {
  it("returns canonical kanban columns in declared order", () => {
    const columns = getKanbanColumns();
    expect(columns.map(column => column.id)).toEqual([
      "pending",
      "ready",
      "in_progress",
      "blocked",
      "completed",
      "failed",
    ]);
  });

  it("maps task status to expected kanban column id", () => {
    expect(resolveKanbanColumnId("draft")).toBe("pending");
    expect(resolveKanbanColumnId("pending")).toBe("pending");
    expect(resolveKanbanColumnId("ready")).toBe("ready");
    expect(resolveKanbanColumnId("in_progress")).toBe("in_progress");
    expect(resolveKanbanColumnId("blocked")).toBe("blocked");
    expect(resolveKanbanColumnId("completed")).toBe("completed");
    expect(resolveKanbanColumnId("failed")).toBe("failed");
    expect(resolveKanbanColumnId("canceled")).toBe("failed");
  });

  it("groups tasks into the expected columns and preserves empty columns", () => {
    const tasks: TaskListItem[] = [
      buildTask("a", "draft"),
      buildTask("b", "pending"),
      buildTask("c", "ready"),
      buildTask("d", "in_progress"),
      buildTask("e", "failed"),
      buildTask("f", "canceled"),
    ];

    const groups = groupTasksForKanban(tasks);
    const byId = new Map(groups.map(group => [group.column.id, group.tasks.map(t => t.id)]));

    expect(byId.get("pending")).toEqual(["a", "b"]);
    expect(byId.get("ready")).toEqual(["c"]);
    expect(byId.get("in_progress")).toEqual(["d"]);
    expect(byId.get("blocked")).toEqual([]);
    expect(byId.get("completed")).toEqual([]);
    expect(byId.get("failed")).toEqual(["e", "f"]);
    expect(groups).toHaveLength(6);
  });
});
