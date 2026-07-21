import { AlertCircle } from "lucide-react";
import { useEffect, useRef, useState, type ReactNode } from "react";

import { Button, cn, Spinner } from "@agh/ui";

const SAVED_FLASH_MS = 1400;

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

/**
 * Floating save bar (design system §07): sticky at the bottom of the content
 * column, hidden until there is something to say — dirty, saving, invalid, an
 * error, or warnings. The bar names its state; the primary action owns the
 * page's single accent.
 */
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
  // Saved flash: when a save completes cleanly the bar lingers briefly with a
  // success dot before dismissing (design system §07), instead of vanishing
  // mid-glance.
  const [justSaved, setJustSaved] = useState(false);
  const wasSaving = useRef(isSaving);
  useEffect(() => {
    const finishedCleanly = wasSaving.current && !isSaving && !error && !isDirty;
    wasSaving.current = isSaving;
    if (!finishedCleanly) return;
    setJustSaved(true);
    const timer = window.setTimeout(() => setJustSaved(false), SAVED_FLASH_MS);
    return () => window.clearTimeout(timer);
  }, [error, isDirty, isSaving]);

  const hasWarnings = Boolean(warnings && warnings.length > 0);
  const visible = isDirty || isSaving || Boolean(error) || hasWarnings || justSaved;
  if (!visible) return null;

  const disabled = !isDirty || isSaving || isInvalid;
  const liveRegion = error ? "assertive" : "polite";

  let dotClass = "bg-warning";
  let message: ReactNode = "Unsaved changes";
  if (isSaving) {
    message = "Saving…";
  } else if (error) {
    dotClass = "bg-danger";
    message = error;
  } else if (isInvalid && isDirty) {
    message = "Resolve validation errors before saving";
  } else if (!isDirty && justSaved) {
    dotClass = "bg-success";
    message = lastAppliedLabel ?? "Saved";
  }

  return (
    <div
      className={cn("pointer-events-none sticky bottom-6 z-10 flex justify-center px-4", className)}
    >
      <div
        aria-atomic="true"
        aria-live={liveRegion}
        className="pointer-events-auto flex w-full max-w-[560px] items-center gap-3 rounded-lg border border-line-strong bg-elevated py-2.5 pr-3 pl-4 shadow-overlay"
        data-dirty={isDirty ? "true" : "false"}
        data-testid={`settings-page-${slug}-save-bar`}
        role="status"
      >
        <span className="flex min-w-0 flex-1 items-center gap-2.5 text-small-body text-fg">
          {isSaving ? (
            <Spinner className="size-3 shrink-0 text-warning" />
          ) : error ? (
            <AlertCircle aria-hidden="true" className="size-3.5 shrink-0 text-danger" />
          ) : (
            <span aria-hidden="true" className={cn("size-[7px] shrink-0 rounded-full", dotClass)} />
          )}
          <span
            className={cn("truncate", error && "text-danger")}
            data-testid={`settings-page-${slug}-save-message`}
          >
            {message}
          </span>
          {hasWarnings && !error ? (
            <span
              className="truncate text-form-hint text-warning"
              data-testid={`settings-page-${slug}-save-warnings`}
            >
              {warnings?.join(" · ")}
            </span>
          ) : null}
        </span>
        <Button
          data-testid={`settings-page-${slug}-reset`}
          disabled={!isDirty || isSaving}
          onClick={onReset}
          size="sm"
          type="button"
          variant="ghost"
        >
          Discard
        </Button>
        <Button
          data-testid={`settings-page-${slug}-save`}
          disabled={disabled}
          onClick={onSave}
          size="sm"
          type="button"
        >
          Save changes
        </Button>
      </div>
    </div>
  );
}

export { SettingsSaveBar };
