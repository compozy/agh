import { Users } from "lucide-react";

import { Empty, Skeleton } from "@agh/ui";

import { cn } from "@/lib/utils";

import type { ChannelMember } from "../../hooks/use-channel-members";
import { MessageAvatar } from "../timeline/message-avatar";

export interface InspectorMembersListProps {
  members: ReadonlyArray<ChannelMember>;
  isLoading?: boolean;
  className?: string;
}

function MembersSkeleton() {
  return (
    <ul
      aria-hidden="true"
      className="flex flex-col"
      data-testid="network-inspector-members-skeleton"
    >
      {[0, 1, 2].map(index => (
        <li
          className="flex items-center gap-3 border-b border-[color:var(--color-divider)] px-4 py-3"
          key={index}
        >
          <Skeleton className="size-8 rounded-md" />
          <div className="flex flex-col gap-1.5">
            <Skeleton className="h-3 w-24" />
            <Skeleton className="h-2.5 w-16" />
          </div>
        </li>
      ))}
    </ul>
  );
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
    <ul
      aria-label="Channel members"
      className={cn("flex min-h-0 flex-1 flex-col overflow-y-auto", className)}
      data-testid="network-inspector-members-list"
    >
      {members.map(member => (
        <li
          className="flex items-center gap-3 border-b border-[color:var(--color-divider)] px-4 py-3 last:border-b-0"
          data-testid={`network-inspector-member-${member.peerId}`}
          key={member.peerId}
        >
          <MessageAvatar
            initialFrom={member.displayName || member.peerId}
            seed={member.peerId}
            sizePx={32}
          />
          <div className="flex min-w-0 flex-1 flex-col">
            <span className="truncate text-[13px] font-medium text-[color:var(--color-text-primary)]">
              {member.displayName || `@${member.peerId}`}
            </span>
            <span
              className="font-mono text-[10px] uppercase tracking-[0.06em] text-[color:var(--color-text-tertiary)]"
              data-testid={`network-inspector-member-role-${member.peerId}`}
            >
              {member.role === "agent" ? "AGENT" : "HUMAN"}
            </span>
          </div>
        </li>
      ))}
    </ul>
  );
}
