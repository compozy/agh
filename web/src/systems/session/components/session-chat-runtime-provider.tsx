import { useMemo, type ReactNode } from "react";
import { AssistantRuntimeProvider, DataRenderers, Tools, useAui } from "@assistant-ui/react";

import { useSessionChatRuntime } from "../hooks/use-session-chat-runtime";
import { createAghPermissionDataUI, sessionToolkit } from "../lib/session-toolkit";

function SessionRuntimeExtensions({ sessionId }: { sessionId: string }) {
  const PermissionDataUI = useMemo(() => createAghPermissionDataUI(sessionId), [sessionId]);

  return <PermissionDataUI />;
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
  const runtime = useSessionChatRuntime({ sessionId, workspaceId });
  const aui = useAui({
    tools: Tools({ toolkit: sessionToolkit }),
    dataRenderers: DataRenderers(),
  });

  return (
    <AssistantRuntimeProvider runtime={runtime} aui={aui}>
      <SessionRuntimeExtensions sessionId={sessionId} />
      {children}
    </AssistantRuntimeProvider>
  );
}
