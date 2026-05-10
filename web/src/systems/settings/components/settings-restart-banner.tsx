import { Loader2, RefreshCw, ShieldAlert, X } from "lucide-react";

import { Alert, AlertDescription, Button, cn } from "@agh/ui";

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

  const tone = restart.isFailed
    ? "danger"
    : restart.isSuccessful
      ? "success"
      : restart.isPolling
        ? "info"
        : "warning";

  const variant = tone === "danger" ? "destructive" : tone;

  const message = restart.isFailed
    ? `Daemon restart failed${restart.failureReason ? `: ${restart.failureReason}` : ""}`
    : restart.isSuccessful
      ? "Daemon restarted successfully"
      : restart.isPolling
        ? `Restarting daemon${restart.status ? ` · ${restart.status}` : ""}`
        : "Changes saved. Restart the daemon to apply.";
  const activeSessionLabel =
    restart.activeSessionCount > 0
      ? `${restart.activeSessionCount} active ${
          restart.activeSessionCount === 1 ? "session" : "sessions"
        }`
      : null;

  const Icon = restart.isPolling ? Loader2 : ShieldAlert;

  return (
    <Alert
      variant={variant}
      role={tone === "danger" ? "alert" : "status"}
      className={cn(
        "flex flex-wrap items-center justify-between gap-3 rounded-none border-x-0 border-t-0 px-8 py-3 md:px-10",
        className
      )}
      data-testid={`settings-page-${slug}-restart-banner`}
      data-tone={tone}
    >
      <div className="flex min-w-0 flex-1 items-center gap-2">
        <Icon
          className={cn("size-4 shrink-0", restart.isPolling && "animate-spin")}
          aria-hidden="true"
        />
        <AlertDescription className="flex min-w-0 flex-wrap items-center gap-2 text-sm">
          <span className="truncate" data-testid={`settings-page-${slug}-restart-banner-message`}>
            {message}
          </span>
          {restart.operationId ? (
            <span
              className="font-mono text-badge font-semibold tracking-badge text-(--muted)"
              data-testid={`settings-page-${slug}-restart-banner-op`}
            >
              {restart.operationId}
            </span>
          ) : null}
          {activeSessionLabel ? (
            <span
              className="font-mono text-badge font-semibold tracking-badge text-(--muted)"
              data-testid={`settings-page-${slug}-restart-banner-active-sessions`}
            >
              {activeSessionLabel}
            </span>
          ) : null}
        </AlertDescription>
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
            {restart.isTriggerPending ? "Starting..." : "Restart daemon"}
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
    </Alert>
  );
}

export { SettingsRestartBanner };
export type { RestartBannerState };
