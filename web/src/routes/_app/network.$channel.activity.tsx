import { createFileRoute } from "@tanstack/react-router";

import {
  ActivityFeed,
  useNetworkDirects,
  useNetworkListFilters,
  useNetworkThreads,
} from "@/systems/network";
import { ListFilterBar } from "@/systems/network/components/shell";

export const Route = createFileRoute("/_app/network/$channel/activity")({
  component: NetworkChannelActivityRoute,
});

function NetworkChannelActivityRoute() {
  const { channel } = Route.useParams();
  const threadsQuery = useNetworkThreads(channel);
  const directsQuery = useNetworkDirects(channel);
  const filters = useNetworkListFilters({
    channel,
    threads: threadsQuery.threads,
    directs: directsQuery.directs,
  });

  return (
    <section
      aria-label={`Activity in #${channel}`}
      className="flex min-h-0 flex-1 flex-col"
      data-testid="network-activity-tab"
    >
      <ListFilterBar
        counts={filters.counts}
        filter={filters.filter}
        isMarkAllReadDisabled={filters.counts.unread === 0}
        onFilterChange={filters.setFilter}
        onMarkAllRead={filters.markAllRead}
        onSortChange={filters.setSort}
        sort={filters.sort}
      />
      <ActivityFeed
        channel={channel}
        directs={filters.filteredDirects}
        isLoading={threadsQuery.isLoading || directsQuery.isLoading}
        threads={filters.filteredThreads}
      />
    </section>
  );
}
