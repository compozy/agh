import { useCallback, useMemo, useState } from "react";
import { Activity, ChevronRight, FileCode, Gauge, Library, PanelRightOpen } from "lucide-react";
import type { AssistantState } from "@assistant-ui/react";

import {
  Button,
  Empty,
  Metric,
  Pill,
  ScrollArea,
  Sheet,
  SheetContent,
  SheetTrigger,
  Tabs,
  TabsList,
  TabsTrigger,
  cn,
  type PillTone,
} from "@agh/ui";

import { isAgentEventPayload, parseToolUseResult } from "../lib/message-parts";

type ThreadMessageState = AssistantState["thread"]["messages"][number];

export type InspectorTraceKind =
  | "start"
  | "user"
  | "agent"
  | "tool"
  | "diff"
  | "system"
  | "approval";

export type InspectorTraceStatus = "ok" | "warn" | "error" | "pending";

export interface InspectorTraceEvent {
  id: string;
  kind: InspectorTraceKind;
  label: string;
  timestamp: number;
  status: InspectorTraceStatus;
}

export interface InspectorUsage {
  tokensIn?: number;
  tokensOut?: number;
  costUsd?: number;
  /** Rate shown as a mono detail (e.g., tokens per second). */
  ratePerSecond?: number;
  /** Delta vs previous turn — positive green, negative red, zero neutral. */
  tokensInDelta?: number;
  tokensOutDelta?: number;
  costDelta?: number;
}

export interface InspectorMemoryDoc {
  id: string;
  kind: string;
  title: string;
  bytes?: number;
}

export interface InspectorFileEntry {
  path: string;
  readCount: number;
}

export interface SessionInspectorProps {
  messages: readonly ThreadMessageState[];
  usage?: InspectorUsage | null;
  memoryDocs?: InspectorMemoryDoc[];
  /** Explicit file list. When omitted, derived from `messages` via `deriveFileReads`. */
  files?: InspectorFileEntry[];
  /** Total trace event count — when greater than `traceLimit`, renders a "View all" link. */
  totalTraceEvents?: number;
  /** Number of latest trace events to render in the Trace section. Defaults to 6. */
  traceLimit?: number;
  onViewAllTrace?: () => void;
  className?: string;
}

const TRACE_LIMIT_DEFAULT = 6;
const INSPECTOR_WIDTH = 320;
const SECTION_LABELS = {
  trace: "Trace",
  usage: "Usage",
  memory: "Memory",
  files: "Files",
} as const;

type TopTab = "trace" | "usage";
type BottomTab = "memory" | "files";

const TRACE_STATUS_TONE: Record<InspectorTraceStatus, PillTone> = {
  ok: "success",
  warn: "warning",
  error: "danger",
  pending: "accent",
};

const TRACE_KIND_LABEL: Record<InspectorTraceKind, string> = {
  start: "START",
  user: "USER",
  agent: "AGENT",
  tool: "TOOL",
  diff: "DIFF",
  system: "SYSTEM",
  approval: "APPROVAL",
};

function traceKindFromRole(role: ThreadMessageState["role"]): InspectorTraceKind {
  switch (role) {
    case "user":
      return "user";
    case "assistant":
      return "agent";
    case "system":
      return "system";
  }

  const _exhaustive: never = role;
  return _exhaustive;
}

function traceStatusFromMessage(message: ThreadMessageState): InspectorTraceStatus {
  if (message.role !== "assistant") {
    return "ok";
  }

  if (message.status?.type === "running" || message.status?.type === "requires-action") {
    return "pending";
  }

  if (message.status?.type === "incomplete") {
    return message.status.reason === "error" ? "error" : "warn";
  }

  return "ok";
}

function toolStatusFromPart(part: ThreadMessageState["parts"][number]): InspectorTraceStatus {
  if (part.type !== "tool-call") {
    return "ok";
  }

  if (part.isError || (part.status.type === "incomplete" && part.status.reason === "error")) {
    return "error";
  }

  if (part.status.type === "running" || part.status.type === "requires-action") {
    return "pending";
  }

  if (part.status.type === "incomplete") {
    return "warn";
  }

  return "ok";
}

