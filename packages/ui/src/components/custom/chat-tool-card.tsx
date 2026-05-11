"use client";

import { ChevronRight } from "lucide-react";
import * as React from "react";
import { Streamdown } from "streamdown";

import { cn } from "../../lib/utils";
import { STREAMDOWN_SAFE_CONFIG } from "./description-card";
import { Eyebrow } from "./eyebrow";
import { MonoId } from "./mono-id";
import { Pill, type PillTone } from "./pill";
import { Time } from "./time";

export type ChatToolStatus = "pending" | "in_progress" | "completed" | "failed";

const TOOL_STATUS_TONE: Record<ChatToolStatus, PillTone> = {
  pending: "neutral",
  in_progress: "info",
  completed: "success",
  failed: "danger",
};

const TOOL_STATUS_LABEL: Record<ChatToolStatus, string> = {
  pending: "Pending",
  in_progress: "Running",
  completed: "Success",
  failed: "Failed",
};

/** Threshold beyond which a string output collapses by default. */
export const CHAT_TOOL_OUTPUT_COLLAPSE_LINES = 200;

export interface ChatToolCardSection {
  /** Markdown / JSON / plain-text source. Strings are rendered through the safe markdown contract. */
  source?: string;
  /** Pre-rendered ReactNode. Takes precedence over `source`. */
  node?: React.ReactNode;
  /** When `source` is set, choose how to render: markdown (default) or as a mono `<pre>` block. */
  format?: "markdown" | "code";
}

export interface ChatToolCardProps extends Omit<React.ComponentProps<"section">, "title"> {
  /** Tool name rendered as a `<MonoId>` in the head row. */
  toolName: string;
  /** Lifecycle status; resolves the pill tone + label. */
  status: ChatToolStatus;
  /** Optional ISO timestamp rendered through `<Time>` in the head row. */
  timestamp?: string;
  /** Optional inputs region (collapsible). Default header label `"Input"`. */
  input?: ChatToolCardSection;
  /** Optional outputs region (collapsible). Default header label `"Output"`. */
  output?: ChatToolCardSection;
  /** Override the collapse threshold for string outputs. Default 200 lines. */
  outputCollapseThreshold?: number;
  /** Force the input region open/collapsed. */
  initialInputCollapsed?: boolean;
  /** Force the output region open/collapsed. Defaults to true when output exceeds the threshold. */
  initialOutputCollapsed?: boolean;
  /** Inline failure message rendered above the output region when `status === "failed"`. */
  errorMessage?: React.ReactNode;
  /** Trailing action slot (retry / cancel / view full transcript). */
  actions?: React.ReactNode;
}

function countLines(source: string): number {
  if (!source) return 0;
  let count = 1;
  for (let index = 0; index < source.length; index += 1) {
    if (source.charCodeAt(index) === 10) count += 1;
  }
  return count;
}

function sectionShouldCollapseByDefault(
  section: ChatToolCardSection | undefined,
  threshold: number
): boolean {
  if (!section) return false;
  if (typeof section.source !== "string") return false;
  return countLines(section.source) > threshold;
}

function ChatToolSection({
  label,
  section,
  defaultOpen,
  slot,
}: {
  label: string;
  section: ChatToolCardSection;
  defaultOpen: boolean;
  slot: string;
}) {
  const [open, setOpen] = React.useState(defaultOpen);
  const id = React.useId();
  return (
    <div data-slot={slot} data-open={open ? "true" : "false"} className="flex flex-col">
      <button
        type="button"
        data-slot={`${slot}-toggle`}
        aria-expanded={open}
        aria-controls={id}
        onClick={() => setOpen(value => !value)}
        className="inline-flex w-full items-center gap-1.5 rounded-(--radius-xs) px-1 py-1 text-left transition-colors duration-(--dur) ease-(--ease) hover:bg-(--hover) focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-(--line-strong)"
      >
        <ChevronRight
          width={12}
          height={12}
          strokeWidth={1.75}
          className={cn(
            "shrink-0 text-(--muted) transition-transform duration-(--dur) ease-(--ease)",
            open && "rotate-90"
          )}
        />
        <Eyebrow className="text-(--muted)">{label}</Eyebrow>
      </button>
      {open ? (
        <div id={id} data-slot={`${slot}-body`} className="mt-1 pl-[18px]">
          <ChatToolSectionBody section={section} />
        </div>
      ) : null}
    </div>
  );
}

function ChatToolSectionBody({ section }: { section: ChatToolCardSection }) {
  if (section.node !== undefined && section.node !== null) {
    return <>{section.node}</>;
  }
  const source = section.source ?? "";
  if (section.format === "code") {
    return (
      <pre className="overflow-x-auto rounded-(--radius) bg-(--canvas) p-3 font-mono text-[12px] text-(--fg)">
        <code>{source}</code>
      </pre>
    );
  }
  return (
    <div
      data-slot="chat-tool-card-markdown"
      className="text-(length:--text-body) leading-[1.6] text-(--fg) [&_code]:font-mono [&_code]:text-[12px] [&_p]:my-2"
    >
      <Streamdown {...STREAMDOWN_SAFE_CONFIG}>{source}</Streamdown>
    </div>
  );
}

function ChatToolCard({
  toolName,
  status,
  timestamp,
  input,
  output,
  outputCollapseThreshold = CHAT_TOOL_OUTPUT_COLLAPSE_LINES,
  initialInputCollapsed,
  initialOutputCollapsed,
  errorMessage,
  actions,
  className,
  ...props
}: ChatToolCardProps) {
  const tone = TOOL_STATUS_TONE[status];
  const statusLabel = TOOL_STATUS_LABEL[status];
  const outputCollapsedDefault =
    initialOutputCollapsed ?? sectionShouldCollapseByDefault(output, outputCollapseThreshold);
  const inputCollapsedDefault = initialInputCollapsed ?? false;
  const failureTint = status === "failed";
  return (
    <section
      data-slot="chat-tool-card"
      data-status={status}
      className={cn(
        "flex flex-col gap-2 rounded-lg px-3 py-2.5",
        failureTint ? "bg-(--danger-tint)" : "bg-(--canvas-soft)",
        className
      )}
      {...props}
    >
      <header data-slot="chat-tool-card-head" className="flex min-w-0 flex-wrap items-center gap-2">
        <MonoId data-slot="chat-tool-card-name" value={toolName} />
        <Pill data-slot="chat-tool-card-status" tone={tone}>
          {statusLabel}
        </Pill>
        {timestamp ? (
          <Time
            data-slot="chat-tool-card-time"
            iso={timestamp}
            mode="relative"
            className="text-[12px] text-(--muted)"
          />
        ) : null}
        {actions ? (
          <div data-slot="chat-tool-card-actions" className="ml-auto flex items-center gap-1.5">
            {actions}
          </div>
        ) : null}
      </header>
      {errorMessage ? (
        <p data-slot="chat-tool-card-error" className="text-[12.5px] text-(--danger)">
          {errorMessage}
        </p>
      ) : null}
      {input ? (
        <ChatToolSection
          label="Input"
          section={input}
          defaultOpen={!inputCollapsedDefault}
          slot="chat-tool-card-input"
        />
      ) : null}
      {output ? (
        <ChatToolSection
          label="Output"
          section={output}
          defaultOpen={!outputCollapsedDefault}
          slot="chat-tool-card-output"
        />
      ) : null}
    </section>
  );
}

export { ChatToolCard, TOOL_STATUS_TONE, TOOL_STATUS_LABEL };
