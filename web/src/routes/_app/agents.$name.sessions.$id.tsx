import { createFileRoute } from "@tanstack/react-router";
import { MessageCircle } from "lucide-react";

import { sessionReturnWorkspaceIdFromState } from "@/systems/session";
import { prefetchAgentSessionRoute } from "./-agent-session-route-loader";
import { SessionPage, SessionRouteLoading } from "./-session-page";

export const Route = createFileRoute("/_app/agents/$name/sessions/$id")({
  beforeLoad: ({ params, location }) => ({
    topbar: { title: `${params.name} · Session`, icon: MessageCircle },
    sessionReturnWorkspaceId: sessionReturnWorkspaceIdFromState(location.state, params.id),
  }),
  loader: ({ context, params, preload }) =>
    prefetchAgentSessionRoute({
      queryClient: context.queryClient,
      sessionId: params.id,
      returnWorkspaceId: context.sessionReturnWorkspaceId,
      preload,
    }),
  pendingComponent: SessionRouteLoading,
  component: SessionRoutePage,
});

function SessionRoutePage() {
  const { name, id } = Route.useParams();
  const { workspaceId } = Route.useLoaderData();
  return <SessionPage name={name} id={id} workspaceId={workspaceId} />;
}
