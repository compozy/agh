import { useThreadOverlay } from "../../hooks/use-thread-overlay";
import { ThreadOverlayHeader } from "./thread-overlay-header";
import { ThreadOverlayReplies } from "./thread-overlay-replies";
import { ThreadOverlayRoot } from "./thread-overlay-root";

export interface ThreadOverlayProps {
  channel: string;
  threadId: string;
  /** Render in full-page mode (no fixed width / no border-left) for `<1024px` or `?view=full`. */
  fullPage?: boolean;
}

export function ThreadOverlay({ channel, threadId, fullPage = false }: ThreadOverlayProps) {
  const overlay = useThreadOverlay({ channel, fullPage, threadId });

  return (
    <section
      aria-label={fullPage ? "Thread" : "Thread overlay"}
      className="flex min-h-0 flex-1 flex-col bg-[color:var(--color-canvas-deep)]"
      data-fullpage={fullPage ? "true" : "false"}
      data-testid="network-thread-overlay"
    >
      <ThreadOverlayHeader channel={channel} detail={overlay.detail} threadId={threadId} />
      <ThreadOverlayRoot isLoading={overlay.isDetailLoading} rootMessage={overlay.rootMessage} />
      <ThreadOverlayReplies
        isLoading={overlay.isMessagesLoading}
        lastReadAt={overlay.lastReadIso}
        messages={overlay.replies}
        replyCount={overlay.replyCount}
      />
    </section>
  );
}
