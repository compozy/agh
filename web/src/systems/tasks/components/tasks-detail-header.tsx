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

/**
 * Task detail body under the unified window head — pills + meta only.
 * Identity and trail actions publish through `TasksDetailHeaderActions` → `useTopbarSlot`.
 */
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
  return (
    <div className="flex flex-col gap-2 pt-4" data-testid="tasks-detail-header">
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
      <div className="flex flex-wrap items-center gap-1.5">
        <TasksDetailHeaderPills detail={detail} />
      </div>
      <TasksDetailHeaderMeta detail={detail} />
    </div>
  );
}
