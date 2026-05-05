import { Link } from "@tanstack/react-router";

import { SidebarSectionLabel, Skeleton } from "@agh/ui";
import { cn } from "@/lib/utils";

import { formatNetworkRelativeTime } from "../../lib/network-formatters";
import type { NetworkRecentEntry } from "../../types";

const RECENTS_HEADING = "Recents";

export interface ChannelRailRecentsProps {
  recents: ReadonlyArray<NetworkRecentEntry>;
  isLoading: boolean;
}

function RecentEntryRow({ entry }: { entry: NetworkRecentEntry }) {
  const tag = entry.surface === "thread" ? "TH" : "DM";
  const timestampLabel = entry.lastActivityAt
    ? formatNetworkRelativeTime(entry.lastActivityAt)
    : null;

  if (entry.surface === "thread") {
    return (
      <Link
        className="group flex items-center gap-2 rounded-[6px] px-2 py-1 transition-colors hover:bg-[color:var(--color-hover)] focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-[color:var(--color-accent)]"
        data-testid={`network-recents-thread-${entry.containerId}`}
        params={{ channel: entry.channel, threadId: entry.containerId }}
        to="/network/$channel/threads/$threadId"
      >
        <span
          aria-hidden="true"
          className="font-mono text-[10px] tracking-[0.06em] text-[color:var(--color-text-tertiary)]"
        >
          [{tag}]
        </span>
        <span
          className={cn(
            "min-w-0 flex-1 truncate text-[12px]",
            entry.hasUnread
              ? "font-semibold text-[color:var(--color-text-primary)]"
              : "text-[color:var(--color-text-secondary)]"
          )}
        >
          <span className="font-mono">#{entry.channel}</span>
          <span className="px-1 text-[color:var(--color-text-tertiary)]">·</span>
          <span>{entry.preview}</span>
        </span>
        {timestampLabel ? (
          <span className="shrink-0 font-mono text-[10px] text-[color:var(--color-text-tertiary)]">
            {timestampLabel}
          </span>
        ) : null}
      </Link>
    );
  }

  return (
    <Link
      className="group flex items-center gap-2 rounded-[6px] px-2 py-1 transition-colors hover:bg-[color:var(--color-hover)] focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-[color:var(--color-accent)]"
      data-testid={`network-recents-direct-${entry.containerId}`}
      params={{ channel: entry.channel, directId: entry.containerId }}
      to="/network/$channel/directs/$directId"
    >
      <span
        aria-hidden="true"
        className="font-mono text-[10px] tracking-[0.06em] text-[color:var(--color-text-tertiary)]"
      >
        [{tag}]
      </span>
      <span
        className={cn(
          "min-w-0 flex-1 truncate text-[12px]",
          entry.hasUnread
            ? "font-semibold text-[color:var(--color-text-primary)]"
            : "text-[color:var(--color-text-secondary)]"
        )}
      >
        <span>{entry.preview}</span>
        <span className="px-1 text-[color:var(--color-text-tertiary)]">in</span>
        <span className="font-mono">#{entry.channel}</span>
      </span>
      {timestampLabel ? (
        <span className="shrink-0 font-mono text-[10px] text-[color:var(--color-text-tertiary)]">
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
            className="px-2 py-1 text-[11px] text-[color:var(--color-text-tertiary)]"
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
