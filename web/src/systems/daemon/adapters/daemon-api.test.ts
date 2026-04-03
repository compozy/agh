import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

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
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(validResponse),
    } as Response);

    const result = await fetchHealth();
    expect(result).toEqual(validResponse.health);
    expect(fetch).toHaveBeenCalledWith("/api/observe/health", { signal: undefined });
  });

  it("passes abort signal to fetch", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(validResponse),
    } as Response);

    const controller = new AbortController();
    await fetchHealth(controller.signal);
    expect(fetch).toHaveBeenCalledWith("/api/observe/health", { signal: controller.signal });
  });

  it("throws on network error (non-ok response)", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: false,
      status: 500,
    } as Response);

    await expect(fetchHealth()).rejects.toThrow("Daemon health check failed: 500");
  });

  it("throws on fetch rejection", async () => {
    vi.mocked(fetch).mockRejectedValue(new TypeError("Failed to fetch"));

    await expect(fetchHealth()).rejects.toThrow("Failed to fetch");
  });

  it("throws on invalid response shape", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ invalid: "shape" }),
    } as Response);

    await expect(fetchHealth()).rejects.toThrow();
  });
});
