import { startTransition, useMemo } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useRemoteThreadListRuntime } from "@assistant-ui/react";
import { AssistantChatTransport, useChatRuntime } from "@assistant-ui/react-ai-sdk";

import { loopsKeys } from "@/systems/loops";
import { sessionKeys } from "../lib/query-keys";
import { createGoalAwareFetch } from "../lib/session-goal-chat-transport";
import { createSessionThreadListAdapter } from "../lib/session-thread-list-adapter";
import { useSessionStore } from "./use-session-store";

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
  const transport = useMemo(() => {
    const goalAwareFetch = createGoalAwareFetch({
      onResult: (result, requestText) => {
        useSessionStore.getState().setGoalResult(sessionId, result, requestText ?? undefined);
        if (result.snapshot !== null || result.outcome === "cleared") {
          queryClient.setQueryData(sessionKeys.goal(workspaceId, sessionId), {
            goal: result.snapshot,
          });
        }
        const runId = result.snapshot?.run_id;
        if (runId) {
          void queryClient.invalidateQueries({
            queryKey: loopsKeys.runDetail(workspaceId, runId),
          });
        }
        void queryClient.invalidateQueries({
          queryKey: loopsKeys.runsByWorkspace(workspaceId),
        });
      },
    });
    return new AssistantChatTransport({
      api: `/api/workspaces/${encodeURIComponent(workspaceId)}/sessions/${encodeURIComponent(sessionId)}/prompt`,
      fetch: goalAwareFetch,
    });
  }, [queryClient, workspaceId, sessionId]);

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
            void queryClient.invalidateQueries({ queryKey: sessionKeys.lists() });
          });
        },
      });
    },
  });
}
