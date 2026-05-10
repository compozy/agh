import { Pencil } from "lucide-react";
import { useReducer } from "react";

import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Field,
  FieldContent,
  FieldGroup,
  FieldLabel,
  FieldSet,
  Input,
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

function stringReducer(_: string, next: string): string {
  return next;
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
  const [content, setContent] = useReducer(stringReducer, initialContent);
  const [description, setDescription] = useReducer(stringReducer, initialDescription ?? "");

  if (!open) {
    return null;
  }

  const handleOpenChange = (next: boolean) => {
    if (next) {
      setContent(initialContent);
      setDescription(initialDescription ?? "");
    }
    onOpenChange(next);
  };

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
    <Dialog onOpenChange={handleOpenChange} open={open}>
      <DialogContent
        className="sm:max-w-2xl"
        data-testid="knowledge-edit-dialog"
        showCloseButton={false}
        unframed
      >
        <DialogHeader variant="ruled">
          <DialogTitle>Edit knowledge entry</DialogTitle>
          <DialogDescription>
            Update <span className="font-mono">{filename}</span> in the {scope} scope. Edits go
            through the controller and produce a new decision.
          </DialogDescription>
        </DialogHeader>
        <FieldSet className="px-5 py-4">
          <FieldGroup>
            <Field>
              <FieldContent>
                <FieldLabel htmlFor="knowledge-edit-description">Description</FieldLabel>
              </FieldContent>
              <Input
                data-testid="knowledge-edit-description"
                id="knowledge-edit-description"
                onChange={event => setDescription(event.target.value)}
                placeholder="Optional summary"
                value={description}
              />
            </Field>
            <Field>
              <FieldContent>
                <FieldLabel htmlFor="knowledge-edit-content">Content</FieldLabel>
              </FieldContent>
              <Textarea
                className="h-60 font-mono text-xs"
                data-testid="knowledge-edit-content"
                id="knowledge-edit-content"
                onChange={event => setContent(event.target.value)}
                value={content}
              />
            </Field>
          </FieldGroup>
        </FieldSet>
        {error ? (
          <div
            className="border-t border-(--line) px-5 py-3 text-xs text-(--danger)"
            data-testid="knowledge-edit-dialog-error"
          >
            {error}
          </div>
        ) : null}
        <DialogFooter variant="ruled">
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
