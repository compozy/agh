import { useCallback, useState } from "react";
import { AlertCircle, PauseCircle, PlayCircle, RotateCw } from "lucide-react";

import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Empty,
  Eyebrow,
  MonoId,
  Pill,
  Textarea,
  Time,
} from "@agh/ui";

import type { SchedulerBacklog, SchedulerStatus } from "../types";

export interface SchedulerControlsPanelProps {
  status: SchedulerStatus | null;
  backlog: SchedulerBacklog | null;
  errorMessage?: string | null;
  backlogErrorMessage?: string | null;
  isLoading?: boolean;
  isBacklogLoading?: boolean;
  pending?: {
    pause?: boolean;
    resume?: boolean;
    drain?: boolean;
  };
  onPause?: (reason: string) => void | Promise<void>;
  onResume?: () => void | Promise<void>;
  onDrain?: () => void | Promise<void>;
}

const BACKLOG_PREVIEW_LIMIT = 5;

export function SchedulerControlsPanel({
  status,
  backlog,
  errorMessage = null,
  backlogErrorMessage = null,
  isLoading = false,
  isBacklogLoading = false,
  pending,
  onPause,
  onResume,
  onDrain,
}: SchedulerControlsPanelProps) {
  const [pauseOpen, setPauseOpen] = useState(false);
  const [pauseReason, setPauseReason] = useState("");
  const [pauseError, setPauseError] = useState<string | null>(null);
  const isPausePending = pending?.pause ?? false;
  const isResumePending = pending?.resume ?? false;
  const isDrainPending = pending?.drain ?? false;
  const isActionPending = isPausePending || isResumePending || isDrainPending;
  const rows = backlog?.runs?.slice(0, BACKLOG_PREVIEW_LIMIT) ?? [];

  const handlePauseOpenChange = useCallback((next: boolean) => {
    setPauseOpen(next);
    if (!next) {
      setPauseError(null);
    }
  }, []);

  const handlePauseConfirm = useCallback(async () => {
    const reason = pauseReason.trim();
    if (!reason) {
      setPauseError("Provide a pause reason.");
      return;
    }
    if (!onPause) {
      return;
    }
    try {
      await onPause(reason);
      setPauseReason("");
      setPauseError(null);
      setPauseOpen(false);
    } catch (error) {
      setPauseError(error instanceof Error ? error.message : "Failed to pause scheduler.");
    }
  }, [onPause, pauseReason]);

  if (errorMessage && !status) {
    return (
      <section
        className="border-b border-line-soft bg-canvas-soft px-5 py-4"
        data-testid="scheduler-controls-panel-error"
      >
        <Empty
          data-testid="scheduler-controls-panel-empty-error"
          description={errorMessage}
          fill={false}
          icon={AlertCircle}
          title="Unable to load scheduler"
        />
      </section>
    );
  }

  return (
    <section
      className="border-b border-line-soft bg-canvas-soft px-5 py-4"
      data-testid="scheduler-controls-panel"
    >
      <div className="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
        <div className="min-w-0">
          <Eyebrow className="text-muted">Scheduler</Eyebrow>
          <div className="mt-1 flex min-w-0 flex-wrap items-center gap-2">
            <h2 className="text-item-title font-medium text-fg-strong">Dispatch controls</h2>
            <Pill
              data-testid="scheduler-controls-state"
              tone={status?.paused ? "warning" : "success"}
            >
              {status?.paused ? "Paused" : "Running"}
            </Pill>
            {isLoading ? (
              <Pill data-testid="scheduler-controls-loading" tone="neutral">
                Loading
              </Pill>
            ) : null}
          </div>
          <div
            className="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1 text-form-label text-muted"
            data-testid="scheduler-controls-meta"
          >
            <span>{status?.active_claim_count ?? 0} active claims</span>
            <span aria-hidden="true" className="text-faint">
              ·
            </span>
            <span>{status?.queued_run_count ?? 0} queued runs</span>
            <span aria-hidden="true" className="text-faint">
              ·
            </span>
            <span>{status?.paused_task_count ?? 0} paused tasks</span>
            {status?.paused_at ? (
              <>
                <span aria-hidden="true" className="text-faint">
                  ·
                </span>
                <span>
                  Paused <Time iso={status.paused_at} mode="relative" />
                </span>
              </>
            ) : null}
          </div>
          {status?.paused_reason ? (
            <p
              className="mt-2 max-w-3xl text-form-label text-muted"
              data-testid="scheduler-controls-reason"
            >
              {status.paused_reason}
            </p>
          ) : null}
        </div>

        <div className="flex shrink-0 flex-wrap items-center gap-2">
          {status?.paused ? (
            <Button
              data-testid="scheduler-controls-resume"
              disabled={isActionPending || !onResume}
              onClick={() => void onResume?.()}
              size="sm"
              type="button"
              variant="neutral"
            >
              <PlayCircle className="size-3" aria-hidden="true" />
              Resume
            </Button>
          ) : (
            <Button
              data-testid="scheduler-controls-pause"
              disabled={isActionPending || !onPause}
              onClick={() => setPauseOpen(true)}
              size="sm"
              type="button"
              variant="neutral"
            >
              <PauseCircle className="size-3" aria-hidden="true" />
              Pause
            </Button>
          )}
          <Button
            data-testid="scheduler-controls-drain"
            disabled={isActionPending || !onDrain}
            onClick={() => void onDrain?.()}
            size="sm"
            title="Pause dispatch and wait for active claims to finish."
            type="button"
          >
            <RotateCw className="size-3" aria-hidden="true" />
            Drain
          </Button>
        </div>
      </div>

      <div className="mt-4 border-t border-line-soft pt-3" data-testid="scheduler-backlog-panel">
        <div className="flex items-center justify-between gap-3">
          <Eyebrow className="text-muted">Backlog</Eyebrow>
          <Pill data-testid="scheduler-backlog-total" tone={backlog?.total ? "warning" : "neutral"}>
            {backlog?.total ?? 0}
          </Pill>
        </div>
        {backlogErrorMessage ? (
          <p className="mt-2 text-form-label text-danger" data-testid="scheduler-backlog-error">
            {backlogErrorMessage}
          </p>
        ) : isBacklogLoading && rows.length === 0 ? (
          <p className="mt-2 text-form-label text-muted" data-testid="scheduler-backlog-loading">
            Loading backlog
          </p>
        ) : rows.length === 0 ? (
          <p className="mt-2 text-form-label text-muted" data-testid="scheduler-backlog-empty">
            No queued runs.
          </p>
        ) : (
          <div className="mt-2 divide-y divide-line-soft" data-testid="scheduler-backlog-rows">
            {rows.map(item => (
              <div
                className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3 py-2 text-form-label"
                data-testid={`scheduler-backlog-row-${item.run.id}`}
                key={item.run.id}
              >
                <div className="min-w-0">
                  <div className="flex min-w-0 items-center gap-2">
                    <MonoId value={item.task.identifier ?? item.task.id} />
                    <span className="truncate text-fg">{item.task.title}</span>
                  </div>
                  <div className="mt-1 flex flex-wrap items-center gap-2 text-muted">
                    <span>Run {item.run.id}</span>
                    <span aria-hidden="true" className="text-faint">
                      ·
                    </span>
                    <span>{item.run.status}</span>
                  </div>
                </div>
                {item.task.effective_paused ? (
                  <Pill tone="warning">Paused</Pill>
                ) : (
                  <Pill tone="neutral">Queued</Pill>
                )}
              </div>
            ))}
          </div>
        )}
      </div>

      <Dialog open={pauseOpen} onOpenChange={handlePauseOpenChange}>
        <DialogContent
          data-testid="scheduler-controls-pause-dialog"
          showCloseButton={!isPausePending}
          className="max-w-md"
        >
          <DialogHeader>
            <DialogTitle>Pause scheduler?</DialogTitle>
            <DialogDescription>New claims stop; active runs continue.</DialogDescription>
          </DialogHeader>
          <div className="flex flex-col gap-2">
            <label className="eyebrow text-muted" htmlFor="scheduler-controls-pause-reason">
              Reason
            </label>
            <Textarea
              aria-invalid={Boolean(pauseError)}
              data-testid="scheduler-controls-pause-reason"
              disabled={isPausePending}
              id="scheduler-controls-pause-reason"
              onChange={event => {
                setPauseReason(event.target.value);
                setPauseError(null);
              }}
              rows={3}
              value={pauseReason}
            />
            {pauseError ? (
              <p
                className="text-form-hint text-danger"
                data-testid="scheduler-controls-pause-error"
              >
                {pauseError}
              </p>
            ) : null}
          </div>
          <DialogFooter className="gap-2">
            <Button
              disabled={isPausePending}
              onClick={() => handlePauseOpenChange(false)}
              size="sm"
              type="button"
              variant="neutral"
            >
              Cancel
            </Button>
            <Button
              data-testid="scheduler-controls-pause-confirm"
              disabled={isPausePending}
              onClick={() => void handlePauseConfirm()}
              size="sm"
              type="button"
            >
              Pause
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </section>
  );
}
