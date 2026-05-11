import { Activity, ListTodo, Users, X, type LucideIcon } from "lucide-react";

import { Button, Eyebrow, Tabs, TabsList, TabsTrigger } from "@agh/ui";

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

interface NetworkInspectorTabsProps {
  activeTab: InspectorTab;
  onTabChange: (tab: InspectorTab) => void;
  workCount: number;
}

function NetworkInspectorTabs({ activeTab, onTabChange, workCount }: NetworkInspectorTabsProps) {
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
    <Tabs
      aria-label="Inspector sections"
      className="gap-0 border-b border-line"
      data-testid="network-inspector-tabs"
      onValueChange={value => {
        if (value === "members" || value === "work" || value === "activity") {
          onTabChange(value);
        }
      }}
      value={activeTab}
    >
      <TabsList className="h-10 w-full bg-transparent p-0">
        {tabs.map(tab => {
          const Icon = tab.icon;
          return (
            <TabsTrigger
              className="h-10 flex-1 gap-1.5 px-3 group-data-horizontal/tabs:after:bottom-0"
              count={tab.count && tab.count > 0 ? tab.count : undefined}
              data-testid={tab.testId}
              key={tab.id}
              value={tab.id}
            >
              <Icon aria-hidden="true" className="size-3.5 shrink-0" />
              <span>{tab.label}</span>
            </TabsTrigger>
          );
        })}
      </TabsList>
    </Tabs>
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
      <header className="flex items-center gap-2 border-b border-line px-4 py-2.5">
        <Eyebrow>Inspector</Eyebrow>
        <div className="ml-auto flex items-center gap-1">
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

      <NetworkInspectorTabs activeTab={activeTab} onTabChange={onTabChange} workCount={workCount} />

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
