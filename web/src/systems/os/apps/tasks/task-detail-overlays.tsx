import {
  TaskDeleteAction,
  TaskFanOutDialog,
  TaskInspectDrawer,
  TaskPauseDialog,
  TaskSetupSheet,
} from "@/systems/tasks";

import type { TaskDetailLocationController } from "./use-task-detail-location";

/** Window-scoped overlays for the task-detail location (dialogs, sheets, drawer). */
export function TaskDetailOverlays({ controller }: { controller: TaskDetailLocationController }) {
  const { page, operator, detail, record } = controller;
  if (!detail || !record) return null;

  return (
    <>
      <TaskPauseDialog dialog={controller.pauseDialog} isPending={page.isPausePending} />
      <TaskDeleteAction
        cancelTestId="tasks-detail-delete-cancel"
        confirmTestId="tasks-detail-delete-confirm"
        dialogTestId="tasks-detail-delete-dialog"
        hideTrigger
        isPending={controller.deleteMutation.isPending}
        onDelete={controller.handleDeleteTask}
        onOpenChange={controller.setDeleteOpen}
        open={controller.deleteOpen}
        taskId={record.id}
        taskTitle={record.title}
        triggerTestId="tasks-detail-delete-trigger"
      />
      <TaskFanOutDialog
        isPending={page.isFanOutPending}
        onFanOut={page.handleFanOutRuns}
        onOpenChange={controller.setFanOutOpen}
        open={controller.fanOutOpen}
      />
      <TaskSetupSheet
        hasActiveRun={Boolean(page.activeRun)}
        isDeletePending={operator.isDeleteProfilePending}
        isSetPending={operator.isSetProfilePending}
        onDeleteProfile={operator.handleDeleteProfile}
        onOpenChange={controller.setSetupOpen}
        onSetProfile={operator.handleSetProfile}
        open={controller.setupOpen}
        profile={page.profile}
        profileErrorMessage={page.profileError?.message ?? null}
        profileLoading={page.profileLoading}
        taskId={controller.taskId}
      />
      <TaskInspectDrawer
        bridges={{
          subscriptions: operator.subscriptions,
          isLoading: operator.subscriptionsLoading,
          errorMessage: operator.subscriptionsError?.message ?? null,
          isCreatePending: operator.isCreateSubscriptionPending,
          isDeletePending: operator.isDeleteSubscriptionPending,
          onCreate: operator.handleCreateSubscription,
          onDelete: operator.handleDeleteSubscription,
        }}
        detail={detail}
        inspect={page.inspect}
        inspectErrorMessage={page.inspectError?.message ?? null}
        inspectLoading={page.inspectLoading}
        onOpenChange={controller.setInspectOpen}
        open={controller.inspectOpen}
        stream={{
          state: operator.streamState,
          errorMessage: operator.streamErrorMessage,
          seedSequence: operator.streamSeedSequence,
          latestEventSeq: controller.latestEventSeq,
        }}
      />
    </>
  );
}
