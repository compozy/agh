import { useCallback, useEffect, useMemo, useRef } from "react";
import { useNavigate } from "@tanstack/react-router";
import { toast } from "sonner";

import type {
  MessageComposerChannel,
  MessageComposerPayload,
  MessageComposerSkill,
} from "@/systems/session/components/message-composer";
import { useSessionChat } from "@/systems/session/hooks/use-session-chat";
import { useResumeSession, useStopSession } from "@/systems/session/hooks/use-session-actions";
import { useSessionStore } from "@/systems/session/hooks/use-session-store";
import { useSessionTranscript } from "@/systems/session/hooks/use-session-transcript";
import { useSession } from "@/systems/session/hooks/use-sessions";
import { useNetworkChannels } from "@/systems/network";
import { useSkills } from "@/systems/skill";
import { useWorkspaces } from "@/systems/workspace";

function useSessionPage(id: string) {
  const navigate = useNavigate();
  const hydratedSessionIdRef = useRef<string | null>(null);

  const { data: session, isLoading, error } = useSession(id);
  const { data: workspaces } = useWorkspaces();
  const messages = useSessionStore(state => state.messages);
  const isStreaming = useSessionStore(state => state.isStreaming);
  const pendingPermission = useSessionStore(state => state.pendingPermission);
  const activeSessionId = useSessionStore(state => state.activeSessionId);

  const {
    transcriptMessages,
    isLoadingTranscript,
    error: transcriptError,
  } = useSessionTranscript(id);
  const canPrompt = session?.state === "active";
  const { sendMessage: sendChatMessage, status } = useSessionChat({ sessionId: id });
  const stopMutation = useStopSession();
  const resumeMutation = useResumeSession();

  const workspaceId = session?.workspace_id ?? "";
  const { data: skillsData } = useSkills(workspaceId);
  const { data: channelsData } = useNetworkChannels({ enabled: canPrompt ?? false });

  useEffect(() => {
    hydratedSessionIdRef.current = null;
    useSessionStore.setState({
      activeSessionId: id,
      isStreaming: false,
      messages: [],
      pendingPermission: null,
    });
  }, [id]);

  useEffect(() => {
    if (!transcriptMessages || activeSessionId !== id || hydratedSessionIdRef.current === id) {
      return;
    }

    useSessionStore.getState().setActiveSession(id, transcriptMessages);
    hydratedSessionIdRef.current = id;
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
    stopMutation.mutate(id);
  }, [id, stopMutation]);

  const handleSend = useCallback(
    (payload: MessageComposerPayload) => {
      sendChatMessage(payload.text);
    },
    [sendChatMessage]
  );

  const workspaceName = workspaces?.find(workspace => workspace.id === session?.workspace_id)?.name;
  const isDisabled =
    !canPrompt || isStreaming || status === "submitted" || pendingPermission !== null;

  const skills = useMemo<MessageComposerSkill[]>(() => {
    return (skillsData ?? [])
      .filter(skill => skill.enabled)
      .map(skill => ({
        id: skill.name,
        name: skill.name,
        description: skill.description ?? undefined,
      }));
  }, [skillsData]);

  const channels = useMemo<MessageComposerChannel[]>(() => {
    return (channelsData?.channels ?? []).map(channel => ({
      id: channel.channel,
      name: channel.channel,
    }));
  }, [channelsData]);

  return {
    canPrompt,
    channels,
    fatalErrorMessage: error?.message ?? transcriptError?.message ?? "Session not found",
    handlePermissionResolved,
    handleResume,
    handleSend,
    handleStop,
    isDisabled,
    isLoading: isLoading || isLoadingTranscript,
    isStreaming,
    messages,
    pendingPermission,
    session,
    skills,
    workspaceName,
  };
}

export { useSessionPage };
