"use client";

import * as React from "react";

import { cn } from "../../lib/utils";

export interface PropertyRowProps extends React.ComponentProps<"div"> {
  /** Row label, left column (12px subtle). */
  label: React.ReactNode;
  /** Renders the value in the mono id voice (ids, models, seeds). */
  mono?: boolean;
  /** Value content; ignored when `editor` is provided. */
  children?: React.ReactNode;
  /**
   * Inline editor trigger replacing the static value (e.g. a DropdownMenu
   * trigger). The slot is rendered flush-right and owns its own semantics.
   */
  editor?: React.ReactNode;
}

/**
 * Detail-rail key/value row: quiet label left, value right, one line, no
 * frame. The rail speaks entirely through these rows — never bespoke dls.
 */
function PropertyRow({
  label,
  mono = false,
  editor,
  className,
  children,
  ...props
}: PropertyRowProps) {
  return (
    <div
      data-slot="property-row"
      className={cn("flex min-h-[26px] items-center justify-between gap-3 py-[3px]", className)}
      {...props}
    >
      <span data-slot="property-row-label" className="shrink-0 text-form-label text-subtle">
        {label}
      </span>
      {editor ?? (
        <span
          data-slot="property-row-value"
          data-mono={mono ? "true" : undefined}
          className={cn(
            "inline-flex min-w-0 items-center gap-1.5 text-right font-medium",
            mono
              ? "font-mono text-eyebrow font-normal tabular-nums text-muted"
              : "text-small-body text-fg"
          )}
        >
          {children}
        </span>
      )}
    </div>
  );
}

export { PropertyRow };
