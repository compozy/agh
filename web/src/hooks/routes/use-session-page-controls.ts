import { useCallback, useEffect, useMemo, useState } from "react";
import { useAui, useAuiState } from "@assistant-ui/react";
import { toast } from "sonner";

import type { QueuedPrompt } from "@/components/assistant-ui/session-composer-queued-prompts";
import {
  cancelSessionPrompt,
  useCancelQueuedSessionPrompt,
  useClearSessionConversation,
  useDeleteSession,
  useInterruptSessionPrompt,
  useQueueSessionPrompt,
  useResumeSession,
  useSteerSessionPrompt,
  useStopSession,
  useSessionTranscriptThreadMessages,
  isSessionRunning,
  isUserControllableSession,
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
  session: SessionPayload,
  options: UseSessionPageControlsOptions = {}
) {
  const aui = useAui();
  const workspaceId = options.workspaceId ?? "";
  const onDeleteSuccess = options.onDeleteSuccess;
  const messages = useAuiState(state => state.thread.messages);
  const transcriptMessages = useSessionTranscriptThreadMessages();
  const isRunning = useAuiState(state => state.thread.isRunning);
  const deleteMutation = useDeleteSession({ workspaceId });
  const stopMutation = useStopSession({ workspaceId });
  const resumeMutation = useResumeSession({ workspaceId });
  const clearMutation = useClearSessionConversation({ workspaceId });
  const queuePromptMutation = useQueueSessionPrompt({ workspaceId });
  const interruptPromptMutation = useInterruptSessionPrompt({ workspaceId });
  const steerPromptMutation = useSteerSessionPrompt({ workspaceId });
  const cancelQueuedPromptMutation = useCancelQueuedSessionPrompt({ workspaceId });
  const [isCancellingPrompt, setIsCancellingPrompt] = useState(false);
  const [resumeFailure, setResumeFailure] = useState<SessionResumeFailure | null>(null);
  const [queuedPrompts, setQueuedPrompts] = useState<QueuedPrompt[]>([]);

  const daemonRunning = isSessionRunning(session);
  const userControllable = isUserControllableSession(session);
  const effectiveRunning = isRunning || daemonRunning;
  const promptControlsAvailable = effectiveRunning && userControllable;
  const canPrompt = session.state === "active" && userControllable;

  // Queued prompts are a client-tracked optimistic mirror of the daemon's durable
  // busy-input queue: every row carries the real `queue_entry_id` the queue mutation
  // returned, so delete/steer act on real entries. When the turn settles the daemon
  // drains the queue, so drop the local rows — never render a queued prompt that the
  // runtime no longer holds (truthful UI). It may briefly under-show a second queued
  // prompt across a multi-dispatch settle, which is safe; it never over-shows.
  useEffect(() => {
    if (!effectiveRunning) {
      setQueuedPrompts(prev => (prev.length > 0 ? [] : prev));
    }
  }, [effectiveRunning]);

  const handleCancelPrompt = useCallback(() => {
    if (!promptControlsAvailable || isCancellingPrompt) {
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
  }, [isCancellingPrompt, promptControlsAvailable, sessionId, workspaceId]);

  const isStopping = stopMutation.isPending || isCancellingPrompt;
  const isResuming = resumeMutation.isPending;
  const isDeleting = deleteMutation.isPending;
  const isClearing = clearMutation.isPending;
  const isBusyInputPending =
    queuePromptMutation.isPending ||
    interruptPromptMutation.isPending ||
    steerPromptMutation.isPending;
  const controlsBusy = isStopping || isResuming || isDeleting || isClearing || isBusyInputPending;
  const hasConversationContent = messages.length > 0 || transcriptMessages.length > 0;
  const canClear = hasConversationContent && !controlsBusy && !effectiveRunning;

  const handleQueuePrompt = useCallback(
    async (message: string) => {
      const text = message.trim();
      if (!promptControlsAvailable || isBusyInputPending || text.length === 0) {
        return;
      }

      try {
        const result = await queuePromptMutation.mutateAsync({ id: sessionId, message: text });
        const queueEntryId = result.queue_entry_id;
        if (result.queued && queueEntryId) {
          setQueuedPrompts(prev => [...prev, { id: queueEntryId, text }]);
        }
        toast.success("Prompt queued.");
      } catch (error) {
        toast.error(describePromptActionError(error, "Couldn't queue prompt."));
        throw error;
      }
    },
    [isBusyInputPending, promptControlsAvailable, queuePromptMutation, sessionId]
  );

  const handleInterruptPrompt = useCallback(
    async (message: string) => {
      const text = message.trim();
      if (!promptControlsAvailable || isBusyInputPending || text.length === 0) {
        return;
      }

      try {
        await interruptPromptMutation.mutateAsync({ id: sessionId, message: text });
        // Interrupt advances the busy-input generation, which cancels every pending
        // queued entry on the daemon — clear the local mirror so it can't lie.
        setQueuedPrompts([]);
        toast.success("Prompt interrupted.");
      } catch (error) {
        toast.error(describePromptActionError(error, "Couldn't interrupt prompt."));
        throw error;
      }
    },
    [interruptPromptMutation, isBusyInputPending, promptControlsAvailable, sessionId]
  );

  const handleSteerPrompt = useCallback(
    async (message: string) => {
      const text = message.trim();
      if (!promptControlsAvailable || isBusyInputPending || text.length === 0) {
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
    [isBusyInputPending, promptControlsAvailable, sessionId, steerPromptMutation]
  );

  const handleRemoveQueuedPrompt = useCallback(
    (queueEntryId: string) => {
      setQueuedPrompts(prev => prev.filter(item => item.id !== queueEntryId));
      cancelQueuedPromptMutation.mutate(
        { id: sessionId, queueEntryId },
        {
          onError: error => {
            toast.error(describePromptActionError(error, "Couldn't remove queued prompt."));
          },
        }
      );
    },
    [cancelQueuedPromptMutation, sessionId]
  );

  const handleSteerQueuedPrompt = useCallback(
    (prompt: QueuedPrompt) => {
      if (!promptControlsAvailable) {
        return;
      }
      setQueuedPrompts(prev => prev.filter(item => item.id !== prompt.id));
      steerPromptMutation.mutate(
        { id: sessionId, message: prompt.text },
        {
          onSuccess: () => {
            // Steer injects the queued text into the live turn now, so drop its
            // durable queue slot to avoid re-running it after the turn ends.
            cancelQueuedPromptMutation.mutate({ id: sessionId, queueEntryId: prompt.id });
            toast.success("Steer staged.");
          },
          onError: error => {
            toast.error(describePromptActionError(error, "Couldn't steer queued prompt."));
          },
        }
      );
    },
    [cancelQueuedPromptMutation, promptControlsAvailable, sessionId, steerPromptMutation]
  );

  const handleStop = useCallback(() => {
    if (controlsBusy) {
      return;
    }

    if (promptControlsAvailable) {
      handleCancelPrompt();
      return;
    }

    stopMutation.mutate(sessionId);
  }, [controlsBusy, handleCancelPrompt, promptControlsAvailable, sessionId, stopMutation]);

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
    if (controlsBusy || effectiveRunning) {
      return;
    }

    clearMutation.mutate(sessionId, {
      onSuccess: () => {
        aui.thread().reset();
      },
    });
  }, [aui, clearMutation, controlsBusy, effectiveRunning, sessionId]);

  return useMemo(
    () => ({
      canClear,
      canPrompt,
      allowBusyInput: userControllable,
      handleCancelPrompt,
      handleClear,
      handleDismissResumeFailure,
      handleDelete,
      handleInterruptPrompt,
      handleQueuePrompt,
      handleRemoveQueuedPrompt,
      handleResume,
      handleSteerPrompt,
      handleSteerQueuedPrompt,
      handleStop,
      isBusyInputPending,
      isClearing,
      isDeleting,
      isResuming,
      isSessionRunning: daemonRunning,
      isStopping,
      messages,
      queuedPrompts,
      resumeFailure,
    }),
    [
      canClear,
      canPrompt,
      userControllable,
      handleCancelPrompt,
      handleClear,
      handleDismissResumeFailure,
      handleDelete,
      handleInterruptPrompt,
      handleQueuePrompt,
      handleRemoveQueuedPrompt,
      handleResume,
      handleSteerPrompt,
      handleSteerQueuedPrompt,
      handleStop,
      isBusyInputPending,
      isClearing,
      isDeleting,
      isResuming,
      daemonRunning,
      isStopping,
      messages,
      queuedPrompts,
      resumeFailure,
    ]
  );
}
