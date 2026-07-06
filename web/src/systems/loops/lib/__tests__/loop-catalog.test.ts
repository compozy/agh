import { describe, expect, it } from "vitest";

import { loopCatalogFixtures } from "../../mocks/fixtures";
import type { LoopCatalogEntry } from "../../types";
import {
  countByKind,
  groupLoopCatalog,
  hasHumanGate,
  isUnboundedCap,
  iterationCapLabel,
  loopCategories,
  loopCategory,
  loopInputCount,
  loopKind,
  matchesLoopFilter,
  successRateLabel,
} from "../loop-catalog";

const [delivery, watch] = loopCatalogFixtures;

describe("loop-catalog", () => {
  it("Should classify read-only vs workspace loops by source", () => {
    expect(loopKind({ source: "marketplace" })).toBe("read-only");
    expect(loopKind({ source: "workspace" })).toBe("workspace");
    expect(loopKind(delivery)).toBe("workspace");
    expect(loopKind(watch)).toBe("read-only");
  });

  it("Should derive only categories actually present, sorted (no invented taxonomy)", () => {
    expect(loopCategory(delivery)).toBe("delivery");
    expect(loopCategories(loopCatalogFixtures)).toEqual(["delivery", "watch"]);
    const blank: LoopCatalogEntry = {
      ...delivery,
      catalog: { ...delivery.catalog, category: "  " },
    };
    expect(loopCategory(blank)).toBeNull();
  });

  it("Should count declared inputs and detect unbounded watch loops", () => {
    expect(loopInputCount(delivery)).toBe(2);
    expect(loopInputCount(watch)).toBe(0);
    expect(isUnboundedCap(watch)).toBe(true);
    expect(isUnboundedCap(delivery)).toBe(false);
  });

  it("Should render iteration cap 0 as the unbounded glyph", () => {
    expect(iterationCapLabel(0)).toBe("∞");
    expect(iterationCapLabel(50)).toBe("50");
  });

  it("Should format a 0..1 success rate as a whole percent", () => {
    expect(successRateLabel(0.9)).toBe("90%");
    expect(successRateLabel(1)).toBe("100%");
    expect(successRateLabel(Number.NaN)).toBe("—");
  });

  it("Should detect a human gate from verification criteria", () => {
    expect(hasHumanGate(delivery)).toBe(false);
    const withHuman: LoopCatalogEntry = {
      ...delivery,
      contract: {
        ...delivery.contract,
        verification: [{ id: "approve", type: "human" }],
      },
    };
    expect(hasHumanGate(withHuman)).toBe(true);
  });

  it("Should filter by kind and category and count candidates per kind", () => {
    expect(matchesLoopFilter(delivery, { kind: "workspace", category: null })).toBe(true);
    expect(matchesLoopFilter(delivery, { kind: "read-only", category: null })).toBe(false);
    expect(matchesLoopFilter(delivery, { kind: "all", category: "delivery" })).toBe(true);
    expect(matchesLoopFilter(delivery, { kind: "all", category: "watch" })).toBe(false);
    expect(countByKind(loopCatalogFixtures, "all")).toBe(2);
    expect(countByKind(loopCatalogFixtures, "read-only")).toBe(1);
    expect(countByKind(loopCatalogFixtures, "workspace")).toBe(1);
  });

  it("Should group into read-only/workspace and drop empty groups", () => {
    const all = groupLoopCatalog(loopCatalogFixtures, { kind: "all", category: null });
    expect(all.map(group => group.kind)).toEqual(["read-only", "workspace"]);
    expect(all[0].entries).toHaveLength(1);

    const readOnlyOnly = groupLoopCatalog(loopCatalogFixtures, {
      kind: "read-only",
      category: null,
    });
    expect(readOnlyOnly).toHaveLength(1);
    expect(readOnlyOnly[0].kind).toBe("read-only");

    const none = groupLoopCatalog(loopCatalogFixtures, { kind: "workspace", category: "watch" });
    expect(none).toHaveLength(0);
  });
});
