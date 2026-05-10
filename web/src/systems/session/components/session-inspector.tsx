import { useCallback, useMemo, useState } from "react";
import {
  Activity,
  AlertCircle,
  ChevronRight,
  FileCode,
  Gauge,
  KeyRound,
  Library,
  Loader2,
  PanelRightOpen,
} from "lucide-react";
import type { AssistantState } from "@assistant-ui/react";

import {
  Button,
  Empty,
  MetadataList,
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
import { SessionLedgerUnavailableError } from "../adapters/session-api";
import type { SessionLedgerEvent, SessionLedgerMeta } from "../types";
import { SessionVaultPanel, type VaultSecret } from "@/systems/vault";

type ThreadMessageState = AssistantState["thread"]["messages"][number];
const EMPTY_VAULT_SECRETS: readonly VaultSecret[] = [];
const EMPTY_INSPECTOR_FILES: InspectorFileEntry[] = [];

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

export interface InspectorSessionLedger {
  meta: SessionLedgerMeta;
  events: readonly SessionLedgerEvent[];
}

export interface InspectorMemoryState {
  ledger?: InspectorSessionLedger | null;
  isLoading?: boolean;
  error?: Error | null;
}

export interface InspectorFileEntry {
  path: string;
  readCount: number;
}

export interface SessionInspectorProps {
  messages: readonly ThreadMessageState[];
  sessionId?: string;
  usage?: InspectorUsage | null;
  /**
   * Forensic Memory v2 session ledger state. The Memory tab renders the
   * lineage meta block plus the full session ledger event stream (transcript,
   * memory, lifecycle, redaction metadata) and surfaces truthful
   * loading/empty/error states without ever exposing editor or replay
   * controls.
   */
  memory?: InspectorMemoryState;
  vaultSecrets?: readonly VaultSecret[];
  vaultIsLoading?: boolean;
  vaultError?: Error | null;
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
const LEDGER_EVENT_LIMIT = 20;
const EMPTY_MEMORY_STATE: InspectorMemoryState = Object.freeze({});
const SECTION_LABELS = {
  trace: "Trace",
  usage: "Usage",
  memory: "Memory",
  files: "Files",
  vault: "Vault",
} as const;

type TopTab = "trace" | "usage";
type BottomTab = "memory" | "files" | "vault";

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
  return message.content.reduce((text, part) => {
    if (part.type !== "text" && part.type !== "reasoning") {
      return text;
    }
    return `${text}${part.text}`;
  }, "");
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
  memory: InspectorMemoryState;
  sessionId?: string;
  vaultSecrets: readonly VaultSecret[];
  vaultIsLoading: boolean;
  vaultError: Error | null;
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
  memory,
  sessionId,
  vaultSecrets,
  vaultIsLoading,
  vaultError,
  files,
}: SectionBodyProps) {
  const [topTab, setTopTab] = useState<TopTab>("trace");
  const [bottomTab, setBottomTab] = useState<BottomTab>("memory");
  const handleTopChange = useCallback((value: string | null | undefined) => {
    if (value === "trace" || value === "usage") setTopTab(value);
  }, []);
  const handleBottomChange = useCallback((value: string | null | undefined) => {
    if (value === "memory" || value === "files" || value === "vault") setBottomTab(value);
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
          className="w-full shrink-0 border-b border-(--line) px-2 group-data-horizontal/tabs:h-12"
        >
          <TabsTrigger
            value="trace"
            data-testid="session-inspector-tab-trace"
            className="h-12 gap-2 group-data-horizontal/tabs:after:-bottom-px"
          >
            <Activity className="size-3.5" />
            <span>{SECTION_LABELS.trace}</span>
          </TabsTrigger>
          <TabsTrigger
            value="usage"
            data-testid="session-inspector-tab-usage"
            className="h-12 gap-2 group-data-horizontal/tabs:after:-bottom-px"
          >
            <Gauge className="size-3.5" />
            <span>{SECTION_LABELS.usage}</span>
          </TabsTrigger>
        </TabsList>
        <ScrollArea className="flex-1 min-h-0">
          <div
            className="flex min-h-full flex-col gap-4 p-4"
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
        aria-label="Memory, files, and vault"
        value={bottomTab}
        onValueChange={handleBottomChange}
        className="flex min-h-0 flex-1 basis-0 flex-col gap-0 border-t border-(--line)"
      >
        <TabsList
          variant="line"
          className="w-full shrink-0 border-b border-(--line) px-2 group-data-horizontal/tabs:h-12"
        >
          <TabsTrigger
            value="memory"
            data-testid="session-inspector-tab-memory"
            className="h-12 gap-2 group-data-horizontal/tabs:after:-bottom-px"
          >
            <Library className="size-3.5" />
            <span>{SECTION_LABELS.memory}</span>
          </TabsTrigger>
          <TabsTrigger
            value="files"
            data-testid="session-inspector-tab-files"
            className="h-12 gap-2 group-data-horizontal/tabs:after:-bottom-px"
          >
            <FileCode className="size-3.5" />
            <span>{SECTION_LABELS.files}</span>
          </TabsTrigger>
          <TabsTrigger
            value="vault"
            data-testid="session-inspector-tab-vault"
            className="h-12 gap-2 group-data-horizontal/tabs:after:-bottom-px"
          >
            <KeyRound className="size-3.5" />
            <span>{SECTION_LABELS.vault}</span>
          </TabsTrigger>
        </TabsList>
        <ScrollArea className="flex-1 min-h-0">
          <div
            className="flex min-h-full flex-col gap-4 p-4"
            data-testid="session-inspector-bottom-panel"
            data-active-tab={bottomTab}
          >
            {bottomTab === "memory" && <MemorySection memory={memory} />}
            {bottomTab === "files" && <FilesSection files={files} />}
            {bottomTab === "vault" && (
              <SessionVaultPanel
                secrets={vaultSecrets}
                isLoading={vaultIsLoading}
                error={vaultError}
                sessionId={sessionId}
              />
            )}
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
  sessionId,
  usage,
  memory,
  vaultSecrets = EMPTY_VAULT_SECRETS,
  vaultIsLoading = false,
  vaultError = null,
  files = EMPTY_INSPECTOR_FILES,
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
  const memoryState = memory ?? EMPTY_MEMORY_STATE;

  return (
    <aside
      data-testid="session-inspector"
      aria-label="Session inspector"
      style={{ width: INSPECTOR_WIDTH }}
      className={cn(
        "hidden shrink-0 flex-col overflow-hidden border-l bg-(--canvas)",
        "border-(--line) min-w-0",
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
        memory={memoryState}
        sessionId={sessionId}
        vaultSecrets={vaultSecrets}
        vaultIsLoading={vaultIsLoading}
        vaultError={vaultError}
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
          className="mt-3 h-7 gap-1 self-start px-1 text-(--muted) hover:text-(--fg)"
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
          className="shrink-0 font-mono text-badge uppercase tracking-mono text-(--subtle)"
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
          className="min-w-0 flex-1 truncate text-small-body text-(--fg)"
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
            className="p-3"
          />
          <Metric
            label="Tokens out"
            value={formatNumber(usage?.tokensOut)}
            tone={deltaTone(usage?.tokensOutDelta)}
            detail={deltaLabel(usage?.tokensOutDelta)}
            data-testid="session-inspector-usage-tokens-out"
            className="p-3"
          />
          <Metric
            label="Total cost"
            value={formatCost(usage?.costUsd)}
            tone={deltaTone(usage?.costDelta)}
            detail={deltaLabel(usage?.costDelta)}
            data-testid="session-inspector-usage-cost"
            className="p-3"
          />
          <Metric
            label="Est. rate"
            value={
              typeof usage?.ratePerSecond === "number" && Number.isFinite(usage.ratePerSecond)
                ? `${usage.ratePerSecond.toFixed(1)}/s`
                : "—"
            }
            data-testid="session-inspector-usage-rate"
            className="p-3"
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
  memory: InspectorMemoryState;
}

function MemorySection({ memory }: MemorySectionProps) {
  if (memory.isLoading) {
    return (
      <div
        data-testid="session-inspector-memory"
        data-state="loading"
        className="flex min-h-full flex-col"
      >
        <div
          data-testid="session-inspector-memory-loading"
          className="flex items-center gap-2 px-1 py-3 text-xs text-(--subtle)"
        >
          <Loader2 aria-hidden="true" className="size-4 animate-spin" />
          Loading session ledger…
        </div>
      </div>
    );
  }

  if (memory.error && !(memory.error instanceof SessionLedgerUnavailableError)) {
    return (
      <div
        data-testid="session-inspector-memory"
        data-state="error"
        className="flex min-h-full flex-col"
      >
        <Empty
          icon={AlertCircle}
          title="Unable to load session ledger"
          description={memory.error.message || "Failed to load forensic session ledger."}
          data-testid="session-inspector-memory-error"
        />
      </div>
    );
  }

  const ledger = memory.ledger;
  if (!ledger) {
    return (
      <div
        data-testid="session-inspector-memory"
        data-state="unavailable"
        className="flex min-h-full flex-col"
      >
        <Empty
          icon={Library}
          title="No session ledger yet"
          description="The forensic ledger materializes once the session stops. Lineage and ledger event metadata appear here after that."
          data-testid="session-inspector-memory-empty"
        />
      </div>
    );
  }

  return (
    <div
      data-testid="session-inspector-memory"
      data-state="ready"
      className="flex min-h-full flex-col gap-4"
    >
      <SessionLedgerMetaPanel meta={ledger.meta} />
      <SessionLedgerEventsPanel events={ledger.events} />
    </div>
  );
}

interface SessionLedgerMetaPanelProps {
  meta: SessionLedgerMeta;
}

function SessionLedgerMetaPanel({ meta }: SessionLedgerMetaPanelProps) {
  const items: Array<{ label: string; value: string; testId: string; mono?: boolean }> = [
    { label: "Workspace", value: meta.workspace_id ?? "—", testId: "workspace", mono: true },
    {
      label: "Root session",
      value: meta.root_session_id ?? meta.session_id,
      testId: "root-session",
      mono: true,
    },
    {
      label: "Parent session",
      value: meta.parent_session_id ?? "—",
      testId: "parent-session",
      mono: true,
    },
    {
      label: "Spawn depth",
      value: String(meta.spawn_depth),
      testId: "spawn-depth",
      mono: true,
    },
    {
      label: "Created",
      value: formatLedgerTimestamp(meta.created_at),
      testId: "created-at",
      mono: true,
    },
    {
      label: "Stopped",
      value: meta.stopped_at ? formatLedgerTimestamp(meta.stopped_at) : "--",
      testId: "stopped-at",
      mono: true,
    },
    { label: "Path", value: meta.path, testId: "path", mono: true },
    { label: "Checksum", value: meta.checksum, testId: "checksum", mono: true },
    { label: "Version", value: `v${meta.version}`, testId: "version", mono: true },
  ];

  return (
    <section
      aria-label="Session ledger lineage"
      data-testid="session-inspector-memory-meta"
      className="flex flex-col gap-2"
    >
      <div className="flex items-center gap-2">
        <Pill mono tone="info" data-testid="session-inspector-memory-meta-kind">
          LEDGER
        </Pill>
        <span className="font-mono text-badge uppercase tracking-mono text-(--muted)">
          Forensic
        </span>
      </div>
      <MetadataList>
        {items.map(item => (
          <MetadataList.Row
            key={item.testId}
            data-testid={`session-inspector-memory-meta-${item.testId}`}
            className="items-baseline justify-between gap-2"
          >
            <MetadataList.Term>{item.label}</MetadataList.Term>
            <MetadataList.Value
              className={cn(
                "min-w-0 flex-1 break-all text-right text-xs text-(--fg)",
                item.mono ? "font-mono text-eyebrow" : null
              )}
              data-testid={`session-inspector-memory-meta-${item.testId}-value`}
            >
              {item.value}
            </MetadataList.Value>
          </MetadataList.Row>
        ))}
      </MetadataList>
    </section>
  );
}

interface SessionLedgerEventsPanelProps {
  events: readonly SessionLedgerEvent[];
}

function SessionLedgerEventsPanel({ events }: SessionLedgerEventsPanelProps) {
  const visible = events.slice(-LEDGER_EVENT_LIMIT);
  return (
    <section
      aria-label="Session ledger events"
      data-testid="session-inspector-memory-events"
      className="flex flex-col gap-2"
    >
      <div className="flex items-center justify-between gap-2">
        <span className="font-mono text-badge uppercase tracking-mono text-(--muted)">
          Ledger events
        </span>
        <span
          className="font-mono text-badge text-(--subtle)"
          data-testid="session-inspector-memory-events-count"
        >
          {events.length}
        </span>
      </div>
      {visible.length === 0 ? (
        <Empty
          icon={Library}
          title="No ledger events"
          description="The session ended without recorded events; nothing was journaled for this run."
          data-testid="session-inspector-memory-events-empty"
        />
      ) : (
        <ul
          data-testid="session-inspector-memory-events-list"
          className="flex flex-col divide-y divide-(--line)"
        >
          {visible.map(event => (
            <li
              key={`${event.sequence}-${event.event_type}`}
              data-testid="session-inspector-memory-event-row"
              className="flex items-center gap-2 py-2"
            >
              <span
                data-testid="session-inspector-memory-event-sequence"
                className="shrink-0 font-mono text-badge uppercase tracking-mono text-(--subtle)"
              >
                #{event.sequence}
              </span>
              <Pill mono tone="neutral" data-testid="session-inspector-memory-event-type">
                {event.event_type}
              </Pill>
              <span
                data-testid="session-inspector-memory-event-timestamp"
                className="ml-auto shrink-0 font-mono text-badge text-(--subtle)"
              >
                {formatLedgerTimestamp(event.emitted_at)}
              </span>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

function formatLedgerTimestamp(value: string): string {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString([], {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
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
          className="max-h-[240px] rounded-md border border-(--line) bg-(--canvas-soft)"
        >
          <ul
            data-testid="session-inspector-files-list"
            className="flex flex-col divide-y divide-(--line)"
          >
            {files.map(file => (
              <li
                key={file.path}
                data-testid="session-inspector-files-row"
                className="flex items-center gap-2 px-2 py-1.5"
              >
                <FileCode aria-hidden="true" className="size-3 shrink-0 text-(--subtle)" />
                <span
                  data-testid="session-inspector-files-path"
                  className="min-w-0 flex-1 truncate font-mono text-eyebrow text-(--fg)"
                >
                  {file.path}
                </span>
                <span
                  data-testid="session-inspector-files-count"
                  className="shrink-0 font-mono text-badge text-(--subtle)"
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
  sessionId,
  usage,
  memory,
  vaultSecrets = EMPTY_VAULT_SECRETS,
  vaultIsLoading = false,
  vaultError = null,
  files = EMPTY_INSPECTOR_FILES,
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
  const memoryState = memory ?? EMPTY_MEMORY_STATE;

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
        className="flex w-[min(88vw,360px)] max-w-[360px] flex-col gap-0 bg-(--canvas) p-0 sm:max-w-[360px]"
      >
        <header className="flex h-12 shrink-0 items-center justify-between border-b border-(--line) px-4">
          <span className="font-mono text-eyebrow font-semibold uppercase tracking-mono text-(--muted)">
            Inspector
          </span>
        </header>
        <InspectorBody
          traceEvents={traceEvents}
          traceTotal={traceTotal}
          traceLimit={traceLimit}
          onViewAllTrace={onViewAllTrace}
          usage={usage}
          memory={memoryState}
          sessionId={sessionId}
          vaultSecrets={vaultSecrets}
          vaultIsLoading={vaultIsLoading}
          vaultError={vaultError}
          files={derivedFiles}
        />
      </SheetContent>
    </Sheet>
  );
}
