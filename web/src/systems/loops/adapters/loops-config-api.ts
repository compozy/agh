import {
  apiClient,
  apiRequestFailed,
  defaultApiErrorMessage,
  requireResponseData,
} from "@/lib/api-client";

import { LoopsApiError } from "./loops-api-errors";
import type {
  LoopAnnotation,
  LoopAnnotationsUpdateRequest,
  LoopConfig,
  LoopConfigUpdateRequest,
} from "../types";

export async function getLoopConfig(
  workspaceId: string,
  name: string,
  signal?: AbortSignal
): Promise<LoopConfig | null> {
  const { data, error, response } = await apiClient.GET(
    "/api/workspaces/{workspace_id}/loops/{name}/config",
    {
      params: { path: { workspace_id: workspaceId, name } },
      signal,
    }
  );

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) return null;
    throw new LoopsApiError(
      defaultApiErrorMessage(`Failed to fetch config for loop "${name}"`, response, error),
      response.status
    );
  }

  return (
    requireResponseData(data, response, `Failed to fetch config for loop "${name}"`).config ?? null
  );
}

export async function putLoopConfig(
  workspaceId: string,
  name: string,
  body: LoopConfigUpdateRequest,
  signal?: AbortSignal
): Promise<LoopConfig | null> {
  const { data, error, response } = await apiClient.PUT(
    "/api/workspaces/{workspace_id}/loops/{name}/config",
    {
      params: { path: { workspace_id: workspaceId, name } },
      body,
      signal,
    }
  );

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) throw new LoopsApiError(`Loop not found: ${name}`, 404);
    throw new LoopsApiError(
      defaultApiErrorMessage(`Failed to update config for loop "${name}"`, response, error),
      response.status
    );
  }

  return (
    requireResponseData(data, response, `Failed to update config for loop "${name}"`).config ?? null
  );
}

export async function getLoopAnnotations(
  workspaceId: string,
  name: string,
  signal?: AbortSignal
): Promise<LoopAnnotation[]> {
  const { data, error, response } = await apiClient.GET(
    "/api/workspaces/{workspace_id}/loops/{name}/annotations",
    {
      params: { path: { workspace_id: workspaceId, name } },
      signal,
    }
  );

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) throw new LoopsApiError(`Loop not found: ${name}`, 404);
    throw new LoopsApiError(
      defaultApiErrorMessage(`Failed to fetch annotations for loop "${name}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to fetch annotations for loop "${name}"`)
    .annotations;
}

export async function putLoopAnnotations(
  workspaceId: string,
  name: string,
  body: LoopAnnotationsUpdateRequest,
  signal?: AbortSignal
): Promise<LoopAnnotation[]> {
  const { data, error, response } = await apiClient.PUT(
    "/api/workspaces/{workspace_id}/loops/{name}/annotations",
    {
      params: { path: { workspace_id: workspaceId, name } },
      body,
      signal,
    }
  );

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) throw new LoopsApiError(`Loop not found: ${name}`, 404);
    throw new LoopsApiError(
      defaultApiErrorMessage(`Failed to save annotations for loop "${name}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to save annotations for loop "${name}"`)
    .annotations;
}
