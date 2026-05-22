import { BookOpen, ClipboardList, MessageSquare, Plus, Tag } from "lucide-react";
import { useState } from "react";

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
  RadioCard,
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

interface TypeOption {
  value: MemoryType;
  title: string;
  description: string;
  icon: typeof BookOpen;
}

const TYPE_OPTIONS: ReadonlyArray<TypeOption> = [
  {
    value: "user",
    title: "User",
    description: "Operator preferences and identity guidance.",
    icon: ClipboardList,
  },
  {
    value: "feedback",
    title: "Feedback",
    description: "Coaching notes captured from prior runs.",
    icon: MessageSquare,
  },
  {
    value: "project",
    title: "Project",
    description: "Long-lived decisions with rationale.",
    icon: BookOpen,
  },
  {
    value: "reference",
    title: "Reference",
    description: "Pointer to docs, code, or external systems.",
    icon: Tag,
  },
];

function KnowledgeCreateDialog({
  open,
  onOpenChange,
  scope,
  defaultType,
  isPending,
  error,
  onConfirm,
}: KnowledgeCreateDialogProps) {
  const [selectedType, setSelectedType] = useState<MemoryType | null>(null);
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [content, setContent] = useState("");
  const type = selectedType ?? defaultType;

  const resetDraft = () => {
    setSelectedType(null);
    setName("");
    setDescription("");
    setContent("");
  };

  const updateDialogOpen = (next: boolean) => {
    if (next) {
      resetDraft();
    }
    onOpenChange(next);
  };

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
    <Dialog onOpenChange={updateDialogOpen} open={open}>
      <DialogContent
        className="grid-rows-[auto_minmax(0,1fr)_auto_auto] max-h-[min(var(--height-modal-tall),calc(100vh-2rem))] gap-0 overflow-hidden p-0 sm:max-w-2xl"
        data-testid="knowledge-create-dialog"
        showCloseButton={false}
      >
        <DialogHeader className="gap-2 border-b border-line px-5 py-4" variant="ruled">
          <DialogTitle>Create knowledge entry</DialogTitle>
          <DialogDescription>
            Add knowledge in the {scope} scope through the controller. The entry is recorded as a
            decision and becomes available to matching future recall.
          </DialogDescription>
        </DialogHeader>
        <div className="min-h-0 overflow-y-auto px-5 py-4">
          <div className="flex flex-col gap-4">
            <div className="flex flex-col gap-2">
              <Label className="eyebrow text-muted" htmlFor="knowledge-create-name">
                Type
              </Label>
              <div
                aria-label="Knowledge type"
                className="grid grid-cols-1 gap-2 sm:grid-cols-2"
                data-testid="knowledge-create-type-grid"
                role="radiogroup"
              >
                {TYPE_OPTIONS.map(option => (
                  <RadioCard
                    data-testid={`knowledge-create-type-${option.value}`}
                    description={option.description}
                    icon={option.icon}
                    key={option.value}
                    onSelect={() => setSelectedType(option.value)}
                    selected={type === option.value}
                    title={option.title}
                  />
                ))}
              </div>
            </div>
            <div className="flex flex-col gap-1.5">
              <Label className="eyebrow text-muted" htmlFor="knowledge-create-name">
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
            <div className="flex flex-col gap-1.5">
              <Label className="eyebrow text-muted" htmlFor="knowledge-create-description">
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
              <Label className="eyebrow text-muted" htmlFor="knowledge-create-content">
                Content
              </Label>
              <Textarea
                className="h-60 font-mono text-small-body"
                data-testid="knowledge-create-content"
                id="knowledge-create-content"
                onChange={event => setContent(event.target.value)}
                value={content}
              />
            </div>
          </div>
        </div>
        {error ? (
          <div
            className="border-t border-line px-5 py-3 text-xs text-danger"
            data-testid="knowledge-create-dialog-error"
          >
            {error}
          </div>
        ) : null}
        <DialogFooter
          className="mx-0 mb-0 rounded-b-xl border-t border-line bg-transparent px-5 py-3"
          variant="ruled"
        >
          <Button
            data-testid="cancel-create-memory-btn"
            onClick={() => updateDialogOpen(false)}
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
            <Plus className="size-3" />
            Create
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export { KnowledgeCreateDialog };
export type { KnowledgeCreateDialogProps, KnowledgeCreateInput };
