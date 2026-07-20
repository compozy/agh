import { useQuery } from "@tanstack/react-query";
import { AlertCircle } from "lucide-react";
import { Suspense, lazy } from "react";

import { Spinner } from "@agh/ui";

import { sessionByIdOptions } from "@/systems/session";

import { useDesktop } from "../../hooks/use-desktop";
import { matchSessionInstance } from "../../lib/app-registry";

/**
 * The session view rehosted in a window (glue stays route-colocated until the
 * multi-instance session task rewrites it; the view itself is prop-driven).
 */
const SessionPage = lazy(() =>
  import("@/routes/_app/-session-page").then(m => ({ default: m.SessionPage }))
);

const SESSION_AGENT_PATTERN = /^\/agents\/([^/]+)\/sessions\//;

/**
 * Interim session window controller: parses `agent + session` identity from
 * the window's WM location and resolves the owning workspace the same way the
 * route loader does (session-by-id), then rehosts the existing session view.
 */
export function SessionWindow({ windowId }: { windowId: string }) {
  const pathname = useDesktop(state => state.windows[windowId]?.location.pathname ?? "");
  const sessionId = matchSessionInstance(pathname);
  const agentMatch = SESSION_AGENT_PATTERN.exec(pathname);
  const agentName = agentMatch ? decodeURIComponent(agentMatch[1]) : null;

  const sessionQuery = useQuery({
    ...sessionByIdOptions(sessionId ?? ""),
    enabled: sessionId !== null,
  });

  if (sessionId === null || agentName === null) {
    return <SessionWindowNotice message="This window does not point at a session." />;
  }
  if (sessionQuery.isLoading) {
    return (
      <div
        className="flex min-h-full items-center justify-center"
        data-testid="session-route-loading"
      >
        <Spinner className="size-5 text-subtle" />
      </div>
    );
  }

  const workspaceId = sessionQuery.data?.workspace_id?.trim() || null;

  return (
    <Suspense
      fallback={
        <div className="flex min-h-full items-center justify-center">
          <Spinner className="size-5 text-subtle" />
        </div>
      }
    >
      <div className="flex min-h-full min-w-0 flex-col">
        <SessionPage name={agentName} id={sessionId} workspaceId={workspaceId} />
      </div>
    </Suspense>
  );
}

function SessionWindowNotice({ message }: { message: string }) {
  return (
    <div className="flex min-h-full items-center justify-center">
      <div className="flex flex-col items-center gap-2 text-center">
        <AlertCircle className="size-6 text-danger" />
        <p className="text-sm text-subtle">{message}</p>
      </div>
    </div>
  );
}
