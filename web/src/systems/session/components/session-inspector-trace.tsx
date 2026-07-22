import {
  type InspectorTraceEvent,
  type InspectorTraceKind,
  type InspectorTraceStatus,
} from "./session-inspector.logic";
import { Button, Empty, Eyebrow, Pill, type PillTone } from "@agh/ui";
import { Activity, ChevronRight } from "lucide-react";
import { formatMessageTimestamp } from "../lib/format-timestamp";

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

interface TraceSectionProps {
  events: InspectorTraceEvent[];
  total: number;
  limit: number;
  onViewAll?: () => void;
}

export function TraceSection({ events, total, limit, onViewAll }: TraceSectionProps) {
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
