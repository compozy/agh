import { memo, useCallback, useEffect, useLayoutEffect, useRef, useMemo, useState } from "react";
import { useVirtualizer } from "@tanstack/react-virtual";
import { ArrowDown, MessageSquare } from "lucide-react";

import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import type { UIMessage } from "../types";
import { MessageBubble } from "./message-bubble";
import { ProcessingIndicator } from "./processing-indicator";
import { ToolCallCard } from "./tool-call-card";

// ── Row model ──

export type RowDescriptor =
  | { kind: "message"; msg: UIMessage }
  | { kind: "tool_group"; tools: UIMessage[] }
  | { kind: "processing" };

const PROCESSING_ROW: RowDescriptor = { kind: "processing" };
const BOTTOM_LOCK_THRESHOLD_PX = 80;

/**
 * Pure function to build row descriptors from messages.
 * Groups consecutive tool_call/tool_result messages into tool_group rows.
 * Adds a processing indicator row when streaming.
 */
export function buildRows(messages: UIMessage[], isStreaming: boolean): RowDescriptor[] {
  const rows: RowDescriptor[] = [];
  let i = 0;

  while (i < messages.length) {
    const msg = messages[i];

    if (msg.role === "tool_call" || msg.role === "tool_result") {
      // Collect consecutive tool messages into a group
      const tools: UIMessage[] = [];
      while (
        i < messages.length &&
        (messages[i].role === "tool_call" || messages[i].role === "tool_result")
      ) {
        tools.push(messages[i]);
        i++;
      }
      rows.push({ kind: "tool_group", tools });
      continue;
    }

    rows.push({ kind: "message", msg });
    i++;
  }

  if (isStreaming) {
    // Only show processing if no message is currently streaming content
    const hasActiveStream = messages.some(
      m => m.role === "assistant" && m.isStreaming && (m.content || m.thinking)
    );
    if (!hasActiveStream) {
      rows.push(PROCESSING_ROW);
    }
  }

  return rows;
}

function getRowKey(row: RowDescriptor, index: number): string {
  if (row.kind === "processing") return "__processing__";
  if (row.kind === "tool_group") return `tg-${row.tools[0]?.id ?? index}`;
  return row.msg.id;
}

function estimateRowHeight(row: RowDescriptor): number {
  if (row.kind === "processing") return 44;
  if (row.kind === "tool_group") return 36 * mergeToolPairs(row.tools).length;
  const msg = row.msg;
  if (msg.role === "user") return Math.max(56, Math.min(200, 56 + msg.content.length / 3));
  if (msg.role === "assistant") return Math.max(56, Math.min(600, 56 + msg.content.length / 2));
  return 48;
}

/**
 * Merge consecutive tool_call + tool_result pairs into single UIMessages.
 * Each tool_call gets its matching tool_result (by id) merged onto it.
 * Only tool_call messages are returned; tool_result-only messages are consumed.
 */
export function mergeToolPairs(tools: UIMessage[]): UIMessage[] {
  const resultMap = new Map<string, UIMessage>();
  for (const t of tools) {
    if (t.role === "tool_result") {
      resultMap.set(t.id, t);
    }
  }

  const merged: UIMessage[] = [];
  for (const t of tools) {
    if (t.role !== "tool_call") continue;
    const result = resultMap.get(t.id);
    if (result) {
      merged.push({
        ...t,
        toolResult: result.toolResult,
        toolError: result.toolError,
      });
    } else {
      merged.push(t);
    }
  }

  return merged;
}

// ── ChatMessageRow ──

interface ChatMessageRowProps {
  row: RowDescriptor;
  agentName?: string;
}

const ChatMessageRow = memo(
  function ChatMessageRow({ row, agentName }: ChatMessageRowProps) {
    if (row.kind === "processing") {
      return <ProcessingIndicator />;
    }

    if (row.kind === "tool_group") {
      const cards = mergeToolPairs(row.tools);
      return (
        <div className="space-y-1 px-4 py-1" data-testid="tool-group">
          {cards.map(tool => (
            <ToolCallCard key={tool.id} message={tool} />
          ))}
        </div>
      );
    }

    return <MessageBubble message={row.msg} agentName={agentName} />;
  },
  (prev, next) => prev.row === next.row && prev.agentName === next.agentName
);

// ── ChatView ──

export interface ChatViewProps {
  messages: UIMessage[];
  isStreaming: boolean;
  agentName?: string;
}

export const ChatView = memo(function ChatView({
  messages,
  isStreaming,
  agentName,
}: ChatViewProps) {
  if (messages.length === 0 && !isStreaming) {
    return (
      <div className="flex flex-1 items-center justify-center" data-testid="chat-empty-state">
        <div className="flex flex-col items-center gap-3">
          <MessageSquare className="size-8 text-[color:var(--color-text-tertiary)]/30" />
          <p className="text-sm italic text-[color:var(--color-text-tertiary)]">
            Send a message to start the conversation
          </p>
        </div>
      </div>
    );
  }

  return <ChatViewContent messages={messages} isStreaming={isStreaming} agentName={agentName} />;
});

