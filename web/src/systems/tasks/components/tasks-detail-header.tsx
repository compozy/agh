import { Link, useRouter } from "@tanstack/react-router";

import { DetailHeader, Pill } from "@agh/ui";

import { taskShortId, taskStatusSignal } from "../lib/task-formatters";
import type { TaskDetailView } from "../types";
import {
  TasksDetailHeaderActions,
  TasksDetailHeaderMeta,
  TasksDetailHeaderPills,
} from "./tasks-detail-header-sections";

export interface TasksDetailHeaderProps {
  detail: TaskDetailView;
  pending?: {
    delete?: boolean;
    publish?: boolean;
    cancel?: boolean;
    enqueue?: boolean;
    pause?: boolean;
    resume?: boolean;
    recover?: boolean;
  };
  onDelete?: (taskId: string) => void;
  onPublish?: () => void;
  onCancel?: () => void;
  onEnqueueRun?: () => void;
  onPause?: (reason: string) => void | Promise<void>;
  onResume?: () => void | Promise<void>;
  onRecover?: () => void | Promise<void>;
}

/** `/tasks/$id` hero composed from the canonical DetailHeader anatomy. */
export function TasksDetailHeader({
  detail,
  pending,
  onDelete,
  onPublish,
  onCancel,
  onEnqueueRun,
  onPause,
  onResume,
  onRecover,
}: TasksDetailHeaderProps) {
  const router = useRouter();
  const record = detail.task;
  const identifier = taskShortId(record);
  const signal = taskStatusSignal(record.status);

  return (
    <DetailHeader
      data-testid="tasks-detail-header"
      back={() => router.history.back()}
      backLabel="Back to tasks"
      crumbs={
        <span data-testid="tasks-detail-breadcrumb" className="inline-flex items-center gap-1.5">
          <Link
            data-testid="tasks-detail-breadcrumb-tasks"
            to="/tasks"
            className="transition-colors duration-base ease-out hover:text-fg"
          >
            Tasks
          </Link>
          <span aria-hidden="true" className="text-faint">
            ·
          </span>
          <span>{identifier}</span>
        </span>
      }
      preTitle="Task"
      title={
        <span data-testid="tasks-detail-title" className="inline-flex min-w-0 items-center gap-2">
          <Pill.Dot tone={signal.tone} pulse={signal.pulse} />
          <span className="truncate">{record.title}</span>
        </span>
      }
      pills={<TasksDetailHeaderPills detail={detail} />}
      meta={<TasksDetailHeaderMeta detail={detail} />}
      actions={
        <TasksDetailHeaderActions
          detail={detail}
          pending={pending}
          onCancel={onCancel}
          onDelete={onDelete}
          onEnqueueRun={onEnqueueRun}
          onPause={onPause}
          onPublish={onPublish}
          onRecover={onRecover}
          onResume={onResume}
        />
      }
    />
  );
}
