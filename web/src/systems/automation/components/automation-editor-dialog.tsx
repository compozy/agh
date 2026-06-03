import type { LucideIcon } from "lucide-react";
import { CalendarClock, Zap } from "lucide-react";
import type { ReactNode } from "react";

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  Eyebrow,
} from "@agh/ui";

import type { AutomationDialogHandle } from "../lib/dialog-handle";
import type { WorkspaceOption } from "../lib/trigger-preview";
import type { CreateAutomationJobRequest, CreateAutomationTriggerRequest } from "../types";
import { AutomationJobForm } from "./automation-job-form";
import { AutomationTriggerForm } from "./automation-trigger-form";

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
  workspaces?: ReadonlyArray<WorkspaceOption>;
}

interface EditorHeaderCopy {
  icon: LucideIcon;
  eyebrow: string;
  title: string;
  description: ReactNode;
}

function jobHeaderCopy(mode: "create" | "edit"): EditorHeaderCopy {
  return {
    icon: CalendarClock,
    eyebrow: "Automation · Job",
    title: mode === "create" ? "Create job" : "Edit job",
    description: (
      <>
        A job runs an agent on a schedule, no operator in the loop. It answers three things:{" "}
        <b className="font-medium text-muted">which agent, what prompt, and when to dispatch.</b>
      </>
    ),
  };
}

function triggerHeaderCopy(mode: "create" | "edit"): EditorHeaderCopy {
  return {
    icon: Zap,
    eyebrow: "Automation · Trigger",
    title: mode === "create" ? "Create trigger" : "Edit trigger",
    description: (
      <>
        A trigger watches for a runtime event and, when it matches, runs an agent with a prompt
        built from that event.{" "}
        <b className="font-medium text-muted">When this happens → run that.</b>
      </>
    ),
  };
}

const WIDE_CONTENT_CLASS =
  "text-fg grid-rows-[auto_minmax(0,1fr)] w-(--width-modal-xl) sm:max-w-(--width-modal-xl) h-(--height-modal-xl) max-h-[92vh]";

export function AutomationEditorDialog({
  activeWorkspaceId,
  editor,
  handle,
  workspaces,
}: AutomationEditorDialogProps) {
  const isEditorOpen = editor !== null;

  return (
    <Dialog
      disablePointerDismissal
      handle={handle}
      open={isEditorOpen}
      onOpenChange={open => {
        if (!open) editor?.onCancel();
      }}
    >
      {editor ? (
        <DialogContent
          unframed
          className={WIDE_CONTENT_CLASS}
          data-testid="automation-editor-dialog"
        >
          <EditorHeader
            copy={
              editor.kind === "jobs" ? jobHeaderCopy(editor.mode) : triggerHeaderCopy(editor.mode)
            }
          />
          {editor.kind === "jobs" ? (
            <AutomationJobForm
              activeWorkspaceId={activeWorkspaceId}
              draft={editor.draft}
              isPending={editor.isPending}
              mode={editor.mode}
              onCancel={editor.onCancel}
              onChange={editor.onChange}
              onSubmit={editor.onSubmit}
              workspaces={workspaces}
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
              workspaces={workspaces}
            />
          )}
        </DialogContent>
      ) : null}
    </Dialog>
  );
}

function EditorHeader({ copy }: { copy: EditorHeaderCopy }) {
  const Icon = copy.icon;
  return (
    <DialogHeader variant="ruled">
      <div className="flex items-start gap-4">
        <div className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-accent-tint text-accent-strong ring-1 ring-accent-dim ring-inset">
          <Icon aria-hidden="true" className="size-4" />
        </div>
        <div className="min-w-0 flex-1">
          <Eyebrow className="text-accent-strong">{copy.eyebrow}</Eyebrow>
          <DialogTitle className="mt-1">{copy.title}</DialogTitle>
          <DialogDescription className="mt-1">{copy.description}</DialogDescription>
        </div>
      </div>
    </DialogHeader>
  );
}
