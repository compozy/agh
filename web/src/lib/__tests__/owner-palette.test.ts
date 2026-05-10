import { describe, expect, it } from "vitest";

import { AGENT_PALETTE, HUMAN_PALETTE, SYSTEM_PALETTE, ownerColor } from "../owner-palette";

describe("ownerColor", () => {
  it("Should be deterministic for the same (label, kind) pair", () => {
    const a = ownerColor("planner-prime", "agent");
    const b = ownerColor("planner-prime", "agent");
    expect(a).toBe(b);
  });

  it("Should pick from the correct palette per kind", () => {
    expect(AGENT_PALETTE).toContain(ownerColor("scout", "agent"));
    expect(HUMAN_PALETTE).toContain(ownerColor("pedro", "human"));
    expect(SYSTEM_PALETTE).toContain(ownerColor("daemon", "system"));
  });

  it("Should tolerate empty labels without throwing", () => {
    expect(() => ownerColor("", "agent")).not.toThrow();
    const result = ownerColor("", "agent");
    expect(AGENT_PALETTE).toContain(result);
  });

  it("Should distribute across the palette for a sample of labels", () => {
    const seen = new Set<string>();
    for (const label of ["alpha", "beta", "gamma", "delta", "epsilon", "zeta", "eta", "theta"]) {
      seen.add(ownerColor(label, "agent"));
    }
    expect(seen.size).toBeGreaterThan(1);
  });
});
