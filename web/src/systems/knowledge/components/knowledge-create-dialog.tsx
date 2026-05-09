import { Plus } from "lucide-react";
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
  NativeSelect,
  NativeSelectOption,
  Textarea,
} from "@agh/ui";

import type { MemoryType } from "@/systems/knowledge/types";

interface KnowledgeCreateInput {
  type: MemoryType;
  name: string;
  description?: string;
  content: string;
}

interface KnowledgeCreateDialogProps {
  open: boolean;
  onOpenChange: (next: boolean) => void;
  scope: string;
  defaultType: MemoryType;
  isPending: boolean;
  error?: string | null;
  onConfirm: (input: KnowledgeCreateInput) => Promise<void>;
}

function KnowledgeCreateDialog({
  open,
  onOpenChange,
  scope,
  defaultType,
  isPending,
  error,
  onConfirm,
}: KnowledgeCreateDialogProps) {
  const [type, setType] = useState<MemoryType>(defaultType);
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [content, setContent] = useState("");

  useEffect(() => {
    if (open) {
      setType(defaultType);
      setName("");
      setDescription("");
      setContent("");
    }
  }, [defaultType, open]);

  const handleSubmit = async () => {
    const trimmedDescription = description.trim();
    try {
      await onConfirm({
        type,
        name: name.trim(),
        description: trimmedDescription === "" ? undefined : trimmedDescription,
        content,
      });
    } catch {
      // Error state is surfaced through `error` and the dialog stays open.
    }
  };

  const submitDisabled = isPending || name.trim().length === 0 || content.trim().length === 0;

  return (
    <Dialog onOpenChange={onOpenChange} open={open}>
      <DialogContent
        className="gap-0 p-0 sm:max-w-2xl"
        data-testid="knowledge-create-dialog"
        showCloseButton={false}
      >
        <DialogHeader className="gap-2 border-b border-[color:var(--color-divider)] px-5 py-4">
          <DialogTitle>Create knowledge entry</DialogTitle>
          <DialogDescription>
            Add knowledge in the {scope} scope through the controller. The entry is recorded as a
            decision and becomes available to matching future recall.
          </DialogDescription>
        </DialogHeader>
        <div className="flex flex-col gap-4 px-5 py-4">
          <div className="grid gap-4 sm:grid-cols-[10rem_1fr]">
            <div className="flex flex-col gap-1.5">
              <Label
                className="font-mono text-[10px] uppercase tracking-[0.08em] text-[color:var(--color-text-label)]"
                htmlFor="knowledge-create-type"
              >
                Type
              </Label>
              <NativeSelect
                data-testid="knowledge-create-type"
                id="knowledge-create-type"
                onChange={event => setType(event.target.value as MemoryType)}
                value={type}
              >
                <NativeSelectOption value="user">user</NativeSelectOption>
                <NativeSelectOption value="feedback">feedback</NativeSelectOption>
                <NativeSelectOption value="project">project</NativeSelectOption>
                <NativeSelectOption value="reference">reference</NativeSelectOption>
              </NativeSelect>
            </div>
            <div className="flex flex-col gap-1.5">
              <Label
                className="font-mono text-[10px] uppercase tracking-[0.08em] text-[color:var(--color-text-label)]"
                htmlFor="knowledge-create-name"
              >
                Name
              </Label>
              <Input
                data-testid="knowledge-create-name"
                id="knowledge-create-name"
                onChange={event => setName(event.target.value)}
                placeholder="Canonical knowledge name"
                value={name}
              />
            </div>
          </div>
          <div className="flex flex-col gap-1.5">
            <Label
              className="font-mono text-[10px] uppercase tracking-[0.08em] text-[color:var(--color-text-label)]"
              htmlFor="knowledge-create-description"
            >
              Description
            </Label>
            <Input
              data-testid="knowledge-create-description"
              id="knowledge-create-description"
              onChange={event => setDescription(event.target.value)}
              placeholder="Optional summary"
              value={description}
            />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label
              className="font-mono text-[10px] uppercase tracking-[0.08em] text-[color:var(--color-text-label)]"
              htmlFor="knowledge-create-content"
            >
              Content
            </Label>
            <Textarea
              className="h-60 font-mono text-[12px]"
              data-testid="knowledge-create-content"
              id="knowledge-create-content"
              onChange={event => setContent(event.target.value)}
              value={content}
            />
          </div>
        </div>
        {error ? (
          <div
            className="border-t border-[color:var(--color-divider)] px-5 py-3 text-xs text-[color:var(--color-danger)]"
            data-testid="knowledge-create-dialog-error"
          >
            {error}
          </div>
        ) : null}
        <DialogFooter className="mx-0 mb-0 rounded-b-xl border-t border-[color:var(--color-divider)] bg-transparent px-5 py-3">
          <Button
            data-testid="cancel-create-memory-btn"
            onClick={() => onOpenChange(false)}
            size="sm"
            type="button"
            variant="ghost"
          >
            Cancel
          </Button>
          <Button
            data-testid="confirm-create-memory-btn"
            disabled={submitDisabled}
            onClick={handleSubmit}
            size="sm"
            type="button"
          >
            <Plus className="size-3.5" />
            Create
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export { KnowledgeCreateDialog };
export type { KnowledgeCreateDialogProps, KnowledgeCreateInput };
