import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import {
  compareKnowledgeScope,
  decisionOpLabel,
  decisionSourceLabel,
  formatKnowledgeDateTime,
  formatKnowledgeRelativeTime,
  knowledgeAgentTierLabel,
  knowledgeAgentTierShortLabel,
  knowledgeMemoryKey,
  knowledgeScopeLabel,
  knowledgeScopeShortLabel,
  memoryScopeTone,
  memoryTypeTone,
} from "../knowledge-formatters";

describe("knowledge-formatters", () => {
  it("Should derive a stable knowledge memory key from scope plus filename", () => {
    expect(knowledgeMemoryKey({ filename: "user.md", scope: "global", key: undefined })).toBe(
      "global:user.md"
    );
    expect(knowledgeMemoryKey({ filename: "user.md", scope: "agent", key: "custom-key" })).toBe(
      "custom-key"
    );
  });

  it("Should sort scopes with global before workspace before agent", () => {
    expect(compareKnowledgeScope("global", "workspace")).toBeLessThan(0);
    expect(compareKnowledgeScope("workspace", "global")).toBeGreaterThan(0);
    expect(compareKnowledgeScope("agent", "workspace")).toBeGreaterThan(0);
    expect(compareKnowledgeScope("global", "global")).toBe(0);
  });

  it("Should expose full and short scope labels for every Memory v2 scope", () => {
    expect(knowledgeScopeLabel("global")).toBe("GLOBAL");
    expect(knowledgeScopeLabel("workspace")).toBe("WORKSPACE");
    expect(knowledgeScopeLabel("agent")).toBe("AGENT");
    expect(knowledgeScopeShortLabel("global")).toBe("GLOBAL");
    expect(knowledgeScopeShortLabel("workspace")).toBe("WS");
    expect(knowledgeScopeShortLabel("agent")).toBe("AGENT");
  });

  it("Should expose agent tier labels", () => {
    expect(knowledgeAgentTierLabel("global")).toBe("AGENT-GLOBAL");
    expect(knowledgeAgentTierLabel("workspace")).toBe("AGENT-WORKSPACE");
    expect(knowledgeAgentTierShortLabel("global")).toBe("AG-GLOBAL");
    expect(knowledgeAgentTierShortLabel("workspace")).toBe("AG-WS");
  });

  it("Should map memory type to semantic knowledge tones", () => {
    expect(memoryTypeTone("user")).toBe("user");
    expect(memoryTypeTone("feedback")).toBe("feedback");
    expect(memoryTypeTone("project")).toBe("project");
    expect(memoryTypeTone("reference")).toBe("reference");
  });

  it("Should map memory scope to semantic knowledge tones", () => {
    expect(memoryScopeTone("global")).toBe("global");
    expect(memoryScopeTone("workspace")).toBe("workspace");
    expect(memoryScopeTone("agent")).toBe("agent");
  });

  it("Should expose decision op and source labels", () => {
    expect(decisionOpLabel("noop")).toBe("NOOP");
    expect(decisionOpLabel("add")).toBe("ADD");
    expect(decisionOpLabel("update")).toBe("UPDATE");
    expect(decisionOpLabel("delete")).toBe("DELETE");
    expect(decisionOpLabel("reject")).toBe("REJECT");
    expect(decisionSourceLabel("rule")).toBe("RULE");
    expect(decisionSourceLabel("llm")).toBe("LLM");
  });

  describe("formatKnowledgeRelativeTime", () => {
    beforeEach(() => {
      vi.useFakeTimers();
      vi.setSystemTime(new Date("2026-04-18T12:00:00Z"));
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    it("Should return 'just now' within the hour", () => {
      expect(formatKnowledgeRelativeTime("2026-04-18T11:45:00Z")).toBe("just now");
    });

    it("Should return hour-granular label within the day", () => {
      expect(formatKnowledgeRelativeTime("2026-04-18T08:00:00Z")).toBe("4h ago");
    });

    it("Should return day-granular label within the week", () => {
      expect(formatKnowledgeRelativeTime("2026-04-15T12:00:00Z")).toBe("3d ago");
    });

    it("Should fall back to an absolute month/day label for older dates", () => {
      const label = formatKnowledgeRelativeTime("2026-04-01T12:00:00Z");
      expect(label).toMatch(/Apr 1/);
    });

    it("Should pass invalid input through unchanged", () => {
      expect(formatKnowledgeRelativeTime("not-a-date")).toBe("not-a-date");
    });
  });

  describe("formatKnowledgeDateTime", () => {
    it("Should format a valid ISO string with month/day/year/time", () => {
      const label = formatKnowledgeDateTime("2026-04-09T10:00:00Z");
      expect(label).toMatch(/Apr 9, 2026/);
    });

    it("Should fall back to the original string for invalid input", () => {
      expect(formatKnowledgeDateTime("not-a-date")).toBe("not-a-date");
    });
  });
});
