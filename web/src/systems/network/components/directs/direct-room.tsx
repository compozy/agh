import { Eyebrow } from "@agh/ui";

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
      ? "var(--accent)"
      : state === "needs_input"
        ? "var(--warning)"
        : "var(--danger)";

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
        data-slot="page-header"
        className="flex min-h-11 flex-col gap-2 border-b border-(--line) px-5 py-2.5"
        data-testid="network-direct-identity-row"
      >
        <div
          data-slot="page-header-main"
          className="flex min-w-0 flex-wrap items-center gap-2 sm:gap-3"
        >
          <div data-slot="page-header-title" className="flex min-w-0 items-center gap-2">
            <h1 className="truncate text-[22px] font-medium tracking-[-0.026em] text-(--fg-strong)">
              <span className="flex min-w-0 items-center gap-3">
                {otherPeerId ? (
                  <MessageAvatar initialFrom={otherPeerId} seed={otherPeerId} sizePx={32} />
                ) : null}
                <span className="truncate">{otherPeerId ? `@${otherPeerId}` : "Direct room"}</span>
              </span>
            </h1>
          </div>
          <div
            data-slot="page-header-meta"
            className="ml-auto flex shrink-0 items-center gap-2 text-[13px] text-(--muted)"
          >
            <Eyebrow weight="medium">agent</Eyebrow>
            <PresenceDot state={room.presence.state} />
          </div>
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
