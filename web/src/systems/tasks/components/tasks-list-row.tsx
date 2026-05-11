import * as React from "react";

import { Pill } from "@agh/ui";
import { cn } from "@/lib/utils";

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
  rail?: boolean;
  onSelect?: (taskId: string) => void;
  /** Optional lane badge -- used by the Inbox (task 18) to tag rows by lane. */
  lane?: TaskInboxLane | null;
  /** Optional slot rendered as the right column (auto-width). */
  trailing?: React.ReactNode;
  /**
   * Optional inline meta line rendered under the title row. Each child should
   * be a span; the row inserts `·` separators between adjacent children.
   * Mirrors `.task-row__meta` from the proposal.
   */
  meta?: React.ReactNode;
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

function MetaSeparator() {
  return (
    <span
      aria-hidden="true"
      className="text-(--faint) opacity-60"
      data-slot="tasks-list-row-meta-sep"
    >
      ·
    </span>
  );
}

function joinMeta(children: React.ReactNode): React.ReactNode[] {
  const items = React.Children.toArray(children);
  return items.flatMap((child, index) =>
    index === 0 ? [child] : [<MetaSeparator key={`sep-${index}`} />, child]
  );
}

/**
 * Shared list-row primitive built on the proposal's `.task-row` grammar
 * (`docs/design/new-proposal/agh-refined-7.html` lines 260-269). 3-column grid:
 * `[ status-dot ] [ main (title + meta) ] [ trailing ]`. Identifier renders
 * as bare mono 10.5 px (NOT a `<Pill>`) per `.task-row__id`. Optional `meta`
 * slot accepts ReactNodes joined by `·` separators inline. The Kanban and
 * Inbox build dedicated row primitives instead of reusing this one.
 */
function TasksListRow({
  task,
  selected = false,
  rail = false,
  onSelect,
  lane = null,
  trailing,
  meta,
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

  const showRail = rail || selected;

  return (
    <div
      aria-pressed={selected}
      data-slot="tasks-list-row"
      data-selected={selected ? "true" : undefined}
      data-status={task.status}
      data-testid={resolvedTestId}
      onClick={clickable ? () => onSelect?.(task.id) : undefined}
      onKeyDown={handleKeyDown}
      role={clickable ? "button" : undefined}
      tabIndex={clickable ? 0 : undefined}
      className={cn(
        "relative grid grid-cols-[14px_minmax(0,1fr)_auto] items-center gap-[14px] border-b border-(--line-soft) py-[11px] pr-[10px] pl-[14px] text-left transition-colors duration-(--dur) ease-(--ease)",
        clickable &&
          "cursor-pointer hover:bg-(--row-hover) focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-(--line-strong) focus-visible:ring-inset",
        selected && "bg-(--row-selected)",
        className
      )}
      {...props}
    >
      {showRail ? (
        <span
          aria-hidden="true"
          className="absolute top-[8px] bottom-[8px] left-0 w-[2px] rounded-tr-[2px] rounded-br-[2px] bg-(--accent)"
        />
      ) : null}

      <span className="flex shrink-0 items-center justify-center">
        <Pill.Dot pulse={signal.pulse} tone={signal.tone} />
      </span>

      <div className="flex min-w-0 flex-col gap-1">
        <div className="flex min-w-0 flex-wrap items-center gap-2">
          <h3
            className="min-w-0 max-w-full truncate text-[13px] font-medium tracking-[-0.014em] text-(--fg-strong)"
            data-slot="tasks-list-row-title"
          >
            {task.title}
          </h3>
          {lane ? (
            <Pill data-slot="tasks-list-row-lane" size="sm" tone={taskLaneTone(lane)}>
              {LANE_LABELS[lane] ?? lane}
            </Pill>
          ) : null}
        </div>

        <div className="flex min-w-0 flex-wrap items-center gap-[9px] text-[11.5px] tracking-[-0.005em] text-(--subtle)">
          <span className="font-mono text-[10.5px] text-(--faint)" data-slot="tasks-list-row-id">
            {identifier}
          </span>
          <MetaSeparator />
          <span
            className="font-mono text-[10.5px] tabular-nums text-(--faint)"
            data-slot="tasks-list-row-timestamp"
          >
            {timestamp}
          </span>
          {meta !== undefined ? (
            <>
              <MetaSeparator />
              {joinMeta(meta)}
            </>
          ) : null}
        </div>
      </div>

      {trailing !== undefined ? (
        <div className="flex shrink-0 items-center gap-1.5" data-slot="tasks-list-row-trailing">
          {trailing}
        </div>
      ) : null}
    </div>
  );
}

export { TasksListRow };
