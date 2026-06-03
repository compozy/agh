import { useEffect } from "react";
import { Zap } from "lucide-react";

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  Eyebrow,
} from "@agh/ui";

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

function triggerDialogTitle(mode: "create" | "edit") {
  return mode === "create" ? "Create trigger" : "Edit trigger";
}

const TRIGGER_CONTENT_CLASS =
  "text-fg grid-rows-[auto_minmax(0,1fr)] w-(--width-modal-xl) sm:max-w-(--width-modal-xl) h-(--height-modal-xl) max-h-[92vh]";

export function AutomationEditorDialog({
  activeWorkspaceId,
  editor,
  handle,
}: AutomationEditorDialogProps) {
  const isControlled = handle === undefined;
  const isEditorOpen = editor !== null;
  const isTrigger = editor?.kind === "triggers";

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
          unframed
          className={isTrigger ? TRIGGER_CONTENT_CLASS : "text-fg sm:max-w-176"}
          data-testid="automation-editor-dialog"
        >
          {editor.kind === "jobs" ? (
            <>
              <DialogHeader variant="ruled">
                <DialogTitle>{jobDialogCopy(editor.mode).title}</DialogTitle>
                <DialogDescription>{jobDialogCopy(editor.mode).description}</DialogDescription>
              </DialogHeader>
              <AutomationJobForm
                activeWorkspaceId={activeWorkspaceId}
                draft={editor.draft}
                isPending={editor.isPending}
                mode={editor.mode}
                onCancel={editor.onCancel}
                onChange={editor.onChange}
                onSubmit={editor.onSubmit}
              />
            </>
          ) : (
            <>
              <DialogHeader variant="ruled">
                <div className="flex items-start gap-4">
                  <div className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-accent-tint text-accent-strong ring-1 ring-accent-dim ring-inset">
                    <Zap aria-hidden="true" className="size-4" />
                  </div>
                  <div className="min-w-0 flex-1">
                    <Eyebrow className="text-accent-strong">Automation · Trigger</Eyebrow>
                    <DialogTitle className="mt-1">{triggerDialogTitle(editor.mode)}</DialogTitle>
                    <DialogDescription className="mt-1">
                      A trigger watches for a runtime event and, when it matches, runs an agent with
                      a prompt built from that event.{" "}
                      <b className="font-medium text-muted">When this happens → run that.</b>
                    </DialogDescription>
                  </div>
                </div>
              </DialogHeader>
              <AutomationTriggerForm
                activeWorkspaceId={activeWorkspaceId}
                draft={editor.draft}
                isPending={editor.isPending}
                mode={editor.mode}
                onCancel={editor.onCancel}
                onChange={editor.onChange}
                onSubmit={editor.onSubmit}
              />
            </>
          )}
        </DialogContent>
      ) : null}
    </Dialog>
  );
}
