"use client";

import * as React from "react";

import { cn } from "../../lib/utils";

export type MetricTone = "default" | "accent" | "success" | "warning" | "danger";

export interface MetricProps extends Omit<React.ComponentProps<"div">, "title"> {
  label: React.ReactNode;
  value: React.ReactNode;
  /**
   * Small inline detail baseline-aligned with the value , mono micro-unit (e.g. "+12%").
   * Mirrors `detail` in `docs/design/web-inspiration/src/primitives.jsx`.
   */
  detail?: React.ReactNode;
  /**
   * Secondary line rendered below the value , Inter 13px per DESIGN.md §4 "Metric Cards With Subtext".
   */
  subtext?: React.ReactNode;
  tone?: MetricTone;
}

const VALUE_COLOR: Record<MetricTone, string> = {
  default: "var(--fg)",
  accent: "var(--accent)",
  success: "var(--success)",
  warning: "var(--warning)",
  danger: "var(--danger)",
};

/**
 * Metric card , mono eyebrow label + Inter 24px/700 value + optional inline detail
 * or subtext line. Surface container with 12px radius; semantic tone colors the value.
 * Per DESIGN.md §4 "Metric Cards" and mock `docs/design/web-inspiration/src/primitives.jsx`.
 */
function Metric({
  label,
  value,
  detail,
  subtext,
  tone = "default",
  className,
  ...props
}: MetricProps) {
  return (
    <div
      data-slot="metric"
      data-tone={tone}
      className={cn(
        "flex min-w-0 flex-col gap-2 rounded-[var(--radius-lg)] border border-[color:var(--line)] bg-[color:var(--canvas-soft)] px-5 py-4",
        className
      )}
      {...props}
    >
      <span
        data-slot="metric-label"
        className="block truncate font-mono text-[11px] font-semibold uppercase leading-4 tracking-[0.06em] text-[color:var(--subtle)]"
      >
        {label}
      </span>
      <div data-slot="metric-value-row" className="flex min-w-0 items-baseline gap-2">
        <span
          data-slot="metric-value"
          className="min-w-0 truncate text-[24px] font-bold leading-[30px] tracking-[-0.02em]"
          style={{ color: VALUE_COLOR[tone] }}
        >
          {value}
        </span>
        {detail !== undefined ? (
          <span
            data-slot="metric-detail"
            className="shrink-0 truncate font-mono text-[11px] leading-4 text-[color:var(--subtle)]"
          >
            {detail}
          </span>
        ) : null}
      </div>
      {subtext !== undefined ? (
        <p
          data-slot="metric-subtext"
          className="truncate text-[13px] leading-5 text-[color:var(--muted)]"
        >
          {subtext}
        </p>
      ) : null}
    </div>
  );
}

export { Metric };
