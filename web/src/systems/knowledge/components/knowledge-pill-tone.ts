import type { PillTone } from "@agh/ui";

import type { KnowledgeTone } from "../lib/knowledge-formatters";
import type { MemoryDecisionOp, MemoryDecisionSource } from "../types";

export function pillToneFromKnowledgeTone(tone: KnowledgeTone): PillTone {
  switch (tone) {
    case "user":
    case "feedback":
      return "accent";
    case "project":
      return "success";
    case "reference":
    case "workspace":
      return "info";
    case "agent":
      return "warning";
    case "global":
    default:
      return "neutral";
  }
}

export function pillToneFromDecisionOp(op: MemoryDecisionOp): PillTone {
  switch (op) {
    case "add":
      return "success";
    case "update":
      return "info";
    case "delete":
      return "danger";
    case "reject":
      return "warning";
    case "noop":
    default:
      return "neutral";
  }
}

export function pillToneFromDecisionSource(source: MemoryDecisionSource): PillTone {
  return source === "llm" ? "info" : "neutral";
}
