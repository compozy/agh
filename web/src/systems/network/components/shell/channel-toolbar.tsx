import { useNavigate } from "@tanstack/react-router";
import { Briefcase, CheckCheck, ChevronDown, ListFilter, UserRound } from "lucide-react";
import { useMemo, type ReactNode } from "react";

import {
  Button,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  LaneTabs,
  SearchInput,
  Tooltip,
  TooltipContent,
  TooltipTrigger,
  type LaneTabsItem,
} from "@agh/ui";
import { Filters, type FilterFieldsConfig } from "@agh/ui";

import { useNetworkListFiltersContext } from "../../hooks/use-network-list-filters-context";
import { type NetworkFilterKey, type NetworkListSort } from "../../hooks/use-network-list-filters";
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
    key: "includes_me",
    label: "Includes me",
    icon: <UserRound aria-hidden="true" className="size-3" />,
    testId: "network-toolbar-field-includes-me",
  },
];

interface TabItem extends LaneTabsItem<ChannelTab> {
  to:
    | "/network/$workspaceId/$channel/threads"
    | "/network/$workspaceId/$channel/directs"
    | "/network/$workspaceId/$channel/activity";
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
      to: "/network/$workspaceId/$channel/threads",
      testId: "network-tab-threads",
    },
    {
      value: "directs",
      label: "Directs",
      count: directCount ?? undefined,
      to: "/network/$workspaceId/$channel/directs",
      testId: "network-tab-directs",
    },
    {
      value: "activity",
      label: "Activity",
      to: "/network/$workspaceId/$channel/activity",
      testId: "network-tab-activity",
    },
  ];
}

export interface ChannelToolbarProps {
  workspaceId: string;
  channel: string;
  activeTab: ChannelTab;
  threadCount: number | null;
  directCount: number | null;
}

export function ChannelToolbar({
  workspaceId,
  channel,
  activeTab,
  threadCount,
  directCount,
}: ChannelToolbarProps) {
  const {
    filters,
    setFilters,
    sort,
    setSort,
    searchQuery,
    setSearchQuery,
    canFilterBySelf,
    markLoadedRead,
    isMarkLoadedReadDisabled,
  } = useNetworkListFiltersContext();
  const navigate = useNavigate();
  const tabs = useMemo(() => buildTabs({ threadCount, directCount }), [threadCount, directCount]);

  const filterFields = useMemo<FilterFieldsConfig<boolean>>(() => {
    return CHIP_FIELDS.filter(field => field.key !== "includes_me" || canFilterBySelf).map(
      field => ({
        key: field.key,
        label: field.label,
        icon: field.icon,
        type: "toggle" as const,
      })
    );
  }, [canFilterBySelf]);

  const handleTabChange = (next: ChannelTab) => {
    const target = tabs.find(tab => tab.value === next);
    if (!target || !workspaceId) return;
    void navigate({ params: { workspaceId, channel }, to: target.to });
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
        <SearchInput
          className="h-8 w-56"
          data-testid="network-list-search"
          onChange={setSearchQuery}
          placeholder={`Search #${channel}…`}
          value={searchQuery}
        />
        <Filters<boolean>
          allowMultiple
          fields={filterFields}
          filters={filters}
          onChange={setFilters}
          showSearchInput={false}
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
                aria-label="Mark loaded conversations as read"
                data-testid="network-list-mark-all-read"
                disabled={isMarkLoadedReadDisabled}
                onClick={markLoadedRead}
                size="icon-sm"
                type="button"
                variant="ghost"
              />
            }
          >
            <CheckCheck aria-hidden="true" className="size-3" />
          </TooltipTrigger>
          <TooltipContent>Mark loaded conversations as read</TooltipContent>
        </Tooltip>
      </div>
    </div>
  );
}
