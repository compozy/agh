import { describe, expect, it } from "vitest";

import type { LoopContract } from "../../types";
import { buildLoopLimits, formatTokenBudget, formatWallClock } from "../loop-limits";

const baseContract: LoopContract = {
  goal: "g",
  definition_of_done: "d",
  iteration_cap: 50,
  budget: { tokens: 0, wall_clock_sec: 0, on_exceeded: "halt" },
  no_progress: { window: 3, hash_fields: [] },
};

describe("loop-limits", () => {
  it("Should format token budgets compactly and off when unset", () => {
    expect(formatTokenBudget(0)).toBe("off");
    expect(formatTokenBudget(500_000)).toBe("500K");
    expect(formatTokenBudget(20_000_000)).toBe("20M");
    expect(formatTokenBudget(2_400_000)).toBe("2.4M");
  });

  it("Should format wall-clock budgets, rendering off when unset", () => {
    expect(formatWallClock(0)).toBe("off");
    expect(formatWallClock(604_800)).toBe("7d");
    expect(formatWallClock(3_600)).toBe("1h");
    expect(formatWallClock(90)).toBe("2m");
  });

  it("Should pair each per-loop default with its hard daemon ceiling", () => {
    const rows = buildLoopLimits(baseContract);
    const byLabel = new Map(rows.map(row => [row.label, row]));
    expect(byLabel.get("Iteration cap")).toMatchObject({ value: "50", ceiling: "/ 100" });
    expect(byLabel.get("Token budget")).toMatchObject({ value: "off", ceiling: "/ 20M" });
    expect(byLabel.get("Wall clock")).toMatchObject({ value: "off", ceiling: "/ 7d" });
    expect(byLabel.get("On exceeded")).toMatchObject({ value: "halt", ceiling: "→ exhausted" });
    expect(byLabel.get("Cost (USD)")).toMatchObject({ value: "—", ceiling: "display-only" });
    expect(byLabel.get("Fan-out breadth")).toMatchObject({ value: "≤ tasks", ceiling: "/ 64" });
  });

  it("Should render the unbounded glyph for watch loops and escalate targets", () => {
    const rows = buildLoopLimits({
      ...baseContract,
      iteration_cap: 0,
      budget: { tokens: 0, wall_clock_sec: 0, on_exceeded: "escalate" },
    });
    const byLabel = new Map(rows.map(row => [row.label, row]));
    expect(byLabel.get("Iteration cap")?.value).toBe("∞");
    expect(byLabel.get("On exceeded")).toMatchObject({
      value: "escalate",
      ceiling: "→ needs-approval",
    });
  });
});
