import { Activity, AlertCircle } from "lucide-react";

import { Pill, PillDot, Section, type PillTone } from "@agh/ui";

import type { StreamConnectionState } from "@/hooks/routes/use-task-detail-orchestration-tab";

export interface TasksStreamResumeCardProps {
  latestEventSeq: number | null;
  hasLatestEventSeq: boolean;
  streamSeedSequence: number;
  streamState: StreamConnectionState;
  streamErrorMessage: string | null;
}

const STREAM_TONE: Record<StreamConnectionState, PillTone> = {
  idle: "neutral",
  connected: "success",
  error: "danger",
  disabled: "neutral",
};

const STREAM_LABEL: Record<StreamConnectionState, string> = {
  idle: "awaiting first frame",
  connected: "connected",
  error: "disconnected",
  disabled: "disabled",
};

export function TasksStreamResumeCard({
  latestEventSeq,
  hasLatestEventSeq,
  streamSeedSequence,
  streamState,
  streamErrorMessage,
}: TasksStreamResumeCardProps) {
  const seqLabel = hasLatestEventSeq && latestEventSeq !== null ? String(latestEventSeq) : "—";
  const seedLabel = String(streamSeedSequence);
  const tone = STREAM_TONE[streamState];
  const label = STREAM_LABEL[streamState];

  return (
    <Section
      aria-label="Stream resume"
      className="w-full gap-4"
      data-testid="tasks-stream-resume-card"
      label="Stream resume"
    >
      <p
        className="text-[12px] text-[color:var(--color-text-tertiary)]"
        data-testid="tasks-stream-resume-disclaimer"
      >
        The web client seeds task SSE through Last-Event-ID derived from the task's latest_event_seq
        projection. Reconnects resume from the seeded sequence; named SSE frames invalidate read
        queries without inferring authority.
      </p>
      <div
        className="grid gap-3 rounded-xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface-elevated)] px-4 py-3 md:grid-cols-3"
        data-testid="tasks-stream-resume-summary"
      >
        <div className="flex flex-col gap-1">
          <span className="font-mono text-[10px] uppercase tracking-[0.14em] text-[color:var(--color-text-tertiary)]">
            Latest event seq
          </span>
          <span
            className="font-mono text-[15px] text-[color:var(--color-text-primary)]"
            data-testid="tasks-stream-resume-latest"
          >
            {seqLabel}
          </span>
        </div>
        <div className="flex flex-col gap-1">
          <span className="font-mono text-[10px] uppercase tracking-[0.14em] text-[color:var(--color-text-tertiary)]">
            SSE resume seed
          </span>
          <span
            className="font-mono text-[15px] text-[color:var(--color-text-primary)]"
            data-testid="tasks-stream-resume-seed"
          >
            {seedLabel}
          </span>
        </div>
        <div className="flex flex-col gap-1">
          <span className="font-mono text-[10px] uppercase tracking-[0.14em] text-[color:var(--color-text-tertiary)]">
            Connection
          </span>
          <div className="inline-flex items-center gap-2" data-testid="tasks-stream-resume-status">
            <PillDot tone={tone} pulse={streamState === "connected"} />
            <Pill tone={tone}>{label}</Pill>
          </div>
        </div>
      </div>
      {streamState === "error" && streamErrorMessage ? (
        <p
          className="inline-flex items-center gap-2 text-[12px] text-[color:var(--color-danger)]"
          data-testid="tasks-stream-resume-error"
        >
          <AlertCircle className="size-3.5" />
          {streamErrorMessage}
        </p>
      ) : null}
      {streamState === "disabled" ? (
        <p
          className="inline-flex items-center gap-2 text-[12px] text-[color:var(--color-text-tertiary)]"
          data-testid="tasks-stream-resume-disabled"
        >
          <Activity className="size-3.5" />
          Stream disabled. Open the orchestration tab on a real task to subscribe.
        </p>
      ) : null}
    </Section>
  );
}
