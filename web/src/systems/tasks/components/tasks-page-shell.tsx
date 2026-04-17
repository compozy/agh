import { ListChecks } from "lucide-react";
import type { ReactNode } from "react";

import { WorkspacePageShell } from "@/systems/workspace/components/workspace-page-shell";

export const TASKS_SHELL_TITLE = "Tasks";

interface TasksPageShellProps {
  count?: number;
  controls?: ReactNode;
  meta?: ReactNode;
  children: ReactNode;
}

export function TasksPageShell({ count, controls, meta, children }: TasksPageShellProps) {
  return (
    <WorkspacePageShell
      title={TASKS_SHELL_TITLE}
      icon={<ListChecks className="size-4" data-testid="tasks-shell-icon" />}
      count={count ?? 0}
      controls={controls}
      meta={meta}
    >
      <div className="flex min-h-0 flex-1 flex-col" data-testid="tasks-shell-body">
        {children}
      </div>
    </WorkspacePageShell>
  );
}
