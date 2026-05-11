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
      icon={Activity}
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
          value={<span className="font-mono tabular-nums">{seqLabel}</span>}
        />
        <Metric
          data-testid="tasks-stream-resume-seed"
          label="SSE resume seed"
          value={<span className="font-mono tabular-nums">{seedLabel}</span>}
        />
        <Metric
          data-testid="tasks-stream-resume-status"
          label="Connection"
          value={
            <Pill tone={tone} pulse={streamState === "connected"}>
              <PillDot />
              {label}
            </Pill>
          }
        />
      </div>
      {streamState === "error" && streamErrorMessage ? (
        <div
          className="flex items-start gap-2 rounded bg-danger-tint px-3 py-2 text-[12px] leading-relaxed text-danger"
          data-testid="tasks-stream-resume-error"
        >
          <AlertCircle className="mt-0.5 size-3 shrink-0" />
          <span>{streamErrorMessage}</span>
        </div>
      ) : null}
      {streamState === "disabled" ? (
        <div
          className="flex items-start gap-2 text-[12px] leading-relaxed text-faint"
          data-testid="tasks-stream-resume-disabled"
        >
          <Activity className="mt-0.5 size-3 shrink-0" />
          <span>Stream disabled. Open the orchestration tab on a real task to subscribe.</span>
        </div>
      ) : null}
    </Section>
  );
}
