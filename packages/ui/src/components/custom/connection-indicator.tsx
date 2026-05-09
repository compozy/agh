"use client";

import * as React from "react";

import { cn } from "../../lib/utils";
import { Pill, type PillDotProps, type PillTone } from "./pill";

type ConnectionStatus = "connected" | "connecting" | "disconnected" | "error";

interface ConnectionIndicatorProps extends React.ComponentProps<"div"> {
  status: ConnectionStatus;
  label?: React.ReactNode;
}

interface ConnectionIndicatorDotProps extends Omit<PillDotProps, "tone" | "pulse"> {
  status?: ConnectionStatus;
}

interface ConnectionIndicatorLabelProps extends React.ComponentProps<"span"> {
  status?: ConnectionStatus;
}

interface StatusConfig {
  tone: PillTone;
  label: string;
  pulse: boolean;
}

const STATUS_CONFIG: Record<ConnectionStatus, StatusConfig> = {
  connected: { tone: "success", label: "Connected", pulse: false },
  connecting: { tone: "warning", label: "Connecting", pulse: true },
  disconnected: { tone: "danger", label: "Disconnected", pulse: false },
  error: { tone: "danger", label: "Connection error", pulse: false },
};

const ConnectionIndicatorContext = React.createContext<{
  label?: React.ReactNode;
  status: ConnectionStatus;
} | null>(null);

function useConnectionIndicatorContext(status?: ConnectionStatus) {
  const context = React.useContext(ConnectionIndicatorContext);
  if (status !== undefined) return { label: undefined, status };
  if (context) return context;
  return { label: undefined, status: "disconnected" as const };
}

function ConnectionIndicator({
  status,
  label,
  className,
  children,
  ...props
}: ConnectionIndicatorProps) {
  return (
    <ConnectionIndicatorContext.Provider value={{ label, status }}>
      <div
        aria-live="polite"
        className={cn("inline-flex items-center gap-2", className)}
        data-slot="connection-indicator"
        data-status={status}
        role="status"
        {...props}
      >
        {children ?? (
          <>
            <ConnectionIndicatorDot />
            <ConnectionIndicatorLabel />
          </>
        )}
      </div>
    </ConnectionIndicatorContext.Provider>
  );
}

function ConnectionIndicatorDot({ status, className, ...props }: ConnectionIndicatorDotProps) {
  const context = useConnectionIndicatorContext(status);
  const config = STATUS_CONFIG[context.status];

  return (
    <Pill.Dot
      aria-hidden="true"
      className={className}
      data-slot="connection-indicator-dot"
      data-status={context.status}
      pulse={config.pulse}
      tone={config.tone}
      {...props}
    />
  );
}

function ConnectionIndicatorLabel({
  status,
  className,
  children,
  ...props
}: ConnectionIndicatorLabelProps) {
  const context = useConnectionIndicatorContext(status);
  const config = STATUS_CONFIG[context.status];

  return (
    <span
      className={cn(
        "font-mono text-eyebrow font-medium uppercase tracking-badge text-(--color-text-label)",
        className
      )}
      data-slot="connection-indicator-label"
      data-status={context.status}
      {...props}
    >
      {children ?? context.label ?? config.label}
    </span>
  );
}

const ConnectionIndicatorCompound = Object.assign(ConnectionIndicator, {
  Dot: ConnectionIndicatorDot,
  Label: ConnectionIndicatorLabel,
});

export { ConnectionIndicatorCompound as ConnectionIndicator, STATUS_CONFIG };
export type {
  ConnectionIndicatorDotProps,
  ConnectionIndicatorLabelProps,
  ConnectionIndicatorProps,
  ConnectionStatus,
};
