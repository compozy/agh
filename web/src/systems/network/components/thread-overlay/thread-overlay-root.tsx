import type { NetworkConversationMessage } from "../../types";
import { MessageRow } from "../timeline/message-row";

export interface ThreadOverlayRootProps {
  rootMessage: NetworkConversationMessage | null;
  isLoading: boolean;
}

export function ThreadOverlayRoot({ rootMessage, isLoading }: ThreadOverlayRootProps) {
  if (isLoading && !rootMessage) {
    return (
      <div
        aria-label="Loading thread root"
        className="flex flex-col gap-2 px-4 py-3"
        data-testid="network-thread-overlay-root-loading"
        role="status"
      >
        <span className="font-mono text-[10px] uppercase tracking-[0.06em] text-[color:var(--color-text-tertiary)]">
          Loading root
        </span>
      </div>
    );
  }

  if (!rootMessage) {
    return null;
  }

  return (
    <div className="flex flex-col gap-1 border-b border-[color:var(--color-divider)] py-2">
      <span
        className="px-4 font-mono text-[10px] uppercase tracking-[0.06em] text-[color:var(--color-text-tertiary)]"
        data-testid="network-thread-overlay-root-badge"
      >
        ROOT
      </span>
      <MessageRow density="overlay" message={rootMessage} />
    </div>
  );
}
