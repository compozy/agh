"use client";

import { ChevronRight, TerminalIcon } from "lucide-react";
import * as React from "react";

import { cn } from "../../lib/utils";
import { CodeBlock } from "./code-block";
import { Eyebrow } from "./eyebrow";
import { Markdown } from "./markdown";
import { ToolCallStatusIcon } from "./tool-call-status-icon";

export type ToolCallStatus = "pending" | "running" | "failed" | "success" | "empty";

export interface ToolCallRowSectionProps {
  children?: React.ReactNode;
  source?: string;
  format?: "markdown" | "code";
  /** Shiki language hint for `format="code"`. Ignored for markdown / children. */
  language?: string;
}

type ToolCallIconComponent = React.ComponentType<{
  className?: string;
  strokeWidth?: number;
}>;

export interface ToolCallRowProps extends Omit<React.ComponentProps<"div">, "title"> {
  toolName: React.ReactNode;
  /** Mono, truncated summary shown after the heading (command, path, pattern…). */
  preview?: React.ReactNode;
  status: ToolCallStatus;
  icon?: ToolCallIconComponent | React.ReactNode;
  /**
   * True only for a genuine runtime error (an AGH `output-error`), which tones the
   * heading danger. Error-shaped tool output keeps `status="failed"` (X glyph) with
   * `runtimeError` false, so its heading stays neutral.
   */
  runtimeError?: boolean;
  errorMessage?: React.ReactNode;
  expanded?: boolean;
  defaultExpanded?: boolean;
  onExpandedChange?: (expanded: boolean) => void;
  children?: React.ReactNode;
}

function isIconComponent(value: unknown): value is ToolCallIconComponent {
  if (typeof value === "function") return true;
  if (typeof value === "object" && value !== null && "render" in value) return true;
  return false;
}

function nativeTitle(value: React.ReactNode): string | undefined {
  return typeof value === "string" ? value : undefined;
}

function renderToolCallIcon(icon: ToolCallRowProps["icon"]): React.ReactNode {
  const iconClass =
    "size-3.5 shrink-0 text-faint transition-colors group-hover/tool-row:text-muted";
  if (icon === undefined) {
    return (
      <TerminalIcon
        aria-hidden="true"
        data-slot="tool-call-row-icon"
        className={iconClass}
        strokeWidth={1.75}
      />
    );
  }
  if (isIconComponent(icon)) {
    const IconComp = icon;
    return (
      <IconComp
        aria-hidden="true"
        data-slot="tool-call-row-icon"
        className={iconClass}
        strokeWidth={1.75}
      />
    );
  }
  return icon;
}

function ToolCallSectionBody({
  children,
  source,
  format,
  language,
}: Pick<ToolCallRowSectionProps, "children" | "source" | "format" | "language">) {
  if (children !== undefined && children !== null && children !== false) {
    return <>{children}</>;
  }
  const content = source ?? "";
  if (format === "code") {
    return (
      <CodeBlock code={content} language={language} showPrompt={false} copyable density="compact" />
    );
  }
  return (
    <Markdown compact className="max-w-none rounded-sm bg-canvas p-2 text-small-body text-muted">
      {content}
    </Markdown>
  );
}

function ToolCallRowSection({
  slot,
  label,
  children,
  source,
  format,
  language,
}: {
  slot: "input" | "output";
  label: string;
} & ToolCallRowSectionProps) {
  return (
    <div data-slot={`tool-call-row-${slot}`} className="flex min-w-0 flex-col gap-1.5">
      <Eyebrow className="text-subtle">{label}</Eyebrow>
      <div data-slot={`tool-call-row-${slot}-body`} className="min-w-0">
        <ToolCallSectionBody source={source} format={format} language={language}>
          {children}
        </ToolCallSectionBody>
      </div>
    </div>
  );
}

function ToolCallRowInput(props: ToolCallRowSectionProps) {
  return <ToolCallRowSection slot="input" label="Input" {...props} />;
}

function ToolCallRowOutput(props: ToolCallRowSectionProps) {
  return <ToolCallRowSection slot="output" label="Output" {...props} />;
}

/**
 * `ToolCallRow` renders one tool call as a single ~26px line —
 * `[icon well] [heading] [mono preview] [chevron] [status glyph]` — that expands
 * an inline indented body (params/outputs) on click or Enter/Space. It replaces
 * the filled `ToolCallCard` chrome with the transcript language: flat depth,
 * neutral `bg-hover` glaze, signal-toned status glyph as information.
 */
