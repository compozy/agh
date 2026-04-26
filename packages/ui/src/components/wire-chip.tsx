"use client";

import * as React from "react";

import { cn } from "../lib/utils";

export interface WireChipProps extends Omit<React.ComponentProps<"button">, "children"> {
  active?: boolean;
  /** Optional CSS color or var() for the leading 7px wire-dot. */
  dotColor?: string;
  children: React.ReactNode;
}

/**
 * Free-floating filter chip — mirrors `.wire-chip` in
 * `docs/design/web-inspiration/styles/app.css`. Used in stand-alone filter
 * rows (e.g. the network channel header `ALL · SAY · DIRECT · …`). For a
 * contained segmented toggle, use {@link Pills}.
 */
function WireChip({
  active = false,
  dotColor,
  children,
  className,
  type = "button",
  ...props
}: WireChipProps) {
  return (
    <button
      {...props}
      type={type}
      data-slot="wire-chip"
      data-active={active ? "true" : undefined}
      aria-pressed={active}
      className={cn(
        "inline-flex cursor-pointer items-center gap-1.5 rounded-[4px] border px-2 py-[3px] font-mono text-[10.5px] transition-colors duration-100",
        active
          ? "border-[color:var(--color-text-tertiary)] bg-[color:var(--color-surface-elevated)] text-[color:var(--color-text-primary)]"
          : "border-[color:var(--color-divider)] bg-[color:var(--color-surface)] text-[color:var(--color-text-secondary)] hover:border-[color:var(--color-text-tertiary)] hover:text-[color:var(--color-text-primary)]",
        "focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-[color:var(--color-accent)]",
        className
      )}
    >
      {dotColor ? (
        <span
          aria-hidden="true"
          data-slot="wire-chip-dot"
          className="inline-block size-[7px] shrink-0 rounded-full"
          style={{ background: dotColor }}
        />
      ) : null}
      <span>{children}</span>
    </button>
  );
}

export { WireChip };
