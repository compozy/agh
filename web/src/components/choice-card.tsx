import type { ReactNode } from "react";
import { Check } from "lucide-react";
import type { LucideIcon } from "lucide-react";

import { cn } from "@agh/ui";

interface ChoiceCardProps {
  selected: boolean;
  onSelect: () => void;
  title: ReactNode;
  description?: ReactNode;
  icon?: LucideIcon;
  /** Trailing badge rendered under the description (e.g. a category pill). */
  badge?: ReactNode;
  className?: string;
  "data-testid"?: string;
  "aria-label"?: string;
}

/**
 * Accent-tint selectable card — the canonical "primary path" choice affordance
 * shared by the Job output-mode segments and the Task template grid. Mirrors the
 * shipped trigger `event-card.tsx` grammar: a flat resting card that fills with
 * `--accent-tint` + an inset `--accent-dim` ring when selected, with a check
 * glyph. This is deliberately NOT the glaze `RadioCard` (reserved for
 * lower-emphasis lists); accent-tint marks the active branch a user is choosing.
 */
export function ChoiceCard({
  selected,
  onSelect,
  title,
  description,
  icon: Icon,
  badge,
  className,
  "data-testid": testId,
  "aria-label": ariaLabel,
}: ChoiceCardProps) {
  return (
    <button
      aria-checked={selected}
      aria-label={ariaLabel}
      className={cn(
        "flex w-full flex-col gap-1.5 rounded-md border p-3 text-left transition-colors outline-none focus-visible:shadow-focus-ring",
        selected
          ? "border-transparent bg-accent-tint ring-1 ring-accent-dim ring-inset"
          : "border-line-soft bg-canvas-tint hover:border-line hover:bg-elevated",
        className
      )}
      data-testid={testId}
      onClick={onSelect}
      role="radio"
      type="button"
    >
      <span className="flex items-center gap-2.5">
        {Icon ? (
          <span
            className={cn(
              "flex size-7 shrink-0 items-center justify-center rounded",
              selected ? "bg-accent-tint-strong text-accent-strong" : "bg-badge-fill text-muted"
            )}
          >
            <Icon aria-hidden="true" className="size-4" />
          </span>
        ) : null}
        <span
          className={cn(
            "min-w-0 flex-1 text-form-label font-semibold tracking-tight",
            selected ? "text-accent-strong" : "text-fg-strong"
          )}
        >
          {title}
        </span>
        <Check
          aria-hidden="true"
          className={cn(
            "size-4 shrink-0 text-accent-strong transition-opacity",
            selected ? "opacity-100" : "opacity-0"
          )}
        />
      </span>
      {description ? (
        <span className="text-form-hint leading-snug text-subtle">{description}</span>
      ) : null}
      {badge ? <span className="mt-1 inline-flex self-start">{badge}</span> : null}
    </button>
  );
}
