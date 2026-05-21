import { Users } from "lucide-react";

import {
  Empty,
  Eyebrow,
  Item,
  ItemContent,
  ItemMedia,
  ItemTitle,
  Skeleton,
  SkeletonRows,
} from "@agh/ui";

import { cn } from "@/lib/utils";

import type { ChannelMember } from "../../hooks/use-channel-members";
import { formatNetworkPresenceLabel } from "../../lib/network-formatters";
import type { NetworkPresenceState } from "../../types";
import { MessageAvatar } from "../timeline/message-avatar";

export interface InspectorMembersListProps {
  members: ReadonlyArray<ChannelMember>;
  isLoading?: boolean;
  className?: string;
}

function MembersSkeleton() {
  return (
    <SkeletonRows
      aria-hidden="true"
      count={3}
      data-testid="network-inspector-members-skeleton"
      rowClassName="flex-row items-center gap-3 border-b border-line px-4 py-3"
    >
      <Skeleton className="size-8 rounded-md" />
      <div className="flex flex-col gap-1.5">
        <Skeleton className="h-3 w-24" />
        <Skeleton className="h-2.5 w-16" />
      </div>
    </SkeletonRows>
  );
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

export function InspectorMembersList({
  members,
  isLoading = false,
  className,
}: InspectorMembersListProps) {
  if (isLoading && members.length === 0) {
    return <MembersSkeleton />;
  }

  if (members.length === 0) {
    return (
      <div className="flex justify-center px-4 py-6">
        <Empty
          className="max-w-sm"
          description="No peers have joined this channel yet."
          fill={false}
          icon={Users}
          title="No members."
        />
      </div>
    );
  }

  return (
    <div
      aria-label="Channel members"
      className={cn("flex min-h-0 flex-1 flex-col overflow-y-auto", className)}
      data-testid="network-inspector-members-list"
      role="list"
    >
      {members.map(member => (
        <Item
          className="rounded-none border-b border-line px-4 py-3 last:border-b-0"
          data-testid={`network-inspector-member-${member.peerId}`}
          key={member.peerId}
          role="listitem"
        >
          <ItemMedia>
            <MessageAvatar
              initialFrom={member.displayName || member.peerId}
              seed={member.peerId}
              sizePx={32}
            />
          </ItemMedia>
          <ItemContent className="min-w-0">
            <ItemTitle className="min-w-0 text-small-body">
              {member.displayName || `@${member.peerId}`}
            </ItemTitle>
            <div className="flex min-w-0 flex-wrap items-center gap-2">
              <Eyebrow data-testid={`network-inspector-member-role-${member.peerId}`}>
                {member.role === "agent" ? "AGENT" : "HUMAN"}
              </Eyebrow>
              <span
                className="inline-flex min-w-0 items-center gap-1 text-form-label text-muted"
                data-testid={`network-inspector-member-presence-${member.peerId}`}
              >
                <span
                  aria-hidden="true"
                  className={cn(
                    "size-1.5 shrink-0 rounded-full",
                    presenceDotTone(member.presenceState)
                  )}
                />
                <span className="min-w-0 truncate">
                  {formatNetworkPresenceLabel(member.presenceState, member.lastSeenAgeSeconds)}
                </span>
              </span>
            </div>
          </ItemContent>
        </Item>
      ))}
    </div>
  );
}
