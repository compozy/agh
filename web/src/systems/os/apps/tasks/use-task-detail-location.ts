import { useNavigate } from "@tanstack/react-router";
import { useState } from "react";
import { toast } from "sonner";

import {
  resolveTaskCommandState,
  resolveTaskDetailSearch,
  TASK_RESULT_ANCHOR_ID,
  useDeleteTask,
  useTaskDetailPage,
  useTaskOperatorLayer,
  useTaskPauseDialog,
  type TaskDetailSearch,
  type TaskDetailTab,
} from "@/systems/tasks";

import { useOsShell } from "../../hooks/use-os-shell";

function scrollToResult() {
  document.getElementById(TASK_RESULT_ANCHOR_ID)?.scrollIntoView({ block: "start" });
}

/**
 * Controller for the task-detail window location: data (via
 * `useTaskDetailPage` + `useTaskOperatorLayer`), overlay state, WM-location
 * navigation, and the §6 command state. The location component only renders.
 */
export function useTaskDetailLocation(taskId: string, rawSearch: TaskDetailSearch) {
  const navigate = useNavigate();
  const { coordinator } = useOsShell();
  const page = useTaskDetailPage(taskId);
  const deleteMutation = useDeleteTask();
  const search = resolveTaskDetailSearch(rawSearch);
  const [inspectOpen, setInspectOpen] = useState(false);
  const [setupOpen, setSetupOpen] = useState(false);
  const [fanOutOpen, setFanOutOpen] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const pauseDialog = useTaskPauseDialog(page.handlePauseTask);
  const latestEventSeq = page.detail?.task?.latest_event_seq ?? null;
  const operator = useTaskOperatorLayer(taskId, { enabled: inspectOpen, latestEventSeq });

  const detail = page.detail;
  const record = detail?.task ?? null;
  const command = detail ? resolveTaskCommandState(detail, page.runs) : null;
  // Fan-out designations are supplied per invocation, so the entry stays
  // available for any non-terminal, published task.
  const showFanOut = Boolean(command?.overflow.edit && record && !record.draft);

  const setTab = (tab: TaskDetailTab) => {
    void navigate({ to: "/tasks/$id", params: { id: taskId }, search: { tab }, replace: true });
  };

  const openRun = (runId: string) => {
    void navigate({ to: "/tasks/$id/runs/$runId", params: { id: taskId, runId } });
  };

  const openTask = (id: string) => {
    void navigate({ to: "/tasks/$id", params: { id } });
  };

  const openEdit = () => {
    void navigate({ to: "/tasks/$id/edit", params: { id: taskId } });
  };

  const backToTasks = () => {
    void navigate({ to: "/tasks" });
  };

  const handleDeleteTask = async (id: string) => {
    coordinator.userOpen({ app: "tasks", location: { pathname: "/tasks", search: {} } });
    try {
      await deleteMutation.mutateAsync({ id });
      toast.success("Task deleted.");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to delete task");
    }
  };

  const copyTaskId = () => {
    if (!record || !navigator.clipboard) return;
    navigator.clipboard
      .writeText(record.id)
      .then(() => toast.success("Task id copied."))
      .catch(() => toast.error("Couldn't copy the task id"));
  };

  return {
    backToTasks,
    command,
    copyTaskId,
    deleteMutation,
    deleteOpen,
    detail,
    fanOutOpen,
    handleDeleteTask,
    inspectOpen,
    latestEventSeq,
    openEdit,
    openRun,
    openTask,
    operator,
    page,
    pauseDialog,
    record,
    scrollToResult,
    search,
    setDeleteOpen,
    setFanOutOpen,
    setInspectOpen,
    setSetupOpen,
    setTab,
    setupOpen,
    showFanOut,
    taskId,
  };
}

export type TaskDetailLocationController = ReturnType<typeof useTaskDetailLocation>;
