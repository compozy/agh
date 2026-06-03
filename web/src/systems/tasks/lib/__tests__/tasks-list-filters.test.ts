import { describe, expect, it, vi } from "vitest";

import {
  applyTaskFilterChips,
  buildTaskFilterFields,
  taskFiltersToChips,
} from "../tasks-list-filters";

describe("taskFiltersToChips", () => {
  it("Should omit chips when filters are unset", () => {
    expect(
      taskFiltersToChips({
        statusFilter: null,
        ownerFilter: null,
        priorityFilter: null,
      })
    ).toEqual([]);
  });

  it("Should emit one chip per active filter with operator 'is' and a single value", () => {
    const chips = taskFiltersToChips({
      statusFilter: "in_progress",
      ownerFilter: "Coder",
      priorityFilter: "high",
    });

    expect(chips).toHaveLength(3);
    const status = chips.find(chip => chip.field === "status");
    expect(status?.operator).toBe("is");
    expect(status?.values).toEqual(["in_progress"]);
    expect(chips.find(chip => chip.field === "owner")?.values).toEqual(["Coder"]);
    expect(chips.find(chip => chip.field === "priority")?.values).toEqual(["high"]);
  });

  it("Should keep chip ids stable across renders for the same field", () => {
    const first = taskFiltersToChips({
      statusFilter: "ready",
      ownerFilter: null,
      priorityFilter: null,
    });
    const second = taskFiltersToChips({
      statusFilter: "ready",
      ownerFilter: null,
      priorityFilter: null,
    });

    expect(first[0]?.id).toBe(second[0]?.id);
  });
});

describe("applyTaskFilterChips", () => {
  it("Should reset every slot to its default when no chips are present", () => {
    const handlers = {
      onStatusChange: vi.fn(),
      onOwnerChange: vi.fn(),
      onPriorityChange: vi.fn(),
    };

    applyTaskFilterChips([], handlers);
    expect(handlers.onStatusChange).toHaveBeenCalledWith(null);
    expect(handlers.onOwnerChange).toHaveBeenCalledWith(null);
    expect(handlers.onPriorityChange).toHaveBeenCalledWith(null);
  });

  it("Should route chip values to the matching typed setter", () => {
    const handlers = {
      onStatusChange: vi.fn(),
      onOwnerChange: vi.fn(),
      onPriorityChange: vi.fn(),
    };

    applyTaskFilterChips(
      [
        { id: "task-filter-status", field: "status", operator: "is", values: ["blocked"] },
        { id: "task-filter-priority", field: "priority", operator: "is", values: ["urgent"] },
        { id: "task-filter-owner", field: "owner", operator: "is", values: ["Coder"] },
      ],
      handlers
    );

    expect(handlers.onStatusChange).toHaveBeenCalledWith("blocked");
    expect(handlers.onPriorityChange).toHaveBeenCalledWith("urgent");
    expect(handlers.onOwnerChange).toHaveBeenCalledWith("Coder");
  });

  it("Should drop invalid values from the typed setters", () => {
    const handlers = {
      onStatusChange: vi.fn(),
      onOwnerChange: vi.fn(),
      onPriorityChange: vi.fn(),
    };

    applyTaskFilterChips(
      [
        { id: "task-filter-status", field: "status", operator: "is", values: ["bogus"] },
        { id: "task-filter-priority", field: "priority", operator: "is", values: ["bogus"] },
      ],
      handlers
    );

    expect(handlers.onStatusChange).toHaveBeenCalledWith(null);
    expect(handlers.onPriorityChange).toHaveBeenCalledWith(null);
  });
});

describe("buildTaskFilterFields", () => {
  it("Should expose status, priority, and owner as single-select chip fields", () => {
    const fields = buildTaskFilterFields([{ ref: "Coder", kind: "agent_session" }]);
    expect(fields).toHaveLength(3);
    const keys = (fields as Array<{ key?: string }>).map(field => field.key);
    expect(keys).toEqual(["status", "priority", "owner"]);
  });

  it("Should mirror the live owner options inside the owner field", () => {
    const fields = buildTaskFilterFields([
      { ref: "pedro@", kind: "human" },
      { ref: "Coder", kind: "agent_session" },
    ]);

    const owner = (fields as Array<{ key?: string; options?: Array<{ value: string }> }>).find(
      field => field.key === "owner"
    );
    expect(owner?.options?.map(option => option.value)).toEqual(["pedro@", "Coder"]);
  });
});
