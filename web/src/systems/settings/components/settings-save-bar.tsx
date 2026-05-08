import { AlertCircle, Check, Loader2, Save, Undo2 } from "lucide-react";

import { Button, cn } from "@agh/ui";

interface SettingsSaveBarProps {
  slug: string;
  isDirty: boolean;
  isSaving: boolean;
  isInvalid?: boolean;
  lastAppliedLabel?: string | null;
  error?: string | null;
  warnings?: string[];
  onSave: () => void;
  onReset: () => void;
  className?: string;
}

function SettingsSaveBar({
  slug,
  isDirty,
  isSaving,
  isInvalid = false,
  lastAppliedLabel,
  error,
  warnings,
  onSave,
  onReset,
  className,
}: SettingsSaveBarProps) {
  const disabled = !isDirty || isSaving || isInvalid;
  const liveRegion = error ? "assertive" : "polite";

  return (
    <div
      className={cn(
        "flex flex-col gap-4 bg-(--color-surface) px-4 py-4 sm:px-6 md:flex-row md:items-center md:justify-between md:px-8 xl:px-10",
        className
      )}
      data-testid={`settings-page-${slug}-save-bar`}
      data-dirty={isDirty ? "true" : "false"}
    >
      <div
        className="flex min-w-0 flex-1 flex-col gap-1 text-xs"
        role="status"
        aria-live={liveRegion}
        aria-atomic="true"
      >
        {error ? (
          <span
            className="flex items-center gap-1.5 text-(--color-danger)"
            data-testid={`settings-page-${slug}-save-error`}
          >
            <AlertCircle className="size-3.5" />
            {error}
          </span>
        ) : warnings && warnings.length > 0 ? (
          <ul
            className="flex flex-col gap-0.5 text-(--color-warning)"
            data-testid={`settings-page-${slug}-save-warnings`}
          >
            {warnings.map(warning => (
              <li key={warning} className="flex items-start gap-1.5">
                <AlertCircle className="mt-0.5 size-3.5 shrink-0" />
                <span>{warning}</span>
              </li>
            ))}
          </ul>
        ) : isInvalid ? (
          <span
            className="flex items-center gap-1.5 text-(--color-warning)"
            data-testid={`settings-page-${slug}-save-invalid`}
          >
            <AlertCircle className="size-3.5" />
            Resolve validation errors before saving
          </span>
        ) : isDirty ? (
          <span
            className="text-(--color-text-tertiary)"
            data-testid={`settings-page-${slug}-save-dirty`}
          >
            Unsaved changes
          </span>
        ) : lastAppliedLabel ? (
          <span
            className="flex items-center gap-1.5 text-(--color-text-tertiary)"
            data-testid={`settings-page-${slug}-save-applied`}
          >
            <Check className="size-3.5 text-success" />
            {lastAppliedLabel}
          </span>
        ) : (
          <span
            className="text-(--color-text-tertiary)"
            data-testid={`settings-page-${slug}-save-clean`}
          >
            No unsaved changes
          </span>
        )}
      </div>
      <div className="flex items-center justify-end gap-2">
        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={onReset}
          disabled={!isDirty || isSaving}
          data-testid={`settings-page-${slug}-reset`}
        >
          <Undo2 className="size-3.5" />
          Discard
        </Button>
        <Button
          type="button"
          variant="default"
          size="sm"
          onClick={onSave}
          disabled={disabled}
          data-testid={`settings-page-${slug}-save`}
        >
          {isSaving ? <Loader2 className="size-3.5 animate-spin" /> : <Save className="size-3.5" />}
          {isSaving ? "Saving…" : "Save changes"}
        </Button>
      </div>
    </div>
  );
}

export { SettingsSaveBar };
