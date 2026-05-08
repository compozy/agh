"use client";

import * as React from "react";

import { cn } from "../../lib/utils";

export interface SectionProps extends React.ComponentProps<"section"> {
  label?: React.ReactNode;
  right?: React.ReactNode;
}

function hasSectionContent(content: React.ReactNode): boolean {
  return content !== undefined && content !== null && content !== false;
}

/**
 * Section shell — mono eyebrow + optional right-aligned slot + children.
 * Mirrors `Section` in `docs/design/web-inspiration/src/primitives.jsx`.
 */
function Section({ label, right, className, children, ...props }: SectionProps) {
  const hasLabel = hasSectionContent(label);
  const hasRight = hasSectionContent(right);

  return (
    <section
      data-slot="section"
      className={cn("flex min-w-0 flex-col gap-3", className)}
      {...props}
    >
      {hasLabel || hasRight ? (
        <header
          data-slot="section-head"
          className="flex items-center justify-between gap-3 border-b border-[color:var(--color-divider)] pb-2"
        >
          {hasLabel ? (
            <h2
              data-slot="section-label"
              className="font-mono text-[11px] font-semibold uppercase tracking-[0.06em] text-[color:var(--color-text-label)]"
            >
              {label}
            </h2>
          ) : (
            <span />
          )}
          {hasRight ? (
            <div data-slot="section-right" className="flex items-center gap-2">
              {right}
            </div>
          ) : null}
        </header>
      ) : null}
      <div data-slot="section-body" className="flex min-w-0 flex-col">
        {children}
      </div>
    </section>
  );
}

export { Section };
