import { Waypoints } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";

import type { TopbarRouteContext } from "@/types/topbar";
import { useBridgeDetailPage } from "@/hooks/routes/use-bridge-detail-page";
import { BridgeDetailPanel, BridgeEditDialog, BridgeTestDeliveryDialog } from "@/systems/bridges";
import { preloadBridgeDetailRoute } from "./-bridges-preload";

export const Route = createFileRoute("/_app/bridges/$id")({
  beforeLoad: ({ params }): { topbar: TopbarRouteContext } => ({
    topbar: { title: params.id, icon: Waypoints },
  }),
  loader: ({ context, params }) => preloadBridgeDetailRoute(context.queryClient, params.id),
  component: BridgeDetailRoute,
});

function BridgeDetailRoute() {
  const { id } = Route.useParams();
  const page = useBridgeDetailPage(id);

  return (
    <>
      <BridgeDetailPanel {...page.detailPanelProps} />
      <BridgeEditDialog {...page.editDialogProps} />
      <BridgeTestDeliveryDialog {...page.testDeliveryDialogProps} />
    </>
  );
}
