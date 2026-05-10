import { Link } from "@tanstack/react-router";

import { Tabs, TabsList, TabsTrigger } from "@agh/ui";

export type ChannelTab = "threads" | "directs" | "activity";

export interface ChannelTabsProps {
  channel: string;
  activeTab: ChannelTab;
  threadCount: number | null;
  directCount: number | null;
}

interface TabDescriptor {
  tab: ChannelTab;
  label: string;
  count: number | null;
  to: "/network/$channel/threads" | "/network/$channel/directs" | "/network/$channel/activity";
  testId: string;
}

function buildTabs({
  threadCount,
  directCount,
}: {
  threadCount: number | null;
  directCount: number | null;
}): TabDescriptor[] {
  return [
    {
      tab: "threads",
      label: "Threads",
      count: threadCount,
      to: "/network/$channel/threads",
      testId: "network-tab-threads",
    },
    {
      tab: "directs",
      label: "Directs",
      count: directCount,
      to: "/network/$channel/directs",
      testId: "network-tab-directs",
    },
    {
      tab: "activity",
      label: "Activity",
      count: null,
      to: "/network/$channel/activity",
      testId: "network-tab-activity",
    },
  ];
}

export function ChannelTabs({ channel, activeTab, threadCount, directCount }: ChannelTabsProps) {
  const tabs = buildTabs({ threadCount, directCount });

  return (
    <Tabs
      aria-label={`Surfaces for #${channel}`}
      className="gap-0 border-b border-(--line) px-5"
      data-testid="network-channel-tabs"
      value={activeTab}
    >
      <TabsList variant="line" className="h-9 bg-transparent p-0">
        {tabs.map(tab => (
          <TabsTrigger
            aria-current={tab.tab === activeTab ? "page" : undefined}
            className="h-9 gap-2 px-3 text-small-body group-data-horizontal/tabs:after:bottom-0"
            count={tab.count ?? undefined}
            data-testid={tab.testId}
            key={tab.tab}
            render={<Link params={{ channel }} to={tab.to} />}
            value={tab.tab}
          >
            {tab.label}
          </TabsTrigger>
        ))}
      </TabsList>
    </Tabs>
  );
}
