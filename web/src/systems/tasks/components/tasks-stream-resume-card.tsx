import { Activity, AlertCircle } from "lucide-react";

import { Alert, AlertDescription, Metric, Pill, PillDot, Section, type PillTone } from "@agh/ui";

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
      bodyClassName="gap-4"
      className="w-full gap-4"
      data-testid="tasks-stream-resume-card"
      icon={Activity}
      label="Stream resume"
    >
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
        <Alert variant="danger" data-testid="tasks-stream-resume-error">
          <AlertCircle />
          <AlertDescription>{streamErrorMessage}</AlertDescription>
        </Alert>
      ) : null}
      {streamState === "disabled" ? (
        <Alert variant="neutral" role="status" data-testid="tasks-stream-resume-disabled">
          <Activity />
          <AlertDescription>
            Stream disabled. Open the orchestration tab on a real task to subscribe.
          </AlertDescription>
        </Alert>
      ) : null}
    </Section>
  );
}
