import { Activity, ListTodo, MoreHorizontal, Users, X, type LucideIcon } from "lucide-react";

import { Button } from "@agh/ui";

import { cn } from "@/lib/utils";

import type { ChannelMember } from "../../hooks/use-channel-members";
import type { InspectorTab } from "../../hooks/use-inspector-state";
import type { OpenWorkEntry } from "../../hooks/use-work";
import type { NetworkDirectRoomSummary, NetworkThreadSummary } from "../../types";
import { WorkInspector } from "../work/work-inspector";
import { InspectorActivityFeed } from "./inspector-activity-feed";
import { InspectorMembersList } from "./inspector-members-list";

export interface NetworkInspectorProps {
  channel: string;
  activeTab: InspectorTab;
  onTabChange: (tab: InspectorTab) => void;
  onClose: () => void;
  members: ReadonlyArray<ChannelMember>;
  isMembersLoading: boolean;
  workEntries: ReadonlyArray<OpenWorkEntry>;
  isWorkLoading: boolean;
  workCount: number;
  onWorkJump?: (entry: OpenWorkEntry) => void;
  threads: ReadonlyArray<NetworkThreadSummary>;
  directs: ReadonlyArray<NetworkDirectRoomSummary>;
  isActivityLoading: boolean;
  className?: string;
}

interface TabDescriptor {
  id: InspectorTab;
  label: string;
  icon: LucideIcon;
  count?: number;
  testId: string;
}

interface InspectorTabNavProps {
  activeTab: InspectorTab;
  onTabChange: (tab: InspectorTab) => void;
  workCount: number;
}

function InspectorTabNav({ activeTab, onTabChange, workCount }: InspectorTabNavProps) {
  const tabs: TabDescriptor[] = [
    {
      id: "members",
      label: "Members",
      icon: Users,
      testId: "network-inspector-tab-members",
    },
    {
      id: "work",
      label: "Work",
      icon: ListTodo,
      count: workCount,
      testId: "network-inspector-tab-work",
    },
    {
      id: "activity",
      label: "Activity",
      icon: Activity,
      testId: "network-inspector-tab-activity",
    },
  ];

  return (
    <nav
      aria-label="Inspector sections"
      className="flex w-full items-stretch border-b border-(--color-divider)"
      data-testid="network-inspector-tabs"
      role="tablist"
    >
      {tabs.map(tab => {
        const Icon = tab.icon;
        const isActive = tab.id === activeTab;
        return (
          <button
            aria-current={isActive ? "page" : undefined}
            aria-selected={isActive}
            className={cn(
              "relative flex flex-1 items-center justify-center gap-1.5 px-3 py-2 font-mono text-eyebrow font-semibold uppercase tracking-mono transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent focus-visible:ring-inset",
              isActive
                ? "text-(--color-text-primary)"
                : "text-(--color-text-tertiary) hover:text-(--color-text-secondary)"
            )}
            data-active={isActive}
            data-testid={tab.testId}
            key={tab.id}
            onClick={() => onTabChange(tab.id)}
            role="tab"
            type="button"
          >
            <Icon aria-hidden="true" className="size-3.5 shrink-0" />
            <span>{tab.label}</span>
            {typeof tab.count === "number" && tab.count > 0 ? (
              <span
                className="inline-flex h-4 min-w-4 items-center justify-center rounded-full border border-(--color-divider) px-1 font-mono text-badge tracking-mono text-(--color-text-tertiary)"
                data-testid={`${tab.testId}-count`}
              >
                {tab.count}
              </span>
            ) : null}
            {isActive ? (
              <span
                aria-hidden="true"
                className="pointer-events-none absolute right-0 bottom-0 left-0 h-[2px] bg-accent"
              />
            ) : null}
          </button>
        );
      })}
    </nav>
  );
}

export function NetworkInspector({
  channel,
  activeTab,
  onTabChange,
  onClose,
  members,
  isMembersLoading,
  workEntries,
  isWorkLoading,
  workCount,
  onWorkJump,
  threads,
  directs,
  isActivityLoading,
  className,
}: NetworkInspectorProps) {
  return (
    <section
      aria-label="Channel inspector"
      className={cn("flex min-h-0 flex-1 flex-col", className)}
      data-testid="network-inspector"
    >
      <header className="flex items-center gap-2 border-b border-(--color-divider) px-4 py-2.5">
        <span className="font-mono text-badge font-semibold uppercase tracking-badge text-(--color-text-tertiary)">
          Inspector
        </span>
        <div className="ml-auto flex items-center gap-1">
          <Button
            aria-disabled="true"
            aria-label="Inspector actions — coming soon"
            data-testid="network-inspector-overflow"
            onClick={event => event.preventDefault()}
            size="icon-sm"
            tabIndex={-1}
            title="More actions · Coming soon"
            type="button"
            variant="ghost"
          >
            <MoreHorizontal aria-hidden="true" className="size-4" />
          </Button>
          <Button
            aria-label="Close inspector"
            data-testid="network-inspector-close"
            onClick={onClose}
            size="icon-sm"
            type="button"
            variant="ghost"
          >
            <X aria-hidden="true" className="size-4" />
          </Button>
        </div>
      </header>

      <InspectorTabNav activeTab={activeTab} onTabChange={onTabChange} workCount={workCount} />

      <div
        className="flex min-h-0 flex-1 flex-col"
        data-testid={`network-inspector-panel-${activeTab}`}
        role="tabpanel"
      >
        {activeTab === "members" ? (
          <InspectorMembersList isLoading={isMembersLoading} members={members} />
        ) : activeTab === "work" ? (
          <WorkInspector
            chromeless
            entries={workEntries}
            isLoading={isWorkLoading}
            onJump={onWorkJump}
          />
        ) : (
          <InspectorActivityFeed
            channel={channel}
            directs={directs}
            isLoading={isActivityLoading}
            threads={threads}
          />
        )}
      </div>
    </section>
  );
}
