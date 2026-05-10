import { cloneElement, isValidElement, type ReactNode, useId } from "react";

import { Field, FieldContent, FieldDescription, FieldError, FieldLabel, cn } from "@agh/ui";

interface SettingsFieldRowProps {
  label: string;
  description?: ReactNode;
  hint?: ReactNode;
  error?: ReactNode;
  control: ReactNode;
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

function SettingsFieldRow({
  label,
  description,
  hint,
  error,
  control,
  className,
  "data-testid": testId,
}: SettingsFieldRowProps) {
  const fallbackId = useId().replace(/:/g, "");
  const baseId =
    testId?.trim().replace(/[^a-zA-Z0-9_-]+/g, "-") || `settings-field-row-${fallbackId}`;
  const labelId = `${baseId}-label`;
  const descriptionId = description ? `${baseId}-description` : undefined;
  const errorId = error ? `${baseId}-error` : undefined;

  const mergeAttributeTokens = (...values: Array<string | undefined>) => {
    const tokens: string[] = [];
    for (const value of values) {
      if (!value) continue;
      for (const token of value.split(" ")) {
        const trimmed = token.trim();
        if (trimmed) tokens.push(trimmed);
      }
    }
    return tokens.length > 0 ? Array.from(new Set(tokens)).join(" ") : undefined;
  };

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
  let renderedLabel: ReactNode = (
    <FieldLabel
      id={labelId}
      className="text-sm font-medium text-(--fg)"
      data-testid={testId ? `${testId}-label` : undefined}
    >
      {label}
    </FieldLabel>
  );

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
        "aria-labelledby": labelledBy,
        "aria-invalid": error ? true : controlElement.props["aria-invalid"],
      });
      if (supportsNativeLabelAssociation) {
        renderedLabel = (
          <FieldLabel
            htmlFor={controlId}
            id={labelId}
            className="text-sm font-medium text-(--fg)"
            data-testid={testId ? `${testId}-label` : undefined}
          >
            {label}
          </FieldLabel>
        );
      }
    }
  }

  return (
    <Field
      orientation="vertical"
      className={cn(
        "grid gap-3 border-t border-(--line) pt-5 first:border-t-0 first:pt-0 lg:grid-cols-[minmax(0,17rem)_minmax(0,1fr)] lg:gap-x-8 lg:gap-y-0",
        className
      )}
      data-testid={testId}
    >
      <FieldContent className="min-w-0 gap-1.5">
        <div className="flex flex-wrap items-center gap-2">
          {renderedLabel}
          {hint ? (
            <span className="font-mono text-badge font-medium uppercase tracking-badge text-(--muted) lg:hidden">
              {hint}
            </span>
          ) : null}
        </div>
        {description ? (
          <FieldDescription
            id={descriptionId}
            className="max-w-136 text-xs leading-5 text-(--muted)"
          >
            {description}
          </FieldDescription>
        ) : null}
        {error ? (
          <FieldError id={errorId} className="text-xs text-(--danger)">
            {error}
          </FieldError>
        ) : null}
      </FieldContent>
      <div className="flex min-w-0 items-start lg:justify-self-start">
        <div className="flex w-full min-w-0 max-w-full flex-wrap items-center gap-3 lg:w-auto [&_input]:max-w-full [&_select]:max-w-full">
          {renderedControl}
          {hint ? (
            <span className="hidden font-mono text-badge font-medium uppercase tracking-badge text-(--muted) lg:inline">
              {hint}
            </span>
          ) : null}
        </div>
      </div>
    </Field>
  );
}

export { SettingsFieldRow };
