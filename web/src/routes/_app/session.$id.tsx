import { AlertCircle, MessageCircle } from "lucide-react";
import type { QueryClient } from "@tanstack/react-query";
import { createFileRoute, redirect } from "@tanstack/react-router";

import type { TopbarRouteContext } from "@/types/topbar";
import type { RouterContext } from "@/integrations/tanstack-query/root-context";
import {
  SessionNotFoundError,
  sessionByIdOptions,
  sessionDetailOptions,
  sessionKeys,
  sessionTranscriptOptions,
  type SessionPayload,
} from "@/systems/session";

interface SessionPermalinkRouteContext {
  topbar: TopbarRouteContext;
  permalinkError?: string;
}

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

export async function redirectSessionPermalinkRoute({
  context,
  params,
}: {
  context: RouterContext;
  params: { id: string };
}): Promise<SessionPermalinkRouteContext> {
  const topbar = { title: "Session", icon: MessageCircle };
  let session: SessionPayload;
  try {
    session = await resolveSessionPermalink({
      queryClient: context.queryClient,
      sessionId: params.id,
    });
  } catch (error) {
    return {
      topbar,
      permalinkError:
        error instanceof SessionNotFoundError
          ? "Session not found"
          : error instanceof Error
            ? error.message
            : "Session not found",
    };
  }

  throw redirect({
    to: "/agents/$name/sessions/$id",
    params: { name: session.agent_name, id: session.id },
    replace: true,
  });
}

export async function resolveSessionPermalink({
  queryClient,
  sessionId,
}: {
  queryClient: QueryClient;
  sessionId: string;
}): Promise<SessionPayload> {
  const session = await queryClient.ensureQueryData(sessionByIdOptions(sessionId));
  const workspaceId = session.workspace_id?.trim();
  if (workspaceId) {
    queryClient.setQueryData(sessionKeys.detail(workspaceId, session.id), session);
    await Promise.allSettled([
      queryClient.ensureInfiniteQueryData(sessionTranscriptOptions(workspaceId, session.id)),
      queryClient.ensureQueryData(sessionDetailOptions(workspaceId, session.id)),
    ]);
  }
  return session;
}

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
