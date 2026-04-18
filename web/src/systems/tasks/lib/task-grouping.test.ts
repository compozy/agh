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
  it("returns the canonical Pending / Running / Done / Failed columns in declared order", () => {
    const columns = getKanbanColumns();
    expect(columns.map(column => column.id)).toEqual(["pending", "running", "done", "failed"]);
    expect(columns.map(column => column.label)).toEqual(["Pending", "Running", "Done", "Failed"]);
  });

  it("maps production task statuses to the collapsed four-column kanban", () => {
    expect(resolveKanbanColumnId("draft")).toBe("pending");
    expect(resolveKanbanColumnId("pending")).toBe("pending");
    expect(resolveKanbanColumnId("ready")).toBe("pending");
    expect(resolveKanbanColumnId("blocked")).toBe("pending");
    expect(resolveKanbanColumnId("in_progress")).toBe("running");
    expect(resolveKanbanColumnId("completed")).toBe("done");
    expect(resolveKanbanColumnId("failed")).toBe("failed");
    expect(resolveKanbanColumnId("canceled")).toBe("failed");
  });

  it("accepts the mock status shorthand `running` and `done` so designer fixtures route correctly", () => {
    expect(resolveKanbanColumnId("running")).toBe("running");
    expect(resolveKanbanColumnId("done")).toBe("done");
  });

  it("groups tasks into the four columns and preserves empty columns", () => {
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

    expect(byId.get("pending")).toEqual(["a", "b", "c"]);
    expect(byId.get("running")).toEqual(["d"]);
    expect(byId.get("done")).toEqual([]);
    expect(byId.get("failed")).toEqual(["e", "f"]);
    expect(groups).toHaveLength(4);
  });
});
