import { Network as NetworkIcon } from "lucide-react";
import { createFileRoute, Outlet } from "@tanstack/react-router";

import type { TopbarRouteContext } from "@/types/topbar";
import {
  ChannelThreadComposer,
  ThreadsList,
  useNetworkChannelThreadsRoute,
  useNetworkListFiltersContext,
} from "@/systems/network";
import { preloadNetworkThreadsRoute } from "./-network-preload";

interface ThreadsRouteSearch {
  view?: "full";
}

export const Route = createFileRoute("/_app/network/$workspaceId/$channel/threads")({
  beforeLoad: ({ params }): { topbar: TopbarRouteContext } => ({
    topbar: { title: `#${params.channel} · Threads`, icon: NetworkIcon },
  }),
  component: NetworkChannelThreadsRoute,
  loader: ({ context, params }) =>
    preloadNetworkThreadsRoute(context.queryClient, params.workspaceId, params.channel),
  validateSearch: (search: Record<string, unknown>): ThreadsRouteSearch => ({
    view: search.view === "full" ? "full" : undefined,
  }),
});

function NetworkChannelThreadsRoute() {
  const { workspaceId, channel } = Route.useParams();
  const search = Route.useSearch();
  const route = useNetworkChannelThreadsRoute({ workspaceId, channel, view: search.view });
  const { activeThreadId, isFullPage, showOverlay, showList, activeSession } = route;
  const { filteredThreads, threadsQuery, isFiltered } = useNetworkListFiltersContext();

  return (
    <section
      aria-label={`Threads in #${channel}`}
      className="flex min-h-0 flex-1 flex-col"
      data-testid="network-threads-tab"
    >
      <div className="flex min-h-0 min-w-0 flex-1">
        {showList ? (
          <ThreadsList
            workspaceId={workspaceId}
            activeThreadId={activeThreadId}
            channel={channel}
            dim={showOverlay && !isFullPage}
            isLoading={threadsQuery.isLoading}
            hasMore={threadsQuery.hasMore}
            isLoadingMore={threadsQuery.isLoadingMore}
            isFiltered={isFiltered}
            onLoadMore={threadsQuery.loadMore}
            threads={filteredThreads}
            total={threadsQuery.total}
          />
        ) : null}

        {showOverlay && isFullPage ? (
          <div
            className="flex min-h-0 flex-1 flex-col bg-canvas"
            data-testid="network-thread-overlay-fullpage"
          >
            <Outlet />
          </div>
        ) : null}
      </div>

      {showList && !showOverlay ? (
        <ChannelThreadComposer
          workspaceId={workspaceId}
          channel={channel}
          disabledReason={activeSession.disabledReason ?? undefined}
          displayName={activeSession.session?.displayName}
          peerFrom={activeSession.session?.peerId ?? ""}
          sessionId={activeSession.session?.sessionId ?? ""}
        />
      ) : null}
    </section>
  );
}
