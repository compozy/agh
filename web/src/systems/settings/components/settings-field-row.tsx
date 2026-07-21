import { useId, type ReactNode } from "react";

import { cn, Field, FieldContent, FieldDescription, FieldError, FieldLabel } from "@agh/ui";
import { associateSettingsControl } from "../lib/control-association";

type SettingsFieldRowVariant = "default" | "modal";

interface SettingsFieldRowProps {
  label: string;
  /** The consequence sentence — what this does to the user's work. */
  description?: ReactNode;
  error?: ReactNode;
  control: ReactNode;
  variant?: SettingsFieldRowVariant;
  className?: string;
  "data-testid"?: string;
}

/**
 * One setting field = one decision (settings design system §05). Default
 * variant renders the srow anatomy — label + consequence left, one control
 * right, 54px minimum. `modal` keeps the vertical form layout for sheets and
 * dialogs. Jargon hint chips are gone; provenance belongs in Advanced only.
 */
function SettingsFieldRow({
  label,
  description,
  error,
  control,
  variant = "default",
  className,
  "data-testid": testId,
}: SettingsFieldRowProps) {
  const fallbackId = useId().replace(/:/g, "");
  const baseId =
    testId?.trim().replace(/[^a-zA-Z0-9_-]+/g, "-") || `settings-field-row-${fallbackId}`;
  const labelId = `${baseId}-label`;
  const descriptionId = description ? `${baseId}-description` : undefined;
  const errorId = error ? `${baseId}-error` : undefined;

  const { control: renderedControl, labelHtmlFor } = associateSettingsControl({
    control,
    controlId: `${baseId}-control`,
    labelId,
    descriptionId,
    errorId,
  });

  if (variant === "modal") {
    return (
      <Field
        orientation="vertical"
        data-variant={variant}
        className={cn(
          "grid gap-3 border-t border-line pt-5 pb-5 first:border-t-0 first:pt-0",
          className
        )}
        data-testid={testId}
      >
        <FieldContent className="min-w-0 gap-1.5">
          <FieldLabel
            className="text-sm font-medium text-fg"
            data-testid={testId ? `${testId}-label` : undefined}
            htmlFor={labelHtmlFor}
            id={labelId}
          >
            {label}
          </FieldLabel>
          {description ? (
            <FieldDescription id={descriptionId} className="max-w-136 text-xs leading-5 text-muted">
              {description}
            </FieldDescription>
          ) : null}
          {error ? (
            <FieldError id={errorId} className="text-xs text-danger">
              {error}
            </FieldError>
          ) : null}
        </FieldContent>
        <div className="flex w-full min-w-0 max-w-full flex-wrap items-center gap-3 [&_input]:max-w-full [&_select]:max-w-full">
          {renderedControl}
        </div>
      </Field>
    );
  }

  const LabelTag = labelHtmlFor ? "label" : "span";

  return (
    <div
      className={cn(
        "flex min-h-setting-row items-center justify-between gap-5 border-t border-line-soft px-4 py-3 first:border-t-0",
        className
      )}
      data-slot="settings-field-row"
      data-testid={testId}
    >
      <div className="min-w-0 flex-1">
        <LabelTag
          className="flex items-center gap-1.5 text-ws-name font-medium text-fg-strong"
          data-testid={testId ? `${testId}-label` : undefined}
          htmlFor={labelHtmlFor}
          id={labelId}
        >
          {label}
        </LabelTag>
        {description ? (
          <p
            className="mt-0.5 max-w-setting-description text-form-label leading-normal text-muted"
            id={descriptionId}
          >
            {description}
          </p>
        ) : null}
        {error ? (
          <p className="mt-1 text-form-hint text-danger" id={errorId} role="alert">
            {error}
          </p>
        ) : null}
      </div>
      <div className="flex shrink-0 flex-wrap items-center justify-end gap-2">
        {renderedControl}
      </div>
    </div>
  );
}

export { SettingsFieldRow };
export type { SettingsFieldRowVariant };
