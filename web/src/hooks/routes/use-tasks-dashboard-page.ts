import { useCallback, useMemo } from "react";
import { toast } from "sonner";

import {
  useDrainScheduler,
  usePauseScheduler,
  useResumeScheduler,
  useSchedulerBacklog,
  useSchedulerStatus,
} from "@/systems/scheduler";
import { useTaskDashboard } from "@/systems/tasks";
import type { TaskDashboardFilter } from "@/systems/tasks";

export function useTasksDashboardPage(filters: TaskDashboardFilter, enabled: boolean) {
  const dashboardQuery = useTaskDashboard(filters, { enabled });
  const schedulerStatusQuery = useSchedulerStatus({ enabled });
  const schedulerBacklogFilters = useMemo(
    () => ({
      scope: filters.scope,
      limit: 5,
      workspace: filters.workspace,
      include_paused: true,
    }),
    [filters.scope, filters.workspace]
  );
  const schedulerBacklogQuery = useSchedulerBacklog(schedulerBacklogFilters, { enabled });
  const pauseMutation = usePauseScheduler();
  const resumeMutation = useResumeScheduler();
  const drainMutation = useDrainScheduler();

  const handlePauseScheduler = useCallback(
    async (reason: string) => {
      try {
        const normalizedReason = reason.trim();
        await pauseMutation.mutateAsync(normalizedReason ? { reason: normalizedReason } : {});
        toast.success("Scheduler paused.");
      } catch (error) {
        toast.error(error instanceof Error ? error.message : "Failed to pause scheduler");
        throw error;
      }
    },
    [pauseMutation]
  );
  const handleResumeScheduler = useCallback(
    async (reason?: string) => {
      try {
        const normalizedReason = reason?.trim();
        await resumeMutation.mutateAsync(normalizedReason ? { reason: normalizedReason } : {});
        toast.success("Scheduler resumed.");
      } catch (error) {
        toast.error(error instanceof Error ? error.message : "Failed to resume scheduler");
        throw error;
      }
    },
    [resumeMutation]
  );
  const handleDrainScheduler = useCallback(
    async ({ reason, timeoutSeconds }: { reason?: string; timeoutSeconds?: number }) => {
      try {
        const normalizedReason = reason?.trim();
        await drainMutation.mutateAsync({
          ...(normalizedReason ? { reason: normalizedReason } : {}),
          timeout_seconds: timeoutSeconds ?? 60,
        });
        toast.success("Scheduler drain requested.");
      } catch (error) {
        toast.error(error instanceof Error ? error.message : "Failed to drain scheduler");
        throw error;
      }
    },
    [drainMutation]
  );

  return {
    dashboard: dashboardQuery.data ?? null,
    dashboardError: dashboardQuery.error ?? null,
    dashboardLoading: dashboardQuery.isLoading && !dashboardQuery.data,
    handleDrainScheduler,
    handlePauseScheduler,
    handleResumeScheduler,
    isSchedulerDrainPending: drainMutation.isPending,
    isSchedulerPausePending: pauseMutation.isPending,
    isSchedulerResumePending: resumeMutation.isPending,
    schedulerBacklog: schedulerBacklogQuery.data ?? null,
    schedulerBacklogError: schedulerBacklogQuery.error ?? null,
    schedulerBacklogLoading: schedulerBacklogQuery.isLoading && !schedulerBacklogQuery.data,
    schedulerStatus: schedulerStatusQuery.data ?? null,
    schedulerStatusError: schedulerStatusQuery.error ?? null,
    schedulerStatusLoading: schedulerStatusQuery.isLoading && !schedulerStatusQuery.data,
  };
}
