import { describe, expectTypeOf, it } from "vitest";

import type { HealthPayload, MemoryHealthPayload, ObserveHealthResponse } from "./types";

describe("daemon contract types", () => {
  it("derives health payloads from the generated observe contract", () => {
    expectTypeOf<HealthPayload>().toMatchTypeOf<{
      status: string;
      uptime_seconds: number;
      active_sessions: number;
      active_agents: number;
      bridges: {
        total_instances: number;
        route_count: number;
        delivery_backlog: number;
        delivery_dropped_total: number;
        delivery_failures_total: number;
        auth_failures_total: number;
        status_counts: {
          disabled: number;
          starting: number;
          ready: number;
          degraded: number;
          auth_required: number;
          error: number;
        };
      };
      global_db_size_bytes: number;
      session_db_size_bytes: number;
      persistence: {
        status: string;
        global_db_size_bytes: number;
        session_db_size_bytes: number;
      };
      retention: {
        enabled: boolean;
        retention_days: number;
        sweep_interval_seconds: number;
        last_sweep_status: string;
        last_sweep_at?: string | null;
        last_cutoff_at?: string | null;
        last_sweep_error?: string;
        deleted_event_summaries: number;
        deleted_token_stats: number;
        deleted_permission_log_rows: number;
      };
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
