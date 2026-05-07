import type { OperationRequestBody, OperationResponse } from "@/lib/api-contract";

export type ProviderModelsListResponse = OperationResponse<"listProviderModelsByProvider", 200>;
export type ProviderModelPayload = ProviderModelsListResponse["models"][number];
export type ProviderModelSource = ProviderModelPayload["sources"][number];

export type ProviderModelStatusResponse = OperationResponse<
  "getProviderModelStatusByProvider",
  200
>;
export type ProviderModelSourceStatus = ProviderModelStatusResponse["sources"][number];

export type ProviderModelsRefreshRequest = OperationRequestBody<"refreshProviderModelsByProvider">;
export type ProviderModelsRefreshResponse = OperationResponse<
  "refreshProviderModelsByProvider",
  200
>;

export const MODEL_AVAILABILITY_STATES = [
  "available_live",
  "available_stale",
  "unavailable_live",
  "unavailable_stale",
  "unknown",
] as const;

export type ModelAvailabilityState = (typeof MODEL_AVAILABILITY_STATES)[number];

export function isKnownAvailabilityState(value: string): value is ModelAvailabilityState {
  return (MODEL_AVAILABILITY_STATES as readonly string[]).includes(value);
}

export interface ProviderModelsQuery {
  providerId: string;
  sourceId?: string;
  includeStale?: boolean;
}

export interface ProviderModelsRefreshInput {
  providerId: string;
  sourceId?: string;
  force?: boolean;
  requestId?: string;
}
