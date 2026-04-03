import { describe, expect, it } from "vitest";

import { healthPayloadSchema, healthResponseSchema } from "./types";

describe("healthPayloadSchema", () => {
  const validHealth = {
    status: "healthy",
    uptime_seconds: 3600,
    active_sessions: 2,
    active_agents: 3,
    global_db_size_bytes: 1048576,
    session_db_size_bytes: 524288,
    version: "0.1.0",
  };

  it("validates a valid health payload", () => {
    const result = healthPayloadSchema.safeParse(validHealth);
    expect(result.success).toBe(true);
  });

  it("rejects missing required field: status", () => {
    const { status: _, ...noStatus } = validHealth;
    const result = healthPayloadSchema.safeParse(noStatus);
    expect(result.success).toBe(false);
  });

  it("rejects missing required field: uptime_seconds", () => {
    const { uptime_seconds: _, ...noUptime } = validHealth;
    const result = healthPayloadSchema.safeParse(noUptime);
    expect(result.success).toBe(false);
  });

  it("rejects missing required field: version", () => {
    const { version: _, ...noVersion } = validHealth;
    const result = healthPayloadSchema.safeParse(noVersion);
    expect(result.success).toBe(false);
  });

  it("rejects non-number for active_sessions", () => {
    const result = healthPayloadSchema.safeParse({
      ...validHealth,
      active_sessions: "two",
    });
    expect(result.success).toBe(false);
  });

  it("validates zero values for numeric fields", () => {
    const result = healthPayloadSchema.safeParse({
      ...validHealth,
      active_sessions: 0,
      active_agents: 0,
      global_db_size_bytes: 0,
      session_db_size_bytes: 0,
      uptime_seconds: 0,
    });
    expect(result.success).toBe(true);
  });
});

describe("healthResponseSchema", () => {
  it("validates a valid health response envelope", () => {
    const result = healthResponseSchema.safeParse({
      health: {
        status: "healthy",
        uptime_seconds: 100,
        active_sessions: 1,
        active_agents: 2,
        global_db_size_bytes: 0,
        session_db_size_bytes: 0,
        version: "0.1.0",
      },
    });
    expect(result.success).toBe(true);
  });

  it("rejects missing health envelope", () => {
    const result = healthResponseSchema.safeParse({
      status: "healthy",
    });
    expect(result.success).toBe(false);
  });
});
