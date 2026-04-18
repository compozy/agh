"use client";

import { cva, type VariantProps } from "class-variance-authority";
import * as React from "react";

import { cn } from "../lib/utils";

/**
 * Pill = static semantic tag rendered as a span.
 * Pills = segmented toggle group — follows the mock at `docs/design/web-inspiration/src/primitives.jsx`.
 * Both live in the same file because Pills renders pill-styled buttons, and keeping them
 * colocated avoids duplicating the variant table.
 */

const pillBase =
  "inline-flex items-center justify-center gap-1.5 whitespace-nowrap font-mono uppercase transition-colors duration-150 ease-out focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[color:var(--color-accent)] focus-visible:ring-offset-0";

const pillVariants = cva(pillBase, {
  variants: {
    variant: {
      default:
        "border border-[color:var(--color-divider)] bg-[color:var(--color-neutral-tint)] text-[color:var(--color-text-secondary)]",
      accent:
        "border-transparent bg-[color:var(--color-accent-tint)] text-[color:var(--color-accent)]",
      success:
        "border-transparent bg-[color:var(--color-success-tint)] text-[color:var(--color-success)]",
      warning:
        "border-transparent bg-[color:var(--color-warning-tint)] text-[color:var(--color-warning)]",
      danger:
        "border-transparent bg-[color:var(--color-danger-tint)] text-[color:var(--color-danger)]",
      info: "border-transparent bg-[color:var(--color-info-tint)] text-[color:var(--color-info)]",
    },
    size: {
      sm: "h-[22px] rounded-[var(--radius-mono-badge)] px-2 text-[10px] font-semibold tracking-[0.08em]",
      md: "h-8 rounded-[var(--radius-xl,20px)] px-3.5 text-[11px] font-semibold tracking-[0.12em]",
    },
  },
  defaultVariants: {
    variant: "default",
    size: "sm",
  },
});

export type PillVariant = NonNullable<VariantProps<typeof pillVariants>["variant"]>;
export type PillSize = NonNullable<VariantProps<typeof pillVariants>["size"]>;

export interface PillProps
  extends React.ComponentProps<"span">, VariantProps<typeof pillVariants> {}

function Pill({ className, variant, size, ...props }: PillProps) {
  return (
    <span
      data-slot="pill"
      data-variant={variant ?? "default"}
      data-size={size ?? "sm"}
      className={cn(pillVariants({ variant, size }), className)}
      {...props}
    />
  );
}

const pillToggleVariants = cva(
  `${pillBase} cursor-pointer border bg-transparent text-[color:var(--color-text-secondary)] hover:text-[color:var(--color-text-primary)] disabled:cursor-not-allowed disabled:opacity-50`,
  {
    variants: {
      active: {
        true: "border-[color:var(--color-accent)] bg-[color:var(--color-accent)] text-white hover:text-white",
        false:
          "border-[color:var(--color-divider)] hover:border-[color:var(--color-text-label)] hover:bg-[color:var(--color-hover)]",
      },
      size: {
        sm: "h-[22px] rounded-[var(--radius-mono-badge)] px-2 text-[10px] font-semibold tracking-[0.08em]",
        md: "h-8 rounded-[var(--radius-xl,20px)] px-3.5 text-[11px] font-semibold tracking-[0.12em]",
      },
    },
    defaultVariants: {
      active: false,
      size: "md",
    },
  }
);

export interface PillsItem<V extends string = string> {
  value: V;
  label: React.ReactNode;
  badge?: number;
  disabled?: boolean;
  testId?: string;
}

export interface PillsProps<V extends string = string> extends Omit<
  React.ComponentProps<"div">,
  "onChange"
> {
  items: ReadonlyArray<PillsItem<V>>;
  value: V;
  onChange: (next: V) => void;
  size?: PillSize;
  "aria-label"?: string;
}

function Pills<V extends string = string>({
  items,
  value,
  onChange,
  size = "md",
  className,
  ...props
}: PillsProps<V>) {
  return (
    <div
      data-slot="pills"
      role="tablist"
      className={cn("inline-flex flex-wrap items-center gap-1.5", className)}
      {...props}
    >
      {items.map(item => {
        const isActive = item.value === value;
        return (
          <button
            key={item.value}
            type="button"
            role="tab"
            data-slot="pills-item"
            data-value={item.value}
            data-active={isActive}
            data-testid={item.testId}
            aria-selected={isActive}
            aria-pressed={isActive}
            disabled={item.disabled}
            onClick={() => onChange(item.value)}
            className={pillToggleVariants({ active: isActive, size })}
          >
            <span>{item.label}</span>
            {typeof item.badge === "number" && item.badge > 0 ? (
              <span
                data-slot="pills-badge"
                className={cn(
                  "inline-flex min-w-[18px] items-center justify-center rounded-full px-1.5 text-[10px] font-semibold tabular-nums",
                  isActive
                    ? "bg-white/20 text-white"
                    : "bg-[color:var(--color-warning)] text-[color:var(--color-accent-ink)]"
                )}
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

export { Pill, Pills, pillVariants, pillToggleVariants };
