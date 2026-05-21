import { useCallback, useMemo, useState } from "react";
import { useAui, useAuiState } from "@assistant-ui/react";
import { toast } from "sonner";

import {
  cancelSessionPrompt,
  useClearSessionConversation,
  useDeleteSession,
  useInterruptSessionPrompt,
  useQueueSessionPrompt,
  useResumeSession,
  useSteerSessionPrompt,
  useStopSession,
  type SessionPayload,
} from "@/systems/session";

interface UseSessionPageControlsOptions {
  onDeleteSuccess?: () => void;
  workspaceId?: string;
}

export interface ResumeProviderUnavailableDetail {
  sessionId: string;
  missingProvider: string;
  agentName?: string;
}

export interface SessionResumeFailure {
  message: string;
  providerUnavailable: ResumeProviderUnavailableDetail | null;
}

const PROVIDER_VALIDATION_PATTERN =
  /validate agent "([^"]+)" with provider "([^"]+)" for session "([^"]+)"/;

function parseProviderUnavailable(
  sessionId: string,
  message: string
): ResumeProviderUnavailableDetail | null {
  const match = message.match(PROVIDER_VALIDATION_PATTERN);
  if (!match) {
    return null;
  }

  const [, agentName, missingProvider, parsedSessionId] = match;
  const providerName = missingProvider?.trim() ?? "";
  if (providerName.length === 0) {
    return null;
  }

  return {
    sessionId: parsedSessionId?.trim().length ? parsedSessionId : sessionId,
    missingProvider: providerName,
    agentName: agentName?.trim().length ? agentName : undefined,
  };
}

function describeResumeError(error: unknown): string {
  if (error instanceof Error && error.message.trim().length > 0) {
    return error.message;
  }
  return "Failed to attach session.";
}

function describePromptActionError(error: unknown, fallback: string): string {
  if (error instanceof Error && error.message.trim().length > 0) {
    return error.message;
  }
  return fallback;
}

export function useSessionPageControls(
  sessionId: string,
  sessionState: SessionPayload["state"],
  options: UseSessionPageControlsOptions = {}
) {
  const aui = useAui();
  const workspaceId = options.workspaceId ?? "";
  const onDeleteSuccess = options.onDeleteSuccess;
  const messages = useAuiState(state => state.thread.messages);
  const isRunning = useAuiState(state => state.thread.isRunning);
  const deleteMutation = useDeleteSession({ workspaceId });
  const stopMutation = useStopSession({ workspaceId });
  const resumeMutation = useResumeSession({ workspaceId });
  const clearMutation = useClearSessionConversation({ workspaceId });
  const queuePromptMutation = useQueueSessionPrompt({ workspaceId });
  const interruptPromptMutation = useInterruptSessionPrompt({ workspaceId });
  const steerPromptMutation = useSteerSessionPrompt({ workspaceId });
  const [isCancellingPrompt, setIsCancellingPrompt] = useState(false);
  const [resumeFailure, setResumeFailure] = useState<SessionResumeFailure | null>(null);

  const canPrompt = sessionState === "active";

  const handleCancelPrompt = useCallback(() => {
    if (!isRunning || isCancellingPrompt) {
      return;
    }

    setIsCancellingPrompt(true);
    void cancelSessionPrompt(workspaceId, sessionId)
      .catch(() => {
        toast.error("Failed to stop the current prompt.");
      })
      .finally(() => {
        setIsCancellingPrompt(false);
      });
  }, [isCancellingPrompt, isRunning, sessionId, workspaceId]);

  const isStopping = stopMutation.isPending || isCancellingPrompt;
  const isResuming = resumeMutation.isPending;
  const isDeleting = deleteMutation.isPending;
  const isClearing = clearMutation.isPending;
  const isBusyInputPending =
    queuePromptMutation.isPending ||
    interruptPromptMutation.isPending ||
    steerPromptMutation.isPending;
  const controlsBusy = isStopping || isResuming || isDeleting || isClearing || isBusyInputPending;
  const canClear = messages.length > 0 && !controlsBusy && !isRunning;

  const handleQueuePrompt = useCallback(
    async (message: string) => {
      const text = message.trim();
      if (!isRunning || isBusyInputPending || text.length === 0) {
        return;
      }

      try {
        await queuePromptMutation.mutateAsync({ id: sessionId, message: text });
        toast.success("Prompt queued.");
      } catch (error) {
        toast.error(describePromptActionError(error, "Couldn't queue prompt."));
        throw error;
      }
    },
    [isBusyInputPending, isRunning, queuePromptMutation, sessionId]
  );

  const handleInterruptPrompt = useCallback(
    async (message: string) => {
      const text = message.trim();
      if (!isRunning || isBusyInputPending || text.length === 0) {
        return;
      }

      try {
        await interruptPromptMutation.mutateAsync({ id: sessionId, message: text });
        toast.success("Prompt interrupted.");
      } catch (error) {
        toast.error(describePromptActionError(error, "Couldn't interrupt prompt."));
        throw error;
      }
    },
    [interruptPromptMutation, isBusyInputPending, isRunning, sessionId]
  );

  const handleSteerPrompt = useCallback(
    async (message: string) => {
      const text = message.trim();
      if (!isRunning || isBusyInputPending || text.length === 0) {
        return;
      }

      try {
        await steerPromptMutation.mutateAsync({ id: sessionId, message: text });
        toast.success("Steer staged.");
      } catch (error) {
        toast.error(describePromptActionError(error, "Couldn't stage steer."));
        throw error;
      }
    },
    [isBusyInputPending, isRunning, sessionId, steerPromptMutation]
  );

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

    setResumeFailure(null);
    resumeMutation.mutate(sessionId, {
      onError: error => {
        const message = describeResumeError(error);
        const providerUnavailable = parseProviderUnavailable(sessionId, message);
        setResumeFailure({ message, providerUnavailable });
        if (providerUnavailable === null) {
          toast.error(message);
        }
      },
      onSuccess: () => {
        setResumeFailure(null);
      },
    });
  }, [controlsBusy, resumeMutation, sessionId]);

  const handleDismissResumeFailure = useCallback(() => {
    setResumeFailure(null);
  }, []);

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

  return useMemo(
    () => ({
      canClear,
      canPrompt,
      handleCancelPrompt,
      handleClear,
      handleDismissResumeFailure,
      handleDelete,
      handleInterruptPrompt,
      handleQueuePrompt,
      handleResume,
      handleSteerPrompt,
      handleStop,
      isBusyInputPending,
      isClearing,
      isDeleting,
      isResuming,
      isStopping,
      messages,
      resumeFailure,
    }),
    [
      canClear,
      canPrompt,
      handleCancelPrompt,
      handleClear,
      handleDismissResumeFailure,
      handleDelete,
      handleInterruptPrompt,
      handleQueuePrompt,
      handleResume,
      handleSteerPrompt,
      handleStop,
      isBusyInputPending,
      isClearing,
      isDeleting,
      isResuming,
      isStopping,
      messages,
      resumeFailure,
    ]
  );
}
