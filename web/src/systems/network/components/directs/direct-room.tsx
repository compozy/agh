import { cn } from "@/lib/utils";

import { useDirectRoom } from "../../hooks/use-direct-room";
import type { NetworkPresenceState } from "../../hooks/use-network-presence";
import { Timeline } from "../timeline/timeline";
import { MessageAvatar } from "../timeline/message-avatar";

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
  const room = useDirectRoom({ channel, directId, selfPeerId });
  const otherPeerId = room.otherPeerId;

  return (
    <section
      aria-label={`Direct room with @${otherPeerId || "peer"}`}
      className="flex min-h-0 flex-1 flex-col"
      data-testid="network-direct-room"
    >
      <header
        className="flex h-12 items-center gap-3 border-b border-[color:var(--color-divider)] px-5"
        data-testid="network-direct-identity-row"
      >
        {otherPeerId ? (
          <MessageAvatar initialFrom={otherPeerId} seed={otherPeerId} sizePx={32} />
        ) : null}
        <div className="flex min-w-0 flex-1 items-center gap-2">
          <h1 className="truncate text-[16px] font-semibold text-[color:var(--color-text-primary)]">
            {otherPeerId ? `@${otherPeerId}` : "Direct room"}
          </h1>
          <span className="font-mono text-[10px] uppercase tracking-[0.06em] text-[color:var(--color-text-tertiary)]">
            agent
          </span>
          <PresenceDot state={room.presence.state} />
        </div>
      </header>

      <Timeline
        ariaLabel={`Direct messages with @${otherPeerId || "peer"}`}
        density="channel"
        emptyState={
          <p className="text-[13px] text-[color:var(--color-text-tertiary)]">Quiet so far.</p>
        }
        isLoading={room.isMessagesLoading}
        lastReadAt={room.lastReadIso}
        messages={room.messages}
      />
    </section>
  );
}
