import type { ReactNode } from "react";

import { Eyebrow, StatusDot } from "@agh/ui";
import { cn } from "@/lib/utils";

import { listGroupDotProps, type TaskListGroupId } from "../lib/task-grouping";

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
 * Header composition: `<StatusDot>` + `<Eyebrow>` label + bare mono count +
 * optional actions slot. One dot per group replaces per-row status dots on the
 * list surface (see `TasksListRow` `showStatusDot`).
 */
function TaskGroup({ id, label, count, children, actions, className }: TaskGroupProps) {
  const { tone, variant } = listGroupDotProps(id);

  return (
    <section
      aria-label={label}
      data-slot="task-group"
      data-group-id={id}
      data-testid={`task-group-${id}`}
      className={cn("flex min-w-0 flex-col gap-1", className)}
    >
      <header className="flex items-center gap-2 px-3 pb-2 pt-4" data-slot="task-group-head">
        <StatusDot
          data-testid={`task-group-dot-${id}`}
          label={label}
          tone={tone}
          variant={variant}
        />
        <Eyebrow className="text-muted" data-testid={`task-group-${id}-label`}>
          {label}
        </Eyebrow>
        <span
          aria-hidden="true"
          className="font-mono text-mono-id tabular-nums text-faint"
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
