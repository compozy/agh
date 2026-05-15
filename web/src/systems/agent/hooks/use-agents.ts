import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { workspaceKeys } from "@/systems/workspace";

import { createAgent } from "../adapters/agent-api";
import { agentsListOptions, agentDetailOptions } from "../lib/query-options";
import { agentKeys } from "../lib/query-keys";
import type { AgentPayload, CreateAgentParams } from "../types";

export function useAgents(workspace?: string | null) {
  return useQuery(agentsListOptions(workspace));
}

export function useAgent(name: string, workspace?: string | null) {
  return useQuery(agentDetailOptions(name, workspace));
}

export function useCreateAgent() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (params: CreateAgentParams) => createAgent(params),
    onSuccess: (agent, params) => {
      const workspace = params.scope === "workspace" ? params.workspace : null;
      queryClient.setQueryData<AgentPayload>(agentKeys.detail(agent.name, workspace), agent);
    },
    onSettled: (_agent, _error, params) => {
      queryClient.invalidateQueries({ queryKey: agentKeys.lists() });
      if (params?.scope === "workspace" && params.workspace) {
        queryClient.invalidateQueries({ queryKey: workspaceKeys.detail(params.workspace) });
      }
    },
  });
}
