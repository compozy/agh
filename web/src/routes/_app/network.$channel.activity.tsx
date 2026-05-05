import { createFileRoute } from "@tanstack/react-router";

import { ActivityFeed, useNetworkDirects, useNetworkThreads } from "@/systems/network";

export const Route = createFileRoute("/_app/network/$channel/activity")({
  component: NetworkChannelActivityRoute,
});

function NetworkChannelActivityRoute() {
  const { channel } = Route.useParams();
  const threadsQuery = useNetworkThreads(channel);
  const directsQuery = useNetworkDirects(channel);

  return (
    <section
      aria-label={`Activity in #${channel}`}
      className="flex min-h-0 flex-1 flex-col"
      data-testid="network-activity-tab"
    >
      <ActivityFeed
        channel={channel}
        directs={directsQuery.directs}
        isLoading={threadsQuery.isLoading || directsQuery.isLoading}
        threads={threadsQuery.threads}
      />
    </section>
  );
}
