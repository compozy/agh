import type { SessionEventPayload, SessionMessage } from "../types";

export function numberFromEventID(value: unknown): number | null {
  if (typeof value !== "string") {
    return null;
  }
  const trimmed = value.trim();
  if (trimmed.length === 0) {
    return null;
  }
  const parsed = Number.parseInt(trimmed, 10);
  return Number.isFinite(parsed) ? parsed : null;
}

export function parseSessionStreamPayload<T>(event: MessageEvent): T | null {
  if (typeof event.data !== "string" || event.data.trim().length === 0) {
    return null;
  }
  try {
    return JSON.parse(event.data) as T;
  } catch {
    return null;
  }
}

function terminalFailureText(payload: SessionEventPayload): string | null {
  const failureSummary = payload.failure?.summary?.trim();
  if (failureSummary) {
    return failureSummary;
  }
  const stopDetail = payload.stop_detail?.trim();
  return stopDetail && stopDetail.length > 0 ? stopDetail : null;
}

export function terminalFailureMessage(
  payload: SessionEventPayload,
  fallbackSessionId: string
): SessionMessage | null {
  const detail = terminalFailureText(payload);
  if (!detail) {
    return null;
  }
  const sessionID = payload.session_id || fallbackSessionId;
  return {
    id: `session-stopped-${sessionID}`,
    role: "assistant",
    parts: [
      {
        type: "data-agh-event",
        data: {
          type: "error",
          session_id: sessionID,
          timestamp: payload.timestamp,
          stop_reason: payload.stop_reason,
          error: detail,
          failure: payload.failure ?? undefined,
        },
      },
    ],
  } as SessionMessage;
}

export function upsertTranscriptMessage(
  existing: SessionMessage[] | undefined,
  message: SessionMessage
): SessionMessage[] {
  const messages = existing ? [...existing] : [];
  const index = messages.findIndex(item => item.id === message.id);
  if (index === -1) {
    messages.push(message);
    return messages;
  }
  messages[index] = message;
  return messages;
}

export function sortMessagesByKnownSequence(
  messages: SessionMessage[],
  sequences: ReadonlyMap<string, number>
): SessionMessage[] {
  return [...messages].sort((left, right) => {
    const leftSequence = sequences.get(left.id);
    const rightSequence = sequences.get(right.id);
    if (leftSequence !== undefined && rightSequence !== undefined) {
      return leftSequence - rightSequence;
    }
    if (leftSequence !== undefined) {
      return 1;
    }
    if (rightSequence !== undefined) {
      return -1;
    }
    return 0;
  });
}
