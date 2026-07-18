import { Pill, type PillTone } from "@agh/ui";

import type { SessionBadge, SessionPayload, SessionState } from "../types";

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

export interface SessionStatusLineProps {
  session: SessionPayload;
}

/**
 * Daemon badge + agent/provider identity line for the session head band.
 * Status summaries are body chrome — never topbar content (route chrome §04).
 */
export function SessionStatusLine({ session }: SessionStatusLineProps) {
  const badge = session.badge ?? STATE_BADGE_FALLBACK[session.state] ?? "unknown";
  const signal = BADGE_SIGNAL[badge] ?? BADGE_SIGNAL.unknown;
  const agentLabel = session.agent_name.trim();
  const providerLabel = session.provider?.trim();

  return (
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
}
