import { cn } from "@/lib/utils";

import { formatDurationMs } from "../lib/task-formatters";
import type { TaskDashboardView } from "../types";

export interface TasksDashboardCardsProps {
  dashboard: TaskDashboardView;
}

type CardTone = "neutral" | "amber" | "green" | "violet" | "danger";

const TONE_CLASSES: Record<CardTone, string> = {
  neutral: "border-[color:var(--color-divider)]",
  amber: "border-[color:var(--color-divider)]",
  green: "border-[color:var(--color-divider)]",
  violet: "border-[color:var(--color-divider)]",
  danger: "border-[color:var(--color-divider)]",
};

const VALUE_TONE_CLASSES: Record<CardTone, string> = {
  neutral: "text-[color:var(--color-text-primary)]",
  amber: "text-[color:var(--color-warning)]",
  green: "text-[color:var(--color-success)]",
  violet: "text-[color:var(--color-info)]",
  danger: "text-[color:var(--color-danger)]",
};

function healthTone(status: string | undefined): CardTone {
  switch (status) {
    case "ok":
    case "healthy":
      return "green";
    case "warning":
      return "amber";
    case "critical":
    case "error":
      return "danger";
    default:
      return "neutral";
  }
}

export function TasksDashboardCards({ dashboard }: TasksDashboardCardsProps) {
  const { cards, totals, active_runs } = dashboard;

  const inProgressSubtitle = [
    active_runs.running > 0 ? `${active_runs.running} running` : null,
    cards.in_progress.claimed_runs > 0 ? `${cards.in_progress.claimed_runs} claimed` : null,
    cards.in_progress.queued_runs > 0 ? `${cards.in_progress.queued_runs} queued` : null,
    cards.in_progress.starting_runs > 0 ? `${cards.in_progress.starting_runs} starting` : null,
  ]
    .filter(Boolean)
    .join(" · ");

  const blockedSubtitle = [
    cards.blocked.awaiting_dependencies > 0
      ? `${cards.blocked.awaiting_dependencies} deps unresolved`
      : null,
    cards.blocked.awaiting_approval > 0
      ? `${cards.blocked.awaiting_approval} awaiting approval`
      : null,
  ]
    .filter(Boolean)
    .join(" · ");

  const failedSubtitle = [
    cards.failed.failed_runs > 0 ? `${cards.failed.failed_runs} failed runs` : null,
    cards.failed.forced_stops > 0 ? `${cards.failed.forced_stops} forced stops` : null,
  ]
    .filter(Boolean)
    .join(" · ");

  const latency = cards.latency;
  const latencySubtitle = `claim avg ${formatDurationMs(latency.claim_latency_ms.average_ms)} · start avg ${formatDurationMs(latency.start_latency_ms.average_ms)}`;

  return (
    <div
      className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4"
      data-testid="tasks-dashboard-cards"
    >
      <DashboardCard
        detail={inProgressSubtitle || "No active work"}
        label="In Progress"
        liveBadge={active_runs.running > 0 ? "live" : null}
        testId="tasks-dashboard-card-in_progress"
        tone={healthTone(cards.in_progress.health_status)}
        value={cards.in_progress.tasks}
      />
      <DashboardCard
        detail={blockedSubtitle || "No blockers"}
        label="Blocked"
        suffix={totals.tasks_total > 0 ? `of ${totals.tasks_total}` : undefined}
        testId="tasks-dashboard-card-blocked"
        tone={healthTone(cards.blocked.health_status)}
        value={cards.blocked.tasks}
      />
      <DashboardCard
        detail={failedSubtitle || "No failures"}
        label="Failed · 24h"
        testId="tasks-dashboard-card-failed"
        tone={healthTone(cards.failed.health_status)}
        value={cards.failed.failed_runs}
      />
      <DashboardCard
        detail={latencySubtitle}
        label="Avg Claim"
        testId="tasks-dashboard-card-latency"
        tone="neutral"
        valueText={formatDurationMs(latency.claim_latency_ms.average_ms)}
      />
    </div>
  );
}

interface DashboardCardProps {
  label: string;
  detail: string;
  tone: CardTone;
  testId: string;
  value?: number;
  valueText?: string;
  suffix?: string;
  liveBadge?: string | null;
}

function DashboardCard({
  label,
  detail,
  tone,
  testId,
  value,
  valueText,
  suffix,
  liveBadge,
}: DashboardCardProps) {
  const display = valueText ?? (typeof value === "number" ? value.toString() : "—");

  return (
    <section
      className={cn(
        "flex flex-col gap-3 rounded-xl border bg-[color:var(--color-surface)] p-4",
        TONE_CLASSES[tone]
      )}
      data-testid={testId}
    >
      <p className="font-mono text-[0.62rem] uppercase tracking-[0.16em] text-[color:var(--color-text-label)]">
        {label}
      </p>
      <div className="flex items-baseline gap-2">
        <span
          className={cn(
            "text-4xl font-semibold leading-none tracking-[-0.03em]",
            VALUE_TONE_CLASSES[tone]
          )}
          data-testid={`${testId}-value`}
        >
          {display}
        </span>
        {liveBadge ? (
          <span
            className="flex items-center gap-1 font-mono text-[0.6rem] uppercase tracking-[0.12em] text-[color:var(--color-accent)]"
            data-testid={`${testId}-live`}
          >
            <span className="inline-block size-1.5 rounded-full bg-[color:var(--color-accent)]" />
            {liveBadge}
          </span>
        ) : null}
        {suffix ? (
          <span className="text-xs text-[color:var(--color-text-tertiary)]">{suffix}</span>
        ) : null}
      </div>
      <p
        className="text-xs leading-5 text-[color:var(--color-text-secondary)]"
        data-testid={`${testId}-detail`}
      >
        {detail}
      </p>
    </section>
  );
}
