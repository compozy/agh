import { useMutation, useQueryClient } from "@tanstack/react-query";

import {
  addTaskDependency,
  approveTask,
  archiveTask,
  attachTaskRunSession,
  cancelTask,
  cancelTaskRun,
  claimTaskRun,
  completeTaskRun,
  createChildTask,
  createTask,
  dismissTask,
  enqueueTaskRun,
  failTaskRun,
  markTaskRead,
  publishTask,
  rejectTask,
  removeTaskDependency,
  startTaskRun,
  updateTask,
} from "../adapters/tasks-api";
import { tasksKeys } from "../lib/query-keys";
import type {
  AddTaskDependencyRequest,
  AttachTaskRunSessionRequest,
  CancelTaskRequest,
  CancelTaskRunRequest,
  ClaimTaskRunRequest,
  CompleteTaskRunRequest,
  CreateChildTaskRequest,
  CreateTaskRequest,
  EnqueueTaskRunRequest,
  FailTaskRunRequest,
  StartTaskRunRequest,
  UpdateTaskRequest,
} from "../types";

type QueryClient = ReturnType<typeof useQueryClient>;

interface TaskIdParams {
  id: string;
}

interface TaskRunIdParams {
  runId: string;
}

interface UpdateTaskParams extends TaskIdParams {
  data: UpdateTaskRequest;
}

interface CancelTaskParams extends TaskIdParams {
  data?: CancelTaskRequest;
}

interface CreateChildTaskParams {
  parentId: string;
  data: CreateChildTaskRequest;
}

interface AddTaskDependencyParams extends TaskIdParams {
  data: AddTaskDependencyRequest;
}

interface RemoveTaskDependencyParams extends TaskIdParams {
  dependsOnId: string;
}

interface EnqueueTaskRunParams extends TaskIdParams {
  data?: EnqueueTaskRunRequest;
}

interface AttachTaskRunSessionParams extends TaskRunIdParams {
  data: AttachTaskRunSessionRequest;
}

interface CancelTaskRunParams extends TaskRunIdParams {
  data?: CancelTaskRunRequest;
}

interface ClaimTaskRunParams extends TaskRunIdParams {
  data?: ClaimTaskRunRequest;
}

interface StartTaskRunParams extends TaskRunIdParams {
  data?: StartTaskRunRequest;
}

interface CompleteTaskRunParams extends TaskRunIdParams {
  data?: CompleteTaskRunRequest;
}

interface FailTaskRunParams extends TaskRunIdParams {
  data: FailTaskRunRequest;
}

function invalidateTaskQueries(queryClient: QueryClient, id?: string) {
  const pending = [
    queryClient.invalidateQueries({ queryKey: tasksKeys.lists() }),
    queryClient.invalidateQueries({ queryKey: tasksKeys.runsRoot() }),
    queryClient.invalidateQueries({ queryKey: tasksKeys.timelineRoot() }),
    queryClient.invalidateQueries({ queryKey: tasksKeys.treeRoot() }),
    queryClient.invalidateQueries({ queryKey: tasksKeys.runDetails() }),
  ];

  if (id) {
    pending.push(queryClient.invalidateQueries({ queryKey: tasksKeys.detail(id) }));
  }

  return Promise.all(pending);
}

function invalidateAggregateQueries(queryClient: QueryClient) {
  return Promise.all([
    queryClient.invalidateQueries({ queryKey: [...tasksKeys.all, "dashboard"] }),
    queryClient.invalidateQueries({ queryKey: [...tasksKeys.all, "inbox"] }),
  ]);
}

function invalidateTriageQueries(queryClient: QueryClient, id?: string) {
  return Promise.all([
    queryClient.invalidateQueries({ queryKey: tasksKeys.triageRoot() }),
    queryClient.invalidateQueries({ queryKey: tasksKeys.lists() }),
    ...(id ? [queryClient.invalidateQueries({ queryKey: tasksKeys.detail(id) })] : []),
    queryClient.invalidateQueries({ queryKey: [...tasksKeys.all, "inbox"] }),
  ]);
}

export function useCreateTask() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateTaskRequest) => createTask(data),
    onSettled: () =>
      Promise.all([invalidateTaskQueries(queryClient), invalidateAggregateQueries(queryClient)]),
  });
}

export function useUpdateTask() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: UpdateTaskParams) => updateTask(id, data),
    onSettled: (_result, _error, { id }) =>
      Promise.all([
        invalidateTaskQueries(queryClient, id),
        invalidateAggregateQueries(queryClient),
      ]),
  });
}

export function usePublishTask() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id }: TaskIdParams) => publishTask(id),
    onSettled: (_result, _error, { id }) =>
      Promise.all([
        invalidateTaskQueries(queryClient, id),
        invalidateAggregateQueries(queryClient),
      ]),
  });
}

