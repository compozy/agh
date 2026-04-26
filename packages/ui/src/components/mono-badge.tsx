"use client";

import { cva, type VariantProps } from "class-variance-authority";
import * as React from "react";

import { cn } from "../lib/utils";

const monoBadgeVariants = cva(
  [
    "inline-flex items-center rounded-[var(--radius-mono-badge)] px-1.5 py-0.5",
    "font-mono text-[11px] font-medium leading-[14px] tracking-[0.06em] whitespace-nowrap",
  ].join(" "),
  {
    variants: {
      tone: {
        default:
          "border border-[color:var(--color-divider)] bg-transparent text-[color:var(--color-text-label)]",
        neutral: "bg-[color:var(--color-neutral-tint)] text-[color:var(--color-text-label)]",
        accent: "bg-[color:var(--color-accent-tint)] text-[color:var(--color-accent)]",
        "solid-accent": "bg-[color:var(--color-accent)] text-[color:var(--color-accent-ink)]",
        success: "bg-[color:var(--color-success-tint)] text-[color:var(--color-success)]",
        warning: "bg-[color:var(--color-warning-tint)] text-[color:var(--color-warning)]",
        danger: "bg-[color:var(--color-danger-tint)] text-[color:var(--color-danger)]",
        info: "bg-[color:var(--color-info-tint)] text-[color:var(--color-info)]",
      },
      uppercase: {
        true: "uppercase",
        false: "",
      },
    },
    defaultVariants: {
      tone: "default",
      uppercase: true,
    },
  }
);

export type MonoBadgeTone = NonNullable<VariantProps<typeof monoBadgeVariants>["tone"]>;

export interface MonoBadgeProps
  extends Omit<React.ComponentProps<"span">, "color">, VariantProps<typeof monoBadgeVariants> {
  "data-slot"?: string;
}

/**
 * Inline mono pill for identifiers (agent IDs, versions, protocol names) and
 * status badges. Uppercase by default, tinted via the DESIGN.md §4 tint formula.
 */
function MonoBadge({ tone, uppercase, className, ...props }: MonoBadgeProps) {
  const dataSlot = props["data-slot"] ?? "mono-badge";

  return (
    <span
      {...props}
      data-slot={dataSlot}
      data-tone={tone ?? "default"}
      className={cn(monoBadgeVariants({ tone, uppercase }), className)}
    />
  );
}

export { MonoBadge, monoBadgeVariants };
