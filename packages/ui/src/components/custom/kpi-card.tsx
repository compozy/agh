"use client";

import * as React from "react";

import { cn } from "../../lib/utils";
import { Eyebrow } from "./eyebrow";

type IconComponent = React.ComponentType<{ className?: string; size?: number }>;

export interface KpiCardProps extends React.ComponentProps<"div"> {
  label: React.ReactNode;
  value: React.ReactNode;
  /** Optional supporting line under the value. */
  detail?: React.ReactNode;
  /** Optional leading icon at the head. */
  icon?: IconComponent;
  /** Optional trailing slot at the head (delta, sparkline, link). */
  trailing?: React.ReactNode;
}

/**
 * Dashboard KPI tile — flat on `--canvas-soft`, no border.
 * 28 px Inter UC value, Eyebrow label, optional 12 px detail line.
 */
function KpiCard({
  label,
  value,
  detail,
  icon: Icon,
  trailing,
  className,
  children,
  ...props
}: KpiCardProps) {
  return (
    <div
      data-slot="kpi-card"
      className={cn("flex min-w-0 flex-col gap-2 rounded-lg bg-canvas-soft px-5 py-4", className)}
      {...props}
    >
      <div data-slot="kpi-card-head" className="flex min-w-0 items-center gap-2">
        {Icon ? (
          <span
            aria-hidden="true"
            data-slot="kpi-card-icon"
            className="inline-flex size-5 shrink-0 items-center justify-center text-muted"
          >
            <Icon className="size-3.5" />
          </span>
        ) : null}
        <Eyebrow data-slot="kpi-card-label" className="min-w-0 truncate text-muted">
          {label}
        </Eyebrow>
        {trailing ? (
          <span data-slot="kpi-card-trailing" className="ml-auto inline-flex items-center">
            {trailing}
          </span>
        ) : null}
      </div>
      <div
        data-slot="kpi-card-value"
        className="text-[28px] font-medium leading-none tracking-detail-h1 text-fg-strong tabular-nums"
      >
        {value}
      </div>
      {detail ? (
        <p data-slot="kpi-card-detail" className="text-[12px] text-muted">
          {detail}
        </p>
      ) : null}
      {children}
    </div>
  );
}

export { KpiCard };
