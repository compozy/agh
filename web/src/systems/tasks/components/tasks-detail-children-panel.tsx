import { Link } from "@tanstack/react-router";
import { ChevronRight } from "lucide-react";

import { Pill } from "@agh/ui";

import {
  formatRelativeTime,
  taskOwnerLabel,
  taskPriorityLabel,
  taskPriorityTone,
  taskStatusLabel,
  taskStatusTone,
} from "../lib/task-formatters";
import type { TaskChildSummary } from "../types";

import { pillVariantFromTone } from "@/lib/pill-variant";
export interface TasksDetailChildrenPanelProps {
  items: TaskChildSummary[];
  errorMessage?: string | null;
}

export function TasksDetailChildrenPanel({
  items,
  errorMessage = null,
}: TasksDetailChildrenPanelProps) {
  if (errorMessage && items.length === 0) {
    return (
      <div
        className="flex min-h-[200px] items-center justify-center px-6 text-center text-sm text-[color:var(--color-danger)]"
        data-testid="tasks-detail-children-error"
      >
        {errorMessage}
      </div>
    );
  }

  if (items.length === 0) {
    return (
      <div
        className="flex min-h-[200px] items-center justify-center px-6 text-center text-sm text-[color:var(--color-text-secondary)]"
        data-testid="tasks-detail-children-empty"
      >
        This task has no children.
      </div>
    );
  }

  return (
    <section
      aria-label="Child tasks"
      className="flex min-h-0 flex-1 flex-col"
      data-testid="tasks-detail-children-panel"
    >
      <ol className="flex flex-col divide-y divide-[color:var(--color-divider)]">
        {items.map(child => (
          <li
            className="flex items-center gap-3 px-6 py-3 hover:bg-[color:var(--color-surface)]"
            data-testid={`tasks-detail-children-item-${child.id}`}
            key={child.id}
          >
            <div className="min-w-0 flex-1">
              <div className="flex flex-wrap items-center gap-2 text-xs text-[color:var(--color-text-secondary)]">
                <Pill variant={pillVariantFromTone(taskStatusTone(child.status))}>
                  {taskStatusLabel(child.status)}
                </Pill>
                {child.priority ? (
                  <Pill variant={pillVariantFromTone(taskPriorityTone(child.priority))}>
                    {taskPriorityLabel(child.priority)}
                  </Pill>
                ) : null}
                <span className="font-mono text-[color:var(--color-text-primary)]">
                  {child.identifier ?? child.id}
                </span>
                <span>· Owner {taskOwnerLabel(child.owner)}</span>
                {child.last_activity_at ? (
                  <span>· Updated {formatRelativeTime(child.last_activity_at)}</span>
                ) : null}
              </div>
              <p className="mt-1 truncate text-sm text-[color:var(--color-text-primary)]">
                {child.title}
              </p>
            </div>
            <Link
              aria-label={`Open task ${child.identifier ?? child.id}`}
              className="flex shrink-0 items-center gap-1 font-mono text-[0.66rem] uppercase tracking-[0.14em] text-[color:var(--color-accent)] hover:underline"
              data-testid={`tasks-detail-children-link-${child.id}`}
              params={{ id: child.id }}
              to="/tasks/$id"
            >
              Open
              <ChevronRight className="size-3" />
            </Link>
          </li>
        ))}
      </ol>
    </section>
  );
}
