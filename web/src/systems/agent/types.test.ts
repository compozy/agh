import { describe, expect, it } from "vitest";

import {
  agentMCPServerSchema,
  agentPayloadSchema,
  agentResponseSchema,
  agentsResponseSchema,
} from "./types";

describe("agentPayloadSchema", () => {
  const validAgent = {
    name: "claude",
    provider: "anthropic",
    prompt: "You are a helpful assistant.",
  };

  it("validates a minimal valid agent", () => {
    const result = agentPayloadSchema.safeParse(validAgent);
    expect(result.success).toBe(true);
  });

  it("validates an agent with all optional fields", () => {
    const full = {
      ...validAgent,
      command: "claude-code",
      model: "claude-opus-4-6",
      tools: ["Read", "Write", "Bash"],
      permissions: "auto-approve",
      mcp_servers: [
        {
          name: "neon",
          command: "npx",
          args: ["@neondatabase/mcp-server"],
          env: { NEON_API_KEY: "test" },
        },
      ],
    };
    const result = agentPayloadSchema.safeParse(full);
    expect(result.success).toBe(true);
  });

  it("rejects missing required field: name", () => {
    const { name: _, ...noName } = validAgent;
    const result = agentPayloadSchema.safeParse(noName);
    expect(result.success).toBe(false);
  });

  it("rejects missing required field: provider", () => {
    const { provider: _, ...noProvider } = validAgent;
    const result = agentPayloadSchema.safeParse(noProvider);
    expect(result.success).toBe(false);
  });

  it("rejects missing required field: prompt", () => {
    const { prompt: _, ...noPrompt } = validAgent;
    const result = agentPayloadSchema.safeParse(noPrompt);
    expect(result.success).toBe(false);
  });
});

describe("agentMCPServerSchema", () => {
  it("validates a minimal MCP server", () => {
    const result = agentMCPServerSchema.safeParse({
      name: "test-server",
      command: "node",
    });
    expect(result.success).toBe(true);
  });

  it("validates an MCP server with all fields", () => {
    const result = agentMCPServerSchema.safeParse({
      name: "test-server",
      command: "node",
      args: ["server.js", "--port", "3000"],
      env: { NODE_ENV: "production" },
    });
    expect(result.success).toBe(true);
  });

  it("rejects missing command", () => {
    const result = agentMCPServerSchema.safeParse({ name: "test" });
    expect(result.success).toBe(false);
  });
});

describe("API response envelopes", () => {
  const validAgent = {
    name: "claude",
    provider: "anthropic",
    prompt: "You are helpful.",
  };

  it("agentsResponseSchema validates agents list", () => {
    const result = agentsResponseSchema.safeParse({
      agents: [validAgent],
    });
    expect(result.success).toBe(true);
  });

  it("agentResponseSchema validates single agent", () => {
    const result = agentResponseSchema.safeParse({
      agent: validAgent,
    });
    expect(result.success).toBe(true);
  });

  it("agentsResponseSchema validates empty list", () => {
    const result = agentsResponseSchema.safeParse({ agents: [] });
    expect(result.success).toBe(true);
  });
});
