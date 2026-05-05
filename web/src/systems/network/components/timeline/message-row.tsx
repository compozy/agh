import { cn } from "@/lib/utils";

import type { NetworkConversationMessage } from "../../types";
import { formatTimelineClock, formatTimelineIso } from "../../lib/format-timestamp";
import { HoverToolbar, type HoverToolbarHandlers } from "./hover-toolbar";
import { MessageAvatar } from "./message-avatar";
import { MessageBodyText } from "./message-body";

export type MessageRowDensity = "channel" | "overlay";

export interface MessageRowProps extends HoverToolbarHandlers {
  message: NetworkConversationMessage;
  density?: MessageRowDensity;
  className?: string;
}

function pickRoleLabel(message: NetworkConversationMessage): "agent" | "human" | "system" {
  if (message.session_id != null && message.session_id !== "") {
    return "agent";
  }
  return message.local ? "human" : "system";
}

function authorSeed(message: NetworkConversationMessage): string {
  return message.peer_from ?? message.display_name ?? message.message_id;
}

const DENSITY_AVATAR: Record<MessageRowDensity, 36 | 32> = {
  channel: 36,
  overlay: 32,
};

export function MessageRow({
  message,
  density = "channel",
  className,
  onReply,
  onPin,
  onFork,
  onMore,
}: MessageRowProps) {
  const role = pickRoleLabel(message);
  const clock = formatTimelineClock(message.timestamp);
  const iso = formatTimelineIso(message.timestamp);
  const displayName = message.display_name?.trim() || message.peer_from || "Unknown peer";
  const avatarSize = DENSITY_AVATAR[density];

  return (
    <article
      aria-label={`${displayName} message`}
      className={cn(
        "group relative flex gap-3 px-5 py-1.5",
        density === "overlay" && "px-4",
        className
      )}
      data-density={density}
      data-testid="network-message-row-full"
      data-message-id={message.message_id}
      data-variant="full"
    >
      <MessageAvatar initialFrom={displayName} seed={authorSeed(message)} sizePx={avatarSize} />

      <div className="flex min-w-0 flex-1 flex-col gap-1">
        <div className="flex items-baseline gap-2">
          <span className="truncate text-[14px] font-semibold text-[color:var(--color-text-primary)]">
            {displayName}
          </span>
          <span
            className="font-mono text-[10px] uppercase tracking-[0.06em] text-[color:var(--color-text-tertiary)]"
            data-testid="network-message-role-chip"
          >
            {role}
          </span>
          <time
            className="text-[12px] text-[color:var(--color-text-tertiary)]"
            data-testid="network-message-timestamp"
            dateTime={iso}
            title={iso}
          >
            {clock}
          </time>
        </div>

        <MessageBodyText message={message} />
      </div>

      <HoverToolbar
        onFork={onFork}
        onMore={onMore}
        onPin={onPin}
        onReply={onReply}
        testIdSuffix={message.message_id}
      />
    </article>
  );
}
