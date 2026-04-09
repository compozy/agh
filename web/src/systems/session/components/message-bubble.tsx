import { lazy, memo, Suspense } from "react";

import { cn } from "@/lib/utils";
import type { UIMessage } from "../types";
import { ThinkingBlock } from "./thinking-block";

export interface MessageBubbleProps {
  message: UIMessage;
  agentName?: string;
}

const LazyMessageMarkdown = lazy(() =>
  import("./message-markdown").then(module => ({ default: module.MessageMarkdown }))
);

function formatTimestamp(ts: number): string {
  const d = new Date(ts);
  return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}

export const MessageBubble = memo(
  function MessageBubble({ message, agentName }: MessageBubbleProps) {
    const isUser = message.role === "user";

    if (isUser) {
      return (
        <div
          className="flex justify-end px-4 py-2"
          data-testid="message-bubble-user"
          data-message-id={message.id}
        >
          <div
            className={cn(
              "max-w-[85%] rounded-xl px-5 py-4",
              "bg-[color:var(--color-surface-elevated)]"
            )}
            data-testid="user-bubble"
          >
            {message.content && (
              <div
                className={cn(
                  "prose prose-sm prose-invert max-w-none",
                  "prose-p:my-1 prose-headings:mb-2 prose-headings:mt-4",
                  "prose-ul:my-1 prose-ol:my-1 prose-li:my-0.5",
                  "prose-pre:my-0 prose-pre:bg-transparent prose-pre:p-0",
                  "text-sm leading-relaxed text-[color:var(--color-text-primary)]"
                )}
              >
                <Suspense fallback={<span className="whitespace-pre-wrap">{message.content}</span>}>
                  <LazyMessageMarkdown content={message.content} />
                </Suspense>
              </div>
            )}
          </div>
        </div>
      );
    }

    return (
      <div
        className="px-4 py-2"
        data-testid="message-bubble-assistant"
        data-message-id={message.id}
      >
        {/* Agent label row */}
        <div className="mb-1.5 flex items-center gap-2" data-testid="agent-label">
          <span className="size-1.5 rounded-full bg-[color:var(--color-success)]" />
          <span className="font-mono text-[11px] font-medium uppercase tracking-wider text-[color:var(--color-text-tertiary)]">
            {agentName ?? "Agent"}
          </span>
          <span className="text-[11px] text-[color:var(--color-text-tertiary)]">
            {formatTimestamp(message.timestamp)}
          </span>
        </div>

        {message.thinking && (
          <ThinkingBlock thinking={message.thinking} thinkingComplete={message.thinkingComplete} />
        )}

        {message.content && (
          <div
            className={cn(
              "prose prose-sm prose-invert max-w-none",
              "prose-p:my-1 prose-headings:mb-2 prose-headings:mt-4",
              "prose-ul:my-1 prose-ol:my-1 prose-li:my-0.5",
              "prose-pre:my-0 prose-pre:bg-transparent prose-pre:p-0",
              "text-sm leading-relaxed text-[color:var(--color-text-secondary)]"
            )}
          >
            <Suspense fallback={<span className="whitespace-pre-wrap">{message.content}</span>}>
              <LazyMessageMarkdown content={message.content} />
            </Suspense>
          </div>
        )}

        {!message.content && message.isStreaming && (
          <span className="text-xs italic text-[color:var(--color-text-tertiary)]">...</span>
        )}
      </div>
    );
  },
  (prev, next) =>
    prev.message.id === next.message.id &&
    prev.message.content === next.message.content &&
    prev.message.thinking === next.message.thinking &&
    prev.message.thinkingComplete === next.message.thinkingComplete &&
    prev.message.isStreaming === next.message.isStreaming &&
    prev.agentName === next.agentName
);
