import { useParams } from "@tanstack/react-router";

import { useActiveNetworkSession, type UseActiveNetworkSessionResult } from "./use-active-session";
import { useNetworkThreads, type UseNetworkThreadsResult } from "./use-threads";
import { useThreadViewMode, type ThreadViewMode } from "./use-thread-view-mode";

export interface UseNetworkChannelThreadsRouteArgs {
  channel: string;
  view?: "full";
}

export interface UseNetworkChannelThreadsRouteResult {
  activeThreadId: string | null;
  viewMode: ThreadViewMode;
  isFullPage: boolean;
  showOverlay: boolean;
  showList: boolean;
  threadsQuery: UseNetworkThreadsResult;
  activeSession: UseActiveNetworkSessionResult;
}

interface ThreadDetailParams {
  threadId?: string;
}

/**
 * Composition hook for `<NetworkChannelThreadsRoute>` keeping it under the
 * `compozy-react(max-component-complexity)` cap.
 */
export function useNetworkChannelThreadsRoute({
  channel,
  view,
}: UseNetworkChannelThreadsRouteArgs): UseNetworkChannelThreadsRouteResult {
  const detailParams = useParams({ strict: false }) as ThreadDetailParams;
  const activeThreadId = detailParams.threadId ?? null;
  const viewMode = useThreadViewMode();
  const isFullPage = view === "full" || viewMode === "fullpage";
  const showOverlay = activeThreadId != null;
  const showList = !showOverlay || !isFullPage;

  const threadsQuery = useNetworkThreads(channel);
  const activeSession = useActiveNetworkSession(channel);

  return {
    activeThreadId,
    viewMode,
    isFullPage,
    showOverlay,
    showList,
    threadsQuery,
    activeSession,
  };
}
