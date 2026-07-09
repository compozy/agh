import { ChevronDown, ListFilter } from "lucide-react";

import {
  Button,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@agh/ui";

import type { TaskListSortKey } from "../types";

const SORT_LABELS: Record<TaskListSortKey, string> = {
  recent: "Most recent",
  priority: "Priority",
};

const SORT_OPTIONS: TaskListSortKey[] = ["recent", "priority"];

export interface TasksListSortProps {
  sortBy: TaskListSortKey;
  onSortChange: (next: TaskListSortKey) => void;
}

/**
 * Sort control for the tasks listing toolbar trailing slot. Tasks is the only
 * listing surface with a real sort, so the "Sorted by …" label is allowed here
 * (LISTING-STANDARD forbids it only when no sort control exists).
 */
export function TasksListSort({ sortBy, onSortChange }: TasksListSortProps) {
  return (
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
  );
}
