import { startTransition, useMemo } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useRemoteThreadListRuntime } from "@assistant-ui/react";
import { AssistantChatTransport, useChatRuntime } from "@assistant-ui/react-ai-sdk";

import { useActiveWorkspace } from "@/systems/workspace";

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
  const { activeWorkspaceId } = useActiveWorkspace();
  const resolvedWorkspaceId = workspaceId ?? activeWorkspaceId ?? "";
  const history = useMemo(
    () => createSessionHistoryAdapter(resolvedWorkspaceId, sessionId, queryClient),
    [queryClient, resolvedWorkspaceId, sessionId]
  );
  const threadListAdapter = useMemo(
    () => createSessionThreadListAdapter({ queryClient, workspaceId: resolvedWorkspaceId }),
    [queryClient, resolvedWorkspaceId]
  );
  const transport = useMemo(
    () =>
      new AssistantChatTransport({
        api: `/api/workspaces/${encodeURIComponent(resolvedWorkspaceId)}/sessions/${encodeURIComponent(sessionId)}/prompt`,
      }),
    [resolvedWorkspaceId, sessionId]
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
            void queryClient.invalidateQueries({
              queryKey: sessionKeys.detail(resolvedWorkspaceId, sessionId),
            });
            void queryClient.invalidateQueries({
              queryKey: sessionKeys.history(resolvedWorkspaceId, sessionId),
            });
            void queryClient.invalidateQueries({
              queryKey: sessionKeys.transcript(resolvedWorkspaceId, sessionId),
            });
            void queryClient.invalidateQueries({ queryKey: sessionKeys.lists() });
          });
        },
      });
    },
  });
}
