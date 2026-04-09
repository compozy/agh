import { lazy, memo, Suspense } from "react";
import { User, Bot } from "lucide-react";

import { cn } from "@/lib/utils";
import type { UIMessage } from "../types";
import { ThinkingBlock } from "./thinking-block";

export interface MessageBubbleProps {
  message: UIMessage;
}

const LazyMessageMarkdown = lazy(() =>
  import("./message-markdown").then(module => ({ default: module.MessageMarkdown }))
);

export const MessageBubble = memo(
  function MessageBubble({ message }: MessageBubbleProps) {
    const isUser = message.role === "user";

    return (
      <div
        className={cn("flex gap-3 px-4 py-3", isUser && "flex-row-reverse")}
        data-testid={`message-bubble-${message.role}`}
        data-message-id={message.id}
      >
        <div
          className={cn(
            "flex size-7 shrink-0 items-center justify-center rounded-full",
            isUser
              ? "bg-[color:var(--color-accent)]/10 text-[color:var(--color-accent)]"
              : "bg-[color:var(--color-info)]/10 text-[color:var(--color-info)]"
          )}
        >
          {isUser ? <User className="size-3.5" /> : <Bot className="size-3.5" />}
        </div>

        <div className={cn("min-w-0 max-w-[85%] flex-1", isUser && "flex flex-col items-end")}>
          {message.thinking && (
            <ThinkingBlock
              thinking={message.thinking}
              thinkingComplete={message.thinkingComplete}
            />
          )}

          {message.content && (
            <div
              className={cn(
                "prose prose-sm prose-invert max-w-none",
                "prose-p:my-1 prose-headings:mb-2 prose-headings:mt-4",
                "prose-ul:my-1 prose-ol:my-1 prose-li:my-0.5",
                "prose-pre:my-0 prose-pre:bg-transparent prose-pre:p-0",
                "text-sm leading-relaxed",
                isUser
                  ? "text-[color:var(--color-text-primary)]"
                  : "text-[color:var(--color-text-secondary)]"
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
      </div>
    );
  },
  (prev, next) =>
    prev.message.id === next.message.id &&
    prev.message.content === next.message.content &&
    prev.message.thinking === next.message.thinking &&
    prev.message.thinkingComplete === next.message.thinkingComplete &&
    prev.message.isStreaming === next.message.isStreaming
);
