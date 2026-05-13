import { Network as NetworkIcon } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";

import type { TopbarRouteContext } from "@/types/topbar";
import {
  ActivityFeed,
  useNetworkDirects,
  useNetworkListFiltersContext,
  useNetworkThreads,
} from "@/systems/network";

export const Route = createFileRoute("/_app/network/$workspaceId/$channel/activity")({
  beforeLoad: ({ params }): { topbar: TopbarRouteContext } => ({
    topbar: { title: `#${params.channel} · Activity`, icon: NetworkIcon },
  }),
  component: NetworkChannelActivityRoute,
});

function NetworkChannelActivityRoute() {
  const { workspaceId, channel } = Route.useParams();
  const threadsQuery = useNetworkThreads(channel);
  const directsQuery = useNetworkDirects(channel);
  const { filteredThreads, filteredDirects } = useNetworkListFiltersContext();

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
        isLoading={threadsQuery.isLoading || directsQuery.isLoading}
        threads={filteredThreads}
      />
    </section>
  );
}
