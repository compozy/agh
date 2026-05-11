import * as React from "react";

import { cn } from "@/lib/utils";

export interface TasksDashboardPanelProps extends Omit<React.ComponentProps<"section">, "title"> {
  title: React.ReactNode;
  /** Optional mono-faint meta slot rendered next to the title. */
  meta?: React.ReactNode;
  /** Optional action slot right-aligned in the head (pills, buttons). */
  right?: React.ReactNode;
  /** Optional override for the inner body padding. Defaults to 18px on all sides. */
  bodyClassName?: string;
}

/**
 * `.dash__panel` primitive — flat warm card with a 12.5 px head and 18 px body
 * padding. Used by the Tasks Dashboard panels (Queue health, Status breakdown,
 * Active runs). Mirrors `docs/design/new-proposal/agh-refined-7.html` exactly,
 * so this primitive does NOT use the page-level `<Section>` (which renders a
 * 22 px H2 — incompatible with panel head size).
 */
export function TasksDashboardPanel({
  title,
  meta,
  right,
  className,
  bodyClassName,
  children,
  ...props
}: TasksDashboardPanelProps) {
  return (
    <section
      className={cn("rounded-lg bg-canvas-soft", className)}
      data-slot="tasks-dashboard-panel"
      {...props}
    >
      <header
        className="flex items-center justify-between gap-2 border-b border-line-soft px-[18px] py-[13px]"
        data-slot="tasks-dashboard-panel-head"
      >
        <div className="flex min-w-0 items-baseline gap-2">
          <h3
            className="truncate text-[12.5px] font-medium tracking-eyebrow text-fg-strong"
            data-slot="tasks-dashboard-panel-title"
          >
            {title}
          </h3>
          {meta !== undefined ? (
            <span
              className="shrink-0 font-mono text-[10.5px] tabular-nums text-faint"
              data-slot="tasks-dashboard-panel-meta"
            >
              {meta}
            </span>
          ) : null}
        </div>
        {right !== undefined ? (
          <div className="flex shrink-0 items-center gap-2" data-slot="tasks-dashboard-panel-right">
            {right}
          </div>
        ) : null}
      </header>
      <div className={cn("p-[18px]", bodyClassName)} data-slot="tasks-dashboard-panel-body">
        {children}
      </div>
    </section>
  );
}
