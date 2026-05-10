"use client";

import * as React from "react";

import { cn } from "../../lib/utils";
import type { PillTone } from "./pill";

type IconComponent = React.ComponentType<{ className?: string; size?: number }>;

export interface TimelineEventProps extends Omit<React.ComponentProps<"li">, "title"> {
  title: React.ReactNode;
  /** Body / description rendered under the title. */
  description?: React.ReactNode;
  /** Time stamp (relative or absolute) rendered top-right. */
  time?: React.ReactNode;
  /** Trailing meta row (kind chip, ids, etc.). */
  meta?: React.ReactNode;
  icon?: IconComponent;
  tone?: PillTone;
  /** Show or hide the leading column dot/icon. Defaults to true. */
  hasMarker?: boolean;
}

const TONE_DOT: Record<PillTone, string> = {
  neutral: "bg-(--muted)",
  accent: "bg-(--accent)",
  success: "bg-(--success)",
  warning: "bg-(--warning)",
  danger: "bg-(--danger)",
  info: "bg-(--info)",
};

function TimelineEvent({
  title,
  description,
  time,
  meta,
  icon: Icon,
  tone = "neutral",
  hasMarker = true,
  className,
  children,
  ...props
}: TimelineEventProps) {
  return (
    <li
      data-slot="timeline-event"
      data-tone={tone}
      className={cn("relative flex gap-3 pl-6", className)}
      {...props}
    >
      {hasMarker ? (
        <span
          aria-hidden="true"
          data-slot="timeline-event-marker"
          className="absolute left-2 top-2 inline-flex size-2 -translate-x-1/2 items-center justify-center"
        >
          {Icon ? (
            <span className="inline-flex size-3.5 items-center justify-center rounded-full bg-(--canvas-soft) text-(--muted)">
              <Icon className="size-3" />
            </span>
          ) : (
            <span className={cn("size-1.5 rounded-full", TONE_DOT[tone])} />
          )}
        </span>
      ) : null}
      <div data-slot="timeline-event-body" className="flex min-w-0 flex-1 flex-col gap-1 pb-3">
        <div className="flex min-w-0 items-baseline gap-2">
          <p className="min-w-0 truncate text-[13px] font-medium tracking-[-0.005em] text-(--fg-strong)">
            {title}
          </p>
          {time ? (
            <span
              data-slot="timeline-event-time"
              className="ml-auto shrink-0 font-mono text-[10.5px] tabular-nums text-(--subtle)"
            >
              {time}
            </span>
          ) : null}
        </div>
        {description ? (
          <p data-slot="timeline-event-description" className="text-[12px] text-(--muted)">
            {description}
          </p>
        ) : null}
        {meta ? (
          <div
            data-slot="timeline-event-meta"
            className="flex flex-wrap items-center gap-2 text-[11.5px] text-(--subtle)"
          >
            {meta}
          </div>
        ) : null}
        {children}
      </div>
    </li>
  );
}

export { TimelineEvent };
