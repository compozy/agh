"use client";

import type { LucideIcon } from "lucide-react";
import * as React from "react";

import { cn } from "../../lib/utils";

type PageHeadVariant = "index" | "detail" | "compact";

type PageHeadProps = Omit<React.ComponentProps<"header">, "title"> & {
  /** Page family: PH1 index collection (default), PH2 entity detail, PH3 split-pane compact. */
  variant?: PageHeadVariant;
  /** Route glyph rendered in the 24px elevated icon well. */
  icon?: LucideIcon;
  /** Custom leading mark (entity avatar, kind icon); wins over `icon`. */
  leading?: React.ReactNode;
  /** Body-side summary title. The route's single H1 remains in the shell Topbar. */
  title: React.ReactNode;
  /** Mono count chip beside the title. */
  count?: React.ReactNode;
  countTestId?: string;
  /** Mono pre-title above the H1 (compact variant: source id, file name). */
  pretitle?: React.ReactNode;
  /** Status/kind pills row under the title (detail variant). */
  pills?: React.ReactNode;
  /** Dot-separated context line (purpose · scope · freshness). */
  meta?: React.ReactNode;
  /**
   * Trailing hero controls (runtime selectors, inline editors) on the head's
   * right edge. Route actions stay in the topbar (route chrome §08).
   */
  actions?: React.ReactNode;
};

function PageHead({
  variant = "index",
  icon: Icon,
  leading,
  title,
  count,
  countTestId,
  pretitle,
  pills,
  meta,
  actions,
  className,
  children,
  ...props
}: PageHeadProps) {
  const mark =
    leading ??
    (Icon ? (
      <span
        aria-hidden="true"
        data-slot="page-head-icon"
        className="inline-flex size-6 items-center justify-center rounded-sm bg-elevated text-accent"
      >
        <Icon className="size-3" />
      </span>
    ) : null);

  return (
    <header
      data-slot="page-head"
      data-variant={variant}
      className={cn(
        "flex min-w-0 items-start",
        variant === "compact" ? "gap-2.5" : "gap-3",
        variant === "detail" && "border-b border-line pb-4.5",
        className
      )}
      {...props}
    >
      {mark ? (
        <div data-slot="page-head-leading" className="mt-0.5 shrink-0">
          {mark}
        </div>
      ) : null}
      <div data-slot="page-head-main" className="flex min-w-0 flex-1 flex-col gap-1">
        {pretitle ? (
          <div
            data-slot="page-head-pretitle"
            className="font-mono text-mono-id tracking-mono-id text-subtle"
          >
            {pretitle}
          </div>
        ) : null}
        <div data-slot="page-head-title-row" className="flex min-w-0 items-center gap-2">
          <div
            data-slot="page-head-title"
            className={cn(
              "min-w-0 truncate font-semibold text-fg-strong",
              variant === "compact"
                ? "text-compact-h1 tracking-compact-h1"
                : "text-detail-h1 tracking-detail-h1"
            )}
          >
            {title}
          </div>
          {count !== undefined ? (
            <span
              data-slot="page-head-count"
              data-testid={countTestId}
              className="inline-flex h-count-chip min-w-count-chip items-center justify-center rounded-mono-badge bg-canvas-soft px-1.5 font-mono text-mono-id font-medium tabular-nums text-muted"
            >
              {count}
            </span>
          ) : null}
        </div>
        {pills ? (
          <div data-slot="page-head-pills" className="flex flex-wrap items-center gap-1.5 pt-0.5">
            {pills}
          </div>
        ) : null}
        {meta !== undefined ? (
          <div
            data-slot="page-head-meta"
            className="flex flex-wrap items-center gap-2 text-small-body text-muted"
          >
            {meta}
          </div>
        ) : null}
        {children}
      </div>
      {actions ? (
        <div
          data-slot="page-head-actions"
          className="ml-auto flex shrink-0 items-center gap-2 pt-0.5"
        >
          {actions}
        </div>
      ) : null}
    </header>
  );
}

function PageHeadMetaDot({ className, ...props }: React.ComponentProps<"span">) {
  return (
    <span
      aria-hidden="true"
      data-slot="page-head-meta-dot"
      className={cn("size-0.5 rounded-full bg-faint", className)}
      {...props}
    />
  );
}

const PageHeadCompound = Object.assign(PageHead, {
  MetaDot: PageHeadMetaDot,
});

export { PageHeadCompound as PageHead };
export type { PageHeadProps, PageHeadVariant };
