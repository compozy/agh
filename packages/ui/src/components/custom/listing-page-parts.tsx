import * as React from "react";

import { cn } from "../../lib/utils";

export type ListingPageHeadProps = Omit<React.ComponentProps<"header">, "title"> & {
  title: React.ReactNode;
  count?: React.ReactNode;
  countTestId?: string;
  meta?: React.ReactNode;
};

export function ListingPageHead({
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

export function ListingPageMetaDot({ className, ...props }: React.ComponentProps<"span">) {
  return (
    <span
      aria-hidden="true"
      data-slot="listing-page-meta-dot"
      className={cn("size-0.5 rounded-full bg-faint", className)}
      {...props}
    />
  );
}
