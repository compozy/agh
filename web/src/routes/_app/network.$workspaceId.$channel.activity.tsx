import { createFileRoute } from "@tanstack/react-router";

import type { TopbarRouteContext } from "@/types/topbar";
import { ActivityFeed, useNetworkListFiltersContext } from "@/systems/network";
import { preloadNetworkActivityRoute } from "./-network-preload";

export const Route = createFileRoute("/_app/network/$workspaceId/$channel/activity")({
  beforeLoad: ({ params }): { topbar: TopbarRouteContext } => ({
    topbar: { crumb: { label: `#${params.channel} · Activity` } },
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
        isFiltered={isFiltered}
        status={threadsQuery.isLoading || directsQuery.isLoading ? "loading" : "ready"}
        pagination={{
          threads:
            threadsQuery.hasMore || threadsQuery.isLoadingMore
              ? {
                  status: threadsQuery.isLoadingMore ? "loading" : "available",
                  onLoadMore: threadsQuery.loadMore,
                }
              : undefined,
          directs:
            directsQuery.hasMore || directsQuery.isLoadingMore
              ? {
                  status: directsQuery.isLoadingMore ? "loading" : "available",
                  onLoadMore: directsQuery.loadMore,
                }
              : undefined,
        }}
        sort={sort}
        threadTotal={threadsQuery.total}
        threads={filteredThreads}
      />
    </section>
  );
}
