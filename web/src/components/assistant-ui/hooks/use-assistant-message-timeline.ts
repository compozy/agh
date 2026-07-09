import { useAuiState } from "@assistant-ui/react";
import { useCallback, useMemo, useRef, useState } from "react";
import { flushSync } from "react-dom";

import {
  computeStableSessionRows,
  deriveSessionRows,
  EMPTY_STABLE_SESSION_ROWS,
  isStreamingState,
  type SessionRow,
  type SessionTimelinePart,
  type SessionTimelineWorkingPart,
  type StableSessionRowsState,
} from "../session-timeline.logic";
import { isRecord, stringField, toTimelineParts } from "../timeline-message-parts";

// A turn is "working" while it streams: assistant-ui marks the message status
// `running`, and/or a part is still live — a text/reasoning part in a streaming
// state, or a tool call awaiting its result.
function messageStatusIsRunning(message: { status?: unknown }): boolean {
  const status = message.status;
  return isRecord(status) && stringField(status, "type") === "running";
}

function partIsLive(part: SessionTimelinePart): boolean {
  if (part.kind === "tool") {
    return part.status === "running";
  }
  return isStreamingState(part.state);
}

// The turn start anchors the "Working for Xs" timer. Parts carry ISO timestamps
// (data events, or re-attached turn metadata); take the earliest known one so the
// timer counts from when the turn began. Undefined falls back to a static label.
function earliestTimestampMs(parts: readonly SessionTimelinePart[]): number | undefined {
  let earliest: number | undefined;
  for (const part of parts) {
    if (!part.timestamp) continue;
    const ms = Date.parse(part.timestamp);
    if (!Number.isFinite(ms)) continue;
    if (earliest === undefined || ms < earliest) earliest = ms;
  }
  return earliest;
}

// When the turn is streaming, append a trailing `working` part so the timeline
// renders the streaming indicator below the turn's content; the fold keeps the
// live turn inline because a `working` row marks the group active.
function deriveWorkingPart(
  message: { id?: string; status?: unknown },
  parts: readonly SessionTimelinePart[]
): SessionTimelineWorkingPart | null {
  if (!messageStatusIsRunning(message) && !parts.some(partIsLive)) {
    return null;
  }
  const turnId = typeof message.id === "string" && message.id.length > 0 ? message.id : undefined;
  return {
    kind: "working",
    id: `${message.id ?? "message"}:working`,
    turnId,
    startedAt: earliestTimestampMs(parts),
  };
}

// The runtime signals an operator stop through a data-event `stop_reason`; map it
// to the message's turn so the fold labels the interruption and stays expanded.
// The runtime normalizes agh-event parts to `{ type: "data", name: "agh-event" }`.
const INTERRUPT_STOP_REASONS = new Set([
  "cancelled",
  "canceled",
  "interrupted",
  "aborted",
  "stopped",
]);

function isAghEventPart(part: Record<string, unknown>): boolean {
  const type = stringField(part, "type");
  if (type === "data-agh-event") return true;
  return type === "data" && stringField(part, "name") === "agh-event";
}

function interruptedTurnIds(
  content: unknown,
  fallbackTurnId: string | undefined
): ReadonlySet<string> | undefined {
  if (!Array.isArray(content)) return undefined;
  const ids = new Set<string>();
  for (const part of content) {
    if (!isRecord(part) || !isAghEventPart(part)) continue;
    const data = part.data;
    if (!isRecord(data)) continue;
    const stopReason = stringField(data, "stop_reason")?.toLowerCase();
    if (!stopReason || !INTERRUPT_STOP_REASONS.has(stopReason)) continue;
    const turnId = stringField(data, "turn_id") ?? stringField(part, "turnId") ?? fallbackTurnId;
    if (turnId) ids.add(turnId);
  }
  return ids.size > 0 ? ids : undefined;
}

