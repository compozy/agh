import { Link } from "@tanstack/react-router";
import { Hash, Star } from "lucide-react";

import { Item, ItemContent, ItemMedia, ItemTitle } from "@agh/ui";

import { cn } from "@/lib/utils";
import type { NetworkChannelSummary } from "../../types";

export interface ChannelRailRowProps {
  workspaceId: string;
  channel: NetworkChannelSummary;
  active: boolean;
  hasUnread: boolean;
  isPinned: boolean;
  onTogglePinned: (channel: string) => void;
}

export function ChannelRailRow({
  workspaceId,
  channel,
  active,
  hasUnread,
  isPinned,
  onTogglePinned,
}: ChannelRailRowProps) {
  const ariaLabel = isPinned ? `Unpin #${channel.channel}` : `Pin #${channel.channel}`;

  return (
    <div
      className="group relative flex items-center"
      data-testid={`network-channel-row-${channel.channel}`}
    >
      <Item
        aria-current={active ? "page" : undefined}
        className={cn(
          "min-w-0 flex-1 rounded-mono-badge border-transparent py-1 pr-7 pl-2 text-small-body",
          !active && hasUnread && "font-medium text-fg"
        )}
        data-active={active}
        data-testid={`network-channel-link-${channel.channel}`}
        indicator={active ? "rail" : "none"}
        render={
          <Link
            params={{ workspaceId, channel: channel.channel }}
            to="/network/$workspaceId/$channel/threads"
          />
        }
        selectable
        selected={active}
        size="xs"
      >
        <ItemMedia>
          <Hash
            aria-hidden="true"
            className={cn("size-3 shrink-0", active ? "text-fg" : "text-subtle")}
          />
        </ItemMedia>
        <ItemContent className="min-w-0">
          <ItemTitle className="min-w-0 text-small-body">
            <span className="truncate">{channel.channel}</span>
          </ItemTitle>
        </ItemContent>
      </Item>

      <button
        aria-label={ariaLabel}
        aria-pressed={isPinned}
        className={cn(
          "absolute right-1 top-1/2 -translate-y-1/2 rounded-chip p-1 text-subtle opacity-0 transition-opacity focus-visible:opacity-100 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent group-hover:opacity-100",
          (isPinned || active) && "opacity-100"
        )}
        data-testid={`network-channel-pin-${channel.channel}`}
        onClick={event => {
          event.preventDefault();
          event.stopPropagation();
          onTogglePinned(channel.channel);
        }}
        type="button"
      >
        <Star
          aria-hidden="true"
          className={cn("size-3", isPinned ? "fill-accent text-accent" : null)}
        />
      </button>
    </div>
  );
}
