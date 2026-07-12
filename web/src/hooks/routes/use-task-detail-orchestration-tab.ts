import { useState } from "react";
import { toast } from "sonner";

import {
  useCreateTaskBridgeNotificationSubscription,
  useDeleteTaskBridgeNotificationSubscription,
  useDeleteTaskExecutionProfile,
  useFanOutTaskRuns,
  useSetTaskExecutionProfile,
  useTaskBridgeNotificationSubscriptions,
  useTaskExecutionProfile,
  useTaskReviews,
  useTaskStream,
} from "@/systems/tasks";
import type {
  TaskBridgeNotificationSubscriptionCreateRequest,
  TaskExecutionProfileSetRequest,
  FanOutTaskRunsRequest,
} from "@/systems/tasks";

interface UseTaskDetailOrchestrationTabOptions {
  enabled?: boolean;
  latestEventSeq?: number | null;
}

type StreamConnectionState = "idle" | "connected" | "error" | "disabled";

function useTaskDetailOrchestrationTab(
  taskId: string,
  options: UseTaskDetailOrchestrationTabOptions = {}
) {
  const enabled = options.enabled ?? true;
  const hasTaskId = taskId.trim() !== "";
  const hasLatestEventSeq =
    typeof options.latestEventSeq === "number" && Number.isFinite(options.latestEventSeq);
  const seedSequence = hasLatestEventSeq ? Math.max(0, options.latestEventSeq ?? 0) : 0;
  const streamEnabled = enabled && hasTaskId;

  const profileQuery = useTaskExecutionProfile(taskId, { enabled: streamEnabled });
  const reviewsQuery = useTaskReviews(taskId, {}, { enabled: streamEnabled });
  const subscriptionsQuery = useTaskBridgeNotificationSubscriptions(
    taskId,
    {},
    { enabled: streamEnabled }
  );

  const setProfileMutation = useSetTaskExecutionProfile();
  const deleteProfileMutation = useDeleteTaskExecutionProfile();
  const fanOutMutation = useFanOutTaskRuns();
  const createSubscriptionMutation = useCreateTaskBridgeNotificationSubscription();
  const deleteSubscriptionMutation = useDeleteTaskBridgeNotificationSubscription();

  const streamKey = streamEnabled ? `${taskId}:${seedSequence}` : "disabled";
  const [streamStatus, setStreamStatus] = useState<{
    error: string | null;
    key: string;
    state: StreamConnectionState;
  }>(() => ({ error: null, key: streamKey, state: streamEnabled ? "idle" : "disabled" }));
  const currentStreamStatus =
    streamStatus.key === streamKey
      ? streamStatus
      : {
          error: null,
          key: streamKey,
          state: streamEnabled ? ("idle" as const) : ("disabled" as const),
        };

  const handleStreamEvent = () => {
    setStreamStatus({ error: null, key: streamKey, state: "connected" });
  };

  const handleStreamError = (error: unknown) => {
    setStreamStatus({
      error:
        error instanceof Error
          ? error.message
          : typeof error === "string"
            ? error
            : "Stream connection failed",
      key: streamKey,
      state: "error",
    });
  };

  useTaskStream(taskId, {
    enabled: streamEnabled,
    afterSequence: seedSequence,
    onEvent: handleStreamEvent,
    onError: handleStreamError,
  });

  const handleSetProfile = async (data: TaskExecutionProfileSetRequest) => {
    if (!hasTaskId) return;
    try {
      await setProfileMutation.mutateAsync({ id: taskId, data });
      toast.success("Execution profile updated.");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to update execution profile");
      throw error;
    }
  };

  const handleDeleteProfile = async () => {
    if (!hasTaskId) {
      return;
    }
    try {
      await deleteProfileMutation.mutateAsync({ id: taskId });
      toast.success("Execution profile deleted.");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to delete execution profile");
      throw error;
    }
  };

  const handleFanOutRuns = async (data: FanOutTaskRunsRequest) => {
    if (!hasTaskId) return;
    try {
      const result = await fanOutMutation.mutateAsync({ id: taskId, data });
      const runCount = result.runs.length;
      toast.success(`${runCount} designated ${runCount === 1 ? "run" : "runs"} queued.`);
      return result;
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to fan out task runs");
      throw error;
    }
  };

  const handleCreateSubscription = async (
    data: TaskBridgeNotificationSubscriptionCreateRequest
  ) => {
    if (!hasTaskId) return;
    try {
      await createSubscriptionMutation.mutateAsync({ taskId, data });
      toast.success("Bridge notification subscription created.");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to create bridge subscription");
      throw error;
    }
  };

  const handleDeleteSubscription = async (subscriptionId: string) => {
    if (!hasTaskId || subscriptionId.trim() === "") return;
    try {
      await deleteSubscriptionMutation.mutateAsync({ taskId, subscriptionId });
      toast.success("Bridge notification subscription deleted.");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to delete bridge subscription");
      throw error;
    }
  };

  const profile = profileQuery.data ?? null;
  const reviews = reviewsQuery.data ?? [];
  const subscriptions = subscriptionsQuery.data ?? [];

  return {
    profile,
    profileError: profileQuery.error ?? null,
    profileLoading: profileQuery.isLoading && !profile,
    reviews,
    reviewsError: reviewsQuery.error ?? null,
    reviewsLoading: reviewsQuery.isLoading && reviews.length === 0,
    subscriptions,
    subscriptionsError: subscriptionsQuery.error ?? null,
    subscriptionsLoading: subscriptionsQuery.isLoading && subscriptions.length === 0,
    isSetProfilePending: setProfileMutation.isPending,
    isDeleteProfilePending: deleteProfileMutation.isPending,
    isFanOutPending: fanOutMutation.isPending,
    isCreateSubscriptionPending: createSubscriptionMutation.isPending,
    isDeleteSubscriptionPending: deleteSubscriptionMutation.isPending,
    handleSetProfile,
    handleDeleteProfile,
    handleFanOutRuns,
    handleCreateSubscription,
    handleDeleteSubscription,
    streamState: currentStreamStatus.state,
    streamErrorMessage: currentStreamStatus.error,
    streamSeedSequence: seedSequence,
    hasLatestEventSeq,
  };
}

export { useTaskDetailOrchestrationTab };
export type { StreamConnectionState, UseTaskDetailOrchestrationTabOptions };