function getTextPartText(message: ThreadMessageState): string {
  return message.content
    .filter(
      (
        part
      ): part is Extract<ThreadMessageState["content"][number], { type: "text" | "reasoning" }> =>
        part.type === "text" || part.type === "reasoning"
    )
    .map(part => part.text)
    .join("");
}

function traceLabelFromMessage(message: ThreadMessageState): string {
  if (message.role === "system") {
    const first = getTextPartText(message).split("\n")[0] ?? "";
    return first || "system event";
  }

  if (message.role === "user") {
    return "Prompt sent";
  }

  return "Agent response";
}

/**
 * Map the current thread messages into trace rows for the Inspector.
 * Pure — no hooks. The first message is tagged as `start` so the session-resume
 * line reads like the mock. The last `limit` events are returned.
 */
export function deriveTraceEvents(
  messages: readonly ThreadMessageState[],
  limit = TRACE_LIMIT_DEFAULT
): InspectorTraceEvent[] {
  if (messages.length === 0) {
    return [];
  }

  const events: InspectorTraceEvent[] = [];

  const firstTimestamp = messages[0]?.createdAt.getTime() ?? Date.now();
  events.push({
    id: `start-${messages[0]?.id ?? "session"}`,
    kind: "start",
    label: "Session started",
    timestamp: firstTimestamp,
    status: "ok",
  });

  for (const message of messages) {
    const timestamp = message.createdAt.getTime();

    if (message.role === "user" || message.role === "system") {
      events.push({
        id: message.id,
        kind: traceKindFromRole(message.role),
        label: traceLabelFromMessage(message),
        timestamp,
        status: traceStatusFromMessage(message),
      });
      continue;
    }

    const hasAssistantNarration = message.content.some(
      part => part.type === "text" || part.type === "reasoning"
    );

    if (hasAssistantNarration) {
      events.push({
        id: message.id,
        kind: "agent",
        label: traceLabelFromMessage(message),
        timestamp,
        status: traceStatusFromMessage(message),
      });
    }

    for (const part of message.parts) {
      if (part.type === "tool-call") {
        events.push({
          id: part.toolCallId,
          kind: "tool",
          label: part.toolName || "tool call",
          timestamp,
          status: toolStatusFromPart(part),
        });
      }

      if (part.type === "data" && part.name === "agh-permission") {
        const raw = part.data as { title?: string; decision?: string } | undefined;
        events.push({
          id: `${message.id}-${part.name}`,
          kind: "approval",
          label: raw?.title || "Permission required",
          timestamp,
          status: raw?.decision ? "ok" : "pending",
        });
      }
    }
  }

  return events.slice(-limit);
}

/**
 * Aggregate tool messages by their file path into a file-read summary. Pulls
 * `toolResult.filePath` first, then falls back to known input fields
 * (`file_path` / `filePath` / `path`). Order is preserved by first appearance.
 */
export function deriveFileReads(messages: readonly ThreadMessageState[]): InspectorFileEntry[] {
  const index = new Map<string, InspectorFileEntry>();
  for (const message of messages) {
    if (message.role !== "assistant") {
      continue;
    }

    for (const part of message.parts) {
      if (part.type !== "tool-call") {
        continue;
      }

      const result = isAgentEventPayload(part.result) ? parseToolUseResult(part.result) : null;
      const path = result?.filePath ?? readFilePathFromInput(part.args);
      if (!path) {
        continue;
      }

      const existing = index.get(path);
      if (existing) {
        existing.readCount += 1;
      } else {
        index.set(path, { path, readCount: 1 });
      }
    }
  }
  return Array.from(index.values());
}

function readFilePathFromInput(input: Record<string, unknown> | undefined): string | undefined {
  if (!input) {
    return undefined;
  }
  const raw = input.file_path ?? input.filePath ?? input.path;
  return typeof raw === "string" && raw.length > 0 ? raw : undefined;
}

function formatTimestamp(ts: number): string {
  if (!Number.isFinite(ts) || ts <= 0) return "";
  return new Date(ts).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}

function formatNumber(value?: number): string {
  if (typeof value !== "number" || !Number.isFinite(value)) return "—";
  return value.toLocaleString();
}

