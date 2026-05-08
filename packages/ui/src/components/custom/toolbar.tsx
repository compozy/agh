"use client";

import * as React from "react";

import { cn } from "../../lib/utils";

export interface ToolbarProps extends React.ComponentProps<"div"> {
  sticky?: boolean;
}

/**
 * Horizontal toolbar shell — flex row with wrap on narrow viewports.
 * Composition-first: host decides which children (SearchInput, PillGroup, Button, etc.) go inside.
 */
function Toolbar({ className, sticky, ...props }: ToolbarProps) {
  return (
    <div
      data-slot="toolbar"
      data-sticky={sticky ? "true" : undefined}
      role="toolbar"
      className={cn(
        "flex min-h-11 flex-wrap items-center gap-2 border-b border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-4 py-2",
        sticky ? "sticky top-0 z-10" : "",
        className
      )}
      {...props}
    />
  );
}

export { Toolbar };
