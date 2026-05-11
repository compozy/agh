import { Link } from "@tanstack/react-router";
import { AlertCircle, ChevronRight, ListTree } from "lucide-react";
import { Fragment } from "react";

import { Empty, LinkedRecordTable, MonoId, Pill, Time } from "@agh/ui";
import { TASK_STATUS_TONE, type TaskStatus as UiTaskStatus } from "@/lib/status-tone";

import {
  taskOwnerLabel,
  taskPriorityLabel,
  taskPriorityTone,
  taskStatusLabel,
} from "../lib/task-formatters";
import type { TaskChildSummary } from "../types";

export interface TasksDetailChildrenPanelProps {
  items: TaskChildSummary[];
  errorMessage?: string | null;
}

const ACTIVE_RUN_STATUSES = new Set(["running", "starting", "claimed"]);

/**
 * UI-only derivation: a child has running descendants when it owns descendants
 * (`child_count > 0`) AND its own active run / status indicates active
 * execution. The backend exposes neither a `has_running_descendants` flag nor
 * descendant-state, so the row strip lights only when the child itself is
 * orchestrating something. Per ADR-007 §9.
 */
function hasRunningDescendants(child: TaskChildSummary): boolean {
  if ((child.child_count ?? 0) <= 0) return false;
  if (child.active_run && ACTIVE_RUN_STATUSES.has(child.active_run.status)) return true;
  return child.status === "in_progress";
}

/**
 * Child task table — STATUS column header per ADR-007 §9 + 4 px progress strip
 * below the row when the child has running descendants.
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
      columns={["Title", "Status", "Owner", "Updated"]}
      data-testid="tasks-detail-children-panel"
    >
      <LinkedRecordTable.Body>
        {items.map(child => {
          const tone = TASK_STATUS_TONE[child.status as UiTaskStatus] ?? "neutral";
          const showProgress = hasRunningDescendants(child);
          return (
            <Fragment key={child.id}>
              <LinkedRecordTable.Row data-testid={`tasks-detail-children-item-${child.id}`}>
                <LinkedRecordTable.Cell className="w-8 pl-4" />
                <LinkedRecordTable.Cell className="max-w-[360px]">
                  <LinkedRecordTable.Title>
                    <span className="truncate text-small-body text-(--fg)">{child.title}</span>
                    <div className="flex flex-wrap items-center gap-1.5 text-eyebrow">
                      <MonoId value={child.identifier ?? child.id} />
                      {child.priority ? (
                        <Pill tone={taskPriorityTone(child.priority)}>
                          {taskPriorityLabel(child.priority)}
                        </Pill>
                      ) : null}
                    </div>
                  </LinkedRecordTable.Title>
                </LinkedRecordTable.Cell>
                <LinkedRecordTable.Cell>
                  <Pill data-testid={`tasks-detail-children-status-${child.id}`} tone={tone}>
                    {taskStatusLabel(child.status)}
                  </Pill>
                </LinkedRecordTable.Cell>
                <LinkedRecordTable.Cell className="text-xs text-(--muted)">
                  {taskOwnerLabel(child.owner)}
                </LinkedRecordTable.Cell>
                <LinkedRecordTable.Cell className="font-mono text-eyebrow text-(--subtle)">
                  {child.last_activity_at ? (
                    <Time iso={child.last_activity_at} mode="relative" />
                  ) : (
                    "--"
                  )}
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
              {showProgress ? (
                <tr
                  data-testid={`tasks-detail-children-progress-row-${child.id}`}
                  className="border-b border-(--line)"
                >
                  <td colSpan={6} className="p-0">
                    <div
                      aria-hidden="true"
                      className="h-1 w-full overflow-hidden bg-(--accent-tint)"
                      data-testid={`tasks-detail-children-progress-${child.id}`}
                    >
                      <div className="h-full w-1/3 animate-pulse bg-(--accent)" />
                    </div>
                  </td>
                </tr>
              ) : null}
            </Fragment>
          );
        })}
      </LinkedRecordTable.Body>
    </LinkedRecordTable>
  );
}
