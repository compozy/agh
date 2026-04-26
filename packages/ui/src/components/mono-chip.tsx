"use client";

import * as React from "react";

import { cn } from "../lib/utils";

export interface MonoChipProps extends React.ComponentProps<"span"> {}

/**
 * Neutral inline chip — mirrors `.mono-chip` (default tone) in
 * `docs/design/web-inspiration/styles/app.css`. Used for capability
 * descriptors, tag rows, and other identifier strings rendered alongside
 * message bodies. For tinted semantic variants use {@link MonoBadge}.
 */
function MonoChip({ className, ...props }: MonoChipProps) {
  return (
    <span
      {...props}
      data-slot="mono-chip"
      className={cn(
        "inline-flex items-center gap-1 rounded-[5px] bg-[color:var(--color-surface-elevated)] px-1.5 py-px",
        "font-mono text-[10px] font-medium leading-[14px] tracking-[0.04em] text-[color:var(--color-text-secondary)] whitespace-nowrap",
        className
      )}
    />
  );
}

export { MonoChip };
