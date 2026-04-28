import { Pill, type PillTone, cn } from "@agh/ui";
import * as React from "react";

export type ConnectionStatus = "connected" | "disconnected" | "reconnecting";

export interface ConnectionIndicatorProps extends React.ComponentProps<"div"> {
  status: ConnectionStatus;
  /** Override the default label for this status. */
  label?: React.ReactNode;
}

interface StatusConfig {
  tone: PillTone;
  label: string;
  pulse: boolean;
}

const STATUS_CONFIG: Record<ConnectionStatus, StatusConfig> = {
  connected: { tone: "success", label: "Connected", pulse: false },
  disconnected: { tone: "danger", label: "Disconnected", pulse: false },
  reconnecting: { tone: "warning", label: "Reconnecting", pulse: true },
};

/**
 * `Pill.Dot` + monospace label composite — canonical chrome for daemon /
 * socket connection state across the operator UI. Wraps the dot in an
 * `aria-live=polite` region so screen readers announce reconnects.
 */
export function ConnectionIndicator({
  status,
  label,
  className,
  ...props
}: ConnectionIndicatorProps) {
  const config = STATUS_CONFIG[status];
  return (
    <div
      data-slot="connection-indicator"
      data-status={status}
      role="status"
      aria-live="polite"
      className={cn("inline-flex items-center gap-2", className)}
      {...props}
    >
      <Pill.Dot tone={config.tone} pulse={config.pulse} />
      <span
        data-slot="connection-indicator-label"
        className="font-mono text-[11px] font-medium uppercase tracking-[0.08em] text-(--color-text-label)"
      >
        {label ?? config.label}
      </span>
    </div>
  );
}
