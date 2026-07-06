import { describe, expect, it } from "vitest";

import { buildCheckDescriptors } from "../loop-config-checks";
import {
  buildConfigureModel,
  buildLoopConfigRequest,
  initialConfigDraft,
  resetConfigDraft,
} from "../loop-config-draft";
import { loopConfigFixture, loopDetailByName } from "../../mocks/fixtures";
import type { LoopConfig } from "../../types";

const delivery = loopDetailByName.get("software-delivery")!;
const watch = loopDetailByName.get("reviews-watch")!;
const contract = delivery.definition.contract;
const descriptors = buildCheckDescriptors(contract);

describe("initialConfigDraft", () => {
  it("Should open on inherited defaults when no config is stored", () => {
    const draft = initialConfigDraft(contract, descriptors, null);
    expect(draft.humanGateEnabled).toBe(false);
    expect(draft.reattemptStrategy).toBe("failed_only");
    expect(draft.limits.values).toEqual({});
    expect(draft.limits.budgetOnExceeded).toBe("halt");
    expect(draft.checks.verify_build).toEqual({ enabled: true, command: "npm test" });
  });

  it("Should seed the draft from a stored config, converting wall-clock seconds to minutes", () => {
    const stored: LoopConfig = {
      ...loopConfigFixture,
      budget_wall_sec: 7_200,
      reattempt_strategy: "full_body",
    };
    const draft = initialConfigDraft(contract, descriptors, stored);
    expect(draft.humanGateEnabled).toBe(true);
    expect(draft.reattemptStrategy).toBe("full_body");
    expect(draft.limits.budgetOnExceeded).toBe("escalate");
    expect(draft.limits.values.iteration_cap).toBe(16);
    expect(draft.limits.values.budget_tokens).toBe(750_000);
    expect(draft.limits.values.budget_wall_sec).toBe(120);
  });

  it("Should clamp a stored value that exceeds its ceiling", () => {
    const draft = initialConfigDraft(contract, descriptors, {
      ...loopConfigFixture,
      iteration_cap: 5_000,
      gate_max_revisions: 99,
    });
    expect(draft.limits.values.iteration_cap).toBe(100);
    expect(draft.limits.values.gate_max_revisions).toBe(10);
  });
});

describe("resetConfigDraft", () => {
  it("Should restore the inherited defaults", () => {
    const draft = resetConfigDraft(contract, descriptors);
    expect(draft.humanGateEnabled).toBe(false);
    expect(draft.reattemptStrategy).toBe("failed_only");
    expect(draft.limits.values).toEqual({});
    expect(draft.checks.verify_build).toEqual({ enabled: true, command: "npm test" });
  });
});

describe("buildLoopConfigRequest", () => {
  it("Should inherit untouched limits and checks (blank = null, enabled+declared omitted)", () => {
    const draft = initialConfigDraft(contract, descriptors, null);
    const { config } = buildLoopConfigRequest(draft, descriptors);
    expect(config.iteration_cap).toBeNull();
    expect(config.budget_tokens).toBeNull();
    expect(config.budget_on_exceeded).toBe("halt");
    expect(config.human_gate_enabled).toBe(false);
    expect(config.reattempt_strategy).toBe("failed_only");
    expect(config.enabled_checks_json).toBeNull();
  });

  it("Should carry overridden limits, toggles, and disabled checks", () => {
    const draft = initialConfigDraft(contract, descriptors, null);
    draft.limits.values.iteration_cap = 80;
    draft.humanGateEnabled = true;
    draft.reattemptStrategy = "full_body";
    draft.checks.verify_build = { enabled: false, command: "go test ./..." };
    const { config } = buildLoopConfigRequest(draft, descriptors);
    expect(config.iteration_cap).toBe(80);
    expect(config.human_gate_enabled).toBe(true);
    expect(config.reattempt_strategy).toBe("full_body");
    expect(config.enabled_checks_json).toEqual({
      verify_build: { enabled: false, command: "go test ./..." },
    });
  });

  it("Should pin a typed numeric even when it equals the definition default (R-004)", () => {
    const draft = initialConfigDraft(contract, descriptors, null);
    // The delivery contract's no_progress.window default is 3; typing it must pin, not inherit.
    draft.limits.values.no_progress_window = 3;
    const { config } = buildLoopConfigRequest(draft, descriptors);
    expect(config.no_progress_window).toBe(3);
  });

  it("Should convert the wall-clock minutes field back to seconds", () => {
    const draft = initialConfigDraft(contract, descriptors, null);
    draft.limits.values.budget_wall_sec = 120;
    const { config } = buildLoopConfigRequest(draft, descriptors);
    expect(config.budget_wall_sec).toBe(7_200);
  });

  it("Should NEVER emit a cost field (cost is display-only)", () => {
    const draft = initialConfigDraft(contract, descriptors, loopConfigFixture);
    const { config } = buildLoopConfigRequest(draft, descriptors);
    expect(config).not.toHaveProperty("budget_usd");
    expect(config).not.toHaveProperty("cost");
    expect(config).not.toHaveProperty("cost_cap");
  });
});

describe("buildConfigureModel", () => {
  it("Should return descriptors and a seeded draft for a watch loop with no checks", () => {
    const model = buildConfigureModel(watch.definition.contract, null);
    expect(model.descriptors).toEqual([]);
    expect(model.draft.reattemptStrategy).toBe("failed_only");
  });
});
