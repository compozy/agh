import { healthResponseSchema, type HealthPayload } from "../types";

export async function fetchHealth(signal?: AbortSignal): Promise<HealthPayload> {
  const res = await fetch("/api/observe/health", { signal });
  if (!res.ok) {
    throw new Error(`Daemon health check failed: ${res.status}`);
  }
  const json = await res.json();
  const parsed = healthResponseSchema.parse(json);
  return parsed.health;
}
