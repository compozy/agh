import { MessagesSquare } from "lucide-react";

import { PillDot } from "@agh/ui";

import type { LoopChannelMessage } from "../../lib/loop-events";

interface LoopRunChannelProps {
  messages: LoopChannelMessage[];
  /** Channel label (`#delivery-r8f3a`); defaults to a neutral placeholder. */
  channelLabel?: string;
  /** True while the run is live (pulse the streaming indicator). */
  isLive?: boolean;
}

/** Two-letter avatar seed for a channel author. */
function avatarSeed(author: string): string {
  return (
    author
      .replace(/[^a-z0-9]/gi, "")
      .slice(0, 2)
      .toLowerCase() || "ag"
  );
}

/**
 * The embedded converse-and-decide channel (§4.4 / ADR-014): the transcript of an
 * `agh__network_send` action's channel, ending in the harvested `result` message. Fed
 * by `channel_msg` SSE frames; empty until the daemon streams them (truthful — no
 * fabricated transcript).
 */
export function LoopRunChannel({ messages, channelLabel, isLive }: LoopRunChannelProps) {
  return (
    <div
      className="mt-2 overflow-hidden rounded-md border border-line bg-rail"
      data-testid="loop-run-channel"
    >
      <div className="flex items-center gap-2 border-b border-line-soft px-3 py-2 text-form-hint text-subtle">
        <MessagesSquare className="size-3.5" aria-hidden="true" />
        <span className="font-mono text-mono-id text-muted">{channelLabel ?? "channel"}</span>
        {isLive ? (
          <span className="ml-auto inline-flex items-center gap-1.5 text-badge font-medium text-accent-strong">
            <PillDot tone="accent" pulse size="sm" />
            streaming
          </span>
        ) : null}
      </div>
      {messages.length === 0 ? (
        <p className="px-3 py-3 text-form-hint text-subtle" data-testid="loop-run-channel-empty">
          No channel messages yet. Converse-and-decide messages appear here as the loop posts them.
        </p>
      ) : (
        <div className="flex flex-col">
          {messages.map(message => (
            <div
              key={message.id}
              className="flex gap-2.5 border-t border-line-soft px-3 py-2.5 first:border-t-0"
              data-testid="loop-run-channel-message"
              data-result={message.isResult ? "true" : undefined}
            >
              <span
                aria-hidden="true"
                className="grid size-5 shrink-0 place-items-center rounded-xs bg-badge-fill font-mono text-pill-group-badge font-semibold text-muted"
              >
                {avatarSeed(message.author)}
              </span>
              <div className="min-w-0">
                <div className="text-form-hint font-semibold text-fg">
                  {message.author}
                  {message.role ? (
                    <span className="ml-1.5 text-badge font-normal text-faint">{message.role}</span>
                  ) : null}
                </div>
                <div
                  className={
                    message.isResult
                      ? "mt-1 rounded-sm border border-success/25 bg-success-tint px-2.5 py-1.5 text-form-label leading-relaxed text-fg"
                      : "text-form-label leading-relaxed text-muted"
                  }
                >
                  {message.isResult ? (
                    <b className="font-semibold text-success">result · </b>
                  ) : null}
                  {message.text}
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
