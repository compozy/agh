"use client";

import * as React from "react";

import { cn } from "../../lib/utils";

type IconComponent = React.ComponentType<{ className?: string; size?: number }>;

export type DashboardCardLabelCase = "sentence" | "upper";

export interface DashboardCardProps extends React.ComponentProps<"div"> {
  label: React.ReactNode;
  value: React.ReactNode;
  /** Optional supporting line under the value. */
  detail?: React.ReactNode;
  /** Optional leading icon at the head. */
  icon?: IconComponent;
  /** Optional trailing slot at the head (delta, sparkline, link). */
  trailing?: React.ReactNode;
  /** Casing for the label; defaults to upper-mono for dashboard rhythm. */
  labelCase?: DashboardCardLabelCase;
}

function DashboardCard({
  label,
  value,
  detail,
  icon: Icon,
  trailing,
  labelCase = "upper",
  className,
  children,
  ...props
}: DashboardCardProps) {
  return (
    <div
      data-slot="dashboard-card"
      className={cn(
        "flex min-w-0 flex-col gap-2 rounded-(--radius-lg) border border-(--line) bg-(--canvas-soft) px-5 py-4",
        className
      )}
      {...props}
    >
      <div data-slot="dashboard-card-head" className="flex min-w-0 items-center gap-2">
        {Icon ? (
          <span
            aria-hidden="true"
            data-slot="dashboard-card-icon"
            className="inline-flex size-5 shrink-0 items-center justify-center text-(--muted)"
          >
            <Icon className="size-3.5" />
          </span>
        ) : null}
        <span
          data-slot="dashboard-card-label"
          data-case={labelCase}
          className={cn(
            "min-w-0 truncate",
            labelCase === "upper"
              ? "font-mono text-[10.5px] font-medium uppercase tracking-[0.05em] text-(--muted)"
              : "text-[12px] font-medium tracking-[-0.005em] text-(--muted)"
          )}
        >
          {label}
        </span>
        {trailing ? (
          <span data-slot="dashboard-card-trailing" className="ml-auto inline-flex items-center">
            {trailing}
          </span>
        ) : null}
      </div>
      <div
        data-slot="dashboard-card-value"
        className="text-[28px] font-medium leading-none tracking-[-0.028em] text-(--fg-strong) tabular-nums"
      >
        {value}
      </div>
      {detail ? (
        <p data-slot="dashboard-card-detail" className="text-[12px] text-(--muted)">
          {detail}
        </p>
      ) : null}
      {children}
    </div>
  );
}

export { DashboardCard };
