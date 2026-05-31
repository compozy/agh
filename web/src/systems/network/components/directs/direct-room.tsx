import { DetailHeader, Eyebrow } from "@agh/ui";

import { cn } from "@/lib/utils";

import { formatNetworkPresenceLabel } from "../../lib/network-formatters";
import type { NetworkPresence, NetworkPresenceState } from "../../types";
import { DetailComposer } from "../composer/detail-composer";
import { ConversationError } from "../empty-states/conversation-error";
import { DirectEmpty } from "../empty-states/direct-empty";
import { Timeline } from "../timeline/timeline";
import { MessageAvatar } from "../timeline/message-avatar";
import { useMessageCopyActions } from "../timeline/use-message-copy-actions";
import { WorkBanner } from "../work/work-banner";
import { useDirectRoomView } from "./use-direct-room-view";

export interface DirectRoomProps {
  workspaceId: string;
  channel: string;
  directId: string;
  /** Used to render the *other* party's identity at the top per `_design.md` §5.6. */
  selfPeerId?: string;
}

interface PresenceBadgeProps {
  presence: NetworkPresence;
}

function presenceDotTone(state: NetworkPresenceState): string {
  switch (state) {
    case "local":
      return "bg-info";
    case "active":
      return "bg-success";
    case "inactive":
      return "bg-warning";
    case "expired":
      return "bg-danger";
    default:
      return "bg-muted";
  }
}

function PresenceBadge({ presence }: PresenceBadgeProps) {
  const label = formatNetworkPresenceLabel(presence.state, presence.lastSeenAgeSeconds);
  return (
    <span
      aria-label={`peer presence ${label}`}
      className={cn(
        "inline-flex min-w-0 items-center gap-1 text-form-label text-muted",
        presence.state === "active" && "text-fg"
      )}
      data-state={presence.state}
      data-testid="network-direct-presence"
      title="Derived from network greet activity: active within GreetInterval, inactive within the 2x window."
    >
      <span
        aria-hidden="true"
        className={cn(
          "inline-block size-1.5 shrink-0 rounded-full",
          presenceDotTone(presence.state),
          presence.state === "active" && "motion-safe:animate-pulse"
        )}
        data-testid="network-direct-presence-dot"
      />
      <span className="min-w-0 truncate">{label}</span>
    </span>
  );
}

export function DirectRoom({ workspaceId, channel, directId, selfPeerId }: DirectRoomProps) {
  const view = useDirectRoomView({ channel, directId, selfPeerId });
  const { room, session, disabledReason, openWork, handleRetry, handleDiscard } = view;
  const otherPeerId = room.otherPeerId;
  const detailError = room.detailError;
  const isResolvingDetail = !detailError && !room.detail;
  const toolbarHandlers = useMessageCopyActions({
    surface: "direct",
    workspaceId,
    channel,
    conversationId: directId,
  });

  return (
    <section
      aria-label={`Direct room with @${otherPeerId || "peer"}`}
      className="flex min-h-0 flex-1 flex-col"
      data-testid="network-direct-room"
    >
      <DetailHeader
        actions={
          <div
            data-slot="direct-room-meta"
            data-testid="network-direct-identity-row-meta"
            className="flex items-center gap-2 text-small-body text-muted"
          >
            <Eyebrow>agent</Eyebrow>
            <PresenceBadge presence={room.presence} />
          </div>
        }
        className="px-5 py-3"
        data-testid="network-direct-identity-row"
        title={
          <span className="flex min-w-0 items-center gap-3">
            {otherPeerId ? (
              <MessageAvatar
                initialFrom={otherPeerId}
                name={otherPeerId}
                ownerRole="agent"
                seed={otherPeerId}
                sizePx={32}
              />
            ) : null}
            <span className="truncate">{otherPeerId ? `@${otherPeerId}` : "Direct room"}</span>
          </span>
        }
      />

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
            toolbarHandlers={toolbarHandlers}
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
