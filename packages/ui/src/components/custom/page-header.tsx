"use client";

import * as React from "react";

import { cn } from "../../lib/utils";

type IconComponent = React.ComponentType<{ className?: string; size?: number }>;

export interface PageHeaderProps extends Omit<React.ComponentProps<"header">, "title"> {
  title: React.ReactNode;
  icon?: IconComponent;
  count?: number | string;
  breadcrumb?: React.ReactNode;
  subtitle?: React.ReactNode;
  statusRow?: React.ReactNode;
  controls?: React.ReactNode;
  meta?: React.ReactNode;
}

/**
 * Page chrome header: icon + title + count on the left, controls in the middle,
 * meta on the right. Mirrors `PageHeader` in `docs/design/web-inspiration/src/primitives.jsx`.
 */
function PageHeader({
  title,
  icon: Icon,
  count,
  breadcrumb,
  subtitle,
  statusRow,
  controls,
  meta,
  className,
  ...props
}: PageHeaderProps) {
  return (
    <header
      data-slot="page-header"
      className={cn(
        "flex min-h-11 flex-col gap-2 border-b border-[color:var(--color-divider)] px-4 py-2.5",
        className
      )}
      {...props}
    >
      {breadcrumb ? (
        <div
          data-slot="page-header-breadcrumb"
          className="min-w-0 font-mono text-[11px] font-semibold uppercase tracking-mono text-[color:var(--color-text-label)]"
        >
          {breadcrumb}
        </div>
      ) : null}
      <div data-slot="page-header-main" className="flex min-w-0 flex-wrap items-center gap-2 sm:gap-3">
        <div data-slot="page-header-title" className="flex min-w-0 items-center gap-2">
          {Icon ? (
            <span
              aria-hidden="true"
              data-slot="page-header-icon"
              className="inline-flex size-6 shrink-0 items-center justify-center rounded-md bg-[color:var(--color-surface-elevated)] text-[color:var(--color-accent)]"
            >
              <Icon className="size-3.5" />
            </span>
          ) : null}
          <h1 className="truncate text-[15px] font-semibold text-[color:var(--color-text-primary)]">
            {title}
          </h1>
          {count !== undefined ? (
            <span
              data-slot="page-header-count"
              className="inline-flex h-5 min-w-5 items-center justify-center rounded-full border border-[color:var(--color-divider)] px-1.5 font-mono text-[10px] font-semibold tracking-[0.08em] text-[color:var(--color-text-tertiary)]"
            >
              {count}
            </span>
          ) : null}
        </div>
        {controls ? (
          <div
            data-slot="page-header-controls"
            className="order-3 flex w-full min-w-0 items-center gap-2 sm:order-none sm:w-auto"
          >
            {controls}
          </div>
        ) : null}
        {meta ? (
          <div
            data-slot="page-header-meta"
            className="ml-auto flex shrink-0 items-center gap-2 text-[13px] text-[color:var(--color-text-secondary)]"
          >
            {meta}
          </div>
        ) : null}
      </div>
      {subtitle ? (
        <div
          data-slot="page-header-subtitle"
          className="max-w-152 text-small-body text-[color:var(--color-text-secondary)]"
        >
          {subtitle}
        </div>
      ) : null}
      {statusRow ? (
        <div
          data-slot="page-header-status-row"
          className="flex flex-wrap items-center gap-x-4 gap-y-2 text-small-body text-[color:var(--color-text-secondary)]"
        >
          {statusRow}
        </div>
      ) : null}
    </header>
  );
}

export { PageHeader };
