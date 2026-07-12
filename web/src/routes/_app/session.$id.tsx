import { AlertCircle } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";

import {
  redirectSessionPermalinkRoute,
  type SessionPermalinkRouteContext,
} from "./-session-permalink-route";

/**
 * Permalink-by-id redirect. Resolves the agent name for a session and
 * forwards to the canonical `/agents/$name/sessions/$id` route. Used by
 * external surfaces (automation history, task tree) that hold a session id
 * without the originating agent in scope.
 */
export const Route = createFileRoute("/_app/session/$id")({
  beforeLoad: redirectSessionPermalinkRoute,
  component: SessionPermalinkPage,
});

function SessionPermalinkPage() {
  const routeContext = Route.useRouteContext() as SessionPermalinkRouteContext;

  return (
    <div
      className="flex flex-1 items-center justify-center"
      data-testid="session-permalink-not-found"
    >
      <div className="flex flex-col items-center gap-2 text-center">
        <AlertCircle className="size-6 text-danger" />
        <p className="text-sm text-subtle">{routeContext.permalinkError ?? "Session not found"}</p>
      </div>
    </div>
  );
}
