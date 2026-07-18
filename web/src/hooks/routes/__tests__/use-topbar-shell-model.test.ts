import { describe, expect, it } from "vitest";

import { collectCrumbs } from "../use-topbar-shell-model";
import type { TopbarRouteContext } from "@/types/topbar";

function matchWithTopbar(topbar: TopbarRouteContext) {
  return { context: { topbar } };
}

describe("collectCrumbs", () => {
  it("Should fold inherited Marketplace crumbs that share topbar reference identity", () => {
    const marketplaceTopbar: TopbarRouteContext = {
      crumb: { label: "Marketplace", to: "/marketplace" },
    };

    const crumbs = collectCrumbs([
      { context: {} },
      matchWithTopbar(marketplaceTopbar),
      matchWithTopbar(marketplaceTopbar),
      matchWithTopbar(marketplaceTopbar),
    ]);

    expect(crumbs).toEqual([{ label: "Marketplace", to: "/marketplace" }]);
  });

  it("Should emit duplicate Marketplace crumbs when each match gets a fresh topbar object", () => {
    const crumbs = collectCrumbs([
      { context: {} },
      matchWithTopbar({ crumb: { label: "Marketplace", to: "/marketplace" } }),
      matchWithTopbar({ crumb: { label: "Marketplace", to: "/marketplace" } }),
      matchWithTopbar({ crumb: { label: "Marketplace", to: "/marketplace" } }),
    ]);

    expect(crumbs).toHaveLength(3);
    expect(crumbs.every(crumb => crumb.label === "Marketplace")).toBe(true);
  });

  it("Should model redirect-mediated entry, exact kind crumbs, then clean sibling navigation", () => {
    const marketplaceTopbar: TopbarRouteContext = {
      crumb: { label: "Marketplace", to: "/marketplace" },
    };
    const skillsTopbar: TopbarRouteContext = { crumb: { label: "Skills" } };

    const redirectEntry = collectCrumbs([
      { context: {} },
      matchWithTopbar(marketplaceTopbar),
      matchWithTopbar(marketplaceTopbar),
    ]);
    expect(redirectEntry).toEqual([{ label: "Marketplace", to: "/marketplace" }]);

    const onSkills = collectCrumbs([
      { context: {} },
      matchWithTopbar(marketplaceTopbar),
      matchWithTopbar(skillsTopbar),
    ]);
    expect(onSkills).toEqual([{ label: "Marketplace", to: "/marketplace" }, { label: "Skills" }]);
    expect(onSkills.filter(crumb => crumb.label === "Marketplace")).toHaveLength(1);

    const onTriggers = collectCrumbs([
      { context: {} },
      matchWithTopbar({ crumb: { label: "Triggers", to: "/triggers" } }),
    ]);

    expect(onTriggers.some(crumb => crumb.label === "Marketplace")).toBe(false);
    expect(onTriggers).toEqual([{ label: "Triggers", to: "/triggers" }]);
  });
});
