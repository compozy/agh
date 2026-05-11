"use client";

import * as React from "react";

import { cn } from "../../lib/utils";
import type { PillTone } from "./pill";

export interface ReviewRowProps extends Omit<React.ComponentProps<"div">, "title"> {
  /** Avatar / leading visual node. */
  leading?: React.ReactNode;
  title: React.ReactNode;
  description?: React.ReactNode;
  /** Status pill + secondary metadata. */
  meta?: React.ReactNode;
  actions?: React.ReactNode;
  tone?: PillTone;
}

const TONE_BORDER: Record<PillTone, string> = {
  neutral: "border-(--line)",
  accent: "border-(--accent)/40",
  success: "border-(--success)/40",
  warning: "border-(--warning)/40",
  danger: "border-(--danger)/40",
  info: "border-(--info)/40",
};

function ReviewRow({
  leading,
  title,
  description,
  meta,
  actions,
  tone = "neutral",
  className,
  children,
  ...props
}: ReviewRowProps) {
  return (
    <div
      data-slot="review-row"
      data-tone={tone}
      className={cn(
        "flex min-w-0 items-start gap-3 rounded-(--radius) border bg-(--canvas-soft) px-3 py-2.5",
        TONE_BORDER[tone],
        className
      )}
      {...props}
    >
      {leading ? (
        <div data-slot="review-row-leading" className="shrink-0">
          {leading}
        </div>
      ) : null}
      <div data-slot="review-row-body" className="min-w-0 flex-1 flex flex-col gap-1">
        <div data-slot="review-row-title-line" className="flex min-w-0 items-center gap-2">
          <p className="min-w-0 truncate text-[13px] font-medium tracking-eyebrow text-(--fg-strong)">
            {title}
          </p>
        </div>
        {description ? (
          <p data-slot="review-row-description" className="text-[12px] text-(--muted)">
            {description}
          </p>
        ) : null}
        {meta ? (
          <div
            data-slot="review-row-meta"
            className="flex flex-wrap items-center gap-2 text-[11.5px] text-(--subtle)"
          >
            {meta}
          </div>
        ) : null}
        {children}
      </div>
      {actions ? (
        <div data-slot="review-row-actions" className="shrink-0 flex items-center gap-2">
          {actions}
        </div>
      ) : null}
    </div>
  );
}

export { ReviewRow };
