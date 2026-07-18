import { createFileRoute } from "@tanstack/react-router";

import { BundleActivationDetail } from "@/systems/extensions";
import type { TopbarRouteContext } from "@/types/topbar";

export const Route = createFileRoute("/_app/marketplace/bundles/activations/$id")({
  beforeLoad: ({ params }): { topbar: TopbarRouteContext } => ({
    topbar: {
      crumb: { label: params.id },
    },
  }),
  component: BundleActivationDetailRoute,
});

function BundleActivationDetailRoute() {
  return <BundleActivationDetail id={Route.useParams().id} />;
}
