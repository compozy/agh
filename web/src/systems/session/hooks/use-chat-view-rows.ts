import { useMemo, useRef } from "react";

import type { UIMessage } from "../types";

export type RowDescriptor =
  | { kind: "message"; msg: UIMessage }
  | { kind: "tool_group"; tools: UIMessage[] }
  | { kind: "processing" };

const PROCESSING_ROW: RowDescriptor = { kind: "processing" };

export function buildRows(messages: UIMessage[], isStreaming: boolean): RowDescriptor[] {
  const rows: RowDescriptor[] = [];
  let index = 0;

  while (index < messages.length) {
    const message = messages[index];

    if (message.role === "tool_call" || message.role === "tool_result") {
      const tools: UIMessage[] = [];
      while (
        index < messages.length &&
        (messages[index].role === "tool_call" || messages[index].role === "tool_result")
      ) {
        tools.push(messages[index]);
        index++;
      }

      rows.push({ kind: "tool_group", tools });
      continue;
    }

    rows.push({ kind: "message", msg: message });
    index++;
  }

  if (isStreaming) {
    let lastUserIndex = -1;
    for (let messageIndex = messages.length - 1; messageIndex >= 0; messageIndex -= 1) {
      if (messages[messageIndex]?.role === "user") {
        lastUserIndex = messageIndex;
        break;
      }
    }
    const currentTurnMessages = lastUserIndex >= 0 ? messages.slice(lastUserIndex + 1) : messages;
    const hasActiveAssistantStream = currentTurnMessages.some(
      message =>
        message.role === "assistant" && message.isStreaming && (message.content || message.thinking)
    );
    const hasVisibleToolActivity = currentTurnMessages.some(
      message => message.role === "tool_call" || message.role === "tool_result"
    );

    if (!hasActiveAssistantStream && !hasVisibleToolActivity) {
      rows.push(PROCESSING_ROW);
    }
  }

  return rows;
}

export function mergeToolPairs(tools: UIMessage[]): UIMessage[] {
  const resultMap = new Map<string, UIMessage>();

  for (const tool of tools) {
    if (tool.role === "tool_result") {
      resultMap.set(tool.id, tool);
    }
  }

  const merged: UIMessage[] = [];

  for (const tool of tools) {
    if (tool.role !== "tool_call") {
      continue;
    }

    const result = resultMap.get(tool.id);
    if (result) {
      merged.push({
        ...tool,
        toolResult: result.toolResult,
        toolError: result.toolError,
      });
      continue;
    }

    merged.push(tool);
  }

  return merged;
}

export function getRowKey(row: RowDescriptor, index: number): string {
  if (row.kind === "processing") {
    return "__processing__";
  }

  if (row.kind === "tool_group") {
    return `tg-${row.tools[0]?.id ?? index}`;
  }

  return row.msg.id;
}

export function estimateRowHeight(row: RowDescriptor): number {
  if (row.kind === "processing") {
    return 44;
  }

  if (row.kind === "tool_group") {
    return 36 * mergeToolPairs(row.tools).length;
  }

  const message = row.msg;
  if (message.role === "user") {
    return Math.max(56, Math.min(200, 56 + message.content.length / 3));
  }

  if (message.role === "assistant") {
    return Math.max(56, Math.min(600, 56 + message.content.length / 2));
  }

  return 48;
}

function isRowUnchanged(previous: RowDescriptor, next: RowDescriptor): boolean {
  if (previous.kind !== next.kind) {
    return false;
  }

  if (previous.kind === "processing") {
    return true;
  }

  if (previous.kind === "tool_group" && next.kind === "tool_group") {
    return (
      previous.tools === next.tools ||
      (previous.tools.length === next.tools.length &&
        previous.tools.every((tool, index) => tool === next.tools[index]))
    );
  }

  if (previous.kind === "message" && next.kind === "message") {
    return previous.msg === next.msg;
  }

  return false;
}

function computeStableRows(previous: RowDescriptor[], next: RowDescriptor[]): RowDescriptor[] {
  if (previous.length === 0) {
    return next;
  }

  let changed = false;
  const stableRows = next.map((row, index) => {
    if (index < previous.length && isRowUnchanged(previous[index], row)) {
      return previous[index];
    }

    changed = true;
    return row;
  });

  if (!changed && previous.length === next.length) {
    return previous;
  }

  return stableRows;
}

export function useStableRows(messages: UIMessage[], isStreaming: boolean): RowDescriptor[] {
  const previousRowsRef = useRef<RowDescriptor[]>([]);

  return useMemo(() => {
    const nextRows = buildRows(messages, isStreaming);
    const stableRows = computeStableRows(previousRowsRef.current, nextRows);
    previousRowsRef.current = stableRows;
    return stableRows;
  }, [messages, isStreaming]);
}
