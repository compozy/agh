import type { ReactNode } from "react";

import { Separator } from "@agh/ui";

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
      <Separator
        className="px-4 py-3"
        data-testid="network-thread-overlay-replies-divider"
        label={replyLabel}
        labelClassName="text-badge"
      />
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
