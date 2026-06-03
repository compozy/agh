import {
  apiClient,
  apiRequestFailed,
  defaultApiErrorMessage,
  requireResponseData,
} from "@/lib/api-client";
import type { OperationRequestBody } from "@/lib/api-contract";

import type { WorkspaceDetailPayload, WorkspacePayload } from "../types";

export type ResolveWorkspaceParams = OperationRequestBody<"resolveWorkspace">;

export class WorkspaceApiError extends Error {
  constructor(
    message: string,
    public readonly status: number
  ) {
    super(message);
    this.name = "WorkspaceApiError";
  }
}

export async function fetchWorkspaces(signal?: AbortSignal): Promise<WorkspacePayload[]> {
  const { data, error, response } = await apiClient.GET("/api/workspaces", { signal });
  if (apiRequestFailed(response, error)) {
    throw new WorkspaceApiError(
      defaultApiErrorMessage("Failed to fetch workspaces", response, error),
      response.status
    );
  }

  return requireResponseData(data, response, "Failed to fetch workspaces").workspaces;
}

export async function fetchWorkspace(
  workspaceID: string,
  signal?: AbortSignal
): Promise<WorkspaceDetailPayload> {
  const { data, error, response } = await apiClient.GET("/api/workspaces/{id}", {
    params: { path: { id: workspaceID } },
    signal,
  });
  if (apiRequestFailed(response, error)) {
    throw new WorkspaceApiError(
      defaultApiErrorMessage("Failed to fetch workspace", response, error),
      response.status
    );
  }

  return requireResponseData(data, response, "Failed to fetch workspace");
}

export async function deleteWorkspace(workspaceID: string, signal?: AbortSignal): Promise<void> {
  const { error, response } = await apiClient.DELETE("/api/workspaces/{id}", {
    params: { path: { id: workspaceID } },
    signal,
  });
  if (apiRequestFailed(response, error)) {
    throw new WorkspaceApiError(
      defaultApiErrorMessage("Failed to delete workspace", response, error),
      response.status
    );
  }
}

export async function resolveWorkspace(
  params: ResolveWorkspaceParams,
  signal?: AbortSignal
): Promise<WorkspacePayload> {
  const { data, error, response } = await apiClient.POST("/api/workspaces/resolve", {
    body: params,
    signal,
  });
  if (apiRequestFailed(response, error)) {
    throw new WorkspaceApiError(
      defaultApiErrorMessage("Failed to resolve workspace", response, error),
      response.status
    );
  }

  return requireResponseData(data, response, "Failed to resolve workspace").workspace;
}
