"use client";

import { SaveIcon, Undo2Icon } from "lucide-react";
import * as React from "react";

import { cn } from "../../lib/utils";
import { Button } from "../button";
import { Spinner } from "../spinner";

export interface PageActionsTopbarSlotProps extends React.ComponentProps<"div"> {
  /** Whether the underlying page has unsaved changes. */
  dirty: boolean;
  /** Save handler. */
  onSave: () => void;
  /** Discard handler. */
  onDiscard: () => void;
  /** Disables both buttons + swaps the save icon for a spinner. */
  saving?: boolean;
  /** Optional override label for the save button. */
  saveLabel?: React.ReactNode;
  /** Optional override label for the discard button. */
  discardLabel?: React.ReactNode;
  /** Optional override label rendered while `saving` is true. */
  savingLabel?: React.ReactNode;
}

function PageActionsTopbarSlot({
  dirty,
  onSave,
  onDiscard,
  saving = false,
  saveLabel = "Save changes",
  discardLabel = "Discard",
  savingLabel = "Saving...",
  className,
  ...props
}: PageActionsTopbarSlotProps) {
  const disabled = !dirty || saving;
  return (
    <div
      data-slot="page-actions-topbar-slot"
      data-dirty={dirty ? "true" : "false"}
      data-saving={saving ? "true" : undefined}
      className={cn("flex flex-wrap items-center justify-end gap-2", className)}
      {...props}
    >
      <Button
        type="button"
        variant="ghost"
        size="sm"
        data-slot="page-actions-topbar-slot-discard"
        disabled={disabled}
        onClick={onDiscard}
      >
        <Undo2Icon className="size-3.5" />
        {discardLabel}
      </Button>
      <Button
        type="button"
        variant="default"
        size="sm"
        data-slot="page-actions-topbar-slot-save"
        disabled={disabled}
        onClick={onSave}
      >
        {saving ? (
          <Spinner aria-hidden="true" className="size-3.5" />
        ) : (
          <SaveIcon className="size-3.5" />
        )}
        {saving ? savingLabel : saveLabel}
      </Button>
    </div>
  );
}

export { PageActionsTopbarSlot };
