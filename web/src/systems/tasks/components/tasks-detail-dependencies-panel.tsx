import { Link } from "@tanstack/react-router";
import { ChevronRight } from "lucide-react";

import { Pill } from "@/components/design-system";

import { taskOwnerLabel, taskStatusLabel, taskStatusTone } from "../lib/task-formatters";
import type { TaskDetailView } from "../types";

type DependencyReference = NonNullable<TaskDetailView["dependency_references"]>[number];

export interface TasksDetailDependenciesPanelProps {
  dependencies: DependencyReference[];
  errorMessage?: string | null;
}

export function TasksDetailDependenciesPanel({
  dependencies,
  errorMessage = null,
}: TasksDetailDependenciesPanelProps) {
  if (errorMessage && dependencies.length === 0) {
    return (
      <div
        className="flex min-h-[200px] items-center justify-center px-6 text-center text-sm text-[color:var(--color-danger)]"
        data-testid="tasks-detail-dependencies-error"
      >
        {errorMessage}
      </div>
    );
  }

  if (dependencies.length === 0) {
    return (
      <div
        className="flex min-h-[200px] items-center justify-center px-6 text-center text-sm text-[color:var(--color-text-secondary)]"
        data-testid="tasks-detail-dependencies-empty"
      >
        This task has no dependencies.
      </div>
    );
  }

  return (
    <section
      aria-label="Task dependencies"
      className="flex min-h-0 flex-1 flex-col"
      data-testid="tasks-detail-dependencies-panel"
    >
      <ol className="flex flex-col divide-y divide-[color:var(--color-divider)]">
        {dependencies.map(dep => {
          const target = dep.depends_on;
          return (
            <li
              className="flex items-center gap-3 px-6 py-3 hover:bg-[color:var(--color-surface)]"
              data-testid={`tasks-detail-dependencies-item-${target.id}`}
              key={target.id}
            >
              <div className="min-w-0 flex-1">
                <div className="flex flex-wrap items-center gap-2 text-xs text-[color:var(--color-text-secondary)]">
                  <Pill emphasis="strong" kind="state" tone={taskStatusTone(target.status)}>
                    {taskStatusLabel(target.status)}
                  </Pill>
                  <span className="font-mono text-[color:var(--color-text-primary)]">
                    {target.identifier ?? target.id}
                  </span>
                  <span>· Owner {taskOwnerLabel(target.owner)}</span>
                </div>
                <p className="mt-1 truncate text-sm text-[color:var(--color-text-primary)]">
                  {target.title}
                </p>
              </div>
              <Link
                aria-label={`Open dependency ${target.identifier ?? target.id}`}
                className="flex shrink-0 items-center gap-1 font-mono text-[0.66rem] uppercase tracking-[0.14em] text-[color:var(--color-accent)] hover:underline"
                data-testid={`tasks-detail-dependencies-link-${target.id}`}
                params={{ id: target.id }}
                to="/tasks/$id"
              >
                Open
                <ChevronRight className="size-3" />
              </Link>
            </li>
          );
        })}
      </ol>
    </section>
  );
}
