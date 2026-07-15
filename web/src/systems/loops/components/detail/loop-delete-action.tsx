import { Trash2 } from "lucide-react";

import { Button, ConfirmDialog, DialogTrigger } from "@agh/ui";

interface LoopDeleteActionProps {
  loopName: string;
  isPending: boolean;
  error: string | null;
  onConfirm: () => Promise<void>;
  onReset: () => void;
  defaultOpen?: boolean;
}

/** Workspace-shadow deletion with an exact-name confirmation boundary. */
export function LoopDeleteAction({
  loopName,
  isPending,
  error,
  onConfirm,
  onReset,
  defaultOpen = false,
}: LoopDeleteActionProps) {
  return (
    <ConfirmDialog
      defaultOpen={defaultOpen}
      title={`Delete ${loopName}?`}
      description="Delete this workspace-owned definition. If a bundled Loop shares the name, it becomes visible again."
      confirmLabel="Delete loop"
      cancelLabel="Cancel"
      tone="danger"
      confirmTyping={loopName}
      confirmIcon={Trash2}
      isPending={isPending}
      error={error}
      onConfirm={onConfirm}
      onOpenChange={() => onReset()}
      cancelButtonProps={{ disabled: isPending }}
      contentProps={{ "data-testid": "loop-delete-dialog" }}
    >
      <DialogTrigger
        render={
          <Button type="button" variant="destructive" size="sm" data-testid="loop-delete-action">
            <Trash2 aria-hidden="true" className="size-3.5" />
            Delete loop
          </Button>
        }
      />
    </ConfirmDialog>
  );
}
