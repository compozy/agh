import { Link } from "@tanstack/react-router";
import { AlertCircle, ChevronRight, GitBranch } from "lucide-react";

import { Empty, LinkedRecordTable, Pill } from "@agh/ui";

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
      <Empty
        icon={AlertCircle}
        title="Unable to load dependencies"
        description={errorMessage}
        data-testid="tasks-detail-dependencies-error"
      />
    );
  }

  if (dependencies.length === 0) {
    return (
      <Empty
        icon={GitBranch}
        title="This task has no dependencies"
        data-testid="tasks-detail-dependencies-empty"
      />
    );
  }

  return (
    <LinkedRecordTable
      aria-label="Task dependencies"
      className="w-full gap-6 px-6 py-5"
      columns={["Title", "Owner"]}
      data-testid="tasks-detail-dependencies-panel"
    >
      <LinkedRecordTable.Body>
        {dependencies.map(dep => {
          const target = dep.depends_on;
          const signal = taskStatusSignal(target.status);
          return (
            <LinkedRecordTable.Row
              data-testid={`tasks-detail-dependencies-item-${target.id}`}
              key={target.id}
            >
              <LinkedRecordTable.Cell className="w-8 pl-4">
                <Pill.Dot tone={signal.tone} pulse={signal.pulse} />
              </LinkedRecordTable.Cell>
              <LinkedRecordTable.Cell className="max-w-[360px]">
                <LinkedRecordTable.Title>
                  <span className="truncate text-small-body text-(--fg)">{target.title}</span>
                  <div className="flex flex-wrap items-center gap-1.5 text-eyebrow">
                    <Pill mono>
                      {taskShortId({ id: target.id, identifier: target.identifier })}
                    </Pill>
                    <Pill tone={taskStatusTone(target.status)}>
                      {taskStatusLabel(target.status)}
                    </Pill>
                  </div>
                </LinkedRecordTable.Title>
              </LinkedRecordTable.Cell>
              <LinkedRecordTable.Cell className="text-xs text-(--muted)">
                {taskOwnerLabel(target.owner)}
              </LinkedRecordTable.Cell>
              <LinkedRecordTable.OpenCell>
                <Pill.Link
                  aria-label={`Open dependency ${target.identifier ?? target.id}`}
                  data-testid={`tasks-detail-dependencies-link-${target.id}`}
                  render={<Link params={{ id: target.id }} to="/tasks/$id" />}
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
