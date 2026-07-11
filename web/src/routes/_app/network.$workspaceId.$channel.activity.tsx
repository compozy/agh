import { Network as NetworkIcon } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";

import type { TopbarRouteContext } from "@/types/topbar";
import { ActivityFeed, useNetworkListFiltersContext } from "@/systems/network";
import { preloadNetworkActivityRoute } from "./-network-preload";

export const Route = createFileRoute("/_app/network/$workspaceId/$channel/activity")({
  beforeLoad: ({ params }): { topbar: TopbarRouteContext } => ({
    topbar: { title: `#${params.channel} · Activity`, icon: NetworkIcon },
  }),
  component: NetworkChannelActivityRoute,
  loader: ({ context, params }) =>
    preloadNetworkActivityRoute(context.queryClient, params.workspaceId, params.channel),
});

function NetworkChannelActivityRoute() {
  const { workspaceId, channel } = Route.useParams();
  const { filteredThreads, filteredDirects, threadsQuery, directsQuery, isFiltered, sort } =
    useNetworkListFiltersContext();

  return (
    <section
      aria-label={`Activity in #${channel}`}
      className="flex min-h-0 flex-1 flex-col"
      data-testid="network-activity-tab"
    >
      <ActivityFeed
        workspaceId={workspaceId}
        channel={channel}
        directs={filteredDirects}
        directTotal={directsQuery.total}
        hasMoreDirects={directsQuery.hasMore}
        hasMoreThreads={threadsQuery.hasMore}
        isFiltered={isFiltered}
        isLoading={threadsQuery.isLoading || directsQuery.isLoading}
        isLoadingMoreDirects={directsQuery.isLoadingMore}
        isLoadingMoreThreads={threadsQuery.isLoadingMore}
        onLoadMoreDirects={directsQuery.loadMore}
        onLoadMoreThreads={threadsQuery.loadMore}
        sort={sort}
        threadTotal={threadsQuery.total}
        threads={filteredThreads}
      />
    </section>
  );
}
