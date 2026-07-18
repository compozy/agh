import { createFileRoute } from "@tanstack/react-router";

import { ExtensionDetail } from "@/systems/extensions";
import type { TopbarRouteContext } from "@/types/topbar";

export const Route = createFileRoute("/_app/extensions/$name")({
  beforeLoad: ({ params }): { topbar: TopbarRouteContext } => ({
    topbar: { crumb: { label: params.name } },
  }),
  component: ExtensionDetailRoute,
});

function ExtensionDetailRoute() {
  return <ExtensionDetail name={Route.useParams().name} />;
}