function formatCost(value?: number): string {
  if (typeof value !== "number" || !Number.isFinite(value)) return "—";
  const abs = Math.abs(value);
  if (abs < 1) return `$${value.toFixed(3)}`;
  return `$${value.toFixed(2)}`;
}

function formatBytes(bytes?: number): string {
  if (typeof bytes !== "number" || !Number.isFinite(bytes)) return "—";
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} kB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function deltaTone(delta?: number): "default" | "success" | "danger" {
  if (typeof delta !== "number" || !Number.isFinite(delta) || delta === 0) return "default";
  return delta > 0 ? "success" : "danger";
}

function deltaLabel(delta?: number): string | undefined {
  if (typeof delta !== "number" || !Number.isFinite(delta) || delta === 0) return undefined;
  const prefix = delta > 0 ? "+" : "−";
  return `${prefix}${Math.abs(delta).toLocaleString()}`;
}

interface SectionBodyProps {
  traceEvents: InspectorTraceEvent[];
  traceTotal: number;
  traceLimit: number;
  onViewAllTrace?: () => void;
  usage: InspectorUsage | null | undefined;
  memoryDocs: InspectorMemoryDoc[];
  files: InspectorFileEntry[];
}

/**
 * Inner inspector body — two stacked tabbed groups sharing the column 50/50.
 * Top row switches between Trace and Usage; bottom row switches between
 * Memory and Files. Backs both the fixed 320px `SessionInspector` column and
 * the `SessionInspectorDrawer` Sheet.
 */
function InspectorBody({
  traceEvents,
  traceTotal,
  traceLimit,
  onViewAllTrace,
  usage,
  memoryDocs,
  files,
}: SectionBodyProps) {
  const [topTab, setTopTab] = useState<TopTab>("trace");
  const [bottomTab, setBottomTab] = useState<BottomTab>("memory");
  const handleTopChange = useCallback((value: string | null | undefined) => {
    if (value === "trace" || value === "usage") setTopTab(value);
  }, []);
  const handleBottomChange = useCallback((value: string | null | undefined) => {
    if (value === "memory" || value === "files") setBottomTab(value);
  }, []);

  return (
    <div data-testid="session-inspector-body" className="flex min-h-0 flex-1 flex-col">
      <Tabs
        aria-label="Trace and usage"
        value={topTab}
        onValueChange={handleTopChange}
        className="flex min-h-0 flex-1 basis-0 flex-col gap-0"
      >
        <TabsList
          variant="line"
          className="w-full shrink-0 border-b border-[color:var(--color-divider)] px-2 group-data-horizontal/tabs:h-12"
        >
          <TabsTrigger
            value="trace"
            data-testid="session-inspector-tab-trace"
            className="h-12 gap-2 group-data-horizontal/tabs:after:bottom-[-1px]"
          >
            <Activity className="size-3.5" />
            <span>{SECTION_LABELS.trace}</span>
          </TabsTrigger>
          <TabsTrigger
            value="usage"
            data-testid="session-inspector-tab-usage"
            className="h-12 gap-2 group-data-horizontal/tabs:after:bottom-[-1px]"
          >
            <Gauge className="size-3.5" />
            <span>{SECTION_LABELS.usage}</span>
          </TabsTrigger>
        </TabsList>
        <ScrollArea className="flex-1 min-h-0">
          <div
            className="flex min-h-full flex-col gap-4 px-4 py-4"
            data-testid="session-inspector-top-panel"
            data-active-tab={topTab}
          >
            {topTab === "trace" && (
              <TraceSection
                events={traceEvents}
                total={traceTotal}
                limit={traceLimit}
                onViewAll={onViewAllTrace}
              />
            )}
            {topTab === "usage" && <UsageSection usage={usage} />}
          </div>
        </ScrollArea>
      </Tabs>
      <Tabs
        aria-label="Memory and files"
        value={bottomTab}
        onValueChange={handleBottomChange}
        className="flex min-h-0 flex-1 basis-0 flex-col gap-0 border-t border-[color:var(--color-divider)]"
      >
        <TabsList
          variant="line"
          className="w-full shrink-0 border-b border-[color:var(--color-divider)] px-2 group-data-horizontal/tabs:h-12"
        >
          <TabsTrigger
            value="memory"
            data-testid="session-inspector-tab-memory"
            className="h-12 gap-2 group-data-horizontal/tabs:after:bottom-[-1px]"
          >
            <Library className="size-3.5" />
            <span>{SECTION_LABELS.memory}</span>
          </TabsTrigger>
          <TabsTrigger
            value="files"
            data-testid="session-inspector-tab-files"
            className="h-12 gap-2 group-data-horizontal/tabs:after:bottom-[-1px]"
          >
            <FileCode className="size-3.5" />
            <span>{SECTION_LABELS.files}</span>
          </TabsTrigger>
        </TabsList>
        <ScrollArea className="flex-1 min-h-0">
          <div
            className="flex min-h-full flex-col gap-4 px-4 py-4"
            data-testid="session-inspector-bottom-panel"
            data-active-tab={bottomTab}
          >
            {bottomTab === "memory" && <MemorySection docs={memoryDocs} />}
            {bottomTab === "files" && <FilesSection files={files} />}
          </div>
        </ScrollArea>
      </Tabs>
    </div>
  );
}

