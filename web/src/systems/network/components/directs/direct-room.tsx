import { cn } from "@/lib/utils";

import type { NetworkPresenceState } from "../../hooks/use-network-presence";
import { DetailComposer } from "../composer/detail-composer";
import { ConversationError } from "../empty-states/conversation-error";
import { DirectEmpty } from "../empty-states/direct-empty";
import { Timeline } from "../timeline/timeline";
import { MessageAvatar } from "../timeline/message-avatar";
import { WorkBanner } from "../work/work-banner";
import { useDirectRoomView } from "./use-direct-room-view";

export interface DirectRoomProps {
  channel: string;
  directId: string;
  /** Used to render the *other* party's identity at the top per `_design.md` §5.6. */
  selfPeerId?: string;
}

interface PresenceDotProps {
  state: NetworkPresenceState;
}

function PresenceDot({ state }: PresenceDotProps) {
  if (state === "idle") {
    return null;
  }

  const tone =
    state === "running"
      ? "var(--color-accent)"
      : state === "needs_input"
        ? "var(--color-warning)"
        : "var(--color-danger)";

  const ariaLabel =
    state === "running" ? "running" : state === "needs_input" ? "needs input" : "errored";

  return (
    <span
      aria-label={ariaLabel}
      className={cn(
        "ml-1 inline-block size-1.5 rounded-full",
        state === "running" && "motion-safe:animate-pulse"
      )}
      data-state={state}
      data-testid="network-direct-presence-dot"
      style={{ backgroundColor: tone }}
    />
  );
}

export function DirectRoom({ channel, directId, selfPeerId }: DirectRoomProps) {
  const view = useDirectRoomView({ channel, directId, selfPeerId });
  const { room, session, disabledReason, openWork, handleRetry, handleDiscard } = view;
  const otherPeerId = room.otherPeerId;
  const detailError = room.detailError;
  const isResolvingDetail = !detailError && !room.detail;

  return (
    <section
      aria-label={`Direct room with @${otherPeerId || "peer"}`}
      className="flex min-h-0 flex-1 flex-col"
      data-testid="network-direct-room"
    >
      <header
        className="flex h-12 items-center gap-3 border-b border-(--color-divider) px-5"
        data-testid="network-direct-identity-row"
      >
        {otherPeerId ? (
          <MessageAvatar initialFrom={otherPeerId} seed={otherPeerId} sizePx={32} />
        ) : null}
        <div className="flex min-w-0 flex-1 items-center gap-2">
          <h1 className="truncate text-base font-semibold text-(--color-text-primary)">
            {otherPeerId ? `@${otherPeerId}` : "Direct room"}
          </h1>
          <span className="font-mono text-badge uppercase tracking-mono text-(--color-text-tertiary)">
            agent
          </span>
          <PresenceDot state={room.presence.state} />
        </div>
      </header>

      {detailError ? (
        <div className="flex flex-1 items-center justify-center px-5 py-10" role="alert">
          <ConversationError
            description={`Could not load direct room ${directId}. Choose an existing direct room from #${channel}.`}
            testId="network-direct-room-error"
            title="Direct room unavailable"
          />
        </div>
      ) : isResolvingDetail ? (
        <Timeline
          ariaLabel={`Direct messages with @${otherPeerId || "peer"}`}
          density="channel"
          isLoading
          messages={[]}
        />
      ) : (
        <>
          <WorkBanner
            hasNeedsInput={openWork.hasNeedsInput}
            needsInputCount={openWork.needsInputCount}
            openCount={openWork.openCount}
            workingCount={openWork.workingCount}
          />

          <Timeline
            ariaLabel={`Direct messages with @${otherPeerId || "peer"}`}
            density="channel"
            emptyState={<DirectEmpty />}
            isLoading={room.isDetailLoading || room.isMessagesLoading}
            lastReadAt={room.lastReadIso}
            messages={room.messages}
            onDiscardOptimistic={handleDiscard}
            onRetryOptimistic={handleRetry}
          />

          <DetailComposer
            channel={channel}
            directId={directId}
            disabledReason={disabledReason ?? undefined}
            displayName={session?.displayName}
            peerFrom={session?.peerId ?? ""}
            peerLabel={otherPeerId ? `@${otherPeerId}` : "@peer"}
            peerTo={otherPeerId || undefined}
            sessionId={session?.sessionId ?? ""}
            surface="direct"
          />
        </>
      )}
    </section>
  );
}
