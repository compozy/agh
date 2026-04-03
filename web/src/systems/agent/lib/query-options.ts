import { queryOptions } from "@tanstack/react-query";

import { fetchAgent, fetchAgents } from "../adapters/agent-api";
import { agentKeys } from "./query-keys";

export function agentsListOptions() {
  return queryOptions({
    queryKey: agentKeys.list(),
    queryFn: ({ signal }) => fetchAgents(signal),
    staleTime: 60_000,
  });
}

export function agentDetailOptions(name: string) {
  return queryOptions({
    queryKey: agentKeys.detail(name),
    queryFn: ({ signal }) => fetchAgent(name, signal),
    staleTime: 60_000,
    enabled: !!name,
  });
}
