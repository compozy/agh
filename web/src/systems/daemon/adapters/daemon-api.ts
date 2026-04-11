import {
  apiClient,
  apiRequestFailed,
  defaultApiErrorMessage,
  requireResponseData,
} from "@/lib/api-client";

import type { DaemonStatusPayload, HealthPayload } from "../types";

export async function fetchHealth(signal?: AbortSignal): Promise<HealthPayload> {
  const { data, error, response } = await apiClient.GET("/api/observe/health", { signal });
  if (apiRequestFailed(response, error)) {
    throw new Error(defaultApiErrorMessage("Daemon health check failed", response, error));
  }
  return requireResponseData(data, response, "Daemon health check failed").health;
}

export async function fetchDaemonStatus(signal?: AbortSignal): Promise<DaemonStatusPayload> {
  const { data, error, response } = await apiClient.GET("/api/daemon/status", { signal });
  if (apiRequestFailed(response, error)) {
    throw new Error(defaultApiErrorMessage("Daemon status check failed", response, error));
  }
  return requireResponseData(data, response, "Daemon status check failed").daemon;
}
