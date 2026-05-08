import { Link } from "@tanstack/react-router";

import { cn } from "@/lib/utils";

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

function CountChip({ count }: { count: number }) {
  return (
    <span
      aria-label={`${count} entries`}
      className="font-mono text-badge text-(--color-text-tertiary)"
    >
      {count}
    </span>
  );
}

export function ChannelTabs({ channel, activeTab, threadCount, directCount }: ChannelTabsProps) {
  const tabs = buildTabs({ threadCount, directCount });

  return (
    <nav
      aria-label={`Surfaces for #${channel}`}
      className="flex items-center gap-1 border-b border-(--color-divider) px-5"
      data-testid="network-channel-tabs"
      role="tablist"
    >
      {tabs.map(tab => (
        <Link
          aria-current={tab.tab === activeTab ? "page" : undefined}
          aria-selected={tab.tab === activeTab}
          className={cn(
            "relative flex h-9 items-center gap-2 px-3 text-small-body focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent",
            tab.tab === activeTab
              ? "text-(--color-text-primary)"
              : "text-(--color-text-secondary) hover:text-(--color-text-primary)"
          )}
          data-testid={tab.testId}
          key={tab.tab}
          params={{ channel }}
          role="tab"
          to={tab.to}
        >
          <span className="capitalize">{tab.label}</span>
          {tab.count != null ? <CountChip count={tab.count} /> : null}
          {tab.tab === activeTab ? (
            <span
              aria-hidden="true"
              className="pointer-events-none absolute right-3 bottom-0 left-3 h-[2px] bg-accent"
            />
          ) : null}
        </Link>
      ))}
    </nav>
  );
}
