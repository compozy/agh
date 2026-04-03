import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { fetchAgent, fetchAgents } from "./agent-api";

describe("fetchAgents", () => {
  const validResponse = {
    agents: [
      {
        name: "claude-agent",
        provider: "claude",
        prompt: "You are a helpful assistant",
      },
      {
        name: "codex-agent",
        provider: "codex",
        model: "o3",
        prompt: "Code reviewer",
      },
    ],
  };

  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("returns array of AgentPayload on success", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(validResponse),
    } as Response);

    const result = await fetchAgents();
    expect(result).toEqual(validResponse.agents);
    expect(result).toHaveLength(2);
  });

  it("passes abort signal to fetch", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(validResponse),
    } as Response);

    const controller = new AbortController();
    await fetchAgents(controller.signal);
    expect(fetch).toHaveBeenCalledWith("/api/agents", { signal: controller.signal });
  });

  it("throws on non-ok response", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: false,
      status: 500,
    } as Response);

    await expect(fetchAgents()).rejects.toThrow("Failed to fetch agents: 500");
  });

  it("returns empty array when server returns empty list", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ agents: [] }),
    } as Response);

    const result = await fetchAgents();
    expect(result).toEqual([]);
  });
});

describe("fetchAgent", () => {
  const validResponse = {
    agent: {
      name: "claude-agent",
      provider: "claude",
      prompt: "You are a helpful assistant",
    },
  };

  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("returns single AgentPayload on success", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(validResponse),
    } as Response);

    const result = await fetchAgent("claude-agent");
    expect(result).toEqual(validResponse.agent);
    expect(fetch).toHaveBeenCalledWith("/api/agents/claude-agent", { signal: undefined });
  });

  it("throws 404 error for unknown agent", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: false,
      status: 404,
    } as Response);

    await expect(fetchAgent("unknown")).rejects.toThrow("Agent not found: unknown");
  });

  it("throws generic error for other failures", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: false,
      status: 503,
    } as Response);

    await expect(fetchAgent("claude-agent")).rejects.toThrow(
      'Failed to fetch agent "claude-agent": 503'
    );
  });

  it("encodes agent name in URL", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(validResponse),
    } as Response);

    await fetchAgent("my agent");
    expect(fetch).toHaveBeenCalledWith("/api/agents/my%20agent", { signal: undefined });
  });
});
