import * as React from "react";

import { Pill } from "@agh/ui";
import { cn } from "@/lib/utils";
import { pillToneFromLegacyTone } from "@/lib/pill-variant";

import {
  formatRelativeTime,
  taskLaneTone,
  taskShortId,
  taskStatusSignal,
} from "../lib/task-formatters";
import type { TaskInboxLane, TaskListItem } from "../types";

export interface TasksListRowProps extends Omit<React.ComponentProps<"div">, "onSelect"> {
  task: TaskListItem;
  selected?: boolean;
  onSelect?: (taskId: string) => void;
  /** Optional lane badge — used by the Inbox (task 18) to tag rows by lane. */
  lane?: TaskInboxLane | null;
  /** Optional slot rendered after the metadata row (e.g. action buttons). */
  trailing?: React.ReactNode;
  /** Optional slot rendered under the metadata row (e.g. failure reason). */
  footer?: React.ReactNode;
  /** Optional test-id override. Defaults to `task-card-${task.id}` for back-compat. */
  testId?: string;
}

const LANE_LABELS: Record<TaskInboxLane, string> = {
  my_work: "Mine",
  approvals: "Approvals",
  failed_runs: "Failed",
  blocked: "Blocked",
  archived: "Archived",
};

/**
 * Shared list row primitive — `StatusDot` tone + title + `MonoBadge` id + timestamp
 * + optional lane `Pills` badge. Consumed by `tasks-list-panel`, `task-card`, the
 * Kanban cards (task 18), and Inbox rows (task 18). DESIGN.md §4 list-row
 * composition; visual shape mirrors the mock at
 * `docs/design/web-inspiration/src/pages-core.jsx`.
 */
function TasksListRow({
  task,
  selected = false,
  onSelect,
  lane = null,
  trailing,
  footer,
  testId,
  className,
  ...props
}: TasksListRowProps) {
  const signal = taskStatusSignal(task.status);
  const identifier = taskShortId(task);
  const lastActivity = task.last_activity_at ?? task.updated_at;
  const timestamp = formatRelativeTime(lastActivity);
  const resolvedTestId = testId ?? `task-card-${task.id}`;

  const clickable = onSelect !== undefined;
  const handleKeyDown = clickable
    ? (event: React.KeyboardEvent<HTMLDivElement>) => {
        if (event.target !== event.currentTarget) return;
        if (event.key === "Enter" || event.key === " ") {
          event.preventDefault();
          onSelect?.(task.id);
        }
      }
    : undefined;

  return (
    <div
      role={clickable ? "button" : undefined}
      tabIndex={clickable ? 0 : undefined}
      aria-pressed={selected}
      data-slot="tasks-list-row"
      data-testid={resolvedTestId}
      data-selected={selected ? "true" : undefined}
      onClick={clickable ? () => onSelect?.(task.id) : undefined}
      onKeyDown={handleKeyDown}
      className={cn(
        "group relative flex w-full flex-col gap-2 border-b border-[color:var(--color-divider)] px-4 py-3.5 text-left transition-colors",
        clickable &&
          "cursor-pointer hover:bg-[color:var(--color-surface)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[color:var(--color-accent)]",
        selected && "bg-[color:var(--color-surface)]",
        className
      )}
      {...props}
    >
      {selected ? (
        <span
          aria-hidden="true"
          data-slot="tasks-list-row-indicator"
          className="absolute top-1 bottom-1 left-0 w-[3px] rounded-r bg-[color:var(--color-accent)]"
        />
      ) : null}

      <div className="flex min-w-0 items-center gap-2">
        <Pill.Dot tone={signal.tone} pulse={signal.pulse} />
        <span
          data-slot="tasks-list-row-title"
          className="min-w-0 flex-1 truncate text-[13px] font-medium text-[color:var(--color-text-primary)]"
        >
          {task.title}
        </span>
        {lane ? (
          <Pill
            data-slot="tasks-list-row-lane"
            size="sm"
            tone={pillToneFromLegacyTone(taskLaneTone(lane))}
          >
            {LANE_LABELS[lane] ?? lane}
          </Pill>
        ) : null}
      </div>

      <div className="flex min-w-0 items-center gap-2 text-[11px]">
        <Pill mono>{identifier}</Pill>
        <span
          data-slot="tasks-list-row-timestamp"
          className="font-mono text-[10px] text-[color:var(--color-text-tertiary)]"
        >
          {timestamp}
        </span>
        {trailing !== undefined ? (
          <div
            data-slot="tasks-list-row-trailing"
            className="ml-auto flex shrink-0 items-center gap-1.5"
          >
            {trailing}
          </div>
        ) : null}
      </div>

      {footer !== undefined ? (
        <div data-slot="tasks-list-row-footer" className="flex min-w-0 flex-col gap-1">
          {footer}
        </div>
      ) : null}
    </div>
  );
}

export { TasksListRow };
