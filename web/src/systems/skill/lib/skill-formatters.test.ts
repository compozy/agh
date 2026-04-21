import { describe, expect, it } from "vitest";

import type { SkillPayload } from "../types";

import {
  compareSkillSource,
  deriveSkillAuthor,
  deriveSkillCapabilities,
  deriveSkillRecentCalls,
  deriveSkillTags,
  filterSkillsByQuery,
  formatSkillRelativeTime,
  matchesMarketplaceCategory,
  skillSourceTone,
  skillStatusTone,
} from "./skill-formatters";

function makeSkill(overrides: Partial<SkillPayload> = {}): SkillPayload {
  return {
    name: "test-skill",
    description: "desc",
    source: "bundled",
    enabled: true,
    dir: "/path",
    ...overrides,
  };
}

describe("skill-formatters", () => {
  it("Should order sources as bundled → workspace → marketplace → user → additional", () => {
    const sorted = ["additional", "user", "marketplace", "workspace", "bundled"].sort(
      compareSkillSource
    );
    expect(sorted).toEqual(["bundled", "workspace", "marketplace", "user", "additional"]);
  });

  it("Should map sources to MonoBadge tones", () => {
    expect(skillSourceTone("bundled")).toBe("success");
    expect(skillSourceTone("workspace")).toBe("info");
    expect(skillSourceTone("marketplace")).toBe("accent");
    expect(skillSourceTone("user")).toBe("warning");
    expect(skillSourceTone("additional")).toBe("neutral");
    expect(skillSourceTone("unknown")).toBe("neutral");
  });

  it("Should map enabled flag to status dot tone", () => {
    expect(skillStatusTone(true)).toBe("success");
    expect(skillStatusTone(false)).toBe("neutral");
  });

  it("Should prefer provenance slug for author, falling back to metadata.author", () => {
    expect(deriveSkillAuthor(makeSkill())).toBeUndefined();
    expect(
      deriveSkillAuthor(
        makeSkill({
          provenance: {
            slug: "compozy",
            registry: "official",
            installed_at: "",
            version: "1",
          },
        })
      )
    ).toBe("compozy");
    expect(
      deriveSkillAuthor(
        makeSkill({
          provenance: {
            slug: "workspace",
            registry: "workspace",
            installed_at: "",
            version: "1",
          },
          metadata: { author: "pedronauck" },
        })
      )
    ).toBe("pedronauck");
  });

  it("Should filter tags down to string entries", () => {
    expect(
      deriveSkillTags(makeSkill({ metadata: { tags: ["a", 1, null, "b"] as unknown as string[] } }))
    ).toEqual(["a", "b"]);
  });

  it("Should match marketplace category against tags (case-insensitive)", () => {
    const skill = makeSkill({ metadata: { tags: ["Testing", "AI"] } });
    expect(matchesMarketplaceCategory(skill, "ALL")).toBe(true);
    expect(matchesMarketplaceCategory(skill, "TESTING")).toBe(true);
    expect(matchesMarketplaceCategory(skill, "AI")).toBe(true);
    expect(matchesMarketplaceCategory(skill, "DATABASE")).toBe(false);
  });

  it("Should filter skills by name, description, and tags", () => {
    const skills: SkillPayload[] = [
      makeSkill({ name: "alpha", description: "first", metadata: { tags: ["testing"] } }),
      makeSkill({ name: "beta", description: "second" }),
    ];
    expect(filterSkillsByQuery(skills, "")).toHaveLength(2);
    expect(filterSkillsByQuery(skills, "alpha")).toHaveLength(1);
    expect(filterSkillsByQuery(skills, "SECOND")).toHaveLength(1);
    expect(filterSkillsByQuery(skills, "testing")).toHaveLength(1);
    expect(filterSkillsByQuery(skills, "zzz")).toHaveLength(0);
  });

  it("Should parse capabilities metadata into string array", () => {
    expect(
      deriveSkillCapabilities(
        makeSkill({
          metadata: { capabilities: ["shell.run", 42, "git.stage"] as unknown as string[] },
        })
      )
    ).toEqual(["shell.run", "git.stage"]);
    expect(deriveSkillCapabilities(makeSkill())).toEqual([]);
  });

  it("Should parse recent_calls metadata, defaulting status to success", () => {
    const calls = deriveSkillRecentCalls(
      makeSkill({
        metadata: {
          recent_calls: [
            { label: "skill.run", status: "success", timestamp: "2026-04-17T12:00:00Z" },
            { label: "skill.fail", status: "error" },
            { label: "skill.pending" },
            { status: "success" },
          ],
        },
      })
    );
    expect(calls).toEqual([
      { label: "skill.run", status: "success", timestamp: "2026-04-17T12:00:00Z" },
      { label: "skill.fail", status: "error", timestamp: undefined },
      { label: "skill.pending", status: "success", timestamp: undefined },
    ]);
  });

  it("Should format relative time (minutes, hours, days)", () => {
    const now = Date.now();
    const minutesAgo = new Date(now - 5 * 60 * 1000).toISOString();
    const hoursAgo = new Date(now - 3 * 60 * 60 * 1000).toISOString();
    const daysAgo = new Date(now - 2 * 24 * 60 * 60 * 1000).toISOString();
    expect(formatSkillRelativeTime(minutesAgo)).toBe("5m ago");
    expect(formatSkillRelativeTime(hoursAgo)).toBe("3h ago");
    expect(formatSkillRelativeTime(daysAgo)).toBe("2d ago");
    expect(formatSkillRelativeTime("not-a-date")).toBe("not-a-date");
  });
});
