import { cn } from "@agh/ui";

/**
 * Signal-strength intensity meter for reasoning effort (design `.meter`). Seven
 * ascending bars map to the canonical effort positions; `position` (1..7) fills
 * bars with the accent, `hollow` renders every bar faint for the
 * provider-default state.
 */
const BAR_HEIGHTS = ["h-px", "h-0.5", "h-1", "h-1.5", "h-2", "h-2.5", "h-3"] as const;

export interface IntensityMeterProps {
  /** 1-based fill count (0 = none filled). */
  position: number;
  /** When true, every bar renders in the faint "default" tone. */
  hollow?: boolean;
  className?: string;
}

export function IntensityMeter({ position, hollow = false, className }: IntensityMeterProps) {
  const filledCount = hollow ? 0 : Math.max(0, Math.min(BAR_HEIGHTS.length, position));
  return (
    <span
      aria-hidden="true"
      data-slot="runtime-intensity-meter"
      data-position={filledCount}
      data-hollow={hollow ? "true" : "false"}
      className={cn("inline-flex h-3 items-end gap-0.5", className)}
    >
      {BAR_HEIGHTS.map((heightClass, index) => {
        const filled = !hollow && index + 1 <= position;
        return (
          <span
            key={heightClass}
            data-fill={hollow ? "hollow" : filled ? "on" : "off"}
            className={cn(
              "w-0.5 rounded-full",
              heightClass,
              hollow ? "bg-line-strong" : filled ? "bg-accent-strong" : "bg-faint"
            )}
          />
        );
      })}
    </span>
  );
}
