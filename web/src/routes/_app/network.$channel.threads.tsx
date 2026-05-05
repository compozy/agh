import { createFileRoute, Outlet } from "@tanstack/react-router";

import {
  ChannelThreadComposer,
  ThreadsList,
  useNetworkChannelThreadsRoute,
} from "@/systems/network";

interface ThreadsRouteSearch {
  view?: "full";
}

export const Route = createFileRoute("/_app/network/$channel/threads")({
  component: NetworkChannelThreadsRoute,
  validateSearch: (search: Record<string, unknown>): ThreadsRouteSearch => ({
    view: search.view === "full" ? "full" : undefined,
  }),
});

function NetworkChannelThreadsRoute() {
  const { channel } = Route.useParams();
  const search = Route.useSearch();
  const route = useNetworkChannelThreadsRoute({ channel, view: search.view });
  const { activeThreadId, isFullPage, showOverlay, showList, threadsQuery, activeSession } = route;

  return (
    <section
      aria-label={`Threads in #${channel}`}
      className="flex min-h-0 flex-1 flex-col"
      data-testid="network-threads-tab"
    >
      <div className="flex min-h-0 flex-1">
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
      </div>

      {showList && !showOverlay ? (
        <ChannelThreadComposer
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
