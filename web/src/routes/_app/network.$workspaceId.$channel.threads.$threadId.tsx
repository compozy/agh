import { Network as NetworkIcon } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";

import type { TopbarRouteContext } from "@/types/topbar";
import { ThreadOverlay, useThreadViewMode } from "@/systems/network";
import { preloadNetworkThreadDetailRoute } from "./-network-preload";

interface ThreadDetailSearch {
  view?: "full";
}

export const Route = createFileRoute("/_app/network/$workspaceId/$channel/threads/$threadId")({
  beforeLoad: ({ params }): { topbar: TopbarRouteContext } => ({
    topbar: { title: `#${params.channel} · Thread`, icon: NetworkIcon },
  }),
  component: NetworkChannelThreadDetailRoute,
  loader: ({ context, params }) =>
    preloadNetworkThreadDetailRoute(
      context.queryClient,
      params.workspaceId,
      params.channel,
      params.threadId
    ),
  validateSearch: (search: Record<string, unknown>): ThreadDetailSearch => ({
    view: search.view === "full" ? "full" : undefined,
  }),
});

function NetworkChannelThreadDetailRoute() {
  const { workspaceId, channel, threadId } = Route.useParams();
  const search = Route.useSearch();
  const viewMode = useThreadViewMode();
  const fullPage = search.view === "full" || viewMode === "fullpage";

  return (
    <ThreadOverlay
      workspaceId={workspaceId}
      channel={channel}
      fullPage={fullPage}
      threadId={threadId}
    />
  );
}
