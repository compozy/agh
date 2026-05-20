import {
  apiClient,
  apiRequestFailed,
  defaultApiErrorMessage,
  requireResponseData,
} from "@/lib/api-client";

import type { DaemonStatusPayload, HealthPayload, StatusPayload } from "../types";

export async function fetchStatus(signal?: AbortSignal): Promise<StatusPayload> {
  const { data, error, response } = await apiClient.GET("/api/status", { signal });
  if (apiRequestFailed(response, error)) {
    throw new Error(defaultApiErrorMessage("Runtime status check failed", response, error));
  }
  return requireResponseData(data, response, "Runtime status check failed");
}

export async function fetchHealth(signal?: AbortSignal): Promise<HealthPayload> {
  return (await fetchStatus(signal)).health;
}

export async function fetchDaemonStatus(signal?: AbortSignal): Promise<DaemonStatusPayload> {
  return (await fetchStatus(signal)).daemon;
}
