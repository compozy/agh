import { type ConnectionStatus, Pill, cn } from "@agh/ui";

import { type HealthPayload, useDaemonHealth } from "@/systems/status";

export type RuntimeConnectionTone = "success" | "danger";

export interface RuntimeConnectionIndicatorState {
  tone: RuntimeConnectionTone;
  pulse: boolean;
  label: string;
}

export interface RuntimeConnectionIndicatorProps extends React.ComponentProps<"div"> {
  /** Override the resolved connection status. Tests inject this to bypass TanStack Query. */
  status?: ConnectionStatus;
  /** Override the daemon health status. Tests inject this to bypass TanStack Query. */
  healthStatus?: HealthPayload["status"];
  /** Hide the textual label and render only the dot (sidebar collapsed / rail-only mode). */
  dotOnly?: boolean;
}

/**
 * Encapsulates tone-and-pulse rule for the daemon connection LED:
 *  - success solid → daemon reachable and healthy
 *  - success pulse → daemon reachable but degraded
 *  - danger solid  → daemon unreachable
 *
 * Single owner of the connection LED across the runtime shell. The sidebar
 * footer mounts this exactly once; no other surface renders a daemon LED.
 */
export function resolveRuntimeConnectionState(
  status: ConnectionStatus,
  healthStatus: HealthPayload["status"] | undefined
): RuntimeConnectionIndicatorState {
  if (status === "connected") {
    if (healthStatus === "degraded") {
      return { tone: "success", pulse: true, label: "Degraded" };
    }
    return { tone: "success", pulse: false, label: "Connected" };
  }
  if (status === "connecting") {
    return { tone: "success", pulse: true, label: "Connecting" };
  }
  if (status === "error") {
    return { tone: "danger", pulse: false, label: "Connection error" };
  }
  return { tone: "danger", pulse: false, label: "Disconnected" };
}

export function RuntimeConnectionIndicator({
  status: statusOverride,
  healthStatus: healthStatusOverride,
  dotOnly = false,
  className,
  ...rest
}: RuntimeConnectionIndicatorProps) {
  const { connectionStatus, health } = useDaemonHealth();
  const status = statusOverride ?? connectionStatus;
  const healthStatus = healthStatusOverride ?? health?.status;
  const { tone, pulse, label } = resolveRuntimeConnectionState(status, healthStatus);
  return (
    <div
      aria-live="polite"
      className={cn("inline-flex items-center gap-2", className)}
      data-slot="connection-indicator"
      data-status={status}
      data-pulse={pulse ? "true" : "false"}
      data-tone={tone}
      data-variant={dotOnly ? "rail-dot" : "footer"}
      data-testid="runtime-connection-indicator"
      role="status"
      {...rest}
    >
      <Pill.Dot
        aria-hidden="true"
        data-slot="connection-indicator-dot"
        data-tone={tone}
        pulse={pulse}
        tone={tone}
      />
      {dotOnly ? null : (
        <span className="eyebrow text-muted" data-slot="connection-indicator-label">
          {label}
        </span>
      )}
    </div>
  );
}
