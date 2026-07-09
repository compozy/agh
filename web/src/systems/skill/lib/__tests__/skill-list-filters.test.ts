import type { Filter } from "@agh/ui/components/reui/filters";
import { describe, expect, it, vi } from "vitest";

import type { SkillPayload } from "../../types";
import {
  applySkillFilterChips,
  filterInstalledSkills,
  parseSkillEnabledFilter,
  parseSkillSourceFilter,
  skillFiltersToChips,
} from "../skill-list-filters";

function makeSkill(overrides: Partial<SkillPayload> = {}): SkillPayload {
  return {
    name: "test-skill",
    description: "desc",
    source: "bundled",
    enabled: true,
    dir: "/path/to/skill",
    ...overrides,
  };
}

const SKILLS: SkillPayload[] = [
  makeSkill({ name: "ws-tool", source: "workspace", enabled: true }),
  makeSkill({ name: "alpha", source: "bundled", enabled: true }),
  makeSkill({ name: "beta", source: "bundled", enabled: false }),
  makeSkill({ name: "mp-plugin", source: "marketplace", enabled: true }),
];

describe("filterInstalledSkills", () => {
  it("Should return every skill sorted by name when no query or filters are set", () => {
    const result = filterInstalledSkills(SKILLS, "", { source: null, enabled: null });
    expect(result.map(skill => skill.name)).toEqual(["alpha", "beta", "mp-plugin", "ws-tool"]);
  });

  it("Should match name, description, and tags case-insensitively", () => {
    const tagged = [
      ...SKILLS,
      makeSkill({ name: "tagged", description: "another", metadata: { tags: ["DATABASE"] } }),
    ];
    expect(
      filterInstalledSkills(tagged, "ALPHA", { source: null, enabled: null }).map(s => s.name)
    ).toEqual(["alpha"]);
    expect(
      filterInstalledSkills(tagged, "database", { source: null, enabled: null }).map(s => s.name)
    ).toEqual(["tagged"]);
  });

  it("Should filter by source and by enabled state independently", () => {
    expect(
      filterInstalledSkills(SKILLS, "", { source: "workspace", enabled: null }).map(s => s.name)
    ).toEqual(["ws-tool"]);
    expect(
      filterInstalledSkills(SKILLS, "", { source: null, enabled: "false" }).map(s => s.name)
    ).toEqual(["beta"]);
    expect(
      filterInstalledSkills(SKILLS, "", { source: "bundled", enabled: "true" }).map(s => s.name)
    ).toEqual(["alpha"]);
  });
});

describe("skillFiltersToChips", () => {
  it("Should emit no chips when nothing is filtered", () => {
    expect(skillFiltersToChips({ source: null, enabled: null })).toEqual([]);
  });

  it("Should emit one 'is' chip per active filter", () => {
    const chips = skillFiltersToChips({ source: "workspace", enabled: "false" });
    expect(chips).toHaveLength(2);
    const source = chips.find(chip => chip.field === "source");
    expect(source?.id).toBe("skill-filter-source");
    expect(source?.operator).toBe("is");
    expect(source?.values).toEqual(["workspace"]);
    expect(chips.find(chip => chip.field === "enabled")?.values).toEqual(["false"]);
  });
});

describe("applySkillFilterChips", () => {
  it("Should route recognized chip values to the matching handler", () => {
    const onSourceChange = vi.fn();
    const onEnabledChange = vi.fn();
    applySkillFilterChips(skillFiltersToChips({ source: "marketplace", enabled: "true" }), {
      onSourceChange,
      onEnabledChange,
    });
    expect(onSourceChange).toHaveBeenCalledWith("marketplace");
    expect(onEnabledChange).toHaveBeenCalledWith("true");
  });

  it("Should clear filters to null when their chips are absent", () => {
    const onSourceChange = vi.fn();
    const onEnabledChange = vi.fn();
    applySkillFilterChips([], { onSourceChange, onEnabledChange });
    expect(onSourceChange).toHaveBeenCalledWith(null);
    expect(onEnabledChange).toHaveBeenCalledWith(null);
  });

  it("Should coerce an unknown source chip value to null", () => {
    const onSourceChange = vi.fn();
    const onEnabledChange = vi.fn();
    const chips: Filter<string>[] = [
      { id: "skill-filter-source", field: "source", operator: "is", values: ["bogus"] },
    ];
    applySkillFilterChips(chips, { onSourceChange, onEnabledChange });
    expect(onSourceChange).toHaveBeenCalledWith(null);
    expect(onEnabledChange).toHaveBeenCalledWith(null);
  });
});

describe("parseSkillSourceFilter / parseSkillEnabledFilter", () => {
  it("Should accept known source values and reject the rest", () => {
    expect(parseSkillSourceFilter("workspace")).toBe("workspace");
    expect(parseSkillSourceFilter("bundled")).toBe("bundled");
    expect(parseSkillSourceFilter("bogus")).toBeUndefined();
    expect(parseSkillSourceFilter(42)).toBeUndefined();
    expect(parseSkillSourceFilter(undefined)).toBeUndefined();
  });

  it("Should accept only the boolean-string enabled values", () => {
    expect(parseSkillEnabledFilter("true")).toBe("true");
    expect(parseSkillEnabledFilter("false")).toBe("false");
    expect(parseSkillEnabledFilter("yes")).toBeUndefined();
    expect(parseSkillEnabledFilter(null)).toBeUndefined();
  });
});
