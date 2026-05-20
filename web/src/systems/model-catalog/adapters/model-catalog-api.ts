import {
  apiClient,
  apiRequestFailed,
  defaultApiErrorMessage,
  requireResponseData,
} from "@/lib/api-client";

import type {
  ProviderModelsListResponse,
  ProviderModelsQuery,
  ProviderModelsRefreshInput,
  ProviderModelsRefreshResponse,
  ProviderModelStatusResponse,
} from "../types";

export class ModelCatalogApiError extends Error {
  constructor(
    message: string,
    public readonly status: number
  ) {
    super(message);
    this.name = "ModelCatalogApiError";
  }
}

export async function listProviderModels(
  input: ProviderModelsQuery,
  signal?: AbortSignal
): Promise<ProviderModelsListResponse> {
  const providerId = input.providerId.trim();
  if (providerId.length === 0) {
    throw new ModelCatalogApiError("provider_id is required", 400);
  }
  const query = buildListQuery(input);
  const { data, error, response } = await apiClient.GET(
    "/api/model-catalog/providers/{provider_id}/models",
    {
      params: { path: { provider_id: providerId }, query },
      signal,
    }
  );
  if (apiRequestFailed(response, error)) {
    throw new ModelCatalogApiError(
      defaultApiErrorMessage(`Failed to load models for "${providerId}"`, response, error),
      response.status
    );
  }
  return requireResponseData(data, response, `Failed to load models for "${providerId}"`);
}

export async function getProviderModelStatus(
  providerId: string,
  signal?: AbortSignal
): Promise<ProviderModelStatusResponse> {
  const trimmed = providerId.trim();
  if (trimmed.length === 0) {
    throw new ModelCatalogApiError("provider_id is required", 400);
  }
  const { data, error, response } = await apiClient.GET(
    "/api/model-catalog/providers/{provider_id}/models/status",
    {
      params: { path: { provider_id: trimmed } },
      signal,
    }
  );
  if (apiRequestFailed(response, error)) {
    throw new ModelCatalogApiError(
      defaultApiErrorMessage(
        `Failed to load model source status for "${trimmed}"`,
        response,
        error
      ),
      response.status
    );
  }
  return requireResponseData(data, response, `Failed to load model source status for "${trimmed}"`);
}

export async function refreshProviderModels(
  input: ProviderModelsRefreshInput,
  signal?: AbortSignal
): Promise<ProviderModelsRefreshResponse> {
  const providerId = input.providerId.trim();
  if (providerId.length === 0) {
    throw new ModelCatalogApiError("provider_id is required", 400);
  }
  const body = buildRefreshBody(input);
  const { data, error, response } = await apiClient.POST(
    "/api/model-catalog/providers/{provider_id}/models/refresh",
    {
      params: { path: { provider_id: providerId } },
      body,
      signal,
    }
  );
  if (apiRequestFailed(response, error)) {
    throw new ModelCatalogApiError(
      defaultApiErrorMessage(`Failed to refresh models for "${providerId}"`, response, error),
      response.status
    );
  }
  return requireResponseData(data, response, `Failed to refresh models for "${providerId}"`);
}

function buildListQuery(input: ProviderModelsQuery): {
  source_id?: string;
  include_stale?: boolean;
} {
  const query: { source_id?: string; include_stale?: boolean } = {};
  const sourceId = input.sourceId?.trim();
  if (sourceId) {
    query.source_id = sourceId;
  }
  if (input.includeStale) {
    query.include_stale = true;
  }
  return query;
}

function buildRefreshBody(input: ProviderModelsRefreshInput): {
  force?: boolean;
  request_id?: string;
  source_id?: string;
} {
  const body: { force?: boolean; request_id?: string; source_id?: string } = {};
  const sourceId = input.sourceId?.trim();
  if (sourceId) {
    body.source_id = sourceId;
  }
  const requestId = input.requestId?.trim();
  if (requestId) {
    body.request_id = requestId;
  }
  if (input.force) {
    body.force = true;
  }
  return body;
}
