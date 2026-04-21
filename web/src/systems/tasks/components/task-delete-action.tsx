import { useCallback, useState, type ComponentProps } from "react";
import { Loader2, Trash2 } from "lucide-react";

import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@agh/ui";

type ButtonSize = ComponentProps<typeof Button>["size"];
type ButtonVariant = ComponentProps<typeof Button>["variant"];

export interface TaskDeleteActionProps {
  taskId: string;
  taskTitle: string;
  onDelete: (taskId: string) => void;
  isPending?: boolean;
  size?: ButtonSize;
  triggerLabel?: string;
  triggerVariant?: ButtonVariant;
  triggerTestId?: string;
  dialogTestId?: string;
  cancelTestId?: string;
  confirmTestId?: string;
}

export function TaskDeleteAction({
  taskId,
  taskTitle,
  onDelete,
  isPending = false,
  size = "sm",
  triggerLabel = "Delete",
  triggerVariant = "outline",
  triggerTestId = "task-delete-trigger",
  dialogTestId = "task-delete-dialog",
  cancelTestId = "task-delete-cancel",
  confirmTestId = "task-delete-confirm",
}: TaskDeleteActionProps) {
  const [open, setOpen] = useState(false);

  const handleConfirm = useCallback(() => {
    setOpen(false);
    onDelete(taskId);
  }, [onDelete, taskId]);

  return (
    <>
      <Button
        data-testid={triggerTestId}
        disabled={isPending}
        onClick={() => setOpen(true)}
        size={size}
        type="button"
        variant={triggerVariant}
      >
        {isPending ? (
          <Loader2 className="size-3.5 animate-spin" />
        ) : (
          <Trash2 className="size-3.5" />
        )}
        {triggerLabel}
      </Button>

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="max-w-md" data-testid={dialogTestId} showCloseButton={!isPending}>
          <DialogHeader>
            <DialogTitle>Delete task?</DialogTitle>
            <DialogDescription>
              This permanently removes <strong>{taskTitle}</strong> and its stored runs, events, and
              triage state. Delete is blocked while the task still has child tasks or active runs.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="gap-2">
            <Button
              data-testid={cancelTestId}
              disabled={isPending}
              onClick={() => setOpen(false)}
              type="button"
              variant="ghost"
            >
              Cancel
            </Button>
            <Button
              data-testid={confirmTestId}
              disabled={isPending}
              onClick={handleConfirm}
              type="button"
              variant="destructive"
            >
              {isPending ? (
                <>
                  <Loader2 className="size-3.5 animate-spin" />
                  Deleting
                </>
              ) : (
                <>
                  <Trash2 className="size-3.5" />
                  Delete task
                </>
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
