import { createFileRoute } from "@tanstack/react-router";
import { User2 } from "lucide-react";

import { validateAgentDetailSearch } from "@/systems/agent";
import type { TopbarRouteContext } from "@/types/topbar";
import { AgentDetailPage } from "./-agent-detail-page";
import { preloadAgentDetailRoute } from "./-app-preload";

export const Route = createFileRoute("/_app/agents/$name")({
  beforeLoad: ({ params }): { topbar: TopbarRouteContext } => ({
    topbar: { title: params.name, icon: User2 },
  }),
  validateSearch: validateAgentDetailSearch,
  loader: ({ context, location, params }) =>
    location.pathname.split("/").filter(Boolean).length === 2
      ? preloadAgentDetailRoute(context.queryClient, params.name)
      : Promise.resolve(),
  component: AgentDetailRoute,
});

function AgentDetailRoute() {
  const { name } = Route.useParams();
  const rawSearch = Route.useSearch();
  return <AgentDetailPage name={name} rawSearch={rawSearch} />;
}
