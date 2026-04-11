import {
  apiClient,
  apiRequestFailed,
  defaultApiErrorMessage,
  requireResponseData,
} from "@/lib/api-client";

import type {
  MemoryConsolidateResponse,
  MemoryHeader,
  MemoryMutationResponse,
  MemoryScope,
} from "../types";

export class KnowledgeApiError extends Error {
  constructor(
    message: string,
    public readonly status: number
  ) {
    super(message);
    this.name = "KnowledgeApiError";
  }
}

export async function listMemories(
  scope?: MemoryScope,
  workspace?: string,
  signal?: AbortSignal
): Promise<MemoryHeader[]> {
  const { data, error, response } = await apiClient.GET("/api/memory", {
    params: {
      query: {
        scope,
        workspace,
      },
    },
    signal,
  });
  if (apiRequestFailed(response, error)) {
    throw new KnowledgeApiError(
      defaultApiErrorMessage("Failed to fetch memories", response, error),
      response.status
    );
  }
  return requireResponseData(data, response, "Failed to fetch memories");
}

export async function readMemory(
  scope: MemoryScope,
  filename: string,
  workspace?: string,
  signal?: AbortSignal
): Promise<string> {
  const { data, error, response } = await apiClient.GET("/api/memory/{filename}", {
    params: {
      path: { filename },
      query: {
        scope,
        workspace,
      },
    },
    signal,
  });
  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new KnowledgeApiError(`Memory not found: ${filename}`, 404);
    }
    throw new KnowledgeApiError(
      defaultApiErrorMessage(`Failed to read memory "${filename}"`, response, error),
      response.status
    );
  }
  return requireResponseData(data, response, `Failed to read memory "${filename}"`).content;
}

export async function writeMemory(
  filename: string,
  content: string,
  scope?: MemoryScope,
  workspace?: string,
  signal?: AbortSignal
): Promise<MemoryMutationResponse> {
  const { data, error, response } = await apiClient.PUT("/api/memory/{filename}", {
    params: { path: { filename } },
    body: { content, scope, workspace },
    signal,
  });
  if (apiRequestFailed(response, error)) {
    throw new KnowledgeApiError(
      defaultApiErrorMessage(`Failed to write memory "${filename}"`, response, error),
      response.status
    );
  }
  return requireResponseData(data, response, `Failed to write memory "${filename}"`);
}

export async function deleteMemory(
  scope: MemoryScope,
  filename: string,
  workspace?: string,
  signal?: AbortSignal
): Promise<MemoryMutationResponse> {
  const { data, error, response } = await apiClient.DELETE("/api/memory/{filename}", {
    params: {
      path: { filename },
      query: {
        scope,
        workspace,
      },
    },
    signal,
  });
  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new KnowledgeApiError(`Memory not found: ${filename}`, 404);
    }
    throw new KnowledgeApiError(
      defaultApiErrorMessage(`Failed to delete memory "${filename}"`, response, error),
      response.status
    );
  }
  return requireResponseData(data, response, `Failed to delete memory "${filename}"`);
}

export async function consolidateMemory(
  workspace?: string,
  signal?: AbortSignal
): Promise<MemoryConsolidateResponse> {
  const { data, error, response } = await apiClient.POST("/api/memory/consolidate", {
    body: { workspace },
    signal,
  });
  if (apiRequestFailed(response, error)) {
    throw new KnowledgeApiError(
      defaultApiErrorMessage("Failed to consolidate memory", response, error),
      response.status
    );
  }
  return requireResponseData(data, response, "Failed to consolidate memory");
}
