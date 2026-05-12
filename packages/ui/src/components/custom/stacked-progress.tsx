"use client";

import * as React from "react";

import { cn } from "../../lib/utils";
import type { PillTone } from "./pill";

export interface StackedProgressSegment {
  value: number;
  tone?: PillTone;
  label?: React.ReactNode;
}

export interface StackedProgressProps extends React.ComponentProps<"div"> {
  segments: ReadonlyArray<StackedProgressSegment>;
  /** Optional total used to compute share; defaults to the sum of segment values. */
  total?: number;
  ariaLabel?: string;
}

const TONE_CLASS: Record<PillTone, string> = {
  neutral: "bg-muted",
  accent: "bg-accent",
  success: "bg-success",
  warning: "bg-warning",
  danger: "bg-danger",
  info: "bg-info",
};

function StackedProgress({
  segments,
  total,
  ariaLabel,
  className,
  ...props
}: StackedProgressProps) {
  const sum = total ?? segments.reduce((acc, segment) => acc + segment.value, 0);
  const segmentOccurrences = new Map<string, number>();
  return (
    <div
      data-slot="stacked-progress"
      role="img"
      aria-label={ariaLabel}
      className={cn("flex h-1.5 w-full overflow-hidden rounded-pill bg-canvas", className)}
      {...props}
    >
      {segments.map(segment => {
        const ratio = sum > 0 ? Math.max(0, Math.min(1, segment.value / sum)) : 0;
        const tone: PillTone = segment.tone ?? "neutral";
        const baseKey = `${String(segment.label ?? tone)}-${segment.value}`;
        const occurrence = segmentOccurrences.get(baseKey) ?? 0;
        segmentOccurrences.set(baseKey, occurrence + 1);
        if (ratio <= 0) return null;
        return (
          <span
            key={`${baseKey}-${occurrence}`}
            data-slot="stacked-progress-segment"
            data-tone={tone}
            aria-hidden="true"
            className={cn("h-full", TONE_CLASS[tone])}
            style={{ width: `${Math.round(ratio * 100)}%` }}
          />
        );
      })}
    </div>
  );
}

export { StackedProgress };
