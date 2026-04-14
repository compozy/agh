import {
  apiClient,
  apiRequestFailed,
  defaultApiErrorMessage,
  requireResponseData,
} from "@/lib/api-client";

import type {
  BridgeDetailResponse,
  BridgeRoute,
  BridgeProvider,
  BridgesListResponse,
  CreateBridgeRequest,
  CreateBridgeResponse,
  TestBridgeDeliveryRequest,
  TestBridgeDeliveryResponse,
} from "../types";

export class BridgesApiError extends Error {
  constructor(
    message: string,
    public readonly status: number
  ) {
    super(message);
    this.name = "BridgesApiError";
  }
}

export async function listBridges(signal?: AbortSignal): Promise<BridgesListResponse> {
  const { data, error, response } = await apiClient.GET("/api/bridges", { signal });

  if (apiRequestFailed(response, error)) {
    throw new BridgesApiError(
      defaultApiErrorMessage("Failed to fetch bridges", response, error),
      response.status
    );
  }

  return requireResponseData(data, response, "Failed to fetch bridges");
}

export async function listBridgeProviders(signal?: AbortSignal): Promise<BridgeProvider[]> {
  const { data, error, response } = await apiClient.GET("/api/bridges/providers", { signal });

  if (apiRequestFailed(response, error)) {
    throw new BridgesApiError(
      defaultApiErrorMessage("Failed to fetch bridge providers", response, error),
      response.status
    );
  }

  return requireResponseData(data, response, "Failed to fetch bridge providers").providers;
}

export async function getBridge(id: string, signal?: AbortSignal): Promise<BridgeDetailResponse> {
  const { data, error, response } = await apiClient.GET("/api/bridges/{id}", {
    params: { path: { id } },
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new BridgesApiError(`Bridge not found: ${id}`, 404);
    }

    throw new BridgesApiError(
      defaultApiErrorMessage(`Failed to load bridge "${id}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to load bridge "${id}"`);
}

export async function listBridgeRoutes(id: string, signal?: AbortSignal): Promise<BridgeRoute[]> {
  const { data, error, response } = await apiClient.GET("/api/bridges/{id}/routes", {
    params: { path: { id } },
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new BridgesApiError(`Bridge not found: ${id}`, 404);
    }

    throw new BridgesApiError(
      defaultApiErrorMessage(`Failed to load routes for bridge "${id}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to load routes for bridge "${id}"`).routes;
}

export async function createBridge(
  data: CreateBridgeRequest,
  signal?: AbortSignal
): Promise<CreateBridgeResponse> {
  const {
    data: responseData,
    error,
    response,
  } = await apiClient.POST("/api/bridges", {
    body: data,
    signal,
  });

  if (apiRequestFailed(response, error)) {
    throw new BridgesApiError(
      defaultApiErrorMessage("Failed to create bridge", response, error),
      response.status
    );
  }

  return requireResponseData(responseData, response, "Failed to create bridge");
}

export async function testBridgeDelivery(
  id: string,
  data: TestBridgeDeliveryRequest,
  signal?: AbortSignal
): Promise<TestBridgeDeliveryResponse> {
  const {
    data: responseData,
    error,
    response,
  } = await apiClient.POST("/api/bridges/{id}/test-delivery", {
    body: data,
    params: { path: { id } },
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new BridgesApiError(`Bridge not found: ${id}`, 404);
    }
    if (response.status === 409) {
      throw new BridgesApiError(
        defaultApiErrorMessage(`Bridge "${id}" is unavailable`, response, error),
        409
      );
    }

    throw new BridgesApiError(
      defaultApiErrorMessage(`Failed to test delivery for bridge "${id}"`, response, error),
      response.status
    );
  }

  return requireResponseData(responseData, response, `Failed to test delivery for bridge "${id}"`);
}
