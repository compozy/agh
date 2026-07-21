"use client";

import { Dialog, DialogContent } from "@agh/ui";

import type { TaskRecord } from "../types";
import {
  TaskEditorSurface,
  type TaskEditorSurfaceMode,
  type TaskEditorSurfaceProps,
} from "./task-editor-surface";

export type TaskEditorModalMode = TaskEditorSurfaceMode;

export interface TaskEditorModalProps extends Omit<TaskEditorSurfaceProps, "onCancel"> {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Retained for modal callers that bind the persisted edit record. */
  task?: TaskRecord | null;
}

const MODAL_CONTENT_CLASS =
  "text-fg w-(--width-modal-md) max-w-[calc(100vw-2rem)] sm:max-w-(--width-modal-md) grid-rows-[minmax(0,1fr)] h-(--height-modal-md) max-h-[min(var(--height-modal-md),calc(100vh-2rem))]";

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
        <TaskEditorSurface {...surfaceProps} onCancel={() => onOpenChange(false)} />
      </DialogContent>
    </Dialog>
  );
}
