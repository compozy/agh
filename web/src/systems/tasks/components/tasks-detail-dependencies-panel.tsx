import { Link } from "@tanstack/react-router";
import { ChevronRight } from "lucide-react";

import {
  MonoBadge,
  Pill,
  Section,
  StatusDot,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@agh/ui";
import { pillVariantFromTone } from "@/lib/pill-variant";

import {
  taskOwnerLabel,
  taskShortId,
  taskStatusLabel,
  taskStatusSignal,
  taskStatusTone,
} from "../lib/task-formatters";
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
    <Section
      aria-label="Task dependencies"
      className="w-full gap-6 px-6 py-5"
      data-testid="tasks-detail-dependencies-panel"
    >
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead />
            <TableHead>Title</TableHead>
            <TableHead>Owner</TableHead>
            <TableHead className="w-8" />
          </TableRow>
        </TableHeader>
        <TableBody>
          {dependencies.map(dep => {
            const target = dep.depends_on;
            const signal = taskStatusSignal(target.status);
            return (
              <TableRow data-testid={`tasks-detail-dependencies-item-${target.id}`} key={target.id}>
                <TableCell className="w-8 pl-4">
                  <StatusDot tone={signal.tone} pulse={signal.pulse} />
                </TableCell>
                <TableCell className="max-w-[360px]">
                  <div className="flex min-w-0 flex-col gap-1">
                    <span className="truncate text-[13px] text-[color:var(--color-text-primary)]">
                      {target.title}
                    </span>
                    <div className="flex flex-wrap items-center gap-1.5 text-[11px]">
                      <MonoBadge>
                        {taskShortId({ id: target.id, identifier: target.identifier })}
                      </MonoBadge>
                      <Pill variant={pillVariantFromTone(taskStatusTone(target.status))}>
                        {taskStatusLabel(target.status)}
                      </Pill>
                    </div>
                  </div>
                </TableCell>
                <TableCell className="text-[12px] text-[color:var(--color-text-secondary)]">
                  {taskOwnerLabel(target.owner)}
                </TableCell>
                <TableCell className="w-8 pr-4">
                  <Link
                    aria-label={`Open dependency ${target.identifier ?? target.id}`}
                    className="inline-flex items-center gap-1 font-mono text-[10px] uppercase tracking-[0.14em] text-[color:var(--color-accent)] hover:underline"
                    data-testid={`tasks-detail-dependencies-link-${target.id}`}
                    params={{ id: target.id }}
                    to="/tasks/$id"
                  >
                    Open <ChevronRight className="size-3" />
                  </Link>
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </Section>
  );
}
