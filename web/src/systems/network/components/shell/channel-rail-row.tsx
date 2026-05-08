import { Hash, Star } from "lucide-react";
import { Link } from "@tanstack/react-router";

import { cn } from "@/lib/utils";
import {
  ACTIVE_NAV_INDICATOR_CLASS,
  ACTIVE_NAV_ROW_CLASS,
  NAV_ROW_CLASS,
} from "@/components/sidebar-nav-classes";
import type { NetworkChannelSummary } from "../../types";

export interface ChannelRailRowProps {
  channel: NetworkChannelSummary;
  active: boolean;
  hasUnread: boolean;
  isPinned: boolean;
  onTogglePinned: (channel: string) => void;
}

export function ChannelRailRow({
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
      <Link
        aria-current={active ? "page" : undefined}
        className={cn(
          NAV_ROW_CLASS,
          "min-w-0 flex-1 pr-7",
          active && ACTIVE_NAV_ROW_CLASS,
          !active && hasUnread && "font-semibold text-(--color-text-primary)"
        )}
        data-active={active}
        data-testid={`network-channel-link-${channel.channel}`}
        params={{ channel: channel.channel }}
        to="/network/$channel/threads"
      >
        {active ? <span aria-hidden="true" className={ACTIVE_NAV_INDICATOR_CLASS} /> : null}
        <Hash
          aria-hidden="true"
          className={cn(
            "size-3.5 shrink-0",
            active ? "text-(--color-text-primary)" : "text-(--color-text-tertiary)"
          )}
        />
        <span className="min-w-0 truncate">{channel.channel}</span>
      </Link>

      <button
        aria-label={ariaLabel}
        aria-pressed={isPinned}
        className={cn(
          "absolute right-1 top-1/2 -translate-y-1/2 rounded-chip p-1 text-(--color-text-tertiary) opacity-0 transition-opacity focus-visible:opacity-100 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent group-hover:opacity-100",
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
