"use client";

import * as React from "react";

import { cn } from "../../lib/utils";
import { Eyebrow } from "./eyebrow";

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
        "flex min-w-0 flex-col gap-2 rounded-lg border border-(--line) bg-(--canvas-soft) px-5 py-4",
        className
      )}
      {...props}
    >
      <Eyebrow
        data-slot="metric-label"
        case="upper"
        tone="subtle"
        className="block truncate leading-4"
      >
        {label}
      </Eyebrow>
      <div data-slot="metric-value-row" className="flex min-w-0 items-baseline gap-2">
        <span
          data-slot="metric-value"
          className="min-w-0 truncate text-[24px] font-medium leading-[30px] tracking-[-0.02em]"
          style={{ color: VALUE_COLOR[tone], fontWeight: 510 }}
        >
          {value}
        </span>
        {detail !== undefined ? (
          <span
            data-slot="metric-detail"
            className="shrink-0 truncate font-mono text-eyebrow leading-4 text-(--subtle)"
          >
            {detail}
          </span>
        ) : null}
      </div>
      {subtext !== undefined ? (
        <p data-slot="metric-subtext" className="truncate text-[13px] leading-5 text-(--muted)">
          {subtext}
        </p>
      ) : null}
    </div>
  );
}

export { Metric };
