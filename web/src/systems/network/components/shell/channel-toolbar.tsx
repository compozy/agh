import { useNavigate } from "@tanstack/react-router";
import {
  AtSign,
  Briefcase,
  CheckCheck,
  ChevronDown,
  CircleDot,
  ListFilter,
  Pin,
} from "lucide-react";
import { useMemo, type ReactNode } from "react";

import {
  Button,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  LaneTabs,
  Tooltip,
  TooltipContent,
  TooltipTrigger,
  type LaneTabsItem,
} from "@agh/ui";
import { Filters, type FilterFieldsConfig } from "@agh/ui/components/reui/filters";

import { useNetworkListFiltersContext } from "../../hooks/use-network-list-filters-context";
import {
  NETWORK_FILTER_KEYS,
  type NetworkFilterKey,
  type NetworkListSort,
} from "../../hooks/use-network-list-filters";
import type { ChannelTab } from "./channel-tabs-types";

const SORT_LABELS: Record<NetworkListSort, string> = {
  recent_activity: "Recent activity",
  created: "Created",
  alphabetical: "Alphabetical",
};

interface ChipFieldDescriptor {
  key: NetworkFilterKey;
  label: string;
  icon: ReactNode;
  testId: string;
}

const CHIP_FIELDS: ReadonlyArray<ChipFieldDescriptor> = [
  {
    key: "has_work",
    label: "Has work",
    icon: <Briefcase aria-hidden="true" className="size-3" />,
    testId: "network-toolbar-field-has-work",
  },
  {
    key: "mentions_me",
    label: "@me",
    icon: <AtSign aria-hidden="true" className="size-3" />,
    testId: "network-toolbar-field-mentions-me",
  },
  {
    key: "pinned",
    label: "Pinned",
    icon: <Pin aria-hidden="true" className="size-3" />,
    testId: "network-toolbar-field-pinned",
  },
  {
    key: "unread",
    label: "Unread",
    icon: <CircleDot aria-hidden="true" className="size-3" />,
    testId: "network-toolbar-field-unread",
  },
];

interface TabItem extends LaneTabsItem<ChannelTab> {
  to: "/network/$channel/threads" | "/network/$channel/directs" | "/network/$channel/activity";
  testId: string;
}

function buildTabs({
  threadCount,
  directCount,
}: {
  threadCount: number | null;
  directCount: number | null;
}): ReadonlyArray<TabItem> {
  return [
    {
      value: "threads",
      label: "Threads",
      count: threadCount ?? undefined,
      to: "/network/$channel/threads",
      testId: "network-tab-threads",
    },
    {
      value: "directs",
      label: "Directs",
      count: directCount ?? undefined,
      to: "/network/$channel/directs",
      testId: "network-tab-directs",
    },
    {
      value: "activity",
      label: "Activity",
      to: "/network/$channel/activity",
      testId: "network-tab-activity",
    },
  ];
}

export interface ChannelToolbarProps {
  channel: string;
  activeTab: ChannelTab;
  threadCount: number | null;
  directCount: number | null;
}

export function ChannelToolbar({
  channel,
  activeTab,
  threadCount,
  directCount,
}: ChannelToolbarProps) {
  const { filters, setFilters, sort, setSort, markAllRead, isMarkAllReadDisabled } =
    useNetworkListFiltersContext();
  const navigate = useNavigate();
  const tabs = useMemo(() => buildTabs({ threadCount, directCount }), [threadCount, directCount]);

  const filterFields = useMemo<FilterFieldsConfig<boolean>>(() => {
    return CHIP_FIELDS.map(field => ({
      key: field.key,
      label: field.label,
      icon: field.icon,
      type: "toggle" as const,
    }));
  }, []);

  const handleTabChange = (next: ChannelTab) => {
    const target = tabs.find(tab => tab.value === next);
    if (!target) return;
    void navigate({ params: { channel }, to: target.to });
  };

  return (
    <div
      className="flex items-center gap-3 border-b border-line px-5"
      data-testid="network-channel-toolbar"
    >
      <LaneTabs<ChannelTab>
        ariaLabel={`Surfaces for #${channel}`}
        className="border-b-0"
        data-testid="network-channel-tabs"
        items={tabs}
        onChange={handleTabChange}
        value={activeTab}
      />

      <div className="ml-auto flex items-center gap-1.5">
        <Filters<boolean>
          allowMultiple={false}
          fields={filterFields}
          filters={filters}
          onChange={setFilters}
          showSearchInput={NETWORK_FILTER_KEYS.length > 4}
          size="sm"
          trigger={
            <Button
              aria-label="Add filter"
              data-testid="network-toolbar-add-filter"
              size="sm"
              variant="ghost"
            >
              <ListFilter aria-hidden="true" className="size-3" />
              Filter
            </Button>
          }
        />

        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <Button
                aria-label="Sort list"
                data-testid="network-list-sort-trigger"
                size="sm"
                type="button"
                variant="ghost"
              />
            }
          >
            <span className="text-muted">{SORT_LABELS[sort]}</span>
            <ChevronDown aria-hidden="true" className="size-3 text-subtle" />
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            {(Object.keys(SORT_LABELS) as NetworkListSort[]).map(option => (
              <DropdownMenuItem
                data-active={option === sort ? "true" : undefined}
                data-testid={`network-list-sort-${option}`}
                key={option}
                onSelect={event => {
                  event.preventDefault();
                  setSort(option);
                }}
              >
                {SORT_LABELS[option]}
              </DropdownMenuItem>
            ))}
          </DropdownMenuContent>
        </DropdownMenu>

        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                aria-label="Mark all visible items as read"
                data-testid="network-list-mark-all-read"
                disabled={isMarkAllReadDisabled}
                onClick={markAllRead}
                size="icon-sm"
                type="button"
                variant="ghost"
              />
            }
          >
            <CheckCheck aria-hidden="true" className="size-3" />
          </TooltipTrigger>
          <TooltipContent>Mark all read</TooltipContent>
        </Tooltip>
      </div>
    </div>
  );
}
