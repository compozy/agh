import type { AgentPayload } from "../types";

export const agentFixtures: AgentPayload[] = [
  {
    name: "claude-agent",
    provider: "claude",
    prompt: "Review recent changes and explain the trade-offs.",
  },
  {
    name: "codex-agent",
    provider: "codex",
    model: "gpt-5.4",
    prompt: "Own implementation details and verification for coding tasks.",
  },
  {
    name: "gemini-agent",
    provider: "gemini",
    prompt: "Summarize architectural decisions for the current workspace.",
  },
];

export const primaryAgentFixture: AgentPayload = agentFixtures[0];
