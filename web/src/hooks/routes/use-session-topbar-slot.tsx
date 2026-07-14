import { Eraser, Play, Square, Trash2 } from "lucide-react";

import { Button, Pill, Spinner, useTopbarSlot, type PillTone } from "@agh/ui";

import {
  getSessionDisplayTitle,
  isUserControllableSession,
  type SessionBadge,
  type SessionPayload,
  type SessionState,
} from "@/systems/session";

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
  isClearing: boolean;
  canClear: boolean;
  onDelete: () => void;
  onStop: () => void;
  onResume: () => void;
  onClear: () => void;
}

/**
 * Composes the session detail-route topbar slot — persisted session identity as
 * the slot title, daemon badge + agent/provider as the meta slot, and the lifecycle
 * controls (clear/delete/stop/attach) as the actions slot. Clear-conversation
 * lives here rather than beside the composer input so a destructive reset is not
 * one keystroke from the prompt field.
 */
export function useSessionTopbarSlot({
  session,
  isDeleting,
  isStopping,
  isResuming,
  isClearing,
  canClear,
  onDelete,
  onStop,
  onResume,
  onClear,
}: UseSessionTopbarSlotInput): void {
  const badge = session.badge ?? STATE_BADGE_FALLBACK[session.state] ?? "unknown";
  const signal = BADGE_SIGNAL[badge] ?? BADGE_SIGNAL.unknown;
  const agentLabel = session.agent_name.trim();
  const providerLabel = session.provider?.trim();
  const isActive = session.state === "active" || session.state === "starting";
  const isAttachable = session.attachable === true;
  const canResume = isAttachable && isUserControllableSession(session);
  const controlsBusy = isStopping || isResuming || isDeleting;

  const meta = (
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
      {agentLabel ? (
        <>
          <span aria-hidden="true" className="text-subtle">
            ·
          </span>
          <span data-testid="session-topbar-agent" className="truncate text-eyebrow text-muted">
            {agentLabel}
          </span>
        </>
      ) : null}
      {providerLabel ? (
        <>
          <span aria-hidden="true" className="text-subtle">
            ·
          </span>
          <span data-testid="session-topbar-provider" className="font-mono text-eyebrow text-faint">
            {providerLabel}
          </span>
        </>
      ) : null}
    </span>
  );

  const actions = (
    <div className="flex shrink-0 items-center gap-1" data-testid="session-topbar-actions">
      <Button
        type="button"
        variant="ghost"
        size="icon-sm"
        onClick={onClear}
        disabled={!canClear}
        data-testid="composer-clear-button"
        aria-label="Clear conversation"
      >
        {isClearing ? <Spinner className="size-3" /> : <Eraser className="size-3" />}
      </Button>
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
      {canResume ? (
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
  );

  const slot = { title: getSessionDisplayTitle(session), meta, actions };

  useTopbarSlot(slot);
}
