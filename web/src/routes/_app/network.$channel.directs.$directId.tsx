import { Network as NetworkIcon } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";

import type { TopbarRouteContext } from "@/types/topbar";
import { DirectRoom } from "@/systems/network";

export const Route = createFileRoute("/_app/network/$channel/directs/$directId")({
  beforeLoad: ({ params }): { topbar: TopbarRouteContext } => ({
    topbar: { title: `#${params.channel} · Direct`, icon: NetworkIcon },
  }),
  component: NetworkChannelDirectDetailRoute,
});

function NetworkChannelDirectDetailRoute() {
  const { channel, directId } = Route.useParams();
  return <DirectRoom channel={channel} directId={directId} />;
}
