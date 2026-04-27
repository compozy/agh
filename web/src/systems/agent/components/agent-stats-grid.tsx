import { Metric, cn } from "@agh/ui";

import { isAgentSessionFailure } from "../lib/session-status";
import type { SessionPayload } from "@/systems/session";

export interface AgentStatsGridProps {
  sessions: SessionPayload[];
  className?: string;
}

export function AgentStatsGrid({ sessions, className }: AgentStatsGridProps) {
  const totals = computeTotals(sessions);
  return (
    <div
      data-testid="agent-stats-grid"
      className={cn("grid gap-3 sm:grid-cols-2 xl:grid-cols-4", className)}
    >
      <Metric
        label="Active sessions"
        value={totals.active}
        tone={totals.active > 0 ? "success" : "default"}
        detail={`${totals.total} total`}
        data-testid="agent-stat-active"
      />
      <Metric
        label="Total runtime"
        value={formatDuration(totals.elapsedSeconds)}
        data-testid="agent-stat-runtime"
      />
      <Metric
        label="Failed"
        value={totals.failed}
        tone={totals.failed > 0 ? "danger" : "default"}
        data-testid="agent-stat-failed"
      />
      <Metric
        label="Last activity"
        value={formatRelative(totals.lastActivityMs)}
        data-testid="agent-stat-last-activity"
      />
    </div>
  );
}

interface SessionTotals {
  total: number;
  active: number;
  failed: number;
  elapsedSeconds: number;
  lastActivityMs: number | null;
}

function computeTotals(sessions: SessionPayload[]): SessionTotals {
  let active = 0;
  let failed = 0;
  let elapsedSeconds = 0;
  let lastActivityMs: number | null = null;

  for (const session of sessions) {
    if (session.state === "active") active += 1;
    if (isAgentSessionFailure(session)) {
      failed += 1;
    }
    const elapsed = session.activity?.elapsed_seconds;
    if (typeof elapsed === "number" && Number.isFinite(elapsed) && elapsed > 0) {
      elapsedSeconds += elapsed;
    }
    const candidate = session.activity?.last_activity_at ?? session.updated_at;
    if (candidate) {
      const ts = new Date(candidate).getTime();
      if (Number.isFinite(ts) && (lastActivityMs === null || ts > lastActivityMs)) {
        lastActivityMs = ts;
      }
    }
  }

  return {
    total: sessions.length,
    active,
    failed,
    elapsedSeconds,
    lastActivityMs,
  };
}

function formatDuration(totalSeconds: number): string {
  if (!Number.isFinite(totalSeconds) || totalSeconds <= 0) return "—";
  const total = Math.round(totalSeconds);
  if (total < 60) return `${total}s`;
  const minutes = Math.floor(total / 60);
  if (minutes < 60) {
    const remainder = total % 60;
    return remainder === 0 ? `${minutes}m` : `${minutes}m ${remainder}s`;
  }
  const hours = Math.floor(minutes / 60);
  const remainderMinutes = minutes % 60;
  if (hours < 24) {
    return remainderMinutes === 0 ? `${hours}h` : `${hours}h ${remainderMinutes}m`;
  }
  const days = Math.floor(hours / 24);
  const remainderHours = hours % 24;
  return remainderHours === 0 ? `${days}d` : `${days}d ${remainderHours}h`;
}

function formatRelative(ts: number | null): string {
  if (ts === null || !Number.isFinite(ts)) return "—";
  const diffMs = Date.now() - ts;
  if (diffMs < 0) return "just now";
  const seconds = Math.floor(diffMs / 1000);
  if (seconds < 45) return "just now";
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  if (days < 7) return `${days}d ago`;
  return new Date(ts).toLocaleDateString();
}
