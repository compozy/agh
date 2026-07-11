import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { expectFetchRequest, mockJsonResponse } from "@/test/fetch-test-utils";

import type { CreateAgentParams, DuplicateAgentParams, UpdateAgentParams } from "../../types";
import {
  AgentDigestConflictError,
  AgentTargetExistsError,
  createAgent,
  deleteAgent,
  duplicateAgent,
  fetchAgent,
  fetchAgents,
  isAgentDigestConflict,
  isAgentTargetExists,
  updateAgent,
} from "../agent-api";

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
    mockJsonResponse(validResponse);

    const result = await fetchAgents();

    expect(result).toEqual(validResponse.agents);
    expect(result).toHaveLength(2);
    await expectFetchRequest({ path: "/api/agents" });
  });

  it("passes abort signal to fetch", async () => {
    mockJsonResponse(validResponse);

    const controller = new AbortController();
    await fetchAgents(null, controller.signal);

    await expectFetchRequest({
      path: "/api/agents",
      signal: controller.signal,
    });
  });

  it("passes workspace context to the daemon", async () => {
    mockJsonResponse(validResponse);

    await fetchAgents("ws_alpha");

    await expectFetchRequest({ path: "/api/agents?workspace=ws_alpha" });
  });

  it("throws on non-ok response", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 500 }));

    await expect(fetchAgents()).rejects.toThrow("Failed to fetch agents: 500");
  });

  it("returns empty array when server returns empty list", async () => {
    mockJsonResponse({ agents: [] });

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
    mockJsonResponse(validResponse);

    const result = await fetchAgent("claude-agent");

    expect(result).toEqual(validResponse.agent);
    await expectFetchRequest({ path: "/api/agents/claude-agent" });
  });

  it("passes workspace context when fetching one agent", async () => {
    mockJsonResponse(validResponse);

    await fetchAgent("claude-agent", "ws_alpha");

    await expectFetchRequest({ path: "/api/agents/claude-agent?workspace=ws_alpha" });
  });

  it("throws 404 error for unknown agent", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 404 }));

    await expect(fetchAgent("unknown")).rejects.toThrow("Agent not found: unknown");
  });

  it("throws generic error for other failures", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 503 }));

    await expect(fetchAgent("claude-agent")).rejects.toThrow(
      'Failed to fetch agent "claude-agent": 503'
    );
  });

  it("encodes agent name in URL", async () => {
    mockJsonResponse(validResponse);

    await fetchAgent("my agent");

    await expectFetchRequest({ path: "/api/agents/my%20agent" });
  });
});

describe("createAgent", () => {
  const request: CreateAgentParams = {
    scope: "workspace",
    workspace: "ws_alpha",
    agent: {
      name: "release-captain",
      provider: "codex",
      prompt: "Own release readiness.",
      model: "gpt-5.4",
      tools: ["agh__skill_view"],
    },
  };

  const response = {
    agent: {
      name: "release-captain",
      provider: "codex",
      model: "gpt-5.4",
      prompt: "Own release readiness.",
      tools: ["agh__skill_view"],
    },
  };

  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("posts the create-agent payload and returns the created agent", async () => {
    mockJsonResponse(response, { status: 201 });

    const result = await createAgent(request);

    expect(result).toEqual(response.agent);
    await expectFetchRequest({
      path: "/api/agents",
      method: "POST",
      body: request,
    });
  });

  it("passes abort signal to the create request", async () => {
    mockJsonResponse(response, { status: 201 });

    const controller = new AbortController();
    await createAgent(request, controller.signal);

    await expectFetchRequest({
      path: "/api/agents",
      method: "POST",
      signal: controller.signal,
    });
  });

  it("throws backend duplicate errors with the response status", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(
      new Response(JSON.stringify({ error: "agent definition already exists" }), {
        status: 409,
        headers: { "Content-Type": "application/json" },
      })
    );

    await expect(createAgent(request)).rejects.toSatisfy(
      (error: unknown) =>
        error instanceof AgentTargetExistsError &&
        isAgentTargetExists(error) &&
        error.message.includes("agent definition already exists")
    );
  });
});

describe("updateAgent", () => {
  const params: UpdateAgentParams = {
    expected_digest: "abc",
    agent: {
      name: "claude-agent",
      provider: "claude",
      prompt: "Updated prompt",
    },
  };

  const response = {
    agent: {
      name: "claude-agent",
      provider: "claude",
      prompt: "Updated prompt",
      definition_digest: "def",
    },
  };

  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("Should put the update payload and return the agent", async () => {
    mockJsonResponse(response);

    const result = await updateAgent("claude-agent", params);

    expect(result).toEqual(response.agent);
    await expectFetchRequest({
      path: "/api/agents/claude-agent",
      method: "PUT",
      body: params,
    });
  });

  it("Should throw AgentDigestConflictError on 409", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(
      new Response(JSON.stringify({ error: "definition digest conflict" }), {
        status: 409,
        headers: { "Content-Type": "application/json" },
      })
    );

    await expect(updateAgent("claude-agent", params)).rejects.toSatisfy(
      (error: unknown) => error instanceof AgentDigestConflictError && isAgentDigestConflict(error)
    );
  });
});

describe("deleteAgent", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("Should delete with workspace query and return unshadowed_origin", async () => {
    mockJsonResponse({
      name: "claude-agent",
      origin: "workspace",
      unshadowed_origin: "global",
    });

    const result = await deleteAgent("claude-agent", "ws_alpha");

    expect(result).toEqual({
      name: "claude-agent",
      origin: "workspace",
      unshadowed_origin: "global",
    });
    await expectFetchRequest({
      path: "/api/agents/claude-agent?workspace=ws_alpha",
      method: "DELETE",
    });
  });

  it("Should throw 404 for unknown agents", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 404 }));

    await expect(deleteAgent("missing")).rejects.toThrow("Agent not found: missing");
  });
});

describe("duplicateAgent", () => {
  const params: DuplicateAgentParams = {
    name: "claude-agent-copy",
    scope: "workspace",
    workspace: "ws_alpha",
    overrides: { prompt: "Copy prompt" },
  };

  const response = {
    agent: {
      name: "claude-agent-copy",
      provider: "claude",
      prompt: "Copy prompt",
    },
  };

  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("Should post duplicate and return the new agent", async () => {
    mockJsonResponse(response, { status: 201 });

    const result = await duplicateAgent("claude-agent", params);

    expect(result).toEqual(response.agent);
    await expectFetchRequest({
      path: "/api/agents/claude-agent/duplicate",
      method: "POST",
      body: params,
    });
  });

  it("Should throw AgentTargetExistsError on 409", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(
      new Response(JSON.stringify({ error: "agent definition already exists" }), {
        status: 409,
        headers: { "Content-Type": "application/json" },
      })
    );

    await expect(duplicateAgent("claude-agent", params)).rejects.toSatisfy(
      (error: unknown) => error instanceof AgentTargetExistsError && isAgentTargetExists(error)
    );
  });
});
