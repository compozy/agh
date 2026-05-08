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

import type { KnowledgeScope } from "../types";
import { knowledgeScopeLabel } from "../lib/knowledge-formatters";

interface KnowledgeDeleteDialogProps {
  open: boolean;
  onOpenChange: (next: boolean) => void;
  filename: string;
  scope: KnowledgeScope;
  isPending: boolean;
  error?: string | null;
  onConfirm: () => Promise<void>;
}

function KnowledgeDeleteDialog({
  open,
  onOpenChange,
  filename,
  scope,
  isPending,
  error,
  onConfirm,
}: KnowledgeDeleteDialogProps) {
  return (
    <Dialog onOpenChange={onOpenChange} open={open}>
      <DialogContent
        className="gap-0 p-0 sm:max-w-md"
        data-testid="knowledge-delete-dialog"
        showCloseButton={false}
      >
        <DialogHeader className="gap-2 border-b border-(--color-divider) px-5 py-4">
          <DialogTitle>Delete knowledge entry?</DialogTitle>
          <DialogDescription>
            This removes <span className="font-mono">{filename}</span> from the {scope} scope. The
            controller records the delete decision; the file is removed from{" "}
            {knowledgeScopeLabel(scope)} after the decision applies.
          </DialogDescription>
        </DialogHeader>
        {error ? (
          <div
            className="border-t border-(--color-divider) px-5 py-3 text-xs text-(--color-danger)"
            data-testid="knowledge-delete-dialog-error"
          >
            {error}
          </div>
        ) : null}
        <DialogFooter className="mx-0 mb-0 rounded-b-xl border-t border-(--color-divider) bg-transparent px-5 py-3">
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
