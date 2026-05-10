"use client";

import * as React from "react";

import { cn } from "../../lib/utils";

type IconComponent = React.ComponentType<{ className?: string; size?: number }>;

export interface SectionProps extends React.ComponentProps<"section"> {
  label?: React.ReactNode;
  note?: React.ReactNode;
  right?: React.ReactNode;
  divided?: boolean;
  bodyClassName?: string;
  count?: number | string;
  icon?: IconComponent;
  tabs?: React.ReactNode;
}

function hasSectionContent(content: React.ReactNode): boolean {
  return content !== undefined && content !== null && content !== false;
}

function Section({
  label,
  note,
  right,
  divided = false,
  bodyClassName,
  className,
  children,
  count,
  icon: Icon,
  tabs,
  ...props
}: SectionProps) {
  const hasLabel = hasSectionContent(label);
  const hasNote = hasSectionContent(note);
  const hasRight = hasSectionContent(right);
  const hasChildren = hasSectionContent(children);
  const hasTabs = hasSectionContent(tabs);
  const hasCount = count !== undefined && count !== null && count !== "";
  const hasHeader = hasLabel || hasNote || hasRight || hasTabs;

  return (
    <section
      data-slot="section"
      className={cn(
        "flex min-w-0 flex-col gap-3",
        divided && "border-t border-(--line) pt-5 first:border-t-0 first:pt-0",
        className
      )}
      {...props}
    >
      {hasHeader ? (
        <header
          data-slot="section-head"
          className="flex flex-col gap-3 border-b border-(--line) pb-2 lg:flex-row lg:items-start lg:justify-between"
        >
          <div className="flex min-w-0 flex-col gap-2">
            {hasLabel ? (
              <div className="flex min-w-0 items-center gap-2">
                {Icon ? (
                  <span
                    aria-hidden="true"
                    data-slot="section-icon"
                    className="inline-flex size-5 shrink-0 items-center justify-center text-(--accent)"
                  >
                    <Icon className="size-3.5" />
                  </span>
                ) : null}
                <h2
                  data-slot="section-label"
                  className="truncate text-[22px] font-medium tracking-[-0.026em] text-(--fg-strong)"
                >
                  {label}
                </h2>
                {hasCount ? (
                  <span
                    data-slot="section-count"
                    className="inline-flex h-[19px] min-w-[19px] items-center justify-center rounded-(--radius-mono-badge) bg-(--canvas-soft) px-1.5 font-mono text-[10.5px] font-medium tabular-nums text-(--muted)"
                  >
                    {count}
                  </span>
                ) : null}
              </div>
            ) : null}
            {hasNote ? (
              <div data-slot="section-note" className="max-w-152 text-small-body text-(--muted)">
                {note}
              </div>
            ) : null}
          </div>
          {hasRight || hasTabs ? (
            <div
              data-slot="section-right"
              className="flex w-full items-center gap-2 self-start lg:w-auto lg:shrink-0"
            >
              {hasTabs ? <div data-slot="section-tabs">{tabs}</div> : null}
              {hasRight ? right : null}
            </div>
          ) : null}
        </header>
      ) : null}
      {hasChildren ? (
        <div data-slot="section-body" className={cn("flex min-w-0 flex-col", bodyClassName)}>
          {children}
        </div>
      ) : null}
    </section>
  );
}

export { Section };
