"use client";

import { CircleCheckIcon, CircleIcon, CircleXIcon } from "lucide-react";
import type * as React from "react";

import { cn } from "../../lib/utils";
import { Spinner } from "../spinner";
import type { ToolCallStatus } from "./tool-call-card";

const LABEL: Record<ToolCallStatus, string> = {
  pending: "Pending",
  in_progress: "Running",
  completed: "Done",
  failed: "Error",
};

const TONE_CLASS: Record<Exclude<ToolCallStatus, "in_progress">, string> = {
  pending: "text-faint",
  completed: "text-success",
  failed: "text-danger",
};

const ICON: Record<Exclude<ToolCallStatus, "in_progress">, React.ElementType> = {
  pending: CircleIcon,
  completed: CircleCheckIcon,
  failed: CircleXIcon,
};

export interface ToolCallStatusIconProps {
  status: ToolCallStatus;
  className?: string;
}

/**
 * Status indicator for `ToolCallCard`. Renders a `Spinner` for `in_progress`
 * and a signal-toned Lucide icon (`Circle` / `CircleCheck` / `CircleX`) for
 * the resolved states. Icon-only by design — no background, no text — so it
 * reads as a flat status badge in the card header.
 */
export function ToolCallStatusIcon({ status, className }: ToolCallStatusIconProps) {
  const label = LABEL[status];
  if (status === "in_progress") {
    return (
      <Spinner
        data-slot="tool-call-card-status"
        data-status={status}
        aria-label={label}
        className={cn("size-4 shrink-0 text-info", className)}
      />
    );
  }
  const Icon = ICON[status];
  return (
    <Icon
      data-slot="tool-call-card-status"
      data-status={status}
      role="img"
      aria-label={label}
      strokeWidth={1.75}
      className={cn("size-4 shrink-0", TONE_CLASS[status], className)}
    />
  );
}
