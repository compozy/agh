"use client";

import * as React from "react";

import { cn } from "../lib/utils";
import { StatusDot, type StatusDotTone } from "./status-dot";

export type ConnectionStatus = "connected" | "disconnected" | "reconnecting";

export interface ConnectionIndicatorProps extends React.ComponentProps<"div"> {
  status: ConnectionStatus;
  /** Override the default label for this status. */
  label?: React.ReactNode;
}

interface StatusConfig {
  tone: StatusDotTone;
  label: string;
  pulse: boolean;
}

const STATUS_CONFIG: Record<ConnectionStatus, StatusConfig> = {
  connected: { tone: "success", label: "Connected", pulse: false },
  disconnected: { tone: "danger", label: "Disconnected", pulse: false },
  reconnecting: { tone: "warning", label: "Reconnecting", pulse: true },
};

/**
 * StatusDot + mono label composite — one canonical shape for daemon / socket
 * connection state across the operator UI. Mirrors DESIGN.md §4 "Status Indicators".
 */
function ConnectionIndicator({ status, label, className, ...props }: ConnectionIndicatorProps) {
  const config = STATUS_CONFIG[status];
  return (
    <div
      data-slot="connection-indicator"
      data-status={status}
      className={cn("inline-flex items-center gap-2", className)}
      {...props}
    >
      <StatusDot tone={config.tone} pulse={config.pulse} />
      <span
        data-slot="connection-indicator-label"
        className="font-mono text-[11px] font-medium uppercase tracking-[0.08em] text-[color:var(--color-text-label)]"
      >
        {label ?? config.label}
      </span>
    </div>
  );
}

export { ConnectionIndicator };
