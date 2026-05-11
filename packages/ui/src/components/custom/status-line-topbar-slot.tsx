"use client";

import * as React from "react";

import { cn } from "../../lib/utils";
import { ConnectionIndicator, type ConnectionStatus } from "./connection-indicator";
import { Eyebrow } from "./eyebrow";
import type { PillTone } from "./pill";

export interface StatusLineTopbarSlotItem {
  /** Optional structural prefix rendered as an `<Eyebrow>` (uppercase). */
  label?: string;
  /** Item content (count, identifier, short message). */
  value: React.ReactNode;
  /** Tone applied to the value text. Defaults to `neutral`. */
  tone?: PillTone;
  /** Stable React key. Falls back to the array index when omitted. */
  key?: React.Key;
}

export interface StatusLineTopbarSlotProps extends React.ComponentProps<"div"> {
  /** Daemon connection status. */
  status: ConnectionStatus;
  /** Optional override for the connection LED label. */
  daemonLabel?: React.ReactNode;
  /** Typed item array per ADR-015 §1 + N-013 / N-005. Replaces the legacy `ReactNode[]` shape. */
  items: ReadonlyArray<StatusLineTopbarSlotItem>;
}

const TONE_TEXT_CLASS: Record<PillTone, string> = {
  neutral: "text-(--muted)",
  accent: "text-(--accent)",
  success: "text-(--success)",
  warning: "text-(--warning)",
  danger: "text-(--danger)",
  info: "text-(--info)",
};

function StatusLineTopbarSlot({
  status,
  daemonLabel,
  items,
  className,
  ...props
}: StatusLineTopbarSlotProps) {
  return (
    <div
      data-slot="status-line-topbar-slot"
      data-status={status}
      className={cn("flex flex-wrap items-center gap-x-4 gap-y-1", className)}
      {...props}
    >
      <ConnectionIndicator label={daemonLabel} status={status} />
      {items.map((item, index) => {
        const tone: PillTone = item.tone ?? "neutral";
        return (
          <span
            key={item.key ?? index}
            data-slot="status-line-topbar-slot-item"
            data-tone={tone}
            className="flex items-center gap-1.5"
          >
            <span aria-hidden="true" className="text-(--subtle)">
              ·
            </span>
            {item.label ? (
              <Eyebrow data-slot="status-line-topbar-slot-item-label" className="text-(--muted)">
                {item.label}
              </Eyebrow>
            ) : null}
            <span
              data-slot="status-line-topbar-slot-item-value"
              className={cn(
                "font-sans text-[12px] tracking-[-0.005em] tabular-nums",
                TONE_TEXT_CLASS[tone]
              )}
            >
              {item.value}
            </span>
          </span>
        );
      })}
    </div>
  );
}

export { StatusLineTopbarSlot };
