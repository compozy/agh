import {
  apiClient,
  apiRequestFailed,
  defaultApiErrorMessage,
  requireResponseData,
} from "@/lib/api-client";

import { NetworkApiError } from "./network-api-error";

export type NetworkCoordinationResponse = Awaited<ReturnType<typeof getNetworkCoordination>>;
export type NetworkUsageResponse = Awaited<ReturnType<typeof getNetworkUsage>>;

export async function getNetworkCoordination(
  workspaceId: string,
  options: { taskId?: string; signal?: AbortSignal } = {}
) {
  const { data, error, response } = await apiClient.GET(
    "/api/workspaces/{workspace_id}/network-coordination",
    {
      params: {
        path: { workspace_id: workspaceId },
        query: options.taskId ? { task_id: options.taskId } : undefined,
      },
      signal: options.signal,
    }
  );
  if (apiRequestFailed(response, error)) {
    throw new NetworkApiError(
      defaultApiErrorMessage("Failed to fetch network coordination", response, error),
      response.status
    );
  }
  return requireResponseData(data, response, "Failed to fetch network coordination").coordination;
}

export async function putNetworkCoordination(
  workspaceId: string,
  body: { enabled: boolean },
  options: { taskId?: string; signal?: AbortSignal } = {}
) {
  const { data, error, response } = await apiClient.PUT(
    "/api/workspaces/{workspace_id}/network-coordination",
    {
      params: {
        path: { workspace_id: workspaceId },
        query: options.taskId ? { task_id: options.taskId } : undefined,
      },
      body,
      signal: options.signal,
    }
  );
  if (apiRequestFailed(response, error)) {
    throw new NetworkApiError(
      defaultApiErrorMessage("Failed to update network coordination", response, error),
      response.status
    );
  }
  return requireResponseData(data, response, "Failed to update network coordination").coordination;
}

export async function putNetworkCoordinationInvitation(
  workspaceId: string,
  body: { scope: string; task_id?: string; dismissed: boolean },
  options: { signal?: AbortSignal } = {}
) {
  const { data, error, response } = await apiClient.PUT(
    "/api/workspaces/{workspace_id}/network-coordination/invitation",
    {
      params: { path: { workspace_id: workspaceId } },
      body,
      signal: options.signal,
    }
  );
  if (apiRequestFailed(response, error)) {
    throw new NetworkApiError(
      defaultApiErrorMessage("Failed to update coordination invitation", response, error),
      response.status
    );
  }
  return requireResponseData(data, response, "Failed to update coordination invitation")
    .coordination;
}

export async function getNetworkUsage(workspaceId: string, signal?: AbortSignal) {
  const { data, error, response } = await apiClient.GET(
    "/api/workspaces/{workspace_id}/network/usage",
    {
      params: { path: { workspace_id: workspaceId } },
      signal,
    }
  );
  if (apiRequestFailed(response, error)) {
    throw new NetworkApiError(
      defaultApiErrorMessage("Failed to fetch network usage", response, error),
      response.status
    );
  }
  return requireResponseData(data, response, "Failed to fetch network usage");
}
