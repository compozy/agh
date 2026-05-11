import { Activity, AlertCircle } from "lucide-react";

import { Metric, Pill, PillDot, Section, type PillTone } from "@agh/ui";

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
  const seqLabel = hasLatestEventSeq && latestEventSeq !== null ? String(latestEventSeq) : "--";
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
      <p className="text-xs text-subtle" data-testid="tasks-stream-resume-disclaimer">
        The web client seeds task SSE through Last-Event-ID derived from the task's latest_event_seq
        projection. Reconnects resume from the seeded sequence; named SSE frames invalidate read
        queries without inferring authority.
      </p>
      <div className="grid gap-3 md:grid-cols-3" data-testid="tasks-stream-resume-summary">
        <Metric
          data-testid="tasks-stream-resume-latest"
          label="Latest event seq"
          value={seqLabel}
        />
        <Metric data-testid="tasks-stream-resume-seed" label="SSE resume seed" value={seedLabel} />
        <Metric
          data-testid="tasks-stream-resume-status"
          label="Connection"
          value={
            <span className="inline-flex items-center gap-2">
              <PillDot tone={tone} pulse={streamState === "connected"} />
              <Pill tone={tone}>{label}</Pill>
            </span>
          }
        />
      </div>
      {streamState === "error" && streamErrorMessage ? (
        <p
          className="inline-flex items-center gap-2 text-xs text-danger"
          data-testid="tasks-stream-resume-error"
        >
          <AlertCircle className="size-3.5" />
          {streamErrorMessage}
        </p>
      ) : null}
      {streamState === "disabled" ? (
        <p
          className="inline-flex items-center gap-2 text-xs text-subtle"
          data-testid="tasks-stream-resume-disabled"
        >
          <Activity className="size-3.5" />
          Stream disabled. Open the orchestration tab on a real task to subscribe.
        </p>
      ) : null}
    </Section>
  );
}
