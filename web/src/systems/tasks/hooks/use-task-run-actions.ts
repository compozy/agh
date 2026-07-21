import { useMutation, useQueryClient } from "@tanstack/react-query";

import {
  attachTaskRunSession,
  cancelTaskRun,
  completeTaskRun,
  failTaskRun,
  forceFailTaskRun,
  forceReleaseTaskRun,
  retryTaskRun,
  startTaskRun,
} from "../adapters/tasks-api";
import { tasksKeys } from "../lib/query-keys";
import type {
  AttachTaskRunSessionRequest,
  CancelTaskRunRequest,
  CompleteTaskRunRequest,
  FailTaskRunRequest,
  ForceFailTaskRunRequest,
  ForceReleaseTaskRunRequest,
  RetryTaskRunRequest,
  StartTaskRunRequest,
} from "../types";
import { invalidateAggregateQueries, invalidateTaskQueries } from "./task-query-invalidation";

interface TaskRunIdParams {
  runId: string;
}

interface AttachTaskRunSessionParams extends TaskRunIdParams {
  data: AttachTaskRunSessionRequest;
}

interface CancelTaskRunParams extends TaskRunIdParams {
  data?: CancelTaskRunRequest;
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

interface ForceReleaseTaskRunParams extends TaskRunIdParams {
  data?: ForceReleaseTaskRunRequest;
}

interface ForceFailTaskRunParams extends TaskRunIdParams {
  data: ForceFailTaskRunRequest;
}

interface RetryTaskRunParams extends TaskRunIdParams {
  data?: RetryTaskRunRequest;
}

function invalidateAfterRunSettlement<T>(error: unknown, invalidate: () => T): T {
  // mutateAsync callers report failures; settled handlers converge cached run reads.
  void error;
  return invalidate();
}

export function useAttachTaskRunSession() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ runId, data }: AttachTaskRunSessionParams) => attachTaskRunSession(runId, data),
    onSettled: (_result, error, { runId }) =>
      invalidateAfterRunSettlement(error, () =>
        Promise.all([
          queryClient.invalidateQueries({ queryKey: tasksKeys.runDetail(runId) }),
          invalidateTaskQueries(queryClient),
        ])
      ),
  });
}

export function useCancelTaskRun() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ runId, data }: CancelTaskRunParams) => cancelTaskRun(runId, data ?? {}),
    onSettled: (_result, error, { runId }) =>
      invalidateAfterRunSettlement(error, () =>
        Promise.all([
          queryClient.invalidateQueries({ queryKey: tasksKeys.runDetail(runId) }),
          invalidateTaskQueries(queryClient),
          invalidateAggregateQueries(queryClient),
        ])
      ),
  });
}

export function useStartTaskRun() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ runId, data }: StartTaskRunParams) => startTaskRun(runId, data ?? {}),
    onSettled: (_result, error, { runId }) =>
      invalidateAfterRunSettlement(error, () =>
        Promise.all([
          queryClient.invalidateQueries({ queryKey: tasksKeys.runDetail(runId) }),
          invalidateTaskQueries(queryClient),
        ])
      ),
  });
}

export function useCompleteTaskRun() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ runId, data }: CompleteTaskRunParams) => completeTaskRun(runId, data ?? {}),
    onSettled: (_result, error, { runId }) =>
      invalidateAfterRunSettlement(error, () =>
        Promise.all([
          queryClient.invalidateQueries({ queryKey: tasksKeys.runDetail(runId) }),
          invalidateTaskQueries(queryClient),
          invalidateAggregateQueries(queryClient),
        ])
      ),
  });
}

export function useFailTaskRun() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ runId, data }: FailTaskRunParams) => failTaskRun(runId, data),
    onSettled: (_result, error, { runId }) =>
      invalidateAfterRunSettlement(error, () =>
        Promise.all([
          queryClient.invalidateQueries({ queryKey: tasksKeys.runDetail(runId) }),
          invalidateTaskQueries(queryClient),
          invalidateAggregateQueries(queryClient),
        ])
      ),
  });
}

export function useForceReleaseTaskRun() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ runId, data }: ForceReleaseTaskRunParams) =>
      forceReleaseTaskRun(runId, data ?? {}),
    onSettled: (_result, error, { runId }) =>
      invalidateAfterRunSettlement(error, () =>
        Promise.all([
          queryClient.invalidateQueries({ queryKey: tasksKeys.runDetail(runId) }),
          invalidateTaskQueries(queryClient),
          invalidateAggregateQueries(queryClient),
        ])
      ),
  });
}

export function useForceFailTaskRun() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ runId, data }: ForceFailTaskRunParams) => forceFailTaskRun(runId, data),
    onSettled: (_result, error, { runId }) =>
      invalidateAfterRunSettlement(error, () =>
        Promise.all([
          queryClient.invalidateQueries({ queryKey: tasksKeys.runDetail(runId) }),
          invalidateTaskQueries(queryClient),
          invalidateAggregateQueries(queryClient),
        ])
      ),
  });
}

export function useRetryTaskRun() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ runId, data }: RetryTaskRunParams) => retryTaskRun(runId, data ?? {}),
    onSettled: (_result, error, { runId }) =>
      invalidateAfterRunSettlement(error, () =>
        Promise.all([
          queryClient.invalidateQueries({ queryKey: tasksKeys.runDetail(runId) }),
          invalidateTaskQueries(queryClient),
          invalidateAggregateQueries(queryClient),
        ])
      ),
  });
}
