"use client";

import { ChevronRight, TerminalIcon } from "lucide-react";
import * as React from "react";

import { cn } from "../../lib/utils";
import { CodeBlock } from "./code-block";
import { Eyebrow } from "./eyebrow";
import {
  ToolCallCardContext,
  TOOL_CALL_INPUT_SLOT,
  TOOL_CALL_OUTPUT_SLOT,
  useToolCallCardContext,
  useToolCallCardState,
  type ToolCallSectionSlot,
} from "./hooks/use-tool-call-card";
import { Markdown } from "./markdown";
import { ToolCallStatusIcon } from "./tool-call-status-icon";

export type ToolCallStatus = "pending" | "in_progress" | "completed" | "failed";

export interface ToolCallCardSectionProps {
  children?: React.ReactNode;
  source?: string;
  format?: "markdown" | "code";
  /** Shiki language hint for `format="code"`. Ignored for markdown / children. */
  language?: string;
  defaultOpen?: boolean;
}

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
  actions?: React.ReactNode;
  errorMessage?: React.ReactNode;
  children?: React.ReactNode;
}

const STATUS_LABEL: Record<ToolCallStatus, string> = {
  pending: "Pending",
  in_progress: "Running",
  completed: "Done",
  failed: "Error",
};

const SLOT_ORDER: ToolCallSectionSlot[] = ["input", "output"];

const SLOT_LABEL: Record<ToolCallSectionSlot, string> = {
  input: "Input",
  output: "Output",
};

function isIconComponent(value: unknown): value is ToolCallIconComponent {
  if (typeof value === "function") return true;
  if (typeof value === "object" && value !== null && "render" in value) return true;
  return false;
}

function ToolCallSectionBody({
  children,
  source,
  format,
  language,
}: Pick<ToolCallCardSectionProps, "children" | "source" | "format" | "language">) {
  if (children !== undefined && children !== null && children !== false) {
    return <>{children}</>;
  }
  const content = source ?? "";
  if (format === "code") {
    // Routes through the canonical CodeBlock so shiki highlighting + copy
    // chrome match every other code surface in the runtime UI.
    return <CodeBlock code={content} language={language} showPrompt={false} copyable />;
  }
  return (
    <Markdown
      compact
      className="max-w-none rounded-sm border border-line bg-canvas p-2.5 text-card-title text-muted"
    >
      {content}
    </Markdown>
  );
}

function ToolCallDisclosureChip({
  slot,
  label,
  open,
  panelId,
  onToggle,
}: {
  slot: ToolCallSectionSlot;
  label: string;
  open: boolean;
  panelId: string;
  onToggle: () => void;
}) {
  return (
    <button
      type="button"
      data-slot={`tool-call-card-${slot}-toggle`}
      data-open={open ? "true" : "false"}
      aria-expanded={open}
      aria-controls={panelId}
      onClick={onToggle}
      className={cn(
        "inline-flex h-5 shrink-0 cursor-pointer items-center gap-1.5 rounded-xs border px-2 pl-1.5 transition-colors duration-base ease-out",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-line-strong",
        open
          ? "border-accent/32 bg-accent-tint text-accent"
          : "border-line text-muted hover:border-line-strong hover:bg-hover hover:text-fg"
      )}
    >
      <ChevronRight
        width={9}
        height={9}
        strokeWidth={1.75}
        aria-hidden="true"
        className={cn(
          "shrink-0 text-faint transition-transform duration-base ease-out motion-reduce:transition-none",
          open ? "rotate-90 text-accent" : "text-faint"
        )}
      />
      <Eyebrow className={cn(open ? "text-accent" : "text-muted")}>{label}</Eyebrow>
    </button>
  );
}

function ToolCallCardHeaderChips() {
  const { registeredSlots, openSlots, toggleSlot, panelIds } = useToolCallCardContext();
  const registered = SLOT_ORDER.filter(slot => registeredSlots[slot] === true);
  if (registered.length === 0) return null;

  return (
    <div
      data-slot="tool-call-card-chip-group"
      className="inline-flex shrink-0 items-center gap-1.5"
    >
      {registered.map(slot => (
        <ToolCallDisclosureChip
          key={slot}
          slot={slot}
          label={SLOT_LABEL[slot]}
          open={openSlots[slot]}
          panelId={panelIds[slot]}
          onToggle={() => toggleSlot(slot)}
        />
      ))}
    </div>
  );
}

/**
 * Renders the body panel for a registered slot **in its declared JSX
 * position**. Returning the same tree shape across re-renders (instead of
 * pulling children out of context state) is what keeps async children like
 * `<CodeBlock>` mounted, so their shiki highlight promises survive parent
 * re-renders.
 */
