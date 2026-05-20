import { type ConnectionStatus, Pill, cn } from "@agh/ui";

import { useDaemonConnectionStatus } from "@/systems/status/hooks/use-daemon-connection-status";

import { useNavCounts } from "../hooks/use-nav-counts";

export type RuntimeConnectionTone = "success" | "danger";

export interface RuntimeConnectionIndicatorState {
  tone: RuntimeConnectionTone;
  pulse: boolean;
  label: string;
}

export interface RuntimeConnectionIndicatorProps extends React.ComponentProps<"div"> {
  /** Override the resolved connection status. Tests inject this to bypass TanStack Query. */
  status?: ConnectionStatus;
  /** Force the degraded heartbeat path. Tests inject this to bypass useNavCounts(). */
  degraded?: boolean;
  /** Hide the textual label and render only the dot (sidebar collapsed / rail-only mode). */
  dotOnly?: boolean;
}

/**
 * Encapsulates tone-and-pulse rule for the daemon connection LED:
 *  - success solid → daemon reachable, recent activity within heartbeat window
 *  - success pulse → daemon reachable, degraded heartbeat
 *  - danger solid  → daemon unreachable
 *
 * Single owner of the connection LED across the runtime shell. The sidebar
 * footer mounts this exactly once; no other surface renders a daemon LED.
 */
export function resolveRuntimeConnectionState(
  status: ConnectionStatus,
  degraded: boolean
): RuntimeConnectionIndicatorState {
  if (status === "connected") {
    if (degraded) return { tone: "success", pulse: true, label: "Degraded" };
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
  degraded: degradedOverride,
  dotOnly = false,
  className,
  ...rest
}: RuntimeConnectionIndicatorProps) {
  const resolvedStatus = useDaemonConnectionStatus();
  const status = statusOverride ?? resolvedStatus;
  const navCounts = useNavCounts();
  const resolvedDegraded =
    degradedOverride ?? Object.values(navCounts.counts).some(entry => entry?.stale === true);
  const { tone, pulse, label } = resolveRuntimeConnectionState(status, resolvedDegraded);
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
