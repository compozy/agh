"use client";

import { cva, type VariantProps } from "class-variance-authority";
import * as React from "react";

import { cn } from "../lib/utils";

/**
 * Pill = static semantic tag rendered as a span.
 * Pills = segmented toggle group — mirrors `.pills` + `.pill` in
 * `docs/design/web-inspiration/styles/app.css`. The segments live inside a
 * contained track (panel surface, 1px divider border, 3px inner padding).
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
  "inline-flex items-center justify-center gap-1.5 whitespace-nowrap font-mono uppercase font-semibold tracking-[0.08em] transition-colors duration-150 ease-out cursor-pointer focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-[color:var(--color-accent)] focus-visible:ring-offset-0 disabled:cursor-not-allowed disabled:opacity-50",
  {
    variants: {
      active: {
        true: "bg-[color:var(--color-surface-elevated)] text-[color:var(--color-text-primary)]",
        false:
          "bg-transparent text-[color:var(--color-text-tertiary)] hover:text-[color:var(--color-text-secondary)]",
      },
      size: {
        sm: "h-[20px] rounded-[5px] px-2 text-[10px]",
        md: "h-[22px] rounded-[5px] px-2.5 text-[10px]",
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
      role="group"
      className={cn(
        "inline-flex items-center gap-[2px] rounded-[8px] border border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] p-[3px]",
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
            data-slot="pills-item"
            data-value={item.value}
            data-active={isActive}
            data-testid={item.testId}
            aria-pressed={isActive}
            disabled={item.disabled}
            onClick={() => {
              if (isActive) return;
              onChange(item.value);
            }}
            className={pillToggleVariants({ active: isActive, size })}
          >
            <span>{item.label}</span>
            {typeof item.badge === "number" && item.badge > 0 ? (
              <span
                data-slot="pills-badge"
                className="inline-flex h-[14px] min-w-[14px] items-center justify-center rounded-[7px] bg-[color:var(--color-accent)] px-1 font-mono text-[9px] font-bold tabular-nums text-[color:var(--color-accent-ink)]"
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
