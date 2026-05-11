import { DetailComposer } from "../composer/detail-composer";
import { ConversationError } from "../empty-states/conversation-error";
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
  const detailError = overlay.detailError;
  const isResolvingDetail = !detailError && !overlay.detail;

  return (
    <section
      aria-label={fullPage ? "Thread" : "Thread overlay"}
      className="flex min-h-0 flex-1 flex-col bg-canvas"
      data-fullpage={fullPage ? "true" : "false"}
      data-testid="network-thread-overlay"
    >
      <ThreadOverlayHeader channel={channel} detail={overlay.detail} threadId={threadId} />
      {detailError ? (
        <div className="flex flex-1 items-center justify-center px-5 py-10" role="alert">
          <ConversationError
            description={`Could not load thread ${threadId}. Choose an existing thread from #${channel}.`}
            testId="network-thread-overlay-error"
            title="Thread unavailable"
          />
        </div>
      ) : isResolvingDetail ? (
        <>
          <ThreadOverlayRoot isLoading rootMessage={null} />
          <ThreadOverlayReplies isLoading messages={[]} replyCount={0} />
        </>
      ) : (
        <>
          <WorkBanner
            hasNeedsInput={openWork.hasNeedsInput}
            needsInputCount={openWork.needsInputCount}
            openCount={openWork.openCount}
            workingCount={openWork.workingCount}
          />
          <ThreadOverlayRoot
            isLoading={overlay.isDetailLoading}
            rootMessage={overlay.rootMessage}
          />
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
        </>
      )}
    </section>
  );
}
