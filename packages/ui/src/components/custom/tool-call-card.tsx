"use client";

import { ChevronRight, TerminalIcon } from "lucide-react";
import * as React from "react";
import { Streamdown } from "streamdown";

import { cn } from "../../lib/utils";
import { STREAMDOWN_SAFE_CONFIG } from "./description-card";
import { Eyebrow } from "./eyebrow";
import { Pill, type PillTone } from "./pill";
import { Time } from "./time";

export type ToolCallStatus = "pending" | "in_progress" | "completed" | "failed";

type ToolCallIconComponent = React.ComponentType<{
  className?: string;
  size?: number;
  strokeWidth?: number;
}>;

export interface ToolCallCardProps extends Omit<React.ComponentProps<"section">, "title"> {
  toolName: React.ReactNode;
  filePath?: React.ReactNode;
  status: ToolCallStatus;
  icon?: ToolCallIconComponent | React.ReactNode;
  timestamp?: string;
  actions?: React.ReactNode;
  errorMessage?: React.ReactNode;
  children?: React.ReactNode;
}

export interface ToolCallCardSectionProps {
  children?: React.ReactNode;
  source?: string;
  format?: "markdown" | "code";
  defaultOpen?: boolean;
}

const STATUS_TONE: Record<ToolCallStatus, PillTone> = {
  pending: "neutral",
  in_progress: "info",
  completed: "success",
  failed: "danger",
};

const STATUS_LABEL: Record<ToolCallStatus, string> = {
  pending: "Pending",
  in_progress: "Running",
  completed: "Done",
  failed: "Error",
};

function isIconComponent(value: unknown): value is ToolCallIconComponent {
  if (typeof value === "function") return true;
  if (typeof value === "object" && value !== null && "render" in value) return true;
  return false;
}

type ToolCallSectionSlot = "input" | "output";

function ToolCallSection({
  label,
  slot,
  defaultOpen = false,
  children,
  source,
  format = "markdown",
}: ToolCallCardSectionProps & { label: string; slot: ToolCallSectionSlot }) {
  const [userOpen, setUserOpen] = React.useState<boolean | null>(null);
  const panelId = React.useId();
  const open = userOpen ?? defaultOpen;
  return (
    <div
      data-slot={`tool-call-card-${slot}`}
      data-open={open ? "true" : "false"}
      className="flex flex-col"
    >
      <button
        type="button"
        data-slot={`tool-call-card-${slot}-toggle`}
        aria-expanded={open}
        aria-controls={panelId}
        onClick={() => setUserOpen(prev => !(prev ?? defaultOpen))}
        className="inline-flex w-full items-center gap-1.5 rounded-xs p-1 text-left transition-colors duration-base ease-out hover:bg-hover focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-line-strong"
      >
        <ChevronRight
          width={12}
          height={12}
          strokeWidth={1.75}
          className={cn(
            "shrink-0 text-muted transition-transform duration-base ease-out",
            open && "rotate-90"
          )}
        />
        <Eyebrow className="text-muted">{label}</Eyebrow>
      </button>
      {open ? (
        <div id={panelId} data-slot={`tool-call-card-${slot}-body`} className="mt-1 pl-5">
          <ToolCallSectionBody source={source} format={format}>
            {children}
          </ToolCallSectionBody>
        </div>
      ) : null}
    </div>
  );
}

function ToolCallSectionBody({
  children,
  source,
  format,
}: Pick<ToolCallCardSectionProps, "children" | "source" | "format">) {
  if (children !== undefined && children !== null && children !== false) {
    return <>{children}</>;
  }
  const content = source ?? "";
  if (format === "code") {
    return (
      <pre className="overflow-x-auto rounded bg-canvas p-3 font-mono text-form-label text-fg">
        <code>{content}</code>
      </pre>
    );
  }
  return (
    <div
      data-slot="tool-call-card-markdown"
      className="text-card-title leading-relaxed text-fg [&_code]:font-mono [&_code]:text-form-label [&_p]:my-2"
    >
      <Streamdown {...STREAMDOWN_SAFE_CONFIG}>{content}</Streamdown>
    </div>
  );
}

