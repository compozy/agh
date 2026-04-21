import { useCallback, useEffect, useEffectEvent, useMemo } from "react";
import { useNavigate } from "@tanstack/react-router";
import { toast } from "sonner";

import type {
  MessageComposerChannel,
  MessageComposerPayload,
} from "@/systems/session/components/message-composer";
import {
  useClearSessionConversation,
  useResumeSession,
  useStopSession,
} from "@/systems/session/hooks/use-session-actions";
import { useSessionChat } from "@/systems/session/hooks/use-session-chat";
import { useSessionStore } from "@/systems/session/hooks/use-session-store";
import { useSessionTranscript } from "@/systems/session/hooks/use-session-transcript";
import { useSession } from "@/systems/session/hooks/use-sessions";
import { useNetworkChannels } from "@/systems/network";
import { useWorkspaces } from "@/systems/workspace";

function useSessionPage(id: string) {
  const navigate = useNavigate();

  const { data: session, isLoading, error } = useSession(id);
  const { data: workspaces } = useWorkspaces();
  const activeSessionId = useSessionStore(state => state.activeSessionId);
  const historyMessages = useSessionStore(state => state.historyMessages);
  const liveMessages = useSessionStore(state => state.liveMessages);
  const isStreaming = useSessionStore(state => state.isStreaming);
  const awaitingTranscriptSync = useSessionStore(state => state.awaitingTranscriptSync);
  const pendingPermission = useSessionStore(state => state.pendingPermission);

  const {
    transcriptMessages,
    isLoadingTranscript,
    error: transcriptError,
  } = useSessionTranscript(id);
  const canPrompt = session?.state === "active";
  const {
    sendMessage: sendChatMessage,
    stop: stopChatPrompt,
    resetLiveState,
    status,
    isStoppingPrompt,
  } = useSessionChat({ sessionId: id });
  const stopMutation = useStopSession();
  const resumeMutation = useResumeSession();
  const clearMutation = useClearSessionConversation();

  const { data: channelsData } = useNetworkChannels({ enabled: canPrompt ?? false });

  const resetSessionView = useEffectEvent(() => {
    resetLiveState();
    useSessionStore.getState().setActiveSession(id, []);
  });

  useEffect(() => {
    resetSessionView();
  }, [id]);

  const syncTranscriptHistory = useEffectEvent(
    (messages: NonNullable<typeof transcriptMessages>) => {
      useSessionStore.getState().replaceHistoryMessages(messages);
      if (useSessionStore.getState().awaitingTranscriptSync) {
        resetLiveState();
      }
    }
  );

  useEffect(() => {
    if (!transcriptMessages || activeSessionId !== id) {
      return;
    }

    syncTranscriptHistory(transcriptMessages);
  }, [activeSessionId, id, transcriptMessages]);

  useEffect(() => {
    if (error?.message?.includes("not found")) {
      toast.error("Session not found");
      navigate({ to: "/" });
    }
  }, [error, navigate]);

  const handlePermissionResolved = useCallback(() => {
    useSessionStore.getState().setPendingPermission(null);
  }, []);

  const handleResume = useCallback(() => {
    resumeMutation.mutate(id);
  }, [id, resumeMutation]);

  const handleStop = useCallback(() => {
    if (isStreaming || status === "submitted" || status === "streaming") {
      stopChatPrompt();
      return;
    }

    stopMutation.mutate(id);
  }, [id, isStreaming, status, stopChatPrompt, stopMutation]);

  const handleClear = useCallback(() => {
    if (clearMutation.isPending) {
      return;
    }

    resetLiveState();
    clearMutation.mutate(id);
  }, [clearMutation, id, resetLiveState]);

  const handleSend = useCallback(
    (payload: MessageComposerPayload) => {
      sendChatMessage(payload.text);
    },
    [sendChatMessage]
  );

  const workspaceName = workspaces?.find(workspace => workspace.id === session?.workspace_id)?.name;

  const channels = useMemo<MessageComposerChannel[]>(() => {
    return (channelsData?.channels ?? []).map(channel => ({
      id: channel.channel,
      name: channel.channel,
    }));
  }, [channelsData]);

  const messages = useMemo(() => {
    return [...historyMessages, ...liveMessages];
  }, [historyMessages, liveMessages]);

  const isStopping = stopMutation.isPending || isStoppingPrompt;
  const isResuming = resumeMutation.isPending;
  const isClearing = clearMutation.isPending;
  const controlsBusy = isStopping || isResuming || isClearing;
  const isDisabled = !canPrompt || isStreaming || awaitingTranscriptSync || isClearing;
  const isInert = pendingPermission !== null;
  const canClear =
    messages.length > 0 &&
    !controlsBusy &&
    !isStreaming &&
    !awaitingTranscriptSync &&
    pendingPermission === null &&
    status !== "submitted" &&
    status !== "streaming";

  return {
    canClear,
    canPrompt,
    channels,
    fatalErrorMessage: error?.message ?? transcriptError?.message ?? "Session not found",
    handleClear,
    handlePermissionResolved,
    handleResume,
    handleSend,
    handleStop,
    isClearing,
    isDisabled,
    isInert,
    isLoading: isLoading || isLoadingTranscript,
    isResuming,
    isStopping,
    isStreaming,
    messages,
    pendingPermission,
    session,
    workspaceName,
  };
}

export { useSessionPage };
