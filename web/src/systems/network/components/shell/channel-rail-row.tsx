import { Star } from "lucide-react";
import { Link } from "@tanstack/react-router";

import { cn } from "@/lib/utils";
import type { NetworkChannelSummary } from "../../types";

const HASH_LABEL = "#";

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
      className={cn(
        "group relative mx-1.5 flex items-center gap-2 rounded-[6px] pr-1.5 transition-colors",
        active ? "bg-[color:var(--color-accent-tint)]" : "hover:bg-[color:var(--color-hover)]"
      )}
      data-testid={`network-channel-row-${channel.channel}`}
    >
      <Link
        aria-current={active ? "page" : undefined}
        className="flex min-w-0 flex-1 items-center gap-2 rounded-[6px] px-2 py-1.5 text-left focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-[color:var(--color-accent)]"
        data-testid={`network-channel-link-${channel.channel}`}
        params={{ channel: channel.channel }}
        to="/network/$channel/threads"
      >
        <span
          aria-hidden="true"
          className={cn(
            "font-mono text-[13px] leading-none",
            active
              ? "text-[color:var(--color-text-primary)]"
              : "text-[color:var(--color-text-tertiary)]"
          )}
        >
          {HASH_LABEL}
        </span>
        <span
          className={cn(
            "min-w-0 truncate text-[13px] tracking-[-0.005em]",
            active
              ? "font-semibold text-[color:var(--color-text-primary)]"
              : hasUnread
                ? "font-semibold text-[color:var(--color-text-primary)]"
                : "font-normal text-[color:var(--color-text-secondary)]"
          )}
        >
          {channel.channel}
        </span>
      </Link>

      <button
        aria-label={ariaLabel}
        aria-pressed={isPinned}
        className={cn(
          "rounded-[4px] p-1 text-[color:var(--color-text-tertiary)] opacity-0 transition-opacity focus-visible:outline-none focus-visible:opacity-100 focus-visible:ring-1 focus-visible:ring-[color:var(--color-accent)] group-hover:opacity-100",
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
          className={cn(
            "size-3",
            isPinned ? "fill-[color:var(--color-accent)] text-[color:var(--color-accent)]" : null
          )}
        />
      </button>
    </div>
  );
}
