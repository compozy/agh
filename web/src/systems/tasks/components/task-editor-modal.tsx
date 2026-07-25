"use client";

import { ClipboardCheck } from "lucide-react";

import { Dialog, DialogContent, EntityDialogHeader, dialogShellClass } from "@agh/ui";

import type { TaskRecord } from "../types";
import {
  TASK_DESCRIPTION,
  TaskEditorSurface,
  type TaskEditorSurfaceMode,
  type TaskEditorSurfaceProps,
} from "./task-editor-surface";

export type TaskEditorModalMode = TaskEditorSurfaceMode;

export interface TaskEditorModalProps extends Omit<TaskEditorSurfaceProps, "onCancel" | "header"> {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Retained for modal callers that bind the persisted edit record. */
  task?: TaskRecord | null;
}

/** The surface is the single grid child and owns its own header/body/footer rows. */
const MODAL_CONTENT_CLASS = `text-fg grid-rows-[minmax(0,1fr)] ${dialogShellClass("md", {
  fill: true,
})}`;

export function TaskEditorModal({
  open,
  onOpenChange,
  task: _task,
  ...surfaceProps
}: TaskEditorModalProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        unframed
        className={MODAL_CONTENT_CLASS}
        data-mode={surfaceProps.mode}
        data-testid="task-editor-modal"
        showCloseButton={false}
      >
        <TaskEditorSurface
          {...surfaceProps}
          header={
            <EntityDialogHeader
              description={TASK_DESCRIPTION}
              eyebrow="Autonomy · Task"
              icon={ClipboardCheck}
              onClose={() => onOpenChange(false)}
              title={surfaceProps.mode === "new" ? "Create task" : "Edit task"}
            />
          }
          onCancel={() => onOpenChange(false)}
        />
      </DialogContent>
    </Dialog>
  );
}
