import { createFileRoute } from "@tanstack/react-router";
import { Puzzle } from "lucide-react";

import { ExtensionDetail } from "@/systems/extensions";
import type { TopbarRouteContext } from "@/types/topbar";

export const Route = createFileRoute("/_app/extensions/$name")({
  beforeLoad: ({ params }): { topbar: TopbarRouteContext } => ({
    topbar: { title: params.name, icon: Puzzle },
  }),
  component: ExtensionDetailRoute,
});

function ExtensionDetailRoute() {
  return <ExtensionDetail name={Route.useParams().name} />;
}
