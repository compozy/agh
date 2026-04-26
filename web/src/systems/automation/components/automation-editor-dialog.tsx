import { useEffect } from "react";

import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@agh/ui";

import { AutomationJobForm } from "./automation-job-form";
import { AutomationTriggerForm } from "./automation-trigger-form";
import type { AutomationDialogHandle } from "../lib/dialog-handle";
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
  handle?: AutomationDialogHandle;
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

export function AutomationEditorDialog({
  activeWorkspaceId,
  editor,
  handle,
}: AutomationEditorDialogProps) {
  const isControlled = handle === undefined;
  const isEditorOpen = editor !== null;
  const copy = editor
    ? editor.kind === "jobs"
      ? jobDialogCopy(editor.mode)
      : triggerDialogCopy(editor.mode)
    : { title: "", description: "" };

  useEffect(() => {
    if (!handle) {
      return;
    }

    if (isEditorOpen) {
      if (!handle.isOpen) {
        handle.open(null);
      }
      return;
    }

    if (handle.isOpen) {
      handle.close();
    }
  }, [handle, isEditorOpen]);

  return (
    <Dialog
      handle={handle}
      open={isControlled ? isEditorOpen : undefined}
      onOpenChange={open => {
        if (!open) editor?.onCancel();
      }}
    >
      {editor ? (
        <DialogContent
          className="gap-0 p-0 text-[color:var(--color-text-primary)] sm:max-w-[44rem]"
          data-testid="automation-editor-dialog"
        >
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
        </DialogContent>
      ) : null}
    </Dialog>
  );
}
