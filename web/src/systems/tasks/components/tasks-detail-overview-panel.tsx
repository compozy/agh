import { Link } from "@tanstack/react-router";
import { Radio } from "lucide-react";

import { MetadataList, Metric, Pill, Section } from "@agh/ui";
import { pillToneFromLegacyTone } from "@/lib/pill-variant";

import {
  formatRelativeTime,
  runCoordinationChannelLabel,
  runIsCoordinated,
  taskLifecyclePhase,
  taskLifecyclePhaseDescription,
  taskOwnerLabel,
  taskRunStatusTone,
  taskStatusSignal,
} from "../lib/task-formatters";
import type { TaskDetailView } from "../types";

export interface TasksDetailOverviewPanelProps {
  detail: TaskDetailView;
}

/**
 * Overview tab -- three `Metric` cards across the top (children / dependencies /
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
    <section className="flex w-full flex-col gap-6 px-6 py-5" data-testid="tasks-detail-overview">
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
          <div className="flex flex-col gap-2 rounded-xl border border-(--line) bg-(--elevated) px-4 py-3">
            <div className="flex flex-wrap items-center gap-2">
              <Pill.Dot tone={activeSignal.tone} pulse={activeSignal.pulse} />
              <Pill mono>{activeRun.id}</Pill>
              <Pill tone={pillToneFromLegacyTone(taskRunStatusTone(activeRun.status))}>
                {activeRun.status}
              </Pill>
              {activeChannelLabel ? (
                <Pill
                  data-testid="tasks-detail-active-run-channel"
                  title="Coordination channel is bound to the active run. Channel messages support coordination only -- claim, heartbeat, and terminal status stay in the task service."
                  tone={pillToneFromLegacyTone("violet")}
                >
                  <span className="inline-flex items-center gap-1">
                    <Radio className="size-3" aria-hidden="true" />
                    Channel: {activeChannelLabel}
                  </span>
                </Pill>
              ) : null}
            </div>
            <MetadataList className="gap-y-2">
              <MetadataList.Row label="Attempt">
                attempt {activeRun.attempt}
                {activeRun.max_attempts ? ` of ${activeRun.max_attempts}` : ""}
              </MetadataList.Row>
              {activeRun.session_id ? (
                <MetadataList.Row label="Session">
                  <span className="font-mono">session {activeRun.session_id}</span>
                </MetadataList.Row>
              ) : null}
              <MetadataList.Row label="Queued">
                {formatRelativeTime(activeRun.queued_at)}
              </MetadataList.Row>
              {activeRun.started_at ? (
                <MetadataList.Row label="Started">
                  {formatRelativeTime(activeRun.started_at)}
                </MetadataList.Row>
              ) : null}
            </MetadataList>
          </div>
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
          <p className="max-w-prose whitespace-pre-wrap text-small-body leading-relaxed text-(--fg)">
            {description}
          </p>
        ) : (
          <p className="text-small-body italic text-(--subtle)">No description provided.</p>
        )}
      </Section>
    </section>
  );
}
