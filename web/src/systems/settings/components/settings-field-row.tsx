import { cloneElement, isValidElement, useId, type ReactNode } from "react";

import { cn, Field, FieldContent, FieldDescription, FieldError, FieldLabel } from "@agh/ui";

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

const LABELABLE_TAGS = new Set([
  "button",
  "input",
  "meter",
  "output",
  "progress",
  "select",
  "textarea",
]);

function mergeAttributeTokens(...values: Array<string | undefined>): string | undefined {
  const tokens: string[] = [];
  for (const value of values) {
    if (!value) continue;
    for (const token of value.split(" ")) {
      const trimmed = token.trim();
      if (trimmed) tokens.push(trimmed);
    }
  }
  return tokens.length > 0 ? Array.from(new Set(tokens)).join(" ") : undefined;
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

  type ControlProps = {
    id?: string;
    role?: string;
    "aria-describedby"?: string;
    "aria-labelledby"?: string;
    "aria-invalid"?: boolean;
  };
  const controlElement = isValidElement<ControlProps>(control) ? control : null;
  const isGroupWrapper =
    controlElement !== null &&
    typeof controlElement.type === "string" &&
    controlElement.type === "div";
  const supportsNativeLabelAssociation =
    controlElement !== null &&
    typeof controlElement.type === "string" &&
    LABELABLE_TAGS.has(controlElement.type);

  let renderedControl = control;
  let labelHtmlFor: string | undefined;

  if (controlElement) {
    const describedBy = mergeAttributeTokens(
      controlElement.props["aria-describedby"],
      descriptionId,
      errorId
    );
    const labelledBy = mergeAttributeTokens(controlElement.props["aria-labelledby"], labelId);

    if (isGroupWrapper) {
      renderedControl = cloneElement(controlElement, {
        role: controlElement.props.role ?? "group",
        "aria-labelledby": labelledBy,
        "aria-describedby": describedBy,
      });
    } else {
      const controlId = controlElement.props.id ?? `${baseId}-control`;
      renderedControl = cloneElement(controlElement, {
        id: controlId,
        "aria-describedby": describedBy,
        "aria-labelledby": supportsNativeLabelAssociation
          ? controlElement.props["aria-labelledby"]
          : labelledBy,
        "aria-invalid": error ? true : controlElement.props["aria-invalid"],
      });
      if (supportsNativeLabelAssociation) labelHtmlFor = controlId;
    }
  }

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
        "flex min-h-[54px] items-center justify-between gap-5 border-t border-line-soft px-4 py-3 first:border-t-0",
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
            className="mt-0.5 max-w-[52ch] text-form-label leading-normal text-muted"
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
