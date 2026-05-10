"use client";

import * as React from "react";

import { cn } from "../../lib/utils";

/**
 * Bordered protocol card , mirrors `.wire-card` (head/body/foot) in
 * `docs/design/web-inspiration/styles/app.css`. Used to embed wire-protocol
 * payloads (recipes, receipts, capability descriptors) inside message
 * threads. Pair with {@link WireCardHead} / {@link WireCardFoot}.
 */
export interface WireCardProps extends React.ComponentProps<"div"> {
  /** Render as a single-line inline strip (used for receipts). */
  inline?: boolean;
}

function WireCard({ inline = false, className, ...props }: WireCardProps) {
  return (
    <div
      {...props}
      data-slot="wire-card"
      data-inline={inline ? "true" : undefined}
      className={cn(
        "border border-[color:var(--line)] bg-[color:var(--canvas-soft)]",
        inline
          ? "inline-flex items-center gap-2 rounded-[6px] px-2.5 py-1.5"
          : "max-w-[520px] overflow-hidden rounded-[6px]",
        className
      )}
    />
  );
}

function WireCardHead({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      {...props}
      data-slot="wire-card-head"
      className={cn(
        "flex items-center gap-1.5 border-b border-[color:var(--line)] bg-[color:var(--canvas)] px-2.5 py-1.5",
        "font-mono text-[10.5px] uppercase tracking-[0.06em] text-[color:var(--subtle)]",
        className
      )}
    />
  );
}

function WireCardBody({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      {...props}
      data-slot="wire-card-body"
      className={cn("px-3 py-2 font-mono text-[11px]", className)}
    />
  );
}

function WireCardFoot({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      {...props}
      data-slot="wire-card-foot"
      className={cn(
        "flex items-center gap-1.5 border-t border-[color:var(--line)] bg-[color:var(--canvas)] px-2.5 py-1.5",
        className
      )}
    />
  );
}

export { WireCard, WireCardHead, WireCardBody, WireCardFoot };
