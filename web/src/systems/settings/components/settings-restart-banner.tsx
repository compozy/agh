import { Loader2, RefreshCw, ShieldAlert, X } from "lucide-react";

import { Button } from "@agh/ui";
import { cn } from "@/lib/utils";

interface RestartBannerState {
  isVisible: boolean;
  isRestartRequired: boolean;
  isPolling: boolean;
  isSuccessful: boolean;
  isFailed: boolean;
  operationId: string | null;
  status: string | null;
  failureReason?: string;
  activeSessionCount: number;
  trigger: () => void;
  isTriggerPending: boolean;
  triggerError: unknown;
  dismiss: () => void;
}

interface SettingsRestartBannerProps {
  slug: string;
  restart: RestartBannerState;
  className?: string;
}

function SettingsRestartBanner({ slug, restart, className }: SettingsRestartBannerProps) {
  if (!restart.isVisible) {
    return null;
  }

  const tone = restart.isFailed ? "danger" : restart.isSuccessful ? "success" : "warning";

  const message = restart.isFailed
    ? `Daemon restart failed${restart.failureReason ? `: ${restart.failureReason}` : ""}`
    : restart.isSuccessful
      ? "Daemon restarted successfully"
      : restart.isPolling
        ? `Restarting daemon${restart.status ? ` · ${restart.status}` : ""}`
        : "Changes saved. Restart the daemon to apply.";

  return (
    <div
      className={cn(
        "flex flex-wrap items-center justify-between gap-3 border-b px-8 py-3 text-sm",
        tone === "danger" &&
          "border-[color:var(--color-danger)] bg-[color:var(--color-danger-tint)] text-[color:var(--color-danger)]",
        tone === "success" &&
          "border-[color:var(--color-success)] bg-[color:var(--color-success-tint)] text-[color:var(--color-success)]",
        tone === "warning" &&
          "border-[color:var(--color-warning)] bg-[color:var(--color-warning-tint)] text-[color:var(--color-warning)]",
        className
      )}
      data-testid={`settings-page-${slug}-restart-banner`}
      data-tone={tone}
      role={tone === "danger" ? "alert" : "status"}
    >
      <div className="flex min-w-0 items-center gap-2">
        {restart.isPolling ? (
          <Loader2 className="size-4 shrink-0 animate-spin" aria-hidden="true" />
        ) : (
          <ShieldAlert className="size-4 shrink-0" aria-hidden="true" />
        )}
        <span className="truncate" data-testid={`settings-page-${slug}-restart-banner-message`}>
          {message}
        </span>
        {restart.operationId ? (
          <span
            className="font-mono text-[0.64rem] tracking-[0.1em] text-[color:var(--color-text-label)]"
            data-testid={`settings-page-${slug}-restart-banner-op`}
          >
            {restart.operationId}
          </span>
        ) : null}
      </div>
      <div className="flex items-center gap-2">
        {restart.isRestartRequired && !restart.isPolling && !restart.isSuccessful ? (
          <Button
            type="button"
            variant="outline"
            size="sm"
            data-testid={`settings-page-${slug}-restart-banner-trigger`}
            disabled={restart.isTriggerPending}
            onClick={restart.trigger}
          >
            <RefreshCw className="size-3.5" />
            {restart.isTriggerPending ? "Starting…" : "Restart daemon"}
          </Button>
        ) : null}
        {restart.isSuccessful || restart.isFailed ? (
          <Button
            type="button"
            variant="ghost"
            size="sm"
            data-testid={`settings-page-${slug}-restart-banner-dismiss`}
            onClick={restart.dismiss}
          >
            <X className="size-3.5" />
            Dismiss
          </Button>
        ) : null}
      </div>
    </div>
  );
}

export { SettingsRestartBanner };
export type { RestartBannerState };
