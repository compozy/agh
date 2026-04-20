import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@agh/ui";

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

function jobDialogCopy(mode: "create" | "edit") {
  return {
    title: mode === "create" ? "Create job" : "Edit job",
    description: "Scheduled jobs dispatch prompts to agents on a time-based cadence.",
  };
}

function triggerDialogCopy(mode: "create" | "edit") {
  return {
    title: mode === "create" ? "Create trigger" : "Edit trigger",
    description: "Event-driven triggers react to daemon events, webhooks, and extension signals.",
  };
}

export function AutomationEditorDialog({ activeWorkspaceId, editor }: AutomationEditorDialogProps) {
  const isOpen = editor !== null;
  const copy = editor
    ? editor.kind === "jobs"
      ? jobDialogCopy(editor.mode)
      : triggerDialogCopy(editor.mode)
    : { title: "", description: "" };

  return (
    <Dialog
      onOpenChange={open => {
        if (!open) editor?.onCancel();
      }}
      open={isOpen}
    >
      <DialogContent
        className="gap-0 p-0 text-[color:var(--color-text-primary)] sm:max-w-[44rem]"
        data-testid="automation-editor-dialog"
      >
        {editor ? (
          <>
            <DialogHeader className="border-b border-[color:var(--color-divider)] px-5 py-4">
              <DialogTitle>{copy.title}</DialogTitle>
              <DialogDescription>{copy.description}</DialogDescription>
            </DialogHeader>

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
          </>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}
