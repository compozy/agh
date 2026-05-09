import { ChevronDown } from "lucide-react";

import {
  Button,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  Eyebrow,
  PillGroup,
  type PillGroupItem,
} from "@agh/ui";

import { cn } from "@/lib/utils";

export type NetworkListFilter = "all" | "has_work" | "me" | "pinned" | "unread";

export type NetworkListSort = "recent_activity" | "created" | "alphabetical";

export interface NetworkListFilterCounts {
  all: number;
  hasWork: number;
  me: number;
  pinned: number;
  unread: number;
}

const SORT_LABELS: Record<NetworkListSort, string> = {
  recent_activity: "Recent activity",
  created: "Created",
  alphabetical: "Alphabetical",
};

export interface ListFilterBarProps {
  filter: NetworkListFilter;
  sort: NetworkListSort;
  counts: NetworkListFilterCounts;
  onFilterChange: (next: NetworkListFilter) => void;
  onSortChange: (next: NetworkListSort) => void;
  onMarkAllRead?: () => void;
  isMarkAllReadDisabled?: boolean;
  className?: string;
}

export function ListFilterBar({
  filter,
  sort,
  counts,
  onFilterChange,
  onSortChange,
  onMarkAllRead,
  isMarkAllReadDisabled,
  className,
}: ListFilterBarProps) {
  const items: PillGroupItem<NetworkListFilter>[] = [
    {
      value: "all",
      label: <FilterLabel count={counts.all}>All</FilterLabel>,
      testId: "network-filter-all",
    },
    {
      value: "has_work",
      label: <FilterLabel count={counts.hasWork}>Has work</FilterLabel>,
      testId: "network-filter-has-work",
    },
    {
      value: "me",
      label: <FilterLabel>@me</FilterLabel>,
      testId: "network-filter-me",
    },
    {
      value: "pinned",
      label: <FilterLabel count={counts.pinned}>Pinned</FilterLabel>,
      testId: "network-filter-pinned",
    },
    {
      value: "unread",
      label: <FilterLabel count={counts.unread}>Unread</FilterLabel>,
      testId: "network-filter-unread",
    },
  ];

  return (
    <div
      className={cn(
        "flex flex-wrap items-center gap-3 border-b border-(--color-divider) px-5 py-2",
        className
      )}
      data-testid="network-list-filter-bar"
    >
      <Eyebrow>Filter</Eyebrow>
      <PillGroup
        aria-label="List filter"
        data-testid="network-list-filter-pills"
        items={items}
        onChange={onFilterChange}
        size="sm"
        value={filter}
      />

      <div className="ml-auto flex items-center gap-3">
        <Eyebrow>Sort</Eyebrow>
        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <Button
                aria-label="Sort list"
                data-testid="network-list-sort-trigger"
                size="sm"
                type="button"
                variant="outline"
              />
            }
          >
            <span>{SORT_LABELS[sort]}</span>
            <ChevronDown aria-hidden="true" className="size-3.5" />
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            {(Object.keys(SORT_LABELS) as NetworkListSort[]).map(option => (
              <DropdownMenuItem
                data-active={option === sort ? "true" : undefined}
                data-testid={`network-list-sort-${option}`}
                key={option}
                onSelect={event => {
                  event.preventDefault();
                  onSortChange(option);
                }}
              >
                {SORT_LABELS[option]}
              </DropdownMenuItem>
            ))}
          </DropdownMenuContent>
        </DropdownMenu>

        <Button
          aria-label="Mark all visible items as read"
          data-testid="network-list-mark-all-read"
          disabled={isMarkAllReadDisabled || !onMarkAllRead}
          onClick={onMarkAllRead}
          size="sm"
          type="button"
          variant="outline"
        >
          Mark all read
        </Button>
      </div>
    </div>
  );
}

function FilterLabel({ children, count }: { children: React.ReactNode; count?: number }) {
  return (
    <span className="inline-flex items-center gap-1.5">
      <span>{children}</span>
      {typeof count === "number" && count > 0 ? <Eyebrow weight="medium">{count}</Eyebrow> : null}
    </span>
  );
}
