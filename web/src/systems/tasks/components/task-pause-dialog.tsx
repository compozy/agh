import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Textarea,
} from "@agh/ui";

import type { useTaskPauseDialog } from "../hooks/use-task-pause-dialog";

export interface TaskPauseDialogProps {
  dialog: ReturnType<typeof useTaskPauseDialog>;
  isPending?: boolean;
}

/** Pause confirmation with a required reason (recorded on the task record). */
export function TaskPauseDialog({ dialog, isPending = false }: TaskPauseDialogProps) {
  return (
    <Dialog onOpenChange={dialog.onOpenChange} open={dialog.isOpen}>
      <DialogContent
        className="max-w-md"
        data-testid="tasks-detail-pause-dialog"
        showCloseButton={!isPending}
      >
        <DialogHeader>
          <DialogTitle>Pause task?</DialogTitle>
          <DialogDescription>
            New scheduler claims stop for this task; active runs continue.
          </DialogDescription>
        </DialogHeader>
        <div className="flex flex-col gap-2">
          <label className="eyebrow text-muted" htmlFor="tasks-detail-pause-reason">
            Reason
          </label>
          <Textarea
            aria-invalid={Boolean(dialog.error)}
            data-testid="tasks-detail-pause-reason"
            disabled={isPending}
            id="tasks-detail-pause-reason"
            onChange={dialog.onReasonChange}
            rows={3}
            value={dialog.reason}
          />
          {dialog.error ? (
            <p className="text-form-hint text-danger" data-testid="tasks-detail-pause-error">
              {dialog.error}
            </p>
          ) : null}
        </div>
        <DialogFooter className="gap-2">
          <Button
            disabled={isPending}
            onClick={dialog.close}
            size="sm"
            type="button"
            variant="neutral"
          >
            Cancel
          </Button>
          <Button
            data-testid="tasks-detail-pause-confirm"
            disabled={isPending}
            onClick={() => void dialog.confirm()}
            size="sm"
            type="button"
          >
            Pause task
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
