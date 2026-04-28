import { Link } from "@tanstack/react-router";
import { AlertCircle, ChevronRight, Inbox, Loader2, Radio } from "lucide-react";

import {
  Empty,
  Pill,
  Section,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@agh/ui";
import { pillToneFromLegacyTone } from "@/lib/pill-variant";

import {
  formatRelativeTime,
  runCoordinationChannelLabel,
  runIsCoordinated,
  taskRunStatusTone,
  taskStatusSignal,
} from "../lib/task-formatters";
import type { TaskRun } from "../types";

export interface TasksDetailRunsPanelProps {
  taskId: string;
  runs: TaskRun[];
  isLoading?: boolean;
  errorMessage?: string | null;
}

export function TasksDetailRunsPanel({
  taskId,
  runs,
  isLoading = false,
  errorMessage = null,
}: TasksDetailRunsPanelProps) {
  if (isLoading && runs.length === 0) {
    return (
      <div
        className="flex min-h-[240px] items-center justify-center"
        data-testid="tasks-detail-runs-loading"
      >
        <Loader2 className="size-5 animate-spin text-[color:var(--color-text-tertiary)]" />
      </div>
    );
  }

  if (errorMessage && runs.length === 0) {
    return (
      <Empty
        icon={AlertCircle}
        title="Unable to load runs"
        description={errorMessage}
        data-testid="tasks-detail-runs-error"
      />
    );
  }

  if (runs.length === 0) {
    return (
      <Empty
        icon={Inbox}
        title="Saved intent only — no runs yet"
        description="Publish, start, or approve this task to enqueue an executable run for the coordinator. Manual workers may also claim it."
        data-testid="tasks-detail-runs-empty"
      />
    );
  }

  return (
    <Section
      aria-label="Task runs"
      className="w-full gap-6 px-6 py-5"
      data-testid="tasks-detail-runs-panel"
    >
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead />
            <TableHead>Run</TableHead>
            <TableHead>Attempt</TableHead>
            <TableHead>Queued</TableHead>
            <TableHead>Ended</TableHead>
            <TableHead className="w-8" />
          </TableRow>
        </TableHeader>
        <TableBody>
          {runs.map(run => {
            const signal = taskStatusSignal(run.status);
            const channelLabel = runIsCoordinated(run) ? runCoordinationChannelLabel(run) : null;
            return (
              <TableRow data-testid={`tasks-detail-runs-item-${run.id}`} key={run.id}>
                <TableCell className="w-8 pl-4">
                  <Pill.Dot tone={signal.tone} pulse={signal.pulse} />
                </TableCell>
                <TableCell className="max-w-[360px]">
                  <div className="flex min-w-0 flex-col gap-1">
                    <div className="flex min-w-0 flex-wrap items-center gap-2">
                      <Pill mono>{run.id}</Pill>
                      <Pill tone={pillToneFromLegacyTone(taskRunStatusTone(run.status))}>
                        {run.status}
                      </Pill>
                      {channelLabel ? (
                        <Pill
                          data-testid={`tasks-detail-runs-channel-${run.id}`}
                          title="Coordination channel is bound to this run. Channel messages support coordination only — claim, heartbeat, and terminal status stay in the task service."
                          tone={pillToneFromLegacyTone("violet")}
                        >
                          <span className="inline-flex items-center gap-1">
                            <Radio className="size-3" aria-hidden="true" />
                            Channel: {channelLabel}
                          </span>
                        </Pill>
                      ) : null}
                      {run.session_id ? (
                        <span className="font-mono text-[11px] text-[color:var(--color-text-secondary)]">
                          session {run.session_id}
                        </span>
                      ) : null}
                    </div>
                    {run.error ? (
                      <p
                        className="text-[11px] text-[color:var(--color-danger)]"
                        data-testid={`tasks-detail-runs-error-${run.id}`}
                      >
                        {run.error}
                      </p>
                    ) : null}
                  </div>
                </TableCell>
                <TableCell className="font-mono text-[11px] text-[color:var(--color-text-secondary)]">
                  attempt {run.attempt}
                </TableCell>
                <TableCell className="font-mono text-[11px] text-[color:var(--color-text-tertiary)]">
                  {formatRelativeTime(run.queued_at)}
                </TableCell>
                <TableCell className="font-mono text-[11px] text-[color:var(--color-text-tertiary)]">
                  {run.ended_at ? formatRelativeTime(run.ended_at) : "—"}
                </TableCell>
                <TableCell className="w-8 pr-4">
                  <Link
                    aria-label={`Open run ${run.id}`}
                    className="inline-flex items-center gap-1 font-mono text-[10px] uppercase tracking-[0.14em] text-[color:var(--color-accent)] hover:underline"
                    data-testid={`tasks-detail-runs-link-${run.id}`}
                    params={{ id: taskId, runId: run.id }}
                    to="/tasks/$id/runs/$runId"
                  >
                    Open <ChevronRight className="size-3" />
                  </Link>
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </Section>
  );
}
