import { memo } from "react";

import { ChatMessageBubble, CodeBlock, StatusDot } from "@agh/ui";

import { cn } from "@/lib/utils";
import { MessageMarkdown } from "@/systems/session/components/message-markdown";
import type { UIMessage } from "../types";
import { CopyButton } from "./copy-button";
import { ThinkingBlock } from "./thinking-block";

export interface MessageBubbleProps {
  message: UIMessage;
  agentName?: string;
}

const copyButtonClasses = cn(
  "rounded-md p-1 text-[color:var(--color-text-tertiary)]",
  "opacity-0 transition-opacity duration-200",
  "group-hover/msgbubble:opacity-100 hover:text-[color:var(--color-text-primary)]"
);

function formatTimestamp(ts: number): string {
  if (!Number.isFinite(ts) || ts <= 0) return "";
  const d = new Date(ts);
  if (Number.isNaN(d.getTime())) return "";
  return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}

function formatDiffSummary(additions?: number, removals?: number): string {
  const parts: string[] = [];
  if (typeof additions === "number") parts.push(`+${additions}`);
  if (typeof removals === "number") parts.push(`−${removals}`);
  return parts.join(" ");
}

export const MessageBubble = memo(
  function MessageBubble({ message, agentName }: MessageBubbleProps) {
    const timestamp = formatTimestamp(message.timestamp);

    if (message.role === "user") {
      const showMetaRow = Boolean(message.content) || Boolean(timestamp);
      return (
        <div className="group/msgbubble px-4 py-2" data-testid="message-bubble-user-wrapper">
          <ChatMessageBubble
            role="user"
            data-testid="message-bubble-user"
            data-message-id={message.id}
            meta={
              <span className="inline-flex items-center gap-2">
                <span>YOU</span>
                {timestamp ? (
                  <span className="font-normal tracking-normal normal-case text-[color:var(--color-text-tertiary)]/80">
                    · {timestamp}
                  </span>
                ) : null}
              </span>
            }
          >
            <div data-testid="user-bubble" data-slot="user-bubble-body">
              {message.content ? (
                <div className="prose prose-sm prose-invert max-w-none prose-p:my-1 prose-headings:mb-2 prose-headings:mt-4 prose-ul:my-1 prose-ol:my-1 prose-li:my-0.5 prose-pre:my-0 prose-pre:bg-transparent prose-pre:p-0">
                  <MessageMarkdown content={message.content} />
                </div>
              ) : null}
              {showMetaRow && message.content ? (
                <div className="mt-2 flex items-center justify-end gap-2">
                  <CopyButton
                    text={message.content}
                    ariaLabel="Copy message"
                    className={copyButtonClasses}
                  />
                </div>
              ) : null}
            </div>
          </ChatMessageBubble>
        </div>
      );
    }

    if (message.role === "system") {
      return (
        <div className="px-4 py-2">
          <ChatMessageBubble
            role="system"
            data-testid="message-bubble-system"
            data-message-id={message.id}
          >
            {message.content}
          </ChatMessageBubble>
        </div>
      );
    }

    if (message.role === "diff") {
      const diff = message.diff;
      const summary = formatDiffSummary(diff?.additions, diff?.removals);
      return (
        <div className="px-4 py-2">
          <ChatMessageBubble
            role="diff"
            data-testid="message-bubble-diff"
            data-message-id={message.id}
            meta={
              diff?.path || summary ? (
                <span className="flex min-w-0 items-center gap-2">
                  {diff?.path ? (
                    <span
                      data-slot="chat-message-diff-path"
                      className="min-w-0 truncate text-[color:var(--color-text-primary)]"
                    >
                      {diff.path}
                    </span>
                  ) : null}
                  {summary ? (
                    <span
                      data-slot="chat-message-diff-summary"
                      className="ml-auto font-mono text-[11px] text-[color:var(--color-text-tertiary)]"
                    >
                      {summary}
                    </span>
                  ) : null}
                </span>
              ) : undefined
            }
          >
            <CodeBlock
              code={diff?.content ?? ""}
              language={diff?.language}
              showPrompt={false}
              copyable={Boolean(diff?.content)}
              data-testid="message-bubble-diff-code"
            />
          </ChatMessageBubble>
        </div>
      );
    }

    // assistant (agent)
    return (
      <div className="group/msgbubble px-4 py-2" data-testid="message-bubble-assistant-wrapper">
        <ChatMessageBubble
          role="agent"
          data-testid="message-bubble-assistant"
          data-message-id={message.id}
          meta={
            <span
              className="inline-flex items-center gap-2"
              data-testid="agent-label"
              data-agent-name={agentName ?? "agent"}
            >
              <StatusDot
                size="md"
                tone={message.isStreaming ? "accent" : "success"}
                pulse={Boolean(message.isStreaming)}
                data-testid="agent-status-dot"
              />
              <span className="font-mono">{agentName ?? "Agent"}</span>
              {timestamp ? (
                <span className="font-mono font-normal normal-case tracking-[0.04em] text-[color:var(--color-text-tertiary)]">
                  · {timestamp}
                </span>
              ) : null}
            </span>
          }
        >
          {message.thinking ? (
            <ThinkingBlock
              thinking={message.thinking}
              thinkingComplete={message.thinkingComplete}
            />
          ) : null}
          {message.content ? (
            <div className="prose prose-sm prose-invert max-w-none prose-p:my-1 prose-headings:mb-2 prose-headings:mt-4 prose-ul:my-1 prose-ol:my-1 prose-li:my-0.5 prose-pre:my-0 prose-pre:bg-transparent prose-pre:p-0 text-[color:var(--color-text-secondary)]">
              <MessageMarkdown content={message.content} />
            </div>
          ) : null}
          {!message.content && message.isStreaming ? (
            <span className="text-xs italic text-[color:var(--color-text-tertiary)]">...</span>
          ) : null}
          {message.content ? (
            <div className="mt-1.5 flex items-center gap-2">
              <CopyButton
                text={message.content}
                ariaLabel="Copy message"
                className={copyButtonClasses}
              />
            </div>
          ) : null}
        </ChatMessageBubble>
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
    prev.message.diff === next.message.diff &&
    prev.agentName === next.agentName
);
