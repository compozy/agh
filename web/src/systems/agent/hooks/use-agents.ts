import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { getNavCountsStore } from "@/systems/runtime";
import { workspaceKeys } from "@/systems/workspace";

import { createAgent, deleteAgent, duplicateAgent, updateAgent } from "../adapters/agent-api";
import { agentsListOptions, agentDetailOptions } from "../lib/query-options";
import { agentKeys } from "../lib/query-keys";
import type {
  AgentPayload,
  CreateAgentParams,
  DeleteAgentResponse,
  DuplicateAgentParams,
  UpdateAgentParams,
} from "../types";

interface UseAgentsOptions {
  enabled?: boolean;
}

export function useAgents(workspace?: string | null, options: UseAgentsOptions = {}) {
  return useQuery({
    ...agentsListOptions(workspace),
    enabled: options.enabled ?? true,
  });
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

export interface UpdateAgentVariables {
  name: string;
  params: UpdateAgentParams;
}

export function useUpdateAgent() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ name, params }: UpdateAgentVariables) => updateAgent(name, params),
    onSuccess: (agent, variables) => {
      const workspace = variables.params.workspace ?? null;
      queryClient.setQueryData<AgentPayload>(agentKeys.detail(agent.name, workspace), agent);
    },
    onSettled: (_agent, _error, variables) => {
      queryClient.invalidateQueries({ queryKey: agentKeys.lists() });
      if (variables?.params.workspace) {
        queryClient.invalidateQueries({
          queryKey: workspaceKeys.detail(variables.params.workspace),
        });
      }
    },
  });
}

export interface DeleteAgentVariables {
  name: string;
  workspace?: string | null;
}

export function useDeleteAgent() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ name, workspace }: DeleteAgentVariables) => deleteAgent(name, workspace),
    onSuccess: (_result: DeleteAgentResponse, variables) => {
      const workspace = variables.workspace ?? null;
      queryClient.removeQueries({ queryKey: agentKeys.detail(variables.name, workspace) });
      getNavCountsStore(workspace).getState().refresh();
    },
    onSettled: (_result, _error, variables) => {
      queryClient.invalidateQueries({ queryKey: agentKeys.lists() });
      if (variables?.workspace) {
        queryClient.invalidateQueries({ queryKey: workspaceKeys.detail(variables.workspace) });
      }
    },
  });
}

export interface DuplicateAgentVariables {
  sourceName: string;
  params: DuplicateAgentParams;
}

export function useDuplicateAgent() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ sourceName, params }: DuplicateAgentVariables) =>
      duplicateAgent(sourceName, params),
    onSuccess: (agent, variables) => {
      const workspace =
        variables.params.scope === "workspace" ? (variables.params.workspace ?? null) : null;
      queryClient.setQueryData<AgentPayload>(agentKeys.detail(agent.name, workspace), agent);
    },
    onSettled: (_agent, _error, variables) => {
      queryClient.invalidateQueries({ queryKey: agentKeys.lists() });
      if (variables?.params.scope === "workspace" && variables.params.workspace) {
        queryClient.invalidateQueries({
          queryKey: workspaceKeys.detail(variables.params.workspace),
        });
      }
    },
  });
}
