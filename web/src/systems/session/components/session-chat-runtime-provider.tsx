import { useMemo, type ReactNode } from "react";
import { AssistantRuntimeProvider, DataRenderers, Tools, useAui } from "@assistant-ui/react";

import { useActiveWorkspace } from "@/systems/workspace";

import { useSessionChatRuntime } from "../hooks/use-session-chat-runtime";
import {
  createAghEventDataUI,
  createAghPermissionDataUI,
  sessionToolkit,
} from "../lib/session-toolkit";

function SessionRuntimeExtensions({
  sessionId,
  workspaceId,
}: {
  sessionId: string;
  workspaceId: string;
}) {
  const PermissionDataUI = useMemo(
    () => createAghPermissionDataUI(workspaceId, sessionId),
    [sessionId, workspaceId]
  );
  const EventDataUI = useMemo(() => createAghEventDataUI(), []);

  return (
    <>
      <PermissionDataUI />
      <EventDataUI />
    </>
  );
}

export interface SessionChatRuntimeProviderProps {
  sessionId: string;
  workspaceId?: string;
  children: ReactNode;
}

export function SessionChatRuntimeProvider({
  sessionId,
  workspaceId,
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
      <SessionRuntimeExtensions sessionId={sessionId} workspaceId={resolvedWorkspaceId} />
      {children}
    </AssistantRuntimeProvider>
  );
}
