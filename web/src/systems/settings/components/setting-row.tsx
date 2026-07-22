import { ChevronRight } from "lucide-react";
import { useId, type ComponentProps, type ReactNode } from "react";
import { Link } from "@tanstack/react-router";

import { cn, Field, FieldContent, FieldDescription, FieldError, FieldLabel } from "@agh/ui";
import { associateSettingsControl } from "../lib/control-association";

export interface SettingRowProps {
  /** Plain-language decision name (13px/500). */
  label: ReactNode;
  /** The consequence sentence — what this does to the user's work (≤52ch). */
  description?: ReactNode;
  error?: ReactNode;
  /** Single control, right-aligned. Labelled via aria association automatically. */
  control?: ReactNode;
  className?: string;
  "data-testid"?: string;
}

function settingRowIds(testId: string | undefined, fallbackId: string) {
  const baseId = testId?.trim().replace(/[^a-zA-Z0-9_-]+/g, "-") || `setting-row-${fallbackId}`;
  return {
    baseId,
    labelId: `${baseId}-label`,
  };
}

/**
 * One row = one decision (settings design system §05). Label + consequence
 * left, one control right, 54px minimum, hairline separators inside a
 * panelbox. Jargon chips are banned here — provenance lives in Advanced.
 */
export function SettingRow({
  label,
  description,
  error,
  control,
  className,
  "data-testid": testId,
}: SettingRowProps) {
  const fallbackId = useId().replace(/:/g, "");
  const { baseId, labelId } = settingRowIds(testId, fallbackId);
  const descriptionId = description ? `${baseId}-description` : undefined;
  const errorId = error ? `${baseId}-error` : undefined;
  const { control: renderedControl, labelHtmlFor } = associateSettingsControl({
    control,
    controlId: `${baseId}-control`,
    labelId,
    descriptionId,
    errorId,
  });

  const LabelTag = labelHtmlFor ? "label" : "span";

  return (
    <div
      className={cn(
        "flex min-h-setting-row items-center justify-between gap-5 border-t border-line-soft px-4 py-3 first:border-t-0",
        className
      )}
      data-slot="setting-row"
      data-testid={testId}
    >
      <div className="min-w-0 flex-1">
        <LabelTag
          className="flex items-center gap-1.5 text-ws-name font-medium text-fg-strong"
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
      {renderedControl ? (
        <div className="flex shrink-0 items-center gap-2">{renderedControl}</div>
      ) : null}
    </div>
  );
}

/** Modal/dialog form of SettingRow — stacked Field layout instead of inline srow. */
export function ModalSettingRow({
  label,
  description,
  error,
  control,
  className,
  "data-testid": testId,
}: SettingRowProps) {
  const fallbackId = useId().replace(/:/g, "");
  const { baseId, labelId } = settingRowIds(testId, fallbackId);
  const descriptionId = description ? `${baseId}-description` : undefined;
  const errorId = error ? `${baseId}-error` : undefined;
  const { control: renderedControl, labelHtmlFor } = associateSettingsControl({
    control,
    controlId: `${baseId}-control`,
    labelId,
    descriptionId,
    errorId,
  });

  return (
    <Field
      className={cn(
        "grid gap-3 border-t border-line pt-5 pb-5 first:border-t-0 first:pt-0",
        className
      )}
      data-testid={testId}
      orientation="vertical"
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
          <FieldDescription className="max-w-136 text-xs leading-5 text-muted" id={descriptionId}>
            {description}
          </FieldDescription>
        ) : null}
        {error ? (
          <FieldError className="text-xs text-danger" id={errorId}>
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

interface SettingNavigationRowContentProps {
  label: ReactNode;
  description?: ReactNode;
  /** Optional trailing value before the chevron. */
  value?: ReactNode;
}

export interface SettingLinkRowProps
  extends
    Omit<ComponentProps<typeof Link>, "children" | "value">,
    SettingNavigationRowContentProps {}

/** Row that leads somewhere else — a sheet, a top-level route, a marketplace. */
export function SettingLinkRow({
  label,
  description,
  value,
  className,
  ...props
}: SettingLinkRowProps) {
  return (
    <Link
      className={cn(
        "flex min-h-setting-row w-full items-center justify-between gap-5 border-t border-line-soft px-4 py-3 text-left first:border-t-0",
        "cursor-pointer transition-colors duration-base hover:bg-row-hover",
        "focus-visible:outline-none focus-visible:shadow-focus-ring",
        className
      )}
      data-slot="setting-link-row"
      {...props}
    >
      <span className="min-w-0 flex-1">
        <span className="flex items-center gap-1.5 text-ws-name font-medium text-fg-strong">
          {label}
        </span>
        {description ? (
          <span className="mt-0.5 block max-w-setting-description text-form-label leading-normal text-muted">
            {description}
          </span>
        ) : null}
      </span>
      <span className="flex shrink-0 items-center gap-2">
        {value}
        <ChevronRight aria-hidden="true" className="size-3.5 text-faint" />
      </span>
    </Link>
  );
}

export interface SettingActionRowProps
  extends Omit<ComponentProps<"button">, "children" | "value">, SettingNavigationRowContentProps {}

/** Row that performs an in-place action, such as opening a settings sheet. */
export function SettingActionRow({
  label,
  description,
  value,
  className,
  ...props
}: SettingActionRowProps) {
  return (
    <button
      className={cn(
        "flex min-h-setting-row w-full items-center justify-between gap-5 border-t border-line-soft px-4 py-3 text-left first:border-t-0",
        "cursor-pointer transition-colors duration-base hover:bg-row-hover",
        "focus-visible:outline-none focus-visible:shadow-focus-ring",
        className
      )}
      data-slot="setting-action-row"
      type="button"
      {...props}
    >
      <span className="min-w-0 flex-1">
        <span className="flex items-center gap-1.5 text-ws-name font-medium text-fg-strong">
          {label}
        </span>
        {description ? (
          <span className="mt-0.5 block max-w-setting-description text-form-label leading-normal text-muted">
            {description}
          </span>
        ) : null}
      </span>
      <span className="flex shrink-0 items-center gap-2">
        {value}
        <ChevronRight aria-hidden="true" className="size-3.5 text-faint" />
      </span>
    </button>
  );
}

/** Mono read-only value voice for facts the daemon owns. */
export function SettingValue({ mono = false, children }: { mono?: boolean; children: ReactNode }) {
  return (
    <span
      className={cn(
        "text-small-body font-medium text-fg",
        mono && "font-mono text-eyebrow font-normal text-muted"
      )}
    >
      {children}
    </span>
  );
}
