import type { RestartBannerProps, RestartBannerTone } from "@agh/ui";

export type SettingsRestartBannerProps = RestartBannerProps & {
  "data-testid": string;
};

export interface SettingsRestartViewState {
  isVisible: boolean;
  isRestartRequired: boolean;
  isPolling: boolean;
  isSuccessful: boolean;
  isFailed: boolean;
  operationId: string | null;
  status: string | null;
  failureReason?: string;
  activeSessionCount: number;
  isTriggerPending: boolean;
  trigger: () => void;
  dismiss: () => void;
}

function toneFor(state: SettingsRestartViewState): RestartBannerTone {
  if (state.isFailed) return "danger";
  if (state.isSuccessful) return "success";
  if (state.isPolling) return "info";
  return "warning";
}

function messageFor(state: SettingsRestartViewState): string {
  if (state.isFailed) {
    return `Daemon restart failed${state.failureReason ? `: ${state.failureReason}` : ""}`;
  }
  if (state.isSuccessful) return "Daemon restarted successfully";
  if (state.isPolling) {
    return `Restarting daemon${state.status ? ` · ${state.status}` : ""}`;
  }
  return "Changes saved. Restart the daemon to apply.";
}

function activeSessionLabel(state: SettingsRestartViewState): string | null {
  if (state.activeSessionCount <= 0) return null;
  const noun = state.activeSessionCount === 1 ? "session" : "sessions";
  return `${state.activeSessionCount} active ${noun}`;
}

export function restartBannerPropsFor(
  slug: string,
  state: SettingsRestartViewState
): SettingsRestartBannerProps | null {
  if (!state.isVisible) return null;
  const tone = toneFor(state);
  const sessions = activeSessionLabel(state);
  const detailParts: React.ReactNode[] = [];
  if (state.operationId) {
    detailParts.push(
      <span
        key="op"
        data-testid={`settings-page-${slug}-restart-banner-op`}
        className="font-mono text-[10.5px] font-medium text-(--muted)"
      >
        {state.operationId}
      </span>
    );
  }
  if (sessions) {
    detailParts.push(
      <span
        key="sessions"
        data-testid={`settings-page-${slug}-restart-banner-active-sessions`}
        className="font-mono text-[10.5px] font-medium text-(--muted)"
      >
        {sessions}
      </span>
    );
  }
  const showRestartNow = state.isRestartRequired && !state.isPolling && !state.isSuccessful;
  return {
    "data-testid": `settings-page-${slug}-restart-banner`,
    tone,
    message: (
      <span data-testid={`settings-page-${slug}-restart-banner-message`}>{messageFor(state)}</span>
    ),
    detail: detailParts.length > 0 ? <>{detailParts}</> : undefined,
    busy: state.isPolling,
    restartNow: showRestartNow ? state.trigger : undefined,
    isPending: state.isTriggerPending,
    onDismiss: state.isSuccessful || state.isFailed ? state.dismiss : undefined,
  };
}
