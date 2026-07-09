import { useAuiState } from "@assistant-ui/react";
import { useMemo } from "react";

import { cn } from "@/lib/utils";
import {
  formatMessageTimestamp,
  formatMessageTimestampFull,
} from "@/systems/session/lib/format-timestamp";
import { CopyIconButton } from "@agh/ui";

interface MessageActionsState {
  /** The message's markdown text answer, joined and trimmed (empty for a pure tool turn). */
  source: string;
  /** Latest real event time across the message's parts, or null when none is recorded. */
  timestampMs: number | null;
  streaming: boolean;
  /** Copy is offered only for a settled message that carries a text answer. */
  visible: boolean;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function stringField(record: Record<string, unknown>, key: string): string | undefined {
  const value = record[key];
  return typeof value === "string" && value.length > 0 ? value : undefined;
}

// Narration parts stream with either the AI SDK `streaming` state or AGH's own
// `running`; both mean the turn is live. Tool inputs arrive complete, so only
// text/reasoning parts signal an in-flight turn.
function isStreamingState(state: string | undefined): boolean {
  return state === "streaming" || state === "running";
}

// Transcript text/tool parts carry no per-part timestamp (only data-event and
// usage payloads do), so the turn time is the latest timestamp found across the
// parts and their data payloads. Absent entirely for turns with no such event.
function partTimestampMs(part: Record<string, unknown>): number | null {
  const own = stringField(part, "timestamp");
  const nested = isRecord(part.data) ? stringField(part.data, "timestamp") : undefined;
  const raw = own ?? nested;
  if (!raw) return null;
  const parsed = Date.parse(raw);
  return Number.isNaN(parsed) ? null : parsed;
}

/**
 * Derives the copy source, turn timestamp, and streaming flag from a thread
 * message. `visible` is true only for a non-empty, settled answer — never
 * mid-stream, never for a pure tool-work turn with no terminal text.
 */
export function deriveMessageActions(message: {
  content?: unknown;
  status?: { type?: string };
}): MessageActionsState {
  const parts = Array.isArray(message.content) ? message.content : [];
  const textSegments: string[] = [];
  let streaming = message.status?.type === "running";
  let timestampMs: number | null = null;

  for (const part of parts) {
    if (!isRecord(part)) continue;
    const type = stringField(part, "type");
    if ((type === "text" || type === "reasoning") && isStreamingState(stringField(part, "state"))) {
      streaming = true;
    }
    if (type === "text") {
      const text = stringField(part, "text");
      if (text) textSegments.push(text);
    }
    const ts = partTimestampMs(part);
    if (ts !== null && (timestampMs === null || ts > timestampMs)) {
      timestampMs = ts;
    }
  }

  const source = textSegments.join("\n\n").trim();
  return { source, timestampMs, streaming, visible: source.length > 0 && !streaming };
}

// Reveal-on-hover/focus toolbar (`opacity-0 → group-hover:opacity-100`, mapped
// to AGH's neutral ramp + `--duration-slow`). `focus-within` reveals it for
// keyboard users; `pointer-events` gate keeps the hidden row from intercepting
// clicks over the message body.
const REVEAL_CLASS_NAME = cn(
  "flex items-center gap-2 text-small-body text-subtle tabular-nums",
  "opacity-0 pointer-events-none transition-opacity duration-slow motion-reduce:transition-none",
  "group-hover/message:opacity-100 group-hover/message:pointer-events-auto",
  "focus-within:opacity-100 focus-within:pointer-events-auto"
);

export interface MessageActionsProps {
  /** `start` aligns the row under a flat assistant message; `end` under the right-aligned user bubble. */
  align: "start" | "end";
  copyLabel: string;
  testId: string;
}

export function MessageActions({ align, copyLabel, testId }: MessageActionsProps) {
  const message = useAuiState(
    state => state.message as { content?: unknown; status?: { type?: string } }
  );
  const { source, timestampMs, visible } = useMemo(() => deriveMessageActions(message), [message]);

  if (!visible) {
    return null;
  }

  const timestamp =
    timestampMs !== null ? (
      <time
        data-testid={`${testId}-timestamp`}
        dateTime={new Date(timestampMs).toISOString()}
        title={formatMessageTimestampFull(timestampMs)}
        className="text-subtle tabular-nums"
      >
        {formatMessageTimestamp(timestampMs)}
      </time>
    ) : null;

  const copy = (
    <CopyIconButton
      value={source}
      copyLabel={copyLabel}
      copiedLabel="Copied"
      className="text-muted hover:text-fg"
      data-testid={`${testId}-copy`}
    />
  );

  return (
    <div
      data-testid={testId}
      className={cn(REVEAL_CLASS_NAME, align === "end" ? "justify-end" : "justify-start")}
    >
      {align === "end" ? (
        <>
          {timestamp}
          {copy}
        </>
      ) : (
        <>
          {copy}
          {timestamp}
        </>
      )}
    </div>
  );
}
