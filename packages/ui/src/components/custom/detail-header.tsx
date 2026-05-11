"use client";

import { ChevronLeft } from "lucide-react";
import * as React from "react";

import { cn } from "../../lib/utils";
import { Eyebrow } from "./eyebrow";

export interface DetailHeaderCrumb {
  label: React.ReactNode;
  to?: string;
  onSelect?: () => void;
}

export interface DetailHeaderProps extends Omit<React.ComponentProps<"header">, "title"> {
  /** Title row (24px Inter UC) — the sole H1 on a detail surface. */
  title: React.ReactNode;
  /**
   * Optional crumb trail. Accepts either a structured list (rendered with `·` separators)
   * or any ReactNode (rendered inside a single Eyebrow). Crumbs render in row 1 of the
   * 6-row anatomy.
   */
  crumbs?: ReadonlyArray<DetailHeaderCrumb> | React.ReactNode;
  /** Optional eyebrow-style pre-title row (row 2). */
  preTitle?: React.ReactNode;
  /** Optional pill row immediately under the title (row 4). */
  pills?: React.ReactNode;
  /** Optional compact metadata row (row 5) — id, time, owner, etc. */
  meta?: React.ReactNode;
  /** Trailing action cluster (row 6) — buttons, dropdowns. */
  actions?: React.ReactNode;
  /**
   * Back-button slot Consumers wire `router.history.back` with a
   * parent-route fallback. When omitted the chevron is not rendered.
   */
  back?: () => void;
  /** Accessible label for the back button when `back` is set. */
  backLabel?: string;
}

function isCrumbArray(value: unknown): value is ReadonlyArray<DetailHeaderCrumb> {
  return Array.isArray(value);
}

function DetailHeader({
  title,
  crumbs,
  preTitle,
  pills,
  meta,
  actions,
  back,
  backLabel = "Go back",
  className,
  children,
  ...props
}: DetailHeaderProps) {
  return (
    <header
      data-slot="detail-header"
      className={cn("flex flex-col gap-2 border-b border-(--line) px-9 py-7", className)}
      {...props}
    >
      {crumbs ? (
        <div data-slot="detail-header-crumbs" className="flex min-w-0 items-center gap-2">
          {back ? (
            <button
              type="button"
              data-slot="detail-header-back"
              aria-label={backLabel}
              onClick={back}
              className="-ml-1 inline-flex size-5 shrink-0 items-center justify-center rounded-(--radius-xs) text-(--muted) transition-colors duration-(--dur) ease-(--ease) hover:bg-(--hover) hover:text-(--fg) focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-(--line-strong)"
            >
              <ChevronLeft width={12} height={12} strokeWidth={1.75} />
            </button>
          ) : null}
          <Eyebrow
            data-slot="detail-header-crumbs-label"
            className="min-w-0 truncate text-(--muted)"
          >
            {isCrumbArray(crumbs) ? <DetailHeaderCrumbList crumbs={crumbs} /> : crumbs}
          </Eyebrow>
        </div>
      ) : null}
      {preTitle ? (
        <Eyebrow data-slot="detail-header-pre-title" className="text-(--muted)">
          {preTitle}
        </Eyebrow>
      ) : null}
      <div data-slot="detail-header-row" className="flex min-w-0 flex-wrap items-start gap-3">
        <div data-slot="detail-header-title-block" className="flex min-w-0 flex-col gap-1.5">
          <h1
            data-slot="detail-header-title"
            className="truncate text-[length:var(--text-detail-h1)] font-medium tracking-(--tracking-detail-h1) text-(--fg-strong)"
            style={{ fontWeight: 510 }}
          >
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

function DetailHeaderCrumbList({ crumbs }: { crumbs: ReadonlyArray<DetailHeaderCrumb> }) {
  return (
    <span
      data-slot="detail-header-crumbs-list"
      className="inline-flex min-w-0 items-center gap-1.5"
    >
      {crumbs.map((crumb, index) => {
        const key = `${index}-${typeof crumb.label === "string" ? crumb.label : "crumb"}`;
        const interactive = Boolean(crumb.to || crumb.onSelect);
        return (
          <React.Fragment key={key}>
            {index > 0 ? (
              <span aria-hidden="true" className="text-(--faint)">
                ·
              </span>
            ) : null}
            {interactive ? (
              <a
                data-slot="detail-header-crumb"
                href={crumb.to ?? "#"}
                onClick={
                  crumb.onSelect
                    ? event => {
                        if (!crumb.to) event.preventDefault();
                        crumb.onSelect?.();
                      }
                    : undefined
                }
                className="truncate transition-colors duration-(--dur) ease-(--ease) hover:text-(--fg)"
              >
                {crumb.label}
              </a>
            ) : (
              <span data-slot="detail-header-crumb" className="truncate">
                {crumb.label}
              </span>
            )}
          </React.Fragment>
        );
      })}
    </span>
  );
}

export { DetailHeader };
