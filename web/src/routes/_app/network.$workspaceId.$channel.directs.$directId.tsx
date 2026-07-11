import { Network as NetworkIcon } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";

import type { TopbarRouteContext } from "@/types/topbar";
import { DirectRoom } from "@/systems/network";
import { preloadNetworkDirectDetailRoute } from "./-network-preload";

export const Route = createFileRoute("/_app/network/$workspaceId/$channel/directs/$directId")({
  beforeLoad: ({ params }): { topbar: TopbarRouteContext } => ({
    topbar: { title: `#${params.channel} · Direct`, icon: NetworkIcon },
  }),
  component: NetworkChannelDirectDetailRoute,
  loader: ({ context, params }) =>
    preloadNetworkDirectDetailRoute(
      context.queryClient,
      params.workspaceId,
      params.channel,
      params.directId
    ),
});

function NetworkChannelDirectDetailRoute() {
  const { workspaceId, channel, directId } = Route.useParams();
  return <DirectRoom channel={channel} directId={directId} workspaceId={workspaceId} />;
}
