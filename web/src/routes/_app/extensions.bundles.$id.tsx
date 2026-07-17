import { createFileRoute } from "@tanstack/react-router";
import { Box } from "lucide-react";

import { BundleActivationDetail } from "@/systems/extensions";
import type { TopbarRouteContext } from "@/types/topbar";

export const Route = createFileRoute("/_app/extensions/bundles/$id")({
  beforeLoad: ({ params }): { topbar: TopbarRouteContext } => ({
    topbar: { title: params.id, icon: Box },
  }),
  component: BundleActivationDetailRoute,
});

function BundleActivationDetailRoute() {
  return <BundleActivationDetail id={Route.useParams().id} />;
}
