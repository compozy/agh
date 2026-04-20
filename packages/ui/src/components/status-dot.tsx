"use client";

import { useReducedMotionConfig } from "motion/react";
import * as React from "react";

import { cn } from "../lib/utils";

export type StatusDotTone = "success" | "warning" | "danger" | "info" | "accent" | "neutral";

export type StatusDotSize = "sm" | "md";

export interface StatusDotProps extends Omit<React.ComponentProps<"span">, "color"> {
  tone?: StatusDotTone;
  pulse?: boolean;
  size?: StatusDotSize;
}

const TONE_COLOR: Record<StatusDotTone, string> = {
  success: "var(--color-success)",
  warning: "var(--color-warning)",
  danger: "var(--color-danger)",
  info: "var(--color-info)",
  accent: "var(--color-accent)",
  neutral: "var(--color-text-tertiary)",
};

const SIZE_CLASS: Record<StatusDotSize, string> = {
  sm: "size-1.5",
  md: "size-2",
};

/**
 * Tinted signal dot — `tone` maps to a semantic color, optional `pulse` drives a
 * subtle opacity loop. Respects `prefers-reduced-motion` via `useReducedMotion`.
 * Mirrors `.dot` in `docs/design/web-inspiration/styles/app.css` and DESIGN.md §4.
 */
function StatusDot({
  tone = "neutral",
  pulse = false,
  size = "md",
  className,
  style,
  ...props
}: StatusDotProps) {
  const reduced = useReducedMotionConfig();
  const shouldAnimate = pulse && !reduced;
  return (
    <span
      aria-hidden="true"
      data-slot="status-dot"
      data-tone={tone}
      data-size={size}
      data-pulse={shouldAnimate ? "true" : undefined}
      className={cn(
        "inline-block shrink-0 rounded-full",
        SIZE_CLASS[size],
        shouldAnimate && "animate-pulse",
        className
      )}
      style={{ backgroundColor: TONE_COLOR[tone], ...style }}
      {...props}
    />
  );
}

export { StatusDot };
