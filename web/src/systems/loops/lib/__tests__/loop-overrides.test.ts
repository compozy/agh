import { describe, expect, it } from "vitest";

import {
  buildConfigOverrides,
  buildOverrideFields,
  clampOverrideValue,
  hasActiveOverrides,
  initialOverrideDraft,
} from "../loop-overrides";
import { loopDetailByName } from "../../mocks/fixtures";

const contract = loopDetailByName.get("software-delivery")!.definition.contract;

describe("loop-overrides model", () => {
  it("Should build exactly the 6 clamped fields with no cost cap", () => {
    const fields = buildOverrideFields(contract);
    expect(fields.map(field => field.key)).toEqual([
      "iteration_cap",
      "budget_tokens",
      "budget_wall_sec",
      "no_progress_window",
      "fan_out_width",
      "gate_max_revisions",
    ]);
    expect(fields.some(field => field.label.toLowerCase().includes("cost"))).toBe(false);
  });

  it("Should read per-loop defaults from the contract (wall clock in minutes)", () => {
    const fields = buildOverrideFields(contract);
    const byKey = Object.fromEntries(fields.map(field => [field.key, field.defaultValue]));
    expect(byKey.iteration_cap).toBe(50);
    expect(byKey.budget_tokens).toBe(500_000);
    expect(byKey.budget_wall_sec).toBe(60); // 3600s / 60
    expect(byKey.no_progress_window).toBe(3);
    expect(byKey.fan_out_width).toBeNull();
  });

  it("Should clamp a value to [0, ceiling]", () => {
    const [iteration] = buildOverrideFields(contract);
    expect(clampOverrideValue(iteration, 150)).toBe(100);
    expect(clampOverrideValue(iteration, -5)).toBe(0);
    expect(clampOverrideValue(iteration, 42)).toBe(42);
  });

  it("Should flip the overrides-set badge only when a value diverges from its default", () => {
    const draft = initialOverrideDraft(contract);
    expect(hasActiveOverrides(draft, contract)).toBe(false);
    expect(hasActiveOverrides({ ...draft, values: { iteration_cap: 50 } }, contract)).toBe(false);
    expect(hasActiveOverrides({ ...draft, values: { iteration_cap: 80 } }, contract)).toBe(true);
    expect(hasActiveOverrides({ ...draft, budgetOnExceeded: "escalate" }, contract)).toBe(true);
  });

  it("Should treat 0 as the default on an off-by-default budget field (no redundant override)", () => {
    // reviews-watch's budgets are off (defaultValue null), so 0 == off == default.
    const watchContract = loopDetailByName.get("reviews-watch")!.definition.contract;
    const zeroTokens = { values: { budget_tokens: 0 }, budgetOnExceeded: "halt" as const };
    expect(hasActiveOverrides(zeroTokens, watchContract)).toBe(false);
    expect(buildConfigOverrides(zeroTokens, watchContract)).toBeNull();
    expect(
      hasActiveOverrides(
        { values: { budget_tokens: 100 }, budgetOnExceeded: "halt" },
        watchContract
      )
    ).toBe(true);
  });

  it("Should project only changed fields into config_overrides, converting wall minutes to seconds", () => {
    const draft = initialOverrideDraft(contract);
    expect(buildConfigOverrides(draft, contract)).toBeNull();
    const overrides = buildConfigOverrides(
      { values: { iteration_cap: 80, budget_wall_sec: 120 }, budgetOnExceeded: "escalate" },
      contract
    );
    expect(overrides).toEqual({
      iteration_cap: 80,
      budget_wall_sec: 7_200,
      budget_on_exceeded: "escalate",
    });
  });
});
