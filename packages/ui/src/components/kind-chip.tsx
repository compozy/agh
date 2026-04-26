"use client";

import * as React from "react";

import { cn } from "../lib/utils";

export interface KindChipProps extends Omit<React.ComponentProps<"span">, "children"> {
  kind: string;
  /** Optional explicit label; defaults to `kind`. */
  label?: React.ReactNode;
}

/**
 * Protocol kind marker — mirrors `.intent-badge` + `.wire-dot` in
 * `docs/design/web-inspiration/styles/app.css`. Transparent surface, neutral
 * border + tertiary label, leading 7px colored dot keyed off the protocol
 * kind. Unknown kinds (platform names, event ids) render without a dot.
 */
const KIND_DOT_COLORS: Record<string, string> = {
  say: "#8E8E93",
  greet: "#5BA6FF",
  direct: "var(--color-accent)",
  receipt: "var(--color-success)",
  recipe: "var(--color-warning)",
  trace: "#B892FF",
  whois: "#4FD1C5",
};

function KindChip({ kind, label, className, ...props }: KindChipProps) {
  const dotColor = KIND_DOT_COLORS[kind.toLowerCase()];

  return (
    <span
      {...props}
      data-slot="kind-chip"
      data-kind={kind}
      className={cn(
        "inline-flex items-center gap-1.5 rounded-[3px] border border-[color:var(--color-divider)] bg-transparent px-1.5 py-px",
        "font-mono text-[9.5px] font-semibold uppercase leading-[14px] tracking-[0.08em] whitespace-nowrap text-[color:var(--color-text-tertiary)]",
        className
      )}
    >
      {dotColor ? (
        <span
          aria-hidden="true"
          data-slot="kind-chip-dot"
          className="inline-block size-[7px] shrink-0 rounded-full"
          style={{ background: dotColor }}
        />
      ) : null}
      <span>{label ?? kind}</span>
    </span>
  );
}

export { KindChip, KIND_DOT_COLORS };
