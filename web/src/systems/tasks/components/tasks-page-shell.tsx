import { ListChecks } from "lucide-react";
import type { ReactNode } from "react";

import { PageHeader } from "@agh/ui";

export const TASKS_SHELL_TITLE = "Tasks";

interface TasksPageShellProps {
  count?: number;
  controls?: ReactNode;
  meta?: ReactNode;
  children: ReactNode;
}

/**
 * Tasks domain chrome -- composes `@agh/ui` `PageHeader` (icon + title + count +
 * controls + meta) above a flex body. View-switching `Pills`, filter toggles,
 * and create CTAs slot through `controls` / `meta`; the body slot holds the list,
 * Kanban, Dashboard, or Inbox views.
 */
export function TasksPageShell({ count, controls, meta, children }: TasksPageShellProps) {
  return (
    <div className="flex min-h-0 flex-1 flex-col overflow-hidden" data-testid="tasks-shell">
      <PageHeader
        title={<span data-testid="tasks-shell-title">{TASKS_SHELL_TITLE}</span>}
        icon={() => <ListChecks className="size-3.5" data-testid="tasks-shell-icon" />}
        count={count ?? 0}
        controls={controls}
        meta={meta}
      />
      <div className="flex min-h-0 flex-1 flex-col overflow-hidden" data-testid="tasks-shell-body">
        {children}
      </div>
    </div>
  );
}