function ToolCallCardInput(props: ToolCallCardSectionProps) {
  return <ToolCallSection {...props} label="Input" slot="input" />;
}

function ToolCallCardOutput(props: ToolCallCardSectionProps) {
  return <ToolCallSection {...props} label="Output" slot="output" />;
}

function hasBodyContent(errorMessage: React.ReactNode, children: React.ReactNode): boolean {
  if (errorMessage !== undefined && errorMessage !== null && errorMessage !== false) {
    return true;
  }
  return React.Children.count(children) > 0;
}

/**
 * Inline tool-execution card per DESIGN.md §4 "Tool Call Card". Surface bg with
 * a 1px divider border between header and body, terminal icon + tool name +
 * optional file path, status pill + optional timestamp + actions pinned right.
 *
 * Compose `<ToolCallCard.Input>` and `<ToolCallCard.Output>` as children for
 * collapsible argument/result regions (closed by default), or pass raw children
 * for diffs, stdout, or any other body content.
 */
const ToolCallCard = Object.assign(
  function ToolCallCard({
    toolName,
    filePath,
    status,
    icon,
    timestamp,
    actions,
    errorMessage,
    children,
    className,
    ...props
  }: ToolCallCardProps) {
    let iconContent: React.ReactNode;
    const iconClass = "size-3 shrink-0 text-subtle";
    if (icon === undefined) {
      iconContent = (
        <TerminalIcon
          aria-hidden="true"
          data-slot="tool-call-card-icon"
          className={iconClass}
          strokeWidth={1.75}
        />
      );
    } else if (isIconComponent(icon)) {
      const IconComp = icon;
      iconContent = (
        <IconComp
          aria-hidden="true"
          data-slot="tool-call-card-icon"
          className={iconClass}
          strokeWidth={1.75}
        />
      );
    } else {
      iconContent = icon;
    }
    const showBody = hasBodyContent(errorMessage, children);
    return (
      <section
        data-slot="tool-call-card"
        data-status={status}
        className={cn(
          "overflow-hidden rounded-md bg-canvas-soft",
          "data-[status=failed]:border data-[status=failed]:border-danger/40",
          className
        )}
        {...props}
      >
        <header
          data-slot="tool-call-card-header"
          className="flex min-w-0 items-center gap-3 px-4 py-2.5"
        >
          {iconContent}
          <span data-slot="tool-call-card-tool" className="text-card-title font-medium text-fg">
            {toolName}
          </span>
          {filePath ? (
            <span
              data-slot="tool-call-card-path"
              className="min-w-0 truncate text-small-body text-subtle"
            >
              {filePath}
            </span>
          ) : null}
          <div className="ml-auto flex shrink-0 items-center gap-2">
            <Pill tone={STATUS_TONE[status]} data-slot="tool-call-card-status">
              {STATUS_LABEL[status]}
            </Pill>
            {timestamp ? (
              <Time
                data-slot="tool-call-card-time"
                iso={timestamp}
                mode="relative"
                className="text-form-label text-muted"
              />
            ) : null}
            {actions ? (
              <div data-slot="tool-call-card-actions" className="flex items-center gap-1.5">
                {actions}
              </div>
            ) : null}
          </div>
        </header>
        {showBody ? (
          <div
            data-slot="tool-call-card-body"
            className="flex flex-col gap-2 border-t border-line px-4 py-3"
          >
            {errorMessage ? (
              <p data-slot="tool-call-card-error" className="text-form-input text-danger">
                {errorMessage}
              </p>
            ) : null}
            {children}
          </div>
        ) : null}
      </section>
    );
  },
  {
    Input: ToolCallCardInput,
    Output: ToolCallCardOutput,
  }
);

export {
  ToolCallCard,
  STATUS_LABEL as TOOL_CALL_STATUS_LABEL,
  STATUS_TONE as TOOL_CALL_STATUS_TONE,
};
