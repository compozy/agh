import type { ComponentProps } from "react";

import { cn } from "@/lib/utils";

import { StatusDot } from "./status-dot";

type ConnectionStatus = "connected" | "disconnected" | "reconnecting";

interface ConnectionIndicatorProps extends ComponentProps<"div"> {
  status: ConnectionStatus;
}

const statusConfig: Record<
  ConnectionStatus,
  { dotTone: "green" | "danger" | "amber"; label: string }
> = {
  connected: { dotTone: "green", label: "Connected" },
  disconnected: { dotTone: "danger", label: "Offline" },
  reconnecting: { dotTone: "amber", label: "Reconnecting..." },
};

function ConnectionIndicator({ className, status, ...props }: ConnectionIndicatorProps) {
  const config = statusConfig[status];

  return (
    <div className={cn("flex items-center gap-2", className)} {...props}>
      <StatusDot
        className={cn(status === "reconnecting" && "animate-pulse")}
        tone={config.dotTone}
      />
      <span className="font-mono text-[0.62rem] uppercase tracking-[0.2em] text-[color:var(--color-text-label)]">
        {config.label}
      </span>
    </div>
  );
}

export { ConnectionIndicator };
export type { ConnectionStatus };
