import { z } from "zod";

// --- HealthPayload ---

export const healthPayloadSchema = z.object({
  status: z.string(),
  uptime_seconds: z.number(),
  active_sessions: z.number(),
  active_agents: z.number(),
  global_db_size_bytes: z.number(),
  session_db_size_bytes: z.number(),
  version: z.string(),
});

export type HealthPayload = z.infer<typeof healthPayloadSchema>;

// --- API Response Envelope ---

export const healthResponseSchema = z.object({
  health: healthPayloadSchema,
});

export type HealthResponse = z.infer<typeof healthResponseSchema>;
