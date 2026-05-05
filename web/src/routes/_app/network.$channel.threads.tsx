import { createFileRoute, Outlet, useParams } from "@tanstack/react-router";

import { ThreadsList, useNetworkThreads, useThreadViewMode } from "@/systems/network";

interface ThreadsRouteSearch {
  view?: "full";
}

interface ThreadDetailParams {
  threadId?: string;
}

export const Route = createFileRoute("/_app/network/$channel/threads")({
  component: NetworkChannelThreadsRoute,
  validateSearch: (search: Record<string, unknown>): ThreadsRouteSearch => ({
    view: search.view === "full" ? "full" : undefined,
  }),
});

function NetworkChannelThreadsRoute() {
  const { channel } = Route.useParams();
  const detailParams = useParams({ strict: false }) as ThreadDetailParams;
  const search = Route.useSearch();
  const activeThreadId = detailParams.threadId ?? null;
  const viewMode = useThreadViewMode();
  const isFullPage = search.view === "full" || viewMode === "fullpage";
  const showOverlay = activeThreadId != null;
  const showList = !showOverlay || !isFullPage;

  const threadsQuery = useNetworkThreads(channel);

  return (
    <section
      aria-label={`Threads in #${channel}`}
      className="flex min-h-0 flex-1"
      data-testid="network-threads-tab"
    >
      {showList ? (
        <ThreadsList
          activeThreadId={activeThreadId}
          channel={channel}
          dim={showOverlay && !isFullPage}
          isLoading={threadsQuery.isLoading}
          threads={threadsQuery.threads}
        />
      ) : null}

      {showOverlay && isFullPage ? (
        <div
          className="flex min-h-0 flex-1 flex-col bg-[color:var(--color-canvas-deep)]"
          data-testid="network-thread-overlay-fullpage"
        >
          <Outlet />
        </div>
      ) : null}
    </section>
  );
}