export function useCancelTask() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: CancelTaskParams) => cancelTask(id, data ?? {}),
    onSettled: (_result, _error, { id }) =>
      Promise.all([
        invalidateTaskQueries(queryClient, id),
        invalidateAggregateQueries(queryClient),
      ]),
  });
}

export function useApproveTask() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id }: TaskIdParams) => approveTask(id),
    onSettled: (_result, _error, { id }) =>
      Promise.all([
        invalidateTaskQueries(queryClient, id),
        invalidateAggregateQueries(queryClient),
      ]),
  });
}

export function useRejectTask() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id }: TaskIdParams) => rejectTask(id),
    onSettled: (_result, _error, { id }) =>
      Promise.all([
        invalidateTaskQueries(queryClient, id),
        invalidateAggregateQueries(queryClient),
      ]),
  });
}

export function useCreateChildTask() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ parentId, data }: CreateChildTaskParams) => createChildTask(parentId, data),
    onSettled: (_result, _error, { parentId }) =>
      Promise.all([
        invalidateTaskQueries(queryClient, parentId),
        invalidateAggregateQueries(queryClient),
      ]),
  });
}

export function useAddTaskDependency() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: AddTaskDependencyParams) => addTaskDependency(id, data),
    onSettled: (_result, _error, { id }) =>
      Promise.all([
        invalidateTaskQueries(queryClient, id),
        invalidateAggregateQueries(queryClient),
      ]),
  });
}

export function useRemoveTaskDependency() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, dependsOnId }: RemoveTaskDependencyParams) =>
      removeTaskDependency(id, dependsOnId),
    onSettled: (_result, _error, { id }) =>
      Promise.all([
        invalidateTaskQueries(queryClient, id),
        invalidateAggregateQueries(queryClient),
      ]),
  });
}

export function useEnqueueTaskRun() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: EnqueueTaskRunParams) => enqueueTaskRun(id, data ?? {}),
    onSettled: (_result, _error, { id }) =>
      Promise.all([
        invalidateTaskQueries(queryClient, id),
        invalidateAggregateQueries(queryClient),
      ]),
  });
}

export function useAttachTaskRunSession() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ runId, data }: AttachTaskRunSessionParams) => attachTaskRunSession(runId, data),
    onSettled: (_result, _error, { runId }) =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: tasksKeys.runDetail(runId) }),
        invalidateTaskQueries(queryClient),
      ]),
  });
}

export function useCancelTaskRun() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ runId, data }: CancelTaskRunParams) => cancelTaskRun(runId, data ?? {}),
    onSettled: (_result, _error, { runId }) =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: tasksKeys.runDetail(runId) }),
        invalidateTaskQueries(queryClient),
        invalidateAggregateQueries(queryClient),
      ]),
  });
}

export function useClaimTaskRun() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ runId, data }: ClaimTaskRunParams) => claimTaskRun(runId, data ?? {}),
    onSettled: (_result, _error, { runId }) =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: tasksKeys.runDetail(runId) }),
        invalidateTaskQueries(queryClient),
      ]),
  });
}

export function useStartTaskRun() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ runId, data }: StartTaskRunParams) => startTaskRun(runId, data ?? {}),
    onSettled: (_result, _error, { runId }) =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: tasksKeys.runDetail(runId) }),
        invalidateTaskQueries(queryClient),
      ]),
  });
}

export function useCompleteTaskRun() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ runId, data }: CompleteTaskRunParams) => completeTaskRun(runId, data ?? {}),
    onSettled: (_result, _error, { runId }) =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: tasksKeys.runDetail(runId) }),
        invalidateTaskQueries(queryClient),
        invalidateAggregateQueries(queryClient),
      ]),
  });
}

export function useFailTaskRun() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ runId, data }: FailTaskRunParams) => failTaskRun(runId, data),
    onSettled: (_result, _error, { runId }) =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: tasksKeys.runDetail(runId) }),
        invalidateTaskQueries(queryClient),
        invalidateAggregateQueries(queryClient),
      ]),
  });
}

export function useMarkTaskRead() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id }: TaskIdParams) => markTaskRead(id),
    onSettled: (_result, _error, { id }) => invalidateTriageQueries(queryClient, id),
  });
}

export function useArchiveTask() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id }: TaskIdParams) => archiveTask(id),
    onSettled: (_result, _error, { id }) => invalidateTriageQueries(queryClient, id),
  });
}

export function useDismissTask() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id }: TaskIdParams) => dismissTask(id),
    onSettled: (_result, _error, { id }) => invalidateTriageQueries(queryClient, id),
  });
}