function ToolCallRowInner({
  toolName,
  preview,
  status,
  icon,
  runtimeError = false,
  errorMessage,
  expanded,
  defaultExpanded = false,
  onExpandedChange,
  children,
  className,
  ...props
}: ToolCallRowProps) {
  const [localExpanded, setLocalExpanded] = React.useState(defaultExpanded);
  const isExpanded = expanded ?? localExpanded;
  const expandable = Boolean(errorMessage) || React.Children.toArray(children).length > 0;
  const iconContent = renderToolCallIcon(icon);
  const headingDanger = status === "failed" && runtimeError;

  const setExpanded = React.useCallback(
    (next: boolean) => {
      if (expanded === undefined) {
        setLocalExpanded(next);
      }
      onExpandedChange?.(next);
    },
    [expanded, onExpandedChange]
  );

  const toggle = React.useCallback(() => {
    if (!expandable) return;
    setExpanded(!isExpanded);
  }, [expandable, isExpanded, setExpanded]);

  const handleTriggerKeyDown = React.useCallback(
    (event: React.KeyboardEvent<HTMLDivElement>) => {
      if (event.key !== "Enter" && event.key !== " ") return;
      event.preventDefault();
      toggle();
    },
    [toggle]
  );

  const rowContent = (
    <>
      <span
        data-slot="tool-call-row-icon-well"
        className="flex size-5 shrink-0 items-center justify-center"
      >
        {iconContent}
      </span>
      <span className="flex min-w-0 flex-1 items-baseline gap-1.5">
        <span
          data-slot="tool-call-row-tool"
          className={cn(
            "min-w-0 shrink truncate font-medium transition-colors",
            headingDanger ? "text-danger" : "text-muted group-hover/tool-row:text-fg"
          )}
          title={nativeTitle(toolName)}
        >
          {toolName}
        </span>
        {preview ? (
          <span
            data-slot="tool-call-row-preview"
            className="min-w-0 flex-1 truncate font-mono text-subtle"
            title={nativeTitle(preview)}
          >
            {preview}
          </span>
        ) : (
          <span className="min-w-0 flex-1" />
        )}
      </span>
      <span className="flex shrink-0 items-center gap-1 text-subtle">
        {expandable ? (
          <ChevronRight
            aria-hidden="true"
            data-slot="tool-call-row-chevron"
            className={cn(
              "size-3 shrink-0 text-faint transition-transform duration-base ease-out motion-reduce:transition-none",
              isExpanded ? "rotate-90 text-muted" : null
            )}
            strokeWidth={1.75}
          />
        ) : null}
        <ToolCallStatusIcon status={status} />
      </span>
    </>
  );

  return (
    <div
      data-slot="tool-call-row"
      data-status={status}
      data-expanded={expandable ? String(isExpanded) : undefined}
      className={cn("group/tool-row min-w-0", className)}
      {...props}
    >
      {expandable ? (
        <div
          data-slot="tool-call-row-trigger"
          role="button"
          tabIndex={0}
          aria-expanded={isExpanded}
          onClick={toggle}
          onKeyDown={handleTriggerKeyDown}
          className={cn(
            "flex min-h-6 w-full min-w-0 cursor-pointer items-center gap-1.5 rounded-sm px-1 text-left text-small-body",
            "transition-colors duration-base ease-out hover:bg-hover",
            "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-line-strong"
          )}
        >
          {rowContent}
        </div>
      ) : (
        <div
          data-slot="tool-call-row-static"
          className="flex min-h-6 w-full min-w-0 items-center gap-1.5 rounded-sm px-1 text-small-body"
        >
          {rowContent}
        </div>
      )}
      {expandable && isExpanded ? (
        <div
          data-slot="tool-call-row-body"
          className="mt-1 ml-7 flex max-h-64 min-w-0 cursor-default flex-col gap-2 overflow-auto border-l border-line pl-3 text-small-body text-muted select-text"
          onClick={event => event.stopPropagation()}
          onPointerDown={event => event.stopPropagation()}
        >
          {errorMessage ? (
            <p data-slot="tool-call-row-error" className="text-small-body text-danger">
              {errorMessage}
            </p>
          ) : null}
          {children}
        </div>
      ) : null}
    </div>
  );
}

const ToolCallRow = Object.assign(ToolCallRowInner, {
  Input: ToolCallRowInput,
  Output: ToolCallRowOutput,
});

export { ToolCallRow };
