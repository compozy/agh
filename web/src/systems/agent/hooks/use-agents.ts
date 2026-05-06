import { useQuery } from "@tanstack/react-query";

import { agentsListOptions, agentDetailOptions } from "../lib/query-options";

export function useAgents(workspace?: string | null) {
  return useQuery(agentsListOptions(workspace));
}

export function useAgent(name: string, workspace?: string | null) {
  return useQuery(agentDetailOptions(name, workspace));
}
