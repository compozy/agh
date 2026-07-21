import { AlertTriangle } from "lucide-react";

import { Button, cn, Spinner } from "@agh/ui";

import type { SettingsRestartViewState } from "../lib/restart-banner-mapper";

type RestartState = SettingsRestartViewState;

/**
 * Inline restart notice (design system §04): states what happens if you don't
 * act, offers Restart / Not now. Replaces the RestartBanner on settings pages.
 */
export function SettingsRestartNotice({ restart, slug }: { restart: RestartState; slug: string }) {
  if (!restart.isVisible) return null;

  const failed = restart.isFailed;
  const polling = restart.isPolling;
  const successful = restart.isSuccessful;

  let title = "Restart needed";
  let body =
    "Some saved changes apply after the daemon restarts. Running sessions keep their current rules until then.";
  if (polling) {
    title = "Restarting…";
    body = "The daemon comes back in a few seconds. Sessions reconnect on their own.";
  } else if (successful) {
    title = "Restarted";
    body = "All saved changes are now active.";
  } else if (failed) {
    title = "Restart failed";
    body =
      restart.failureReason?.trim() ||
      "The daemon did not come back. Check its logs, then try again.";
  }

  return (
    <div
      className={cn(
        "flex items-start gap-2.5 rounded-md px-3.5 py-3 text-form-label leading-normal text-muted",
        failed ? "bg-danger-tint" : "bg-warning-tint"
      )}
      data-testid={`settings-page-${slug}-restart-notice`}
      role="status"
    >
      {polling ? (
        <Spinner className="mt-0.5 size-3.5 shrink-0 text-warning" />
      ) : (
        <AlertTriangle
          aria-hidden="true"
          className={cn("mt-0.5 size-3.5 shrink-0", failed ? "text-danger" : "text-warning")}
        />
      )}
      <div className="min-w-0 flex-1">
        <span className="block text-ws-name font-medium text-fg-strong">{title}</span>
        {body}
        {restart.activeSessionCount != null && restart.isRestartRequired && !polling ? (
          <span className="mt-0.5 block text-form-hint text-subtle">
            {restart.activeSessionCount} active session
            {restart.activeSessionCount === 1 ? "" : "s"} right now.
          </span>
        ) : null}
      </div>
      {successful ? (
        <div className="ml-2 flex shrink-0 items-center">
          <Button
            data-testid={`settings-page-${slug}-restart-dismiss`}
            onClick={restart.dismiss}
            size="sm"
            type="button"
            variant="ghost"
          >
            Dismiss
          </Button>
        </div>
      ) : !polling ? (
        <div className="ml-2 flex shrink-0 items-center gap-1.5">
          <Button
            data-testid={`settings-page-${slug}-restart-dismiss`}
            onClick={restart.dismiss}
            size="sm"
            type="button"
            variant="ghost"
          >
            Not now
          </Button>
          <Button
            data-testid={`settings-page-${slug}-restart-trigger`}
            disabled={restart.isTriggerPending}
            onClick={() => restart.trigger()}
            size="sm"
            type="button"
            variant="neutral"
          >
            {failed ? "Try again" : "Restart daemon"}
          </Button>
        </div>
      ) : null}
    </div>
  );
}
