"use client";

import * as React from "react";

import { cn } from "../../lib/utils";
import { Pill, type PillDotProps, type PillTone } from "./pill";

type ConnectionStatus = "connected" | "connecting" | "disconnected" | "error";
type ConnectionVariant = "footer" | "rail-dot" | "inline";

interface ConnectionIndicatorProps extends React.ComponentProps<"div"> {
  status: ConnectionStatus;
  label?: React.ReactNode;
  variant?: ConnectionVariant;
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

interface ConnectionIndicatorContextValue {
  label?: React.ReactNode;
  status: ConnectionStatus;
  variant: ConnectionVariant;
}

const ConnectionIndicatorContext = React.createContext<ConnectionIndicatorContextValue | null>(
  null
);

function useConnectionIndicatorContext(status?: ConnectionStatus): ConnectionIndicatorContextValue {
  const context = React.use(ConnectionIndicatorContext);
  if (status !== undefined) return { label: undefined, status, variant: "footer" };
  if (context) return context;
  return { label: undefined, status: "disconnected", variant: "footer" };
}

function ConnectionIndicator({
  status,
  label,
  variant = "footer",
  className,
  children,
  ...props
}: ConnectionIndicatorProps) {
  const value = React.useMemo<ConnectionIndicatorContextValue>(
    () => ({ label, status, variant }),
    [label, status, variant]
  );

  return (
    <ConnectionIndicatorContext.Provider value={value}>
      <div
        aria-live="polite"
        className={cn("inline-flex items-center gap-2", className)}
        data-slot="connection-indicator"
        data-status={status}
        data-variant={variant}
        role="status"
        {...props}
      >
        {children ??
          (variant === "rail-dot" ? (
            <ConnectionIndicatorDot />
          ) : (
            <>
              <ConnectionIndicatorDot />
              <ConnectionIndicatorLabel />
            </>
          ))}
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
      data-variant={context.variant}
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
        context.variant === "inline"
          ? "font-sans text-[12px] tracking-[-0.005em] text-(--muted)"
          : "font-mono text-eyebrow font-medium uppercase tracking-mono text-(--muted)",
        className
      )}
      data-slot="connection-indicator-label"
      data-status={context.status}
      data-variant={context.variant}
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
  ConnectionVariant,
};