function ChatViewContent({ messages, isStreaming, agentName }: ChatViewProps) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const bottomLockedRef = useRef(true);
  const userScrollIntentRef = useRef(0);
  const [showScrollButton, setShowScrollButton] = useState(false);

  const rows = useMemo(() => buildRows(messages, isStreaming), [messages, isStreaming]);

  const virtualizer = useVirtualizer({
    count: rows.length,
    getScrollElement: () => scrollRef.current,
    estimateSize: i => estimateRowHeight(rows[i]),
    overscan: 10,
    getItemKey: i => getRowKey(rows[i], i),
  });

  // ── Bottom-lock scroll ──

  const scrollToBottom = useCallback(() => {
    const el = scrollRef.current;
    if (el) {
      el.scrollTop = el.scrollHeight;
      bottomLockedRef.current = true;
      setShowScrollButton(false);
    }
  }, []);

  const scheduleFollowBottom = useCallback(() => {
    if (!bottomLockedRef.current) return;
    requestAnimationFrame(() => {
      const el = scrollRef.current;
      if (el && bottomLockedRef.current) {
        el.scrollTop = el.scrollHeight;
      }
    });
  }, []);

  // Follow bottom on new rows
  useEffect(() => {
    scheduleFollowBottom();
  }, [rows.length, scheduleFollowBottom]);

  // Follow bottom during streaming content growth
  useEffect(() => {
    if (!isStreaming) return;
    const el = scrollRef.current;
    if (!el) return;

    const observer = new ResizeObserver(() => {
      scheduleFollowBottom();
    });
    const inner = el.firstElementChild;
    if (inner) observer.observe(inner);
    return () => observer.disconnect();
  }, [isStreaming, scheduleFollowBottom]);

  // Initial scroll to bottom
  useLayoutEffect(() => {
    const el = scrollRef.current;
    if (el) {
      el.scrollTop = el.scrollHeight;
    }
  }, []);

  const markUserIntent = useCallback(() => {
    userScrollIntentRef.current = Date.now() + 250;
  }, []);

  const handleScroll = useCallback(() => {
    const el = scrollRef.current;
    if (!el) return;
    const { scrollTop, scrollHeight, clientHeight } = el;
    const distanceFromBottom = scrollHeight - scrollTop - clientHeight;
    const hasRecentUserIntent = Date.now() <= userScrollIntentRef.current;

    if (hasRecentUserIntent && distanceFromBottom > BOTTOM_LOCK_THRESHOLD_PX) {
      bottomLockedRef.current = false;
      setShowScrollButton(true);
    } else if (distanceFromBottom <= BOTTOM_LOCK_THRESHOLD_PX) {
      bottomLockedRef.current = true;
      setShowScrollButton(false);
    }
  }, []);

  // Passive wheel/touch listeners
  useEffect(() => {
    const el = scrollRef.current;
    if (!el) return;
    el.addEventListener("wheel", markUserIntent, { passive: true });
    el.addEventListener("touchmove", markUserIntent, { passive: true });
    return () => {
      el.removeEventListener("wheel", markUserIntent);
      el.removeEventListener("touchmove", markUserIntent);
    };
  }, [markUserIntent]);

  return (
    <div className="relative flex flex-1 flex-col overflow-hidden" data-testid="chat-view">
      <div
        ref={scrollRef}
        className="min-h-0 flex-1 overflow-y-auto"
        style={{ overscrollBehaviorY: "contain" }}
        onScroll={handleScroll}
        onPointerDown={markUserIntent}
      >
        <div
          style={{
            height: `${virtualizer.getTotalSize()}px`,
            width: "100%",
            position: "relative",
          }}
        >
          {virtualizer.getVirtualItems().map(virtualRow => {
            const row = rows[virtualRow.index];
            return (
              <div
                key={virtualRow.key}
                ref={virtualizer.measureElement}
                data-index={virtualRow.index}
                style={{
                  position: "absolute",
                  top: 0,
                  left: 0,
                  width: "100%",
                  transform: `translateY(${virtualRow.start}px)`,
                }}
              >
                <ChatMessageRow row={row} agentName={agentName} />
              </div>
            );
          })}
        </div>
      </div>

      {showScrollButton && (
        <div className="absolute bottom-2 left-1/2 -translate-x-1/2">
          <Button
            variant="secondary"
            size="sm"
            onClick={scrollToBottom}
            className={cn("border border-[color:var(--color-divider)]")}
            data-testid="scroll-to-bottom"
          >
            <ArrowDown className="size-3.5" />
            <span className="text-xs">Scroll to bottom</span>
          </Button>
        </div>
      )}
    </div>
  );
}
