import { AlertCircle, Loader2, Trash2 } from "lucide-react";
import type { ReactNode } from "react";

import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@agh/ui";

interface SettingsDeleteDialogProps {
  open: boolean;
  slug: string;
  title: string;
  description?: ReactNode;
  fallbackNote?: ReactNode;
  error?: string | null;
  isDeleting: boolean;
  confirmLabel?: string;
  onConfirm: () => void;
  onOpenChange: (open: boolean) => void;
}

function SettingsDeleteDialog({
  open,
  slug,
  title,
  description,
  fallbackNote,
  error,
  isDeleting,
  confirmLabel,
  onConfirm,
  onOpenChange,
}: SettingsDeleteDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="grid w-[calc(100%-2rem)] max-w-md gap-4"
        data-testid={`settings-${slug}-delete`}
      >
        <DialogHeader>
          <DialogTitle data-testid={`settings-${slug}-delete-title`}>{title}</DialogTitle>
          {description ? (
            <DialogDescription data-testid={`settings-${slug}-delete-description`}>
              {description}
            </DialogDescription>
          ) : null}
        </DialogHeader>

        {fallbackNote ? (
          <div
            className="rounded-md border border-[color:var(--color-info)] bg-[color:var(--color-info-tint)] px-3 py-2 text-xs text-[color:var(--color-info)]"
            data-testid={`settings-${slug}-delete-fallback`}
          >
            {fallbackNote}
          </div>
        ) : null}

        {error ? (
          <div
            className="flex items-start gap-2 rounded-md border border-[color:var(--color-danger)] bg-[color:var(--color-danger-tint)] px-3 py-2 text-xs text-[color:var(--color-danger)]"
            role="alert"
            data-testid={`settings-${slug}-delete-error`}
          >
            <AlertCircle className="mt-0.5 size-3.5 shrink-0" />
            <span>{error}</span>
          </div>
        ) : null}

        <div className="flex items-center justify-end gap-2">
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => onOpenChange(false)}
            disabled={isDeleting}
            data-testid={`settings-${slug}-delete-cancel`}
          >
            Cancel
          </Button>
          <Button
            type="button"
            variant="destructive"
            size="sm"
            onClick={onConfirm}
            disabled={isDeleting}
            data-testid={`settings-${slug}-delete-confirm`}
          >
            {isDeleting ? (
              <Loader2 className="size-3.5 animate-spin" />
            ) : (
              <Trash2 className="size-3.5" />
            )}
            {isDeleting ? "Deleting…" : (confirmLabel ?? "Delete")}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}

export { SettingsDeleteDialog };