/**
 * Right-hand 320px session inspector. Composes `Tabs` / `Metric` /
 * `MonoBadge` / `StatusDot` / `ScrollArea` / `Empty` from `@agh/ui`.
 * Hidden on viewports narrower than 1200px — pair with `SessionInspectorDrawer`
 * to expose the same body inside a Sheet on compact viewports.
 */
export function SessionInspector({
  messages,
  usage,
  memoryDocs = [],
  files,
  totalTraceEvents,
  traceLimit = TRACE_LIMIT_DEFAULT,
  onViewAllTrace,
  className,
}: SessionInspectorProps) {
  const traceEvents = useMemo(
    () => deriveTraceEvents(messages, traceLimit),
    [messages, traceLimit]
  );
  const derivedFiles = useMemo(() => files ?? deriveFileReads(messages), [files, messages]);
  const traceTotal = totalTraceEvents ?? messages.length;

  return (
    <aside
      data-testid="session-inspector"
      aria-label="Session inspector"
      style={{ width: INSPECTOR_WIDTH }}
      className={cn(
        "hidden shrink-0 flex-col overflow-hidden border-l bg-[color:var(--color-canvas)]",
        "border-[color:var(--color-divider)] min-w-0",
        "xl:flex",
        className
      )}
    >
      <InspectorBody
        traceEvents={traceEvents}
        traceTotal={traceTotal}
        traceLimit={traceLimit}
        onViewAllTrace={onViewAllTrace}
        usage={usage}
        memoryDocs={memoryDocs}
        files={derivedFiles}
      />
    </aside>
  );
}

interface TraceSectionProps {
  events: InspectorTraceEvent[];
  total: number;
  limit: number;
  onViewAll?: () => void;
}

function TraceSection({ events, total, limit, onViewAll }: TraceSectionProps) {
  const hasOverflow = total > limit;
  return (
    <div data-testid="session-inspector-trace" className="flex min-h-full flex-col">
      {events.length === 0 ? (
        <Empty
          icon={Activity}
          title="No trace events yet"
          description="Trace rows appear as the agent sends prompts, runs tools, and receives responses."
          data-testid="session-inspector-trace-empty"
        />
      ) : (
        <ol data-testid="session-inspector-trace-list" className="flex flex-col gap-3">
          {events.map(event => (
            <TraceRow key={event.id} event={event} />
          ))}
        </ol>
      )}
      {hasOverflow && onViewAll ? (
        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={onViewAll}
          data-testid="session-inspector-trace-view-all"
          className="mt-3 h-7 gap-1 self-start px-1 text-[color:var(--color-text-secondary)] hover:text-[color:var(--color-text-primary)]"
        >
          View all
          <ChevronRight className="size-3" />
        </Button>
      ) : null}
    </div>
  );
}

