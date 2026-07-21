import { useState } from "react";
import { toast } from "sonner";

import {
  useCreateTaskBridgeNotificationSubscription,
  useDeleteTaskBridgeNotificationSubscription,
  useTaskBridgeNotificationSubscriptions,
} from "./use-task-notifications";
import { useDeleteTaskExecutionProfile, useSetTaskExecutionProfile } from "./use-task-profile";
import { useTaskStream } from "./use-task-stream";
import type {
  TaskBridgeNotificationSubscriptionCreateRequest,
  TaskExecutionProfileSetRequest,
} from "../types";

interface UseTaskOperatorLayerOptions {
  /** Gate the bridge-subscription query + status stream to the open drawer. */
  enabled?: boolean;
  latestEventSeq?: number | null;
}

type StreamConnectionState = "idle" | "connected" | "error" | "disabled";

/**
 * Operator-layer data for the Inspect drawer and Setup sheet (§4.7–4.8):
 * bridge subscriptions, execution-profile writes, and a status-bearing SSE
 * probe whose connection state feeds the drawer's Stream pane.
 */
function useTaskOperatorLayer(taskId: string, options: UseTaskOperatorLayerOptions = {}) {
  const enabled = options.enabled ?? true;
  const hasTaskId = taskId.trim() !== "";
  const hasLatestEventSeq =
    typeof options.latestEventSeq === "number" && Number.isFinite(options.latestEventSeq);
  const seedSequence = hasLatestEventSeq ? Math.max(0, options.latestEventSeq ?? 0) : 0;
  const layerEnabled = enabled && hasTaskId;

  const subscriptionsQuery = useTaskBridgeNotificationSubscriptions(
    taskId,
    {},
    { enabled: layerEnabled }
  );

  const setProfileMutation = useSetTaskExecutionProfile();
  const deleteProfileMutation = useDeleteTaskExecutionProfile();
  const createSubscriptionMutation = useCreateTaskBridgeNotificationSubscription();
  const deleteSubscriptionMutation = useDeleteTaskBridgeNotificationSubscription();

  const streamKey = layerEnabled ? `${taskId}:${seedSequence}` : "disabled";
  const [streamStatus, setStreamStatus] = useState<{
    error: string | null;
    key: string;
    state: StreamConnectionState;
  }>(() => ({ error: null, key: streamKey, state: layerEnabled ? "idle" : "disabled" }));
  const currentStreamStatus =
    streamStatus.key === streamKey
      ? streamStatus
      : {
          error: null,
          key: streamKey,
          state: layerEnabled ? ("idle" as const) : ("disabled" as const),
        };

  useTaskStream(taskId, {
    enabled: layerEnabled && hasLatestEventSeq,
    afterSequence: seedSequence,
    onEvent: () => setStreamStatus({ error: null, key: streamKey, state: "connected" }),
    onError: error =>
      setStreamStatus({
        error:
          error instanceof Error
            ? error.message
            : typeof error === "string"
              ? error
              : "Stream connection failed",
        key: streamKey,
        state: "error",
      }),
  });

  const handleSetProfile = async (data: TaskExecutionProfileSetRequest) => {
    try {
      await setProfileMutation.mutateAsync({ id: taskId, data });
      toast.success("Setup saved.");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to save setup");
      throw error;
    }
  };

  const handleDeleteProfile = async () => {
    await deleteProfileMutation.mutateAsync({ id: taskId });
  };

  const handleCreateSubscription = async (
    request: TaskBridgeNotificationSubscriptionCreateRequest
  ) => {
    try {
      await createSubscriptionMutation.mutateAsync({ taskId, data: request });
      toast.success("Subscription added.");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to add subscription");
      throw error;
    }
  };

  const handleDeleteSubscription = async (subscriptionId: string) => {
    try {
      await deleteSubscriptionMutation.mutateAsync({ taskId, subscriptionId });
      toast.success("Subscription removed.");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to remove subscription");
    }
  };

  return {
    handleCreateSubscription,
    handleDeleteProfile,
    handleDeleteSubscription,
    handleSetProfile,
    isCreateSubscriptionPending: createSubscriptionMutation.isPending,
    isDeleteProfilePending: deleteProfileMutation.isPending,
    isDeleteSubscriptionPending: deleteSubscriptionMutation.isPending,
    isSetProfilePending: setProfileMutation.isPending,
    streamErrorMessage: currentStreamStatus.error,
    streamSeedSequence: seedSequence,
    streamState: currentStreamStatus.state,
    subscriptions: subscriptionsQuery.data ?? [],
    subscriptionsError: subscriptionsQuery.error ?? null,
    subscriptionsLoading: subscriptionsQuery.isLoading && !subscriptionsQuery.data,
  };
}

export { useTaskOperatorLayer };
export type { UseTaskOperatorLayerOptions };
