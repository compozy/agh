import { Metric, MetricGrid } from "@agh/ui";

export interface AgentStatsGridProps {
  total: number;
  active: number;
  resumable: number;
  lastActivityAt: string | null;
  className?: string;
}

export function AgentStatsGrid({
  total,
  active,
  resumable,
  lastActivityAt,
  className,
}: AgentStatsGridProps) {
  return (
    <MetricGrid data-testid="agent-stats-grid" className={className}>
      <Metric
        label="Active sessions"
        value={active}
        tone={active > 0 ? "success" : "default"}
        data-testid="agent-stat-active"
      />
      <Metric label="Total sessions" value={total} data-testid="agent-stat-total" />
      <Metric label="Resumable" value={resumable} data-testid="agent-stat-resumable" />
      <Metric
        label="Last activity"
        value={formatRelative(lastActivityAt)}
        data-testid="agent-stat-last-activity"
      />
    </MetricGrid>
  );
}

function formatRelative(value: string | null): string {
  if (!value) return "--";
  const timestamp = new Date(value).getTime();
  if (!Number.isFinite(timestamp)) return "--";
  const diffMs = Date.now() - timestamp;
  if (diffMs < 0) return "just now";
  const seconds = Math.floor(diffMs / 1_000);
  if (seconds < 45) return "just now";
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  if (days < 7) return `${days}d ago`;
  return new Date(timestamp).toLocaleDateString();
}
