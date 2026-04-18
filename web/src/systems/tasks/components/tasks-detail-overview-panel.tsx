import { Link } from "@tanstack/react-router";

import { Metric, MonoBadge, Pill, Section, StatusDot } from "@agh/ui";
import { pillVariantFromTone } from "@/lib/pill-variant";

import {
  formatRelativeTime,
  taskOwnerLabel,
  taskRunStatusTone,
  taskStatusSignal,
} from "../lib/task-formatters";
import type { TaskDetailView } from "../types";

export interface TasksDetailOverviewPanelProps {
  detail: TaskDetailView;
}

/**
 * Overview tab — three `Metric` cards across the top (children / dependencies /
 * runs), then a `Section` for the active run (when present), then a `Section`
 * for the task description. DESIGN.md §4 Metric + Section composition.
 */
export function TasksDetailOverviewPanel({ detail }: TasksDetailOverviewPanelProps) {
  const record = detail.task;
  const summary = detail.summary;
  const activeRun = summary?.active_run ?? null;
  const childCount = detail.children?.length ?? summary?.child_count ?? 0;
  const dependencyReferences = detail.dependency_references ?? detail.dependencies ?? [];
  const dependencyCount = dependencyReferences.length || summary?.dependency_count || 0;
  const runs = detail.runs ?? [];
  const description = record.description?.trim() ?? "";
  const activeSignal = activeRun ? taskStatusSignal(activeRun.status) : null;

  return (
    <section className="flex flex-col gap-6 px-6 py-5" data-testid="tasks-detail-overview">
      <div className="grid gap-4 md:grid-cols-3" data-testid="tasks-detail-overview-counts">
        <Metric
          data-testid="tasks-detail-overview-children"
          label="Children"
          value={childCount}
          subtext={`Owner ${taskOwnerLabel(record.owner)}`}
        />
        <Metric
          data-testid="tasks-detail-overview-dependencies"
          label="Dependencies"
          value={dependencyCount}
        />
        <Metric
          data-testid="tasks-detail-overview-runs"
          label="Runs"
          value={runs.length}
          tone={activeRun ? "accent" : "default"}
        />
      </div>

      {activeRun && activeSignal ? (
        <Section
          data-testid="tasks-detail-active-run"
          label="Active Run"
          right={
            <Link
              className="font-mono text-[11px] uppercase tracking-[0.14em] text-[color:var(--color-accent)] hover:underline"
              data-testid="tasks-detail-active-run-link"
              params={{ id: record.id, runId: activeRun.id }}
              to="/tasks/$id/runs/$runId"
            >
              Open run
            </Link>
          }
        >
          <div className="flex flex-col gap-2 rounded-xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface-elevated)] px-4 py-3">
            <div className="flex items-center gap-2">
              <StatusDot tone={activeSignal.tone} pulse={activeSignal.pulse} />
              <MonoBadge>{activeRun.id}</MonoBadge>
              <Pill variant={pillVariantFromTone(taskRunStatusTone(activeRun.status))}>
                {activeRun.status}
              </Pill>
            </div>
            <div className="flex flex-wrap items-center gap-3 text-[13px] text-[color:var(--color-text-secondary)]">
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
          </div>
        </Section>
      ) : null}

      <Section data-testid="tasks-detail-description" label="Description">
        {description ? (
          <p className="whitespace-pre-wrap text-[13px] leading-relaxed text-[color:var(--color-text-primary)]">
            {description}
          </p>
        ) : (
          <p className="text-[13px] italic text-[color:var(--color-text-tertiary)]">
            No description provided.
          </p>
        )}
      </Section>
    </section>
  );
}
