import { useState } from "react";
import { Network as NetworkIcon, Plus } from "lucide-react";
import { createFileRoute, Outlet, useParams } from "@tanstack/react-router";

import { Button } from "@agh/ui";

import type { TopbarRouteContext } from "@/types/topbar";
import {
  DirectsEmpty,
  DirectsList,
  NewDirectDialog,
  useNetworkChannelDirectsRoute,
  useNetworkListFilters,
} from "@/systems/network";
import { ListFilterBar } from "@/systems/network/components/shell";

interface DirectDetailParams {
  directId?: string;
}

export const Route = createFileRoute("/_app/network/$channel/directs")({
  beforeLoad: ({ params }): { topbar: TopbarRouteContext } => ({
    topbar: { title: `#${params.channel} · Directs`, icon: NetworkIcon },
  }),
  component: NetworkChannelDirectsRoute,
});

function NetworkChannelDirectsRoute() {
  const { channel } = Route.useParams();
  const detailParams = useParams({ strict: false }) as DirectDetailParams;
  const route = useNetworkChannelDirectsRoute(channel);
  const filters = useNetworkListFilters({
    channel,
    threads: [],
    directs: route.directs.directs,
  });
  const [newDirectOpen, setNewDirectOpen] = useState(false);

  if (detailParams.directId) {
    return (
      <section
        aria-label={`Direct room ${detailParams.directId} in #${channel}`}
        className="flex min-h-0 flex-1 flex-col"
        data-testid="network-direct-detail-slot"
      >
        <Outlet />
      </section>
    );
  }

  const directsQuery = route.directs;
  const activeSession = route.session;
  const channelMembers = route.members;
  const visibleDirects = filters.filteredDirects;
  const showEmpty = !directsQuery.isLoading && visibleDirects.length === 0;
  const sessionId = activeSession.session?.sessionId ?? "";
  const totalDirects = directsQuery.directs.length;
  const subheaderLabel =
    totalDirects === 1
      ? "1 DIRECT ROOM IN THIS CHANNEL"
      : `${totalDirects} DIRECT ROOMS IN THIS CHANNEL`;

  return (
    <section
      aria-label={`Direct rooms in #${channel}`}
      className="flex min-h-0 flex-1 flex-col"
      data-testid="network-directs-tab"
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

      <header
        className="flex items-center justify-between gap-3 border-b border-(--line) px-5 py-2"
        data-testid="network-directs-subheader"
      >
        <span className="font-mono text-badge font-semibold uppercase tracking-mono text-(--subtle)">
          {subheaderLabel}
        </span>
        <Button
          aria-label="Open new direct room"
          data-testid="network-directs-new-direct"
          disabled={!sessionId}
          onClick={() => setNewDirectOpen(true)}
          size="sm"
          type="button"
          variant="outline"
        >
          <Plus aria-hidden="true" className="size-3.5" />
          New direct
        </Button>
      </header>

      {showEmpty ? (
        <div className="flex flex-1 items-center justify-center px-6 py-10">
          <DirectsEmpty
            className="max-w-md"
            onNewDirect={sessionId ? () => setNewDirectOpen(true) : undefined}
          />
        </div>
      ) : (
        <DirectsList
          activeDirectId={null}
          channel={channel}
          directs={visibleDirects}
          isLoading={directsQuery.isLoading}
          members={channelMembers.members}
          selfPeerId={activeSession.session?.peerId}
        />
      )}

      <NewDirectDialog
        channel={channel}
        onOpenChange={setNewDirectOpen}
        open={newDirectOpen && Boolean(sessionId)}
        selfPeerId={activeSession.session?.peerId}
        sessionId={sessionId}
      />
    </section>
  );
}
