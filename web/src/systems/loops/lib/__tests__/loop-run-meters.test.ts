import { describe, expect, it } from "vitest";

import { buildRunMeters, deriveCostEstimate } from "../loop-run-meters";
import { loopRunDetailByRunId } from "../../mocks/fixtures";
import type { LoopRunRecord } from "../../types";

const base = loopRunDetailByRunId.get("looprun_running")!.run;

function run(overrides: Partial<LoopRunRecord>): LoopRunRecord {
  return { ...base, ...overrides };
}

function meter(record: LoopRunRecord, key: string, breadth = 0) {
  return buildRunMeters(record, breadth).find(m => m.key === key)!;
}

describe("loop-run-meters model", () => {
  it("Should build the 5 run meters", () => {
    expect(buildRunMeters(base).map(m => m.key)).toEqual([
      "attempts",
      "tokens",
      "wall",
      "cost",
      "breadth",
    ]);
  });

  it("Should warn-tint a budget meter only near its ceiling", () => {
    expect(meter(run({ tokens_used: 500_000, budget_tokens: 1_000_000 }), "tokens").tone).toBe(
      "neutral"
    );
    expect(meter(run({ tokens_used: 950_000, budget_tokens: 1_000_000 }), "tokens").tone).toBe(
      "warn"
    );
    expect(meter(run({ tokens_used: 1_000_000, budget_tokens: 1_000_000 }), "tokens").tone).toBe(
      "danger"
    );
  });

  it("Should render an unset budget with no bar and a neutral tone", () => {
    const tokens = meter(run({ tokens_used: 400_000, budget_tokens: 0 }), "tokens");
    expect(tokens.percent).toBeNull();
    expect(tokens.tone).toBe("neutral");
  });

  it("Should render an unbounded iteration cap as ∞ with no bar", () => {
    const attempts = meter(run({ generation: 5, iteration_cap: 0 }), "attempts");
    expect(attempts.percent).toBeNull();
    expect(attempts.max).toBe("/ ∞");
  });

  it("Should keep Cost display-only: no bar, no ceiling, no warn tint", () => {
    const cost = meter(run({ tokens_used: 99_000_000, budget_tokens: 1_000 }), "cost");
    expect(cost.displayOnly).toBe(true);
    expect(cost.percent).toBeNull();
    expect(cost.max).toBeUndefined();
    expect(cost.tone).toBe("neutral");
    expect(cost.tags).toContain("Display-only");
    expect(cost.value.startsWith("~$")).toBe(true);
  });

  it("Should derive breadth from the materialized branch count against the 64 ceiling", () => {
    const breadth = meter(base, "breadth", 3);
    expect(breadth.value).toBe("3");
    expect(breadth.max).toBe("/ 64");
    expect(breadth.tone).toBe("neutral");
    expect(meter(base, "breadth", 0).value).toBe("—");
  });

  it("Should derive the cost estimate as tokens × rate with a visible estimate qualifier", () => {
    expect(deriveCostEstimate(1_000_000)).toBe("~$5.00");
    expect(deriveCostEstimate(0)).toBe("~$0.00");
  });
});
