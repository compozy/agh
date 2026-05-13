import { Link } from "@tanstack/react-router";
import { AtSign, MessagesSquare } from "lucide-react";

import { cn } from "@/lib/utils";
import {
  Eyebrow,
  Item,
  ItemContent,
  ItemFooter,
  ItemMedia,
  ItemTitle,
  SidebarSectionLabel,
  Skeleton,
  SkeletonRows,
} from "@agh/ui";

import { formatNetworkRelativeTime } from "../../lib/network-formatters";
import type { NetworkRecentEntry } from "../../types";

const RECENTS_HEADING = "Recents";

export interface ChannelRailRecentsProps {
  workspaceId: string;
  recents: ReadonlyArray<NetworkRecentEntry>;
  isLoading: boolean;
}

function RecentEntryRow({
  workspaceId,
  entry,
}: {
  workspaceId: string;
  entry: NetworkRecentEntry;
}) {
  const Icon = entry.surface === "thread" ? MessagesSquare : AtSign;
  const ariaLabel = entry.surface === "thread" ? "Thread" : "Direct room";
  const timestampLabel = entry.lastActivityAt
    ? formatNetworkRelativeTime(entry.lastActivityAt)
    : null;

  if (entry.surface === "thread") {
    return (
      <Item
        className="rounded-mono-badge border-transparent py-1 text-xs"
        data-testid={`network-recents-thread-${entry.containerId}`}
        render={
          <Link
            params={{ workspaceId, channel: entry.channel, threadId: entry.containerId }}
            to="/network/$workspaceId/$channel/threads/$threadId"
          />
        }
        selectable
        size="xs"
      >
        <ItemMedia>
          <Icon aria-label={ariaLabel} className="size-3 shrink-0 text-subtle" />
        </ItemMedia>
        <ItemContent className="min-w-0">
          <ItemTitle
            className={cn(
              "min-w-0 text-xs",
              entry.hasUnread ? "font-medium text-fg" : "text-muted"
            )}
          >
            <span className="truncate">{entry.preview}</span>
            <Eyebrow className="shrink-0">#{entry.channel}</Eyebrow>
          </ItemTitle>
        </ItemContent>
        {timestampLabel ? (
          <ItemFooter className="basis-auto">
            <Eyebrow className="shrink-0">{timestampLabel}</Eyebrow>
          </ItemFooter>
        ) : null}
      </Item>
    );
  }

  return (
    <Item
      className="rounded-mono-badge border-transparent py-1 text-xs"
      data-testid={`network-recents-direct-${entry.containerId}`}
      render={
        <Link
          params={{ workspaceId, channel: entry.channel, directId: entry.containerId }}
          to="/network/$workspaceId/$channel/directs/$directId"
        />
      }
      selectable
      size="xs"
    >
      <ItemMedia>
        <Icon aria-label={ariaLabel} className="size-3 shrink-0 text-subtle" />
      </ItemMedia>
      <ItemContent className="min-w-0">
        <ItemTitle
          className={cn("min-w-0 text-xs", entry.hasUnread ? "font-medium text-fg" : "text-muted")}
        >
          <span className="truncate">{entry.preview}</span>
          <span className="text-subtle">in</span>
          <Eyebrow className="shrink-0">#{entry.channel}</Eyebrow>
        </ItemTitle>
      </ItemContent>
      {timestampLabel ? (
        <ItemFooter className="basis-auto">
          <Eyebrow className="shrink-0">{timestampLabel}</Eyebrow>
        </ItemFooter>
      ) : null}
    </Item>
  );
}

export function ChannelRailRecents({ workspaceId, recents, isLoading }: ChannelRailRecentsProps) {
  return (
    <section aria-label="Cross-channel recents" className="space-y-1" data-testid="network-recents">
      <SidebarSectionLabel>{RECENTS_HEADING}</SidebarSectionLabel>
      <div className="space-y-0.5">
        {isLoading && recents.length === 0 ? (
          <SkeletonRows
            count={3}
            className="gap-1.5 px-2 py-1"
            data-testid="network-recents-loading"
          >
            <Skeleton className="h-3 w-full" />
          </SkeletonRows>
        ) : recents.length === 0 ? (
          <p className="px-2 py-1 text-eyebrow text-subtle" data-testid="network-recents-empty">
            Recent threads and direct rooms appear here.
          </p>
        ) : (
          recents.map(entry => (
            <RecentEntryRow
              entry={entry}
              key={`${entry.surface}:${entry.containerId}`}
              workspaceId={workspaceId}
            />
          ))
        )}
      </div>
    </section>
  );
}
