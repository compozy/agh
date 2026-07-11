import { useParams } from "@tanstack/react-router";

import { useActiveNetworkSession, type UseActiveNetworkSessionResult } from "./use-active-session";
import { useThreadViewMode, type ThreadViewMode } from "./use-thread-view-mode";

export interface UseNetworkChannelThreadsRouteArgs {
  workspaceId: string;
  channel: string;
  view?: "full";
}

export interface UseNetworkChannelThreadsRouteResult {
  activeThreadId: string | null;
  viewMode: ThreadViewMode;
  isFullPage: boolean;
  showOverlay: boolean;
  showList: boolean;
  activeSession: UseActiveNetworkSessionResult;
}

export function useNetworkChannelThreadsRoute({
  workspaceId,
  channel,
  view,
}: UseNetworkChannelThreadsRouteArgs): UseNetworkChannelThreadsRouteResult {
  const detailParams = useParams({ strict: false }) as { threadId?: string };
  const activeThreadId = detailParams.threadId ?? null;
  const viewMode = useThreadViewMode();
  const isFullPage = view === "full" || viewMode === "fullpage";
  const showOverlay = activeThreadId != null;
  return {
    activeThreadId,
    viewMode,
    isFullPage,
    showOverlay,
    showList: !showOverlay || !isFullPage,
    activeSession: useActiveNetworkSession(channel, { workspaceId }),
  };
}
