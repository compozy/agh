import type { ReactNode } from "react";

import { Eyebrow } from "@agh/ui";
import { cn } from "@/lib/utils";

import type { TaskListGroupId } from "../lib/task-grouping";

export interface TaskGroupProps {
  /** Canonical list-view group id (active / blocked / stuck / queued / done / failed). */
  id: TaskListGroupId;
  /** Group header label, rendered through the canonical `<Eyebrow>` utility. */
  label: string;
  /** Item count rendered next to the label as bare mono `--faint` text. */
  count: number;
  /** Row content for this group (typically `<TasksListRow>` / `<TaskCard>` siblings). */
  children: ReactNode;
  /** Optional right-aligned actions slot (e.g. inline `Add` affordance). */
  actions?: ReactNode;
  className?: string;
}

/**
 * Tasks index — List-view status group. Renders a six-group anatomy per
 * (`Active` · `Blocked` · `Stuck` · `Queued` · `Done` · `Failed`).
 *
 * Header composition: `<Eyebrow>` label + bare mono count + optional actions
 * slot. The proposal's leading colored dot is deliberately omitted on the list
 * view — task rows already render a `<Pill.Dot>` per row, and accent overload
 * on the group header would compete with the row-level signal.
 */
function TaskGroup({ id, label, count, children, actions, className }: TaskGroupProps) {
  return (
    <section
      aria-label={label}
      data-slot="task-group"
      data-group-id={id}
      data-testid={`task-group-${id}`}
      className={cn("flex min-w-0 flex-col gap-1", className)}
    >
      <header className="flex items-center gap-2 px-2 pb-1.5 pt-3" data-slot="task-group-head">
        <Eyebrow data-testid={`task-group-${id}-label`}>{label}</Eyebrow>
        <span
          aria-hidden="true"
          className="font-mono text-[10.5px] tabular-nums text-(--faint)"
          data-slot="task-group-count"
          data-testid={`task-group-${id}-count`}
        >
          {count}
        </span>
        {actions ? (
          <div className="ml-auto flex items-center gap-1" data-slot="task-group-actions">
            {actions}
          </div>
        ) : null}
      </header>
      <div className="flex flex-col" data-slot="task-group-rows">
        {children}
      </div>
    </section>
  );
}

export { TaskGroup };
