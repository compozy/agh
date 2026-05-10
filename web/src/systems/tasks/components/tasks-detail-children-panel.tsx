import { Link } from "@tanstack/react-router";
import { AlertCircle, ChevronRight, ListTree } from "lucide-react";

import { Empty, LinkedRecordTable, Pill } from "@agh/ui";
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
 * Child task table -- `LinkedRecordTable` with `Pill.Dot` + `Pill` id +
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
    <LinkedRecordTable
      aria-label="Child tasks"
      className="w-full gap-6 px-6 py-5"
      columns={["Title", "Owner", "Updated"]}
      data-testid="tasks-detail-children-panel"
    >
      <LinkedRecordTable.Body>
        {items.map(child => {
          const signal = taskStatusSignal(child.status);
          return (
            <LinkedRecordTable.Row
              data-testid={`tasks-detail-children-item-${child.id}`}
              key={child.id}
            >
              <LinkedRecordTable.Cell className="w-8 pl-4">
                <Pill.Dot tone={signal.tone} pulse={signal.pulse} />
              </LinkedRecordTable.Cell>
              <LinkedRecordTable.Cell className="max-w-[360px]">
                <LinkedRecordTable.Title>
                  <span className="truncate text-small-body text-(--fg)">{child.title}</span>
                  <div className="flex flex-wrap items-center gap-1.5 text-eyebrow">
                    <Pill mono>{taskShortId({ id: child.id, identifier: child.identifier })}</Pill>
                    <Pill tone={pillToneFromLegacyTone(taskStatusTone(child.status))}>
                      {child.status}
                    </Pill>
                    {child.priority ? (
                      <Pill tone={pillToneFromLegacyTone(taskPriorityTone(child.priority))}>
                        {taskPriorityLabel(child.priority)}
                      </Pill>
                    ) : null}
                  </div>
                </LinkedRecordTable.Title>
              </LinkedRecordTable.Cell>
              <LinkedRecordTable.Cell className="text-xs text-(--muted)">
                {taskOwnerLabel(child.owner)}
              </LinkedRecordTable.Cell>
              <LinkedRecordTable.Cell className="font-mono text-eyebrow text-(--subtle)">
                {child.last_activity_at ? formatRelativeTime(child.last_activity_at) : "--"}
              </LinkedRecordTable.Cell>
              <LinkedRecordTable.OpenCell>
                <Pill.Link
                  aria-label={`Open task ${child.identifier ?? child.id}`}
                  data-testid={`tasks-detail-children-link-${child.id}`}
                  render={<Link params={{ id: child.id }} to="/tasks/$id" />}
                >
                  Open <ChevronRight className="size-3" />
                </Pill.Link>
              </LinkedRecordTable.OpenCell>
            </LinkedRecordTable.Row>
          );
        })}
      </LinkedRecordTable.Body>
    </LinkedRecordTable>
  );
}
