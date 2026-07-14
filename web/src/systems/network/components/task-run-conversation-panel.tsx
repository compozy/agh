import { Spinner } from "@agh/ui";

import { useNetworkUsage } from "../hooks/use-network-coordination";

export interface TaskRunConversationPanelProps {
  conversationEmpty: boolean;
  conversationLoading?: boolean;
  hasMoreMessages?: boolean;
  isFetchingMore?: boolean;
  onLoadMore?: () => void;
  messageCount: number;
  boundsLabel?: string | null;
}

/**
 * Run conversation + bounds/usage surface (UT-057).
 * Empty explains silence; pagination keeps the run view interactive.
 */
export function TaskRunConversationPanel({
  conversationEmpty,
  conversationLoading = false,
  hasMoreMessages = false,
  isFetchingMore = false,
  onLoadMore,
  messageCount,
  boundsLabel,
}: TaskRunConversationPanelProps) {
  const usage = useNetworkUsage();
  const total = usage.data?.total;
  const usageLabel =
    total === undefined
      ? null
      : total.unavailable_wake_count > 0 && total.actual_wake_count === 0
        ? "usage_unavailable"
        : "actual";

  return (
    <section
      className="flex flex-col gap-3 rounded-md border border-border px-3 py-3"
      data-testid="tasks-run-conversation-panel"
    >
      <header className="flex items-baseline justify-between gap-3">
        <h2 className="text-sm font-medium text-foreground">Coordination conversation</h2>
        {boundsLabel ? (
          <p className="text-xs text-muted" data-testid="tasks-run-bounds-label">
            {boundsLabel}
          </p>
        ) : null}
      </header>

      {conversationLoading ? (
        <div className="flex items-center gap-2 text-sm text-muted" role="status">
          <Spinner className="size-4" />
          Loading conversation…
        </div>
      ) : conversationEmpty ? (
        <p className="text-sm text-muted" data-testid="tasks-run-conversation-empty">
          No coordination messages yet. Silence is normal until workers post updates on this run.
        </p>
      ) : (
        <p className="text-sm text-muted" data-testid="tasks-run-conversation-summary">
          {messageCount} message{messageCount === 1 ? "" : "s"} in this run conversation.
        </p>
      )}

      {hasMoreMessages && onLoadMore ? (
        <button
          className="self-start text-xs text-action underline-offset-2 hover:underline"
          data-testid="tasks-run-conversation-load-more"
          disabled={isFetchingMore}
          onClick={onLoadMore}
          type="button"
        >
          {isFetchingMore ? "Loading…" : "Load earlier messages"}
        </button>
      ) : null}

      <div
        className="border-t border-border pt-2 text-xs text-muted"
        data-testid="tasks-run-usage-summary"
      >
        {usage.isLoading
          ? "Loading workspace usage…"
          : usageLabel && total
            ? `Workspace usage (${usageLabel}): ${total.wake_count} wakes · ${total.input_tokens} in / ${total.output_tokens} out`
            : "Workspace usage unavailable."}
      </div>
    </section>
  );
}
