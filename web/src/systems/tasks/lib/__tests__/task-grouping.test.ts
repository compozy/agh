import { describe, expect, it } from "vitest";

import { getKanbanColumns, groupTasksForKanban, resolveKanbanColumnId } from "../task-grouping";
import type { TaskListItem } from "../../types";

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
  it("Should return the canonical Pending / In progress / Blocked / Done columns in declared order", () => {
    const columns = getKanbanColumns();
    expect(columns.map(column => column.id)).toEqual(["pending", "in_progress", "blocked", "done"]);
    expect(columns.map(column => column.label)).toEqual([
      "Pending",
      "In progress",
      "Blocked",
      "Done",
    ]);
  });

  it("Should map production task statuses to the collapsed four-column kanban", () => {
    expect(resolveKanbanColumnId("draft")).toBe("pending");
    expect(resolveKanbanColumnId("pending")).toBe("pending");
    expect(resolveKanbanColumnId("ready")).toBe("pending");
    expect(resolveKanbanColumnId("blocked")).toBe("blocked");
    expect(resolveKanbanColumnId("in_progress")).toBe("in_progress");
    expect(resolveKanbanColumnId("completed")).toBe("done");
    expect(resolveKanbanColumnId("failed")).toBe("done");
    expect(resolveKanbanColumnId("canceled")).toBe("done");
  });

  it("Should accept the mock status shorthand `running` and `done` so designer fixtures route correctly", () => {
    expect(resolveKanbanColumnId("running")).toBe("in_progress");
    expect(resolveKanbanColumnId("done")).toBe("done");
  });

  it("Should group tasks into the four columns and preserve empty columns", () => {
    const tasks: TaskListItem[] = [
      buildTask("a", "draft"),
      buildTask("b", "pending"),
      buildTask("c", "ready"),
      buildTask("d", "in_progress"),
      buildTask("e", "failed"),
      buildTask("f", "canceled"),
      buildTask("g", "blocked"),
    ];

    const groups = groupTasksForKanban(tasks);
    const byId = new Map(groups.map(group => [group.column.id, group.tasks.map(t => t.id)]));

    expect(byId.get("pending")).toEqual(["a", "b", "c"]);
    expect(byId.get("in_progress")).toEqual(["d"]);
    expect(byId.get("blocked")).toEqual(["g"]);
    expect(byId.get("done")).toEqual(["e", "f"]);
    expect(groups).toHaveLength(4);
  });
});
