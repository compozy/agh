import { describe, expectTypeOf, it } from "vitest";

import type { HealthPayload, MemoryHealthPayload, ObserveHealthResponse } from "./types";

describe("daemon contract types", () => {
  it("derives health payloads from the generated observe contract", () => {
    expectTypeOf<HealthPayload>().toMatchTypeOf<{
      status: string;
      uptime_seconds: number;
      active_sessions: number;
      active_agents: number;
      global_db_size_bytes: number;
      session_db_size_bytes: number;
      version: string;
    }>();

    expectTypeOf<MemoryHealthPayload>().toMatchTypeOf<{
      dream_enabled: boolean;
      global_files: number;
      workspace_files: number;
      last_consolidation: string | null;
    }>();

    expectTypeOf<ObserveHealthResponse>().toMatchTypeOf<{
      health: HealthPayload;
      memory: MemoryHealthPayload;
    }>();
  });
});
