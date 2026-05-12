import { AlertCircle } from "lucide-react";
import type { ReactNode } from "react";

import {
  Alert,
  AlertDescription,
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Spinner,
} from "@agh/ui";

type EditorMode = "create" | "edit";

interface SettingsEditorDialogProps {
  open: boolean;
  mode: EditorMode;
  title: string;
  slug?: string;
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
  const testSlug = slug ?? "modal";

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="grid w-[calc(100%-2rem)] max-w-xl gap-4 sm:max-w-2xl"
        unframed
        data-testid={`settings-${testSlug}-editor`}
        data-mode={mode}
      >
        <DialogHeader variant="ruled">
          <DialogTitle data-testid={`settings-${testSlug}-editor-title`}>{title}</DialogTitle>
          {description ? (
            <DialogDescription data-testid={`settings-${testSlug}-editor-description`}>
              {description}
            </DialogDescription>
          ) : null}
          {metadata ? (
            <div data-testid={`settings-${testSlug}-editor-metadata`} className="pt-1">
              {metadata}
            </div>
          ) : null}
        </DialogHeader>

        <div
          className="flex max-h-[60vh] flex-col gap-4 overflow-y-auto px-5 py-4"
          data-testid={`settings-${testSlug}-editor-body`}
        >
          {children}
        </div>

        <div
          className="flex flex-col gap-2 px-5"
          data-testid={`settings-${testSlug}-editor-feedback`}
        >
          {error ? (
            <Alert variant="danger" data-testid={`settings-${testSlug}-editor-error`}>
              <AlertCircle className="mt-0.5 size-3 shrink-0" />
              <AlertDescription className="text-xs">{error}</AlertDescription>
            </Alert>
          ) : null}
          {!error && warnings && warnings.length > 0 ? (
            <Alert variant="warning" data-testid={`settings-${testSlug}-editor-warnings`}>
              <AlertCircle className="mt-0.5 size-3 shrink-0" />
              <AlertDescription>
                <ul className="flex flex-col gap-1 text-xs">
                  {warnings.map(warning => (
                    <li key={warning}>{warning}</li>
                  ))}
                </ul>
              </AlertDescription>
            </Alert>
          ) : null}
        </div>

        <DialogFooter variant="ruled">
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => onOpenChange(false)}
            disabled={isSaving}
            data-testid={`settings-${testSlug}-editor-cancel`}
          >
            Cancel
          </Button>
          <Button
            type="button"
            variant="default"
            size="sm"
            onClick={onSave}
            disabled={!canSave || isSaving}
            data-testid={`settings-${testSlug}-editor-save`}
          >
            {isSaving ? <Spinner className="size-3" /> : null}
            {isSaving ? "Saving..." : computedSaveLabel}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export { SettingsEditorDialog };
export type { EditorMode };
