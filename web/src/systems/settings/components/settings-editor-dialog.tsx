import { AlertCircle, Loader2 } from "lucide-react";
import type { ReactNode } from "react";

import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@agh/ui";

type EditorMode = "create" | "edit";

interface SettingsEditorDialogProps {
  open: boolean;
  mode: EditorMode;
  title: string;
  slug: string;
  description?: ReactNode;
  metadata?: ReactNode;
  error?: string | null;
  warnings?: string[];
  canSave: boolean;
  isSaving: boolean;
  saveLabel?: string;
  onSave: () => void;
  onOpenChange: (open: boolean) => void;
  children: ReactNode;
}

function SettingsEditorDialog({
  open,
  mode,
  title,
  slug,
  description,
  metadata,
  error,
  warnings,
  canSave,
  isSaving,
  saveLabel,
  onSave,
  onOpenChange,
  children,
}: SettingsEditorDialogProps) {
  const computedSaveLabel = saveLabel ?? (mode === "create" ? "Create" : "Save changes");

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="grid w-[calc(100%-2rem)] max-w-xl gap-4 sm:max-w-2xl"
        data-testid={`settings-${slug}-editor`}
        data-mode={mode}
      >
        <DialogHeader>
          <DialogTitle data-testid={`settings-${slug}-editor-title`}>{title}</DialogTitle>
          {description ? (
            <DialogDescription data-testid={`settings-${slug}-editor-description`}>
              {description}
            </DialogDescription>
          ) : null}
          {metadata ? (
            <div data-testid={`settings-${slug}-editor-metadata`} className="pt-1">
              {metadata}
            </div>
          ) : null}
        </DialogHeader>

        <div
          className="flex max-h-[60vh] flex-col gap-4 overflow-y-auto"
          data-testid={`settings-${slug}-editor-body`}
        >
          {children}
        </div>

        <div className="flex flex-col gap-2" data-testid={`settings-${slug}-editor-feedback`}>
          {error ? (
            <div
              className="flex items-start gap-2 rounded-md border border-[color:var(--color-danger)] bg-[color:var(--color-danger-tint)] px-3 py-2 text-xs text-[color:var(--color-danger)]"
              role="alert"
              data-testid={`settings-${slug}-editor-error`}
            >
              <AlertCircle className="mt-0.5 size-3.5 shrink-0" />
              <span>{error}</span>
            </div>
          ) : null}
          {!error && warnings && warnings.length > 0 ? (
            <ul
              className="flex flex-col gap-1 rounded-md border border-[color:var(--color-warning)] bg-[color:var(--color-warning-tint)] px-3 py-2 text-xs text-[color:var(--color-warning)]"
              data-testid={`settings-${slug}-editor-warnings`}
            >
              {warnings.map(warning => (
                <li key={warning} className="flex items-start gap-1.5">
                  <AlertCircle className="mt-0.5 size-3.5 shrink-0" />
                  <span>{warning}</span>
                </li>
              ))}
            </ul>
          ) : null}
        </div>

        <div className="flex items-center justify-end gap-2">
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => onOpenChange(false)}
            disabled={isSaving}
            data-testid={`settings-${slug}-editor-cancel`}
          >
            Cancel
          </Button>
          <Button
            type="button"
            variant="default"
            size="sm"
            onClick={onSave}
            disabled={!canSave || isSaving}
            data-testid={`settings-${slug}-editor-save`}
          >
            {isSaving ? <Loader2 className="size-3.5 animate-spin" /> : null}
            {isSaving ? "Saving…" : computedSaveLabel}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}

export { SettingsEditorDialog };
export type { EditorMode };
