"use client";

import * as React from "react";

import { cn } from "../../lib/utils";

export interface ContextBoxEntry {
  label: React.ReactNode;
  value: React.ReactNode;
}

export interface ContextBoxProps extends Omit<React.ComponentProps<"dl">, "children" | "title"> {
  entries: ReadonlyArray<ContextBoxEntry>;
  /** Optional title rendered above the grid. */
  title?: React.ReactNode;
}

function ContextBox({ entries, title, className, ...props }: ContextBoxProps) {
  return (
    <div data-slot="context-box-root" className={cn("flex flex-col gap-2", className)}>
      {title ? (
        <div
          data-slot="context-box-title"
          className="font-mono text-[10.5px] font-medium uppercase tracking-[0.05em] text-(--muted)"
        >
          {title}
        </div>
      ) : null}
      <dl
        data-slot="context-box"
        className={cn(
          "grid grid-cols-[minmax(0,140px)_minmax(0,1fr)] gap-x-3 gap-y-1.5 rounded-(--radius) border border-(--line) bg-(--canvas-soft) px-3 py-2.5"
        )}
        {...props}
      >
        {entries.map((entry, index) => (
          <React.Fragment key={`${index}-${String(entry.label)}`}>
            <dt
              data-slot="context-box-label"
              className="font-mono text-[10.5px] font-medium uppercase tracking-[0.05em] text-(--muted)"
            >
              {entry.label}
            </dt>
            <dd data-slot="context-box-value" className="text-[12px] text-(--fg)">
              {entry.value}
            </dd>
          </React.Fragment>
        ))}
      </dl>
    </div>
  );
}

export { ContextBox };