function TraceRow({ event }: { event: InspectorTraceEvent }) {
  const tone = TRACE_STATUS_TONE[event.status];
  const pulse = event.status === "pending";
  const ts = formatTimestamp(event.timestamp);
  return (
    <li
      data-testid="session-inspector-trace-row"
      data-kind={event.kind}
      data-status={event.status}
      className="flex items-start gap-2"
    >
      <Pill.Dot
        tone={tone}
        size="md"
        pulse={pulse}
        className="mt-1 shrink-0"
        data-testid="session-inspector-trace-dot"
      />
      <div className="flex min-w-0 flex-1 items-center gap-2">
        <span
          data-testid="session-inspector-trace-timestamp"
          className="shrink-0 font-mono text-[10px] uppercase tracking-[0.06em] text-[color:var(--color-text-tertiary)]"
        >
          {ts}
        </span>
        <Pill
          mono
          tone={tone === "danger" ? "danger" : tone === "warning" ? "warning" : "neutral"}
          className="shrink-0"
          data-testid="session-inspector-trace-kind"
        >
          {TRACE_KIND_LABEL[event.kind]}
        </Pill>
        <span
          data-testid="session-inspector-trace-label"
          className="min-w-0 flex-1 truncate text-[12.5px] text-[color:var(--color-text-primary)]"
        >
          {event.label}
        </span>
      </div>
    </li>
  );
}

interface UsageSectionProps {
  usage: InspectorUsage | null | undefined;
}

function UsageSection({ usage }: UsageSectionProps) {
  const hasUsage =
    usage !== null &&
    usage !== undefined &&
    (usage.tokensIn !== undefined ||
      usage.tokensOut !== undefined ||
      usage.costUsd !== undefined ||
      usage.ratePerSecond !== undefined);

  return (
    <div data-testid="session-inspector-usage" className="flex min-h-full flex-col">
      {hasUsage ? (
        <div data-testid="session-inspector-usage-grid" className="grid grid-cols-2 gap-2">
          <Metric
            label="Tokens in"
            value={formatNumber(usage?.tokensIn)}
            tone={deltaTone(usage?.tokensInDelta)}
            detail={deltaLabel(usage?.tokensInDelta)}
            data-testid="session-inspector-usage-tokens-in"
            className="px-3 py-3"
          />
          <Metric
            label="Tokens out"
            value={formatNumber(usage?.tokensOut)}
            tone={deltaTone(usage?.tokensOutDelta)}
            detail={deltaLabel(usage?.tokensOutDelta)}
            data-testid="session-inspector-usage-tokens-out"
            className="px-3 py-3"
          />
          <Metric
            label="Total cost"
            value={formatCost(usage?.costUsd)}
            tone={deltaTone(usage?.costDelta)}
            detail={deltaLabel(usage?.costDelta)}
            data-testid="session-inspector-usage-cost"
            className="px-3 py-3"
          />
          <Metric
            label="Est. rate"
            value={
              typeof usage?.ratePerSecond === "number" && Number.isFinite(usage.ratePerSecond)
                ? `${usage.ratePerSecond.toFixed(1)}/s`
                : "—"
            }
            data-testid="session-inspector-usage-rate"
            className="px-3 py-3"
          />
        </div>
      ) : (
        <Empty
          icon={Gauge}
          title="No usage yet"
          description="Token counts and cost land here once the agent completes its first turn."
          data-testid="session-inspector-usage-empty"
        />
      )}
    </div>
  );
}

interface MemorySectionProps {
  docs: InspectorMemoryDoc[];
}

