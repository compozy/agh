import { PageHead, Pill } from "@agh/ui";

import { taskStatusSignal } from "../lib/task-formatters";
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

/** `/tasks/$id` hero rendering identity via PageHead; actions publish to the topbar. */
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
  const record = detail.task;
  const signal = taskStatusSignal(record.status);

  return (
    <div className="pt-5">
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
      <PageHead
        data-testid="tasks-detail-header"
        pretitle="Task"
        title={
          <span data-testid="tasks-detail-title" className="inline-flex min-w-0 items-center gap-2">
            <Pill.Dot tone={signal.tone} pulse={signal.pulse} />
            <span className="truncate">{record.title}</span>
          </span>
        }
        variant="detail"
        pills={<TasksDetailHeaderPills detail={detail} />}
        meta={<TasksDetailHeaderMeta detail={detail} />}
      />
    </div>
  );
}
