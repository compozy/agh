"use client";

import { cva, type VariantProps } from "class-variance-authority";
import * as React from "react";

import { cn } from "../lib/utils";

const pillGroupSegmentVariants = cva(
  "inline-flex cursor-pointer items-center justify-center gap-1.5 whitespace-nowrap rounded-[5px] font-mono text-[10px] font-semibold uppercase tracking-[0.08em] transition-colors duration-150 ease-out focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-(--color-accent) focus-visible:ring-offset-0 disabled:cursor-not-allowed disabled:opacity-50",
  {
    variants: {
      active: {
        true: "bg-(--color-surface-elevated) text-(--color-text-primary)",
        false: "bg-transparent text-(--color-text-tertiary) hover:text-(--color-text-secondary)",
      },
      size: {
        sm: "h-[20px] px-2",
        md: "h-[22px] px-2.5",
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
        "inline-flex items-center gap-[2px] rounded-[8px] border border-(--color-divider) bg-(--color-surface-panel) p-[3px]",
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
                className="inline-flex h-[14px] min-w-[14px] items-center justify-center rounded-[7px] bg-(--color-accent) px-1 font-mono text-[9px] font-bold tabular-nums text-(--color-accent-ink)"
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