/**
 * Toggles a disclosure while preserving the reader's viewport anchor: it measures
 * the scroll container before the synchronous state flush, then corrects
 * `scrollTop` by the height delta so newly revealed rows never jump the view.
 */
function useAnchorPreservingToggle() {
  return useCallback((button: HTMLElement | null, update: () => void) => {
    const viewport = button?.closest<HTMLElement>('[data-testid="chat-view"]');
    const previousHeight = viewport?.scrollHeight ?? 0;
    const previousTop = viewport?.scrollTop ?? 0;
    flushSync(update);
    if (!viewport) return;
    const delta = viewport.scrollHeight - previousHeight;
    viewport.scrollTop = previousTop + delta;
  }, []);
}

export function useAssistantMessageTimeline() {
  const message = useAuiState(
    state => state.message as { id?: string; content?: unknown; status?: unknown }
  );
  const parts = useMemo(() => {
    const base = toTimelineParts(message);
    const workingPart = deriveWorkingPart(message, base);
    return workingPart ? [...base, workingPart] : base;
  }, [message]);
  const interruptedTurns = useMemo(
    () =>
      interruptedTurnIds(
        message.content,
        typeof message.id === "string" && message.id.length > 0 ? message.id : undefined
      ),
    [message.content, message.id]
  );
  const [expandedWorkGroups, setExpandedWorkGroups] = useState<Set<string>>(() => new Set());
  const [expandedTurns, setExpandedTurns] = useState<Set<string>>(() => new Set());
  const [expandedChangedFiles, setExpandedChangedFiles] = useState<Set<string>>(() => new Set());
  const stableRef = useRef<StableSessionRowsState>(EMPTY_STABLE_SESSION_ROWS);

  const rows = useMemo(() => {
    const derived = deriveSessionRows(parts, {
      foldSettledTurns: true,
      interruptedTurnIds: interruptedTurns,
      expandedWorkGroupIds: expandedWorkGroups,
      expandedTurnIds: expandedTurns,
      expandedChangedFilesIds: expandedChangedFiles,
    });
    const stable = computeStableSessionRows(derived, stableRef.current);
    stableRef.current = stable;
    return stable.result;
  }, [parts, interruptedTurns, expandedWorkGroups, expandedTurns, expandedChangedFiles]);

  const anchorToggle = useAnchorPreservingToggle();

  const toggleWorkGroup = useCallback(
    (groupId: string, button: HTMLElement | null) => {
      anchorToggle(button, () => {
        setExpandedWorkGroups(previous => {
          const next = new Set(previous);
          if (next.has(groupId)) next.delete(groupId);
          else next.add(groupId);
          return next;
        });
      });
    },
    [anchorToggle]
  );

  const toggleTurn = useCallback(
    (turnId: string, button: HTMLElement | null) => {
      anchorToggle(button, () => {
        setExpandedTurns(previous => {
          const next = new Set(previous);
          if (next.has(turnId)) next.delete(turnId);
          else next.add(turnId);
          return next;
        });
      });
    },
    [anchorToggle]
  );

  const toggleChangedFiles = useCallback(
    (rowId: string, button: HTMLElement | null) => {
      anchorToggle(button, () => {
        setExpandedChangedFiles(previous => {
          const next = new Set(previous);
          if (next.has(rowId)) next.delete(rowId);
          else next.add(rowId);
          return next;
        });
      });
    },
    [anchorToggle]
  );

  return {
    expandedTurns,
    expandedWorkGroups,
    rows,
    toggleChangedFiles,
    toggleTurn,
    toggleWorkGroup,
  } satisfies {
    expandedTurns: ReadonlySet<string>;
    expandedWorkGroups: ReadonlySet<string>;
    rows: SessionRow[];
    toggleChangedFiles: (rowId: string, button: HTMLElement | null) => void;
    toggleTurn: (turnId: string, button: HTMLElement | null) => void;
    toggleWorkGroup: (groupId: string, button: HTMLElement | null) => void;
  };
}
