import { useCallback, useState } from "react";
import { useAui, useAuiState } from "@assistant-ui/react";
import { toast } from "sonner";

import { cancelSessionPrompt } from "@/systems/session/adapters/session-api";
import {
  useClearSessionConversation,
  useResumeSession,
  useStopSession,
} from "@/systems/session/hooks/use-session-actions";
import type { SessionPayload } from "@/systems/session/types";

export function useSessionPageControls(sessionId: string, sessionState: SessionPayload["state"]) {
  const aui = useAui();
  const messages = useAuiState(state => state.thread.messages);
  const isRunning = useAuiState(state => state.thread.isRunning);
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

  const handleStop = useCallback(() => {
    if (isRunning) {
      handleCancelPrompt();
      return;
    }

    stopMutation.mutate(sessionId);
  }, [handleCancelPrompt, isRunning, sessionId, stopMutation]);

  const handleResume = useCallback(() => {
    resumeMutation.mutate(sessionId);
  }, [resumeMutation, sessionId]);

  const handleClear = useCallback(() => {
    if (clearMutation.isPending) {
      return;
    }

    clearMutation.mutate(sessionId, {
      onSuccess: () => {
        aui.thread().reset();
      },
    });
  }, [aui, clearMutation, sessionId]);

  const isStopping = stopMutation.isPending || isCancellingPrompt;
  const isResuming = resumeMutation.isPending;
  const isClearing = clearMutation.isPending;
  const controlsBusy = isStopping || isResuming || isClearing;
  const canClear = messages.length > 0 && !controlsBusy && !isRunning;

  return {
    canClear,
    canPrompt,
    handleCancelPrompt,
    handleClear,
    handleResume,
    handleStop,
    isClearing,
    isResuming,
    isStopping,
    messages,
  };
}
