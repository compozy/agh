import { useEffect } from "react";
import { AlertCircle, Loader2 } from "lucide-react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";

import { useSession } from "@/systems/session";

/**
 * Permalink-by-id redirect. Resolves the agent name for a session and
 * forwards to the canonical `/agents/$name/sessions/$id` route. Used by
 * external surfaces (automation history, task tree) that hold a session id
 * without the originating agent in scope.
 */
export const Route = createFileRoute("/_app/session/$id")({
  component: SessionPermalinkPage,
});

function SessionPermalinkPage() {
  const { id } = Route.useParams();
  const navigate = useNavigate();
  const { data: session, isLoading, error } = useSession(id);

  useEffect(() => {
    if (session?.agent_name) {
      void navigate({
        to: "/agents/$name/sessions/$id",
        params: { name: session.agent_name, id },
        replace: true,
      });
    }
  }, [session?.agent_name, id, navigate]);

  if (isLoading || (session && session.agent_name)) {
    return (
      <div
        className="flex flex-1 items-center justify-center"
        data-testid="session-permalink-loading"
      >
        <Loader2 className="size-5 animate-spin text-[color:var(--color-text-tertiary)]" />
      </div>
    );
  }

  return (
    <div
      className="flex flex-1 items-center justify-center"
      data-testid="session-permalink-not-found"
    >
      <div className="flex flex-col items-center gap-2 text-center">
        <AlertCircle className="size-6 text-[color:var(--color-danger)]" />
        <p className="text-sm text-[color:var(--color-text-tertiary)]">
          {error?.message ?? "Session not found"}
        </p>
      </div>
    </div>
  );
}
