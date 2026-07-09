"use client";

import * as React from "react";

import { cn } from "../../lib/utils";

type ListingPageProps = React.ComponentProps<"div"> & {
  /** Optional banner rendered above the scroll area (e.g. cached-data Alert). */
  banner?: React.ReactNode;
  /** Extra classes for the centered content container. */
  bodyClassName?: string;
};

type ListingPageHeadProps = Omit<React.ComponentProps<"header">, "title"> & {
  title: React.ReactNode;
  /** Mono count chip beside the title. Accepts `6` or `3 of 6`. */
  count?: React.ReactNode;
  /** Test id for the count chip (the header itself takes `data-testid`). */
  countTestId?: string;
  /** Dot-separated meta line under the title. */
  meta?: React.ReactNode;
};

function ListingPage({ banner, bodyClassName, className, children, ...props }: ListingPageProps) {
  return (
    <div
      data-slot="listing-page"
      className={cn("flex min-h-0 flex-1 flex-col overflow-hidden", className)}
      {...props}
    >
      {banner ? <div data-slot="listing-page-banner">{banner}</div> : null}
      <div className="flex min-h-0 flex-1 flex-col overflow-y-auto">
        <div
          data-slot="listing-page-container"
          className={cn(
            "mx-auto flex w-full max-w-[1320px] flex-col gap-5 px-9 pt-7 pb-20",
            bodyClassName
          )}
        >
          {children}
        </div>
      </div>
    </div>
  );
}

function ListingPageHead({
  title,
  count,
  countTestId,
  meta,
  className,
  children,
  ...props
}: ListingPageHeadProps) {
  return (
    <header
      data-slot="listing-page-head"
      className={cn("flex flex-col gap-1", className)}
      {...props}
    >
      <div className="flex min-w-0 items-center gap-2">
        <h1 className="text-detail-h1 font-semibold tracking-detail-h1 text-fg-strong">{title}</h1>
        {count !== undefined ? (
          <span
            data-slot="listing-page-count"
            data-testid={countTestId}
            className="inline-flex h-[19px] min-w-[19px] items-center justify-center rounded-mono-badge bg-canvas-soft px-1.5 font-mono text-mono-id font-medium tabular-nums text-muted"
          >
            {count}
          </span>
        ) : null}
      </div>
      {meta !== undefined ? (
        <p
          data-slot="listing-page-meta"
          className="flex flex-wrap items-center gap-2 text-small-body text-muted"
        >
          {meta}
        </p>
      ) : null}
      {children}
    </header>
  );
}

/** Neutral dot separator for `ListingPage.Head` meta lines. */
function ListingPageMetaDot({ className, ...props }: React.ComponentProps<"span">) {
  return (
    <span
      aria-hidden="true"
      data-slot="listing-page-meta-dot"
      className={cn("size-0.5 rounded-full bg-faint", className)}
      {...props}
    />
  );
}

const ListingPageCompound = Object.assign(ListingPage, {
  Head: ListingPageHead,
  MetaDot: ListingPageMetaDot,
});

export { ListingPageCompound as ListingPage };
export type { ListingPageHeadProps, ListingPageProps };
