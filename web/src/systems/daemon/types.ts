import type { OperationResponse } from "@/lib/api-contract";

export type ObserveHealthResponse = OperationResponse<"getObserveHealth", 200>;
export type HealthPayload = ObserveHealthResponse["health"];
export type MemoryHealthPayload = ObserveHealthResponse["memory"];
export type DaemonStatusResponse = OperationResponse<"getDaemonStatus", 200>;
export type DaemonStatusPayload = DaemonStatusResponse["daemon"];
