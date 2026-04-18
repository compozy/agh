import type { PillVariant } from "@agh/ui";

/**
 * Legacy tone strings emitted by `*-formatters.ts` helpers across domain systems.
 * Maps the historical `design-system/pill` tone palette onto the new `@agh/ui`
 * `Pill`/`Pills` semantic variant system.
 */
export type LegacyPillTone =
  | "neutral"
  | "amber"
  | "green"
  | "violet"
  | "danger"
  | "accent"
  | "warning";

export function pillVariantFromTone(tone: LegacyPillTone | null | undefined): PillVariant {
  switch (tone) {
    case "amber":
    case "accent":
      return "accent";
    case "green":
      return "success";
    case "violet":
      return "info";
    case "danger":
      return "danger";
    case "warning":
      return "warning";
    case "neutral":
    default:
      return "default";
  }
}
