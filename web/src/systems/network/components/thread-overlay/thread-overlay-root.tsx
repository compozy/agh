import { Eyebrow } from "@agh/ui";

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
        <Eyebrow>Loading root</Eyebrow>
      </div>
    );
  }

  if (!rootMessage) {
    return null;
  }

  return (
    <div className="flex flex-col gap-1 border-b border-line py-2">
      <Eyebrow className="px-4" data-testid="network-thread-overlay-root-badge">
        ROOT
      </Eyebrow>
      <MessageRow density="overlay" message={rootMessage} />
    </div>
  );
}
