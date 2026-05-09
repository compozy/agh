import { useCallback, useState, type ComponentProps } from "react";
import { Trash2 } from "lucide-react";

import { Button, ConfirmDialog, DialogTrigger, Spinner } from "@agh/ui";

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
    <ConfirmDialog
      cancelButtonProps={{ "data-testid": cancelTestId, disabled: isPending }}
      cancelLabel="Cancel"
      confirmButtonProps={{ "data-testid": confirmTestId }}
      confirmIcon={isPending ? Spinner : Trash2}
      confirmLabel={isPending ? "Deleting" : "Delete task"}
      contentProps={{ "data-testid": dialogTestId, showCloseButton: !isPending }}
      description={
        <>
          This permanently removes <strong>{taskTitle}</strong> and its stored runs, events, and
          triage state. Delete is blocked while the task still has child tasks or active runs.
        </>
      }
      isPending={isPending}
      onConfirm={handleConfirm}
      onOpenChange={setOpen}
      open={open}
      title="Delete task?"
      tone="danger"
    >
      <DialogTrigger
        render={
          <Button
            data-testid={triggerTestId}
            disabled={isPending}
            size={size}
            type="button"
            variant={triggerVariant}
          />
        }
      >
        {isPending ? <Spinner className="size-3.5" /> : <Trash2 className="size-3.5" />}
        {triggerLabel}
      </DialogTrigger>
    </ConfirmDialog>
  );
}
