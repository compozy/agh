import type { SessionLedgerEvent, SessionLedgerMeta } from "../types";
import { Empty, Eyebrow, MetadataList, Pill, cn } from "@agh/ui";
import { Library } from "lucide-react";

import { LEDGER_EVENT_LIMIT } from "./session-inspector";

interface SessionLedgerMetaPanelProps {
  meta: SessionLedgerMeta;
}

export function SessionLedgerMetaPanel({ meta }: SessionLedgerMetaPanelProps) {
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

export function SessionLedgerEventsPanel({ events }: SessionLedgerEventsPanelProps) {
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
