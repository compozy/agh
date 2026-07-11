import { useState } from "react";
import { useChildMatches, useNavigate } from "@tanstack/react-router";

import { useLoopRuns, type LoopOutcomeValue } from "@/systems/loops";
import { useActiveWorkspace } from "@/systems/workspace";

export interface LoopRunsRouteSearch {
  origin?: "catalog" | "session";
  origin_session?: string;
}

export function useLoopRunsRoute(search: LoopRunsRouteSearch) {
  const childMatches = useChildMatches();
  const hasChildMatch = childMatches.length > 0;
  const { activeWorkspace, activeWorkspaceId } = useActiveWorkspace();
  const workspaceId = activeWorkspaceId ?? "";
  const workspaceLabel = activeWorkspace?.name ?? activeWorkspace?.id ?? "workspace";
  const [outcome, setOutcome] = useState<LoopOutcomeValue>("all");
  const navigate = useNavigate({ from: "/loop-runs" });
  const runsQuery = useLoopRuns(
    workspaceId,
    { origin: search.origin, origin_session: search.origin_session },
    workspaceId !== "" && !hasChildMatch
  );

  const setOrigin = (origin: LoopRunsRouteSearch["origin"]) => {
    void navigate({
      to: "/loop-runs",
      search: current => ({
        ...current,
        origin,
        origin_session: origin === "session" ? current.origin_session : undefined,
      }),
    });
  };

  const setOriginSession = (originSession: string) => {
    void navigate({
      to: "/loop-runs",
      search: current => ({
        ...current,
        origin: "session",
        origin_session: originSession || undefined,
      }),
    });
  };

  return {
    hasChildMatch,
    outcome,
    runsQuery,
    setOrigin,
    setOriginSession,
    setOutcome,
    workspaceId,
    workspaceLabel,
  };
}
