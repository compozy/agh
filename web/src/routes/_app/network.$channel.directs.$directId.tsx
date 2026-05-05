import { createFileRoute } from "@tanstack/react-router";

import { DirectRoom } from "@/systems/network";

export const Route = createFileRoute("/_app/network/$channel/directs/$directId")({
  component: NetworkChannelDirectDetailRoute,
});

function NetworkChannelDirectDetailRoute() {
  const { channel, directId } = Route.useParams();
  return <DirectRoom channel={channel} directId={directId} />;
}
