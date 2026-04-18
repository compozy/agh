import { Link } from "@tanstack/react-router";

import { Pill } from "@agh/ui";

import { formatRelativeTime, taskRunStatusTone } from "../lib/task-formatters";
import type { TaskDetailView } from "../types";

import { pillVariantFromTone } from "@/lib/pill-variant";
export interface TasksDetailOverviewPanelProps {
  detail: TaskDetailView;
}

export function TasksDetailOverviewPanel({ detail }: TasksDetailOverviewPanelProps) {
  const record = detail.task;
  const summary = detail.summary;
  const activeRun = summary?.active_run ?? null;
  const childCount = detail.children?.length ?? summary?.child_count ?? 0;
  const dependencyReferences = detail.dependency_references ?? detail.dependencies ?? [];
  const dependencyCount = dependencyReferences.length || summary?.dependency_count || 0;
  const runs = detail.runs ?? [];
  const description = record.description?.trim() ?? "";

  return (
    <section className="flex flex-col gap-6 px-6 py-5" data-testid="tasks-detail-overview">
      <div className="grid gap-4 md:grid-cols-3" data-testid="tasks-detail-overview-counts">
        <Stat label="Children" value={childCount} testId="tasks-detail-overview-children" />
        <Stat
          label="Dependencies"
          value={dependencyCount}
          testId="tasks-detail-overview-dependencies"
        />
        <Stat label="Runs" value={runs.length} testId="tasks-detail-overview-runs" />
      </div>

      {activeRun ? (
        <section
          className="rounded-xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface-elevated)] px-4 py-3"
          data-testid="tasks-detail-active-run"
        >
          <div className="flex items-center justify-between gap-3">
            <div className="flex items-center gap-2">
              <span className="font-mono text-[0.66rem] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
                Active Run
              </span>
              <Pill variant={pillVariantFromTone(taskRunStatusTone(activeRun.status))}>
                {activeRun.status}
              </Pill>
            </div>
            <Link
              className="font-mono text-[0.66rem] uppercase tracking-[0.14em] text-[color:var(--color-accent)] hover:underline"
              data-testid="tasks-detail-active-run-link"
              params={{ id: record.id, runId: activeRun.id }}
              to="/tasks/$id/runs/$runId"
            >
              Open run
            </Link>
          </div>
          <div className="mt-2 flex flex-wrap items-center gap-3 text-xs text-[color:var(--color-text-secondary)]">
            <span className="font-mono text-[color:var(--color-text-primary)]">{activeRun.id}</span>
            <span>
              attempt {activeRun.attempt}
              {activeRun.max_attempts ? ` of ${activeRun.max_attempts}` : ""}
            </span>
            {activeRun.session_id ? (
              <span className="font-mono">session {activeRun.session_id}</span>
            ) : null}
            <span>queued {formatRelativeTime(activeRun.queued_at)}</span>
            {activeRun.started_at ? (
              <span>started {formatRelativeTime(activeRun.started_at)}</span>
            ) : null}
          </div>
        </section>
      ) : null}

      <section
        className="rounded-xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-4 py-4"
        data-testid="tasks-detail-description"
      >
        <h3 className="font-mono text-[0.66rem] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
          Description
        </h3>
        {description ? (
          <p className="mt-2 whitespace-pre-wrap text-sm leading-relaxed text-[color:var(--color-text-primary)]">
            {description}
          </p>
        ) : (
          <p className="mt-2 text-sm italic text-[color:var(--color-text-tertiary)]">
            No description provided.
          </p>
        )}
      </section>
    </section>
  );
}

interface StatProps {
  label: string;
  value: number;
  testId?: string;
}

function Stat({ label, value, testId }: StatProps) {
  return (
    <div
      className="rounded-xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-4 py-3"
      data-testid={testId}
    >
      <p className="font-mono text-[0.6rem] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
        {label}
      </p>
      <p className="mt-1 text-2xl font-semibold text-[color:var(--color-text-primary)]">{value}</p>
    </div>
  );
}
