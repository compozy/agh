"use client";

import * as React from "react";

import { cn } from "../lib/utils";

export interface KindChipProps extends Omit<React.ComponentProps<"span">, "children"> {
  kind: string;
}

/**
 * Protocol kind marker (e.g. `greet`, `whois`, `say`). 5px radius, lowercase mono,
 * accent-tint background with accent text — per DESIGN.md §4 "Kind Chip".
 */
function KindChip({ kind, className, ...props }: KindChipProps) {
  return (
    <span
      {...props}
      data-slot="kind-chip"
      data-kind={kind}
      className={cn(
        "inline-flex items-center rounded-[var(--radius-chip)] px-1.5 py-0.5",
        "font-mono text-[10.5px] font-medium leading-[14px] lowercase whitespace-nowrap",
        "bg-[color:var(--color-accent-tint)] text-[color:var(--color-accent)]",
        className
      )}
    >
      {kind}
    </span>
  );
}

export { KindChip };
