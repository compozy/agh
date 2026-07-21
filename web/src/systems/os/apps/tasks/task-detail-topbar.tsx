import { useTopbarSlot } from "@agh/ui";

import { TaskPageActions, TaskPageOverflow, TaskPageStatus } from "@/systems/tasks";

import type { TaskDetailLocationController } from "./use-task-detail-location";

/** Publishes the task-detail head (crumbs · status · §6 actions) into the window topbar. */
export function TaskDetailTopbar({ controller }: { controller: TaskDetailLocationController }) {
  const { page, record, command } = controller;

  useTopbarSlot(
    record && command
      ? {
          onBack: controller.backToTasks,
          crumbs: [
            {
              id: "tasks",
              label: <span data-testid="tasks-detail-breadcrumb-tasks">Tasks</span>,
              onSelect: controller.backToTasks,
            },
          ],
          crumb: <span data-testid="tasks-detail-title">{record.title}</span>,
          status: <TaskPageStatus status={record.status} />,
          actions: (
            <TaskPageActions
              command={command}
              handlers={{
                onPublish: () => void page.handlePublishTask(),
                onApprove: () => void page.handleApproveTask(),
                onStartRun: () => void page.handleEnqueueRun(),
                onOpenRun: controller.openRun,
                onResume: () => void page.handleResumeTask(),
                onRecover: () => void page.handleRecoverTask(),
                onRetry: runId => void page.handleRetryRun(runId),
                onEdit: controller.openEdit,
              }}
              pending={
                page.isPublishPending ||
                page.isApprovePending ||
                page.isEnqueuePending ||
                page.isResumePending ||
                page.isRecoverPending ||
                page.isRetryPending
              }
            />
          ),
          overflow: (
            <TaskPageOverflow
              command={command}
              onCancel={() => void page.handleCancelTask()}
              onCopyId={controller.copyTaskId}
              onDelete={() => controller.setDeleteOpen(true)}
              onFanOut={() => controller.setFanOutOpen(true)}
              onPause={controller.pauseDialog.open}
              onResume={() => void page.handleResumeTask()}
              onStartNewRun={() => void page.handleEnqueueRun()}
              pending={{
                cancel: page.isCancelPending,
                pause: page.isPausePending,
                resume: page.isResumePending,
                enqueue: page.isEnqueuePending,
              }}
              showFanOut={controller.showFanOut}
              taskId={record.id}
            />
          ),
        }
      : null
  );

  return null;
}
