import { createFileRoute } from "@tanstack/react-router";

import { createOsRouteSync } from "@/systems/os";
import { sessionReturnWorkspaceIdFromState } from "@/systems/session";
import { prefetchAgentSessionRoute } from "./-agent-session-route-loader";

export const Route = createFileRoute("/_app/agents/$name/sessions/$id")({
  beforeLoad: ({ params, location }) => ({
    topbar: { crumb: { label: "Session" } },
    sessionReturnWorkspaceId: sessionReturnWorkspaceIdFromState(location.state, params.id),
  }),
  loader: ({ context, params, preload }) =>
    prefetchAgentSessionRoute({
      queryClient: context.queryClient,
      sessionId: params.id,
      returnWorkspaceId: context.sessionReturnWorkspaceId,
      preload,
    }),
  component: createOsRouteSync("session"),
});
