import { useState } from "react";
import { Activity, AlertCircle, ChevronRight, FileCode, Gauge, Library } from "lucide-react";

import {
  Button,
  DetailInspector,
  Empty,
  Eyebrow,
  MetadataList,
  Metric,
  Pill,
  ScrollArea,
  Spinner,
  cn,
  type PillTone,
} from "@agh/ui";

import { formatMessageTimestamp } from "../lib/format-timestamp";
import { SessionLedgerUnavailableError } from "../adapters/session-api";
import type { SessionLedgerEvent, SessionLedgerMeta } from "../types";
import { SessionVaultPanel, type VaultSecret } from "@/systems/vault";
import {
  deriveFileReads,
  deriveTraceEvents,
  TRACE_LIMIT_DEFAULT,
  type InspectorFileEntry,
  type InspectorTraceEvent,
  type InspectorTraceKind,
  type InspectorTraceStatus,
  type ThreadMessageState,
} from "./session-inspector.logic";

const EMPTY_VAULT_SECRETS: readonly VaultSecret[] = [];

/**
 * Aggregated token-usage summary for the inspector Usage tab. Every field maps
 * to a value the daemon actually reports (`GET .../sessions/{id}/usage`), so the
 * tab renders real data instead of a structurally-empty metric surface. Absent
 * fields render as "—" rather than fabricated zeros; per-turn rates and deltas
 * are intentionally omitted because the daemon does not expose them truthfully.
 */
export interface InspectorUsage {
  tokensIn?: number;
  tokensOut?: number;
  totalTokens?: number;
  costUsd?: number;
  /** ISO-4217 currency code for `costUsd`; defaults to USD formatting when empty. */
  costCurrency?: string;
  /** Number of turns the aggregate spans; shown as a caption when > 0. */
  turnCount?: number;
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
  /** Drawer open state when viewport is below the inline breakpoint. */
  drawerOpen?: boolean;
  /** Drawer open-state change handler. */
  onDrawerOpenChange?: (open: boolean) => void;
  className?: string;
}

const LEDGER_EVENT_LIMIT = 20;
const EMPTY_MEMORY_STATE: InspectorMemoryState = Object.freeze({});
const SECTION_LABELS = {
  trace: "Trace",
  usage: "Usage",
  memory: "Memory",
  files: "Files",
  vault: "Vault",
} as const;

type InspectorTabId = "trace" | "usage" | "memory" | "files" | "vault";

const SESSION_INSPECTOR_TABS = [
  { id: "trace", label: SECTION_LABELS.trace },
  { id: "usage", label: SECTION_LABELS.usage },
  { id: "memory", label: SECTION_LABELS.memory },
  { id: "files", label: SECTION_LABELS.files },
  { id: "vault", label: SECTION_LABELS.vault },
] as const satisfies ReadonlyArray<{ id: InspectorTabId; label: string }>;

const SESSION_INSPECTOR_TAB_TESTIDS: Record<InspectorTabId, string> = {
  trace: "session-inspector-tab-trace",
  usage: "session-inspector-tab-usage",
  memory: "session-inspector-tab-memory",
  files: "session-inspector-tab-files",
  vault: "session-inspector-tab-vault",
};

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

function formatNumber(value?: number): string {
  if (typeof value !== "number" || !Number.isFinite(value)) return "—";
  return value.toLocaleString();
}

const currencyFormatters = new Map<string, Intl.NumberFormat>();

function currencyFormatter(currency: string, digits: number): Intl.NumberFormat {
  const key = `${currency}:${digits}`;
  const cached = currencyFormatters.get(key);
  if (cached) return cached;
  const formatter = Intl.NumberFormat(undefined, {
    style: "currency",
    currency,
    minimumFractionDigits: digits,
    maximumFractionDigits: digits,
  });
  currencyFormatters.set(key, formatter);
  return formatter;
}

function formatCost(value?: number, currency?: string): string {
  if (typeof value !== "number" || !Number.isFinite(value)) return "—";
  const code = normalizedCurrencyCode(currency);
  const digits = Math.abs(value) < 1 ? 3 : 2;
  return currencyFormatter(code, digits).format(value);
}

function normalizedCurrencyCode(currency?: string): string {
  const code = currency?.trim().toUpperCase();
  return code && /^[A-Z]{3}$/.test(code) ? code : "USD";
}

