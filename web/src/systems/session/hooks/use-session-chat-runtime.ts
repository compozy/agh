import { startTransition, useMemo } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useRemoteThreadListRuntime } from "@assistant-ui/react";
import { AssistantChatTransport, useChatRuntime } from "@assistant-ui/react-ai-sdk";

import { sessionKeys } from "../lib/query-keys";
import { createSessionHistoryAdapter } from "../lib/session-history-adapter";
import { createSessionThreadListAdapter } from "../lib/session-thread-list-adapter";

export function useSessionChatRuntime({
  sessionId,
  workspaceId,
}: {
  sessionId: string;
  workspaceId?: string;
}) {
  const queryClient = useQueryClient();
  const history = useMemo(
    () => createSessionHistoryAdapter(sessionId, queryClient),
    [queryClient, sessionId]
  );
  const threadListAdapter = useMemo(
    () => createSessionThreadListAdapter({ queryClient, workspaceId }),
    [queryClient, workspaceId]
  );
  const transport = useMemo(
    () =>
      new AssistantChatTransport({
        api: `/api/sessions/${sessionId}/prompt`,
      }),
    [sessionId]
  );

  return useRemoteThreadListRuntime({
    threadId: sessionId,
    adapter: threadListAdapter,
    runtimeHook: function SessionRuntimeHook() {
      return useChatRuntime({
        transport,
        adapters: { history },
        onFinish: () => {
          startTransition(() => {
            void queryClient.invalidateQueries({ queryKey: sessionKeys.detail(sessionId) });
            void queryClient.invalidateQueries({ queryKey: sessionKeys.history(sessionId) });
            void queryClient.invalidateQueries({ queryKey: sessionKeys.transcript(sessionId) });
            void queryClient.invalidateQueries({ queryKey: sessionKeys.lists() });
          });
        },
      });
    },
  });
}
