import { useMemo, type ReactNode } from "react";

import { Play, Square, Trash2 } from "lucide-react";

import { Button, Pill, Spinner, useTopbarSlot, type PillTone } from "@agh/ui";

import type { SessionBadge, SessionPayload, SessionState } from "@/systems/session";

interface StateSignal {
  tone: PillTone;
  pulse?: boolean;
  label: string;
}

const BADGE_SIGNAL: Record<SessionBadge, StateSignal> = {
  running: { tone: "success", pulse: true, label: "running" },
  idle: { tone: "info", label: "idle" },
  unhealthy: { tone: "warning", label: "unhealthy" },
  hung: { tone: "danger", pulse: true, label: "hung" },
  "waiting-for-auth": { tone: "warning", label: "waiting-for-auth" },
  stopped: { tone: "neutral", label: "stopped" },
  failed: { tone: "danger", label: "failed" },
  unknown: { tone: "neutral", label: "unknown" },
};

const STATE_BADGE_FALLBACK: Record<SessionState, SessionBadge> = {
  active: "idle",
  starting: "running",
  stopping: "running",
  stopped: "stopped",
};

interface UseSessionTopbarSlotInput {
  session: SessionPayload;
  isDeleting: boolean;
  isStopping: boolean;
  isResuming: boolean;
  onDelete: () => void;
  onStop: () => void;
  onResume: () => void;
}

/**
 * Composes the session detail-route topbar slot — agent name as
 * the slot title, daemon badge + provider as the meta slot, and the lifecycle
 * controls (delete/stop/attach) as the actions slot.
 */
export function useSessionTopbarSlot({
  session,
  isDeleting,
  isStopping,
  isResuming,
  onDelete,
  onStop,
  onResume,
}: UseSessionTopbarSlotInput): void {
  const badge = session.badge ?? STATE_BADGE_FALLBACK[session.state] ?? "unknown";
  const signal = BADGE_SIGNAL[badge] ?? BADGE_SIGNAL.unknown;
  const providerLabel = session.provider?.trim();
  const isActive = session.state === "active" || session.state === "starting";
  const isAttachable = session.attachable === true;
  const controlsBusy = isStopping || isResuming || isDeleting;

  const meta = useMemo<ReactNode>(
    () => (
      <span data-testid="session-topbar-meta" className="flex min-w-0 items-center gap-2">
        <Pill.Dot
          size="md"
          tone={signal.tone}
          pulse={signal.pulse}
          data-testid="agent-status-dot"
          aria-label={`Session badge: ${signal.label}`}
        />
        <span data-testid="session-topbar-badge" className="font-mono text-eyebrow text-faint">
          {signal.label}
        </span>
        {providerLabel ? (
          <>
            <span aria-hidden="true" className="text-subtle">
              ·
            </span>
            <span
              data-testid="session-topbar-provider"
              className="font-mono text-eyebrow text-faint"
            >
              {providerLabel}
            </span>
          </>
        ) : null}
        {session.name?.trim() && session.name.trim() !== session.id ? (
          <>
            <span aria-hidden="true" className="text-subtle">
              ·
            </span>
            <span data-testid="session-topbar-name" className="truncate text-eyebrow text-muted">
              {session.name.trim()}
            </span>
          </>
        ) : null}
      </span>
    ),
    [signal.tone, signal.pulse, signal.label, providerLabel, session.id, session.name]
  );

  const actions = useMemo<ReactNode>(
    () => (
      <div className="flex shrink-0 items-center gap-1" data-testid="session-topbar-actions">
        <Button
          type="button"
          variant="ghost"
          size="icon-sm"
          onClick={onDelete}
          disabled={controlsBusy}
          data-testid="delete-button"
          aria-label="Delete session"
        >
          {isDeleting ? <Spinner className="size-3" /> : <Trash2 className="size-3" />}
        </Button>
        {isActive ? (
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            onClick={onStop}
            disabled={controlsBusy && !isStopping}
            data-testid="stop-button"
            aria-label="Stop session"
          >
            {isStopping ? <Spinner className="size-3" /> : <Square className="size-3" />}
          </Button>
        ) : null}
        {isAttachable ? (
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            onClick={onResume}
            disabled={controlsBusy && !isResuming}
            data-testid="resume-button"
            aria-label="Attach session"
          >
            {isResuming ? <Spinner className="size-3" /> : <Play className="size-3" />}
          </Button>
        ) : null}
      </div>
    ),
    [
      controlsBusy,
      isActive,
      isAttachable,
      isDeleting,
      isStopping,
      isResuming,
      onDelete,
      onStop,
      onResume,
    ]
  );

  const slot = useMemo(
    () => ({ title: session.agent_name, meta, actions }),
    [session.agent_name, meta, actions]
  );

  useTopbarSlot(slot);
}
