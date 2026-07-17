"use client";

import { ChevronLeft } from "lucide-react";
import * as React from "react";

import { cn } from "../../lib/utils";
import { Eyebrow } from "./eyebrow";

export interface DetailHeaderCrumb {
  id: string;
  label: React.ReactNode;
  to?: string;
  onSelect?: () => void;
}

export interface DetailHeaderProps extends Omit<React.ComponentProps<"header">, "title"> {
  /** The sole H1 on a detail surface. */
  title: React.ReactNode;
  /**
   * Optional crumb trail. Accepts either a structured list (rendered with `·` separators)
   * or any ReactNode (rendered inside a single Eyebrow).
   */
  crumbs?: ReadonlyArray<DetailHeaderCrumb> | React.ReactNode;
  /** Optional eyebrow-style pre-title. */
  preTitle?: React.ReactNode;
  /** Optional leading mark beside the title block. */
  leading?: React.ReactNode;
  /**
   * Optional pills on the same flex row as the H1.
   * Long titles truncate inside the title cell; pills stay start-aligned beside it.
   */
  pills?: React.ReactNode;
  /** Optional compact metadata row — id, time, owner, etc. */
  meta?: React.ReactNode;
  /** Trailing action cluster — end-aligned; wraps independently of the title/pill row. */
  actions?: React.ReactNode;
  /**
   * Back-button slot. Consumers wire `router.history.back` with a
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
  leading,
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
      className={cn("flex flex-col gap-2 border-b border-line px-6 py-5", className)}
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
              className="-ml-1 inline-flex size-5 shrink-0 items-center justify-center rounded-xs text-muted transition-colors duration-base ease-out hover:bg-hover hover:text-fg focus-visible:outline-none focus-visible:shadow-focus-ring"
            >
              <ChevronLeft width={12} height={12} strokeWidth={1.75} />
            </button>
          ) : null}
          <Eyebrow data-slot="detail-header-crumbs-label" className="min-w-0 truncate text-muted">
            {isCrumbArray(crumbs) ? <DetailHeaderCrumbList crumbs={crumbs} /> : crumbs}
          </Eyebrow>
        </div>
      ) : null}
      {preTitle ? (
        <Eyebrow data-slot="detail-header-pre-title" className="text-muted">
          {preTitle}
        </Eyebrow>
      ) : null}
      <div data-slot="detail-header-row" className="flex min-w-0 flex-wrap items-start gap-3.5">
        {leading ? (
          <div data-slot="detail-header-leading" className="shrink-0">
            {leading}
          </div>
        ) : null}
        <div data-slot="detail-header-title-block" className="flex min-w-0 flex-1 flex-col gap-1.5">
          <div
            data-slot="detail-header-title-row"
            className="flex min-w-0 flex-wrap items-center gap-2"
          >
            <h1
              data-slot="detail-header-title"
              className="min-w-0 truncate text-detail-h1 font-medium tracking-detail-h1 text-fg-strong"
            >
              {title}
            </h1>
            {pills ? (
              <div
                data-slot="detail-header-pills"
                className="flex shrink-0 flex-wrap items-center gap-1.5"
              >
                {pills}
              </div>
            ) : null}
          </div>
          {meta ? (
            <div
              data-slot="detail-header-meta"
              className="flex flex-wrap items-center gap-3 text-form-label text-muted"
            >
              {meta}
            </div>
          ) : null}
        </div>
        {actions ? (
          <div
            data-slot="detail-header-actions"
            className="ml-auto flex max-w-full shrink-0 flex-wrap items-center justify-end gap-2"
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
        const interactive = Boolean(crumb.to || crumb.onSelect);
        return (
          <React.Fragment key={crumb.id}>
            {index > 0 ? (
              <span aria-hidden="true" className="text-faint">
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
                className="truncate transition-colors duration-base ease-out hover:text-fg"
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
