import { createFileRoute, Outlet, useParams } from "@tanstack/react-router";

import { DirectsList, useNetworkDirects } from "@/systems/network";

interface DirectDetailParams {
  directId?: string;
}

export const Route = createFileRoute("/_app/network/$channel/directs")({
  component: NetworkChannelDirectsRoute,
});

function NetworkChannelDirectsRoute() {
  const { channel } = Route.useParams();
  const detailParams = useParams({ strict: false }) as DirectDetailParams;
  const directsQuery = useNetworkDirects(channel);

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

  return (
    <section
      aria-label={`Direct rooms in #${channel}`}
      className="flex min-h-0 flex-1 flex-col"
      data-testid="network-directs-tab"
    >
      <DirectsList
        activeDirectId={null}
        channel={channel}
        directs={directsQuery.directs}
        isLoading={directsQuery.isLoading}
      />
    </section>
  );
}
