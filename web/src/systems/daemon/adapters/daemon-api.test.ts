import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { expectFetchRequest, mockJsonResponse } from "@/test/fetch-test-utils";

import { fetchHealth } from "./daemon-api";

describe("fetchHealth", () => {
  const validResponse = {
    health: {
      status: "ok",
      uptime_seconds: 3600,
      active_sessions: 2,
      active_agents: 3,
      global_db_size_bytes: 1048576,
      session_db_size_bytes: 524288,
      version: "0.1.0",
    },
  };

  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("returns parsed HealthPayload on success", async () => {
    mockJsonResponse(validResponse);

    const result = await fetchHealth();

    expect(result).toEqual(validResponse.health);
    await expectFetchRequest({ path: "/api/observe/health" });
  });

  it("passes abort signal to fetch", async () => {
    mockJsonResponse(validResponse);

    const controller = new AbortController();
    await fetchHealth(controller.signal);

    await expectFetchRequest({
      path: "/api/observe/health",
      signal: controller.signal,
    });
  });

  it("throws on network error (non-ok response)", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 500 }));

    await expect(fetchHealth()).rejects.toThrow("Daemon health check failed: 500");
  });

  it("throws on fetch rejection", async () => {
    vi.mocked(globalThis.fetch).mockRejectedValue(new TypeError("Failed to fetch"));

    await expect(fetchHealth()).rejects.toThrow("Failed to fetch");
  });
});
