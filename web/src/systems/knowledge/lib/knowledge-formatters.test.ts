import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import {
  compareKnowledgeScope,
  deriveScopeFromFilename,
  formatKnowledgeDateTime,
  formatKnowledgeRelativeTime,
  knowledgeScopeLabel,
  knowledgeScopeShortLabel,
  memoryScopeTone,
  memoryTypeTone,
} from "./knowledge-formatters";

describe("knowledge-formatters", () => {
  it("derives scope from filename prefix", () => {
    expect(deriveScopeFromFilename("workspace/project.md")).toBe("workspace");
    expect(deriveScopeFromFilename("ws/ref_api.md")).toBe("workspace");
    expect(deriveScopeFromFilename("global/user-role.md")).toBe("global");
    expect(deriveScopeFromFilename("random_filename.md")).toBe("global");
  });

  it("sorts scope ordering with global before workspace", () => {
    expect(compareKnowledgeScope("global", "workspace")).toBeLessThan(0);
    expect(compareKnowledgeScope("workspace", "global")).toBeGreaterThan(0);
    expect(compareKnowledgeScope("global", "global")).toBe(0);
  });

  it("exposes full + short scope labels", () => {
    expect(knowledgeScopeLabel("global")).toBe("GLOBAL");
    expect(knowledgeScopeLabel("workspace")).toBe("WORKSPACE");
    expect(knowledgeScopeShortLabel("global")).toBe("GLOBAL");
    expect(knowledgeScopeShortLabel("workspace")).toBe("WS");
  });

  it("maps memory type → MonoBadge tone", () => {
    expect(memoryTypeTone("user")).toBe("accent");
    expect(memoryTypeTone("feedback")).toBe("accent");
    expect(memoryTypeTone("project")).toBe("success");
    expect(memoryTypeTone("reference")).toBe("info");
  });

  it("maps memory scope → MonoBadge tone", () => {
    expect(memoryScopeTone("global")).toBe("neutral");
    expect(memoryScopeTone("workspace")).toBe("info");
  });

  describe("formatKnowledgeRelativeTime", () => {
    beforeEach(() => {
      vi.useFakeTimers();
      vi.setSystemTime(new Date("2026-04-18T12:00:00Z"));
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    it("returns 'just now' within the hour", () => {
      expect(formatKnowledgeRelativeTime("2026-04-18T11:45:00Z")).toBe("just now");
    });

    it("returns hour-granular label within the day", () => {
      expect(formatKnowledgeRelativeTime("2026-04-18T08:00:00Z")).toBe("4h ago");
    });

    it("returns day-granular label within the week", () => {
      expect(formatKnowledgeRelativeTime("2026-04-15T12:00:00Z")).toBe("3d ago");
    });

    it("falls back to an absolute month/day label for older dates", () => {
      const label = formatKnowledgeRelativeTime("2026-04-01T12:00:00Z");
      expect(label).toMatch(/Apr 1/);
    });

    it("passes invalid input through unchanged", () => {
      expect(formatKnowledgeRelativeTime("not-a-date")).toBe("not-a-date");
    });
  });

  describe("formatKnowledgeDateTime", () => {
    it("formats a valid ISO string with month/day/year/time", () => {
      const label = formatKnowledgeDateTime("2026-04-09T10:00:00Z");
      expect(label).toMatch(/Apr 9, 2026/);
    });

    it("falls back to the original string for invalid input", () => {
      expect(formatKnowledgeDateTime("not-a-date")).toBe("not-a-date");
    });
  });
});
