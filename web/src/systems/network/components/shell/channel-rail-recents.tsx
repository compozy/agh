import { AtSign, MessagesSquare } from "lucide-react";
import { Link } from "@tanstack/react-router";

import { SidebarSectionLabel, Skeleton } from "@agh/ui";
import { cn } from "@/lib/utils";
import { NAV_ROW_CLASS } from "@/components/sidebar-nav-classes";

import { formatNetworkRelativeTime } from "../../lib/network-formatters";
import type { NetworkRecentEntry } from "../../types";

const RECENTS_HEADING = "Recents";

export interface ChannelRailRecentsProps {
  recents: ReadonlyArray<NetworkRecentEntry>;
  isLoading: boolean;
}

function RecentEntryRow({ entry }: { entry: NetworkRecentEntry }) {
  const Icon = entry.surface === "thread" ? MessagesSquare : AtSign;
  const ariaLabel = entry.surface === "thread" ? "Thread" : "Direct room";
  const timestampLabel = entry.lastActivityAt
    ? formatNetworkRelativeTime(entry.lastActivityAt)
    : null;

  if (entry.surface === "thread") {
    return (
      <Link
        className={cn(NAV_ROW_CLASS, "py-1 text-xs")}
        data-testid={`network-recents-thread-${entry.containerId}`}
        params={{ channel: entry.channel, threadId: entry.containerId }}
        to="/network/$channel/threads/$threadId"
      >
        <Icon aria-label={ariaLabel} className="size-3.5 shrink-0 text-(--color-text-tertiary)" />
        <span
          className={cn(
            "min-w-0 flex-1 truncate",
            entry.hasUnread
              ? "font-semibold text-(--color-text-primary)"
              : "text-(--color-text-secondary)"
          )}
        >
          <span>{entry.preview}</span>
          <span className="px-1 text-(--color-text-tertiary)">·</span>
          <span className="font-mono text-(--color-text-tertiary)">#{entry.channel}</span>
        </span>
        {timestampLabel ? (
          <span className="shrink-0 font-mono text-badge text-(--color-text-tertiary)">
            {timestampLabel}
          </span>
        ) : null}
      </Link>
    );
  }

  return (
    <Link
      className={cn(NAV_ROW_CLASS, "py-1 text-xs")}
      data-testid={`network-recents-direct-${entry.containerId}`}
      params={{ channel: entry.channel, directId: entry.containerId }}
      to="/network/$channel/directs/$directId"
    >
      <Icon aria-label={ariaLabel} className="size-3.5 shrink-0 text-(--color-text-tertiary)" />
      <span
        className={cn(
          "min-w-0 flex-1 truncate",
          entry.hasUnread
            ? "font-semibold text-(--color-text-primary)"
            : "text-(--color-text-secondary)"
        )}
      >
        <span>{entry.preview}</span>
        <span className="px-1 text-(--color-text-tertiary)">in</span>
        <span className="font-mono text-(--color-text-tertiary)">#{entry.channel}</span>
      </span>
      {timestampLabel ? (
        <span className="shrink-0 font-mono text-badge text-(--color-text-tertiary)">
          {timestampLabel}
        </span>
      ) : null}
    </Link>
  );
}

export function ChannelRailRecents({ recents, isLoading }: ChannelRailRecentsProps) {
  return (
    <section aria-label="Cross-channel recents" className="space-y-1" data-testid="network-recents">
      <SidebarSectionLabel>{RECENTS_HEADING}</SidebarSectionLabel>
      <div className="space-y-0.5">
        {isLoading && recents.length === 0 ? (
          <div className="space-y-1.5 px-2 py-1" data-testid="network-recents-loading">
            <Skeleton className="h-3 w-full" />
            <Skeleton className="h-3 w-5/6" />
            <Skeleton className="h-3 w-2/3" />
          </div>
        ) : recents.length === 0 ? (
          <p
            className="px-2 py-1 text-eyebrow text-(--color-text-tertiary)"
            data-testid="network-recents-empty"
          >
            Recent threads and direct rooms appear here.
          </p>
        ) : (
          recents.map(entry => (
            <RecentEntryRow entry={entry} key={`${entry.surface}:${entry.containerId}`} />
          ))
        )}
      </div>
    </section>
  );
}
