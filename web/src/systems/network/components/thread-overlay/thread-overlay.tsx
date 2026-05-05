import { DetailComposer } from "../composer/detail-composer";
import { ThreadEmpty } from "../empty-states/thread-empty";
import { WorkBanner } from "../work/work-banner";
import { ThreadOverlayHeader } from "./thread-overlay-header";
import { ThreadOverlayReplies } from "./thread-overlay-replies";
import { ThreadOverlayRoot } from "./thread-overlay-root";
import { useThreadOverlayView } from "./use-thread-overlay-view";

export interface ThreadOverlayProps {
  channel: string;
  threadId: string;
  /** Render in full-page mode (no fixed width / no border-left) for `<1024px` or `?view=full`. */
  fullPage?: boolean;
}

export function ThreadOverlay({ channel, threadId, fullPage = false }: ThreadOverlayProps) {
  const view = useThreadOverlayView({ channel, fullPage, threadId });
  const { overlay, session, disabledReason, openWork, handleRetry, handleDiscard } = view;

  return (
    <section
      aria-label={fullPage ? "Thread" : "Thread overlay"}
      className="flex min-h-0 flex-1 flex-col bg-[color:var(--color-canvas-deep)]"
      data-fullpage={fullPage ? "true" : "false"}
      data-testid="network-thread-overlay"
    >
      <ThreadOverlayHeader channel={channel} detail={overlay.detail} threadId={threadId} />
      <WorkBanner hasNeedsInput={openWork.hasNeedsInput} openCount={openWork.openCount} />
      <ThreadOverlayRoot isLoading={overlay.isDetailLoading} rootMessage={overlay.rootMessage} />
      <ThreadOverlayReplies
        emptyOverride={<ThreadEmpty />}
        isLoading={overlay.isMessagesLoading}
        lastReadAt={overlay.lastReadIso}
        messages={overlay.replies}
        onDiscardOptimistic={handleDiscard}
        onRetryOptimistic={handleRetry}
        replyCount={overlay.replyCount}
      />

      <DetailComposer
        channel={channel}
        disabledReason={disabledReason ?? undefined}
        displayName={session?.displayName}
        peerFrom={session?.peerId ?? ""}
        sessionId={session?.sessionId ?? ""}
        surface="thread"
        threadId={threadId}
      />
    </section>
  );
}
