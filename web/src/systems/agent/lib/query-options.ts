import { queryOptions } from "@tanstack/react-query";

import { fetchAgent, fetchAgents } from "../adapters/agent-api";
import { agentKeys } from "./query-keys";

export function agentsListOptions(workspace?: string | null) {
  return queryOptions({
    queryKey: agentKeys.list(workspace),
    queryFn: ({ signal }) => fetchAgents(workspace, signal),
    staleTime: 60_000,
  });
}

export function agentDetailOptions(name: string, workspace?: string | null) {
  return queryOptions({
    queryKey: agentKeys.detail(name, workspace),
    queryFn: ({ signal }) => fetchAgent(name, workspace, signal),
    staleTime: 60_000,
    enabled: !!name,
  });
}
