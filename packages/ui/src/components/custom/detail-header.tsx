"use client";

import * as React from "react";

import { cn } from "../../lib/utils";

export interface DetailHeaderProps extends Omit<React.ComponentProps<"header">, "title"> {
  title: React.ReactNode;
  /** Crumbs/eyebrow row above the title. */
  crumbs?: React.ReactNode;
  /** Pill row immediately under the title. */
  pills?: React.ReactNode;
  /** Compact metadata row (id, time, etc.). */
  meta?: React.ReactNode;
  /** Trailing action cluster (buttons, dropdowns). */
  actions?: React.ReactNode;
}

function DetailHeader({
  title,
  crumbs,
  pills,
  meta,
  actions,
  className,
  children,
  ...props
}: DetailHeaderProps) {
  return (
    <header
      data-slot="detail-header"
      className={cn("flex flex-col gap-2 border-b border-(--line) px-6 py-5", className)}
      {...props}
    >
      {crumbs ? (
        <div
          data-slot="detail-header-crumbs"
          className="font-mono text-[10.5px] font-medium uppercase tracking-[0.05em] text-(--muted)"
        >
          {crumbs}
        </div>
      ) : null}
      <div data-slot="detail-header-row" className="flex min-w-0 flex-wrap items-start gap-3">
        <div data-slot="detail-header-title-block" className="flex min-w-0 flex-col gap-1.5">
          <h1 className="truncate text-[24px] font-medium tracking-[-0.028em] text-(--fg-strong)">
            {title}
          </h1>
          {pills ? (
            <div data-slot="detail-header-pills" className="flex flex-wrap items-center gap-1.5">
              {pills}
            </div>
          ) : null}
          {meta ? (
            <div
              data-slot="detail-header-meta"
              className="flex flex-wrap items-center gap-3 text-[12px] text-(--muted)"
            >
              {meta}
            </div>
          ) : null}
        </div>
        {actions ? (
          <div
            data-slot="detail-header-actions"
            className="ml-auto flex shrink-0 items-center gap-2"
          >
            {actions}
          </div>
        ) : null}
      </div>
      {children}
    </header>
  );
}

export { DetailHeader };
