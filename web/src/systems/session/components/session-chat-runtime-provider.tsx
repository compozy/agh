import { useMemo, type ReactNode } from "react";
import {
  AssistantRuntimeProvider,
  DataRenderers,
  Tools,
  useAui,
  useAuiState,
} from "@assistant-ui/react";

import { useActiveWorkspace } from "@/systems/workspace";

import { useSessionChatRuntime } from "../hooks/use-session-chat-runtime";
import {
  useSessionLiveTail,
  type SessionStreamEventSourceFactory,
} from "../hooks/use-session-live-tail";
import {
  createAghEventDataUI,
  createAghPermissionDataUI,
  sessionToolkit,
} from "../lib/session-toolkit";
import { mergeSessionThreadReadModel } from "../lib/session-thread-read-model";
import { SessionTranscriptThreadProvider } from "../lib/session-transcript-thread-context";

function SessionRuntimeExtensions({
  sessionId,
  workspaceId,
  eventSourceFactory,
  children,
}: {
  sessionId: string;
  workspaceId: string;
  eventSourceFactory?: SessionStreamEventSourceFactory;
  children: ReactNode;
}) {
  const PermissionDataUI = useMemo(
    () => createAghPermissionDataUI(workspaceId, sessionId),
    [sessionId, workspaceId]
  );
  const EventDataUI = useMemo(() => createAghEventDataUI(), []);
  const runtimeMessages = useAuiState(state => state.thread.messages);
  const transcript = useSessionLiveTail({ sessionId, workspaceId, eventSourceFactory });
  const messages = useMemo(
    () =>
      mergeSessionThreadReadModel({
        transcriptMessages: transcript.messages,
        runtimeMessages,
      }),
    [runtimeMessages, transcript.messages]
  );

  return (
    <SessionTranscriptThreadProvider
      messages={messages}
      status={transcript.status}
      isPending={transcript.isPending}
      isError={transcript.isError}
      error={transcript.error}
      retry={transcript.retry}
    >
      <PermissionDataUI />
      <EventDataUI />
      {children}
    </SessionTranscriptThreadProvider>
  );
}

export interface SessionChatRuntimeProviderProps {
  sessionId: string;
  workspaceId?: string;
  eventSourceFactory?: SessionStreamEventSourceFactory;
  children: ReactNode;
}

export function SessionChatRuntimeProvider({
  sessionId,
  workspaceId,
  eventSourceFactory,
  children,
}: SessionChatRuntimeProviderProps) {
  const { activeWorkspaceId } = useActiveWorkspace();
  const resolvedWorkspaceId = workspaceId ?? activeWorkspaceId ?? "";
  const runtime = useSessionChatRuntime({ sessionId, workspaceId: resolvedWorkspaceId });
  const aui = useAui({
    tools: Tools({ toolkit: sessionToolkit }),
    dataRenderers: DataRenderers(),
  });

  return (
    <AssistantRuntimeProvider runtime={runtime} aui={aui}>
      <SessionRuntimeExtensions
        sessionId={sessionId}
        workspaceId={resolvedWorkspaceId}
        eventSourceFactory={eventSourceFactory}
      >
        {children}
      </SessionRuntimeExtensions>
    </AssistantRuntimeProvider>
  );
}
