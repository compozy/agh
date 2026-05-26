import { useMemo, type ReactNode } from "react";
import {
  AssistantRuntimeProvider,
  DataRenderers,
  Tools,
  useAui,
  type ThreadMessageLike,
} from "@assistant-ui/react";

import { useActiveWorkspace } from "@/systems/workspace";

import { useSessionChatRuntime } from "../hooks/use-session-chat-runtime";
import {
  useSessionLiveTail,
  type SessionStreamEventSourceFactory,
} from "../hooks/use-session-live-tail";
import { useRuntimeTranscriptHydration } from "../hooks/use-runtime-transcript-hydration";
import {
  createAghEventDataUI,
  createAghPermissionDataUI,
  sessionToolkit,
} from "../lib/session-toolkit";
import { SessionTranscriptThreadProvider } from "../lib/session-transcript-thread-context";

function RuntimeTranscriptHydrator({ messages }: { messages: readonly ThreadMessageLike[] }) {
  useRuntimeTranscriptHydration(messages);
  return null;
}

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
  const { messages } = useSessionLiveTail({ sessionId, workspaceId, eventSourceFactory });

  return (
    <SessionTranscriptThreadProvider messages={messages}>
      <RuntimeTranscriptHydrator messages={messages} />
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
