import { Link } from "@tanstack/react-router";
import { AlertCircle, ChevronRight, Inbox, Radio } from "lucide-react";

import { BlockLoading, Empty, LinkedRecordTable, Pill } from "@agh/ui";

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
      <BlockLoading
        label="Loading task runs"
        size="md"
        surface="bare"
        data-testid="tasks-detail-runs-loading"
      />
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
        title="Saved intent only -- no runs yet"
        description="Publish, start, or approve this task to enqueue an executable run for the coordinator. Manual workers may also claim it."
        data-testid="tasks-detail-runs-empty"
      />
    );
  }

  return (
    <LinkedRecordTable
      aria-label="Task runs"
      className="w-full gap-6 px-6 py-5"
      columns={["Run", "Attempt", "Queued", "Ended"]}
      data-testid="tasks-detail-runs-panel"
    >
      <LinkedRecordTable.Body>
        {runs.map(run => {
          const signal = taskStatusSignal(run.status);
          const channelLabel = runIsCoordinated(run) ? runCoordinationChannelLabel(run) : null;
          return (
            <LinkedRecordTable.Row data-testid={`tasks-detail-runs-item-${run.id}`} key={run.id}>
              <LinkedRecordTable.Cell className="w-8 pl-4">
                <Pill.Dot tone={signal.tone} pulse={signal.pulse} />
              </LinkedRecordTable.Cell>
              <LinkedRecordTable.Cell className="max-w-[360px]">
                <LinkedRecordTable.Title>
                  <div className="flex min-w-0 flex-wrap items-center gap-2">
                    <Pill mono>{run.id}</Pill>
                    <Pill tone={taskRunStatusTone(run.status)}>{run.status}</Pill>
                    {channelLabel ? (
                      <Pill
                        data-testid={`tasks-detail-runs-channel-${run.id}`}
                        title="Coordination channel is bound to this run. Channel messages support coordination only -- claim, heartbeat, and terminal status stay in the task service."
                        tone="info"
                      >
                        <span className="inline-flex items-center gap-1">
                          <Radio className="size-3" aria-hidden="true" />
                          Channel: {channelLabel}
                        </span>
                      </Pill>
                    ) : null}
                    {run.session_id ? (
                      <span className="font-mono text-eyebrow text-muted">
                        session {run.session_id}
                      </span>
                    ) : null}
                  </div>
                  {run.error ? (
                    <p
                      className="text-eyebrow text-danger"
                      data-testid={`tasks-detail-runs-error-${run.id}`}
                    >
                      {run.error}
                    </p>
                  ) : null}
                </LinkedRecordTable.Title>
              </LinkedRecordTable.Cell>
              <LinkedRecordTable.Cell className="font-mono text-eyebrow text-muted">
                attempt {run.attempt}
              </LinkedRecordTable.Cell>
              <LinkedRecordTable.Cell className="font-mono text-eyebrow text-subtle">
                {formatRelativeTime(run.queued_at)}
              </LinkedRecordTable.Cell>
              <LinkedRecordTable.Cell className="font-mono text-eyebrow text-subtle">
                {run.ended_at ? formatRelativeTime(run.ended_at) : "--"}
              </LinkedRecordTable.Cell>
              <LinkedRecordTable.OpenCell>
                <Pill.Link
                  aria-label={`Open run ${run.id}`}
                  data-testid={`tasks-detail-runs-link-${run.id}`}
                  render={
                    <Link params={{ id: taskId, runId: run.id }} to="/tasks/$id/runs/$runId" />
                  }
                >
                  Open <ChevronRight className="size-3" />
                </Pill.Link>
              </LinkedRecordTable.OpenCell>
            </LinkedRecordTable.Row>
          );
        })}
      </LinkedRecordTable.Body>
    </LinkedRecordTable>
  );
}
