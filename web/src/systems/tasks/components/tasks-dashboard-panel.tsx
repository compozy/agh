import * as React from "react";

import { cn } from "@/lib/utils";

export interface TasksDashboardPanelProps extends Omit<React.ComponentProps<"section">, "title"> {
  title: React.ReactNode;
  /** Optional mono-faint meta slot rendered next to the title (counts, ids, durations). */
  meta?: React.ReactNode;
  /** Optional action slot right-aligned in the head (pills, buttons). */
  right?: React.ReactNode;
  /** Optional override for the inner body padding. */
  bodyClassName?: string;
}

/**
 * Dashboard panel primitive — flat warm card on `--canvas-soft` with a 13 px
 * `--text-section-head` title and an 18 px body padding. Title uses the
 * `<Section>` H2 grammar (13 / 510 / -0.008em) so the four dashboard panels
 * line up with the rest of the runtime body H2 ladder. The page-level 22 px
 * H1 lives in `<DetailHeader>` / `<Topbar>`, never inside a panel head.
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
        className="flex items-center justify-between gap-3 border-b border-line-soft px-5 py-3"
        data-slot="tasks-dashboard-panel-head"
      >
        <div className="flex min-w-0 items-baseline gap-2">
          <h3
            className="truncate text-section-head font-medium tracking-section-head text-fg-strong"
            data-slot="tasks-dashboard-panel-title"
          >
            {title}
          </h3>
          {meta !== undefined && meta !== null ? (
            <span
              className="shrink-0 font-mono text-mono-id tabular-nums text-faint"
              data-slot="tasks-dashboard-panel-meta"
            >
              {meta}
            </span>
          ) : null}
        </div>
        {right !== undefined && right !== null ? (
          <div className="flex shrink-0 items-center gap-2" data-slot="tasks-dashboard-panel-right">
            {right}
          </div>
        ) : null}
      </header>
      <div className={cn("p-5", bodyClassName)} data-slot="tasks-dashboard-panel-body">
        {children}
      </div>
    </section>
  );
}
