"use client";

import * as React from "react";

import { cn } from "../../lib/utils";
import { Eyebrow } from "./eyebrow";

export type MetricTone = "default" | "accent" | "success" | "warning" | "danger";
export type MetricSize = "default" | "lg";

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
  /**
   * `default` — value at 24 px, generic card density.
   * `lg` — value at 28 px with tighter tracking, mirrors `.dash__card-value`
   * from `docs/design/new-proposal/agh-refined-7.html`. Use for top-level
   * dashboard metrics (Active runs, Success rate, etc.).
   */
  size?: MetricSize;
}

const VALUE_COLOR: Record<MetricTone, string> = {
  default: "var(--fg)",
  accent: "var(--accent)",
  success: "var(--success)",
  warning: "var(--warning)",
  danger: "var(--danger)",
};

const SIZE_VALUE_CLASS: Record<MetricSize, string> = {
  default: "text-[24px] leading-[30px] tracking-[-0.02em]",
  lg: "text-[28px] leading-[1.05] tracking-detail-h1",
};

const SIZE_CONTAINER_CLASS: Record<MetricSize, string> = {
  default: "px-5 py-4",
  lg: "px-[18px] py-4",
};

/**
 * Metric card , mono eyebrow label + Inter 24/28px/510 value + optional inline
 * detail or subtext line. Surface container with 12px radius; semantic tone
 * colors the value. Per DESIGN.md §4 "Metric Cards" and the proposal at
 * `docs/design/new-proposal/agh-refined-7.html` (`.dash__card-value` for
 * `size="lg"`).
 */
function Metric({
  label,
  value,
  detail,
  subtext,
  tone = "default",
  size = "default",
  className,
  ...props
}: MetricProps) {
  return (
    <div
      data-slot="metric"
      data-tone={tone}
      data-size={size}
      className={cn(
        "flex min-w-0 flex-col gap-2 rounded-lg bg-(--canvas-soft)",
        SIZE_CONTAINER_CLASS[size],
        className
      )}
      {...props}
    >
      <Eyebrow data-slot="metric-label" className="block truncate leading-4 text-(--subtle)">
        {label}
      </Eyebrow>
      <div data-slot="metric-value-row" className="flex min-w-0 items-baseline gap-2">
        <span
          data-slot="metric-value"
          className={cn("min-w-0 truncate font-medium tabular-nums", SIZE_VALUE_CLASS[size])}
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
