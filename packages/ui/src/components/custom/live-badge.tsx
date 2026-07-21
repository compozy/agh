"use client";

import * as React from "react";

import { cn } from "../../lib/utils";
import type { PillTone } from "./pill";
import { Pill } from "./pill";

export interface LiveBadgeProps extends React.ComponentProps<"span"> {
  /** Signal tone; defaults to `success` (stream attached and healthy). */
  tone?: PillTone;
  /** Label text; defaults to "Live". */
  label?: React.ReactNode;
  /** Disables the dot pulse (e.g. paused/stale streams). */
  pulse?: boolean;
}

const TONE_TEXT: Record<PillTone, string> = {
  neutral: "text-muted",
  accent: "text-accent-strong",
  success: "text-success",
  warning: "text-warning",
  danger: "text-danger",
  info: "text-info",
};

/**
 * The single live-state vocabulary: pulsing dot + short label. One instance
 * per surface — the section that owns the stream — never one per row.
 * Reduced-motion users get a static dot via the Pill.Dot pulse contract.
 */
function LiveBadge({
  tone = "success",
  label = "Live",
  pulse = true,
  className,
  ...props
}: LiveBadgeProps) {
  return (
    <span
      data-slot="live-badge"
      data-tone={tone}
      className={cn(
        "inline-flex items-center gap-1.5 text-badge font-semibold",
        TONE_TEXT[tone],
        className
      )}
      {...props}
    >
      <Pill.Dot aria-hidden="true" pulse={pulse} tone={tone} />
      {label}
    </span>
  );
}

export { LiveBadge };
