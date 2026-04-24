import { useCallback, useState } from "react";
import { useAui, useAuiState } from "@assistant-ui/react";
import { toast } from "sonner";

import {
  cancelSessionPrompt,
  useClearSessionConversation,
  useDeleteSession,
  useResumeSession,
  useStopSession,
  type SessionPayload,
} from "@/systems/session";

interface UseSessionPageControlsOptions {
  onDeleteSuccess?: () => void;
}

export function useSessionPageControls(
  sessionId: string,
  sessionState: SessionPayload["state"],
  options: UseSessionPageControlsOptions = {}
) {
  const aui = useAui();
  const onDeleteSuccess = options.onDeleteSuccess;
  const messages = useAuiState(state => state.thread.messages);
  const isRunning = useAuiState(state => state.thread.isRunning);
  const deleteMutation = useDeleteSession();
  const stopMutation = useStopSession();
  const resumeMutation = useResumeSession();
  const clearMutation = useClearSessionConversation();
  const [isCancellingPrompt, setIsCancellingPrompt] = useState(false);

  const canPrompt = sessionState === "active";

  const handleCancelPrompt = useCallback(() => {
    if (!isRunning || isCancellingPrompt) {
      return;
    }

    setIsCancellingPrompt(true);
    void cancelSessionPrompt(sessionId)
      .catch(() => {
        toast.error("Failed to stop the current prompt.");
      })
      .finally(() => {
        setIsCancellingPrompt(false);
      });
  }, [isCancellingPrompt, isRunning, sessionId]);

  const isStopping = stopMutation.isPending || isCancellingPrompt;
  const isResuming = resumeMutation.isPending;
  const isDeleting = deleteMutation.isPending;
  const isClearing = clearMutation.isPending;
  const controlsBusy = isStopping || isResuming || isDeleting || isClearing;
  const canClear = messages.length > 0 && !controlsBusy && !isRunning;

  const handleStop = useCallback(() => {
    if (controlsBusy) {
      return;
    }

    if (isRunning) {
      handleCancelPrompt();
      return;
    }

    stopMutation.mutate(sessionId);
  }, [controlsBusy, handleCancelPrompt, isRunning, sessionId, stopMutation]);

  const handleResume = useCallback(() => {
    if (controlsBusy) {
      return;
    }

    resumeMutation.mutate(sessionId);
  }, [controlsBusy, resumeMutation, sessionId]);

  const handleDelete = useCallback(() => {
    if (controlsBusy) {
      return;
    }

    deleteMutation.mutate(sessionId, {
      onSuccess: () => {
        aui.thread().reset();
        toast.success("Session deleted.");
        onDeleteSuccess?.();
      },
      onError: error => {
        toast.error(error instanceof Error ? error.message : "Failed to delete session");
      },
    });
  }, [aui, controlsBusy, deleteMutation, onDeleteSuccess, sessionId]);

  const handleClear = useCallback(() => {
    if (controlsBusy || isRunning) {
      return;
    }

    clearMutation.mutate(sessionId, {
      onSuccess: () => {
        aui.thread().reset();
      },
    });
  }, [aui, clearMutation, controlsBusy, isRunning, sessionId]);

  return {
    canClear,
    canPrompt,
    handleCancelPrompt,
    handleClear,
    handleDelete,
    handleResume,
    handleStop,
    isClearing,
    isDeleting,
    isResuming,
    isStopping,
    messages,
  };
}
