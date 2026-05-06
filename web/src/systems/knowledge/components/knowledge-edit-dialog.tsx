import { Pencil } from "lucide-react";
import { useEffect, useState } from "react";

import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Input,
  Label,
  Textarea,
} from "@agh/ui";

interface KnowledgeEditDialogProps {
  open: boolean;
  onOpenChange: (next: boolean) => void;
  filename: string;
  scope: string;
  initialContent: string;
  initialDescription?: string;
  isPending: boolean;
  error?: string | null;
  onConfirm: (input: { content: string; description?: string }) => Promise<void>;
}

function KnowledgeEditDialog({
  open,
  onOpenChange,
  filename,
  scope,
  initialContent,
  initialDescription,
  isPending,
  error,
  onConfirm,
}: KnowledgeEditDialogProps) {
  const [content, setContent] = useState(initialContent);
  const [description, setDescription] = useState(initialDescription ?? "");

  useEffect(() => {
    if (open) {
      setContent(initialContent);
      setDescription(initialDescription ?? "");
    }
  }, [open, initialContent, initialDescription]);

  const handleSubmit = async () => {
    const trimmedDescription = description.trim();
    await onConfirm({
      content,
      description: trimmedDescription === "" ? undefined : trimmedDescription,
    });
  };

  const submitDisabled =
    isPending || content.trim().length === 0 || content === (initialContent ?? "");

  return (
    <Dialog onOpenChange={onOpenChange} open={open}>
      <DialogContent
        className="gap-0 p-0 sm:max-w-2xl"
        data-testid="knowledge-edit-dialog"
        showCloseButton={false}
      >
        <DialogHeader className="gap-2 border-b border-[color:var(--color-divider)] px-5 py-4">
          <DialogTitle>Edit knowledge entry</DialogTitle>
          <DialogDescription>
            Update <span className="font-mono">{filename}</span> in the {scope} scope. Edits go
            through the controller and produce a new decision.
          </DialogDescription>
        </DialogHeader>
        <div className="flex flex-col gap-4 px-5 py-4">
          <div className="flex flex-col gap-1.5">
            <Label
              className="font-mono text-[10px] uppercase tracking-[0.08em] text-[color:var(--color-text-label)]"
              htmlFor="knowledge-edit-description"
            >
              Description
            </Label>
            <Input
              data-testid="knowledge-edit-description"
              id="knowledge-edit-description"
              onChange={event => setDescription(event.target.value)}
              placeholder="Optional summary"
              value={description}
            />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label
              className="font-mono text-[10px] uppercase tracking-[0.08em] text-[color:var(--color-text-label)]"
              htmlFor="knowledge-edit-content"
            >
              Content
            </Label>
            <Textarea
              className="h-60 font-mono text-[12px]"
              data-testid="knowledge-edit-content"
              id="knowledge-edit-content"
              onChange={event => setContent(event.target.value)}
              value={content}
            />
          </div>
        </div>
        {error ? (
          <div
            className="border-t border-[color:var(--color-divider)] px-5 py-3 text-xs text-[color:var(--color-danger)]"
            data-testid="knowledge-edit-dialog-error"
          >
            {error}
          </div>
        ) : null}
        <DialogFooter className="border-t border-[color:var(--color-divider)] bg-transparent px-5 py-3">
          <Button
            data-testid="cancel-edit-memory-btn"
            onClick={() => onOpenChange(false)}
            size="sm"
            type="button"
            variant="ghost"
          >
            Cancel
          </Button>
          <Button
            data-testid="confirm-edit-memory-btn"
            disabled={submitDisabled}
            onClick={handleSubmit}
            size="sm"
            type="button"
          >
            <Pencil className="size-3.5" />
            Save edit
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export { KnowledgeEditDialog };
export type { KnowledgeEditDialogProps };