interface InspectorTabRendererProps {
  activeTab: InspectorTabId;
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

function InspectorTabRenderer({
  activeTab,
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
}: InspectorTabRendererProps) {
  switch (activeTab) {
    case "trace":
      return (
        <TraceSection
          events={traceEvents}
          total={traceTotal}
          limit={traceLimit}
          onViewAll={onViewAllTrace}
        />
      );
    case "usage":
      return <UsageSection usage={usage} />;
    case "memory":
      return <MemorySection memory={memory} />;
    case "files":
      return <FilesSection files={files} />;
    case "vault":
      return (
        <SessionVaultPanel
          secrets={vaultSecrets}
          isLoading={vaultIsLoading}
          error={vaultError}
          sessionId={sessionId}
        />
      );
  }
}

/**
 * Right-hand session inspector — consumes `<DetailInspector>`
 * and renders 5 tabs (Trace · Usage · Memory · Files · Vault) inside the
 * primitive's chrome. Inline at ≥ 1440 px viewport; collapses
 * into a right-anchored sheet drawer below that breakpoint, with the open
 * state controlled by the consumer through `drawerOpen` / `onDrawerOpenChange`.
 */
export function SessionInspector({
  messages,
  sessionId,
  usage,
  memory,
  vaultSecrets = EMPTY_VAULT_SECRETS,
  vaultIsLoading = false,
  vaultError = null,
  files,
  totalTraceEvents,
  traceLimit = TRACE_LIMIT_DEFAULT,
  onViewAllTrace,
  drawerOpen,
  onDrawerOpenChange,
  className,
}: SessionInspectorProps) {
  const [activeTab, setActiveTab] = useState<InspectorTabId>("trace");
  const handleTabChange = (id: string) => {
    if (id === "trace" || id === "usage" || id === "memory" || id === "files" || id === "vault") {
      setActiveTab(id);
    }
  };

  const traceEvents = deriveTraceEvents(messages, traceLimit);
  const derivedFiles = files ?? deriveFileReads(messages);
  const traceTotal = totalTraceEvents ?? messages.length;
  const memoryState = memory ?? EMPTY_MEMORY_STATE;

  const tabs = SESSION_INSPECTOR_TABS.map(tab => ({
    id: tab.id,
    label: <span data-testid={SESSION_INSPECTOR_TAB_TESTIDS[tab.id]}>{tab.label}</span>,
  }));

  return (
    <DetailInspector
      data-testid="session-inspector"
      tabs={tabs}
      activeTab={activeTab}
      onTabChange={handleTabChange}
      open={drawerOpen}
      onOpenChange={onDrawerOpenChange}
      className={cn("min-w-0", className)}
    >
      <div
        className="flex min-h-full flex-col gap-4 p-4"
        data-testid="session-inspector-panel"
        data-active-tab={activeTab}
      >
        <InspectorTabRenderer
          activeTab={activeTab}
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
      </div>
    </DetailInspector>
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
          className="mt-3 h-7 gap-1 self-start px-1 text-muted hover:text-fg"
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
  const ts = formatMessageTimestamp(event.timestamp);
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
        <Eyebrow data-testid="session-inspector-trace-timestamp" className="text-subtle shrink-0">
          {ts}
        </Eyebrow>
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
          className="min-w-0 flex-1 truncate text-small-body text-fg"
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
      usage.totalTokens !== undefined ||
      usage.costUsd !== undefined);
  const turnCount = usage?.turnCount ?? 0;

  return (
    <div data-testid="session-inspector-usage" className="flex min-h-full flex-col gap-3">
      {hasUsage ? (
        <>
          <div data-testid="session-inspector-usage-grid" className="grid grid-cols-2 gap-2">
            <Metric
              label="Tokens in"
              value={formatNumber(usage?.tokensIn)}
              data-testid="session-inspector-usage-tokens-in"
              className="p-3"
            />
            <Metric
              label="Tokens out"
              value={formatNumber(usage?.tokensOut)}
              data-testid="session-inspector-usage-tokens-out"
              className="p-3"
            />
            <Metric
              label="Total tokens"
              value={formatNumber(usage?.totalTokens)}
              data-testid="session-inspector-usage-total-tokens"
              className="p-3"
            />
            <Metric
              label="Total cost"
              value={formatCost(usage?.costUsd, usage?.costCurrency)}
              data-testid="session-inspector-usage-cost"
              className="p-3"
            />
          </div>
          {turnCount > 0 ? (
            <Eyebrow data-testid="session-inspector-usage-turns" className="text-subtle self-start">
              {`Across ${turnCount.toLocaleString()} turn${turnCount === 1 ? "" : "s"}`}
            </Eyebrow>
          ) : null}
        </>
      ) : (
        <Empty
          icon={Gauge}
          title="No usage yet"
          description="Token counts and cost land here once the agent reports its first turn."
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
          className="flex items-center gap-2 px-1 py-3 text-small-body text-subtle"
        >
          <Spinner aria-hidden="true" />
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
        <Eyebrow className="text-muted">Forensic</Eyebrow>
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
                "min-w-0 flex-1 break-all text-right text-form-label text-fg",
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
        <Eyebrow className="text-muted">Ledger events</Eyebrow>
        <span
          className="font-mono text-badge text-subtle"
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
          className="flex flex-col divide-y divide-line"
        >
          {visible.map(event => (
            <li
              key={`${event.sequence}-${event.event_type}`}
              data-testid="session-inspector-memory-event-row"
              className="flex items-center gap-2 py-2"
            >
              <Eyebrow
                data-testid="session-inspector-memory-event-sequence"
                className="text-subtle shrink-0"
              >
                #{event.sequence}
              </Eyebrow>
              <Pill mono tone="neutral" data-testid="session-inspector-memory-event-type">
                {event.event_type}
              </Pill>
              <span
                data-testid="session-inspector-memory-event-timestamp"
                className="ml-auto shrink-0 font-mono text-badge text-subtle"
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
          className="max-h-60 rounded-md border border-line bg-canvas-soft"
        >
          <ul
            data-testid="session-inspector-files-list"
            className="flex flex-col divide-y divide-line"
          >
            {files.map(file => (
              <li
                key={file.path}
                data-testid="session-inspector-files-row"
                className="flex items-center gap-2 px-2 py-1.5"
              >
                <FileCode aria-hidden="true" className="size-3 shrink-0 text-subtle" />
                <span
                  data-testid="session-inspector-files-path"
                  className="min-w-0 flex-1 truncate font-mono text-eyebrow text-fg"
                >
                  {file.path}
                </span>
                <span
                  data-testid="session-inspector-files-count"
                  className="shrink-0 font-mono text-badge text-subtle"
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
