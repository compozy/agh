import {
  apiClient,
  apiRequestFailed,
  defaultApiErrorMessage,
  requireResponseData,
} from "@/lib/api-client";

import type { AgentPayload } from "../types";

export async function fetchAgents(
  workspace?: string | null,
  signal?: AbortSignal
): Promise<AgentPayload[]> {
  const { data, error, response } = await apiClient.GET("/api/agents", {
    params: { query: agentWorkspaceQuery(workspace) },
    signal,
  });
  if (apiRequestFailed(response, error)) {
    throw new Error(defaultApiErrorMessage("Failed to fetch agents", response, error));
  }
  return requireResponseData(data, response, "Failed to fetch agents").agents;
}

export async function fetchAgent(
  name: string,
  workspace?: string | null,
  signal?: AbortSignal
): Promise<AgentPayload> {
  const { data, error, response } = await apiClient.GET("/api/agents/{name}", {
    params: { path: { name }, query: agentWorkspaceQuery(workspace) },
    signal,
  });
  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new Error(`Agent not found: ${name}`);
    }
    throw new Error(defaultApiErrorMessage(`Failed to fetch agent "${name}"`, response, error));
  }
  return requireResponseData(data, response, `Failed to fetch agent "${name}"`).agent;
}

function agentWorkspaceQuery(workspace?: string | null): { workspace: string } | undefined {
  const trimmed = workspace?.trim();
  return trimmed ? { workspace: trimmed } : undefined;
}
