import { AlertCircle, ListChecks, Plus, Search } from "lucide-react";

import { Button, Empty, PillGroup, SearchInput, Section, Skeleton } from "@agh/ui";

import type { TaskListItem, TaskStatus } from "../types";
import { TaskCard } from "./task-card";

export type TasksListLane = "all" | "mine" | "watched";

const LANE_ITEMS = [
  { value: "all" as const, label: "All", testId: "tasks-list-lane-all" },
  { value: "mine" as const, label: "Mine", testId: "tasks-list-lane-mine" },
  { value: "watched" as const, label: "Watched", testId: "tasks-list-lane-watched" },
];

const TASK_LIST_SKELETON_IDS = [
  "task-list-skeleton-1",
  "task-list-skeleton-2",
  "task-list-skeleton-3",
  "task-list-skeleton-4",
  "task-list-skeleton-5",
];

export interface TasksListPanelProps {
  tasks: TaskListItem[];
  totalCount: number;
  selectedTaskId: string | null;
  onSelectTask: (taskId: string) => void;
  searchQuery: string;
  onSearchChange: (next: string) => void;
  isLoading?: boolean;
  errorMessage?: string | null;
  statusFilter?: TaskStatus | null;
  onCreateTask?: () => void;
  /** Optional current mine/watched/all lane selection. Defaults to "all". */
  lane?: TasksListLane;
  /** Optional callback when the lane switcher emits a new value. */
  onLaneChange?: (next: TasksListLane) => void;
  /** Optional watched-set test id of mine/watched counts exposed as lane badges. */
  laneBadges?: Partial<Record<TasksListLane, number>>;
}

const STATUS_HEADLINES: Partial<Record<TaskStatus, string>> = {
  in_progress: "In Progress",
  ready: "Ready",
  blocked: "Blocked",
  pending: "Pending",
  draft: "Draft",
  failed: "Failed",
  completed: "Completed",
  canceled: "Canceled",
};

function getStatusHeadline(filter?: TaskStatus | null): string {
  if (!filter) return "All Tasks";
  return STATUS_HEADLINES[filter] ?? "Tasks";
}

/**
 * Tasks list column -- search + lane switcher + rows, consumed by the `SplitPane`
 * list slot on `/tasks`. Composes `@agh/ui` `SearchInput`, `PillGroup`, `Section`,
 * and `Empty`; rows come from `TaskCard` (built on the shared `TasksListRow`).
 */
export function TasksListPanel({
  tasks,
  totalCount,
  selectedTaskId,
  onSelectTask,
  searchQuery,
  onSearchChange,
  isLoading = false,
  errorMessage = null,
  statusFilter = null,
  onCreateTask,
  lane = "all",
  onLaneChange,
  laneBadges,
}: TasksListPanelProps) {
  const isEmpty = tasks.length === 0;

  return (
    <aside className="flex min-h-0 flex-1 flex-col bg-canvas" data-testid="tasks-list-panel">
      <div className="flex flex-col gap-2 border-b border-line-soft px-3.5 py-2.5">
        <SearchInput
          value={searchQuery}
          onChange={onSearchChange}
          placeholder="Filter tasks..."
          data-testid="tasks-list-search-input"
        />
        <PillGroup
          aria-label="Task lane"
          data-testid="tasks-list-lane-pills"
          items={LANE_ITEMS.map(item => ({
            ...item,
            badge: laneBadges?.[item.value],
          }))}
          onChange={next => onLaneChange?.(next)}
          size="sm"
          value={lane}
        />
        {onCreateTask ? (
          <Button
            className="w-full justify-center"
            data-testid="tasks-list-create"
            onClick={onCreateTask}
            size="sm"
            type="button"
            variant="neutral"
          >
            <Plus className="size-3.5" />
            New task
          </Button>
        ) : null}
      </div>

      <div
        className="flex items-center justify-between border-b border-line-soft px-3.5 py-2"
        data-testid="tasks-list-headline-row"
      >
        <span className="eyebrow text-muted" data-testid="tasks-list-headline">
          {getStatusHeadline(statusFilter)}
        </span>
        <span
          className="font-mono text-[10px] tabular-nums text-faint"
          data-testid="tasks-list-total"
        >
          {totalCount}
        </span>
      </div>

      <div className="flex-1 overflow-y-auto">
        {isLoading && isEmpty ? (
          <div className="flex flex-col" data-testid="tasks-list-loading">
            {TASK_LIST_SKELETON_IDS.map(id => (
              <div
                className="flex items-center gap-3 border-b border-line-soft py-3 pr-3 pl-3.5"
                key={id}
              >
                <Skeleton className="size-1.5 rounded-full" />
                <div className="flex min-w-0 flex-1 flex-col gap-1.5">
                  <Skeleton className="h-3 w-3/5 rounded-xs" />
                  <Skeleton className="h-2.5 w-2/5 rounded-xs" />
                </div>
              </div>
            ))}
          </div>
        ) : errorMessage && isEmpty ? (
          <Empty
            data-testid="tasks-list-error"
            description={errorMessage}
            icon={AlertCircle}
            title="Unable to load tasks"
          />
        ) : isEmpty ? (
          <Empty
            data-testid="tasks-list-empty"
            description="Adjust the search or open a new task contract from the rail."
            icon={searchQuery ? Search : ListChecks}
            title="Nothing matches the current filters"
          />
        ) : (
          <Section data-testid="tasks-list-rows" className="gap-0">
            {tasks.map(task => (
              <TaskCard
                key={task.id}
                onSelect={() => onSelectTask(task.id)}
                selected={task.id === selectedTaskId}
                task={task}
              />
            ))}
          </Section>
        )}
      </div>
    </aside>
  );
}
