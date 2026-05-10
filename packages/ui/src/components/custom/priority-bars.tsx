"use client";

import * as React from "react";

import { cn } from "../../lib/utils";
import type { PillTone } from "./pill";

const PRIORITY_LEVELS = ["low", "medium", "high", "urgent"] as const;
export type PriorityLevel = (typeof PRIORITY_LEVELS)[number];

export interface PriorityBarsProps extends React.ComponentProps<"span"> {
  level: PriorityLevel;
  tone?: PillTone;
  ariaLabel?: string;
}

const TONE_FILL: Record<PillTone, string> = {
  neutral: "bg-(--muted)",
  accent: "bg-(--accent)",
  success: "bg-(--success)",
  warning: "bg-(--warning)",
  danger: "bg-(--danger)",
  info: "bg-(--info)",
};

const PRIORITY_FILL_COUNT: Record<PriorityLevel, number> = {
  low: 1,
  medium: 2,
  high: 3,
  urgent: 4,
};

function PriorityBars({
  level,
  tone = "accent",
  ariaLabel,
  className,
  ...props
}: PriorityBarsProps) {
  const fillCount = PRIORITY_FILL_COUNT[level];
  return (
    <span
      data-slot="priority-bars"
      role="img"
      aria-label={ariaLabel ?? `${level} priority`}
      data-level={level}
      className={cn("inline-flex items-end gap-px", className)}
      {...props}
    >
      {PRIORITY_LEVELS.map((_, index) => {
        const filled = index < fillCount;
        return (
          <span
            key={index}
            data-slot="priority-bars-bar"
            data-filled={filled ? "true" : undefined}
            aria-hidden="true"
            className={cn(
              "w-[2px] rounded-[1px]",
              filled ? TONE_FILL[tone] : "bg-(--line)",
              index === 0 && "h-1.5",
              index === 1 && "h-2.5",
              index === 2 && "h-3.5",
              index === 3 && "h-4"
            )}
          />
        );
      })}
    </span>
  );
}

export { PriorityBars };
