import { useQuery } from "@tanstack/react-query";

import { agentsListOptions, agentDetailOptions } from "../lib/query-options";

export function useAgents() {
  return useQuery(agentsListOptions());
}

export function useAgent(name: string) {
  return useQuery(agentDetailOptions(name));
}
