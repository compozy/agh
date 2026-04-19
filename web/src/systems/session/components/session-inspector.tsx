import { useCallback, useMemo, useState } from "react";
import { Activity, ChevronRight, FileCode, Gauge, Library, PanelRightOpen } from "lucide-react";

import {
  Button,
  Empty,
  Metric,
  MonoBadge,
  ScrollArea,
  Section,
  Sheet,
  SheetContent,
  SheetTrigger,
  StatusDot,
  Tabs,
  TabsList,
  TabsTrigger,
  cn,
  type StatusDotTone,
} from "@agh/ui";

import type { UIMessage } from "../types";

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
  messages: UIMessage[];
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

type InspectorTab = keyof typeof SECTION_LABELS;

const TRACE_STATUS_TONE: Record<InspectorTraceStatus, StatusDotTone> = {
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

function traceKindFromRole(role: UIMessage["role"]): InspectorTraceKind {
  switch (role) {
    case "user":
      return "user";
    case "assistant":
      return "agent";
    case "tool_call":
    case "tool_result":
      return "tool";
    case "diff":
      return "diff";
    case "system":
      return "system";
  }
}

function traceStatusFromMessage(msg: UIMessage): InspectorTraceStatus {
  if (msg.toolError) return "error";
  if (msg.isStreaming) return "pending";
  if (msg.role === "tool_call" && !msg.toolResult) return "pending";
  return "ok";
}

function traceLabelFromMessage(msg: UIMessage): string {
  if (msg.role === "tool_call" || msg.role === "tool_result") {
    return msg.toolName ? msg.toolName : "tool call";
  }
  if (msg.role === "diff") {
    return msg.diff?.path ?? "diff";
  }
  if (msg.role === "system") {
    const first = msg.content.split("\n")[0] ?? "";
    return first || "system event";
  }
  if (msg.role === "user") return "Prompt sent";
  return "Agent response";
}

/**
 * Map a transcript/store UIMessage[] into trace rows for the Inspector.
 * Pure — no hooks. The first message is tagged as `start` so the session-resume
 * line reads like the mock. The last `limit` events are returned.
 */
export function deriveTraceEvents(
  messages: UIMessage[],
  limit = TRACE_LIMIT_DEFAULT
): InspectorTraceEvent[] {
  if (messages.length === 0) return [];
  const events = messages.map<InspectorTraceEvent>((msg, index) => ({
    id: msg.id ?? `trace-${index}`,
    kind: index === 0 ? "start" : traceKindFromRole(msg.role),
    label: index === 0 ? "Session started" : traceLabelFromMessage(msg),
    timestamp: msg.timestamp,
    status: traceStatusFromMessage(msg),
  }));
  return events.slice(-limit);
}

/**
 * Aggregate tool messages by their file path into a file-read summary. Pulls
 * `toolResult.filePath` first, then falls back to known input fields
 * (`file_path` / `filePath` / `path`). Order is preserved by first appearance.
 */
export function deriveFileReads(messages: UIMessage[]): InspectorFileEntry[] {
  const index = new Map<string, InspectorFileEntry>();
  for (const msg of messages) {
    const path = msg.toolResult?.filePath ?? readFilePathFromInput(msg);
    if (!path) continue;
    const existing = index.get(path);
    if (existing) {
      existing.readCount += 1;
    } else {
      index.set(path, { path, readCount: 1 });
    }
  }
  return Array.from(index.values());
}

function readFilePathFromInput(msg: UIMessage): string | undefined {
  if (msg.role !== "tool_call" && msg.role !== "tool_result") return undefined;
  const input = msg.toolInput;
  if (!input) return undefined;
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

const INSPECTOR_CSS = `
[data-session-inspector-body] [data-session-inspector-stacked] { display: flex; }
[data-session-inspector-body] [data-session-inspector-tabbed] { display: none; }
@media (max-height: 680px) {
  [data-session-inspector-body] [data-session-inspector-stacked] { display: none; }
  [data-session-inspector-body] [data-session-inspector-tabbed] { display: flex; }
}
`;

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
 * Inner inspector body — stacked + tabbed layouts with CSS media-query swap.
 * Layout-agnostic so it can live inside the fixed 320px `SessionInspector`
 * column or inside the `SessionInspectorDrawer` Sheet body.
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
  return (
    <div
      data-session-inspector-body
      data-testid="session-inspector-body"
      className="flex min-h-0 flex-1 flex-col"
    >
      <style>{INSPECTOR_CSS}</style>
      <div
        data-session-inspector-stacked
        data-testid="session-inspector-stacked"
        className="min-h-0 flex-1 flex-col"
      >
        <ScrollArea className="flex-1 min-h-0">
          <div className="flex flex-col gap-5 px-4 py-4">
            <TraceSection
              events={traceEvents}
              total={traceTotal}
              limit={traceLimit}
              onViewAll={onViewAllTrace}
            />
            <UsageSection usage={usage} />
            <MemorySection docs={memoryDocs} />
            <FilesSection files={files} />
          </div>
        </ScrollArea>
      </div>
      <TabbedBody
        traceEvents={traceEvents}
        traceTotal={traceTotal}
        traceLimit={traceLimit}
        onViewAllTrace={onViewAllTrace}
        usage={usage}
        memoryDocs={memoryDocs}
        files={files}
      />
    </div>
  );
}

function TabbedBody(props: SectionBodyProps) {
  const [active, setActive] = useState<InspectorTab>("trace");
  const handleChange = useCallback((value: string | null | undefined) => {
    if (value === "trace" || value === "usage" || value === "memory" || value === "files") {
      setActive(value);
    }
  }, []);

  return (
    <div
      data-session-inspector-tabbed
      data-testid="session-inspector-tabbed"
      className="min-h-0 flex-1 flex-col"
    >
      <Tabs
        aria-label="Session inspector tabs"
        value={active}
        onValueChange={handleChange}
        className="flex min-h-0 flex-1 flex-col gap-0"
      >
        <TabsList variant="line" className="h-10 border-b border-[color:var(--color-divider)] px-2">
          <TabsTrigger value="trace" data-testid="session-inspector-tab-trace" className="gap-2">
            <Activity className="size-3.5" />
            <span>{SECTION_LABELS.trace}</span>
          </TabsTrigger>
          <TabsTrigger value="usage" data-testid="session-inspector-tab-usage" className="gap-2">
            <Gauge className="size-3.5" />
            <span>{SECTION_LABELS.usage}</span>
          </TabsTrigger>
          <TabsTrigger value="memory" data-testid="session-inspector-tab-memory" className="gap-2">
            <Library className="size-3.5" />
            <span>{SECTION_LABELS.memory}</span>
          </TabsTrigger>
          <TabsTrigger value="files" data-testid="session-inspector-tab-files" className="gap-2">
            <FileCode className="size-3.5" />
            <span>{SECTION_LABELS.files}</span>
          </TabsTrigger>
        </TabsList>
        <ScrollArea className="flex-1 min-h-0">
          <div
            className="flex flex-col gap-4 px-4 py-4"
            data-testid="session-inspector-tab-panel"
            data-active-tab={active}
          >
            {active === "trace" && (
              <TraceSection
                events={props.traceEvents}
                total={props.traceTotal}
                limit={props.traceLimit}
                onViewAll={props.onViewAllTrace}
                headless
              />
            )}
            {active === "usage" && <UsageSection usage={props.usage} headless />}
            {active === "memory" && <MemorySection docs={props.memoryDocs} headless />}
            {active === "files" && <FilesSection files={props.files} headless />}
          </div>
        </ScrollArea>
      </Tabs>
    </div>
  );
}

/**
 * Right-hand 320px session inspector. Composes `Section` / `Metric` /
 * `MonoBadge` / `StatusDot` / `Tabs` / `ScrollArea` / `Empty` from `@agh/ui`.
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
  headless?: boolean;
}

function TraceSection({ events, total, limit, onViewAll, headless }: TraceSectionProps) {
  const hasOverflow = total > limit;
  const body = (
    <>
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
    </>
  );

  if (headless) {
    return <div data-testid="session-inspector-trace">{body}</div>;
  }
  return (
    <Section label={SECTION_LABELS.trace} data-testid="session-inspector-trace">
      {body}
    </Section>
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
      <StatusDot
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
        <MonoBadge
          tone={tone === "danger" ? "danger" : tone === "warning" ? "warning" : "default"}
          className="shrink-0"
          data-testid="session-inspector-trace-kind"
        >
          {TRACE_KIND_LABEL[event.kind]}
        </MonoBadge>
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
  headless?: boolean;
}

function UsageSection({ usage, headless }: UsageSectionProps) {
  const hasUsage =
    usage !== null &&
    usage !== undefined &&
    (usage.tokensIn !== undefined ||
      usage.tokensOut !== undefined ||
      usage.costUsd !== undefined ||
      usage.ratePerSecond !== undefined);

  const body = hasUsage ? (
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
  );

  if (headless) return <div data-testid="session-inspector-usage">{body}</div>;
  return (
    <Section label={SECTION_LABELS.usage} data-testid="session-inspector-usage">
      {body}
    </Section>
  );
}

interface MemorySectionProps {
  docs: InspectorMemoryDoc[];
  headless?: boolean;
}

function MemorySection({ docs, headless }: MemorySectionProps) {
  const body =
    docs.length === 0 ? (
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
            <MonoBadge tone="info" data-testid="session-inspector-memory-kind">
              {doc.kind}
            </MonoBadge>
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
    );

  if (headless) return <div data-testid="session-inspector-memory">{body}</div>;
  return (
    <Section label={SECTION_LABELS.memory} data-testid="session-inspector-memory">
      {body}
    </Section>
  );
}

interface FilesSectionProps {
  files: InspectorFileEntry[];
  headless?: boolean;
}

function FilesSection({ files, headless }: FilesSectionProps) {
  const body =
    files.length === 0 ? (
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
    );

  if (headless) return <div data-testid="session-inspector-files">{body}</div>;
  return (
    <Section label={SECTION_LABELS.files} data-testid="session-inspector-files">
      {body}
    </Section>
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
