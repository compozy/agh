import type { PillTone } from "@agh/ui";

import type { KnowledgeTone } from "../lib/knowledge-formatters";

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
    case "global":
    default:
      return "neutral";
  }
}
