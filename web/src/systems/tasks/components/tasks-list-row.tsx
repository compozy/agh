import * as React from "react";

import { MonoId, Pill, StatusDot, type StatusDotTone } from "@agh/ui";
import { cn } from "@/lib/utils";

import {
  formatRelativeTime,
  taskLaneTone,
  taskShortId,
  taskStatusSignal,
} from "../lib/task-formatters";
import type { TaskInboxLane, TaskListItem } from "../types";

export interface TasksListRowProps extends Omit<
  React.ComponentProps<"button">,
  "onClick" | "onSelect" | "type"
> {
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
  /** When false, omits the leading status-dot column entirely (list view default). */
  showStatusDot?: boolean;
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
    <span aria-hidden="true" className="text-faint opacity-60" data-slot="tasks-list-row-meta-sep">
      ·
    </span>
  );
}

function joinMeta(children: React.ReactNode): React.ReactNode[] {
  const items = React.Children.toArray(children);
  return items.flatMap((child, index) => {
    if (index === 0) return [child];
    const childKey = React.isValidElement(child) && child.key !== null ? child.key : String(child);
    return [<MetaSeparator key={`sep-${childKey}`} />, child];
  });
}

/**
 * Maps the `taskStatusSignal` tone to a `<StatusDot>` tone. Returns `null`
 * when no dot should render so the row keeps a uniform 14 px leading column
 * width across all statuses while neutral/normal states stay decoration-free
 * (DESIGN.md §2.7 "color is signal, never decoration").
 */
function rowStatusDotTone(tone: ReturnType<typeof taskStatusSignal>["tone"]): StatusDotTone | null {
  switch (tone) {
    case "warning":
      return "warning";
    case "danger":
      return "danger";
    case "accent":
      return "accent";
    default:
      return null;
  }
}

/**
 * Shared list-row primitive built on the proposal's `.task-row` grammar
 * (`docs/design/new-proposal/agh-refined-7.html` lines 260-269). 3-column grid:
 * `[ status-dot? ] [ main (title + meta) ] [ trailing ]`. Identifier renders
 * via `<MonoId>` per the row-context contract. Optional `meta` slot accepts
 * ReactNodes joined by `·` separators inline. The Kanban and Inbox build
 * dedicated row primitives instead of reusing this one.
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
  showStatusDot = false,
  className,
  ...props
}: TasksListRowProps) {
  const signal = showStatusDot ? taskStatusSignal(task.status) : null;
  const dotTone = signal === null ? null : rowStatusDotTone(signal.tone);
  const identifier = taskShortId(task);
  const lastActivity = task.last_activity_at ?? task.updated_at;
  const timestamp = formatRelativeTime(lastActivity);
  const resolvedTestId = testId ?? `task-card-${task.id}`;

  const clickable = onSelect !== undefined;
  const showRail = rail || selected;

  return (
    <button
      aria-pressed={selected}
      data-slot="tasks-list-row"
      data-selected={selected ? "true" : undefined}
      data-status={task.status}
      data-testid={resolvedTestId}
      disabled={!clickable}
      onClick={clickable ? () => onSelect?.(task.id) : undefined}
      type="button"
      className={cn(
        "relative grid items-center gap-3 border-b border-line-soft py-2.5 pr-3 pl-3.5 text-left transition-colors duration-base ease-out",
        showStatusDot ? "grid-cols-[10px_minmax(0,1fr)_auto]" : "grid-cols-[minmax(0,1fr)_auto]",
        clickable &&
          "cursor-pointer hover:bg-row-hover focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-line-strong focus-visible:ring-inset",
        selected && "bg-row-selected",
        className
      )}
      {...props}
    >
      {showRail ? (
        <span
          aria-hidden="true"
          className="absolute top-2 bottom-2 left-0 w-0.5 rounded-tr-xs rounded-br-xs bg-fg-strong"
        />
      ) : null}

      {showStatusDot ? (
        <span
          aria-hidden={dotTone === null ? "true" : undefined}
          className="flex shrink-0 items-center justify-center"
          data-slot="tasks-list-row-dot"
        >
          {dotTone === null ? null : (
            <StatusDot
              tone={dotTone}
              variant={signal?.pulse ? "ring" : "solid"}
              size="default"
              label={signal?.pulse ? "Running" : undefined}
            />
          )}
        </span>
      ) : null}

      <div className="flex min-w-0 flex-col gap-1">
        <div className="flex min-w-0 items-center gap-2">
          <h3
            className="min-w-0 max-w-full truncate text-small-body font-medium text-fg-strong"
            data-slot="tasks-list-row-title"
          >
            {task.title}
          </h3>
          {lane ? (
            <Pill data-slot="tasks-list-row-lane" size="xs" tone={taskLaneTone(lane)}>
              {LANE_LABELS[lane] ?? lane}
            </Pill>
          ) : null}
        </div>

        <div
          className="flex min-w-0 flex-wrap items-center gap-2 text-small-body text-faint"
          data-slot="tasks-list-row-meta"
        >
          <MonoId value={identifier} size="sm" data-slot="tasks-list-row-id" />
          <MetaSeparator />
          <span
            className="font-mono text-badge tabular-nums text-faint"
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
        <div className="flex shrink-0 items-center gap-2" data-slot="tasks-list-row-trailing">
          {trailing}
        </div>
      ) : null}
    </button>
  );
}

export { TasksListRow };
