import { X } from "lucide-react";

import { Button } from "@agh/ui";
import { Dialog, DialogClose, DialogContent } from "@/components/ui/dialog";

import { AutomationJobForm } from "./automation-job-form";
import { AutomationTriggerForm } from "./automation-trigger-form";
import type { CreateAutomationJobRequest, CreateAutomationTriggerRequest } from "../types";

type AutomationDialogEditorState =
  | {
      draft: CreateAutomationJobRequest;
      isPending: boolean;
      kind: "jobs";
      mode: "create" | "edit";
      onCancel: () => void;
      onChange: (draft: CreateAutomationJobRequest) => void;
      onSubmit: () => void;
    }
  | {
      draft: CreateAutomationTriggerRequest;
      isPending: boolean;
      kind: "triggers";
      mode: "create" | "edit";
      onCancel: () => void;
      onChange: (draft: CreateAutomationTriggerRequest) => void;
      onSubmit: () => void;
    };

interface AutomationEditorDialogProps {
  activeWorkspaceId?: string | null;
  editor: AutomationDialogEditorState | null;
}

export function AutomationEditorDialog({ activeWorkspaceId, editor }: AutomationEditorDialogProps) {
  return (
    <Dialog
      open={editor !== null}
      onOpenChange={open => {
        if (!open) {
          editor?.onCancel();
        }
      }}
    >
      {editor ? (
        <DialogContent
          className="max-h-[min(84vh,960px)] max-w-[min(100%-2rem,880px)] overflow-hidden border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] p-0 text-[color:var(--color-text-primary)]"
          showCloseButton={false}
        >
          <DialogClose
            render={
              <Button
                className="absolute top-4 right-4 z-10 text-[color:var(--color-text-tertiary)] hover:bg-[color:var(--color-hover)] hover:text-[color:var(--color-text-primary)]"
                size="icon-sm"
                variant="ghost"
              />
            }
          >
            <X className="size-4" />
            <span className="sr-only">Close editor</span>
          </DialogClose>
          {editor.kind === "jobs" ? (
            <AutomationJobForm
              activeWorkspaceId={activeWorkspaceId}
              draft={editor.draft}
              isPending={editor.isPending}
              mode={editor.mode}
              onCancel={editor.onCancel}
              onChange={editor.onChange}
              onSubmit={editor.onSubmit}
            />
          ) : (
            <AutomationTriggerForm
              activeWorkspaceId={activeWorkspaceId}
              draft={editor.draft}
              isPending={editor.isPending}
              mode={editor.mode}
              onCancel={editor.onCancel}
              onChange={editor.onChange}
              onSubmit={editor.onSubmit}
            />
          )}
        </DialogContent>
      ) : null}
    </Dialog>
  );
}
