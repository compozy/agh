"use client";

import { cva, type VariantProps } from "class-variance-authority";
import * as React from "react";

import { cn } from "../../lib/utils";

const pillGroupSegmentVariants = cva(
  "inline-flex cursor-pointer items-center justify-center gap-1.5 whitespace-nowrap rounded-(--radius-chip) font-mono text-badge font-semibold uppercase tracking-(--tracking-badge) transition-colors duration-(--duration-base) ease-(--ease-out) focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-(--color-accent) focus-visible:ring-offset-0 disabled:cursor-not-allowed disabled:opacity-50",
  {
    variants: {
      active: {
        true: "bg-(--color-surface-elevated) text-(--color-text-primary)",
        false: "bg-transparent text-(--color-text-tertiary) hover:text-(--color-text-secondary)",
      },
      size: {
        sm: "h-(--height-pill-group-segment-sm) px-(--space-pill-group-segment-sm-x)",
        md: "h-(--height-mono-badge) px-(--space-pill-group-segment-md-x)",
      },
    },
    defaultVariants: {
      active: false,
      size: "md",
    },
  }
);

export type PillGroupSize = NonNullable<VariantProps<typeof pillGroupSegmentVariants>["size"]>;

export interface PillGroupItem<V extends string = string> {
  value: V;
  label: React.ReactNode;
  /** Optional unread / count badge rendered inside the segment. */
  badge?: number;
  disabled?: boolean;
  testId?: string;
}

export interface PillGroupProps<V extends string = string> extends Omit<
  React.ComponentProps<"div">,
  "onChange"
> {
  items: ReadonlyArray<PillGroupItem<V>>;
  value: V;
  onChange: (next: V) => void;
  size?: PillGroupSize;
}

function PillGroup<V extends string = string>({
  items,
  value,
  onChange,
  size = "md",
  className,
  ...props
}: PillGroupProps<V>) {
  return (
    <div
      data-slot="pill-group"
      role="group"
      className={cn(
        "inline-flex items-center gap-(--space-pill-group-track-gap) rounded-(--radius) border border-(--color-divider) bg-(--color-surface-panel) p-(--space-pill-group-track-padding)",
        className
      )}
      {...props}
    >
      {items.map(item => {
        const isActive = item.value === value;
        return (
          <button
            key={item.value}
            type="button"
            data-slot="pill-group-item"
            data-value={item.value}
            data-active={isActive}
            data-testid={item.testId}
            aria-pressed={isActive}
            disabled={item.disabled}
            onClick={() => {
              if (isActive) return;
              onChange(item.value);
            }}
            className={pillGroupSegmentVariants({ active: isActive, size })}
          >
            <span>{item.label}</span>
            {typeof item.badge === "number" && item.badge > 0 ? (
              <span
                data-slot="pill-group-badge"
                className="inline-flex h-(--size-pill-group-badge) min-w-(--size-pill-group-badge) items-center justify-center rounded-full bg-(--color-accent) px-(--space-pill-group-badge-x) font-mono text-[var(--text-pill-group-badge)] font-bold tabular-nums text-(--color-accent-ink)"
              >
                {item.badge}
              </span>
            ) : null}
          </button>
        );
      })}
    </div>
  );
}

export { PillGroup, pillGroupSegmentVariants };
