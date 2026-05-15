import {
  apiClient,
  apiRequestFailed,
  defaultApiErrorMessage,
  requireResponseData,
} from "@/lib/api-client";

import type { AgentPayload } from "../types";
import type { CreateAgentParams } from "../types";

export class AgentApiError extends Error {
  status: number;

  constructor(message: string, status: number) {
    super(message);
    this.name = "AgentApiError";
    this.status = status;
  }
}

export async function fetchAgents(
  workspace?: string | null,
  signal?: AbortSignal
): Promise<AgentPayload[]> {
  const { data, error, response } = await apiClient.GET("/api/agents", {
    params: { query: agentWorkspaceQuery(workspace) },
    signal,
  });
  if (apiRequestFailed(response, error)) {
    throw new AgentApiError(
      defaultApiErrorMessage("Failed to fetch agents", response, error),
      response.status
    );
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
      throw new AgentApiError(`Agent not found: ${name}`, 404);
    }
    throw new AgentApiError(
      defaultApiErrorMessage(`Failed to fetch agent "${name}"`, response, error),
      response.status
    );
  }
  return requireResponseData(data, response, `Failed to fetch agent "${name}"`).agent;
}

export async function createAgent(
  params: CreateAgentParams,
  signal?: AbortSignal
): Promise<AgentPayload> {
  const { data, error, response } = await apiClient.POST("/api/agents", {
    body: params,
    signal,
  });

  if (apiRequestFailed(response, error)) {
    throw new AgentApiError(
      defaultApiErrorMessage("Failed to create agent", response, error),
      response.status
    );
  }

  return requireResponseData(data, response, "Failed to create agent").agent;
}

function agentWorkspaceQuery(workspace?: string | null): { workspace: string } | undefined {
  const trimmed = workspace?.trim();
  return trimmed ? { workspace: trimmed } : undefined;
}
