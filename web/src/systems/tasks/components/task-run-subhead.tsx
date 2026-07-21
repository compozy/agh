import { Time } from "@agh/ui";

import { useLiveElapsed } from "../hooks/use-live-elapsed";
import { computeElapsed } from "../lib/task-formatters";
import type { TaskRunDetailView } from "../types";

function MetaDot() {
  return (
    <span aria-hidden="true" className="text-faint">
      ·
    </span>
  );
}

/** Demoted run meta line: mono ids, claimant, freshness, live-ticking elapsed. */
export function TaskRunSubhead({ run }: { run: TaskRunDetailView }) {
  const record = run.run;
  const isRunning = record.status === "running" || record.status === "starting";
  const liveElapsed = useLiveElapsed(record.started_at ?? undefined, isRunning);
  const duration = isRunning ? liveElapsed : computeElapsed(record);
  const sessionId = record.session_id ?? run.session?.session_id ?? null;

  return (
    <div
      className="mb-5 flex min-w-0 flex-wrap items-center gap-2 border-b border-line pb-3.5 text-form-label text-subtle"
      data-testid="tasks-run-subhead"
    >
      <span className="font-mono text-eyebrow tabular-nums">
        {record.id}
        {sessionId ? ` · ${sessionId}` : null}
      </span>
      {record.claimed_by?.ref ? (
        <>
          <MetaDot />
          <span>
            Claimed by <span className="font-medium text-muted">{record.claimed_by.ref}</span>
          </span>
        </>
      ) : null}
      <MetaDot />
      {record.ended_at ? (
        <span className="inline-flex items-center gap-1">
          Ended <Time iso={record.ended_at} mode="relative" />
        </span>
      ) : record.started_at ? (
        <span className="inline-flex items-center gap-1">
          Started <Time iso={record.started_at} mode="relative" />
        </span>
      ) : (
        <span className="inline-flex items-center gap-1">
          Queued <Time iso={record.queued_at} mode="relative" />
        </span>
      )}
      {duration ? (
        <>
          <MetaDot />
          <span className="font-mono text-eyebrow tabular-nums">{duration}</span>
        </>
      ) : null}
    </div>
  );
}