function ToolCallCardSlotPanel({
  slot,
  children,
  source,
  format,
  language,
  defaultOpen,
}: {
  slot: ToolCallSectionSlot;
} & ToolCallCardSectionProps) {
  const { registerSlot, unregisterSlot, openSlots, panelIds } = useToolCallCardContext();
  const open = openSlots[slot];

  React.useLayoutEffect(() => {
    registerSlot(slot, { defaultOpen: defaultOpen ?? false });
    return () => unregisterSlot(slot);
  }, [registerSlot, unregisterSlot, slot, defaultOpen]);

  if (!open) return null;
  return (
    <div data-slot={`tool-call-card-${slot}`} data-open="true" className="flex flex-col gap-1.5">
      <Eyebrow className="text-accent">{SLOT_LABEL[slot]}</Eyebrow>
      <div id={panelIds[slot]} data-slot={`tool-call-card-${slot}-body`}>
        <ToolCallSectionBody source={source} format={format} language={language}>
          {children}
        </ToolCallSectionBody>
      </div>
    </div>
  );
}

function ToolCallCardInput(props: ToolCallCardSectionProps) {
  return <ToolCallCardSlotPanel slot="input" {...props} />;
}
ToolCallCardInput.slotMarker = TOOL_CALL_INPUT_SLOT;

function ToolCallCardOutput(props: ToolCallCardSectionProps) {
  return <ToolCallCardSlotPanel slot="output" {...props} />;
}
ToolCallCardOutput.slotMarker = TOOL_CALL_OUTPUT_SLOT;

function renderToolCallIcon(icon: ToolCallCardProps["icon"]): React.ReactNode {
  const iconClass = "size-3.5 shrink-0 text-faint";
  if (icon === undefined) {
    return (
      <TerminalIcon
        aria-hidden="true"
        data-slot="tool-call-card-icon"
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
        data-slot="tool-call-card-icon"
        className={iconClass}
        strokeWidth={1.75}
      />
    );
  }
  return icon;
}

function ToolCallCardInner({
  toolName,
  filePath,
  status,
  icon,
  actions,
  errorMessage,
  children,
  className,
  ...props
}: ToolCallCardProps) {
  const { contextValue, slotChildren, rawChildren, hasError, hasRawChildren, showBody } =
    useToolCallCardState(children, errorMessage);

  const iconContent = renderToolCallIcon(icon);

  return (
    <ToolCallCardContext.Provider value={contextValue}>
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
          className="flex min-h-11 min-w-0 items-center gap-3 px-4 py-2.5"
        >
          {iconContent}
          <span data-slot="tool-call-card-tool" className="text-card-title font-medium text-fg">
            {toolName}
          </span>
          {filePath ? (
            <span
              data-slot="tool-call-card-path"
              className="min-w-0 flex-1 truncate font-mono text-small-body text-subtle"
            >
              {filePath}
            </span>
          ) : (
            <span className="min-w-0 flex-1" aria-hidden="true" />
          )}
          <div className="ml-auto flex shrink-0 items-center gap-2.5">
            <ToolCallCardHeaderChips />
            <ToolCallStatusIcon status={status} />
            {actions ? (
              <div data-slot="tool-call-card-actions" className="flex items-center gap-1.5">
                {actions}
              </div>
            ) : null}
          </div>
        </header>
        {/* Body div always mounts so slotChildren keep a stable JSX position
            across `showBody` flips — React reconciles `<CodeBlock>` and other
            async children instead of unmount/remount. Chrome classes apply
            only when there's something visible; `data-empty` is the test hook. */}
        <div
          data-slot="tool-call-card-body"
          data-empty={showBody ? undefined : "true"}
          className={cn("flex flex-col", showBody ? "gap-2.5 border-t border-line px-4 py-3" : "")}
        >
          {hasError ? (
            <p data-slot="tool-call-card-error" className="text-form-input text-danger">
              {errorMessage}
            </p>
          ) : null}
          {hasRawChildren ? rawChildren : null}
          {slotChildren}
        </div>
      </section>
    </ToolCallCardContext.Provider>
  );
}

/**
 * Inline tool-execution card per DESIGN.md §4 "Tool Call Card". Surface bg with
 * a 1px divider border between header and body, terminal icon + tool name +
 * optional file path, Input/Output disclosure chips + signal-toned status icon
 * + optional actions pinned right. Collapsed cards are a single header row.
 *
 * Compose `<ToolCallCard.Input>` and `<ToolCallCard.Output>` as children for
 * collapsible argument/result regions (closed by default), or pass raw children
 * for diffs, stdout, or any other body content. Markdown sources render through
 * the canonical `<Markdown />` primitive — same XSS-safe contract everywhere.
 *
 * Slot children render in their declared JSX position (not from context state)
 * so async children like `<CodeBlock>` keep their shiki highlight across parent
 * re-renders.
 */
const ToolCallCard = Object.assign(ToolCallCardInner, {
  Input: ToolCallCardInput,
  Output: ToolCallCardOutput,
});

export { ToolCallCard, STATUS_LABEL as TOOL_CALL_STATUS_LABEL };
