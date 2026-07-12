import { createFileRoute } from "@tanstack/react-router";
import { MessageCircle } from "lucide-react";

import type { TopbarRouteContext } from "@/types/topbar";
import { prefetchAgentSessionRoute } from "./-agent-session-route-loader";
import { SessionPage, SessionRouteLoading } from "./-session-page";

export const Route = createFileRoute("/_app/agents/$name/sessions/$id")({
  beforeLoad: ({ params }): { topbar: TopbarRouteContext } => ({
    topbar: { title: `${params.name} · Session`, icon: MessageCircle },
  }),
  loader: ({ context, params }) =>
    prefetchAgentSessionRoute({
      queryClient: context.queryClient,
      sessionId: params.id,
    }),
  pendingComponent: SessionRouteLoading,
  component: SessionRoutePage,
});

function SessionRoutePage() {
  const { name, id } = Route.useParams();
  const { workspaceId } = Route.useLoaderData();
  return <SessionPage name={name} id={id} workspaceId={workspaceId} />;
}
