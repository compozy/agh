import { Trash2 } from "lucide-react";

import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@agh/ui";

interface KnowledgeDeleteDialogProps {
  open: boolean;
  onOpenChange: (next: boolean) => void;
  filename: string;
  scope: string;
  isPending: boolean;
  onConfirm: () => void;
}

function KnowledgeDeleteDialog({
  open,
  onOpenChange,
  filename,
  scope,
  isPending,
  onConfirm,
}: KnowledgeDeleteDialogProps) {
  return (
    <Dialog onOpenChange={onOpenChange} open={open}>
      <DialogContent
        className="gap-0 p-0 sm:max-w-md"
        data-testid="knowledge-delete-dialog"
        showCloseButton={false}
      >
        <DialogHeader className="gap-2 border-b border-[color:var(--color-divider)] px-5 py-4">
          <DialogTitle>Delete knowledge entry?</DialogTitle>
          <DialogDescription>
            This removes <span className="font-mono">{filename}</span> from the {scope} scope. This
            action cannot be undone.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter className="border-t border-[color:var(--color-divider)] bg-transparent px-5 py-3">
          <Button
            data-testid="cancel-delete-memory-btn"
            onClick={() => onOpenChange(false)}
            size="sm"
            type="button"
            variant="ghost"
          >
            Cancel
          </Button>
          <Button
            data-testid="confirm-delete-memory-btn"
            disabled={isPending}
            onClick={onConfirm}
            size="sm"
            type="button"
            variant="destructive"
          >
            <Trash2 className="size-3.5" />
            Delete
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export { KnowledgeDeleteDialog };
export type { KnowledgeDeleteDialogProps };
