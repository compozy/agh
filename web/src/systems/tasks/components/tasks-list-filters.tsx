import { ChevronDown, ListFilter } from "lucide-react";
import { useMemo } from "react";

import {
  Button,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@agh/ui";
import { Filters, type Filter } from "@agh/ui/components/reui/filters";

import type { TaskListSortKey } from "@/hooks/routes/use-tasks-page";
import {
  applyTaskFilterChips,
  buildTaskFilterFields,
  taskFiltersToChips,
  type TaskFilterHandlers,
  type TaskFilterOwnerOption,
  type TaskFilterState,
  type TaskScopeFilter,
} from "../lib/tasks-list-filters";

const SORT_LABELS: Record<TaskListSortKey, string> = {
  recent: "Most recent",
  priority: "Priority",
};

const SORT_OPTIONS: TaskListSortKey[] = ["recent", "priority"];

export interface TasksListFiltersProps extends TaskFilterState {
  ownerOptions: TaskFilterOwnerOption[];
  sortBy: TaskListSortKey;
  onStatusChange: TaskFilterHandlers["onStatusChange"];
  onOwnerChange: TaskFilterHandlers["onOwnerChange"];
  onPriorityChange: TaskFilterHandlers["onPriorityChange"];
  onScopeChange: (next: TaskScopeFilter) => void;
  onSortChange: (next: TaskListSortKey) => void;
}

/**
 * Inline filter bar for the tasks list page — replaces the old
 * lane PillGroup (All/Mine/Watched/Blocked/Failed). Wraps the shared
 * `<Filters>` chip primitive with task-aware field config and a right-aligned
 * Sort dropdown. State stays owned by `useTasksPage`; this component is a
 * presentation shell that translates typed filters ⇄ chip array via
 * `lib/tasks-list-filters.ts`.
 */
export function TasksListFilters({
  statusFilter,
  ownerFilter,
  priorityFilter,
  scopeFilter,
  ownerOptions,
  sortBy,
  onStatusChange,
  onOwnerChange,
  onPriorityChange,
  onScopeChange,
  onSortChange,
}: TasksListFiltersProps) {
  const fields = useMemo(() => buildTaskFilterFields(ownerOptions), [ownerOptions]);

  const chips = useMemo(
    () => taskFiltersToChips({ statusFilter, ownerFilter, priorityFilter, scopeFilter }),
    [statusFilter, ownerFilter, priorityFilter, scopeFilter]
  );

  const handleFiltersChange = (next: Filter<string>[]) => {
    applyTaskFilterChips(next, {
      onStatusChange,
      onOwnerChange,
      onPriorityChange,
      onScopeChange,
    });
  };

  return (
    <div
      className="flex flex-wrap items-center gap-2 border-b border-line-soft pb-3"
      data-testid="tasks-list-filters"
    >
      <Filters<string>
        allowMultiple={false}
        fields={fields}
        filters={chips}
        onChange={handleFiltersChange}
        size="sm"
        trigger={
          <Button
            aria-label="Add filter"
            data-testid="tasks-list-filters-add"
            size="sm"
            type="button"
            variant="ghost"
          >
            <ListFilter aria-hidden="true" className="size-3" />
            Filter
          </Button>
        }
      />

      <div className="ml-auto flex items-center gap-1.5">
        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <Button
                aria-label="Sort tasks"
                data-testid="tasks-list-sort-trigger"
                size="sm"
                type="button"
                variant="ghost"
              />
            }
          >
            <ListFilter aria-hidden="true" className="size-3 text-subtle" />
            <span className="text-muted">Sorted by</span>
            <span className="text-fg-strong">{SORT_LABELS[sortBy]}</span>
            <ChevronDown aria-hidden="true" className="size-3 text-subtle" />
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            {SORT_OPTIONS.map(option => (
              <DropdownMenuItem
                data-active={option === sortBy ? "true" : undefined}
                data-testid={`tasks-list-sort-${option}`}
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
      </div>
    </div>
  );
}
