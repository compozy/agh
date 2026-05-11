import { Link } from "@tanstack/react-router";
import { Radio } from "lucide-react";

import { DescriptionCard, Metric, Pill, RunCard, Section } from "@agh/ui";

import {
  computeElapsed,
  runCoordinationChannelLabel,
  runIsCoordinated,
  taskLifecyclePhase,
  taskLifecyclePhaseDescription,
  taskOwnerLabel,
  toRunCardStatus,
} from "../lib/task-formatters";
import type { TaskDetailView } from "../types";

export interface TasksDetailOverviewPanelProps {
  detail: TaskDetailView;
}

/**
 * Overview tab — KPI metric grid (3 col / gap-3 ≥ 1100 px, collapses to 1 col),
 * active-run `<RunCard>`, and `<DescriptionCard>` per ADR-007 §9. No
 * `border-l-2 border-l-accent` rail, no Stuck pill, no Watch button, no Block
 * reason placeholder (Out of Scope per ADR-007 §4 / §6 / §8).
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
  const activeChannelLabel = runIsCoordinated(activeRun)
    ? runCoordinationChannelLabel(activeRun)
    : null;
  const lifecyclePhase = taskLifecyclePhase({
    status: record.status,
    approval_state: record.approval_state,
    draft: record.draft,
    active_run: activeRun,
  });

  return (
    <section className="flex w-full flex-col gap-6 px-9 py-7" data-testid="tasks-detail-overview">
      <div
        className="grid grid-cols-1 gap-3 [@media(min-width:1100px)]:grid-cols-3"
        data-testid="tasks-detail-overview-counts"
      >
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

      {activeRun ? (
        <Section
          data-testid="tasks-detail-active-run"
          label="Active run"
          right={
            <Pill.Link
              data-testid="tasks-detail-active-run-link"
              render={
                <Link params={{ id: record.id, runId: activeRun.id }} to="/tasks/$id/runs/$runId" />
              }
            >
              Open run
            </Pill.Link>
          }
        >
          <RunCard
            data-testid="tasks-detail-active-run-card"
            runId={activeRun.id}
            status={toRunCardStatus(activeRun.status)}
            attempt={activeRun.attempt}
            sessionInfo={
              activeRun.session_id ? (
                <span className="inline-flex items-center gap-1.5">
                  <span className="font-mono">session {activeRun.session_id}</span>
                  {activeChannelLabel ? (
                    <Pill
                      data-testid="tasks-detail-active-run-channel"
                      title="Coordination channel is bound to the active run. Channel messages support coordination only -- claim, heartbeat, and terminal status stay in the task service."
                      tone="info"
                    >
                      <span className="inline-flex items-center gap-1">
                        <Radio className="size-3" aria-hidden="true" />
                        Channel: {activeChannelLabel}
                      </span>
                    </Pill>
                  ) : null}
                </span>
              ) : activeChannelLabel ? (
                <Pill
                  data-testid="tasks-detail-active-run-channel"
                  title="Coordination channel is bound to the active run. Channel messages support coordination only -- claim, heartbeat, and terminal status stay in the task service."
                  tone="info"
                >
                  <span className="inline-flex items-center gap-1">
                    <Radio className="size-3" aria-hidden="true" />
                    Channel: {activeChannelLabel}
                  </span>
                </Pill>
              ) : undefined
            }
            queuedAt={activeRun.queued_at ?? undefined}
            startedAt={activeRun.started_at ?? undefined}
            elapsed={computeElapsed(activeRun)}
          />
        </Section>
      ) : (
        <Section data-testid="tasks-detail-active-run-empty" label="Execution">
          <p
            className="text-small-body text-(--muted)"
            data-testid="tasks-detail-active-run-empty-hint"
          >
            {taskLifecyclePhaseDescription(lifecyclePhase)}
          </p>
        </Section>
      )}

      <Section data-testid="tasks-detail-description" label="Description">
        {description ? (
          <DescriptionCard data-testid="tasks-detail-description-card">
            {description}
          </DescriptionCard>
        ) : (
          <p
            data-testid="tasks-detail-description-empty"
            className="text-small-body italic text-(--subtle)"
          >
            No description provided.
          </p>
        )}
      </Section>
    </section>
  );
}
