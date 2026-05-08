import { Link } from "@tanstack/react-router";
import { AlertCircle, ChevronRight, ListTree } from "lucide-react";

import {
  Empty,
  Pill,
  Section,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@agh/ui";
import { pillToneFromLegacyTone } from "@/lib/pill-variant";

import {
  formatRelativeTime,
  taskOwnerLabel,
  taskPriorityLabel,
  taskPriorityTone,
  taskShortId,
  taskStatusSignal,
  taskStatusTone,
} from "../lib/task-formatters";
import type { TaskChildSummary } from "../types";

export interface TasksDetailChildrenPanelProps {
  items: TaskChildSummary[];
  errorMessage?: string | null;
}

/**
 * Child task table — `Section` + `Table` with `StatusDot` + `MonoBadge` id +
 * status/priority pills + owner + last-activity + deep-link.
 */
export function TasksDetailChildrenPanel({
  items,
  errorMessage = null,
}: TasksDetailChildrenPanelProps) {
  if (errorMessage && items.length === 0) {
    return (
      <Empty
        icon={AlertCircle}
        title="Unable to load children"
        description={errorMessage}
        data-testid="tasks-detail-children-error"
      />
    );
  }

  if (items.length === 0) {
    return (
      <Empty
        icon={ListTree}
        title="This task has no children"
        data-testid="tasks-detail-children-empty"
      />
    );
  }

  return (
    <Section
      aria-label="Child tasks"
      className="w-full gap-6 px-6 py-5"
      data-testid="tasks-detail-children-panel"
    >
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead />
            <TableHead>Title</TableHead>
            <TableHead>Owner</TableHead>
            <TableHead>Updated</TableHead>
            <TableHead className="w-8" />
          </TableRow>
        </TableHeader>
        <TableBody>
          {items.map(child => {
            const signal = taskStatusSignal(child.status);
            return (
              <TableRow data-testid={`tasks-detail-children-item-${child.id}`} key={child.id}>
                <TableCell className="w-8 pl-4">
                  <Pill.Dot tone={signal.tone} pulse={signal.pulse} />
                </TableCell>
                <TableCell className="max-w-[360px]">
                  <div className="flex min-w-0 flex-col gap-1">
                    <span className="truncate text-small-body text-(--color-text-primary)">
                      {child.title}
                    </span>
                    <div className="flex flex-wrap items-center gap-1.5 text-eyebrow">
                      <Pill mono>
                        {taskShortId({ id: child.id, identifier: child.identifier })}
                      </Pill>
                      <Pill tone={pillToneFromLegacyTone(taskStatusTone(child.status))}>
                        {child.status}
                      </Pill>
                      {child.priority ? (
                        <Pill tone={pillToneFromLegacyTone(taskPriorityTone(child.priority))}>
                          {taskPriorityLabel(child.priority)}
                        </Pill>
                      ) : null}
                    </div>
                  </div>
                </TableCell>
                <TableCell className="text-xs text-(--color-text-secondary)">
                  {taskOwnerLabel(child.owner)}
                </TableCell>
                <TableCell className="font-mono text-eyebrow text-(--color-text-tertiary)">
                  {child.last_activity_at ? formatRelativeTime(child.last_activity_at) : "—"}
                </TableCell>
                <TableCell className="w-8 pr-4">
                  <Link
                    aria-label={`Open task ${child.identifier ?? child.id}`}
                    className="inline-flex items-center gap-1 font-mono text-badge uppercase tracking-mono text-accent hover:underline"
                    data-testid={`tasks-detail-children-link-${child.id}`}
                    params={{ id: child.id }}
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
