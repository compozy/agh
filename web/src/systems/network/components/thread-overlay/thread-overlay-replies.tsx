import type { ReactNode } from "react";

import type { NetworkConversationMessage } from "../../types";
import { Timeline } from "../timeline/timeline";

export interface ThreadOverlayRepliesProps {
  messages: ReadonlyArray<NetworkConversationMessage>;
  isLoading: boolean;
  /** Reply count from thread detail (excludes the root message). */
  replyCount: number;
  lastReadAt?: string | null;
  now?: Date;
  /** Override the default empty placeholder (used to render `ThreadEmpty`). */
  emptyOverride?: ReactNode;
  onRetryOptimistic?: (message: NetworkConversationMessage) => void;
  onDiscardOptimistic?: (message: NetworkConversationMessage) => void;
  onWorkChipClick?: (message: NetworkConversationMessage) => void;
}

export function ThreadOverlayReplies({
  messages,
  isLoading,
  replyCount,
  lastReadAt,
  now,
  emptyOverride,
  onRetryOptimistic,
  onDiscardOptimistic,
  onWorkChipClick,
}: ThreadOverlayRepliesProps) {
  const replyLabel = replyCount === 1 ? "1 reply" : `${replyCount} replies`;

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div
        className="flex items-center gap-3 px-4 py-3"
        data-testid="network-thread-overlay-replies-divider"
      >
        <span aria-hidden="true" className="h-px flex-1 bg-(--color-divider)" />
        <span className="font-mono text-badge uppercase tracking-mono text-(--color-text-tertiary)">
          {replyLabel}
        </span>
        <span aria-hidden="true" className="h-px flex-1 bg-(--color-divider)" />
      </div>
      <Timeline
        ariaLabel="Thread replies"
        density="overlay"
        emptyState={
          emptyOverride ?? (
            <p className="text-center text-xs text-(--color-text-tertiary)">
              Thread has no replies.
            </p>
          )
        }
        isLoading={isLoading}
        lastReadAt={lastReadAt}
        messages={messages}
        now={now}
        onDiscardOptimistic={onDiscardOptimistic}
        onRetryOptimistic={onRetryOptimistic}
        onWorkChipClick={onWorkChipClick}
      />
    </div>
  );
}
