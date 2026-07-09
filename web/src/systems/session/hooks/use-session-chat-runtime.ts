import { startTransition, useMemo } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useRemoteThreadListRuntime } from "@assistant-ui/react";
import { AssistantChatTransport, useChatRuntime } from "@assistant-ui/react-ai-sdk";

import { sessionKeys } from "../lib/query-keys";
import { createSessionThreadListAdapter } from "../lib/session-thread-list-adapter";

export function useSessionChatRuntime({
  sessionId,
  workspaceId,
}: {
  sessionId: string;
  workspaceId: string;
}) {
  const queryClient = useQueryClient();
  const threadListAdapter = useMemo(
    () => createSessionThreadListAdapter({ queryClient, workspaceId }),
    [queryClient, workspaceId]
  );
  const transport = useMemo(
    () =>
      new AssistantChatTransport({
        api: `/api/workspaces/${encodeURIComponent(workspaceId)}/sessions/${encodeURIComponent(sessionId)}/prompt`,
      }),
    [workspaceId, sessionId]
  );

  return useRemoteThreadListRuntime({
    threadId: sessionId,
    adapter: threadListAdapter,
    runtimeHook: function SessionRuntimeHook() {
      return useChatRuntime({
        transport,
        onFinish: () => {
          startTransition(() => {
            void queryClient.invalidateQueries({
              queryKey: sessionKeys.detail(workspaceId, sessionId),
            });
            void queryClient.invalidateQueries({
              queryKey: sessionKeys.history(workspaceId, sessionId),
            });
            void queryClient.invalidateQueries({
              queryKey: sessionKeys.transcript(workspaceId, sessionId),
            });
            void queryClient.invalidateQueries({ queryKey: sessionKeys.lists() });
          });
        },
      });
    },
  });
}
