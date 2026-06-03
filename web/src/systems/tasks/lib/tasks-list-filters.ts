import type { Filter, FilterFieldsConfig } from "@agh/ui/components/reui/filters";

import type { TaskPriority, TaskStatus } from "../types";
import { taskOwnerKindLabel, taskPriorityLabel, taskStatusLabel } from "./task-formatters";

export type TaskFilterFieldKey = "status" | "owner" | "priority";

export interface TaskFilterOwnerOption {
  ref: string;
  kind?: import("../types").TaskOwnerKind;
}

export interface TaskFilterState {
  statusFilter: TaskStatus | null;
  ownerFilter: string | null;
  priorityFilter: TaskPriority | null;
}

export interface TaskFilterHandlers {
  onStatusChange: (next: TaskStatus | null) => void;
  onOwnerChange: (next: string | null) => void;
  onPriorityChange: (next: TaskPriority | null) => void;
}

const STATUS_OPTIONS: TaskStatus[] = [
  "in_progress",
  "ready",
  "blocked",
  "pending",
  "draft",
  "completed",
  "failed",
  "canceled",
];

const PRIORITY_OPTIONS: TaskPriority[] = ["urgent", "high", "medium", "low"];

/**
 * Build the `FilterFieldsConfig` consumed by `<Filters>` — three single-select
 * chip fields covering status, owner, and priority. Owner options come
 * from the live task list (`ownerOptions` in `useTasksPage`) so the menu only
 * surfaces owners that actually exist in the active workspace.
 */
export function buildTaskFilterFields(
  ownerOptions: TaskFilterOwnerOption[]
): FilterFieldsConfig<string> {
  return [
    {
      key: "status",
      label: "Status",
      type: "select",
      options: STATUS_OPTIONS.map(value => ({ value, label: taskStatusLabel(value) })),
    },
    {
      key: "priority",
      label: "Priority",
      type: "select",
      options: PRIORITY_OPTIONS.map(value => ({ value, label: taskPriorityLabel(value) })),
    },
    {
      key: "owner",
      label: "Owner",
      type: "select",
      searchable: true,
      options: ownerOptions.map(owner => ({
        value: owner.ref,
        label: owner.ref || taskOwnerKindLabel(owner.kind),
      })),
    },
  ];
}

/**
 * Project the typed filter state held by `useTasksPage` onto the `<Filters>`
 * chip array. Chip ids are derived from `{field, value}` so the same logical
 * filter keeps a stable identity across renders without an intermediate cache.
 */
export function taskFiltersToChips(state: TaskFilterState): Filter<string>[] {
  const chips: Filter<string>[] = [];
  if (state.statusFilter) {
    chips.push(buildChip("status", state.statusFilter));
  }
  if (state.priorityFilter) {
    chips.push(buildChip("priority", state.priorityFilter));
  }
  if (state.ownerFilter) {
    chips.push(buildChip("owner", state.ownerFilter));
  }
  return chips;
}

function buildChip(field: TaskFilterFieldKey, value: string): Filter<string> {
  return {
    id: `task-filter-${field}`,
    field,
    operator: "is",
    values: [value],
  };
}

/**
 * Decode the `<Filters>` chip array back into the typed setters owned by
 * `useTasksPage`. Filters that disappear from the array reset their slot to
 * `null` so removing a chip restores the default.
 */
export function applyTaskFilterChips(chips: Filter<string>[], handlers: TaskFilterHandlers): void {
  const lookup = new Map<string, string | undefined>();
  for (const chip of chips) {
    lookup.set(chip.field, chip.values[0]);
  }

  handlers.onStatusChange(asTaskStatus(lookup.get("status")));
  handlers.onPriorityChange(asTaskPriority(lookup.get("priority")));
  handlers.onOwnerChange(lookup.get("owner") ?? null);
}

function asTaskStatus(value: string | undefined): TaskStatus | null {
  if (!value) return null;
  return (STATUS_OPTIONS as readonly string[]).includes(value) ? (value as TaskStatus) : null;
}

function asTaskPriority(value: string | undefined): TaskPriority | null {
  if (!value) return null;
  return (PRIORITY_OPTIONS as readonly string[]).includes(value) ? (value as TaskPriority) : null;
}
