import { useCallback, useEffect, useMemo, useState } from "react";
import { toast } from "sonner";

import {
  useCreateTaskBridgeNotificationSubscription,
  useDeleteTaskBridgeNotificationSubscription,
  useDeleteTaskExecutionProfile,
  useSetTaskExecutionProfile,
  useTaskBridgeNotificationSubscriptions,
  useTaskExecutionProfile,
  useTaskReviews,
  useTaskStream,
} from "@/systems/tasks";
import type {
  TaskBridgeNotificationSubscriptionCreateRequest,
  TaskExecutionProfileSetRequest,
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
  const createSubscriptionMutation = useCreateTaskBridgeNotificationSubscription();
  const deleteSubscriptionMutation = useDeleteTaskBridgeNotificationSubscription();

  const [streamState, setStreamState] = useState<StreamConnectionState>(
    streamEnabled ? "idle" : "disabled"
  );
  const [streamErrorMessage, setStreamErrorMessage] = useState<string | null>(null);

  // useTaskStream below resubscribes whenever streamEnabled flips, so reset the
  // UI status and drop stale error text until the new EventSource emits.
  useEffect(() => {
    setStreamState(streamEnabled ? "idle" : "disabled");
    setStreamErrorMessage(null);
  }, [streamEnabled]);

  const handleStreamEvent = useCallback(() => {
    setStreamState("connected");
    setStreamErrorMessage(null);
  }, []);

  const handleStreamError = useCallback((error: unknown) => {
    setStreamState("error");
    setStreamErrorMessage(
      error instanceof Error
        ? error.message
        : typeof error === "string"
          ? error
          : "Stream connection failed"
    );
  }, []);

  useTaskStream(taskId, {
    enabled: streamEnabled,
    afterSequence: seedSequence,
    onEvent: handleStreamEvent,
    onError: handleStreamError,
  });

  const handleSetProfile = useCallback(
    async (data: TaskExecutionProfileSetRequest) => {
      if (!hasTaskId) {
        return;
      }
      try {
        await setProfileMutation.mutateAsync({ id: taskId, data });
        toast.success("Execution profile updated.");
      } catch (error) {
        toast.error(error instanceof Error ? error.message : "Failed to update execution profile");
        throw error;
      }
    },
    [hasTaskId, setProfileMutation, taskId]
  );

  const handleDeleteProfile = useCallback(async () => {
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
  }, [deleteProfileMutation, hasTaskId, taskId]);

  const handleCreateSubscription = useCallback(
    async (data: TaskBridgeNotificationSubscriptionCreateRequest) => {
      if (!hasTaskId) {
        return;
      }
      try {
        await createSubscriptionMutation.mutateAsync({ taskId, data });
        toast.success("Bridge notification subscription created.");
      } catch (error) {
        toast.error(
          error instanceof Error ? error.message : "Failed to create bridge subscription"
        );
        throw error;
      }
    },
    [createSubscriptionMutation, hasTaskId, taskId]
  );

  const handleDeleteSubscription = useCallback(
    async (subscriptionId: string) => {
      if (!hasTaskId || subscriptionId.trim() === "") {
        return;
      }
      try {
        await deleteSubscriptionMutation.mutateAsync({ taskId, subscriptionId });
        toast.success("Bridge notification subscription deleted.");
      } catch (error) {
        toast.error(
          error instanceof Error ? error.message : "Failed to delete bridge subscription"
        );
        throw error;
      }
    },
    [deleteSubscriptionMutation, hasTaskId, taskId]
  );

  const profile = profileQuery.data ?? null;
  const reviews = useMemo(() => reviewsQuery.data ?? [], [reviewsQuery.data]);
  const subscriptions = useMemo(() => subscriptionsQuery.data ?? [], [subscriptionsQuery.data]);

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
    isCreateSubscriptionPending: createSubscriptionMutation.isPending,
    isDeleteSubscriptionPending: deleteSubscriptionMutation.isPending,
    handleSetProfile,
    handleDeleteProfile,
    handleCreateSubscription,
    handleDeleteSubscription,
    streamState,
    streamErrorMessage,
    streamSeedSequence: seedSequence,
    hasLatestEventSeq,
  };
}

export { useTaskDetailOrchestrationTab };
export type { StreamConnectionState, UseTaskDetailOrchestrationTabOptions };
