import { createFileRoute } from "@tanstack/react-router";

import { BundleActivationDetail } from "@/systems/extensions";
import type { TopbarRouteContext } from "@/types/topbar";

export const Route = createFileRoute("/_app/extensions/bundles/$id")({
  beforeLoad: ({ params }): { topbar: TopbarRouteContext } => ({
    topbar: {
      parentCrumb: { label: "Bundles", search: { tab: "bundles" }, to: "/extensions" },
      crumb: { label: params.id },
    },
  }),
  component: BundleActivationDetailRoute,
});

function BundleActivationDetailRoute() {
  return <BundleActivationDetail id={Route.useParams().id} />;
}