function MemorySection({ docs }: MemorySectionProps) {
  return (
    <div data-testid="session-inspector-memory" className="flex min-h-full flex-col">
      {docs.length === 0 ? (
        <Empty
          icon={Library}
          title="No memory loaded"
          description="Workspace and repository memory docs appear here when they're attached to the session."
          data-testid="session-inspector-memory-empty"
        />
      ) : (
        <ul
          data-testid="session-inspector-memory-list"
          className="flex flex-col divide-y divide-[color:var(--color-divider)]"
        >
          {docs.map(doc => (
            <li
              key={doc.id}
              data-testid="session-inspector-memory-row"
              className="flex items-center gap-2 py-2"
            >
              <Pill mono tone="info" data-testid="session-inspector-memory-kind">
                {doc.kind}
              </Pill>
              <span
                className="min-w-0 flex-1 truncate font-mono text-[11.5px] text-[color:var(--color-text-primary)]"
                data-testid="session-inspector-memory-title"
              >
                {doc.title}
              </span>
              <span
                className="shrink-0 font-mono text-[10px] text-[color:var(--color-text-tertiary)]"
                data-testid="session-inspector-memory-bytes"
              >
                {formatBytes(doc.bytes)}
              </span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

interface FilesSectionProps {
  files: InspectorFileEntry[];
}

function FilesSection({ files }: FilesSectionProps) {
  return (
    <div data-testid="session-inspector-files" className="flex min-h-full flex-col">
      {files.length === 0 ? (
        <Empty
          icon={FileCode}
          title="No files read"
          description="Files the agent reads during this session appear here."
          data-testid="session-inspector-files-empty"
        />
      ) : (
        <ScrollArea
          data-testid="session-inspector-files-scroll"
          className="max-h-[240px] rounded-[var(--radius-md)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)]"
        >
          <ul
            data-testid="session-inspector-files-list"
            className="flex flex-col divide-y divide-[color:var(--color-divider)]"
          >
            {files.map(file => (
              <li
                key={file.path}
                data-testid="session-inspector-files-row"
                className="flex items-center gap-2 px-2 py-1.5"
              >
                <FileCode
                  aria-hidden="true"
                  className="size-3 shrink-0 text-[color:var(--color-text-tertiary)]"
                />
                <span
                  data-testid="session-inspector-files-path"
                  className="min-w-0 flex-1 truncate font-mono text-[11.5px] text-[color:var(--color-text-primary)]"
                >
                  {file.path}
                </span>
                <span
                  data-testid="session-inspector-files-count"
                  className="shrink-0 font-mono text-[10px] text-[color:var(--color-text-tertiary)]"
                >
                  ×{file.readCount}
                </span>
              </li>
            ))}
          </ul>
        </ScrollArea>
      )}
    </div>
  );
}

/**
 * Drawer wrapper around the inspector body for narrow viewports (<1200px).
 * Renders a ghost icon trigger that's only visible on narrow viewports (paired
 * with the inline `SessionInspector` which hides below `xl`). Opening the
 * trigger mounts the same `InspectorBody` inside a right-anchored `Sheet`.
 */
export function SessionInspectorDrawer({
  messages,
  usage,
  memoryDocs = [],
  files,
  totalTraceEvents,
  traceLimit = TRACE_LIMIT_DEFAULT,
  onViewAllTrace,
}: SessionInspectorProps) {
  const traceEvents = useMemo(
    () => deriveTraceEvents(messages, traceLimit),
    [messages, traceLimit]
  );
  const derivedFiles = useMemo(() => files ?? deriveFileReads(messages), [files, messages]);
  const traceTotal = totalTraceEvents ?? messages.length;

  return (
    <Sheet>
      <SheetTrigger
        render={
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            aria-label="Open session inspector"
            data-testid="session-inspector-drawer-trigger"
            className="inline-flex xl:hidden"
          />
        }
      >
        <PanelRightOpen aria-hidden="true" className="size-3.5" />
      </SheetTrigger>
      <SheetContent
        side="right"
        data-testid="session-inspector-drawer"
        className="flex w-[min(88vw,360px)] max-w-[360px] flex-col gap-0 bg-[color:var(--color-canvas)] p-0 sm:max-w-[360px]"
      >
        <header className="flex h-12 shrink-0 items-center justify-between border-b border-[color:var(--color-divider)] px-4">
          <span className="font-mono text-[11px] font-semibold uppercase tracking-[0.06em] text-[color:var(--color-text-label)]">
            Inspector
          </span>
        </header>
        <InspectorBody
          traceEvents={traceEvents}
          traceTotal={traceTotal}
          traceLimit={traceLimit}
          onViewAllTrace={onViewAllTrace}
          usage={usage}
          memoryDocs={memoryDocs}
          files={derivedFiles}
        />
      </SheetContent>
    </Sheet>
  );
}
