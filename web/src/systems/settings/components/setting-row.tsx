import { ChevronRight } from "lucide-react";
import {
  cloneElement,
  Fragment,
  isValidElement,
  useId,
  type ComponentProps,
  type ReactNode,
} from "react";

import { cn } from "@agh/ui";

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
  const baseId = testId?.trim().replace(/[^a-zA-Z0-9_-]+/g, "-") || `setting-row-${fallbackId}`;
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
  const isFragment = controlElement?.type === Fragment;
  const supportsNativeLabel =
    controlElement !== null &&
    typeof controlElement.type === "string" &&
    LABELABLE_TAGS.has(controlElement.type);

  let renderedControl = control;
  let labelHtmlFor: string | undefined;
  if (controlElement && isFragment) {
    renderedControl = (
      <span
        aria-describedby={mergeAttributeTokens(descriptionId, errorId)}
        aria-labelledby={labelId}
        role="group"
      >
        {control}
      </span>
    );
  } else if (controlElement) {
    const describedBy = mergeAttributeTokens(
      controlElement.props["aria-describedby"],
      descriptionId,
      errorId
    );
    const labelledBy = mergeAttributeTokens(controlElement.props["aria-labelledby"], labelId);
    const controlId = controlElement.props.id ?? `${baseId}-control`;
    renderedControl = cloneElement(controlElement, {
      id: controlId,
      "aria-describedby": describedBy,
      "aria-labelledby": supportsNativeLabel ? controlElement.props["aria-labelledby"] : labelledBy,
      "aria-invalid": error ? true : controlElement.props["aria-invalid"],
    });
    if (supportsNativeLabel) labelHtmlFor = controlId;
  }

  const LabelTag = labelHtmlFor ? "label" : "span";

  return (
    <div
      className={cn(
        "flex min-h-[54px] items-center justify-between gap-5 border-t border-line-soft px-4 py-3 first:border-t-0",
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
      {renderedControl ? (
        <div className="flex shrink-0 items-center gap-2">{renderedControl}</div>
      ) : null}
    </div>
  );
}

export interface SettingLinkRowProps extends Omit<ComponentProps<"button">, "children" | "value"> {
  label: ReactNode;
  description?: ReactNode;
  /** Optional trailing value before the chevron. */
  value?: ReactNode;
}

/** Row that leads somewhere else — a sheet, a top-level route, a marketplace. */
export function SettingLinkRow({
  label,
  description,
  value,
  className,
  ...props
}: SettingLinkRowProps) {
  return (
    <button
      className={cn(
        "flex min-h-[54px] w-full items-center justify-between gap-5 border-t border-line-soft px-4 py-3 text-left first:border-t-0",
        "cursor-pointer transition-colors duration-base hover:bg-row-hover",
        "focus-visible:outline-none focus-visible:shadow-focus-ring",
        className
      )}
      data-slot="setting-link-row"
      type="button"
      {...props}
    >
      <span className="min-w-0 flex-1">
        <span className="flex items-center gap-1.5 text-ws-name font-medium text-fg-strong">
          {label}
        </span>
        {description ? (
          <span className="mt-0.5 block max-w-[52ch] text-form-label leading-normal text-muted">
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
