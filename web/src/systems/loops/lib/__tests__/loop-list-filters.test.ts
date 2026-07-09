import { describe, expect, it } from "vitest";

import type { FilterFieldConfig } from "@agh/ui/components/reui/filters";

import { loopCatalogFixtures } from "../../mocks/fixtures";
import type { LoopCatalogFilter } from "../loop-catalog";
import {
  applyLoopFilterChips,
  buildLoopFilterFields,
  filterLoopCatalog,
  filterLoopsByQuery,
  loopFiltersToChips,
  parseLoopCategoryFilter,
  parseLoopKindFilter,
  parseLoopStatusFilter,
} from "../loop-list-filters";

function asFieldConfigs(
  fields: ReturnType<typeof buildLoopFilterFields>
): FilterFieldConfig<string>[] {
  return fields as FilterFieldConfig<string>[];
}

describe("loop-list-filters", () => {
  it("Should build select fields from present categories and statuses", () => {
    const fields = asFieldConfigs(
      buildLoopFilterFields(["delivery", "watch"], ["running", "watching"])
    );
    expect(fields.map(field => field.key)).toEqual(["kind", "category", "status"]);
    const category = fields.find(field => field.key === "category");
    expect(category?.type).toBe("select");
    expect(category?.options?.map(option => option.value)).toEqual(["delivery", "watch"]);
  });

  it("Should round-trip chip state for kind, category, and status", () => {
    const chips = loopFiltersToChips({
      kind: "workspace",
      category: "delivery",
      status: "running",
    });
    expect(chips).toHaveLength(3);

    const next: LoopCatalogFilter = { kind: "all", category: null, status: null };
    applyLoopFilterChips(chips, {
      onKindChange: value => {
        next.kind = value;
      },
      onCategoryChange: value => {
        next.category = value;
      },
      onStatusChange: value => {
        next.status = value;
      },
    });
    expect(next).toEqual({ kind: "workspace", category: "delivery", status: "running" });
  });

  it("Should clear chip fields when chips are removed", () => {
    const next: LoopCatalogFilter = {
      kind: "workspace",
      category: "delivery",
      status: "running",
    };
    applyLoopFilterChips([], {
      onKindChange: value => {
        next.kind = value;
      },
      onCategoryChange: value => {
        next.category = value;
      },
      onStatusChange: value => {
        next.status = value;
      },
    });
    expect(next).toEqual({ kind: "all", category: null, status: null });
  });

  it("Should search by name and goal", () => {
    expect(filterLoopsByQuery(loopCatalogFixtures, "software")).toHaveLength(1);
    expect(filterLoopsByQuery(loopCatalogFixtures, "Ship the requested")).toHaveLength(1);
    expect(filterLoopsByQuery(loopCatalogFixtures, "zzz")).toHaveLength(0);
  });

  it("Should AND search with chip filters", () => {
    const filtered = filterLoopCatalog(loopCatalogFixtures, "watch", {
      kind: "read-only",
      category: null,
      status: null,
    });
    expect(filtered.map(entry => entry.name)).toEqual(["reviews-watch"]);
  });

  it("Should parse URL search values and reject unknowns", () => {
    expect(parseLoopKindFilter("workspace")).toBe("workspace");
    expect(parseLoopKindFilter("all")).toBeUndefined();
    expect(parseLoopKindFilter("builtin")).toBeUndefined();
    expect(parseLoopCategoryFilter(" delivery ")).toBe("delivery");
    expect(parseLoopCategoryFilter("")).toBeUndefined();
    expect(parseLoopStatusFilter("running")).toBe("running");
    expect(parseLoopStatusFilter("mystery")).toBeUndefined();
  });
});
