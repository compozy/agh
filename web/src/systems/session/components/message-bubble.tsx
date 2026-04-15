import { memo, useCallback, useEffect, useRef, useState } from "react";
import { Check, Copy } from "lucide-react";

import { cn } from "@/lib/utils";
import { MessageMarkdown } from "@/systems/session/components/message-markdown";
import type { UIMessage } from "../types";
import { ThinkingBlock } from "./thinking-block";

export interface MessageBubbleProps {
  message: UIMessage;
  agentName?: string;
}

const messageProseClasses =
  "prose prose-sm prose-invert max-w-none prose-p:my-1 prose-headings:mb-2 prose-headings:mt-4 prose-ul:my-1 prose-ol:my-1 prose-li:my-0.5 prose-pre:my-0 prose-pre:bg-transparent prose-pre:p-0";

function formatTimestamp(ts: number): string {
  if (!Number.isFinite(ts) || ts <= 0) {
    return "";
  }

  const d = new Date(ts);
  if (Number.isNaN(d.getTime())) {
    return "";
  }

  return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}

const COPY_RESET_MS = 1200;

function MessageCopyButton({ text }: { text: string }) {
  const [copied, setCopied] = useState(false);
  const timerRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  useEffect(() => {
    return () => clearTimeout(timerRef.current);
  }, []);

  const handleCopy = useCallback(() => {
    void navigator.clipboard.writeText(text);
    setCopied(true);
    clearTimeout(timerRef.current);
    timerRef.current = setTimeout(() => setCopied(false), COPY_RESET_MS);
  }, [text]);

  return (
    <button
      type="button"
      onClick={handleCopy}
      className={cn(
        "rounded-md p-1",
        "opacity-0 transition-opacity duration-200",
        "group-hover/msgbubble:opacity-100",
        "text-[color:var(--color-text-tertiary)] hover:text-[color:var(--color-text-primary)]"
      )}
      aria-label="Copy message"
    >
      {copied ? <Check className="size-3" /> : <Copy className="size-3" />}
    </button>
  );
}

export const MessageBubble = memo(
  function MessageBubble({ message, agentName }: MessageBubbleProps) {
    const isUser = message.role === "user";
    const timestamp = formatTimestamp(message.timestamp);

    if (isUser) {
      return (
        <div
          className="flex justify-end px-4 py-2"
          data-testid="message-bubble-user"
          data-message-id={message.id}
        >
          <div
            className={cn(
              "group/msgbubble relative max-w-[85%] rounded-2xl rounded-br-sm px-5 py-4",
              "border border-[color:var(--color-divider)] bg-[color:var(--color-surface-elevated)]"
            )}
            data-testid="user-bubble"
          >
            {message.content && (
              <div
                className={cn(
                  messageProseClasses,
                  "text-sm leading-relaxed text-[color:var(--color-text-primary)]"
                )}
              >
                <MessageMarkdown content={message.content} />
              </div>
            )}
            <div className="mt-1.5 flex items-center justify-end gap-2">
              <MessageCopyButton text={message.content} />
              {timestamp ? (
                <span className="text-[10px] text-[color:var(--color-text-tertiary)]/50">
                  {timestamp}
                </span>
              ) : null}
            </div>
          </div>
        </div>
      );
    }

    return (
      <div
        className="group/msgbubble px-4 py-2"
        data-testid="message-bubble-assistant"
        data-message-id={message.id}
      >
        {/* Agent label row */}
        <div className="mb-1.5 flex items-center gap-2" data-testid="agent-label">
          <span className="size-1.5 rounded-full bg-[color:var(--color-success)]" />
          <span className="font-mono text-[11px] font-medium uppercase tracking-wider text-[color:var(--color-text-tertiary)]">
            {agentName ?? "Agent"}
          </span>
          {timestamp ? (
            <span className="text-[11px] text-[color:var(--color-text-tertiary)]">{timestamp}</span>
          ) : null}
        </div>

        {message.thinking && (
          <ThinkingBlock thinking={message.thinking} thinkingComplete={message.thinkingComplete} />
        )}

        {message.content && (
          <div
            className={cn(
              messageProseClasses,
              "text-sm leading-relaxed text-[color:var(--color-text-secondary)]"
            )}
          >
            <MessageMarkdown content={message.content} />
          </div>
        )}

        {!message.content && message.isStreaming && (
          <span className="text-xs italic text-[color:var(--color-text-tertiary)]">...</span>
        )}

        {message.content && (
          <div className="mt-1.5 flex items-center gap-2">
            <MessageCopyButton text={message.content} />
          </div>
        )}
      </div>
    );
  },
  (prev, next) =>
    prev.message.role === next.message.role &&
    prev.message.id === next.message.id &&
    prev.message.content === next.message.content &&
    prev.message.thinking === next.message.thinking &&
    prev.message.thinkingComplete === next.message.thinkingComplete &&
    prev.message.timestamp === next.message.timestamp &&
    prev.message.isStreaming === next.message.isStreaming &&
    prev.agentName === next.agentName
);
